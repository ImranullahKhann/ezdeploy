# ezdeploy

ezdeploy is a portfolio-grade hosting platform for web services and static sites.

## Repository layout

- `backend/` Go API and worker foundation.
- `frontend/` React application.
- `infra/` local development and deployment infrastructure.
- `docs/` design notes and supporting documentation.
- `scripts/` developer helper scripts.
- `.github/` CI and repository automation.

## Local development

Run the stack with:

```sh
make dev-up
```

The local stack uses these default ports:

- Frontend: http://localhost:5173
- Backend API: http://localhost:8080
- PostgreSQL: localhost:5432

Set `SESSION_SECRET` in your environment before starting the backend; the sample `.env.example` includes a local default value.
