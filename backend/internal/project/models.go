package project

import "time"

type Project struct {
	ID           string     `json:"id"`
	Name         string     `json:"name"`
	UserID       string     `json:"user_id"`
	GitRepoURL   string     `json:"git_repo_url"`
	Branch       string     `json:"branch"`
	WorkloadType string     `json:"workload_type"`
	Slug         *string    `json:"slug,omitempty"`
	DeletedAt    *time.Time `json:"deleted_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type ProjectConfig struct {
	ProjectID       string                 `json:"project_id"`
	BuildMethod     string                 `json:"build_method"`
	BuildCmd        *string                `json:"build_cmd,omitempty"`
	StartCmd        *string                `json:"start_cmd,omitempty"`
	DockerfilePath  *string                `json:"dockerfile_path,omitempty"`
	OutputDir       *string                `json:"output_dir,omitempty"`
	InstallCmd      *string                `json:"install_cmd,omitempty"`
	Port            *int                   `json:"port,omitempty"`
	HealthcheckPath *string                `json:"healthcheck_path,omitempty"`
	EnvVars         map[string]interface{} `json:"env_vars,omitempty"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
}

type projectRecord struct {
	Project
}

type configRecord struct {
	ProjectConfig
}
