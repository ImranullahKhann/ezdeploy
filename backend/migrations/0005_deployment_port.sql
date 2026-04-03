-- Add port to deployments
ALTER TABLE deployments ADD COLUMN IF NOT EXISTS port INTEGER;
