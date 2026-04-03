# Hosting Platform MVP Design Doc

## 1. Purpose

Build a portfolio-grade hosting platform that lets a user deploy and manage two kinds of application workloads:

1. **Web Services** — APIs, backends, and full-stack applications.
2. **Static Sites** — React, HTML, CSS, JS, and other prebuilt frontend assets.

The system must be implementable with **Go** on the backend and **React** on the frontend, and it must be deployable with **zero recurring infrastructure cost** for the MVP.

This document is written as an execution spec. It is intentionally opinionated. Ambiguity is removed by making concrete decisions wherever a choice exists.

---

## 2. Product definition

### 2.1 One-sentence product statement

A developer can connect a Git repository, choose a workload type, configure environment variables and build settings, and receive a public URL for their deployed app with logs, status, and rollbacks.

### 2.2 MVP goals

The MVP must support:

* Git-based deployment.
* Web service deployments from a Dockerfile or a build/run command.
* Static site deployments from a build output directory.
* Deployment status tracking.
* Build logs and runtime logs.
* Environment variables.
* One-click redeploy from a Git commit.
* Rollback to a previous successful deployment.
* Public access via generated subdomains under a platform-owned domain.

### 2.3 Non-goals for MVP

Do not build these in the MVP:

* Kubernetes.
* Multi-region deployment.
* Autoscaling.
* Private networking between customer workloads.
* Custom domain onboarding with automated DNS validation.
* Team/org permissions beyond a single user owning projects.
* Advanced secrets management with KMS.
* Billing, metering, or paid plans.
* Worker queues for customer workloads beyond a minimal internal job queue.
* A marketplace of templates.
* Database hosting for customer apps.
* Edge functions as a separate workload type.

---

## 3. Target user and use case

### 3.1 Target user

A student or early-career engineer building a portfolio project who needs a realistic cloud-platform architecture without incurring hosting cost.

### 3.2 Primary user journey

1. Sign up.
2. Create a project.
3. Connect a Git repository.
4. Choose workload type: Web Service or Static Site.
5. Configure build settings.
6. Deploy.
7. View status, logs, and public URL.
8. Trigger redeploy or rollback.

---

## 4. System architecture

### 4.1 Architectural style

Use a **modular monolith** for the control plane and a **single-node worker executor** for the build/deploy path.

This is the correct MVP choice because:

* It is easier to ship than microservices.
* It is easier to debug on a free single-VM deployment.
* It avoids unnecessary operational complexity.
* It still maps cleanly to a future distributed architecture.

### 4.2 High-level components

#### A. Frontend web app

* React single-page application.
* Responsible for authentication screens, project management, deployment configuration, logs, and status dashboards.
* Talks only to the backend API.

#### B. Backend API

* Go HTTP server.
* Owns auth, project metadata, deployment orchestration, log retrieval, and static-file serving for deployed static sites.
* Exposes REST endpoints.

#### C. Internal job runner

* Go worker process, or a worker mode within the same binary.
* Executes builds and deployment steps asynchronously.
* Responsible for cloning repos, building artifacts, packaging artifacts, launching containers, and publishing static sites.

#### D. Database

* PostgreSQL.
* Stores users, projects, deployments, build metadata, logs index, and configuration.

#### E. Artifact storage

* Local persistent disk for the MVP.
* Stores build artifacts, static site bundles, and deployment metadata.
* Later replace with object storage without changing core domain models.

#### F. Reverse proxy

* Nginx or Caddy in front of the backend and runtime containers.
* Routes by hostname to the correct deployment.
* Terminates TLS.

### 4.3 Deployment topology for free hosting

Use a **single Always Free VM** as the simplest fully free MVP deployment target.

On that VM run:

* backend API
* worker
* PostgreSQL
* reverse proxy
* user-facing frontend assets
* runtime containers for hosted web services

This is the only free deployment path that keeps the platform self-contained and portfolio-realistic without depending on paid managed services.

---

## 5. Free-hosting strategy

### 5.1 Chosen strategy

The MVP will be deployed on one free infrastructure host, with frontend assets served separately if needed, but the platform itself should remain operable from a single public VM.

### 5.2 Why this choice

This removes dependency on multiple free tiers that may have different limits, vendor lock-in, or account approval issues.

