# Implementation Plan

## Phase 1: Project Foundation ✅

**PR 1.1: Initialize Go module and configuration** ✅
- ~~Initialize `go.mod` with module name~~
- ~~Create `internal/config/config.go` with Config struct and environment loading~~
- ~~Create minimal `cmd/server/main.go` that loads config and exits~~
- ~~Add `.gitignore` for Go projects~~

**PR 1.2: Docker Compose for local development** ✅
- ~~Create `docker-compose.yml` with RabbitMQ (with delayed message plugin) and Mailhog~~
- ~~Create `rabbitmq-enabled-plugins` file~~
- ~~Create `.env.example` with all environment variables~~

## Phase 2: CI/CD Pipeline

**PR 2.1: CI workflow for pull requests**
- Create `.github/workflows/ci.yml`
- Build and test on every pull request to main
- Run `go build`, `go test` with race detector and coverage
- Add golangci-lint for static analysis
- Upload coverage reports to Codecov

**PR 2.2: Release workflow**
- Create `.github/workflows/release.yml`
- Trigger on semantic version tags (`v[0-9]+.[0-9]+.[0-9]+`)
- Run tests before release
- Build multi-architecture Docker images (amd64, arm64)
- Push to GitHub Container Registry (ghcr.io)
- Create GitHub Release with auto-generated release notes
- Tag images with semantic versions (e.g., `1.2.3`, `1.2`, `1`)

## Phase 3: RabbitMQ Infrastructure

**PR 3.1: RabbitMQ connection management**
- Create `internal/queue/rabbitmq.go` with connection handling
- Implement connection with automatic reconnection logic
- Add connection health check method

**PR 3.2: Queue setup and declaration**
- Implement `QueueManager.SetupQueues()` in `internal/queue/rabbitmq.go`
- Declare exchanges (`notifications.delay`, `notifications.dlx`)
- Declare queues and bindings for SMS and Email
- Verify delayed message plugin availability

**PR 3.3: Message consumer**
- Create `internal/queue/consumer.go`
- Implement consumer that reads from notification queues
- Handle message acknowledgment and rejection
- Support prefetch configuration

**PR 3.4: Message publisher with delay support**
- Create `internal/queue/publisher.go`
- Implement `PublishWithDelay()` for retry messages
- Implement `PublishToDLQ()` for dead letter routing

## Phase 4: Core Handler Framework

**PR 4.1: Handler and Provider interfaces**
- Create `internal/handler/handler.go` with `NotificationHandler` interface and `Message`/`Result` types
- Create `internal/provider/provider.go` with `NotificationProvider` interface and `SendRequest`/`SendResult` types

**PR 4.2: Notification router**
- Create `internal/handler/router.go`
- Implement handler registration and lookup by message type
- Route incoming messages to appropriate handlers

**PR 4.3: Retry logic**
- Implement `calculateDelay()` with exponential backoff and jitter
- Integrate retry logic into consumer message processing
- Handle retryable vs non-retryable error classification
- Route to DLQ after max retries exceeded

## Phase 5: SMS Notifications

**PR 5.1: Twilio provider**
- Create `internal/provider/twilio.go`
- Implement `Send()` and `SendBatch()` using Twilio SDK
- Implement `HealthCheck()` for Twilio API
- Classify Twilio errors as retryable/non-retryable

**PR 5.2: SMS handler**
- Create `internal/handler/sms.go`
- Implement SMS payload validation (E.164 phone format)
- Process SMS messages through Twilio provider
- Unit tests with mocked provider

## Phase 6: Email Notifications

**PR 6.1: Template engine**
- Create `internal/template/template.go`
- Load and parse HTML templates from filesystem
- Render templates with variable substitution
- Template caching for performance

**PR 6.2: SMTP provider**
- Create `internal/provider/smtp.go`
- Implement `Send()` and `SendBatch()` via SMTP
- Support TLS and authentication
- Implement `HealthCheck()` for SMTP connection

**PR 6.3: Email handler**
- Create `internal/handler/email.go`
- Implement email payload validation
- Integrate template rendering
- Process emails through SMTP provider
- Unit tests with mocked provider

## Phase 7: Observability

**PR 7.1: Structured logging**
- Add `log/slog` based structured logging throughout application
- Include message_id, message_type, correlation_id, retry_count in log context
- Log levels: debug, info, warn, error

**PR 7.2: Prometheus metrics**
- Add `github.com/prometheus/client_golang` dependency
- Implement metrics: `notifications_processed_total`, `notifications_retry_total`, `notifications_dlq_total`
- Add histograms: `notification_processing_duration_seconds`, `provider_request_duration_seconds`
- Expose `/metrics` endpoint

**PR 7.3: Health check endpoints**
- Create HTTP server for health endpoints
- Implement `/health` liveness probe
- Implement `/ready` readiness probe (checks RabbitMQ connection)

## Phase 8: Production Readiness

**PR 8.1: Graceful shutdown**
- Handle SIGTERM and SIGINT signals
- Stop consuming new messages
- Wait for in-flight messages to complete (with timeout)
- Close connections cleanly

**PR 8.2: Dockerfile**
- Create multi-stage Dockerfile
- Build static binary
- Copy templates to final image
- Run as non-root user

**PR 8.3: Integration tests**
- Test RabbitMQ message flow end-to-end
- Test retry behavior with delayed messages
- Test DLQ routing
- Use testcontainers or docker-compose for test infrastructure

## Creating a Release

```bash
git checkout main
git pull origin main
git tag v1.0.0
git push origin v1.0.0
```
