# Hosting Platform MVP — Step-by-Step Execution Plan

## 1. Purpose

This document converts the MVP design into an execution backlog suitable for a team working from a blank repository. It is ordered by dependency and delivery value, with each item written so it can become a kanban card with a clear definition of done.

The plan assumes:

* Backend: Go
* Frontend: React + TypeScript
* Architecture: modular monolith control plane + single worker
* Database: PostgreSQL
* Deployment model: zero recurring cost for the MVP

---

## 2. Delivery principles

These rules govern every backlog item:

1. **Build the control plane before the deploy plane.**
   You do not deploy customer workloads before you can reliably create, store, and inspect project records.

2. **Ship the thinnest vertical slice first.**
   Prefer one complete path from UI → API → DB → worker → visible result over multiple partially finished subsystems.

3. **Make every side effect observable.**
   If a task creates, mutates, deploys, or deletes something, it must emit events and logs.

4. **Do not add abstractions before the second concrete use case.**
   The MVP has only two workload types. Resist premature plugin systems, generic schedulers, or enterprise auth.

5. **Every backlog item must have a clear completion test.**
   No card is done without an objective acceptance test.

6. **Keep the platform operable on one machine.**
   If a design choice requires distributed infrastructure, it is not MVP work.

---

## 3. Kanban operating model

### 3.1 Board columns

Use these columns:

* Backlog
* Ready
* In Progress
* Review
* QA
* Done

### 3.2 WIP limits

Recommended limits for a small team:

* In Progress: 2 per engineer
* Review: 2 total
* QA: 1–2 total

### 3.3 Card standard

Each card should contain:

* Title
* Objective
* Scope
* Dependencies
* Acceptance criteria
* Notes / risks

### 3.4 Definition of done

A card is done only when:

* code is merged
* tests pass
* behavior is documented if user-facing
* no hidden manual step remains
* observability is added if the change has operational impact

---

## 4. Execution sequence overview

The project should be delivered in this order:

1. Repository bootstrap and developer workflow.
2. Database and backend skeleton.
3. Authentication and session management.
4. Project CRUD.
5. Frontend shell and authenticated navigation.
6. Deployment data model and orchestration skeleton.
7. Web service deployment path.
8. Static site deployment path.
9. Logs, events, and status pages.
10. Webhooks and automatic redeploys.
11. Rollback and retention.
12. Hardening, tests, and production deployment.

Do not reorder this unless a dependency forces it.

---

## 5. Phase 0 — Project foundation

### Card 0.1 — Create repository and mono-repo structure

**Objective:** Establish the folder structure and top-level project layout.

**Scope:**

* Create `backend/`, `frontend/`, `infra/`, `docs/`, `scripts/`, `.github/`.
* Create empty starter files and README.
* Add root `.gitignore`.
* Add root `Makefile`.

**Dependencies:** None.

**Acceptance criteria:**

* Repository matches the agreed structure.
* A contributor can identify where backend, frontend, and infra code belong.

---

### Card 0.2 — Establish local development environment

**Objective:** Make the project runnable from day one.

**Scope:**

* Add Docker Compose for PostgreSQL and local services.
* Add environment variable templates.
* Add scripts for starting/stopping the stack.
* Decide local ports and document them.

**Dependencies:** 0.1

**Acceptance criteria:**

* A new developer can boot the local stack with one command.
* PostgreSQL starts cleanly.
* Configuration does not rely on undocumented manual steps.

---

### Card 0.3 — Add CI skeleton

**Objective:** Create the minimum CI path before feature work begins.

**Scope:**

* Go format and test job.
* Frontend install, typecheck, and build job.
* Lint job placeholder if linting is not yet configured.

**Dependencies:** 0.1

**Acceptance criteria:**

* Every pull request runs automated validation.
* Broken code cannot merge silently.

---

## 6. Phase 1 — Backend foundation

### Card 1.1 — Scaffold Go backend application

**Objective:** Create the production backend entrypoint and server bootstrap.

**Scope:**

* `cmd/api/main.go`
* configuration loading
* structured logging
* HTTP server bootstrap
* graceful shutdown

**Dependencies:** 0.2

**Acceptance criteria:**

* The API process starts and serves a health endpoint.
* Configuration is loaded from environment variables.
* Shutdown does not drop in-flight requests abruptly.

---

### Card 1.2 — Add database connection and migration workflow

