# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

A Go notification service that consumes messages from RabbitMQ and dispatches notifications via SMS (Twilio) and Email (SMTP). Uses the RabbitMQ Delayed Message Exchange plugin for retry logic with exponential backoff.

## Development Workflow

**IMPORTANT: This workflow must be followed for ALL changes to this repository.**

For each piece of development work:

```bash
# 1. Start from main with latest changes
git checkout main
git pull origin main

# 2. Create a feature branch
git checkout -b <branch-name>

# 3. Complete the development work
# ... make changes, run tests ...

# 4. Commit changes
git add .
git commit -m "descriptive commit message"

# 5. Push and create PR
git push -u origin <branch-name>
gh pr create --title "PR Title" --body "Description of changes"
```

Branch naming conventions:
- `feature/<description>` - New functionality
- `fix/<description>` - Bug fixes
- `refactor/<description>` - Code improvements

After creating the PR, wait for review and merge. Do not continue to the next task until the PR is merged.

## Build & Run Commands

```bash
# Build
go build -o notification-service ./cmd/server

# Run
./notification-service

# Run tests
go test ./...

# Run single test
go test -run TestName ./internal/handler/

# Run with coverage
go test -coverprofile=coverage.out ./...

# Docker development
docker compose up

# Docker build
docker build -t notification-service .
```

## Architecture

### Message Flow
```
RabbitMQ (notifications.delay exchange) → Consumer → Router → Handler → Provider → External API
                    ↑                                              ↓
                    └────────── Retry with x-delay header ─────────┘
                                                                   ↓
                                              DLQ (after max retries or non-retryable error)
```

### Key Interfaces

**NotificationHandler** (`internal/handler/handler.go`): Processes messages by type
- `Type() string` - Returns handler type (e.g., "sms", "email")
- `Handle(ctx, msg) Result` - Processes the notification
- `Validate(msg) error` - Validates message payload

**NotificationProvider** (`internal/provider/provider.go`): Delivers notifications
- `Send(ctx, req) SendResult` - Single recipient delivery
- `SendBatch(ctx, requests) []SendResult` - Batch delivery
- `HealthCheck(ctx) error` - Provider health check

### Adding New Notification Types

1. Create provider in `internal/provider/` implementing `NotificationProvider`
2. Create handler in `internal/handler/` implementing `NotificationHandler`
3. Register handler in the router
4. Add the notification type to the queue setup loop in `QueueManager.SetupQueues()`

## RabbitMQ Configuration

Requires the **rabbitmq_delayed_message_exchange** plugin.

**Automatic Setup:** The application creates all exchanges and queues on startup if they don't exist. No manual RabbitMQ configuration required.

**Exchanges:**
- `notifications.delay` (x-delayed-message) - Main routing with delay support
- `notifications.dlx` (direct) - Dead letter exchange

**Queues:**
- `notifications.sms` / `notifications.email` - Primary queues
- `notifications.sms.dlq` / `notifications.email.dlq` - Dead letter queues

**Retry Strategy:** Exponential backoff (5s → 10s → 20s) with ±20% jitter, max 3 retries.

## Environment Variables

Required:
- `RABBITMQ_URL` - RabbitMQ connection string

SMS (Twilio):
- `TWILIO_ACCOUNT_SID`, `TWILIO_AUTH_TOKEN`, `TWILIO_FROM_NUMBER`

Email (SMTP):
- `SMTP_HOST`, `SMTP_PORT`, `SMTP_FROM_ADDRESS`
- Optional: `SMTP_USERNAME`, `SMTP_PASSWORD`, `SMTP_USE_TLS`

## Endpoints

- `/health` - Liveness probe
- `/ready` - Readiness probe (checks RabbitMQ connection)
- `/metrics` - Prometheus metrics

---

## Implementation Plan

### Phase 1: Project Foundation

**PR 1.1: Initialize Go module and configuration**
- Initialize `go.mod` with module name
- Create `internal/config/config.go` with Config struct and environment loading
- Create minimal `cmd/server/main.go` that loads config and exits
- Add `.gitignore` for Go projects