### 5.3 Acceptable fallback options

If the single-VM path becomes too constrained, split the deployment into:

* React frontend on a free static host.
* Go backend on a free web-service host.

That is a contingency plan, not the primary architecture.

---

## 6. Core domain model

### 6.1 Entities

#### User

Represents an authenticated human owner.
Fields:

* id
* email
* password_hash
* created_at
* updated_at

#### Project

Represents a logical application being hosted.
Fields:

* id
* owner_user_id
* name
* slug
* workload_type
* repo_url
* default_branch
* created_at
* updated_at
* deleted_at nullable

#### ProjectConfig

Represents the deployment configuration for a project.
Fields:

* id
* project_id
* build_command nullable
* start_command nullable
* dockerfile_path nullable
* output_dir nullable
* install_command nullable
* node_version nullable
* go_version nullable
* env_json
* healthcheck_path nullable
* port nullable
* created_at
* updated_at

#### Deployment

Represents one immutable deployment attempt.
Fields:

* id
* project_id
* git_commit_sha
* git_branch
* status
* source_type
* artifact_path nullable
* runtime_container_id nullable
* public_url nullable
* created_at
* started_at nullable
* finished_at nullable
* created_by_user_id

#### DeploymentEvent

Represents a timeline event for a deployment.
Fields:

* id
* deployment_id
* event_type
* message
* timestamp
* metadata_json nullable

#### LogLine

Represents indexed logs.
Fields:

* id
* deployment_id
* stream_type
* line_number
* message
* created_at

#### DomainMapping

Represents a hostname routed to a project deployment.
Fields:

* id
* project_id
* hostname
* target_type
* target_id
* verified_at nullable
* created_at

### 6.2 Enumerations

#### workload_type

* `web_service`
* `static_site`

#### deployment status

* `queued`
* `building`
* `build_failed`
* `deploying`
* `running`
* `failed`
* `stopped`
* `rolled_back`

#### event_type

* `created`
* `cloned_repo`
* `build_started`
* `build_finished`
* `artifact_uploaded`
* `container_started`
* `container_healthy`
* `container_unhealthy`
* `deployment_ready`
* `deployment_failed`
* `rollback_started`
* `rollback_finished`

---

## 7. Functional requirements

### 7.1 Authentication

* Email/password sign-up and sign-in.
* Session-based auth using secure HTTP-only cookies.
* Password reset can be omitted from MVP.
* One active session per browser is enough.

### 7.2 Project creation

A project must require:

* project name
* repo URL
* workload type
* default branch

A project may optionally specify:

* build command
* start command
* output directory
* Dockerfile path
* environment variables
* healthcheck path
* exposed port

### 7.3 Web service hosting

A web service deployment must support one of two modes:

#### Mode A: Dockerfile mode

* Build image from a `Dockerfile` at the repository root or specified path.
* Run the resulting container.
* Expose a configured port.

#### Mode B: Build/run command mode

* Detect language runtime from configuration.
* Run install/build/start commands in a clean build environment.
* Package runtime files into an artifact.
* Launch a runtime container from a standard base image.

For the MVP, Dockerfile mode is the primary path. Build/run mode is a convenience path, not the main engineering contract.

### 7.4 Static site hosting

A static site deployment must:

* run a build command
* capture the output directory
* serve the output directory over HTTP
* support client-side routing fallback to `index.html`

### 7.5 Deployments

A deployment must:

* be triggered manually from the UI
* be triggered automatically on repository push via webhook
* run asynchronously
* show live logs while building
* preserve historical logs after completion

### 7.6 Rollbacks

Rollback means redeploying the exact artifact or exact commit hash of a previous successful deployment.
Rollback must not mutate the original deployment record.
A rollback creates a new deployment record referencing the prior deployment as `rollback_source_deployment_id`.

### 7.7 Logs

Logs must support:

* build logs
* deploy logs
* runtime logs

Logs should be visible in the UI as a live tail and as a persisted history.

### 7.8 Status page

Each project must show:

* current deployment status
* most recent successful deployment
* current public URL
* recent events
* recent logs

---

## 8. API design

### 8.1 API principles

* REST over HTTP/JSON.
* Versioned under `/api/v1`.
* All write endpoints require authentication.
* Responses use consistent envelope and error shape.
* All IDs are UUIDs.

