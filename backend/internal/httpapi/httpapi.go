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
	"ezdeploy/backend/internal/deployment"
	"ezdeploy/backend/internal/middleware"
	"ezdeploy/backend/internal/project"
)

type Handler struct {
	pool           *pgxpool.Pool
	authService    *auth.Service
	projectService *project.Service
	deployService  *deployment.Service
}

func New(pool *pgxpool.Pool, authService *auth.Service) http.Handler {
	h := &Handler{pool: pool, authService: authService}
	
	projectService, err := project.New(pool)
	if err == nil {
		h.projectService = projectService
	}

	deployService, err := deployment.New(pool)
	if err == nil {
		h.deployService = deployService
	}
	
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", h.handleHealth)
	mux.HandleFunc("/readyz", h.handleReady)
	if authService != nil {
		mux.HandleFunc("/auth/signup", h.handleSignup)
		mux.HandleFunc("/auth/login", h.handleLogin)
		mux.Handle("/auth/me", middleware.RequireAuth(authService, http.HandlerFunc(h.handleMe)))
		mux.Handle("/auth/logout", middleware.RequireAuth(authService, http.HandlerFunc(h.handleLogout)))
		
		if h.projectService != nil {
			mux.Handle("/projects", middleware.RequireAuth(authService, http.HandlerFunc(h.handleProjects)))
			mux.Handle("/projects/", middleware.RequireAuth(authService, http.HandlerFunc(h.handleProjectByID)))
		}

		if h.deployService != nil {
			mux.Handle("/deployments/", middleware.RequireAuth(authService, http.HandlerFunc(h.handleDeploymentByID)))
		}
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

type createProjectRequest struct {
	Name         string `json:"name"`
	GitRepoURL   string `json:"git_repo_url"`
	Branch       string `json:"branch"`
	WorkloadType string `json:"workload_type"`
}

type updateProjectRequest struct {
	Name         string `json:"name"`
	GitRepoURL   string `json:"git_repo_url"`
	Branch       string `json:"branch"`
	WorkloadType string `json:"workload_type"`
}

type projectResponse struct {
	Project project.Project `json:"project"`
}

type projectListResponse struct {
	Projects []project.Project `json:"projects"`
}

type projectConfigRequest struct {
	BuildCmd        *string                `json:"build_cmd,omitempty"`
	StartCmd        *string                `json:"start_cmd,omitempty"`
	DockerfilePath  *string                `json:"dockerfile_path,omitempty"`
	OutputDir       *string                `json:"output_dir,omitempty"`
	InstallCmd      *string                `json:"install_cmd,omitempty"`
	Port            *int                   `json:"port,omitempty"`
	HealthcheckPath *string                `json:"healthcheck_path,omitempty"`
	EnvVars         map[string]interface{} `json:"env_vars,omitempty"`
}

type projectConfigResponse struct {
	Config project.ProjectConfig `json:"config"`
}

func (h *Handler) handleProjects(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.handleListProjects(w, r, user.ID)
	case http.MethodPost:
		h.handleCreateProject(w, r, user.ID)
	default:
		writeMethodNotAllowed(w, http.MethodGet, http.MethodPost)
	}
}

func (h *Handler) handleListProjects(w http.ResponseWriter, r *http.Request, userID string) {
	projects, err := h.projectService.List(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, projectListResponse{Projects: projects})
}

func (h *Handler) handleCreateProject(w http.ResponseWriter, r *http.Request, userID string) {
	defer r.Body.Close()

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "read request body"})
		return
	}

	var req createProjectRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	proj, err := h.projectService.Create(r.Context(), userID, req.Name, req.GitRepoURL, req.Branch, req.WorkloadType)
	if err != nil {
		handleProjectError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, projectResponse{Project: proj})
}

func (h *Handler) handleProjectByID(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/projects/")
	path = strings.TrimSuffix(path, "/")
	if path == "" {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}

	parts := strings.Split(path, "/")
	projectID := parts[0]

	if len(parts) == 2 {
		switch parts[1] {
		case "config":
			h.handleProjectConfig(w, r, user.ID, projectID)
			return
		case "deploy":
			if r.Method == http.MethodPost {
				h.handleDeployProject(w, r, user.ID, projectID)
				return
			}
			writeMethodNotAllowed(w, http.MethodPost)
			return
		case "deployments":
			if r.Method == http.MethodGet {
				h.handleListDeployments(w, r, user.ID, projectID)
				return
			}
			writeMethodNotAllowed(w, http.MethodGet)
			return
		default:
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}
	}

	if len(parts) > 2 {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.handleGetProject(w, r, user.ID, projectID)
	case http.MethodPut:
		h.handleUpdateProject(w, r, user.ID, projectID)
	case http.MethodDelete:
		h.handleDeleteProject(w, r, user.ID, projectID)
	default:
		writeMethodNotAllowed(w, http.MethodGet, http.MethodPut, http.MethodDelete)
	}
}

func (h *Handler) handleGetProject(w http.ResponseWriter, r *http.Request, userID, projectID string) {
	proj, err := h.projectService.GetByID(r.Context(), userID, projectID)
	if err != nil {
		handleProjectError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, projectResponse{Project: proj})
}

func (h *Handler) handleUpdateProject(w http.ResponseWriter, r *http.Request, userID, projectID string) {
	defer r.Body.Close()

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "read request body"})
		return
	}

	var req updateProjectRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	proj, err := h.projectService.Update(r.Context(), userID, projectID, req.Name, req.GitRepoURL, req.Branch, req.WorkloadType)
	if err != nil {
		handleProjectError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, projectResponse{Project: proj})
}

