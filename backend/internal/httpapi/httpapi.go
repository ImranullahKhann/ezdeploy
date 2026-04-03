package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"ezdeploy/backend/internal/auth"
	"ezdeploy/backend/internal/db"
	"ezdeploy/backend/internal/middleware"
)

type Handler struct {
	pool        *pgxpool.Pool
	authService *auth.Service
}

func New(pool *pgxpool.Pool, authService *auth.Service) http.Handler {
	h := &Handler{pool: pool, authService: authService}
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", h.handleHealth)
	mux.HandleFunc("/readyz", h.handleReady)
	if authService != nil {
		mux.HandleFunc("/auth/signup", h.handleSignup)
		mux.HandleFunc("/auth/login", h.handleLogin)
		mux.Handle("/auth/me", middleware.RequireAuth(authService, http.HandlerFunc(h.handleMe)))
		mux.Handle("/auth/logout", middleware.RequireAuth(authService, http.HandlerFunc(h.handleLogout)))
	}
	return mux
}

type authRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type authResponse struct {
	User auth.User `json:"user"`
}

type statusResponse struct {
	Status string    `json:"status"`
	Time   time.Time `json:"time"`
}

func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, statusResponse{Status: "ok", Time: time.Now().UTC()})
}

func (h *Handler) handleReady(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	if err := db.Health(ctx, h.pool); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{
			"status": "not_ready",
			"error":  err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, statusResponse{Status: "ok", Time: time.Now().UTC()})
}

func (h *Handler) handleSignup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w, http.MethodPost)
		return
	}
	if h.authService == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "auth service unavailable"})
		return
	}

	request, err := decodeAuthRequest(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	user, cookie, err := h.authService.Signup(r.Context(), request.Email, request.Password)
	if err != nil {
		handleAuthError(w, err)
		return
	}

	http.SetCookie(w, cookie)
	writeJSON(w, http.StatusCreated, authResponse{User: user})
}

func (h *Handler) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w, http.MethodPost)
		return
	}
	if h.authService == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "auth service unavailable"})
		return
	}

	request, err := decodeAuthRequest(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	user, cookie, err := h.authService.Login(r.Context(), request.Email, request.Password)
	if err != nil {
		handleAuthError(w, err)
		return
	}

	http.SetCookie(w, cookie)
	writeJSON(w, http.StatusOK, authResponse{User: user})
}

func (h *Handler) handleMe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w, http.MethodGet)
		return
	}

	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	writeJSON(w, http.StatusOK, authResponse{User: user})
}

func (h *Handler) handleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w, http.MethodPost)
		return
	}
	if h.authService == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "auth service unavailable"})
		return
	}

	cookie, err := r.Cookie(auth.SessionCookieName)
	if err != nil || cookie.Value == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	if err := h.authService.Revoke(r.Context(), cookie.Value); err != nil {
		handleAuthError(w, err)
		return
	}

	http.SetCookie(w, h.authService.ClearCookie())
	w.WriteHeader(http.StatusNoContent)
}

func decodeAuthRequest(r *http.Request) (authRequest, error) {
	defer r.Body.Close()

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		return authRequest{}, errors.New("read request body")
	}

	var request authRequest
	if err := json.Unmarshal(body, &request); err != nil {
		return authRequest{}, errors.New("invalid json")
	}

	request.Email = strings.TrimSpace(request.Email)
	if request.Email == "" || request.Password == "" {
		return authRequest{}, errors.New("email and password are required")
	}

	return request, nil
}

func handleAuthError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	message := err.Error()

	switch {
	case errors.Is(err, auth.ErrInvalidInput):
		status = http.StatusBadRequest
	case errors.Is(err, auth.ErrEmailAlreadyUsed):
		status = http.StatusConflict
	case errors.Is(err, auth.ErrInvalidCredentials), errors.Is(err, auth.ErrUnauthorized):
		status = http.StatusUnauthorized
		message = "invalid credentials"
	}

	writeJSON(w, status, map[string]string{"error": message})
}

func writeMethodNotAllowed(w http.ResponseWriter, allowed ...string) {
	w.Header().Set("Allow", strings.Join(allowed, ", "))
	writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
