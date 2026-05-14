package handler

import (
	"encoding/json"
	"errors"
	"insider-one-assessment/internal/constants"
	"insider-one-assessment/internal/dto/resource"
	"insider-one-assessment/internal/utils"
	"insider-one-assessment/pkgs/validation"
	"log/slog"
	"net/http"
	"strconv"

	"insider-one-assessment/internal/dto/request"
	_ "insider-one-assessment/internal/dto/resource"
	"insider-one-assessment/internal/middleware"
	"insider-one-assessment/internal/repository"
	"insider-one-assessment/internal/service"
)

type INotificationHandler interface {
	HandleHealth(w http.ResponseWriter, r *http.Request)
	HandleCreateNotification(w http.ResponseWriter, r *http.Request)
	HandleCreateNotificationBatch(w http.ResponseWriter, r *http.Request)
	HandleGetNotification(w http.ResponseWriter, r *http.Request)
	HandleGetBatch(w http.ResponseWriter, r *http.Request)
	HandleListNotifications(w http.ResponseWriter, r *http.Request)
	HandleCancelNotification(w http.ResponseWriter, r *http.Request)
}

type notificationHandler struct {
	notificationService service.INotificationService
	logger              *slog.Logger
}

func NewNotificationHandler(notificationService service.INotificationService, logger *slog.Logger) INotificationHandler {
	return &notificationHandler{
		notificationService: notificationService,
		logger:              logger,
	}
}

// HandleHealth godoc
// @Summary Health check
// @Description Returns API health status.
// @Tags health
// @Produce json
// @Success 200 {object} map[string]string
// @Router /health [get]
func (h *notificationHandler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	utils.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// HandleCreateNotification godoc
// @Summary Create notification
// @Description Creates a single notification and stores a matching outbox event for async dispatch.
// @Tags notifications
// @Accept json
// @Produce json
// @Param Idempotency-Key header string false "Optional idempotency key for safe retries"
// @Param request body request.CreateNotificationRequest true "Create notification request"
// @Success 202 {object} resource.NotificationResource
// @Failure 400 {object} utils.ErrorResponse
// @Failure 409 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /notifications [post]
func (h *notificationHandler) HandleCreateNotification(w http.ResponseWriter, r *http.Request) {
	var req request.CreateNotificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteError(w, utils.NewErrorBag(constants.BodyParserErrCode, err, constants.BodyParserMsg))
		return
	}
	req.IdempotencyKey = r.Header.Get("Idempotency-Key")
	if err := validation.ValidateCreateNotificationRequest(req); err != nil {
		utils.WriteError(w, utils.NewErrorBag(constants.ValidationErrCode, err, err.Error()))
		return
	}

	resp, err := h.notificationService.CreateNotification(r.Context(), req)
	if err != nil {
		logServiceError(h.logger, r, err)
		utils.WriteError(w, err)
		return
	}

	utils.WriteSuccess(w, http.StatusAccepted, resource.NotificationResponse{Notification: *resp})
}

// HandleCreateNotificationBatch godoc
// @Summary Create notification batch
// @Description Creates one or more notifications in a batch and stores matching outbox events for async dispatch.
// @Tags batches
// @Accept json
// @Produce json
// @Param Idempotency-Key header string false "Optional idempotency key for safe retries"
// @Param request body request.CreateNotificationsRequest true "Create notification batch request"
// @Success 202 {object} resource.CreateNotificationsResponse
// @Failure 400 {object} utils.ErrorResponse
// @Failure 409 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /batches [post]
func (h *notificationHandler) HandleCreateNotificationBatch(w http.ResponseWriter, r *http.Request) {
	var req request.CreateNotificationsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteError(w, utils.NewErrorBag(constants.BodyParserErrCode, err, constants.BodyParserMsg))
		return
	}
	req.IdempotencyKey = r.Header.Get("Idempotency-Key")
	if err := validation.ValidateCreateNotificationsRequest(req); err != nil {
		utils.WriteError(w, utils.NewErrorBag(constants.ValidationErrCode, err, err.Error()))
		return
	}

	resp, err := h.notificationService.CreateBatchNotification(r.Context(), req)
	if err != nil {
		logServiceError(h.logger, r, err)
		utils.WriteError(w, err)
		return
	}

	utils.WriteSuccess(w, http.StatusAccepted, resource.BatchNotificationsCreateResponse{Batch: *resp})
}

// HandleGetNotification godoc
// @Summary Get notification
// @Description Returns a single notification by ID.
// @Tags notifications
// @Produce json
// @Param id path string true "Notification ID"
// @Success 200 {object} resource.NotificationResource
// @Failure 400 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /notifications/{id} [get]
func (h *notificationHandler) HandleGetNotification(w http.ResponseWriter, r *http.Request) {
	notificationID := r.PathValue("id")
	if err := validation.ValidateUUID("notification id", notificationID); err != nil {
		utils.WriteError(w, utils.NewErrorBag(constants.ValidationErrCode, err, err.Error()))
		return
	}

	resp, err := h.notificationService.GetNotificationByID(r.Context(), notificationID)
	if err != nil {
		logServiceError(h.logger, r, err)
		utils.WriteError(w, err)
		return
	}

	utils.WriteSuccess(w, http.StatusOK, resource.NotificationResponse{Notification: *resp})
}

