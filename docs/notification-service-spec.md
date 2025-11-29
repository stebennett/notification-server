# Notification Service Specification

## Overview

A Go application that consumes messages from RabbitMQ queues and dispatches notifications through various channels. The service is designed with extensibility in mind, allowing new notification methods to be added with minimal changes to the core architecture.

## Architecture

### High-Level Design

```
┌─────────────────────────┐     ┌──────────────────────────────────────────────┐
│                         │     │            Notification Service              │
│        RabbitMQ         │     │  ┌────────────────────────────────────────┐  │
│                         │     │  │           Message Consumer             │  │
│  ┌───────────────────┐  │     │  └────────────────┬───────────────────────┘  │
│  │ notifications.sms │──┼─────┼──────────────────►│                          │
│  └───────────────────┘  │     │                   ▼                          │
│           ▲             │     │  ┌────────────────────────────────────────┐  │
│           │ (delayed)   │     │  │          Notification Router           │  │
│  ┌────────┴──────────┐  │     │  └────────────────┬───────────────────────┘  │
│  │notifications.delay│  │     │                   │                          │
│  └────────┬──────────┘  │     │         ┌─────────┴─────────┐                │
│           │ (delayed)   │     │         ▼                   ▼                │
│           ▼             │     │  ┌─────────────┐    ┌─────────────┐          │
│  ┌───────────────────┐  │     │  │ SMS Handler │    │Email Handler│          │
│  │notifications.email│──┼─────┼──►└──────┬──────┘    └──────┬──────┘          │
│  └───────────────────┘  │     │         │                   │                │
│                         │     │         ▼                   ▼                │
│  ┌───────────────────┐  │     │  ┌─────────────┐    ┌─────────────┐          │
│  │notifications.sms  │  │     │  │ Twilio API  │    │ SMTP Server │          │
│  │       .dlq        │  │     │  └─────────────┘    └─────────────┘          │
│  └───────────────────┘  │     └──────────────────────────────────────────────┘
│                         │
│  ┌───────────────────┐  │
│  │notifications.email│  │
│  │       .dlq        │  │
│  └───────────────────┘  │
│                         │
└─────────────────────────┘
```

### Project Structure

```
notification-service/
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── config/
│   │   └── config.go
│   ├── queue/
│   │   ├── consumer.go
│   │   ├── publisher.go
│   │   └── rabbitmq.go
│   ├── handler/
│   │   ├── handler.go          # Interface definition
│   │   ├── router.go           # Routes messages to handlers
│   │   ├── sms.go
│   │   └── email.go
│   ├── provider/
│   │   ├── provider.go         # Interface definition
│   │   ├── twilio.go
│   │   └── smtp.go
│   └── template/
│       └── template.go
├── templates/
│   └── email/
│       └── *.html
├── rabbitmq-enabled-plugins
├── Dockerfile
├── docker-compose.yml
├── go.mod
├── go.sum
└── README.md
```

## Message Schemas

### SMS Message

```json
{
  "id": "uuid-v4",
  "type": "sms",
  "content": "Your verification code is 123456",
  "recipients": [
    "+447700900123",
    "+447700900456"
  ],
  "metadata": {
    "correlation_id": "optional-correlation-id",
    "priority": "normal"
  },
  "retry_count": 0,
  "created_at": "2025-01-15T10:30:00Z"
}
```

### Email Message

```json
{
  "id": "uuid-v4",
  "type": "email",
  "template_name": "welcome_email",
  "template_variables": {
    "user_name": "John Doe",
    "activation_link": "https://example.com/activate?token=abc123",
    "company_name": "Acme Corp"
  },
  "recipients": [
    "john.doe@example.com",
    "jane.doe@example.com"
  ],
  "subject": "Welcome to Acme Corp",
  "from_address": "noreply@acme.com",
  "metadata": {
    "correlation_id": "optional-correlation-id",
    "priority": "normal"
  },
  "retry_count": 0,
  "created_at": "2025-01-15T10:30:00Z"
}
```

## Core Interfaces

### NotificationHandler Interface

All notification handlers must implement this interface to enable extensibility.

