//go:build integration

package queue

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestConnection_Integration(t *testing.T) {
	url := os.Getenv("RABBITMQ_URL")
	if url == "" {
		t.Skip("RABBITMQ_URL not set, skipping integration test")
	}

	t.Run("connects successfully to RabbitMQ", func(t *testing.T) {
		conn := NewConnection(url, nil)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err := conn.Connect(ctx)
		if err != nil {
			t.Fatalf("failed to connect: %v", err)
		}
		defer conn.Close()

		if !conn.IsConnected() {
			t.Error("expected IsConnected to return true after successful connection")
		}
	})

	t.Run("can create a channel", func(t *testing.T) {
		conn := NewConnection(url, nil)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err := conn.Connect(ctx)
		if err != nil {
			t.Fatalf("failed to connect: %v", err)
		}
		defer conn.Close()

		ch, err := conn.Channel()
		if err != nil {
			t.Fatalf("failed to create channel: %v", err)
		}
		defer ch.Close()

		// Verify we can declare a test queue
		_, err = ch.QueueDeclare(
			"test-queue",
			false, // durable
			true,  // auto-delete
			false, // exclusive
			false, // no-wait
			nil,   // args
		)
		if err != nil {
			t.Fatalf("failed to declare queue: %v", err)
		}
	})
}
