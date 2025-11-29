package config

import (
	"time"

	"github.com/caarlos0/env/v10"
)

type Config struct {
	// Server
	ServerPort int    `env:"SERVER_PORT" envDefault:"8080"`
	LogLevel   string `env:"LOG_LEVEL" envDefault:"info"`

	// RabbitMQ
	RabbitMQURL      string `env:"RABBITMQ_URL,required"`
	RabbitMQPrefetch int    `env:"RABBITMQ_PREFETCH" envDefault:"10"`

	// Retry
	RetryMaxAttempts       int           `env:"RETRY_MAX_ATTEMPTS" envDefault:"3"`
	RetryInitialDelay      time.Duration `env:"RETRY_INITIAL_DELAY" envDefault:"5s"`
	RetryMaxDelay          time.Duration `env:"RETRY_MAX_DELAY" envDefault:"60s"`
	RetryBackoffMultiplier float64       `env:"RETRY_BACKOFF_MULTIPLIER" envDefault:"2"`
	RetryJitterFactor      float64       `env:"RETRY_JITTER_FACTOR" envDefault:"0.2"`

	// Twilio
	TwilioAccountSID string `env:"TWILIO_ACCOUNT_SID"`
	TwilioAuthToken  string `env:"TWILIO_AUTH_TOKEN"`
	TwilioFromNumber string `env:"TWILIO_FROM_NUMBER"`

	// SMTP
	SMTPHost        string `env:"SMTP_HOST"`
	SMTPPort        int    `env:"SMTP_PORT" envDefault:"587"`
	SMTPUsername    string `env:"SMTP_USERNAME"`
	SMTPPassword    string `env:"SMTP_PASSWORD"`
	SMTPUseTLS      bool   `env:"SMTP_USE_TLS" envDefault:"true"`
	SMTPFromAddress string `env:"SMTP_FROM_ADDRESS"`

	// Templates
	TemplatesPath string `env:"TEMPLATES_PATH" envDefault:"/app/templates"`

	// Shutdown
	ShutdownTimeout time.Duration `env:"SHUTDOWN_TIMEOUT" envDefault:"30s"`
}

func Load() (*Config, error) {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
