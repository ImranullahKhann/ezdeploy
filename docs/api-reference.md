# API Reference

The `ezdeploy` API is a RESTful service built with Go. It uses session-based authentication through cookies.

## Base URL
The API base URL is typically `http://localhost:8080` in a development environment.

## Endpoints

### Health & Readiness
- **`GET /healthz`**: Returns the health status of the API server.
- **`GET /readyz`**: Returns the readiness status, including database connectivity.

### Authentication
- **`POST /auth/signup`**: Creates a new user account.
  - **Request Body**: `{"email": "user@example.com", "password": "password123"}`
- **`POST /auth/login`**: Authenticates a user and sets a session cookie.
  - **Request Body**: `{"email": "user@example.com", "password": "password123"}`
- **`POST /auth/logout`**: Revokes the current session and clears the session cookie.
- **`GET /auth/me`**: Returns the current user's profile information.

### Projects
- **`GET /projects`**: Lists all projects owned by the authenticated user.
- **`POST /projects`**: Creates a new project.
  - **Request Body**: `{"name": "my-app", "git_repo_url": "https://github.com/user/repo", "branch": "main", "workload_type": "web_service"}`
- **`GET /projects/:id`**: Gets details for a specific project.
- **`PUT /projects/:id`**: Updates project metadata.
- **`DELETE /projects/:id`**: Deletes a project and its associated deployments.

### Project Configuration
- **`GET /projects/:id/config`**: Gets the build and runtime configuration for a project.
- **`PUT /projects/:id/config`**: Updates the build and runtime configuration.
  - **Request Body Fields**: `build_cmd`, `start_cmd`, `dockerfile_path`, `output_dir`, `install_cmd`, `port`, `healthcheck_path`, `env_vars`.

### Deployments
- **`POST /projects/:id/deploy`**: Triggers a new deployment for a project.
  - **Request Body (Optional)**: `{"commit_sha": "...", "branch": "..."}`
- **`GET /projects/:id/deployments`**: Lists all deployments for a specific project.
- **`GET /deployments/:id`**: Gets details for a specific deployment.
- **`GET /deployments/:id/events`**: Lists all events (logs) for a specific deployment.

## Models

### User
```json
{
  "id": "string",
  "email": "string",
  "created_at": "string (ISO 8601)"
}
```

### Project
```json
{
  "id": "string",
  "name": "string",
  "git_repo_url": "string",
  "branch": "string",
  "workload_type": "web_service | static_site",
  "created_at": "string",
  "updated_at": "string"
}
```

### ProjectConfig
```json
{
  "project_id": "string",
  "build_cmd": "string | null",
  "start_cmd": "string | null",
  "dockerfile_path": "string | null",
  "port": "number | null",
  "healthcheck_path": "string | null",
  "env_vars": "object | null"
}
```

### Deployment
```json
{
  "id": "string",
  "project_id": "string",
  "status": "queued | building | build_failed | deploying | running | failed",
  "public_url": "string | null",
  "created_at": "string",
  "started_at": "string | null",
  "finished_at": "string | null"
}
```