```go
package handler

import "context"

type Message struct {
    ID         string            `json:"id"`
    Type       string            `json:"type"`
    RetryCount int               `json:"retry_count"`
    CreatedAt  time.Time         `json:"created_at"`
    Metadata   map[string]string `json:"metadata"`
    Payload    json.RawMessage   `json:"payload"`
}

type Result struct {
    Success      bool
    Error        error
    Retryable    bool
    FailedItems  []string  // For partial failures (e.g., some recipients failed)
}

type NotificationHandler interface {
    // Type returns the notification type this handler processes
    Type() string
    
    // Handle processes the notification message
    Handle(ctx context.Context, msg *Message) Result
    
    // Validate checks if the message payload is valid
    Validate(msg *Message) error
}
```

### NotificationProvider Interface

Providers handle the actual delivery mechanism.

```go
package provider

import "context"

type SendRequest struct {
    Recipient string
    Content   string
    Metadata  map[string]string
}

type SendResult struct {
    Success   bool
    MessageID string
    Error     error
}

type NotificationProvider interface {
    // Send delivers a notification to a single recipient
    Send(ctx context.Context, req SendRequest) SendResult
    
    // SendBatch delivers notifications to multiple recipients
    SendBatch(ctx context.Context, requests []SendRequest) []SendResult
    
    // HealthCheck verifies the provider is operational
    HealthCheck(ctx context.Context) error
}
```

## Queue Configuration

### Prerequisites

This service requires the **RabbitMQ Delayed Message Exchange Plugin** (`rabbitmq_delayed_message_exchange`). Ensure it is enabled on your RabbitMQ instance:

```bash
rabbitmq-plugins enable rabbitmq_delayed_message_exchange
```

### Plugin Limitations

Be aware of the following limitations documented by the plugin maintainers:

- **Not for high volumes** - The plugin is not designed for scenarios with hundreds of thousands or millions of delayed messages. For this notification service with retry delays of seconds to minutes, this is acceptable.
- **Delay range** - Delays must be between 0 and (2^32)-1 milliseconds (~49 days). Messages outside this range are routed immediately.
- **Single node storage** - Delayed messages are stored in Mnesia with a single disk replica on the declaring node. Losing that node or disabling the plugin will lose pending delayed messages.
- **No mandatory flag support** - The `mandatory` flag is not supported on this exchange type.
- **Designed for short delays** - The plugin is intended for delays of seconds, minutes, or hours (a day or two at most), not long-term scheduling.

### RabbitMQ Setup

The service requires the following queues and exchanges:

| Exchange | Type | Queue | Routing Key | Purpose |
|----------|------|-------|-------------|---------|
| notifications.delay | x-delayed-message | notifications.sms | sms | SMS notifications (with delayed retry support) |
| notifications.delay | x-delayed-message | notifications.email | email | Email notifications (with delayed retry support) |
| notifications.dlx | direct | notifications.sms.dlq | sms | SMS dead letters |
| notifications.dlx | direct | notifications.email.dlq | email | Email dead letters |

### Exchange Configuration

The delayed message exchange must be declared with the underlying exchange type:

```go
args := amqp.Table{
    "x-delayed-type": "direct",
}

err := channel.ExchangeDeclare(
    "notifications.delay",    // name
    "x-delayed-message",      // type
    true,                     // durable
    false,                    // auto-deleted
    false,                    // internal
    false,                    // no-wait
    args,                     // arguments
)
```

### Queue Arguments

Primary queues should be configured with dead-letter routing:

```go
args := amqp.Table{
    "x-dead-letter-exchange":    "notifications.dlx",
    "x-dead-letter-routing-key": routingKey,
}
```

### Automatic Queue Setup

The application automatically creates all required exchanges and queues on startup if they do not already exist. This ensures the service can be deployed without manual RabbitMQ configuration.

#### Startup Initialization Sequence

1. Connect to RabbitMQ
2. Verify the delayed message exchange plugin is available
3. Declare exchanges (idempotent - safe if already exists):
   - `notifications.delay` (x-delayed-message with x-delayed-type: direct)
   - `notifications.dlx` (direct)
