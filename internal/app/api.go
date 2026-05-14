package app

import (
	"net/http"
	"time"

	"insider-one-assessment/internal/config"
	"insider-one-assessment/internal/handler"
	"insider-one-assessment/internal/logging"
	"insider-one-assessment/internal/middleware"
	"insider-one-assessment/internal/repository"
	"insider-one-assessment/internal/router"
	"insider-one-assessment/internal/service"
	postgresdb "insider-one-assessment/pkgs/postgres"

	httpSwagger "github.com/swaggo/http-swagger/v2"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func NewAPIServer(cfg config.Config, dbClient postgresdb.IPostgresInstance) *http.Server {
	logger := logging.New(cfg.LogLevel)

	notificationRepo := repository.NewNotificationRepository(dbClient)
	templateRepo := repository.NewTemplateRepository(dbClient)
	metricsRepo := repository.NewMetricRepository(dbClient)

	notificationService := service.NewNotificationService(notificationRepo, notificationRepo, templateRepo)
	templateService := service.NewTemplateService(templateRepo)
	metricsService := service.NewMetricService(metricsRepo)

	notificationHandler := handler.NewNotificationHandler(notificationService, logger)
	templateHandler := handler.NewTemplateHandler(templateService, logger)
	metricsHandler := handler.NewMetricHandler(metricsService)

	rt := router.New(notificationHandler, templateHandler, metricsHandler)

	rootMux := http.NewServeMux()
	rootMux.Handle("/swagger/", httpSwagger.WrapHandler)
	rootMux.Handle("/", rt.Handler())

	httpHandler := middleware.Chain(
		otelhttp.NewHandler(rootMux, "http.server"),
		middleware.RequestID,
		middleware.Recover(logger),
		middleware.Logger(logger),
		middleware.Timeout(15*time.Second),
	)

	return &http.Server{
		Addr:              cfg.HTTPPort,
		Handler:           httpHandler,
		ReadHeaderTimeout: 5 * time.Second,
	}
}
