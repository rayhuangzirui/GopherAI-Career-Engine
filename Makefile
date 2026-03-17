# Makefile
# Usage:
#   make help
#   make up
#   make logs-api
#   make test
#   make down

SHELL := /bin/bash

# ---- Compose ----
COMPOSE ?= docker compose
PROFILE ?= $(if $(COMPOSE_PROFILES),$(COMPOSE_PROFILES),)
DC      := $(COMPOSE) $(if $(PROFILE),--profile $(PROFILE),)

# If your compose file isn't named docker-compose.yml, change this:
# DC := $(COMPOSE) -f compose.yml

# ---- Services (edit these to match your docker-compose.yml) ----
API_SERVICE   ?= api
WEB_SERVICE   ?= web
DB_SERVICE    ?= db

# ---- Misc ----
PROJECT_NAME ?= $(notdir $(CURDIR))

.PHONY: help
help:
	@echo ""
	@echo "$(PROJECT_NAME) — common dev commands"
	@echo ""
	@echo "Core:"
	@echo "  make up              Start all services (build if needed)"
	@echo "  make down            Stop and remove containers"
	@echo "  make restart         Restart all services"
	@echo "  make ps              Show running services"
	@echo "  make logs            Tail logs for all services"
	@echo ""
	@echo "Logs:"
	@echo "  make logs-api        Tail API logs"
	@echo "  make logs-web        Tail Web logs"
	@echo "  make logs-db         Tail DB logs"
	@echo ""
	@echo "Shell:"
	@echo "  make sh-api          Shell into API container"
	@echo "  make sh-web          Shell into Web container"
	@echo "  make sh-db           Shell into DB container"
	@echo ""
	@echo "Health:"
	@echo "  make health          Quick compose health check"
	@echo ""
	@echo "Testing (requires commands inside container):"
	@echo "  make test            Run API tests"
	@echo "  make lint            Run API linter (optional)"
	@echo ""
	@echo "Clean:"
	@echo "  make clean           Remove containers + volumes (DANGEROUS)"
	@echo ""

.PHONY: up
up:
	$(DC) up -d --build

.PHONY: down
down:
	$(DC) down

.PHONY: restart
restart: down up

.PHONY: ps
ps:
	$(DC) ps

.PHONY: logs
logs:
	$(DC) logs -f --tail=200

.PHONY: logs-api
logs-api:
	$(DC) logs -f --tail=200 $(API_SERVICE)

.PHONY: logs-web
logs-web:
	$(DC) logs -f --tail=200 $(WEB_SERVICE)

.PHONY: logs-db
logs-db:
	$(DC) logs -f --tail=200 $(DB_SERVICE)

.PHONY: sh-api
sh-api:
	$(DC) exec $(API_SERVICE) sh

.PHONY: sh-web
sh-web:
	$(DC) exec $(WEB_SERVICE) sh

.PHONY: sh-db
sh-db:
	$(DC) exec $(DB_SERVICE) sh

.PHONY: migrate-up migrate-down
migrate-up:
	$(DC) run --rm migrate
migrate-down:
	$(DC) run --rm migrate /bin/sh -lc "migrate -path=/migrations -database 'mysql://app:app@tcp(mysql:3306)/appdb?multiStatements=true' down 1"

# Simple health check: container status + (optional) curl to API health endpoint
.PHONY: health
health:
	@$(DC) ps
	@echo ""
	@curl -sS http://localhost:8080/health | python3 -m json.tool || true

# ---- API commands (adjust to your stack) ----
# Assumes Go API inside container.
.PHONY: test
test:
	$(DC) exec $(API_SERVICE) sh -lc 'go test ./...'

.PHONY: lint
lint:
	@echo "If you use golangci-lint, add it to your image and enable this target."
	@echo "Example:"
	@echo "  $(DC) exec $(API_SERVICE) sh -lc \"golangci-lint run\""

# ---- Dangerous: wipes volumes (DB data) ----
.PHONY: clean
clean:
	$(DC) down -v


#SHELL := /bin/bash
#COMPOSE ?= docker compose
#
#.PHONY: up down ps logs logs-api sh-api health
#
#up:
#	$(COMPOSE) up -d --build
#
#down:
#	$(COMPOSE) down
#
#ps:
#	$(COMPOSE) ps
#
#logs:
#	$(COMPOSE) logs -f --tail=200
#
#logs-api:
#	$(COMPOSE) logs -f --tail=200 api
#
#sh-api:
#	$(COMPOSE) exec api sh
#
#health:
#	@curl -s http://localhost:8080/health | python3 -m json.tool || true