### 8.2 Standard response envelope

Success:

```json
{
  "data": { }
}
```

Error:

```json
{
  "error": {
    "code": "string",
    "message": "string",
    "details": { }
  }
}
```

### 8.3 Endpoints

#### Auth

* `POST /api/v1/auth/signup`
* `POST /api/v1/auth/login`
* `POST /api/v1/auth/logout`
* `GET /api/v1/auth/me`

#### Projects

* `GET /api/v1/projects`
* `POST /api/v1/projects`
* `GET /api/v1/projects/:projectId`
* `PATCH /api/v1/projects/:projectId`
* `DELETE /api/v1/projects/:projectId`

#### Config

* `GET /api/v1/projects/:projectId/config`
* `PUT /api/v1/projects/:projectId/config`

#### Deployments

* `GET /api/v1/projects/:projectId/deployments`
* `POST /api/v1/projects/:projectId/deployments`
* `GET /api/v1/deployments/:deploymentId`
* `POST /api/v1/deployments/:deploymentId/rollback`
* `POST /api/v1/deployments/:deploymentId/redeploy`

#### Logs

* `GET /api/v1/deployments/:deploymentId/logs`
* `GET /api/v1/deployments/:deploymentId/logs/stream`

#### Domains

* `GET /api/v1/projects/:projectId/domains`
* `POST /api/v1/projects/:projectId/domains`
* `DELETE /api/v1/domains/:domainId`

#### Webhooks

* `POST /api/v1/webhooks/git/:provider`

### 8.4 Webhook contract

A webhook must include:

* repository identifier
* branch name
* commit SHA
* event type

The backend must verify webhook authenticity before enqueueing a deployment.

---

## 9. Deployment lifecycle

### 9.1 State machine

`queued` -> `building` -> `deploying` -> `running`

Failure paths:

* `queued` -> `build_failed`
* `building` -> `build_failed`
* `deploying` -> `failed`
* `running` -> `failed`

Rollback path:

* `running` -> `rolled_back` is not a terminal transition on the same deployment.
* Instead, create a new deployment that becomes `building` and then `running`.

### 9.2 Build process for web services

1. Validate config.
2. Clone repo at specified commit.
3. Detect Dockerfile or use configured commands.
4. Build image or artifact.
5. Store build logs.
6. Start container.
7. Run health check.
8. Mark deployment `running` if healthy.

### 9.3 Build process for static sites

1. Validate config.
2. Clone repo at specified commit.
3. Run install command if present.
4. Run build command.
5. Copy output directory into immutable artifact storage.
6. Bind the artifact to the deployment.
7. Route the hostname to the artifact.
8. Mark deployment `running`.

### 9.4 Failure handling

If any step fails:

* capture the failing step
* persist the error message
* mark the deployment failed
* expose the error in the UI
* keep logs available for later inspection

---

## 10. Backend technical design

### 10.1 Language and framework

* Go 1.24+.
* Use `net/http` or a minimal router such as Chi.
* Keep framework weight low.

### 10.2 Service boundaries inside the Go codebase

Use packages, not ad hoc files.

Recommended package layout:

* `cmd/api`
* `cmd/worker`
* `internal/config`
* `internal/auth`
* `internal/httpapi`
* `internal/domain`
* `internal/project`
* `internal/deployment`
* `internal/webhook`
* `internal/build`
* `internal/runtime`
* `internal/storage`
* `internal/logging`
* `internal/db`
* `internal/middleware`
* `internal/observability`
* `internal/testutil`

### 10.3 Backend responsibilities

The Go backend must own:

* request validation
* auth and session management
* project CRUD
* deployment orchestration
* artifact bookkeeping
* runtime container management
* serving static site assets
* log storage and retrieval
* webhook ingestion

### 10.4 Backend patterns

* Use dependency injection through constructors.
* Never reach into global state.
* Isolate side effects behind interfaces.
* Keep HTTP handlers thin.
* Put business logic in services, not in handlers.
* Use database transactions for multi-entity mutations.
* Use idempotency keys for deployment-trigger endpoints.

---

## 11. Frontend technical design

### 11.1 Language and framework