**Objective:** Make the backend persist data safely.

**Scope:**

* PostgreSQL connection setup
* migration runner
* initial schema migration framework
* database health check

**Dependencies:** 1.1

**Acceptance criteria:**

* Backend can connect to PostgreSQL.
* Migrations can be applied locally and in CI.
* Schema changes are versioned.

---

### Card 1.3 — Add core backend package boundaries

**Objective:** Create the internal structure that prevents the codebase from becoming a flat mess.

**Scope:**

* `internal/auth`
* `internal/project`
* `internal/deployment`
* `internal/webhook`
* `internal/build`
* `internal/runtime`
* `internal/storage`
* `internal/httpapi`
* `internal/middleware`

**Dependencies:** 1.1

**Acceptance criteria:**

* Packages are in place with clear ownership.
* Business logic is not living in HTTP handlers.

---

## 7. Phase 2 — Authentication and identity

### Card 2.1 — Implement user schema and auth tables

**Objective:** Persist user identities.

**Scope:**

* users table
* sessions table
* password reset fields excluded for MVP unless needed later

**Dependencies:** 1.2

**Acceptance criteria:**

* User records can be created and queried.
* Session persistence is available.

---

### Card 2.2 — Implement sign-up and sign-in

**Objective:** Let a user create an account and authenticate.

**Scope:**

* signup endpoint
* login endpoint
* logout endpoint
* auth/me endpoint
* password hashing
* secure session cookies

**Dependencies:** 2.1

**Acceptance criteria:**

* A user can register, log in, stay logged in, and log out.
* Invalid credentials are rejected.
* Passwords are never stored in plaintext.

---

### Card 2.3 — Add auth middleware and route protection

**Objective:** Enforce authentication on private routes.

**Scope:**

* middleware for session validation
* request-scoped user context
* redirect or API 401 behavior as appropriate

**Dependencies:** 2.2

**Acceptance criteria:**

* Protected routes reject unauthenticated access.
* Authenticated requests receive the current user identity.

---

## 8. Phase 3 — Project management core

### Card 3.1 — Implement project schema

**Objective:** Persist project metadata and configuration.

**Scope:**

* projects table
* project_configs table
* domain_mappings table placeholder if needed now

**Dependencies:** 1.2

**Acceptance criteria:**

* Projects and configuration can be stored independently.
* Each project belongs to exactly one user.

---

### Card 3.2 — Implement project CRUD API

**Objective:** Allow the user to manage hosted projects.

**Scope:**

* list projects
* create project
* get project
* update project
* delete project

**Dependencies:** 2.3, 3.1

**Acceptance criteria:**

* User can create and manage projects through the API.
* Access is scoped to the authenticated owner.

---

### Card 3.3 — Implement project configuration API

**Objective:** Save deployment settings for a project.

**Scope:**

* build command
* start command
* Dockerfile path
* output directory
* install command
* port
* healthcheck path
* environment variables

**Dependencies:** 3.2

**Acceptance criteria:**

* Configuration is validated and stored.
* Empty or invalid fields are rejected.

---

## 9. Phase 4 — Frontend foundation

### Card 4.1 — Scaffold React application

**Objective:** Create the frontend shell.

**Scope:**

* Vite setup
* React Router
* TypeScript config
* API client layer
* global layout

**Dependencies:** 0.1

**Acceptance criteria:**

* Frontend builds successfully.
* Route structure exists.

---

### Card 4.2 — Implement auth screens

**Objective:** Give the user a way to sign up and sign in.

**Scope:**

* sign up page
* sign in page
* form validation
* error display

**Dependencies:** 2.2, 4.1

**Acceptance criteria:**

* User can authenticate from the UI.
* Failed sign-in shows a clear error.

---

### Card 4.3 — Implement authenticated app shell

**Objective:** Provide the dashboard structure after login.

**Scope:**

* navigation
* account state
* route guards
* empty states

**Dependencies:** 2.3, 4.1

**Acceptance criteria:**

* Logged-in users see the dashboard shell.
* Logged-out users cannot reach private routes.

---

### Card 4.4 — Implement project list and create project UI

**Objective:** Allow users to create and view projects end-to-end.

**Scope:**

* project list page
* create project form
* project detail route shell

**Dependencies:** 3.2, 4.3

**Acceptance criteria:**

* A project can be created from the frontend.
* The project appears in the list immediately after creation.

