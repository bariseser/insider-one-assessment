package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"insider-one-assessment/internal/constants"
	"insider-one-assessment/internal/dto/request"
	"insider-one-assessment/internal/dto/resource"
	"insider-one-assessment/internal/service"
	"insider-one-assessment/internal/utils"
	"insider-one-assessment/pkgs/validation"
)

type ITemplateHandler interface {
	HandleCreateTemplate(w http.ResponseWriter, r *http.Request)
	HandleGetTemplate(w http.ResponseWriter, r *http.Request)
	HandleListTemplates(w http.ResponseWriter, r *http.Request)
	HandleUpdateTemplate(w http.ResponseWriter, r *http.Request)
}

func NewTemplateHandler(templateService service.ITemplateService, logger *slog.Logger) ITemplateHandler {
	return &templateHandler{
		templateService: templateService,
		logger:          logger,
	}
}

type templateHandler struct {
	templateService service.ITemplateService
	logger          *slog.Logger
}

// HandleCreateTemplate godoc
// @Summary Create template
// @Description Creates a message template that can be referenced by notifications.
// @Tags templates
// @Accept json
// @Produce json
// @Param request body request.CreateTemplateRequest true "Create template request"
// @Success 201 {object} resource.TemplateResponse
// @Failure 400 {object} utils.HTTPErrorResponse
// @Failure 500 {object} utils.HTTPErrorResponse
// @Router /templates [post]
func (h *templateHandler) HandleCreateTemplate(w http.ResponseWriter, r *http.Request) {
	var req request.CreateTemplateRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		appErr := utils.NewErrorBag(constants.BodyParserErrCode, err, constants.BodyParserMsg)
		utils.WriteError(w, appErr)
		return
	}

	if err := validation.ValidateCreateTemplateRequest(req); err != nil {
		appErr := utils.NewErrorBag(constants.ValidationErrCode, err, err.Error())
		utils.WriteError(w, appErr)
		return
	}

	resp, err := h.templateService.CreateTemplate(r.Context(), req)
	if err != nil {
		logTemplateError(h.logger, r, err)
		utils.WriteError(w, err)
		return
	}

	utils.WriteSuccess(w, http.StatusCreated, resource.TemplateResponse{Template: *resp})
}

// HandleGetTemplate godoc
// @Summary Get template
// @Description Returns a template by ID.
// @Tags templates
// @Produce json
// @Param id path string true "Template ID"
// @Success 200 {object} resource.TemplateResponse
// @Failure 400 {object} utils.HTTPErrorResponse
// @Failure 404 {object} utils.HTTPErrorResponse
// @Failure 500 {object} utils.HTTPErrorResponse
// @Router /templates/{id} [get]
func (h *templateHandler) HandleGetTemplate(w http.ResponseWriter, r *http.Request) {
	templateID := r.PathValue("id")
	if err := validation.ValidateUUID("template id", templateID); err != nil {
		utils.WriteError(w, utils.NewErrorBag(constants.ValidationErrCode, err, err.Error()))
		return
	}

	resp, err := h.templateService.GetTemplateByID(r.Context(), templateID)
	if err != nil {
		logTemplateError(h.logger, r, err)
		utils.WriteError(w, err)
		return
	}

	utils.WriteSuccess(w, http.StatusOK, resource.TemplateResponse{Template: *resp})
}

// HandleListTemplates godoc
// @Summary List templates
// @Description Returns all templates.
// @Tags templates
// @Produce json
// @Param channel query string false "Template channel"
// @Success 200 {object} resource.TemplatesResponse
// @Failure 400 {object} utils.HTTPErrorResponse
// @Failure 500 {object} utils.HTTPErrorResponse
// @Router /templates [get]
func (h *templateHandler) HandleListTemplates(w http.ResponseWriter, r *http.Request) {
	resp, err := h.templateService.ListTemplates(r.Context())
	if err != nil {
		logTemplateError(h.logger, r, err)
		utils.WriteError(w, err)
		return
	}
	utils.WriteSuccess(w, http.StatusOK, resource.TemplatesResponse{Templates: resp})
}

// HandleUpdateTemplate godoc
// @Summary Update template
// @Description Updates a template by ID.
// @Tags templates
// @Accept json
// @Produce json
// @Param id path string true "Template ID"
// @Param request body request.UpdateTemplateRequest true "Update template request"
// @Success 200 {object} resource.TemplateResponse
// @Failure 400 {object} utils.HTTPErrorResponse
// @Failure 404 {object} utils.HTTPErrorResponse
// @Failure 500 {object} utils.HTTPErrorResponse
// @Router /templates/{id} [patch]
func (h *templateHandler) HandleUpdateTemplate(w http.ResponseWriter, r *http.Request) {
	templateID := r.PathValue("id")
	if err := validation.ValidateUUID("template id", templateID); err != nil {
		utils.WriteError(w, utils.NewErrorBag(constants.ValidationErrCode, err, err.Error()))
		return
	}

	var req request.UpdateTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteError(w, utils.NewErrorBag(constants.BodyParserErrCode, err, constants.BodyParserMsg))
		return
	}
	if err := validation.ValidateUpdateTemplateRequest(req); err != nil {
		utils.WriteError(w, utils.NewErrorBag(constants.ValidationErrCode, err, err.Error()))
		return
	}

	resp, err := h.templateService.UpdateTemplate(r.Context(), templateID, req)
	if err != nil {
		logTemplateError(h.logger, r, err)
		utils.WriteError(w, err)
		return
	}

	utils.WriteSuccess(w, http.StatusOK, resource.TemplateResponse{Template: *resp})
}

func logTemplateError(logger *slog.Logger, r *http.Request, err error) {
	if logger == nil {
		return
	}

	statusCode := http.StatusInternalServerError
	if errorBag, ok := err.(*utils.ErrorBag); ok {
		statusCode = errorBag.GetStatusCode()
	}

	args := []any{
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