4. Declare queues with appropriate arguments (idempotent):
   - `notifications.sms` (with dead-letter routing to `notifications.dlx`)
   - `notifications.email` (with dead-letter routing to `notifications.dlx`)
   - `notifications.sms.dlq`
   - `notifications.email.dlq`
5. Bind queues to exchanges:
   - `notifications.sms` → `notifications.delay` with routing key `sms`
   - `notifications.email` → `notifications.delay` with routing key `email`
   - `notifications.sms.dlq` → `notifications.dlx` with routing key `sms`
   - `notifications.email.dlq` → `notifications.dlx` with routing key `email`
6. Begin consuming messages

#### Implementation

```go
func (q *QueueManager) SetupQueues(ctx context.Context) error {
    // Declare delayed message exchange
    if err := q.channel.ExchangeDeclare(
        "notifications.delay",
        "x-delayed-message",
        true,  // durable
        false, // auto-deleted
        false, // internal
        false, // no-wait
        amqp.Table{"x-delayed-type": "direct"},
    ); err != nil {
        return fmt.Errorf("failed to declare delay exchange: %w", err)
    }

    // Declare dead letter exchange
    if err := q.channel.ExchangeDeclare(
        "notifications.dlx",
        "direct",
        true,  // durable
        false, // auto-deleted
        false, // internal
        false, // no-wait
        nil,
    ); err != nil {
        return fmt.Errorf("failed to declare DLX exchange: %w", err)
    }

    // Setup queues for each notification type
    for _, notificationType := range []string{"sms", "email"} {
        if err := q.setupNotificationQueue(notificationType); err != nil {
            return err
        }
    }

    return nil
}

func (q *QueueManager) setupNotificationQueue(notificationType string) error {
    queueName := "notifications." + notificationType
    dlqName := queueName + ".dlq"

    // Declare DLQ first (no special arguments)
    if _, err := q.channel.QueueDeclare(
        dlqName,
        true,  // durable
        false, // auto-delete
        false, // exclusive
        false, // no-wait
        nil,
    ); err != nil {
        return fmt.Errorf("failed to declare DLQ %s: %w", dlqName, err)
    }

    // Bind DLQ to dead letter exchange
    if err := q.channel.QueueBind(
        dlqName,
        notificationType,
        "notifications.dlx",
        false,
        nil,
    ); err != nil {
        return fmt.Errorf("failed to bind DLQ %s: %w", dlqName, err)
    }

    // Declare main queue with dead-letter routing
    if _, err := q.channel.QueueDeclare(
        queueName,
        true,  // durable
        false, // auto-delete
        false, // exclusive
        false, // no-wait
        amqp.Table{
            "x-dead-letter-exchange":    "notifications.dlx",
            "x-dead-letter-routing-key": notificationType,
        },
    ); err != nil {
        return fmt.Errorf("failed to declare queue %s: %w", queueName, err)
    }

    // Bind main queue to delay exchange
    if err := q.channel.QueueBind(
        queueName,
        notificationType,
        "notifications.delay",
        false,
        nil,
    ); err != nil {
        return fmt.Errorf("failed to bind queue %s: %w", queueName, err)
    }

    return nil
}
```

#### Error Handling

- If the delayed message exchange plugin is not installed, the application logs a fatal error and exits
- If queue/exchange declaration fails due to conflicting arguments (e.g., existing queue with different settings), the application logs the conflict and exits
- All declaration operations are idempotent when arguments match existing resources

## Retry Strategy

### Configuration

| Parameter | Default | Description |
|-----------|---------|-------------|
| Max Retries | 3 | Maximum number of retry attempts |
| Initial Delay | 5 seconds | Delay before first retry |
| Backoff Multiplier | 2 | Multiplier for exponential backoff |
| Max Delay | 60 seconds | Maximum delay between retries |
| Jitter Factor | 0.2 | Random jitter as a fraction of calculated delay (±20%) |

### Retry Flow

```
Message Received
       │
       ▼
  ┌─────────┐
  │ Process │
  └────┬────┘
       │
       ▼
   Success? ──Yes──► Acknowledge & Complete
       │
       No
       │
       ▼
  Retryable? ──No──► Send to DLQ
       │
      Yes
       │
       ▼
  retry_count < 3? ──No──► Send to DLQ
       │
      Yes
       │
       ▼
  Increment retry_count
       │
       ▼
  Calculate delay (exponential backoff)
       │
       ▼
  Republish to same queue via delayed
  exchange with x-delay header
       │
       ▼
  Acknowledge original message
```

