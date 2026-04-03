package deployment

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound       = errors.New("not found")
	ErrInvalidInput   = errors.New("invalid input")
	ErrInvalidStatus  = errors.New("invalid status transition")
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

func (s *Service) Create(ctx context.Context, projectID, userID string, commitSHA, branch *string) (Deployment, error) {
	deploymentID, err := newID("dep")
	if err != nil {
		return Deployment{}, fmt.Errorf("generate deployment id: %w", err)
	}

	query := `
		INSERT INTO deployments (id, project_id, git_commit_sha, git_branch, status, created_by_user_id)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, project_id, git_commit_sha, git_branch, status, source_type, artifact_path, 
		          runtime_container_id, port, public_url, created_at, started_at, finished_at, created_by_user_id
	`

	var d Deployment
	err = s.pool.QueryRow(ctx, query, deploymentID, projectID, commitSHA, branch, StatusQueued, userID).Scan(
		&d.ID, &d.ProjectID, &d.GitCommitSHA, &d.GitBranch, &d.Status, &d.SourceType, &d.ArtifactPath,
		&d.RuntimeContainerID, &d.Port, &d.PublicURL, &d.CreatedAt, &d.StartedAt, &d.FinishedAt, &d.CreatedByUserID,
	)
	if err != nil {
		return Deployment{}, fmt.Errorf("create deployment: %w", err)
	}

	// Add initial event
	if err := s.AddEvent(ctx, d.ID, "created", "Deployment created", nil); err != nil {
		// Log error but don't fail deployment creation
		fmt.Printf("failed to add deployment event: %v\n", err)
	}

	return d, nil
}

func (s *Service) GetByID(ctx context.Context, deploymentID string) (Deployment, error) {
	query := `
		SELECT id, project_id, git_commit_sha, git_branch, status, source_type, artifact_path, 
		       runtime_container_id, port, public_url, created_at, started_at, finished_at, created_by_user_id
		FROM deployments
		WHERE id = $1
	`

	var d Deployment
	err := s.pool.QueryRow(ctx, query, deploymentID).Scan(
		&d.ID, &d.ProjectID, &d.GitCommitSHA, &d.GitBranch, &d.Status, &d.SourceType, &d.ArtifactPath,
		&d.RuntimeContainerID, &d.Port, &d.PublicURL, &d.CreatedAt, &d.StartedAt, &d.FinishedAt, &d.CreatedByUserID,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Deployment{}, ErrNotFound
		}
		return Deployment{}, fmt.Errorf("get deployment: %w", err)
	}

	return d, nil
}

func (s *Service) ListByProject(ctx context.Context, projectID string) ([]Deployment, error) {
	query := `
		SELECT id, project_id, git_commit_sha, git_branch, status, source_type, artifact_path, 
		       runtime_container_id, port, public_url, created_at, started_at, finished_at, created_by_user_id
		FROM deployments
		WHERE project_id = $1
		ORDER BY created_at DESC
	`

	rows, err := s.pool.Query(ctx, query, projectID)
	if err != nil {
		return nil, fmt.Errorf("list deployments: %w", err)
	}
	defer rows.Close()

	var deployments []Deployment
	for rows.Next() {
		var d Deployment
		err := rows.Scan(
			&d.ID, &d.ProjectID, &d.GitCommitSHA, &d.GitBranch, &d.Status, &d.SourceType, &d.ArtifactPath,
			&d.RuntimeContainerID, &d.Port, &d.PublicURL, &d.CreatedAt, &d.StartedAt, &d.FinishedAt, &d.CreatedByUserID,
		)
		if err != nil {
			return nil, fmt.Errorf("scan deployment: %w", err)
		}
		deployments = append(deployments, d)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate deployments: %w", err)
	}

	if deployments == nil {
		deployments = []Deployment{}
	}

	return deployments, nil
}

func (s *Service) AllocatePort(ctx context.Context, min, max int) (int, error) {
	// Simple allocator: find first port in range not used by a running deployment
	query := `
		SELECT p.port
		FROM generate_series($1::integer, $2::integer) AS p(port)
		LEFT JOIN deployments d ON d.port = p.port AND d.status = 'running'
		WHERE d.port IS NULL
		LIMIT 1
	`
	var port int
	err := s.pool.QueryRow(ctx, query, min, max).Scan(&port)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, fmt.Errorf("no available ports in range %d-%d", min, max)
		}
		return 0, fmt.Errorf("allocate port: %w", err)
	}
	return port, nil
}

func (s *Service) UpdateMetadata(ctx context.Context, deploymentID string, containerID *string, port *int, publicURL *string) error {
	query := `
		UPDATE deployments 
		SET runtime_container_id = COALESCE($2, runtime_container_id),
		    port = COALESCE($3, port),
		    public_url = COALESCE($4, public_url)
		WHERE id = $1
	`
	_, err := s.pool.Exec(ctx, query, deploymentID, containerID, port, publicURL)
	if err != nil {
		return fmt.Errorf("update deployment metadata: %w", err)
	}
	return nil
}

