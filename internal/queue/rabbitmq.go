package queue

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

var (
	ErrNotConnected = errors.New("not connected to RabbitMQ")
	ErrClosed       = errors.New("connection is closed")
)

// Connection manages a RabbitMQ connection with automatic reconnection.
type Connection struct {
	url    string
	conn   *amqp.Connection
	mu     sync.RWMutex
	closed bool
	done   chan struct{}
	logger *slog.Logger

	// Reconnection settings
	reconnectDelay    time.Duration
	maxReconnectDelay time.Duration
}

// NewConnection creates a new Connection instance.
func NewConnection(url string, logger *slog.Logger) *Connection {
	if logger == nil {
		logger = slog.Default()
	}
	return &Connection{
		url:               url,
		done:              make(chan struct{}),
		logger:            logger,
		reconnectDelay:    1 * time.Second,
		maxReconnectDelay: 30 * time.Second,
	}
}

// Connect establishes the initial connection to RabbitMQ.
func (c *Connection) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return ErrClosed
	}

	conn, err := amqp.Dial(c.url)
	if err != nil {
		return err
	}

	c.conn = conn
	c.logger.Info("connected to RabbitMQ")

	go c.handleReconnect(ctx)

	return nil
}

// handleReconnect listens for connection closures and attempts to reconnect.
func (c *Connection) handleReconnect(ctx context.Context) {
	for {
		c.mu.RLock()
		if c.closed {
			c.mu.RUnlock()
			return
		}
		conn := c.conn
		c.mu.RUnlock()

		if conn == nil {
			return
		}

		notifyClose := conn.NotifyClose(make(chan *amqp.Error, 1))

		select {
		case <-c.done:
			return
		case <-ctx.Done():
			return
		case amqpErr, ok := <-notifyClose:
			if !ok {
				// Channel closed without error, connection was closed gracefully
				return
			}

			c.logger.Warn("connection lost", "error", amqpErr)
			c.reconnect(ctx)
		}
	}
}

// reconnect attempts to reconnect with exponential backoff.
func (c *Connection) reconnect(ctx context.Context) {
	delay := c.reconnectDelay

	for {
		c.mu.RLock()
		if c.closed {
			c.mu.RUnlock()
			return
		}
		c.mu.RUnlock()

		select {
		case <-c.done:
			return
		case <-ctx.Done():
			return
		case <-time.After(delay):
		}

		c.logger.Info("attempting to reconnect", "delay", delay)

		conn, err := amqp.Dial(c.url)
		if err != nil {
			c.logger.Warn("reconnection failed", "error", err, "next_delay", delay*2)

			delay *= 2
			if delay > c.maxReconnectDelay {
				delay = c.maxReconnectDelay
			}
			continue
		}

		c.mu.Lock()
		if c.closed {
			conn.Close()
			c.mu.Unlock()
			return
		}
		c.conn = conn
		c.mu.Unlock()

		c.logger.Info("reconnected to RabbitMQ")
		return
	}
}

// Close closes the connection gracefully.
func (c *Connection) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}

	c.closed = true
	close(c.done)

	if c.conn != nil {
		err := c.conn.Close()
		c.conn = nil
		return err
	}

	return nil
}

// IsConnected returns true if the connection is established and healthy.
func (c *Connection) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed || c.conn == nil {
		return false
	}

	return !c.conn.IsClosed()
}

// Channel returns a new channel from the connection.
func (c *Connection) Channel() (*amqp.Channel, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return nil, ErrClosed
	}

	if c.conn == nil || c.conn.IsClosed() {
		return nil, ErrNotConnected
	}

	return c.conn.Channel()
}
