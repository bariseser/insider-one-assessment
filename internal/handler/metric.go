package handler

import (
	"insider-one-assessment/internal/repository"
	"insider-one-assessment/internal/service"
	"insider-one-assessment/internal/utils"
	"net/http"
)

type MetricSnapshot = repository.MetricSnapshot

type IMetricHandler interface {
	Handle(w http.ResponseWriter, r *http.Request)
}

type metricHandler struct {
	service service.IMetricService
}

func NewMetricHandler(service service.IMetricService) IMetricHandler {
	return &metricHandler{service: service}
}

// Handle godoc
// @Summary Metrics snapshot
// @Description Returns a JSON metrics snapshot for queue depth, delivery outcomes, and latency.
// @Tags metrics
// @Produce json
// @Success 200 {object} MetricSnapshot
// @Failure 500 {object} utils.HTTPErrorResponse
// @Router /metrics [get]
func (h *metricHandler) Handle(w http.ResponseWriter, r *http.Request) {
	snapshot, err := h.service.GetSnapshot(r.Context())
	if err != nil {
		utils.WriteError(w, err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, snapshot)
}