### Publishing with Delay

When a message needs to be retried, republish it to the delayed message exchange with the `x-delay` header set to the calculated delay in milliseconds. The header value must be an integer. If the `x-delay` header is absent, the message is routed immediately without delay.

```go
func (p *Publisher) PublishWithDelay(ctx context.Context, routingKey string, msg Message, delay time.Duration) error {
    body, err := json.Marshal(msg)
    if err != nil {
        return err
    }

    return p.channel.PublishWithContext(ctx,
        "notifications.delay", // exchange
        routingKey,            // routing key
        false,                 // mandatory (not supported by delayed exchange)
        false,                 // immediate
        amqp.Publishing{
            ContentType:  "application/json",
            DeliveryMode: amqp.Persistent,
            Headers: amqp.Table{
                "x-delay": int64(delay.Milliseconds()),
            },
            Body: body,
        },
    )
}
```

### Delay Calculation

Jitter is added to prevent thundering herd problems when multiple failed messages retry simultaneously. The jitter applies a random adjustment of ±20% to the calculated delay.

```go
func calculateDelay(retryCount int, initialDelay, maxDelay time.Duration, multiplier, jitterFactor float64) time.Duration {
    // Calculate base exponential delay
    delay := float64(initialDelay) * math.Pow(multiplier, float64(retryCount))
    
    // Apply jitter: random value in range [1-jitterFactor, 1+jitterFactor]
    jitter := 1 + (rand.Float64()*2-1)*jitterFactor
    delay = delay * jitter
    
    // Clamp to max delay
    if delay > float64(maxDelay) {
        return maxDelay
    }
    return time.Duration(delay)
}
```

Example delay progression with 20% jitter (±20% of base value):

| Retry | Base Delay | Delay Range |
|-------|------------|-------------|
| 1 | 5s | 4s - 6s |
| 2 | 10s | 8s - 12s |
| 3 | 20s | 16s - 24s |

## SMS Handler Implementation

### Twilio Provider Configuration

| Environment Variable | Description | Required |
|---------------------|-------------|----------|
| TWILIO_ACCOUNT_SID | Twilio account SID | Yes |
| TWILIO_AUTH_TOKEN | Twilio auth token | Yes |
| TWILIO_FROM_NUMBER | Default sender phone number | Yes |

### Processing Logic

1. Validate message payload contains non-empty content and at least one recipient
2. Validate all recipient phone numbers are in E.164 format
3. For each recipient, call Twilio API to send SMS
4. Track success/failure for each recipient
5. If all succeed, acknowledge message
6. If any fail with retryable error, requeue entire message
7. If all fail with non-retryable errors, send to DLQ

### Retryable vs Non-Retryable Errors

Retryable:
- Network timeouts
- Twilio 5xx errors
- Rate limiting (429)

Non-retryable:
- Invalid phone number format
- Twilio authentication errors
- Invalid message content

## Email Handler Implementation

### SMTP Provider Configuration

| Environment Variable | Description | Required |
|---------------------|-------------|----------|
| SMTP_HOST | SMTP server hostname | Yes |
| SMTP_PORT | SMTP server port | Yes |
| SMTP_USERNAME | SMTP authentication username | No |
| SMTP_PASSWORD | SMTP authentication password | No |
| SMTP_USE_TLS | Enable TLS connection | No (default: true) |
| SMTP_FROM_ADDRESS | Default sender email address | Yes |

### Template System

Templates are stored as HTML files in the `templates/email/` directory. The template engine uses Go's `html/template` package.

Template file naming convention: `{template_name}.html`

Example template (`welcome_email.html`):

```html
<!DOCTYPE html>
<html>
<head>
    <title>Welcome to {{.company_name}}</title>
</head>
<body>
    <h1>Hello {{.user_name}},</h1>
    <p>Welcome to {{.company_name}}!</p>
    <p>Click <a href="{{.activation_link}}">here</a> to activate your account.</p>
</body>
</html>
```

