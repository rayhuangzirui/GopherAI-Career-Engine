# GopherAI Career Engine

GopherAI Career Engine is a backend project I built to practice async processing in a more realistic workflow than a normal CRUD app.

Instead of doing resume analysis in one request, the API creates a task, pushes it to RabbitMQ, and lets a worker process it in the background. I used this project to practice queue-based design, retries, idempotency, caching, file upload, and object storage in one system. 
I also wanted to put the LLM part inside a more controlled backend pipeline.

## Stack

- Go, Gin
- MySQL
- RabbitMQ
- Redis
- Docker Compose
- Next.js / React / TypeScript
- DashScope OpenAI-compatible API
- Amazon S3

## What it can do

- create resume analysis and resume/JD match tasks
- process tasks asynchronously with API/worker separation
- track task states from `pending` to `completed` or `permanently_failed`
- retry failed tasks with backoff
- avoid duplicate finalization with attempt-level idempotency keys
- support both direct text input and uploaded `.txt` files
- store uploaded files and result artifacts in local storage or S3
- cache task reads and rate-limit task creation with Redis
- show results and task history in a simple frontend demo

## Core Task Flow

1. A user submits a resume-analysis or resume/JD matching request
2. The API creates a task record in MySQL
3. The API publishes a message to RabbitMQ
4. A worker consumes the message and moves the task into `processing`
5. The worker resolves input from either:
    - raw text in the task payload, or
    - uploaded file(s) referenced by `file_key`
6. The worker runs the analyzer
7. On temporary failure, the task is retried with delayed backoff
8. On success:
    - the result is stored in MySQL as `result_payload`
    - the result artifact is also persisted to external storage
9. On repeated or non-retryable failure, the task becomes `permanently_failed`

## RabbitMQ usage

RabbitMQ is used to decouple task creation from task execution.

When the API receives a resume analysis request, it first creates a task record in MySQL and then publishes a message to RabbitMQ. A worker consumes the message later and processes the task in the background.

I used RabbitMQ here mainly to support a more realistic async workflow:

- the API does not have to wait for analysis to finish before responding
- workers can process tasks separately from the request path
- failed tasks can be retried with delayed backoff
- multiple workers can consume from the same queue
- duplicate message handling is controlled with attempt-level idempotency keys

## LLM integration

The project can run in either `rules` mode or `llm` mode.

For LLM mode, it uses DashScope through an OpenAI-compatible API style. In other words, the provider is DashScope, but the request/response shape follows an OpenAI-style chat API pattern.

I also tried to keep the LLM part bounded instead of letting it control the whole system:

- resume/JD text is treated as untrusted input data, not as instructions
- prompts tell the model to ignore instructions embedded inside user content
- model output is expected to follow a fixed JSON shape
- output is validated before the result is accepted
- the application code still decides task behavior, retries, and final status
- a rules-based fallback is available when LLM output is not usable

So the LLM is one part of the pipeline, not the source of truth for system behavior.

## MySQL usage

MySQL is the main source of truth in this project.

It stores the persistent state for the async workflow, while RabbitMQ is used for message delivery and Redis is used for caching and rate limiting.

In this project, MySQL is mainly used for:

- task records and task state transitions
- retry counts and error messages
- final analysis results
- upload metadata for user files
- processed message keys for idempotency checks

I wanted task status and final results to be durable, so the system does not depend on in-memory state or queue state to know what has happened.

So MySQL is mainly there to keep the async pipeline reliable and queryable, especially for task history, retries, and final result lookup.

## Storage

Storage is used in this project for files that should live outside the main database.

I added a small storage abstraction so the same workflow can work with either `local` filesystem storage or `AWS S3`.

In this project, storage is mainly used for:

- local filesystem storage
- AWS S3

Storage is used for:

- uploaded resume / JD files
- completed task result artifacts
 
I wanted file inputs and generated artifacts to be stored separately from task state, so MySQL can stay focused on structured application data while storage handles file persistence.

So the storage layer is mainly there to support file-based workflows and make the system easier to extend to cloud object storage.

## Redis usage

Redis is used for a few lightweight backend features around the main task pipeline.

I did not use Redis as the source of truth for task state. Instead, it is used to support request-side protection and reduce repeated reads.

In this project, Redis is mainly used for:

- per-user rate limiting on analysis endpoints
- short-lived caching for task detail reads
- short-lived caching for task history reads
- short-lived caching for final result reads

I wanted these parts to stay fast without changing the main task model, so MySQL still keeps the durable state while Redis handles temporary data.

So Redis is mainly there to protect the async endpoints and reduce repeated database reads during frontend polling.
## API examples

- `POST /uploads`
- `POST /tasks/resume-analysis`
- `POST /tasks/resume-jd-match`
- `GET /tasks`
- `GET /tasks/:id`
- `GET /tasks/:id/result`
- `GET /health`

## Local run

From the project root:

```bash
docker compose up -d --build
```

Frontend:

```bash
cd web
npm install
npm run dev
```

## Demo flow

Upload files:

```bash
curl -s -X POST "http://localhost:8080/uploads" \
  -F "user_id=1" \
  -F "kind=resume" \
  -F "file=@/path/to/resume.txt"

curl -s -X POST "http://localhost:8080/uploads" \
  -F "user_id=1" \
  -F "kind=jd" \
  -F "file=@/path/to/jd.txt"
```

Create a task with uploaded file keys:

```bash
curl -s -X POST "http://localhost:8080/tasks/resume-jd-match" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": 1,
    "resume_file_key": "uploads/resumes/user-1/...",
    "job_description_file_key": "uploads/jds/user-1/..."
  }'
```

## Benchmark

In local rules mode with 2 workers:

- **100 tasks submitted**
- **80 completed**
- **20 permanently_failed** (expected injected failures)
- **0 unfinished**
- **Task creation latency:** 60.9 ms avg, 60.4 ms p50, 81.9 ms p95
- **Successful completion latency:** 3553.9 ms avg, 3578.8 ms p50, 5808.1 ms p95
- **Failure-path latency:** 48.2 s avg due to delayed retries
- **Task behavior validation passed**
- **`processed_keys` / idempotency validation passed**
- **No duplicate finalization observed**

## Current limitations

- no auth yet
- upload parsing is still minimal (`.txt` only right now)
- no OCR or image parsing
- not deployed to production

## Why I made it

I wanted one project where I could practice backend topics that show up a lot in real systems, especially async workflows, retries, caching, and storage.

I also wanted the LLM part to sit inside a more controlled backend pipeline instead of building a project that was just one API call plus a prompt.
