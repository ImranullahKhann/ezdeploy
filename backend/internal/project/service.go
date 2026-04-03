package project

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) (*Service, error) {
	if pool == nil {
		return nil, fmt.Errorf("database pool is nil")
	}
	return &Service{pool: pool}, nil
}

func (s *Service) Create(ctx context.Context, userID, name, gitRepoURL, branch, workloadType string) (Project, error) {
	name = strings.TrimSpace(name)
	gitRepoURL = strings.TrimSpace(gitRepoURL)
	branch = strings.TrimSpace(branch)
	workloadType = strings.TrimSpace(workloadType)

	if name == "" {
		return Project{}, fmt.Errorf("%w: name is required", ErrInvalidInput)
	}
	if gitRepoURL == "" {
		return Project{}, fmt.Errorf("%w: git_repo_url is required", ErrInvalidInput)
	}
	if branch == "" {
		branch = "main"
	}
	if workloadType == "" {
		workloadType = "web_service"
	}

	projectID, err := newID("prj")
	if err != nil {
		return Project{}, fmt.Errorf("generate project id: %w", err)
	}

	query := `
		INSERT INTO projects (id, name, user_id, git_repo_url, branch, workload_type)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, name, user_id, git_repo_url, branch, workload_type, slug, deleted_at, created_at, updated_at
	`

	var p Project
	err = s.pool.QueryRow(ctx, query, projectID, name, userID, gitRepoURL, branch, workloadType).Scan(
		&p.ID, &p.Name, &p.UserID, &p.GitRepoURL, &p.Branch, &p.WorkloadType, &p.Slug, &p.DeletedAt, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return Project{}, fmt.Errorf("create project: %w", err)
	}

	return p, nil
}

func (s *Service) List(ctx context.Context, userID string) ([]Project, error) {
	query := `
		SELECT id, name, user_id, git_repo_url, branch, workload_type, slug, deleted_at, created_at, updated_at
		FROM projects
		WHERE user_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC
	`

	rows, err := s.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	defer rows.Close()

	var projects []Project
	for rows.Next() {
		var p Project
		if err := rows.Scan(&p.ID, &p.Name, &p.UserID, &p.GitRepoURL, &p.Branch, &p.WorkloadType, &p.Slug, &p.DeletedAt, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan project: %w", err)
		}
		projects = append(projects, p)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate projects: %w", err)
	}

	if projects == nil {
		projects = []Project{}
	}

	return projects, nil
}

func (s *Service) GetByID(ctx context.Context, userID, projectID string) (Project, error) {
	query := `
		SELECT id, name, user_id, git_repo_url, branch, workload_type, slug, deleted_at, created_at, updated_at
		FROM projects
		WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL
	`

	var p Project
	err := s.pool.QueryRow(ctx, query, projectID, userID).Scan(
		&p.ID, &p.Name, &p.UserID, &p.GitRepoURL, &p.Branch, &p.WorkloadType, &p.Slug, &p.DeletedAt, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Project{}, ErrNotFound
		}
		return Project{}, fmt.Errorf("get project: %w", err)
	}

	return p, nil
}

func (s *Service) Update(ctx context.Context, userID, projectID, name, gitRepoURL, branch, workloadType string) (Project, error) {
	name = strings.TrimSpace(name)
	gitRepoURL = strings.TrimSpace(gitRepoURL)
	branch = strings.TrimSpace(branch)
	workloadType = strings.TrimSpace(workloadType)

	if name == "" {
		return Project{}, fmt.Errorf("%w: name is required", ErrInvalidInput)
	}
	if gitRepoURL == "" {
		return Project{}, fmt.Errorf("%w: git_repo_url is required", ErrInvalidInput)
	}
	if branch == "" {
		return Project{}, fmt.Errorf("%w: branch is required", ErrInvalidInput)
	}
	if workloadType == "" {
		return Project{}, fmt.Errorf("%w: workload_type is required", ErrInvalidInput)
	}

	query := `
		UPDATE projects
		SET name = $3, git_repo_url = $4, branch = $5, workload_type = $6, updated_at = NOW()
		WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL
		RETURNING id, name, user_id, git_repo_url, branch, workload_type, slug, deleted_at, created_at, updated_at
	`

	var p Project
	err := s.pool.QueryRow(ctx, query, projectID, userID, name, gitRepoURL, branch, workloadType).Scan(
		&p.ID, &p.Name, &p.UserID, &p.GitRepoURL, &p.Branch, &p.WorkloadType, &p.Slug, &p.DeletedAt, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Project{}, ErrNotFound
		}
		return Project{}, fmt.Errorf("update project: %w", err)
	}

	return p, nil
}