* React with TypeScript.
* Vite for local dev and build.
* React Router for routing.
* TanStack Query for API state.
* Form library for validations.

### 11.2 Frontend responsibilities

* Authentication UI.
* Project list and project detail views.
* Deployment configuration form.
* Deployment log viewer.
* Deployment status badges.
* Rollback and redeploy actions.

### 11.3 Frontend architecture

Use a feature-oriented structure, not a dump of components.

Recommended folder layout:

* `src/app`
* `src/features/auth`
* `src/features/projects`
* `src/features/deployments`
* `src/features/domains`
* `src/components`
* `src/lib/api`
* `src/lib/auth`
* `src/lib/query`
* `src/styles`
* `src/routes`
* `src/types`

### 11.4 UI design principles

* Dense but readable dashboard.
* High contrast status indicators.
* Deployment logs in monospace.
* Clear primary CTA: “Deploy”.
* No decorative clutter.
* Strong empty states.

### 11.5 Frontend pages

#### Public

* Landing page
* Sign in
* Sign up

#### Authenticated

* Dashboard
* Create project
* Project detail
* Deployment detail
* Settings

### 11.6 UX rules

* A deploy action must always show a confirmation of what will happen.
* Build logs must update without page refresh.
* Failed deployments must show the first actionable error line.
* Empty states must tell the user the next step.

---

## 12. Repository and project folder structure

### 12.1 Monorepo layout

```
/
├─ backend/
│  ├─ cmd/
│  │  ├─ api/
│  │  │  └─ main.go
│  │  └─ worker/
│  │     └─ main.go
│  ├─ internal/
│  │  ├─ auth/
│  │  ├─ config/
│  │  ├─ db/
│  │  ├─ deployment/
│  │  ├─ domain/
│  │  ├─ httpapi/
│  │  ├─ logging/
│  │  ├─ middleware/
│  │  ├─ observability/
│  │  ├─ project/
│  │  ├─ runtime/
│  │  ├─ storage/
│  │  ├─ webhook/
│  │  └─ build/
│  ├─ migrations/
│  ├─ test/
│  ├─ go.mod
│  └─ go.sum
├─ frontend/
│  ├─ public/
│  ├─ src/
│  │  ├─ app/
│  │  ├─ components/
│  │  ├─ features/
│  │  ├─ lib/
│  │  ├─ routes/
│  │  ├─ styles/
│  │  └─ types/
│  ├─ index.html
│  ├─ package.json
│  └─ vite.config.ts
├─ infra/
│  ├─ docker/
│  ├─ nginx/
│  ├─ compose/
│  ├─ terraform/
│  └─ scripts/
├─ docs/
├─ scripts/
├─ .github/
│  └─ workflows/
└─ README.md
```

### 12.2 File ownership rules

* `backend/internal/*` contains all production Go application logic.
* `backend/cmd/*` contains only bootstrapping.
* `frontend/src/features/*` contains feature-specific UI and hooks.
* `infra/*` contains deployment files only.
* `scripts/*` contains human-run convenience scripts only.
* No business logic belongs in `scripts`, `infra`, or `cmd`.

---

## 13. Database design

### 13.1 Database choice

PostgreSQL.

### 13.2 Schema rules

* All tables use UUID primary keys.
* All rows have `created_at` and `updated_at` unless immutable.
* Soft delete only for user-facing project records.
* Immutable deployment records are never updated after creation except for status-transition columns.

### 13.3 Indexing requirements

Add indexes on:

* `users.email`
* `projects.owner_user_id`
* `projects.slug`
* `deployments.project_id`
* `deployments.status`
* `deployments.git_commit_sha`
* `log_lines.deployment_id`
* `domain_mappings.hostname`

### 13.4 Migration strategy

* Use migration files committed to version control.
* One migration per schema change.
* Migrations must be reversible where practical.
* Never edit a migration after it has been shared or applied.

---

## 14. Runtime hosting model for customer apps

### 14.1 Web services

Each deployed web service runs as a Docker container on the host VM.

#### Runtime contract

* Container must listen on one configured port.
* The platform injects `PORT`.
* Health checks probe the configured health path.
* Logs are collected from stdout/stderr.

### 14.2 Static sites

Static sites are served from immutable directories behind the reverse proxy.