### Processing Logic

1. Validate message payload contains template name and at least one recipient
2. Validate all recipient email addresses
3. Load and parse the email template
4. Render template with provided variables
5. For each recipient, send email via SMTP
6. Track success/failure for each recipient
7. If all succeed, acknowledge message
8. If any fail with retryable error, requeue entire message
9. If all fail with non-retryable errors, send to DLQ

## Configuration

### Application Configuration

Configuration is loaded from environment variables with optional `.env` file support.

```go
type Config struct {
    // Server
    ServerPort int    `env:"SERVER_PORT" default:"8080"`
    LogLevel   string `env:"LOG_LEVEL" default:"info"`
    
    // RabbitMQ
    RabbitMQURL      string `env:"RABBITMQ_URL" required:"true"`
    RabbitMQPrefetch int    `env:"RABBITMQ_PREFETCH" default:"10"`
    
    // Retry
    RetryMaxAttempts       int           `env:"RETRY_MAX_ATTEMPTS" default:"3"`
    RetryInitialDelay      time.Duration `env:"RETRY_INITIAL_DELAY" default:"5s"`
    RetryMaxDelay          time.Duration `env:"RETRY_MAX_DELAY" default:"60s"`
    RetryBackoffMultiplier float64       `env:"RETRY_BACKOFF_MULTIPLIER" default:"2"`
    RetryJitterFactor      float64       `env:"RETRY_JITTER_FACTOR" default:"0.2"`
    
    // Twilio
    TwilioAccountSID string `env:"TWILIO_ACCOUNT_SID"`
    TwilioAuthToken  string `env:"TWILIO_AUTH_TOKEN"`
    TwilioFromNumber string `env:"TWILIO_FROM_NUMBER"`
    
    // SMTP
    SMTPHost        string `env:"SMTP_HOST"`
    SMTPPort        int    `env:"SMTP_PORT" default:"587"`
    SMTPUsername    string `env:"SMTP_USERNAME"`
    SMTPPassword    string `env:"SMTP_PASSWORD"`
    SMTPUseTLS      bool   `env:"SMTP_USE_TLS" default:"true"`
    SMTPFromAddress string `env:"SMTP_FROM_ADDRESS"`
    
    // Templates
    TemplatesPath string `env:"TEMPLATES_PATH" default:"/app/templates"`
}
```

## Observability

### Logging

Use structured logging (slog or zerolog) with the following fields on every log entry:

| Field | Description |
|-------|-------------|
| message_id | Unique message identifier |
| message_type | sms/email |
| correlation_id | Optional correlation ID from metadata |
| retry_count | Current retry attempt number |

### Metrics

Expose Prometheus metrics on `/metrics` endpoint:

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| notifications_processed_total | Counter | type, status | Total notifications processed |
| notifications_retry_total | Counter | type | Total retry attempts |
| notifications_dlq_total | Counter | type | Total messages sent to DLQ |
| notification_processing_duration_seconds | Histogram | type | Processing time per notification |
| provider_request_duration_seconds | Histogram | provider | External API call duration |

### Health Checks

Expose health check endpoints:

| Endpoint | Purpose |
|----------|---------|
| /health | Liveness probe - returns 200 if process is running |
| /ready | Readiness probe - returns 200 if RabbitMQ connection is healthy |

## Docker Configuration

### Dockerfile

```dockerfile
FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /notification-service ./cmd/server

FROM alpine:3.19

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /notification-service .
COPY templates/ ./templates/

USER nobody:nobody

EXPOSE 8080

ENTRYPOINT ["/app/notification-service"]
```

### docker-compose.yml (Development)

