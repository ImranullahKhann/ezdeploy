# Database Schema

`ezdeploy` uses a PostgreSQL database for metadata storage and asynchronous task management.

## Tables

### `users`
Stores user account information and hashed credentials.
- `id` (TEXT, PK): Unique user ID.
- `email` (TEXT, UNIQUE): User's email address.
- `password_hash` (TEXT): Hashed password.
- `created_at` (TIMESTAMPTZ): User creation timestamp.

### `sessions`
Manages authenticated user sessions.
- `id` (TEXT, PK): Unique session ID.
- `user_id` (TEXT, FK): References `users(id)`.
- `token_hash` (TEXT, UNIQUE): Hashed session token.
- `expires_at` (TIMESTAMPTZ): Session expiration time.
- `revoked_at` (TIMESTAMPTZ, NULL): Time when the session was manually revoked (e.g., on logout).

### `projects`
Stores metadata for deployed applications.
- `id` (TEXT, PK): Unique project ID.
- `name` (TEXT): Project name.
- `user_id` (TEXT, FK): References `users(id)`.
- `git_repo_url` (TEXT): URL to the Git repository.
- `branch` (TEXT): Default Git branch to use for deployments.
- `workload_type` (TEXT): Type of workload (e.g., `web_service`, `static_site`).

### `project_configs`
Stores detailed build and runtime configuration for projects.
- `project_id` (TEXT, PK, FK): References `projects(id)`.
- `build_cmd` (TEXT, NULL): Command to build the project.
- `start_cmd` (TEXT, NULL): Command to start the project.
- `dockerfile_path` (TEXT, NULL): Path to the Dockerfile.
- `port` (INTEGER, NULL): The port the application listens on.
- `healthcheck_path` (TEXT, NULL): The path to use for health checking.
- `env_vars` (JSONB, NULL): Map of environment variables.

### `deployments`
Tracks historical and current deployment attempts for projects.
- `id` (TEXT, PK): Unique deployment ID.
- `project_id` (TEXT, FK): References `projects(id)`.
- `git_commit_sha` (TEXT, NULL): The specific commit SHA being deployed.
- `status` (TEXT): Current deployment status (e.g., `queued`, `running`, `failed`).
- `runtime_container_id` (TEXT, NULL): ID of the running Docker container.
- `port` (INTEGER, NULL): The host port assigned to this deployment.
- `public_url` (TEXT, NULL): The public URL where the deployment can be accessed.

### `deployment_events`
Stores timestamped logs and events related to a specific deployment.
- `id` (TEXT, PK): Unique event ID.
- `deployment_id` (TEXT, FK): References `deployments(id)`.
- `event_type` (TEXT): Type of event (e.g., `build_started`).
- `message` (TEXT): Human-readable event message.
- `metadata_json` (JSONB, NULL): Additional event-specific data.

### `job_queue`
Asynchronous task management queue used by the background worker.
- `id` (TEXT, PK): Unique job ID.
- `job_type` (TEXT): Type of job (e.g., `deploy`).
- `payload` (JSONB): Job parameters (e.g., `deployment_id`).
- `status` (TEXT): Current job status (`queued`, `claimed`, `completed`, `failed`).
- `locked_until` (TIMESTAMPTZ, NULL): Lock expiration for claimed jobs.
- `error` (TEXT, NULL): Error message if the job failed.

## Relationships

- **Users** can have many **Projects**.
- **Projects** can have many **Deployments**.
- **Projects** have a one-to-one relationship with **ProjectConfigs**.
- **Deployments** can have many **DeploymentEvents**.
- **Sessions** belong to a **User**.
