package config

import (
	"os"
	"strconv"
)

type Config struct {
	HTTPPort             string
	DatabaseURL          string
	AMQPURL              string
	ProviderURL          string
	ProviderTimeout      string
	ProviderMockResponse string
	ChannelRateLimit     int
	LogLevel             string
	OTelExporterEndpoint string
}

func Load() Config {
	return Config{
		HTTPPort:             getEnv("HTTP_PORT", ""),
		DatabaseURL:          getEnv("DATABASE_URL", ""),
		AMQPURL:              getEnv("AMQP_URL", ""),
		ProviderURL:          getEnv("PROVIDER_URL", ""),
		ProviderTimeout:      getEnv("PROVIDER_TIMEOUT", ""),
		ProviderMockResponse: getEnv("PROVIDER_MOCK_RESPONSE", ""),
		ChannelRateLimit:     getEnvAsInt("CHANNEL_RATE_LIMIT", 100),
		LogLevel:             getEnv("LOG_LEVEL", "ERROR"),
		OTelExporterEndpoint: getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", ""),
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}

func getEnvAsInt(key string, fallback int) int {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}

	return fallback
}
