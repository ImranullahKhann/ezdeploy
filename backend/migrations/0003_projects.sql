CREATE TABLE IF NOT EXISTS projects (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	git_repo_url TEXT NOT NULL,
	branch TEXT NOT NULL DEFAULT 'main',
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS projects_user_id_idx ON projects (user_id);

CREATE TABLE IF NOT EXISTS project_configs (
	project_id TEXT PRIMARY KEY REFERENCES projects(id) ON DELETE CASCADE,
	build_cmd TEXT,
	start_cmd TEXT,
	dockerfile_path TEXT,
	output_dir TEXT,
	install_cmd TEXT,
	port INTEGER,
	healthcheck_path TEXT,
	env_vars JSONB,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
