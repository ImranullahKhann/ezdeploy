package auth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

const SessionCookieName = "ezdeploy_session"

type Service struct {
	pool            *pgxpool.Pool
	sessionSecret   []byte
	sessionDuration time.Duration
	secureCookie    bool
}

func New(pool *pgxpool.Pool, sessionSecret, appEnv string) (*Service, error) {
	if pool == nil {
		return nil, fmt.Errorf("database pool is nil")
	}
	if strings.TrimSpace(sessionSecret) == "" {
		return nil, fmt.Errorf("session secret is required")
	}

	return &Service{
		pool:            pool,
		sessionSecret:   []byte(sessionSecret),
		sessionDuration: 7 * 24 * time.Hour,
		secureCookie:    strings.EqualFold(appEnv, "production"),
	}, nil
}

func (s *Service) Signup(ctx context.Context, email, password string) (User, *http.Cookie, error) {
	normalizedEmail, err := normalizeEmail(email)
	if err != nil {
		return User{}, nil, err
	}
	if err := validatePassword(password); err != nil {
		return User{}, nil, err
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return User{}, nil, fmt.Errorf("hash password: %w", err)
	}

	user, err := s.createUser(ctx, normalizedEmail, string(passwordHash))
	if err != nil {
		return User{}, nil, err
	}

	cookie, err := s.issueSession(ctx, user.ID)
	if err != nil {
		return User{}, nil, err
	}

	return user, cookie, nil
}

func (s *Service) Login(ctx context.Context, email, password string) (User, *http.Cookie, error) {
	normalizedEmail, err := normalizeEmail(email)
	if err != nil {
		return User{}, nil, err
	}

	account, err := s.accountByEmail(ctx, normalizedEmail)
	if err != nil {
		if errors.Is(err, ErrUnauthorized) {
			return User{}, nil, ErrInvalidCredentials
		}
		return User{}, nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(account.PasswordHash), []byte(password)); err != nil {
		return User{}, nil, ErrInvalidCredentials
	}

	cookie, err := s.issueSession(ctx, account.ID)
	if err != nil {
		return User{}, nil, err
	}

	return account.User, cookie, nil
}

func (s *Service) Authenticate(ctx context.Context, cookieValue string) (User, error) {
	claims, err := s.parseSessionCookie(cookieValue)
	if err != nil {
		return User{}, ErrUnauthorized
	}

	user, err := s.userByTokenHash(ctx, claims.tokenHash)
	if err != nil {
		return User{}, ErrUnauthorized
	}

	return user, nil
}

func (s *Service) Revoke(ctx context.Context, cookieValue string) error {
	claims, err := s.parseSessionCookie(cookieValue)
	if err != nil {
		return ErrUnauthorized
	}

	commandTag, err := s.pool.Exec(ctx, `UPDATE sessions SET revoked_at = NOW() WHERE token_hash = $1 AND revoked_at IS NULL`, claims.tokenHash)
	if err != nil {
		return fmt.Errorf("revoke session: %w", err)
	}
	if commandTag.RowsAffected() == 0 {
		return ErrUnauthorized
	}

	return nil
}

func (s *Service) ClearCookie() *http.Cookie {
	return &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		Secure:   s.secureCookie,
		SameSite: http.SameSiteLaxMode,
	}
}

func (s *Service) createUser(ctx context.Context, email, passwordHash string) (User, error) {
	userID, err := newID("usr")
	if err != nil {
		return User{}, err
	}

	var user User
	if err := s.pool.QueryRow(ctx, `
		INSERT INTO users (id, email, password_hash)
		VALUES ($1, $2, $3)
		RETURNING id, email, created_at, updated_at
	`, userID, email, passwordHash).Scan(&user.ID, &user.Email, &user.CreatedAt, &user.UpdatedAt); err != nil {
		if isUniqueViolation(err) {
			return User{}, ErrEmailAlreadyUsed
		}
		return User{}, fmt.Errorf("create user: %w", err)
	}

	return user, nil
}

func (s *Service) accountByEmail(ctx context.Context, email string) (accountRecord, error) {
	var account accountRecord
	if err := s.pool.QueryRow(ctx, `
		SELECT id, email, password_hash, created_at, updated_at
		FROM users
		WHERE email = $1
	`, email).Scan(&account.ID, &account.Email, &account.PasswordHash, &account.CreatedAt, &account.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return accountRecord{}, ErrUnauthorized
		}
		return accountRecord{}, fmt.Errorf("query user by email: %w", err)
	}

	return account, nil
}

