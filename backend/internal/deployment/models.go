package deployment

import (
	"encoding/json"
	"time"
)

type Status string

const (
	StatusQueued         Status = "queued"
	StatusBuilding       Status = "building"
	StatusBuildFailed    Status = "build_failed"
	StatusDeploying      Status = "deploying"
	StatusRunning        Status = "running"
	StatusFailed         Status = "failed"
	StatusStopped        Status = "stopped"
	StatusRolledBack     Status = "rolled_back"
)

type Deployment struct {
	ID                 string     `json:"id"`
	ProjectID          string     `json:"project_id"`
	GitCommitSHA       *string    `json:"git_commit_sha,omitempty"`
	GitBranch          *string    `json:"git_branch,omitempty"`
	Status             Status     `json:"status"`
	SourceType         *string    `json:"source_type,omitempty"`
	ArtifactPath       *string    `json:"artifact_path,omitempty"`
	RuntimeContainerID *string    `json:"runtime_container_id,omitempty"`
	Port               *int       `json:"port,omitempty"`
	PublicURL          *string    `json:"public_url,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	StartedAt          *time.Time `json:"started_at,omitempty"`
	FinishedAt         *time.Time `json:"finished_at,omitempty"`
	CreatedByUserID    *string    `json:"created_by_user_id,omitempty"`
}

type DeploymentEvent struct {
	ID           string          `json:"id"`
	DeploymentID string          `json:"deployment_id"`
	EventType    string          `json:"event_type"`
	Message      string          `json:"message"`
	Timestamp    time.Time       `json:"timestamp"`
	MetadataJSON json.RawMessage `json:"metadata_json,omitempty"`
}

type JobStatus string

const (
	JobStatusQueued    JobStatus = "queued"
	JobStatusClaimed   JobStatus = "claimed"
	JobStatusCompleted JobStatus = "completed"
	JobStatusFailed    JobStatus = "failed"
)

type Job struct {
	ID          string          `json:"id"`
	JobType     string          `json:"job_type"`
	Payload     json.RawMessage `json:"payload"`
	Status      JobStatus       `json:"status"`
	LockedUntil *time.Time      `json:"locked_until,omitempty"`
	Attempts    int             `json:"attempts"`
	Error       *string         `json:"error,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}
