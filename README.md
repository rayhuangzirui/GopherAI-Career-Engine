# GopherAI Career Engine

GopherAI Career Engine is an asynchronous resume/JD matching platform.

It takes a resume and a job description, creates an analysis task, processes it in the background, and returns a user-friendly match report. The system includes task state tracking, delayed retries, attempt-level idempotency, and an LLM-backed analysis path with a deterministic rules fallback.

## Tech Stack

- **Backend:** Go, Gin
- **Queue / Infra:** RabbitMQ, Docker Compose
- **Database:** MySQL
- **Frontend:** Next.js, React, TypeScript
- **Analysis:** LLM-backed matcher + rules-based fallback

## What it does

- Submit resume/JD matching tasks through the frontend or API
- Process tasks asynchronously with background workers
- Track task states from `queued` to `completed` / `permanently_failed`
- Retry failed tasks with delayed backoff
- Prevent duplicate finalization with attempt-level idempotency keys
- Return a readable report instead of raw JSON
- Show analysis history in the frontend

## Architecture

```text
Next.js Frontend
      |
      v
 Go API (Gin)
      |
      v
  RabbitMQ Queue
      |
      v
 Go Workers
      |
      +--> LLM analyzer
      |      |
      |      +--> rules fallback
      |
      v
    MySQL
```

## Task flow

1. A user submits a resume and a job description
2. The API creates a task record in MySQL and publishes a queue message
3. A worker picks up the task and updates its state
4. The worker runs the analyzer
5. On temporary failure, the task is retried with delayed backoff
6. On success, the result is stored and shown in the frontend
7. On repeated failure, the task becomes `permanently_failed`

## Key features

- Async task processing with API / worker separation
- Explicit task states:
  - `pending`
  - `queued`
  - `processing`
  - `retrying`
  - `completed`
  - `permanently_failed`
- Delayed retry with capped backoff
- Attempt-level idempotency via `processed_keys`
- LLM output guardrails:
  - input sanitization
  - bounded prompts
  - validated JSON output
  - rules fallback
- Frontend polling for real-time status updates
- Demo workspace with historical analyses

## Benchmark

Local rules-mode benchmark with **2 workers**:

- **100 tasks submitted**
- **80 completed**
- **20 permanently_failed** (expected injected failures)
- **0 unfinished**
- **Task creation latency:** 60.9 ms avg, 60.4 ms p50, 81.9 ms p95
- **Successful completion latency:** 3553.9 ms avg, 3578.8 ms p50, 5808.1 ms p95
- **Failure-path latency:** 48.2 s avg due to delayed retries
- **Task behavior validation passed**
- **processed_keys / idempotency validation passed**
- **No duplicate finalization observed**

## Frontend

The frontend is a small Next.js workspace for:

- submitting a resume/JD analysis
- showing task status changes
- rendering a readable match report
- viewing recent analysis history

Current UX focus is the end-to-end workflow, not authentication.

## Demo notes

This project currently uses a **single pre-seeded demo user** for the frontend workflow.

Authentication is intentionally deferred so the project can stay focused on async processing, reliability, and LLM-backed analysis.

## Local setup

### 1. Start backend services

From the project root:

```bash
docker compose up --build -d
```

### 2. Run the frontend

```bash
cd web
npm install
npm run dev
```

Frontend:
- `http://localhost:3000`

Backend API:
- `http://localhost:8080`

## Environment variables

Typical backend/frontend configuration includes:

- `MYSQL_DSN`
- `RABBITMQ_URL`
- `ANALYZER_MODE` (`rules` or `llm`)
- `LLM_PROVIDER`
- `LLM_BASE_URL`
- `LLM_MODEL`
- `DASHSCOPE_API_KEY`

For LLM mode, the current setup uses DashScope's OpenAI-compatible API.

## Modes

### Rules mode
Used for deterministic local benchmarking and failure-path validation.

### LLM mode
Used for real resume/JD matching analysis.
The current implementation supports:
- input sanitization
- JSON schema validation
- bounded outputs
- rules fallback when LLM output is unusable

## Example API endpoints

- `POST /tasks/resume-analysis`
- `POST /tasks/resume-jd-match`
- `GET /tasks`
- `GET /tasks/:id`
- `GET /tasks/:id/result`
- `GET /health`

## Current scope

Included:
- async backend pipeline
- retry and idempotency handling
- LLM-backed analysis flow
- frontend demo workspace
- analysis history

Not included yet:
- user authentication
- image/OCR upload
- production deployment
- DLQ / outbox patterns

## Why I built it

I wanted a project that was more than a single API endpoint or a toy LLM demo.

The goal was to build a backend-heavy system that could:
- handle long-running work asynchronously
- recover from failure
- keep state consistent
- expose a clean user workflow on top of it

That made it a better fit for backend and full-stack SWE roles than a simple one-shot analysis app.
