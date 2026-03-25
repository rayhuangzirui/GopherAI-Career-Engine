SHELL := /bin/bash

COMPOSE ?= docker compose
DC := $(COMPOSE)

API_SERVICE      ?= api
MYSQL_SERVICE    ?= mysql
REDIS_SERVICE    ?= redis
RABBITMQ_SERVICE ?= rabbitmq
MIGRATE_SERVICE  ?= migrate

PROJECT_NAME ?= $(notdir $(CURDIR))

.PHONY: help
help:
	@echo ""
	@echo "$(PROJECT_NAME) — common dev commands"
	@echo ""
	@echo "Core:"
	@echo "  make up              Start infra -> run migrations -> start api"
	@echo "  make infra           Start mysql, redis, rabbitmq"
	@echo "  make api             Start api only"
	@echo "  make migrate-up      Run migrations"
	@echo "  make migrate-down    Roll back 1 migration"
	@echo "  make down            Stop and remove containers"
	@echo "  make restart         Restart all services"
	@echo "  make ps              Show running services"
	@echo ""
	@echo "Logs:"
	@echo "  make logs            Tail logs for all services"
	@echo "  make logs-api        Tail API logs"
	@echo "  make logs-mysql      Tail MySQL logs"
	@echo "  make logs-redis      Tail Redis logs"
	@echo "  make logs-rabbitmq   Tail RabbitMQ logs"
	@echo "  make logs-migrate    Show migration logs"
	@echo ""
	@echo "Shell:"
	@echo "  make sh-api          Shell into API container"
	@echo "  make sh-mysql        Shell into MySQL container"
	@echo ""
	@echo "Health:"
	@echo "  make health          Quick health check"
	@echo ""
	@echo "Testing:"
	@echo "  make test            Run Go tests in API container"
	@echo ""
	@echo "Clean:"
	@echo "  make clean           Remove containers + volumes (DANGEROUS)"
	@echo ""

.PHONY: up
up:
	$(DC) up -d $(MYSQL_SERVICE) $(REDIS_SERVICE) $(RABBITMQ_SERVICE)
	$(DC) run --rm $(MIGRATE_SERVICE)
	$(DC) up -d $(API_SERVICE)

.PHONY: infra
infra:
	$(DC) up -d $(MYSQL_SERVICE) $(REDIS_SERVICE) $(RABBITMQ_SERVICE)

.PHONY: api
api:
	$(DC) up -d $(API_SERVICE)

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

.PHONY: logs-mysql
logs-mysql:
	$(DC) logs -f --tail=200 $(MYSQL_SERVICE)

.PHONY: logs-redis
logs-redis:
	$(DC) logs -f --tail=200 $(REDIS_SERVICE)

.PHONY: logs-rabbitmq
logs-rabbitmq:
	$(DC) logs -f --tail=200 $(RABBITMQ_SERVICE)

.PHONY: logs-migrate
logs-migrate:
	$(DC) logs --tail=200 $(MIGRATE_SERVICE)

.PHONY: sh-api
sh-api:
	$(DC) exec $(API_SERVICE) sh

.PHONY: sh-mysql
sh-mysql:
	$(DC) exec $(MYSQL_SERVICE) sh

.PHONY: migrate-up
migrate-up:
	$(DC) run --rm $(MIGRATE_SERVICE)

.PHONY: migrate-down
migrate-down:
	$(DC) run --rm $(MIGRATE_SERVICE) /bin/sh -lc "migrate -path=/migrations -database 'mysql://app:app@tcp(mysql:3306)/appdb?multiStatements=true' down 1"

.PHONY: health
health:
	@$(DC) ps
	@echo ""
	@curl -sS http://localhost:8080/health | python3 -m json.tool || true

.PHONY: test
test:
	$(DC) exec $(API_SERVICE) sh -lc 'go test ./...'

.PHONY: clean
clean:
	$(DC) down -v