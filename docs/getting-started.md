# Getting Started

`ezdeploy` is designed to be easily set up for local development using Docker Compose.

## Prerequisites

- **Docker**: For running the services and the deployment containers.
- **Docker Compose**: For orchestrating the multi-container setup.
- **Make**: For running common development tasks (optional, but recommended).

## Quick Start

1. **Clone the Repository**:
   ```bash
   git clone https://github.com/your-username/ezdeploy.git
   cd ezdeploy
   ```

2. **Configure Environment Variables**:
   Copy the example environment file and update values if needed:
   ```bash
   cp .env.example .env
   ```

3. **Start the Development Services**:
   This will build and start the API Server, Background Worker, PostgreSQL, and the React Frontend:
   ```bash
   make dev-up
   ```
   *Note: This command runs `docker compose up -d --build`.*

4. **Access the Application**:
   - **Frontend**: `http://localhost:5173`
   - **API Server**: `http://localhost:8080`
   - **Database**: `localhost:5432`

## Development Workflow

### View Logs
To follow the logs from all services:
```bash
make dev-logs
```

### Run Tests
To execute both backend and frontend tests:
```bash
make test
```

### Stop Services
To stop and remove all development containers:
```bash
make dev-down
```

## Useful Makefile Targets

- **`make dev-up`**: Build and start services in the background.
- **`make dev-down`**: Stop and remove containers.
- **`make dev-logs`**: Follow service logs.
- **`make test`**: Run the full test suite.
- **`make fmt`**: Format the codebase (Go and TypeScript).
- **`make ci`**: Run a full CI check locally (tests, linting, build).

## Manual Setup (Optional)

If you prefer to run services individually without Docker:

### Backend
1. `cd backend`
2. `go run cmd/api/main.go`
3. `go run cmd/worker/main.go`

### Frontend
1. `cd frontend`
2. `npm install`
3. `npm run dev`