#### Runtime contract

* Files are copied to a versioned directory.
* The active deployment path is switched atomically.
* Client-side routes fall back to `index.html`.

### 14.3 Resource isolation

MVP isolation is container-based only.

No promise is made that this is production-grade multi-tenant isolation. It is good enough for a portfolio project and unsafe to present as a hard production guarantee.

---

## 15. Security design

### 15.1 Authentication security

* Passwords hashed with Argon2id or bcrypt.
* Cookies are `HttpOnly`, `Secure`, and `SameSite=Lax` or stricter.
* CSRF protection required for cookie-authenticated state-changing requests.

### 15.2 Repository security

* GitHub App or webhook secret verification preferred.
* Repo URL validation required.
* Clone operations must be time-limited and resource-limited.

### 15.3 Deployment security

* Build and runtime containers run with minimal privileges.
* Containers must not run as root where avoidable.
* Drop unnecessary Linux capabilities.
* Mount only required volumes.
* Reject dangerous environment variables at config validation time, such as values that would let a deployed app escape its sandbox through platform internals.

### 15.4 Secrets handling

* Secrets never stored in frontend code.
* Secrets stored encrypted at rest if possible.
* At minimum, secrets stored in the backend database and rendered only at deploy time.
* Secrets must never appear in logs.

### 15.5 Abuse prevention

* Rate limit login, signup, webhook ingestion, and deployment triggers.
* Reject excessively large repositories for the MVP.
* Enforce max build duration.
* Enforce max artifact size.

---

## 16. Observability

### 16.1 Logging

Use structured JSON logs in the backend and worker.
Each log entry must include:

* timestamp
* level
* request_id
* user_id when available
* project_id when available
* deployment_id when available
* message

### 16.2 Metrics

Expose a `/metrics` endpoint for internal scraping if feasible.
Minimum metrics:

* total deployments
* deployment success rate
* build duration histogram
* queue depth
* runtime container count
* API error rate

### 16.3 Tracing

Tracing is optional in the MVP, but request IDs are mandatory.

### 16.4 Health endpoints

* `GET /healthz` for process health.
* `GET /readyz` for dependency readiness.

---

## 17. Build and deploy pipeline

### 17.1 Local development

Use Docker Compose for local development.
The local stack must start:

* PostgreSQL
* backend API
* worker
* frontend dev server

### 17.2 CI pipeline

Every pull request must run:

* Go formatting
* Go tests
* Go lint
* frontend typecheck
* frontend unit tests
* frontend build
* migration sanity check

### 17.3 CD pipeline

On merge to main:

* build backend binary
* build frontend assets
* build Docker image
* push image
* deploy to the VM
* run smoke test against `/healthz`

### 17.4 Release tagging

Use git tags for release versions.
Production deploys should be tied to a tag or a main-branch commit SHA, never an untraceable manual build.

---

## 18. Local developer experience

### 18.1 Required commands

* `make dev`
* `make test`
* `make lint`
* `make build`
* `make migrate-up`
* `make migrate-down`
* `make seed`
* `make docker-up`
* `make docker-down`

### 18.2 Environment files

* `.env.example` committed.
* `.env.local` ignored.
* `backend/.env` allowed only for local development.

### 18.3 Determinism rules

* No hidden environment assumptions.
* No manual database steps.
* No “works on my machine” setup.
* All local dependencies must be scriptable.

---

## 19. Configuration model

### 19.1 Backend environment variables

Required:

* `APP_ENV`
* `APP_BASE_URL`
* `DATABASE_URL`
* `SESSION_SECRET`
* `WEBHOOK_SECRET`
* `GIT_PROVIDER_TOKEN` if applicable
* `STORAGE_ROOT`
* `PUBLIC_DOMAIN_SUFFIX`

Optional:

* `LOG_LEVEL`
* `METRICS_ENABLED`
* `MAX_BUILD_TIME_SECONDS`
* `MAX_ARTIFACT_BYTES`
* `MAX_CONCURRENT_BUILDS`

### 19.2 Project-level config fields

* build command
* start command
* output directory
* dockerfile path
* port
* healthcheck path
* install command
* environment variables

### 19.3 Validation rules

