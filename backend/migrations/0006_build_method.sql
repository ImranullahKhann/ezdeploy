-- Add build_method field to project_configs table
-- This allows users to choose between building from a Dockerfile or from build/start commands
ALTER TABLE project_configs ADD COLUMN IF NOT EXISTS build_method TEXT NOT NULL DEFAULT 'dockerfile';

-- Possible values: 'dockerfile' or 'buildpack'
-- 'dockerfile': Build using Dockerfile (existing behavior)
-- 'buildpack': Build using build_cmd and start_cmd (new behavior)
