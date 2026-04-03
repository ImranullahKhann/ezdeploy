# Current Progress

## Built So Far

- Monorepo scaffolding, local Docker Compose workflow, and CI skeleton are in place.
- Backend foundation is implemented: config, logging, PostgreSQL access, migrations, and health/readiness endpoints.
- Auth is implemented: users and sessions tables, signup/login/logout/me routes, secure session cookies, and auth middleware.
- Project management core is implemented: projects and project_configs tables, full CRUD API for projects, configuration management API, ownership validation, and proper error handling.
- Frontend starter app is in place as a Vite + React + TypeScript shell.
- Docker Compose and backend validation have passed with the current stack.
- Frontend foundation is implemented: React Router setup, API client layer, auth context, authentication screens (signup/login), protected routes, navigation, and project management UI (list and create).
- CORS middleware is implemented: configurable allowed origins, credentials support, preflight handling, and integration with backend server.
- Deployment model and orchestration skeleton are implemented:
    - Deployments and deployment events schema and state machine.
    - Internal job queue with claiming, completion, and failure handling.
    - Worker process bootstrap with mock deployment execution.
    - Deployment trigger API with project ownership validation.
- Enhanced project management: added `workload_type` support to projects and project configs, including UI selection and badge display.

## Progress vs Build Plan

- Phase 0: complete.
- Phase 1: complete.
- Phase 2: complete.
- Phase 3: complete.
- Phase 4: complete.
- Phase 5: complete.
- Next up: Phase 6 web service deployment path.