func (h *Handler) handleDeleteProject(w http.ResponseWriter, r *http.Request, userID, projectID string) {
	if err := h.projectService.Delete(r.Context(), userID, projectID); err != nil {
		handleProjectError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleProjectConfig(w http.ResponseWriter, r *http.Request, userID, projectID string) {
	switch r.Method {
	case http.MethodGet:
		h.handleGetConfig(w, r, userID, projectID)
	case http.MethodPut:
		h.handleUpdateConfig(w, r, userID, projectID)
	default:
		writeMethodNotAllowed(w, http.MethodGet, http.MethodPut)
	}
}

func (h *Handler) handleGetConfig(w http.ResponseWriter, r *http.Request, userID, projectID string) {
	config, err := h.projectService.GetConfig(r.Context(), userID, projectID)
	if err != nil {
		handleProjectError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, projectConfigResponse{Config: config})
}

func (h *Handler) handleUpdateConfig(w http.ResponseWriter, r *http.Request, userID, projectID string) {
	defer r.Body.Close()

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "read request body"})
		return
	}

	var req projectConfigRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	config := project.ProjectConfig{
		ProjectID:       projectID,
		BuildCmd:        req.BuildCmd,
		StartCmd:        req.StartCmd,
		DockerfilePath:  req.DockerfilePath,
		OutputDir:       req.OutputDir,
		InstallCmd:      req.InstallCmd,
		Port:            req.Port,
		HealthcheckPath: req.HealthcheckPath,
		EnvVars:         req.EnvVars,
	}

	config, err = h.projectService.UpdateConfig(r.Context(), userID, projectID, config)
	if err != nil {
		handleProjectError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, projectConfigResponse{Config: config})
}

type deployProjectRequest struct {
	CommitSHA string `json:"commit_sha"`
	Branch    string `json:"branch"`
}

type deploymentResponse struct {
	Deployment deployment.Deployment `json:"deployment"`
}

type deploymentListResponse struct {
	Deployments []deployment.Deployment `json:"deployments"`
}

type eventListResponse struct {
	Events []deployment.DeploymentEvent `json:"events"`
}

func (h *Handler) handleDeployProject(w http.ResponseWriter, r *http.Request, userID, projectID string) {
	// Verify project ownership
	proj, err := h.projectService.GetByID(r.Context(), userID, projectID)
	if err != nil {
		handleProjectError(w, err)
		return
	}

	var req deployProjectRequest
	if r.Body != http.NoBody {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
			return
		}
	}

	branch := req.Branch
	if branch == "" {
		branch = proj.Branch
	}

	var commitSHAPtr *string
	if req.CommitSHA != "" {
		commitSHAPtr = &req.CommitSHA
	}

	// Create deployment record
	dep, err := h.deployService.Create(r.Context(), projectID, userID, commitSHAPtr, &branch)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// Enqueue job
	_, err = h.deployService.EnqueueJob(r.Context(), "deploy", map[string]any{
		"deployment_id": dep.ID,
		"project_id":    projectID,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusAccepted, deploymentResponse{Deployment: dep})
}

func (h *Handler) handleListDeployments(w http.ResponseWriter, r *http.Request, userID, projectID string) {
	// Verify project ownership
	if _, err := h.projectService.GetByID(r.Context(), userID, projectID); err != nil {
		handleProjectError(w, err)
		return
	}

	deployments, err := h.deployService.ListByProject(r.Context(), projectID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, deploymentListResponse{Deployments: deployments})
}

func (h *Handler) handleDeploymentByID(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/deployments/")
	path = strings.TrimSuffix(path, "/")
	if path == "" {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}

	parts := strings.Split(path, "/")
	deploymentID := parts[0]

	if len(parts) == 2 && parts[1] == "events" {
		if r.Method == http.MethodGet {
			h.handleListDeploymentEvents(w, r, user.ID, deploymentID)
			return
		}
		writeMethodNotAllowed(w, http.MethodGet)
		return
	}

	if len(parts) > 1 {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}

	if r.Method == http.MethodGet {
		h.handleGetDeployment(w, r, user.ID, deploymentID)
		return
	}

	writeMethodNotAllowed(w, http.MethodGet)
}

func (h *Handler) handleGetDeployment(w http.ResponseWriter, r *http.Request, userID, deploymentID string) {
	dep, err := h.deployService.GetByID(r.Context(), deploymentID)
	if err != nil {
		if errors.Is(err, deployment.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// Verify project ownership
	if _, err := h.projectService.GetByID(r.Context(), userID, dep.ProjectID); err != nil {
		handleProjectError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, deploymentResponse{Deployment: dep})
}

func (h *Handler) handleListDeploymentEvents(w http.ResponseWriter, r *http.Request, userID, deploymentID string) {
	dep, err := h.deployService.GetByID(r.Context(), deploymentID)
	if err != nil {
		if errors.Is(err, deployment.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// Verify project ownership
	if _, err := h.projectService.GetByID(r.Context(), userID, dep.ProjectID); err != nil {
		handleProjectError(w, err)
		return
	}

	events, err := h.deployService.ListEvents(r.Context(), deploymentID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, eventListResponse{Events: events})
}


func handleProjectError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	message := err.Error()

	switch {
	case errors.Is(err, project.ErrInvalidInput):
		status = http.StatusBadRequest
	case errors.Is(err, project.ErrNotFound), errors.Is(err, project.ErrConfigNotFound):
		status = http.StatusNotFound
		message = "not found"
	case errors.Is(err, project.ErrUnauthorized):
		status = http.StatusForbidden
		message = "forbidden"
	}

	writeJSON(w, status, map[string]string{"error": message})
}

