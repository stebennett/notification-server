# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Install ca-certificates for HTTPS calls
RUN apk add --no-cache ca-certificates

# Copy go mod files first for better caching
COPY go.mod go.sum* ./
RUN go mod download

# Copy source code
COPY . .

# Build static binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o notification-service ./cmd/server

# Final stage
FROM scratch

# Copy CA certificates for HTTPS
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy binary
COPY --from=builder /app/notification-service /notification-service

# Copy templates directory (will be populated later)
# COPY --from=builder /app/templates /templates

# Run as non-root user
USER 65534:65534

EXPOSE 8080

ENTRYPOINT ["/notification-service"]
