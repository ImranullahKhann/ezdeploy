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
- Web service deployment path is fully functional:
    - Repository cloning and checkout (Git).
    - Docker build integration with dynamic image tagging.
    - Container orchestration: automated launching, network management, and host-port mapping.
    - Internal health checking: automated polling of container endpoints over user-defined bridge networks.
    - Dynamic port allocation system with PostgreSQL-backed reservation.
    - Sanitized container naming for Docker DNS compatibility.
- Deployment Orchestration UI is implemented:
    - Dashboard with project-level status summaries and live indicators.
    - Project Detail page with comprehensive operational view.
    - Real-time deployment status polling and event history visualization.
    - Configuration management for Web Services and Static Sites (build/start commands, ports, paths).
    - Secrets management (Environment Variables) with masked inputs.
    - One-click deployment triggering and direct access to live URLs.
- Flexible build system for web services:
    - Build method selection: Dockerfile or Build Commands (buildpack-style).
    - Dockerfile method: uses existing Dockerfile in repository (default behavior).
    - Build Commands method: auto-generates Dockerfile from install/build/start commands.
    - Automatic base image detection from project structure (Node.js, Python, Go, Ruby).
    - Dynamic Dockerfile generation with support for multi-stage builds, environment variables, and custom commands.
    - UI with radio button selection and conditional field rendering based on build method.
- Static site deployment path is fully functional:
    - Static build pipeline: automated `npm install`, `npm run build` (or custom commands) and output directory capture.
    - Immutable artifact storage for static sites.
    - Atomic deployment switching for static sites.
    - Built-in static file server in the API with SPA support (index.html fallback).
    - Automatic public URL generation for static sites.

## Progress vs Build Plan

- Phase 0: complete.
- Phase 1: complete.
- Phase 2: complete.
- Phase 3: complete.
- Phase 4: complete.
- Phase 5: complete.
- Phase 6: complete.
- Phase 7: complete.
- Next up: Phase 8 logs, events, and status pages.
