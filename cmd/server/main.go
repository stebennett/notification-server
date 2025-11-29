package main

import (
	"log"
	"os"

	"github.com/stebennett/notification-server/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load configuration: %v", err)
	}

	log.Printf("configuration loaded successfully")
	log.Printf("server port: %d", cfg.ServerPort)
	log.Printf("log level: %s", cfg.LogLevel)
	log.Printf("rabbitmq prefetch: %d", cfg.RabbitMQPrefetch)

	os.Exit(0)
}