// HandleGetBatch godoc
// @Summary Get notification batch
// @Description Returns a notification batch with its notifications.
// @Tags batches
// @Produce json
// @Param id path string true "Batch ID"
// @Success 200 {object} resource.NotificationBatchResource
// @Failure 400 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /batches/{id} [get]
func (h *notificationHandler) HandleGetBatch(w http.ResponseWriter, r *http.Request) {
	batchID := r.PathValue("id")
	if err := validation.ValidateUUID("batch id", batchID); err != nil {
		utils.WriteError(w, utils.NewErrorBag(constants.ValidationErrCode, err, err.Error()))
		return
	}

	resp, err := h.notificationService.GetBatchByID(r.Context(), batchID)
	if err != nil {
		logServiceError(h.logger, r, err)
		utils.WriteError(w, err)
		return
	}

	utils.WriteSuccess(w, http.StatusOK, resource.BatchNotificationsResponse{Batch: *resp})
}

// HandleListNotifications godoc
// @Summary List notifications
// @Description Lists notifications with filtering and pagination.
// @Tags notifications
// @Produce json
// @Param status query string false "Notification status"
// @Param channel query string false "Notification channel"
// @Param date_from query string false "Created at lower bound (RFC3339)"
// @Param date_to query string false "Created at upper bound (RFC3339)"
// @Param page query int false "Page number"
// @Param page_size query int false "Page size"
// @Success 200 {object} resource.ListNotificationsResponse
// @Failure 400 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /notifications [get]
func (h *notificationHandler) HandleListNotifications(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))

	req := request.ListNotificationsRequest{
		Status:   r.URL.Query().Get("status"),
		Channel:  r.URL.Query().Get("channel"),
		DateFrom: r.URL.Query().Get("date_from"),
		DateTo:   r.URL.Query().Get("date_to"),
		Page:     page,
		PageSize: pageSize,
	}
	if err := validation.ValidateListNotificationsRequest(req); err != nil {
		utils.WriteError(w, utils.NewErrorBag(constants.ValidationErrCode, err, err.Error()))
		return
	}

	resp, err := h.notificationService.ListNotifications(r.Context(), req)
	if err != nil {
		logServiceError(h.logger, r, err)
		utils.WriteError(w, err)
		return
	}

	utils.WriteSuccess(w, http.StatusOK, resp)
}

// HandleCancelNotification godoc
// @Summary Cancel notification
// @Description Cancels a pending or queued notification.
// @Tags notifications
// @Produce json
// @Param id path string true "Notification ID"
// @Success 200 {object} resource.NotificationResource
// @Failure 400 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 409 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /notifications/{id}/cancel [post]
func (h *notificationHandler) HandleCancelNotification(w http.ResponseWriter, r *http.Request) {
	notificationID := r.PathValue("id")
	if err := validation.ValidateUUID("notification id", notificationID); err != nil {
		utils.WriteError(w, utils.NewErrorBag(constants.ValidationErrCode, err, err.Error()))
		return
	}

	resp, err := h.notificationService.CancelNotification(r.Context(), notificationID)
	if err != nil {
		logServiceError(h.logger, r, err)
		utils.WriteError(w, err)
		return
	}

	utils.WriteSuccess(w, http.StatusOK, resource.NotificationResponse{Notification: *resp})
}

func logServiceError(logger *slog.Logger, r *http.Request, err error) {
	statusCode := http.StatusInternalServerError
	switch {
	case errors.Is(err, service.ErrTooManyNotifications),
		errors.Is(err, service.ErrUnsupportedChannel),
		errors.Is(err, service.ErrInvalidPriority),
		errors.Is(err, service.ErrEmptyRecipient),
		errors.Is(err, service.ErrEmptyContent),
		errors.Is(err, service.ErrEmptyTemplateName),
		errors.Is(err, service.ErrEmptyTemplateContent),
		errors.Is(err, service.ErrEmptyTemplateUpdate),
		errors.Is(err, service.ErrTemplateChannelMismatch),
		errors.Is(err, service.ErrInvalidNotificationContent),
		errors.Is(err, service.ErrContentTooLongSMS),
		errors.Is(err, service.ErrInvalidRequest),
		errors.Is(err, service.ErrInvalidPagination),
		errors.Is(err, service.ErrInvalidDateRange):
		statusCode = http.StatusBadRequest
	case errors.Is(err, repository.ErrIdempotencyConflict),
		errors.Is(err, repository.ErrTemplateAlreadyExists),
		errors.Is(err, service.ErrNotificationNotCancellable):
		statusCode = http.StatusConflict
	case errors.Is(err, service.ErrNotificationNotFound),
		errors.Is(err, service.ErrTemplateNotFound),
		errors.Is(err, service.ErrBatchNotFound):
		statusCode = http.StatusNotFound
	}

	if logger == nil {
		return
	}

	args := []any{
		"request_id", middleware.GetRequestID(r.Context()),
		"method", r.Method,
		"path", r.URL.Path,
		"status_code", statusCode,
		"error", err.Error(),
	}

	if statusCode >= http.StatusInternalServerError {
		logger.Error("request failed", args...)
		return
	}

	logger.Warn("request failed", args...)
}