```yaml
version: '3.8'

services:
  notification-service:
    build: .
    ports:
      - "8080:8080"
    environment:
      - RABBITMQ_URL=amqp://guest:guest@rabbitmq:5672/
      - TWILIO_ACCOUNT_SID=${TWILIO_ACCOUNT_SID}
      - TWILIO_AUTH_TOKEN=${TWILIO_AUTH_TOKEN}
      - TWILIO_FROM_NUMBER=${TWILIO_FROM_NUMBER}
      - SMTP_HOST=mailhog
      - SMTP_PORT=1025
      - SMTP_USE_TLS=false
      - SMTP_FROM_ADDRESS=noreply@example.com
    depends_on:
      rabbitmq:
        condition: service_healthy

  rabbitmq:
    image: rabbitmq:3.13-management-alpine
    ports:
      - "5672:5672"
      - "15672:15672"
    environment:
      - RABBITMQ_PLUGINS=rabbitmq_delayed_message_exchange
    volumes:
      - ./rabbitmq-enabled-plugins:/etc/rabbitmq/enabled_plugins
    healthcheck:
      test: rabbitmq-diagnostics -q ping
      interval: 10s
      timeout: 5s
      retries: 5

  mailhog:
    image: mailhog/mailhog
    ports:
      - "1025:1025"
      - "8025:8025"
```

Create `rabbitmq-enabled-plugins` file in the project root:

```
[rabbitmq_management,rabbitmq_delayed_message_exchange].
```

## Kubernetes Deployment

### Deployment Manifest

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: notification-service
  labels:
    app: notification-service
spec:
  replicas: 2
  selector:
    matchLabels:
      app: notification-service
  template:
    metadata:
      labels:
        app: notification-service
    spec:
      containers:
        - name: notification-service
          image: notification-service:latest
          ports:
            - containerPort: 8080
          envFrom:
            - configMapRef:
                name: notification-service-config
            - secretRef:
                name: notification-service-secrets
          resources:
            requests:
              cpu: 100m
              memory: 128Mi
            limits:
              cpu: 500m
              memory: 256Mi
          livenessProbe:
            httpGet:
              path: /health
              port: 8080
            initialDelaySeconds: 10
            periodSeconds: 10
          readinessProbe:
            httpGet:
              path: /ready
              port: 8080
            initialDelaySeconds: 5
            periodSeconds: 5
```

### ConfigMap

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: notification-service-config
data:
  SERVER_PORT: "8080"
  LOG_LEVEL: "info"
  RABBITMQ_PREFETCH: "10"
  RETRY_MAX_ATTEMPTS: "3"
  RETRY_INITIAL_DELAY: "5s"
  RETRY_MAX_DELAY: "60s"
  RETRY_JITTER_FACTOR: "0.2"
  SMTP_PORT: "587"
  SMTP_USE_TLS: "true"
  TEMPLATES_PATH: "/app/templates"
```

### Secret (template)

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: notification-service-secrets
type: Opaque
stringData:
  RABBITMQ_URL: "amqp://user:password@rabbitmq:5672/"
  TWILIO_ACCOUNT_SID: ""
  TWILIO_AUTH_TOKEN: ""
  TWILIO_FROM_NUMBER: ""
  SMTP_HOST: ""
  SMTP_USERNAME: ""
  SMTP_PASSWORD: ""
  SMTP_FROM_ADDRESS: ""
```

## Adding New Notification Types

To add a new notification type (e.g., push notifications):

1. Define the message payload structure in the message schema
2. Create a new provider implementing `NotificationProvider` interface
3. Create a new handler implementing `NotificationHandler` interface
4. Register the handler in the application's handler registry
5. Create the corresponding RabbitMQ queues:
   - Bind a new queue to `notifications.delay` exchange with appropriate routing key
   - Create a DLQ bound to `notifications.dlx` exchange
6. Add configuration for the new provider

Example for adding push notifications:

```go
// internal/provider/firebase.go
type FirebaseProvider struct {
    client *messaging.Client
}

func (p *FirebaseProvider) Send(ctx context.Context, req SendRequest) SendResult {
    // Implementation
}

// internal/handler/push.go
type PushHandler struct {
    provider provider.NotificationProvider
}

func (h *PushHandler) Type() string {
    return "push"
}