func (s *Service) userByTokenHash(ctx context.Context, tokenHash string) (User, error) {
	var session sessionRecord
	var user User
	if err := s.pool.QueryRow(ctx, `
		SELECT s.id, s.user_id, s.token_hash, s.expires_at, s.revoked_at,
		       u.id, u.email, u.created_at, u.updated_at
		FROM sessions s
		JOIN users u ON u.id = s.user_id
		WHERE s.token_hash = $1
	`, tokenHash).Scan(&session.ID, &session.UserID, &session.TokenHash, &session.ExpiresAt, &session.RevokedAt, &user.ID, &user.Email, &user.CreatedAt, &user.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrUnauthorized
		}
		return User{}, fmt.Errorf("query session: %w", err)
	}

	if session.RevokedAt != nil || time.Now().UTC().After(session.ExpiresAt) {
		return User{}, ErrUnauthorized
	}

	return user, nil
}

func (s *Service) issueSession(ctx context.Context, userID string) (*http.Cookie, error) {
	rawToken, err := randomToken(32)
	if err != nil {
		return nil, err
	}

	tokenHash := hashToken(rawToken)
	sessionID, err := newID("ses")
	if err != nil {
		return nil, err
	}

	expiresAt := time.Now().UTC().Add(s.sessionDuration)
	if _, err := s.pool.Exec(ctx, `
		INSERT INTO sessions (id, user_id, token_hash, expires_at)
		VALUES ($1, $2, $3, $4)
	`, sessionID, userID, tokenHash, expiresAt); err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	return &http.Cookie{
		Name:     SessionCookieName,
		Value:    encodeSessionCookie(rawToken, s.sessionSecret),
		Path:     "/",
		Expires:  expiresAt,
		MaxAge:   int(s.sessionDuration.Seconds()),
		HttpOnly: true,
		Secure:   s.secureCookie,
		SameSite: http.SameSiteLaxMode,
	}, nil
}

type sessionClaims struct {
	tokenHash string
}

func (s *Service) parseSessionCookie(cookieValue string) (sessionClaims, error) {
	parts := strings.Split(cookieValue, ".")
	if len(parts) != 2 {
		return sessionClaims{}, ErrUnauthorized
	}

	rawToken, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil || len(rawToken) == 0 {
		return sessionClaims{}, ErrUnauthorized
	}

	signature, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return sessionClaims{}, ErrUnauthorized
	}

	expected := signToken(rawToken, s.sessionSecret)
	if !hmac.Equal(signature, expected) {
		return sessionClaims{}, ErrUnauthorized
	}

	return sessionClaims{tokenHash: hashToken(rawToken)}, nil
}

func normalizeEmail(email string) (string, error) {
	trimmed := strings.TrimSpace(email)
	if trimmed == "" {
		return "", ErrInvalidInput
	}

	addr, err := mail.ParseAddress(trimmed)
	if err != nil {
		return "", ErrInvalidInput
	}

	normalized := strings.ToLower(strings.TrimSpace(addr.Address))
	if normalized == "" || !strings.Contains(normalized, "@") {
		return "", ErrInvalidInput
	}

	return normalized, nil
}

func validatePassword(password string) error {
	if len(password) < 8 {
		return ErrInvalidInput
	}
	return nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func randomToken(length int) ([]byte, error) {
	token := make([]byte, length)
	if _, err := rand.Read(token); err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}
	return token, nil
}

func hashToken(token []byte) string {
	sum := sha256.Sum256(token)
	return hex.EncodeToString(sum[:])
}

func encodeSessionCookie(rawToken, secret []byte) string {
	return base64.RawURLEncoding.EncodeToString(rawToken) + "." + base64.RawURLEncoding.EncodeToString(signToken(rawToken, secret))
}

func signToken(rawToken, secret []byte) []byte {
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write(rawToken)
	return mac.Sum(nil)
}

func newID(prefix string) (string, error) {
	raw, err := randomToken(16)
	if err != nil {
		return "", err
	}
	return prefix + "_" + hex.EncodeToString(raw), nil
}
