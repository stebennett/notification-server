package queue

import (
	"context"
	"log/slog"
	"testing"
	"time"
)

func TestNewConnection(t *testing.T) {
	url := "amqp://guest:guest@localhost:5672/"

	t.Run("creates connection with provided logger", func(t *testing.T) {
		logger := slog.Default()
		conn := NewConnection(url, logger)

		if conn == nil {
			t.Fatal("expected connection to be created")
		}
		if conn.url != url {
			t.Errorf("expected url %q, got %q", url, conn.url)
		}
		if conn.logger != logger {
			t.Error("expected logger to be set")
		}
	})

	t.Run("creates connection with default logger when nil", func(t *testing.T) {
		conn := NewConnection(url, nil)

		if conn == nil {
			t.Fatal("expected connection to be created")
		}
		if conn.logger == nil {
			t.Error("expected default logger to be set")
		}
	})

	t.Run("initializes with correct default values", func(t *testing.T) {
		conn := NewConnection(url, nil)

		if conn.reconnectDelay != 1*time.Second {
			t.Errorf("expected reconnectDelay 1s, got %v", conn.reconnectDelay)
		}
		if conn.maxReconnectDelay != 30*time.Second {
			t.Errorf("expected maxReconnectDelay 30s, got %v", conn.maxReconnectDelay)
		}
		if conn.closed {
			t.Error("expected closed to be false")
		}
	})
}

func TestConnection_IsConnected(t *testing.T) {
	t.Run("returns false when not connected", func(t *testing.T) {
		conn := NewConnection("amqp://localhost:5672/", nil)

		if conn.IsConnected() {
			t.Error("expected IsConnected to return false when not connected")
		}
	})

	t.Run("returns false after close", func(t *testing.T) {
		conn := NewConnection("amqp://localhost:5672/", nil)
		conn.Close()

		if conn.IsConnected() {
			t.Error("expected IsConnected to return false after close")
		}
	})
}

func TestConnection_Close(t *testing.T) {
	t.Run("closes cleanly when not connected", func(t *testing.T) {
		conn := NewConnection("amqp://localhost:5672/", nil)

		err := conn.Close()
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}

		if !conn.closed {
			t.Error("expected closed to be true")
		}
	})

	t.Run("is idempotent", func(t *testing.T) {
		conn := NewConnection("amqp://localhost:5672/", nil)

		err1 := conn.Close()
		err2 := conn.Close()

		if err1 != nil {
			t.Errorf("expected no error on first close, got %v", err1)
		}
		if err2 != nil {
			t.Errorf("expected no error on second close, got %v", err2)
		}
	})
}

func TestConnection_Channel(t *testing.T) {
	t.Run("returns error when not connected", func(t *testing.T) {
		conn := NewConnection("amqp://localhost:5672/", nil)

		_, err := conn.Channel()
		if err != ErrNotConnected {
			t.Errorf("expected ErrNotConnected, got %v", err)
		}
	})

	t.Run("returns error when closed", func(t *testing.T) {
		conn := NewConnection("amqp://localhost:5672/", nil)
		conn.Close()

		_, err := conn.Channel()
		if err != ErrClosed {
			t.Errorf("expected ErrClosed, got %v", err)
		}
	})
}

func TestConnection_Connect(t *testing.T) {
	t.Run("returns error when closed", func(t *testing.T) {
		conn := NewConnection("amqp://localhost:5672/", nil)
		conn.Close()

		err := conn.Connect(context.Background())
		if err != ErrClosed {
			t.Errorf("expected ErrClosed, got %v", err)
		}
	})

	t.Run("returns error for invalid URL", func(t *testing.T) {
		conn := NewConnection("amqp://invalid:5672/", nil)

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		err := conn.Connect(ctx)
		if err == nil {
			t.Error("expected error for invalid connection")
			conn.Close()
		}
	})
}
