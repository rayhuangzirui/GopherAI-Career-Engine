# GopherAI-Career-Engine

GopherAI-Career-Engine is a Dockerized Go backend for asynchronous career-analysis tasks. It uses RabbitMQ to decouple request handling from background processing, MySQL to persist task state and results, and Redis as supporting infrastructure for future extensions.

## Features

- Asynchronous task processing with separate API and worker services
- Multiple task types:
    - `resume_analysis`
    - `resume_jd_match`
- Task lifecycle tracking:
    - `pending`
    - `queued`
    - `processing`
    - `completed`
    - `failed`
- Result persistence and task result query endpoints
- Idempotent consumer handling with `processed_keys`
- Versioned SQL migrations as the source of truth for schema
- Reproducible local development with Docker Compose
- Local validation with multiple worker instances

## Tech Stack

- **Go** + **Gin**
- **MySQL**
- **RabbitMQ**
- **Redis**
- **GORM**
- **SQL migrations**
- **Docker Compose**

## Architecture

1. Client submits a task request
2. API stores the task in MySQL
3. API publishes a message to RabbitMQ
4. Worker consumes the message and processes the task
5. Worker updates task state and stores the result
6. Client queries task status or result through the API

## Current Endpoints

- `POST /tasks/resume-analysis`
- `POST /tasks/resume-jd-match`
- `GET /tasks`
- `GET /tasks/:id`
- `GET /tasks/:id/result`
- `GET /health`
- `GET /debug/db`

## Local Development

### Start services

```bash
docker compose up -d --build
```

### Run migrations

```bash
make migrate-up
```

### Example requests

Create a resume analysis task:

```bash
curl -s -X POST http://localhost:8080/tasks/resume-analysis \
  -H "Content-Type: application/json" \
  -d '{"user_id":1,"resume_text":"Backend engineer with Go, MySQL, Redis, RabbitMQ, Docker, and REST API experience."}'
```

Create a resume-JD match task:

```bash
curl -s -X POST http://localhost:8080/tasks/resume-jd-match \
  -H "Content-Type: application/json" \
  -d '{"user_id":1,"resume_text":"Backend engineer with Go, MySQL, Redis, RabbitMQ, Docker, and REST API experience.","job_description_text":"We are looking for a backend engineer with Go, Docker, AWS, Kubernetes, and distributed systems experience."}'
```

List tasks:

```bash
curl -s "http://localhost:8080/tasks?user_id=1&limit=10"
```

## Notes

- The worker supports multiple task types through a shared queue-based pipeline.
- Duplicate message processing is mitigated with `processed_keys`.
- Failed tasks persist error information and retry counts.
- The current implementation uses a mock analyzer for deterministic local testing.

## Roadmap

- Automatic retry policy for failed tasks
- More realistic analysis backend
- Additional task types
- Further cloud deployment and scaling improvements
