# GopherAI Chat Platform

Backend-first chat platform built with Go (Gin), designed for reproducible local development with MySQL, Redis, and RabbitMQ via Docker Compose.

## What this repo demonstrates

- **One-command local dev:** `make up` starts API + MySQL + Redis + RabbitMQ via Docker Compose
- **Health check:** `GET /health` verifies API + MySQL/Redis/RabbitMQ connectivity
- **Version pinning:** Go toolchain pinned (**WSL:** Go `1.23.0`, **container build:** Go `1.23.12`)

---

## Quick Start

### Requirements
- Docker + Docker Compose
- Make
- Go (optional, for local IDE tooling)

### Run
```bash
cp .env.example .env
make up
```

### Verify
```bash
curl -sS http://localhost:8080/health
```

### Expected response (example)
```json
{"ok":true,"env":"dev","mysql":"true","redis":"true","rabbitmq":"true"}
```

### Useful commands
```bash
make ps          # List running containers
make logs-api    # View API logs
make down        # Stop and remove containers
```

## Runtime & Toolchain Versions (Pinned)
To keep builds reproducible, this project pins toolchian and service versions.

### Toolchain
- Go (local IDE / WSL): `1.23.0`
- Go (container build): `1.23.12`

### Services (Docker images)
- MySQL: `mysql:8.4`
- Redis: `redis:7-alpine`
- RabbitMQ (management UI): `rabbitmq:3-management`

### Verify versions
Local (WSL)
```bash
go version
docker --version
docker-compose --version
```

## Endpoints
- `GET /health`: Check API and service connectivity

## RabbitMQ Management UI
- URL: `http://localhost:15672`
- Default credentials: `guest` / `guest`
If you see a login/permission error, create a dedicated user in the RabbitMQ UI and update RABBITMQ_USER and `RABBITMQ_URL` in `.env`.

## Project Layout
```.
├── api/                # Go backend code (Gin)
├── web/                # Frontend code (planned / optional)
├── docker-compose.yml  # Docker Compose configuration
├── .env.example        # sample configuration
├── Makefile            # Make commands for local development
└── README.md           
```
