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