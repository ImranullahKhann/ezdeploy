# Backend

Go API, worker, and database foundation for ezdeploy.

## Auth API

Phase 2 adds session-based authentication with secure HTTP-only cookies. The HTTP API now exposes:

- `POST /auth/signup`
- `POST /auth/login`
- `POST /auth/logout`
- `GET /auth/me`

## Required environment

- `DATABASE_URL`
- `SESSION_SECRET`
- `APP_ENV` defaults to `development` and enables secure cookies in `production`