func (s *Service) Delete(ctx context.Context, userID, projectID string) error {
	// Soft delete for the MVP
	query := `UPDATE projects SET deleted_at = NOW() WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL`

	result, err := s.pool.Exec(ctx, query, projectID, userID)
	if err != nil {
		return fmt.Errorf("delete project: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

func (s *Service) UpdateConfig(ctx context.Context, userID, projectID string, config ProjectConfig) (ProjectConfig, error) {
	if err := s.verifyOwnership(ctx, userID, projectID); err != nil {
		return ProjectConfig{}, err
	}

	if config.Port != nil && (*config.Port < 1 || *config.Port > 65535) {
		return ProjectConfig{}, fmt.Errorf("%w: port must be between 1 and 65535", ErrInvalidInput)
	}

	envVarsJSON, err := json.Marshal(config.EnvVars)
	if err != nil {
		return ProjectConfig{}, fmt.Errorf("marshal env vars: %w", err)
	}

	query := `
		INSERT INTO project_configs (
			project_id, build_cmd, start_cmd, dockerfile_path, output_dir,
			install_cmd, port, healthcheck_path, env_vars
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (project_id) DO UPDATE SET
			build_cmd = EXCLUDED.build_cmd,
			start_cmd = EXCLUDED.start_cmd,
			dockerfile_path = EXCLUDED.dockerfile_path,
			output_dir = EXCLUDED.output_dir,
			install_cmd = EXCLUDED.install_cmd,
			port = EXCLUDED.port,
			healthcheck_path = EXCLUDED.healthcheck_path,
			env_vars = EXCLUDED.env_vars,
			updated_at = NOW()
		RETURNING project_id, build_cmd, start_cmd, dockerfile_path, output_dir,
			install_cmd, port, healthcheck_path, env_vars, created_at, updated_at
	`

	var c ProjectConfig
	var envVarsRaw []byte
	err = s.pool.QueryRow(ctx, query,
		projectID, config.BuildCmd, config.StartCmd, config.DockerfilePath,
		config.OutputDir, config.InstallCmd, config.Port, config.HealthcheckPath, envVarsJSON,
	).Scan(
		&c.ProjectID, &c.BuildCmd, &c.StartCmd, &c.DockerfilePath, &c.OutputDir,
		&c.InstallCmd, &c.Port, &c.HealthcheckPath, &envVarsRaw, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return ProjectConfig{}, fmt.Errorf("upsert config: %w", err)
	}

	if len(envVarsRaw) > 0 {
		if err := json.Unmarshal(envVarsRaw, &c.EnvVars); err != nil {
			return ProjectConfig{}, fmt.Errorf("unmarshal env vars: %w", err)
		}
	}

	return c, nil
}

func (s *Service) GetConfig(ctx context.Context, userID, projectID string) (ProjectConfig, error) {
	if err := s.verifyOwnership(ctx, userID, projectID); err != nil {
		return ProjectConfig{}, err
	}

	query := `
		SELECT project_id, build_cmd, start_cmd, dockerfile_path, output_dir,
			install_cmd, port, healthcheck_path, env_vars, created_at, updated_at
		FROM project_configs
		WHERE project_id = $1
	`

	var c ProjectConfig
	var envVarsRaw []byte
	err := s.pool.QueryRow(ctx, query, projectID).Scan(
		&c.ProjectID, &c.BuildCmd, &c.StartCmd, &c.DockerfilePath, &c.OutputDir,
		&c.InstallCmd, &c.Port, &c.HealthcheckPath, &envVarsRaw, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ProjectConfig{ProjectID: projectID}, nil
		}
		return ProjectConfig{}, fmt.Errorf("get config: %w", err)
	}

	if len(envVarsRaw) > 0 {
		if err := json.Unmarshal(envVarsRaw, &c.EnvVars); err != nil {
			return ProjectConfig{}, fmt.Errorf("unmarshal env vars: %w", err)
		}
	}

	return c, nil
}

func (s *Service) GetByIDInternal(ctx context.Context, projectID string) (Project, error) {
	query := `
		SELECT id, name, user_id, git_repo_url, branch, workload_type, slug, deleted_at, created_at, updated_at
		FROM projects
		WHERE id = $1 AND deleted_at IS NULL
	`

	var p Project
	err := s.pool.QueryRow(ctx, query, projectID).Scan(
		&p.ID, &p.Name, &p.UserID, &p.GitRepoURL, &p.Branch, &p.WorkloadType, &p.Slug, &p.DeletedAt, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Project{}, ErrNotFound
		}
		return Project{}, fmt.Errorf("get project internal: %w", err)
	}

	return p, nil
}

func (s *Service) GetConfigInternal(ctx context.Context, projectID string) (ProjectConfig, error) {
	query := `
		SELECT project_id, build_cmd, start_cmd, dockerfile_path, output_dir,
			install_cmd, port, healthcheck_path, env_vars, created_at, updated_at
		FROM project_configs
		WHERE project_id = $1
	`

	var c ProjectConfig
	var envVarsRaw []byte
	err := s.pool.QueryRow(ctx, query, projectID).Scan(
		&c.ProjectID, &c.BuildCmd, &c.StartCmd, &c.DockerfilePath, &c.OutputDir,
		&c.InstallCmd, &c.Port, &c.HealthcheckPath, &envVarsRaw, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ProjectConfig{ProjectID: projectID}, nil
		}
		return ProjectConfig{}, fmt.Errorf("get config internal: %w", err)
	}

	if len(envVarsRaw) > 0 {
		if err := json.Unmarshal(envVarsRaw, &c.EnvVars); err != nil {
			return ProjectConfig{}, fmt.Errorf("unmarshal env vars internal: %w", err)
		}
	}

	return c, nil
}

func (s *Service) verifyOwnership(ctx context.Context, userID, projectID string) error {
	var ownerID string
	err := s.pool.QueryRow(ctx, "SELECT user_id FROM projects WHERE id = $1 AND deleted_at IS NULL", projectID).Scan(&ownerID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("check project ownership: %w", err)
	}

	if ownerID != userID {
		return ErrUnauthorized
	}

	return nil
}

func newID(prefix string) (string, error) {
	raw, err := randomToken(16)
	if err != nil {
		return "", err
	}
	return prefix + "_" + hex.EncodeToString(raw), nil
}

func randomToken(length int) ([]byte, error) {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}
	return b, nil
}