func (s *Service) UpdateStatus(ctx context.Context, deploymentID string, status Status) error {
	var query string
	if status == StatusBuilding || status == StatusDeploying {
		query = `UPDATE deployments SET status = $2, started_at = COALESCE(started_at, NOW()) WHERE id = $1`
	} else if status == StatusRunning || status == StatusFailed || status == StatusBuildFailed || status == StatusStopped || status == StatusRolledBack {
		query = `UPDATE deployments SET status = $2, finished_at = NOW() WHERE id = $1`
	} else {
		query = `UPDATE deployments SET status = $2 WHERE id = $1`
	}

	_, err := s.pool.Exec(ctx, query, deploymentID, status)
	if err != nil {
		return fmt.Errorf("update deployment status: %w", err)
	}

	return nil
}

func (s *Service) AddEvent(ctx context.Context, deploymentID, eventType, message string, metadata map[string]any) error {
	eventID, err := newID("evt")
	if err != nil {
		return fmt.Errorf("generate event id: %w", err)
	}

	var metadataJSON []byte
	if metadata != nil {
		metadataJSON, err = json.Marshal(metadata)
		if err != nil {
			return fmt.Errorf("marshal metadata: %w", err)
		}
	}

	query := `
		INSERT INTO deployment_events (id, deployment_id, event_type, message, metadata_json)
		VALUES ($1, $2, $3, $4, $5)
	`

	_, err = s.pool.Exec(ctx, query, eventID, deploymentID, eventType, message, metadataJSON)
	if err != nil {
		return fmt.Errorf("insert deployment event: %w", err)
	}

	return nil
}

func (s *Service) ListEvents(ctx context.Context, deploymentID string) ([]DeploymentEvent, error) {
	query := `
		SELECT id, deployment_id, event_type, message, timestamp, metadata_json
		FROM deployment_events
		WHERE deployment_id = $1
		ORDER BY timestamp ASC
	`

	rows, err := s.pool.Query(ctx, query, deploymentID)
	if err != nil {
		return nil, fmt.Errorf("list events: %w", err)
	}
	defer rows.Close()

	var events []DeploymentEvent
	for rows.Next() {
		var e DeploymentEvent
		err := rows.Scan(&e.ID, &e.DeploymentID, &e.EventType, &e.Message, &e.Timestamp, &e.MetadataJSON)
		if err != nil {
			return nil, fmt.Errorf("scan event: %w", err)
		}
		events = append(events, e)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate events: %w", err)
	}

	if events == nil {
		events = []DeploymentEvent{}
	}

	return events, nil
}

// Job Queue implementation

func (s *Service) EnqueueJob(ctx context.Context, jobType string, payload any) (string, error) {
	jobID, err := newID("job")
	if err != nil {
		return "", fmt.Errorf("generate job id: %w", err)
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal job payload: %w", err)
	}

	query := `
		INSERT INTO job_queue (id, job_type, payload, status)
		VALUES ($1, $2, $3, $4)
	`

	_, err = s.pool.Exec(ctx, query, jobID, jobType, payloadJSON, JobStatusQueued)
	if err != nil {
		return "", fmt.Errorf("enqueue job: %w", err)
	}

	return jobID, nil
}

func (s *Service) ClaimJob(ctx context.Context, jobType string, leaseDuration time.Duration) (*Job, error) {
	query := `
		UPDATE job_queue
		SET status = $1, locked_until = $2, attempts = attempts + 1, updated_at = NOW()
		WHERE id = (
			SELECT id
			FROM job_queue
			WHERE (status = $3 OR (status = $1 AND locked_until < NOW()))
			AND job_type = $4
			ORDER BY created_at ASC
			LIMIT 1
			FOR UPDATE SKIP LOCKED
		)
		RETURNING id, job_type, payload, status, locked_until, attempts, error, created_at, updated_at
	`

	lockedUntil := time.Now().Add(leaseDuration)
	
	var j Job
	err := s.pool.QueryRow(ctx, query, JobStatusClaimed, lockedUntil, JobStatusQueued, jobType).Scan(
		&j.ID, &j.JobType, &j.Payload, &j.Status, &j.LockedUntil, &j.Attempts, &j.Error, &j.CreatedAt, &j.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // No jobs available
		}
		return nil, fmt.Errorf("claim job: %w", err)
	}

	return &j, nil
}

func (s *Service) CompleteJob(ctx context.Context, jobID string) error {
	query := `UPDATE job_queue SET status = $1, locked_until = NULL, updated_at = NOW() WHERE id = $2`
	_, err := s.pool.Exec(ctx, query, JobStatusCompleted, jobID)
	if err != nil {
		return fmt.Errorf("complete job: %w", err)
	}
	return nil
}

func (s *Service) FailJob(ctx context.Context, jobID string, reason string) error {
	query := `UPDATE job_queue SET status = $1, locked_until = NULL, error = $2, updated_at = NOW() WHERE id = $3`
	_, err := s.pool.Exec(ctx, query, JobStatusFailed, reason, jobID)
	if err != nil {
		return fmt.Errorf("fail job: %w", err)
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
