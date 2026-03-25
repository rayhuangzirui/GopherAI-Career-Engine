# GopherAI-Career-Engine

GopherAI-Career-Engine is a Dockerized Go backend task processing system for career workflows.  
The project is designed to be reproducible, explainable, and extensible, with an initial focus on asynchronous resume and job description analysis tasks.

## Why this project

Instead of building a generic AI chat demo, this project focuses on a concrete backend use case:

- create career-related analysis tasks
- process them asynchronously
- persist task status and results
- support future expansion into multiple career workflows

The goal is to demonstrate backend engineering skills through:

- task lifecycle management
- queue-based asynchronous processing
- versioned SQL migrations
- reproducible local development with Docker Compose
- clean separation between API and worker responsibilities

## Current architecture

### Stack

- **Go + Gin** for the backend API
- **MySQL** for persistent storage
- **Redis** for cache / short-lived state
- **RabbitMQ** for asynchronous task delivery
- **GORM** for database access
- **SQL migrations** as the single source of truth for schema
- **Docker Compose** for local reproducible development

### Current system direction

The system is being built as a **career task engine**, not a chat-first application.

Initial v1 workflow:

1. client submits a career analysis task
2. API creates a task record in MySQL
3. task is queued for asynchronous processing
4. worker processes the task
5. task result is persisted and can be queried later

## Current progress

### Completed

- Docker Compose local environment
- MySQL / Redis / RabbitMQ service startup
- API health endpoint
- GORM MySQL initialization
- `/debug/db` database connectivity check
- versioned SQL migrations
- initial `tasks` table for task processing workflows

### In progress

- task repository
- task creation / query endpoints
- RabbitMQ producer / consumer flow
- async worker processing
- task status transitions
- idempotent message handling with `processed_keys`

## Database design

Current core tables include:

- `users`
- `tasks`
- `processed_keys`
- `schema_migrations`

The `tasks` table is designed around an async processing lifecycle:

- `task_type`
- `status`
- `input_payload`
- `result_payload`
- `error_message`
- `retry_count`
- `started_at`
- `completed_at`

## Task lifecycle

Planned task states:

- `pending`
- `queued`
- `processing`
- `completed`
- `failed`

This makes task execution observable and allows future support for retries and worker scaling.

## Local development

### Requirements

- Docker
- Docker Compose

### Start services

```bash
make up
```
### Check service health

```bash
curl http://localhost:8080/health
```

### Check database connectivity

```bash
curl http://localhost:8080/debug/db
```

### Example health response

```json
{
  "env": "dev",
  "mysql": "true",
  "redis": "true",
  "rabbitmq": "true",
  "ok": true
}
```
