SHELL := /bin/sh

COMPOSE_FILE := docker-compose.yml

.PHONY: help dev-up dev-down dev-logs fmt test ci frontend-install backend-test

help:
	@printf '%s\n' "Targets: dev-up dev-down dev-logs fmt test ci frontend-install backend-test"

dev-up:
	docker compose -f $(COMPOSE_FILE) up -d --build

dev-down:
	docker compose -f $(COMPOSE_FILE) down

dev-logs:
	docker compose -f $(COMPOSE_FILE) logs -f

frontend-install:
	cd frontend && npm install

backend-test:
	cd backend && go test ./...

fmt:
	cd backend && go fmt ./...
	cd frontend && npm run format

test: backend-test
	cd frontend && npm install && npm run test

ci: backend-test
	cd frontend && npm install && npm run typecheck && npm run build
