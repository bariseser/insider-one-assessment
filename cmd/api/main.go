package main

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"insider-one-assessment/internal/app"
	"insider-one-assessment/internal/config"
	"insider-one-assessment/internal/tracing"
	postgresdb "insider-one-assessment/pkgs/postgres"

	_ "github.com/jackc/pgx/v5/stdlib"
	_ "insider-one-assessment/docs"
)

// @title Insider One Assessment Notification API
// @version 1.0
// @description Event-driven notification system assessment API.
// @host 127.0.0.1:18080
// @BasePath /
// @schemes http
func main() {
	cfg := config.Load()
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	shutdownTracing, err := tracing.Init(ctx, "notification-api", cfg.OTelExporterEndpoint)
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

	dbClient, err := postgresdb.NewClient(cfg.DatabaseURL, cfg.LogLevel == "debug", postgresdb.Options{
		MaxOpenConns: 20,
		MaxIdleConns: 5,
	})
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

	server := app.NewAPIServer(cfg, dbClient)

	go func() {
		<-ctx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("shutdown server: %v", err)
		}
	}()

	log.Printf("api listening on %s", cfg.HTTPPort)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("serve http: %v", err)
	}
}
