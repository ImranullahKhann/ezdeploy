-- Add missing fields to projects table
ALTER TABLE projects ADD COLUMN IF NOT EXISTS workload_type TEXT NOT NULL DEFAULT 'web_service';
ALTER TABLE projects ADD COLUMN IF NOT EXISTS slug TEXT UNIQUE;
ALTER TABLE projects ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

-- Create deployments table
CREATE TABLE IF NOT EXISTS deployments (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    git_commit_sha TEXT,
    git_branch TEXT,
    status TEXT NOT NULL DEFAULT 'queued',
    source_type TEXT,
    artifact_path TEXT,
    runtime_container_id TEXT,
    public_url TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    finished_at TIMESTAMPTZ,
    created_by_user_id TEXT REFERENCES users(id)
);

CREATE INDEX IF NOT EXISTS deployments_project_id_idx ON deployments (project_id);

-- Create deployment_events table
CREATE TABLE IF NOT EXISTS deployment_events (
    id TEXT PRIMARY KEY,
    deployment_id TEXT NOT NULL REFERENCES deployments(id) ON DELETE CASCADE,
    event_type TEXT NOT NULL,
    message TEXT NOT NULL,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    metadata_json JSONB
);

CREATE INDEX IF NOT EXISTS deployment_events_deployment_id_idx ON deployment_events (deployment_id);

-- Create job_queue table
CREATE TABLE IF NOT EXISTS job_queue (
    id TEXT PRIMARY KEY,
    job_type TEXT NOT NULL,
    payload JSONB NOT NULL,
    status TEXT NOT NULL DEFAULT 'queued', -- queued, claimed, completed, failed
    locked_until TIMESTAMPTZ,
    attempts INTEGER NOT NULL DEFAULT 0,
    error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS job_queue_status_idx ON job_queue (status);
CREATE INDEX IF NOT EXISTS job_queue_locked_until_idx ON job_queue (locked_until);