---

## 10. Phase 5 — Deployment model and orchestration skeleton

### Card 5.1 — Implement deployment schema and state machine

**Objective:** Persist deployments as immutable operational records.

**Scope:**

* deployments table
* deployment_events table
* deployment status transitions
* rollback source references

**Dependencies:** 1.2, 3.1

**Acceptance criteria:**

* A deployment can be created, updated through permitted states, and audited.
* Invalid status transitions are blocked.

---

### Card 5.2 — Implement internal job queue

**Objective:** Run deployment work asynchronously.

**Scope:**

* queue table
* job lease/claim logic
* retry logic
* failure handling

**Dependencies:** 5.1

**Acceptance criteria:**

* Jobs can be queued and processed by the worker.
* A crash does not permanently lose queued jobs.

---

### Card 5.3 — Add worker process bootstrap

**Objective:** Create the separate process that executes jobs.

**Scope:**

* `cmd/worker/main.go`
* job polling loop
* graceful shutdown
* shared service wiring

**Dependencies:** 5.2

**Acceptance criteria:**

* Worker starts independently of the API.
* Worker can claim and process jobs.

---

### Card 5.4 — Implement deployment trigger API

**Objective:** Let a project create a deployment request.

**Scope:**

* manual deploy endpoint
* idempotency handling
* initial status creation
* job enqueueing

**Dependencies:** 5.2, 5.3, 3.2

**Acceptance criteria:**

* A deploy request creates a deployment record and job.
* Repeated retries do not create duplicate unwanted deploys.

---

## 11. Phase 6 — Web service deployment path

### Card 6.1 — Implement repo checkout and build sandbox

**Objective:** Safely clone source and prepare a build workspace.

**Scope:**

* repository clone logic
* commit checkout
* workspace cleanup
* timeouts
* disk usage controls

**Dependencies:** 5.3

**Acceptance criteria:**

* A repo can be checked out reproducibly.
* Cleanup happens even on failure.

---

### Card 6.2 — Implement Dockerfile-based web service build

**Objective:** Build a runnable image for web service workloads.

**Scope:**

* Dockerfile discovery
* docker build execution
* image tagging
* build logs capture

**Dependencies:** 6.1

**Acceptance criteria:**

* A valid Dockerfile produces a runnable image.
* Build errors are captured and visible.

---

### Card 6.3 — Implement runtime container launch

**Objective:** Start a deployed web service container.

**Scope:**

* container create/start
* port mapping
* environment injection
* runtime logs capture
* container lifecycle tracking

**Dependencies:** 6.2

**Acceptance criteria:**

* Container starts and stays reachable.
* Logs are captured from stdout/stderr.

---

### Card 6.4 — Implement health checking and readiness gate

**Objective:** Prevent broken deployments from becoming live.

**Scope:**

* healthcheck endpoint polling
* timeout handling
* unhealthy rollback/fail state

**Dependencies:** 6.3

**Acceptance criteria:**

* Healthy deployments become live.
* Failed health checks mark the deployment failed.

---

### Card 6.5 — Implement public routing for web services

**Objective:** Route hostnames to the correct running container.

**Scope:**

* hostname assignment
* reverse proxy config generation
* atomic route updates

**Dependencies:** 6.4

**Acceptance criteria:**

* Public URL resolves to the deployed service.
* Switching deployments does not require downtime by design.

---

## 12. Phase 7 — Static site deployment path

### Card 7.1 — Implement static build pipeline

**Objective:** Build frontend assets into a deployable artifact.

**Scope:**

* install command execution
* build command execution
* output directory capture
* artifact packaging

**Dependencies:** 6.1

**Acceptance criteria:**

* A React or HTML project produces a static artifact.
* The build output is persisted reliably.

---

### Card 7.2 — Implement static site publishing

**Objective:** Serve the generated artifact as a public website.

**Scope:**

* immutable artifact directory structure
* atomic path switch
* index.html fallback
* cache behavior

**Dependencies:** 7.1

**Acceptance criteria:**

* A static site is publicly reachable.
* Client-side routes work correctly.

---

### Card 7.3 — Implement routing for static sites

**Objective:** Bind the project hostname to the correct artifact.

**Scope:**

* hostname mapping
* reverse proxy or file-server config
* deployment activation

**Dependencies:** 7.2

**Acceptance criteria:**