func (h *PushHandler) Handle(ctx context.Context, msg *Message) Result {
    // Implementation
}
```

## Testing Requirements

### Unit Tests

- All handlers must have unit tests with mocked providers
- All providers must have unit tests with mocked external APIs
- Retry logic must be tested with various failure scenarios

### Integration Tests

- Test RabbitMQ message consumption and acknowledgement
- Test delayed retry flow with actual delays (use short delays in tests)
- Test DLQ routing after max retries exceeded
- Verify delayed message exchange is correctly configured

### Test Coverage

Target minimum 80% code coverage for:
- `internal/handler/`
- `internal/provider/`
- `internal/queue/`

## Dependencies

| Package | Purpose |
|---------|---------|
| github.com/rabbitmq/amqp091-go | RabbitMQ client |
| github.com/twilio/twilio-go | Twilio SDK |
| github.com/caarlos0/env/v10 | Environment configuration |
| github.com/prometheus/client_golang | Metrics |
| github.com/google/uuid | UUID generation |

## Graceful Shutdown

The application must handle SIGTERM and SIGINT signals gracefully:

1. Stop accepting new messages from queues
2. Wait for in-flight messages to complete (with timeout)
3. Close RabbitMQ connections
4. Close provider connections
5. Exit cleanly

Shutdown timeout: 30 seconds (configurable via `SHUTDOWN_TIMEOUT` environment variable)

## CI/CD Pipeline

### Overview

The project uses GitHub Actions for continuous integration and release automation.

### Pull Request Workflow

Every pull request triggers a build and test workflow:

```yaml
# .github/workflows/ci.yml
name: CI

on:
  pull_request:
    branches: [main]

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.23'

      - name: Download dependencies
        run: go mod download

      - name: Build
        run: go build -v ./...

      - name: Test
        run: go test -v -race -coverprofile=coverage.out ./...

      - name: Upload coverage
        uses: codecov/codecov-action@v4
        with:
          files: ./coverage.out
          fail_ci_if_error: false

  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.23'

      - name: golangci-lint
        uses: golangci/golangci-lint-action@v6
        with:
          version: latest
```

### Release Workflow

Releases are triggered by pushing semantic version tags matching `vA.B.C` (e.g., `v1.0.0`, `v2.1.3`):

```yaml
# .github/workflows/release.yml
name: Release

on:
  push:
    tags:
      - 'v[0-9]+.[0-9]+.[0-9]+'

env:
  REGISTRY: ghcr.io
  IMAGE_NAME: ${{ github.repository }}

jobs:
  build-and-release:
    runs-on: ubuntu-latest
    permissions:
      contents: write
      packages: write

    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.23'

      - name: Run tests
        run: go test -v -race ./...

      - name: Set up QEMU
        uses: docker/setup-qemu-action@v3

      - name: Set up Docker Buildx
        uses: docker/setup-buildx-action@v3

      - name: Log in to Container Registry
        uses: docker/login-action@v3
        with:
          registry: ${{ env.REGISTRY }}
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}

      - name: Extract metadata for Docker
        id: meta
        uses: docker/metadata-action@v5
        with:
          images: ${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}
          tags: |
            type=semver,pattern={{version}}
            type=semver,pattern={{major}}.{{minor}}
            type=semver,pattern={{major}}

      - name: Build and push Docker image
        uses: docker/build-push-action@v6
        with:
          context: .
          platforms: linux/amd64,linux/arm64
          push: true
          tags: ${{ steps.meta.outputs.tags }}
          labels: ${{ steps.meta.outputs.labels }}
          cache-from: type=gha
          cache-to: type=gha,mode=max

      - name: Create GitHub Release
        uses: softprops/action-gh-release@v2
        with:
          generate_release_notes: true
          draft: false
          prerelease: false
```

### Release Process

To create a new release:

```bash
# Ensure you're on main with latest changes
git checkout main
git pull origin main

# Create and push a semantic version tag
git tag v1.0.0
git push origin v1.0.0
```

The release workflow will:
1. Run all tests
2. Build multi-architecture Docker images (amd64, arm64)
3. Push images to GitHub Container Registry (ghcr.io)
4. Create a GitHub Release with auto-generated release notes

### Docker Image Tags

For a release tagged `v1.2.3`, the following Docker image tags are created:
- `ghcr.io/<owner>/notification-server:1.2.3`
- `ghcr.io/<owner>/notification-server:1.2`
- `ghcr.io/<owner>/notification-server:1`
