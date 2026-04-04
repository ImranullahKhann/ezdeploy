# Deployment Flow

`ezdeploy` manages the entire deployment lifecycle, from source code to a running container.

## Lifecycle States

A deployment transitions through several states:

- **`queued`**: The deployment request is waiting to be processed by the worker.
- **`building`**: The background worker is cloning the repository and building a Docker image.
- **`build_failed`**: The build process failed (e.g., due to a compilation error or a missing Dockerfile).
- **`deploying`**: The worker is allocating resources and starting the container.
- **`running`**: The container is started, and its health check is passing.
- **`failed`**: The container failed to start or failed the health check.
- **`stopped`**: The deployment has been stopped by the user.

## Detailed Steps

1. **Job Claim**: The worker polls the `job_queue` and claims a "deploy" job using `ClaimJob`. This locks the job to prevent other workers from processing it simultaneously.
2. **Setup & Build**:
    - The worker retrieves project and deployment metadata.
    - It updates the deployment status to `building`.
    - It uses the `BuildService` to clone the Git repository and build a Docker image using the provided `dockerfile_path` or a default.
3. **Allocation**:
    - Once the build is successful, the deployment status is updated to `deploying`.
    - The worker allocates a host port for the container using `AllocatePort` within the configured range (e.g., 9000-10000).
4. **Runtime Management**:
    - The worker uses the `RuntimeService` to start the Docker container.
    - Containers are named following a pattern: `ezd-<project-short-id>-<deployment-short-id>`.
    - Host and container ports are mapped according to the project configuration.
5. **Health Verification**:
    - The worker periodically polls the container's health check path (e.g., `/healthz` or `/`) using `PollHealth`.
    - It waits for a configured timeout (e.g., 1 minute) for the container to become healthy.
6. **Final Status**:
    - If the health check passes, the status is updated to `running`.
    - If the health check fails or the build fails, the status is updated to `failed` or `build_failed`.

## Deployment Events

Throughout the lifecycle, the worker records detailed events in the `deployment_events` table. These events provide a real-time log of the deployment process, including:

- `build_started`
- `build_finished`
- `container_started`
- `deployment_ready`
- `deployment_failed`

These events are available through the API for the frontend to display progress logs.