* The latest successful static deployment serves traffic.
* Switching between deployments is deterministic.

---

## 13. Phase 8 — Logs, events, and status UX

### Card 8.1 — Implement event recording

**Objective:** Persist meaningful lifecycle events.

**Scope:**

* deployment event writes
* event timelines
* error event capture

**Dependencies:** 5.1

**Acceptance criteria:**

* Deployment progress is visible as a history of events.
* Failures are traceable to the step that failed.

---

### Card 8.2 — Implement log persistence and retrieval

**Objective:** Store and display build/runtime logs.

**Scope:**

* log ingestion
* line indexing
* log query endpoint
* live tail endpoint

**Dependencies:** 5.3

**Acceptance criteria:**

* Logs are available after the deployment finishes.
* Logs can be tailed while a deployment is running.

---

### Card 8.3 — Build project detail dashboard

**Objective:** Give the user a single operational view per project.

**Scope:**

* current status
* recent deployments
* deployment CTA
* public URL
* events panel
* logs panel

**Dependencies:** 4.4, 8.1, 8.2

**Acceptance criteria:**

* A user can understand deployment state at a glance.
* The dashboard supports real operational use.

---

### Card 8.4 — Build deployment detail page

**Objective:** Show the full history and logs for a single deployment.

**Scope:**

* status timeline
* event stream
* log viewer
* error summary
* deployment metadata

**Dependencies:** 8.1, 8.2

**Acceptance criteria:**

* The user can inspect any deployment in detail.
* The first actionable failure is obvious.

---

## 14. Phase 9 — Webhooks and automated deploys

### Card 9.1 — Implement webhook secret verification

**Objective:** Secure inbound repo event processing.

**Scope:**

* webhook signing secret
* request validation
* replay protection if feasible

**Dependencies:** 5.4

**Acceptance criteria:**

* Invalid webhook requests are rejected.
* Valid webhook requests create deploy jobs.

---

### Card 9.2 — Implement repository event mapping

**Objective:** Translate repo events into deployment triggers.

**Scope:**

* branch matching
* commit SHA extraction
* project lookup
* automatic job enqueueing

**Dependencies:** 9.1, 3.2

**Acceptance criteria:**

* A push event on the configured branch triggers a deployment.
* Non-matching branches are ignored.

---

### Card 9.3 — Add webhook registration UX

**Objective:** Show the user what webhook configuration is needed.

**Scope:**

* display webhook URL
* display secret setup instructions
* verification status indicator

**Dependencies:** 9.1, 8.3

**Acceptance criteria:**

* A user can configure automatic deploys without guessing.

---

## 15. Phase 10 — Rollback and retention

### Card 10.1 — Implement rollback API

**Objective:** Redeploy a previous successful deployment.

**Scope:**

* rollback endpoint
* source deployment validation
* new deployment creation from old source

**Dependencies:** 5.4, 5.1

**Acceptance criteria:**

* Rollback creates a new deployment record.
* The old deployment remains unchanged.

---

### Card 10.2 — Implement artifact retention policy

**Objective:** Prevent uncontrolled storage growth.

**Scope:**

* retention job
* artifact deletion rules
* deployment history preservation

**Dependencies:** 7.2, 6.3

**Acceptance criteria:**

* Old artifacts are cleaned up according to policy.
* Deployment records remain intact.

---

### Card 10.3 — Implement deployment redeploy action

**Objective:** Let the user rerun the same deployment source.

**Scope:**

* redeploy endpoint
* same commit/source reuse
* new deployment record creation

**Dependencies:** 5.4

**Acceptance criteria:**

* A redeploy produces a fresh deployment attempt.
* The UI exposes the action clearly.

---

## 16. Phase 11 — Security and operational hardening

### Card 11.1 — Add request validation and error normalization

**Objective:** Make the API predictable and safe.

**Scope:**

* validation layer
* standard error envelope
* field-level error messages

**Dependencies:** 1.1

**Acceptance criteria:**

* Invalid requests fail consistently.
* Error handling is uniform across endpoints.

---

### Card 11.2 — Add CSRF protection and session hardening

**Objective:** Protect cookie-authenticated actions.

**Scope:**

* CSRF tokens or equivalent protection
* secure cookie settings
* session expiry rules

**Dependencies:** 2.2

**Acceptance criteria:**

* State-changing requests are protected against CSRF.
* Cookies are not exposed unnecessarily.

