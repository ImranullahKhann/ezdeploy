# Architecture Overview

`ezdeploy` is a container-based deployment platform that allows users to deploy web services and static sites directly from Git repositories.

## System Components

- **API Server (Go)**: A RESTful API built with Go that handles user authentication, project management, and deployment requests. It uses a session-based authentication mechanism.
- **Background Worker (Go)**: A dedicated service that polls the database for pending deployment jobs. It handles the end-to-end deployment lifecycle, including cloning repositories, building Docker images, and managing container runtimes.
- **PostgreSQL Database**: The central data store for user metadata, project configurations, deployment history, and the asynchronous job queue.
- **Docker Engine**: Used by the background worker to build images and run deployment containers. The worker communicates with the Docker daemon via a Unix socket.

## Data Flow

1. **User Interaction**: Users interact with the platform through the React-based frontend.
2. **API Requests**: The frontend sends requests to the API Server for operations like creating projects or triggering deployments.
3. **Job Queueing**: When a deployment is triggered, the API Server creates a deployment record and enqueues a "deploy" job in the `job_queue` table.
4. **Asynchronous Processing**: The Background Worker polls the `job_queue`. When it claims a job, it starts the deployment process.
5. **Build & Deploy**: The worker clones the Git repo, builds a Docker image, allocates a host port, and starts a container on the host machine.
6. **Health Monitoring**: The worker monitors the health of the container. Once healthy, the deployment status is updated to `running`.

## Security

- **Authentication**: Session-based authentication using cookies.
- **Isolation**: Each deployment runs in its own Docker container.
- **Worker Privileges**: The worker requires access to the Docker socket (`/var/run/docker.sock`) to manage containers.

## Storage

- **Database**: Metadata for users, projects, deployments, and logs.
- **File System**: Cloned repositories and build artifacts are temporarily stored in a configurable `STORAGE_ROOT` directory.
