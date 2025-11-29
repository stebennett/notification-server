# Notification Server

A Go service that consumes messages from RabbitMQ and dispatches notifications via SMS (Twilio) and Email (SMTP). Features retry logic with exponential backoff using the RabbitMQ Delayed Message Exchange plugin.

## Features

- **Multi-channel notifications**: SMS via Twilio, Email via SMTP
- **Reliable delivery**: Automatic retries with exponential backoff and jitter
- **Dead letter queues**: Failed messages are preserved for investigation
- **Template support**: HTML email templates with variable substitution
- **Observability**: Prometheus metrics, structured logging, health checks

## Quick Start

### Prerequisites

- Go 1.23+
- Docker and Docker Compose

### Local Development

1. Start the infrastructure services:

```bash
docker compose up -d
```

This starts:
- **RabbitMQ** on `localhost:5672` (management UI at http://localhost:15672, login: guest/guest)
- **Mailhog** for email testing (web UI at http://localhost:8025)

2. Copy the example environment file:

```bash
cp .env.example .env
```

3. Build and run the service:

```bash
go build -o notification-service ./cmd/server
RABBITMQ_URL=amqp://guest:guest@localhost:5672/ ./notification-service
```

## Configuration

Configuration is loaded from environment variables. See `.env.example` for all available options.

### Required

| Variable | Description |
|----------|-------------|
| `RABBITMQ_URL` | RabbitMQ connection string |

### SMS (Twilio)

| Variable | Description |
|----------|-------------|
| `TWILIO_ACCOUNT_SID` | Twilio account SID |
| `TWILIO_AUTH_TOKEN` | Twilio auth token |
| `TWILIO_FROM_NUMBER` | Default sender phone number |

### Email (SMTP)

| Variable | Description | Default |
|----------|-------------|---------|
| `SMTP_HOST` | SMTP server hostname | - |
| `SMTP_PORT` | SMTP server port | 587 |
| `SMTP_USERNAME` | SMTP username | - |
| `SMTP_PASSWORD` | SMTP password | - |
| `SMTP_USE_TLS` | Enable TLS | true |
| `SMTP_FROM_ADDRESS` | Default sender email | - |

## Architecture

```
RabbitMQ (notifications.delay exchange) → Consumer → Router → Handler → Provider → External API
                    ↑                                              ↓
                    └────────── Retry with x-delay header ─────────┘
                                                                   ↓
                                              DLQ (after max retries or non-retryable error)
```

### Message Flow

1. Messages are published to the `notifications.delay` exchange with a routing key (`sms` or `email`)
2. The consumer picks up messages and routes them to the appropriate handler
3. Handlers validate the message and use providers to send notifications
4. On failure, retryable errors trigger republishing with an exponential backoff delay
5. After max retries (default: 3), messages are sent to the dead letter queue

### Retry Strategy

- **Initial delay**: 5 seconds
- **Backoff multiplier**: 2x
- **Max delay**: 60 seconds
- **Jitter**: ±20% to prevent thundering herd

Example progression: 5s → 10s → 20s (with jitter applied)

## Development

### Build Commands

```bash
# Build
go build -o notification-service ./cmd/server

# Run tests
go test ./...

# Run tests with coverage
go test -coverprofile=coverage.out ./...

# Run a specific test
go test -run TestName ./internal/handler/
```

### Docker

```bash
# Start development services
docker compose up -d

# Stop services
docker compose down

# Build Docker image
docker build -t notification-service .
```

## API Endpoints

| Endpoint | Description |
|----------|-------------|
| `/health` | Liveness probe - returns 200 if process is running |
| `/ready` | Readiness probe - returns 200 if RabbitMQ connection is healthy |
| `/metrics` | Prometheus metrics |

## License

MIT