* Required fields must be explicit.
* Empty strings are invalid.
* Paths must be normalized.
* Dangerous characters in repo URLs and domain names must be rejected.

---

## 20. Static site serving rules

### 20.1 Directory layout

Each deployment gets a unique artifact directory:
`/data/static-sites/<projectId>/<deploymentId>/`

### 20.2 Serving behavior

* Serve files directly.
* `index.html` fallback for routes without file extensions.
* Cache immutable assets aggressively.
* Do not cache HTML too long.

### 20.3 Cleanup policy

Keep:

* latest successful deployment
* previous successful deployment
* latest failed deployment for debugging

Delete older artifacts according to a retention job.

---

## 21. Web service runtime rules

### 21.1 Runtime lifecycle

* Container launched from build output or Docker image.
* Platform assigns a public hostname.
* Health check must pass before traffic is marked live.
* If health check fails, deployment is marked failed and does not receive traffic.

### 21.2 Runtime constraints

* One container per deployment for MVP.
* One exposed port.
* One hostname.
* No sidecars.
* No background worker scaling.

### 21.3 Traffic switching

Use atomic proxy configuration replacement.
A new deployment is staged first, then traffic is shifted only after health passes.

---

## 22. Hostname and routing design

### 22.1 Default hostname format

Use:
`<project-slug>.<platform-domain>`

### 22.2 Routing rules

* Static site requests route to static artifact service.
* Web service requests route to runtime container.
* Platform UI and API live under separate hostnames or path prefixes.

### 22.3 Domain ownership

For MVP, only platform-owned subdomains are supported.
Custom domains are deferred.

---

## 23. Internal queues and job processing

### 23.1 Queue implementation

Use PostgreSQL-backed job tables for MVP.
Do not add Redis unless there is a concrete need.

### 23.2 Job types

* deploy project
* rollback deployment
* refresh webhook deployment
* cleanup artifacts
* cleanup logs

### 23.3 Worker semantics

* At-least-once execution.
* Idempotent handlers.
* Row locking or `FOR UPDATE SKIP LOCKED` equivalent.

---

## 24. Data retention

### 24.1 Logs

Keep at least 30 days for the MVP, or until disk pressure requires eviction.

### 24.2 Artifacts

Keep the most recent successful deployment and one prior known-good deployment per project.

### 24.3 Audit history

Never delete deployment records.
Event history is part of the product value and debugging story.

---

## 25. Testing strategy

### 25.1 Backend tests

* Unit tests for validation and business rules.
* Integration tests for database operations.
* HTTP handler tests.
* Worker job execution tests.

### 25.2 Frontend tests

* Component tests for forms and status display.
* API hook tests.
* Smoke tests for routing.

### 25.3 End-to-end tests

Must cover:

* sign up
* create project
* deploy static site
* deploy web service
* view logs
* rollback deployment

### 25.4 Test data

Use factories and seeded fixtures.
Do not manually create state by hand during tests.

---

## 26. Quality bars

A feature is not complete unless all of the following are true:

* it is validated by tests
* it is documented
* it has a migration if it changes persistence
* it is visible in the UI
* it has clear failure handling
* it can be operated locally

---

## 27. Engineering principles to enforce

### 27.1 Explicitness

Never rely on implicit runtime behavior for anything user-facing.

### 27.2 Separation of concerns

* UI renders state.
* API validates and coordinates.
* services contain business logic.
* repositories persist data.
* workers execute side effects.

### 27.3 Idempotency

All deployment trigger endpoints must tolerate retries.

### 27.4 Immutable deployments

Once created, a deployment record should behave like an immutable event.

### 27.5 Low operational overhead

Every system added must justify itself. If a component is not necessary for the MVP, it does not belong.

---

## 28. Future roadmap

### Phase 2

* Custom domains.
* Team support.
* Preview deployments.
* Better rollback UX.
* Build cache.
* Image-based artifact handling.

### Phase 3

* Multi-node scaling.
* Managed queue.
* Managed object storage.
* Multi-region routing.
* Usage metering.
* Billing.

---

## 29. Final MVP decision

The MVP should be built as a Go control plane and worker with a React dashboard, deployed on a single always-free VM, and limited to two workload types: web services and static sites.

That is the highest-signal, lowest-complexity path.
Anything broader is scope creep.

