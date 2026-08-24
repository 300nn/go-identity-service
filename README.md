# go-identity-service

Production-like identity service written in Go.

The project demonstrates how to build a backend service with authentication, PostgreSQL persistence, Redis cache and rate limiting, Kafka-based transactional outbox, gRPC API, Prometheus metrics, Docker packaging, and GitHub Actions CI/CD.

## Features

- User registration and authentication
- JWT access tokens
- Refresh token rotation
- Role-based authorization
- PostgreSQL persistence
- Database migrations with Goose
- Redis-backed user cache
- Redis-backed rate limiting
- Transactional outbox pattern
- Kafka producer and consumer with franz-go
- Protobuf events
- gRPC API
- HTTP REST API
- Prometheus metrics
- Health and readiness probes
- Graceful shutdown
- Docker and Docker Compose support
- GitHub Actions CI/CD
- Docker image publishing to GitHub Container Registry

## Tech Stack

- Go
- PostgreSQL
- Redis
- Apache Kafka
- franz-go
- gRPC
- Protobuf / Buf
- Prometheus
- Docker / Docker Compose
- GitHub Actions
- GitHub Container Registry

## Architecture Overview

The service is built around a modular internal structure:

```text
cmd/
  api/           HTTP + gRPC application entrypoint
  migrate/       Database migration runner
  healthcheck/   Container healthcheck binary

internal/
  app/           Application composition root
  auth/          JWT, password hashing, refresh tokens, auth middleware
  user/          User domain, repository, service, cache
  outbox/        Transactional outbox worker and Kafka publisher
  kafkaconsumer/ Kafka consumer, idempotency, audit handling
  grpcapi/       gRPC service and interceptors
  middleware/    HTTP middleware
  metrics/       Prometheus metrics
  postgres/      PostgreSQL connection pool
  redisclient/   Redis client
  config/        Application configuration
  validation/    Request validation
```

The API writes domain changes and outbox events in the same PostgreSQL transaction. A background outbox worker publishes events to Kafka. Kafka consumers process events idempotently and store audit records.

## Local Development

### Requirements

- Go
- Docker
- Docker Compose
- Buf CLI

### Start dependencies and service

```bash
docker compose up --build
```

The service exposes:

```text
HTTP:  http://localhost:8080
gRPC:  localhost:50051
```

### Health checks

```bash
curl http://localhost:8080/health
curl http://localhost:8080/ready
```

### Metrics

```bash
curl http://localhost:8080/metrics
```

## Production-like Docker Compose

The project provides a production-like compose file that runs the published Docker image from GitHub Container Registry.

Create local environment file:

```bash
cp .env.prod.example .env.prod
```

Update at least:

```env
DATABASE_PASSWORD=change-me
AUTH_JWT_SECRET=change-me-use-real-random-secret-at-least-64-characters-long
IMAGE_TAG=latest
```

Start the stack:

```bash
docker compose -f docker-compose.prod.yml --env-file .env.prod pull
docker compose -f docker-compose.prod.yml --env-file .env.prod up -d
```

Run migrations manually if needed:

```bash
docker compose -f docker-compose.prod.yml --env-file .env.prod run --rm migrate
```

Stop the stack:

```bash
docker compose -f docker-compose.prod.yml --env-file .env.prod down
```

Remove volumes:

```bash
docker compose -f docker-compose.prod.yml --env-file .env.prod down -v
```

## Database Migrations

Migrations are stored in:

```text
migrations/
```

The production-like compose file runs migrations through a separate `migrate` service before starting the API container.

The migration binary is included in the Docker image:

```text
/app/migrate
```

## REST API

The service provides REST endpoints for user and authentication flows.

Main endpoint groups:

```text
/auth
/users
/health
/ready
/metrics
```

Example registration request:

```bash
curl -X POST http://localhost:8080/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Alex",
    "email": "alex@example.com",
    "age": 25,
    "password": "password123"
  }'
```

The OpenAPI specification is available in:

```text
docs/openapi.yml
```

It describes authentication, user management, health/readiness endpoints, error responses, and JWT bearer
authentication.

## gRPC API

The gRPC API is defined in:

```text
proto/api/user/v1/user_service.proto
```

Generated Go files are committed under:

```text
internal/gen/api/user/v1/
```

The service exposes gRPC on:

```text
localhost:50051
```

## Kafka and Transactional Outbox

The service uses the transactional outbox pattern.

Flow:

```text
1. API handles command
2. PostgreSQL transaction writes domain data
3. The same transaction writes an outbox event
4. Outbox worker publishes the event to Kafka
5. Kafka consumer processes the event idempotently
6. Audit/read-side data is written to PostgreSQL
```

Kafka topic:

```text
outbox.events
```

Events are serialized with Protobuf.

## Observability

The service exposes Prometheus metrics at:

```text
GET /metrics
```

Metrics include:

- HTTP request count and duration
- gRPC request count and duration
- Outbox processed/failed events
- Kafka consumer processed/failed events
- Redis cache hits/misses

Health endpoints:

```text
GET /health
GET /ready
```

`/health` is a liveness probe.

`/ready` checks dependencies such as PostgreSQL, Redis, Kafka, and shutdown state.

## Security Notes

Implemented:

- Password hashing with bcrypt
- JWT access tokens
- Refresh token rotation
- Role-based authorization
- Redis rate limiting
- CORS configuration
- Security headers
- Production config validation
- govulncheck in CI
- gosec in CI

Secrets are not committed.

Use example files as templates:

```text
.env.goose.example
.env.prod.example
config.example.yml
```

Local secret files are ignored:

```text
.env
.env.prod
config.yml
```

## CI/CD

GitHub Actions checks:

- gofmt
- goimports
- tests
- govulncheck
- gosec
- Buf format
- Buf lint
- Protobuf generation
- Docker build

The Docker image is published to GitHub Container Registry:

```text
ghcr.io/300nn/go-identity-service
```

Release workflow is triggered by semantic version tags:

```bash
git tag v0.1.0
git push origin v0.1.0
```

Release tags produce Docker image tags such as:

```text
ghcr.io/300nn/go-identity-service:0.1.0
ghcr.io/300nn/go-identity-service:0.1
ghcr.io/300nn/go-identity-service:0
```

## Development Commands

Format code:

```bash
gofmt -w ./cmd ./internal
goimports -local github.com/300nn/go-identity-service -w ./cmd ./internal
```

Run tests:

```bash
go test ./... -short
```

Run security checks:

```bash
govulncheck ./...
gosec -exclude-generated ./...
```

Run Protobuf checks:

```bash
buf format -w
buf lint
buf generate
```

Build Docker image:

```bash
docker build -t identity-service:local .
```

### Production-like smoke test

```bash
powershell -ExecutionPolicy Bypass -File .\scripts\smoke-prod.ps1
```

The smoke test starts the production-like Docker Compose stack, waits for /ready, checks /health and /metrics, and
verifies that database migrations were applied.

## Roadmap

Planned improvements:

- Add database operation timeouts to repositories
- Add OpenTelemetry tracing
- Improve REST API examples
- Add architecture diagram