**PR 1.2: Docker Compose for local development**
- Create `docker-compose.yml` with RabbitMQ (with delayed message plugin) and Mailhog
- Create `rabbitmq-enabled-plugins` file
- Create `.env.example` with all environment variables

### Phase 2: RabbitMQ Infrastructure

**PR 2.1: RabbitMQ connection management**
- Create `internal/queue/rabbitmq.go` with connection handling
- Implement connection with automatic reconnection logic
- Add connection health check method

**PR 2.2: Queue setup and declaration**
- Implement `QueueManager.SetupQueues()` in `internal/queue/rabbitmq.go`
- Declare exchanges (`notifications.delay`, `notifications.dlx`)
- Declare queues and bindings for SMS and Email
- Verify delayed message plugin availability

**PR 2.3: Message consumer**
- Create `internal/queue/consumer.go`
- Implement consumer that reads from notification queues
- Handle message acknowledgment and rejection
- Support prefetch configuration

**PR 2.4: Message publisher with delay support**
- Create `internal/queue/publisher.go`
- Implement `PublishWithDelay()` for retry messages
- Implement `PublishToDLQ()` for dead letter routing

### Phase 3: Core Handler Framework

**PR 3.1: Handler and Provider interfaces**
- Create `internal/handler/handler.go` with `NotificationHandler` interface and `Message`/`Result` types
- Create `internal/provider/provider.go` with `NotificationProvider` interface and `SendRequest`/`SendResult` types

**PR 3.2: Notification router**
- Create `internal/handler/router.go`
- Implement handler registration and lookup by message type
- Route incoming messages to appropriate handlers

**PR 3.3: Retry logic**
- Implement `calculateDelay()` with exponential backoff and jitter
- Integrate retry logic into consumer message processing
- Handle retryable vs non-retryable error classification
- Route to DLQ after max retries exceeded

### Phase 4: SMS Notifications

**PR 4.1: Twilio provider**
- Create `internal/provider/twilio.go`
- Implement `Send()` and `SendBatch()` using Twilio SDK
- Implement `HealthCheck()` for Twilio API
- Classify Twilio errors as retryable/non-retryable

**PR 4.2: SMS handler**
- Create `internal/handler/sms.go`
- Implement SMS payload validation (E.164 phone format)
- Process SMS messages through Twilio provider
- Unit tests with mocked provider

### Phase 5: Email Notifications

**PR 5.1: Template engine**
- Create `internal/template/template.go`
- Load and parse HTML templates from filesystem
- Render templates with variable substitution
- Template caching for performance

**PR 5.2: SMTP provider**
- Create `internal/provider/smtp.go`
- Implement `Send()` and `SendBatch()` via SMTP
- Support TLS and authentication
- Implement `HealthCheck()` for SMTP connection

**PR 5.3: Email handler**
- Create `internal/handler/email.go`
- Implement email payload validation
- Integrate template rendering
- Process emails through SMTP provider
- Unit tests with mocked provider

### Phase 6: Observability

**PR 6.1: Structured logging**
- Add `log/slog` based structured logging throughout application
- Include message_id, message_type, correlation_id, retry_count in log context
- Log levels: debug, info, warn, error

**PR 6.2: Prometheus metrics**
- Add `github.com/prometheus/client_golang` dependency
- Implement metrics: `notifications_processed_total`, `notifications_retry_total`, `notifications_dlq_total`
- Add histograms: `notification_processing_duration_seconds`, `provider_request_duration_seconds`
- Expose `/metrics` endpoint

**PR 6.3: Health check endpoints**
- Create HTTP server for health endpoints
- Implement `/health` liveness probe
- Implement `/ready` readiness probe (checks RabbitMQ connection)

### Phase 7: Production Readiness

**PR 7.1: Graceful shutdown**
- Handle SIGTERM and SIGINT signals
- Stop consuming new messages
- Wait for in-flight messages to complete (with timeout)
- Close connections cleanly

**PR 7.2: Dockerfile**
- Create multi-stage Dockerfile
- Build static binary
- Copy templates to final image
- Run as non-root user

**PR 7.3: Integration tests**
- Test RabbitMQ message flow end-to-end
- Test retry behavior with delayed messages
- Test DLQ routing
- Use testcontainers or docker-compose for test infrastructure
