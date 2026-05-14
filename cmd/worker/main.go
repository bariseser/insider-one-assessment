package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"
	"time"

	"insider-one-assessment/internal/config"
	"insider-one-assessment/internal/logging"
	"insider-one-assessment/internal/queue/rabbitmq"
	"insider-one-assessment/internal/repository"
	"insider-one-assessment/internal/tracing"
	"insider-one-assessment/internal/worker"
	"insider-one-assessment/pkgs/notification_provider"
	postgresdb "insider-one-assessment/pkgs/postgres"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	cfg := config.Load()
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	shutdownTracing, err := tracing.Init(ctx, "notification-worker", cfg.OTelExporterEndpoint)
	if err != nil {
		log.Fatalf("init tracing: %v", err)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := shutdownTracing(shutdownCtx); err != nil {
			log.Printf("shutdown tracing: %v", err)
		}
	}()

	timeout, err := time.ParseDuration(cfg.ProviderTimeout)
	if err != nil {
		log.Fatalf("parse provider timeout: %v", err)
	}

	dbClient, err := postgresdb.NewClient(cfg.DatabaseURL, cfg.LogLevel == "debug", postgresdb.Options{})
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer func() {
		if err := dbClient.Close(); err != nil {
			log.Printf("close db: %v", err)
		}
	}()

	if err := dbClient.Ping(ctx); err != nil {
		log.Fatalf("ping db: %v", err)
	}

	queueClient, err := rabbitmq.NewClient(cfg.AMQPURL)
	if err != nil {
		log.Fatalf("connect rabbitmq: %v", err)
	}
	defer func() {
		if err := queueClient.Close(); err != nil {
			log.Printf("close rabbitmq: %v", err)
		}
	}()

	logger := logging.New(cfg.LogLevel)
	repo := repository.NewNotificationRepository(dbClient)
	providerClient := notification_provider.NewNotificationClient(cfg.ProviderURL, timeout, cfg.ProviderMockResponse)
	w := worker.New(logger, repo, repo, queueClient, providerClient, cfg.ChannelRateLimit)

	if err := w.Run(ctx); err != nil {
		log.Fatalf("run worker: %v", err)
	}
}