---

### Card 11.3 — Add rate limiting and abuse controls

**Objective:** Prevent obvious abuse and accidental overload.

**Scope:**

* login throttling
* signup throttling
* webhook throttling
* deploy trigger throttling

**Dependencies:** 2.2, 9.1, 5.4

**Acceptance criteria:**

* Repeated abusive requests are blocked.
* Normal user flows still work.

---

### Card 11.4 — Sandbox build and runtime execution

**Objective:** Minimize blast radius from customer workloads.

**Scope:**

* non-root execution where possible
* resource/time limits
* restricted mounts and capabilities
* workspace cleanup

**Dependencies:** 6.1, 6.3

**Acceptance criteria:**

* Build/runtime processes are constrained.
* The platform does not run customer code with unnecessary privilege.

---

## 17. Phase 12 — Observability and operational readiness

### Card 12.1 — Add structured logging everywhere

**Objective:** Ensure all important actions can be traced.

**Scope:**

* request IDs
* user IDs
* project IDs
* deployment IDs
* standardized log format

**Dependencies:** 1.1

**Acceptance criteria:**

* A production operator can trace a request through logs.

---

### Card 12.2 — Add health and readiness endpoints

**Objective:** Support deployment checks and operational visibility.

**Scope:**

* `/healthz`
* `/readyz`

**Dependencies:** 1.1, 1.2

**Acceptance criteria:**

* Infrastructure can verify the service is alive and ready.

---

### Card 12.3 — Add basic metrics

**Objective:** Provide a minimal operational signal set.

**Scope:**

* deployment counts
* failure rates
* build duration
* queue depth

**Dependencies:** 5.2

**Acceptance criteria:**

* Basic platform health can be measured.

---

## 18. Phase 13 — Testing and release preparation

### Card 13.1 — Add backend unit and integration tests

**Objective:** Lock down business rules.

**Scope:**

* auth tests
* project tests
* deployment state machine tests
* webhook verification tests

**Dependencies:** Core backend work completed incrementally

**Acceptance criteria:**

* Main domain behavior is covered by automated tests.

---

### Card 13.2 — Add frontend component and flow tests

**Objective:** Prevent basic UI regressions.

**Scope:**

* auth form tests
* project creation flow tests
* deployment state rendering tests

**Dependencies:** Frontend foundation and project UI work

**Acceptance criteria:**

* Core user journeys are covered.

---

### Card 13.3 — Add end-to-end smoke test suite

**Objective:** Prove the full system works end to end.

**Scope:**

* sign up
* create project
* deploy static site
* deploy web service
* view logs
* rollback

**Dependencies:** 8.4, 10.1

**Acceptance criteria:**

* The MVP can be validated from the outside as a user would use it.

---

### Card 13.4 — Prepare production deployment

**Objective:** Move from local stack to the free production host.

**Scope:**

* production environment variables
* reverse proxy config
* TLS setup
* deployment scripts
* smoke tests after release

**Dependencies:** All prior phases

**Acceptance criteria:**

* The app runs reliably in a production environment.
* Deployment is repeatable.

---

## 19. Recommended milestone cuts

### Milestone A — Developer foundation

Deliver cards 0.1 through 1.3.

### Milestone B — Core product skeleton

Deliver cards 2.1 through 4.4.

### Milestone C — First deployable vertical slice

Deliver cards 5.1 through 6.5 for web services only.

### Milestone D — Static site support

Deliver cards 7.1 through 7.3.

### Milestone E — Operational completeness

Deliver cards 8.1 through 10.3.

### Milestone F — Hardening and launch readiness

Deliver cards 11.1 through 13.4.

---

## 20. Suggested implementation order inside the team

A sensible small-team order is:

1. Backend foundation and schema.
2. Auth.
3. Project CRUD.
4. Frontend shell and auth UI.
5. Deployment queue and worker.
6. Web service build/run path.
7. Project dashboard and logs.
8. Static site path.
9. Webhooks and rollback.
10. Hardening, tests, and production release.

This sequence avoids the common trap of building a polished dashboard before there is a real system behind it.

---

## 21. Final rule

Do not start on custom domains, billing, autoscaling, or multi-tenant infrastructure until the MVP can already:

* authenticate a user,
* create a project,
* deploy a web service,
* deploy a static site,
* show logs,
* and roll back safely.

Anything before that is distraction.
