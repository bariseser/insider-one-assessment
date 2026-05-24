package router

import (
	"net/http"

	"insider-one-assessment/internal/handler"
)

type Router struct {
	notificationHandler handler.INotificationHandler
	templateHandler     handler.ITemplateHandler
	metricsHandler      handler.IMetricHandler
}

func New(notificationHandler handler.INotificationHandler, templateHandler handler.ITemplateHandler, metricsHandler handler.IMetricHandler) *Router {
	return &Router{
		notificationHandler: notificationHandler,
		templateHandler:     templateHandler,
		metricsHandler:      metricsHandler,
	}
}

func (r *Router) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", r.notificationHandler.HandleHealth)
	mux.HandleFunc("GET /metrics", r.metricsHandler.Handle)
	mux.HandleFunc("POST /templates", r.templateHandler.HandleCreateTemplate)
	mux.HandleFunc("GET /templates", r.templateHandler.HandleListTemplates)
	mux.HandleFunc("GET /templates/{id}", r.templateHandler.HandleGetTemplate)
	mux.HandleFunc("PATCH /templates/{id}", r.templateHandler.HandleUpdateTemplate)
	mux.HandleFunc("GET /batches/{id}", r.notificationHandler.HandleGetBatch)
	mux.HandleFunc("POST /batches", r.notificationHandler.HandleCreateNotificationBatch)
	mux.HandleFunc("GET /notifications", r.notificationHandler.HandleListNotifications)
	mux.HandleFunc("GET /notifications/{id}", r.notificationHandler.HandleGetNotification)
	mux.HandleFunc("POST /notifications", r.notificationHandler.HandleCreateNotification)
	mux.HandleFunc("POST /notifications/{id}/cancel", r.notificationHandler.HandleCancelNotification)

	return mux
}
