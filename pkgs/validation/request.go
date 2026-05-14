package validation

import (
	"errors"
	"fmt"
	"strings"
	"time"

	requestdto "insider-one-assessment/internal/dto/request"
	"insider-one-assessment/internal/model"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

var validate *validator.Validate

func init() {
	validate = validator.New()
	validate.RegisterStructValidation(notificationCreateItemValidation, requestdto.NotificationItem{})
	validate.RegisterStructValidation(updateTemplateRequestValidation, requestdto.UpdateTemplateRequest{})
	validate.RegisterStructValidation(listNotificationsRequestValidation, requestdto.ListNotificationsRequest{})
}

func ValidateCreateNotificationRequest(req requestdto.CreateNotificationRequest) error {
	return validateStruct(req)
}

func ValidateCreateNotificationsRequest(req requestdto.CreateNotificationsRequest) error {
	return validateStruct(req)
}

func ValidateCreateTemplateRequest(req requestdto.CreateTemplateRequest) error {
	return validateStruct(req)
}

func ValidateUpdateTemplateRequest(req requestdto.UpdateTemplateRequest) error {
	return validateStruct(req)
}

func ValidateListNotificationsRequest(req requestdto.ListNotificationsRequest) error {
	return validateStruct(req)
}

func ValidateUUID(fieldName, value string) error {
	if _, err := uuid.Parse(value); err != nil {
		return fmt.Errorf("invalid request: %s must be a valid uuid", fieldName)
	}

	return nil
}

func validateStruct(value any) error {
	if err := validate.Struct(value); err != nil {
		var validationErrs validator.ValidationErrors
		if errors.As(err, &validationErrs) {
			return formatValidationError(validationErrs[0])
		}
		return err
	}

	return nil
}

func formatValidationError(fieldErr validator.FieldError) error {
	switch fieldErr.Tag() {
	case "required":
		return fmt.Errorf("%s is required", toSnakeCase(fieldErr.Field()))
	case "oneof":
		switch fieldErr.Field() {
		case "Channel":
			return errors.New("unsupported channel")
		case "Priority":
			return errors.New("invalid priority")
		}
	case "min":
		if fieldErr.Field() == "Notifications" {
			return errors.New("notifications is required")
		}
		if fieldErr.Field() == "Page" || fieldErr.Field() == "PageSize" {
			return errors.New("invalid pagination")
		}
	case "max":
		if fieldErr.Field() == "Notifications" {
			return errors.New("batch size exceeds 1000")
		}
		if fieldErr.Field() == "PageSize" {
			return errors.New("invalid pagination")
		}
	case "content_or_template":
		return errors.New("either content or template_id must be provided, but not both")
	case "variables_require_template":
		return errors.New("variables can only be provided with template_id")
	case "sms_content_length":
		return errors.New("sms content exceeds 160 characters")
	case "scheduled_at_future":
		return errors.New("scheduled_at must be in the future")
	case "empty_update":
		return errors.New("at least one template field must be provided")
	case "invalid_date_range":
		return errors.New("invalid date range")
	}

	return errors.New("invalid request")
}

func notificationCreateItemValidation(sl validator.StructLevel) {
	item := sl.Current().Interface().(requestdto.NotificationItem)

	if item.TemplateID != nil && item.Content != "" {
		sl.ReportError(item.Content, "Content", "content", "content_or_template", "")
		return
	}

	if item.TemplateID == nil && strings.TrimSpace(item.Content) == "" {
		sl.ReportError(item.Content, "Content", "content", "content_or_template", "")
		return
	}

	if item.TemplateID == nil && len(item.Variables) > 0 {
		sl.ReportError(item.Variables, "Variables", "variables", "variables_require_template", "")
		return
	}

	if item.TemplateID == nil && item.Channel == model.ChannelSMS && len(item.Content) > 160 {
		sl.ReportError(item.Content, "Content", "content", "sms_content_length", "")
	}

	if item.ScheduledAt != nil && !item.ScheduledAt.UTC().After(time.Now().UTC()) {
		sl.ReportError(item.ScheduledAt, "ScheduledAt", "scheduled_at", "scheduled_at_future", "")
	}
}

func updateTemplateRequestValidation(sl validator.StructLevel) {
	req := sl.Current().Interface().(requestdto.UpdateTemplateRequest)

	if req.Name == nil && req.Channel == nil && req.Content == nil {
		sl.ReportError(req, "UpdateTemplateRequest", "update_template_request", "empty_update", "")
		return
	}

	if req.Name != nil && strings.TrimSpace(*req.Name) == "" {
		sl.ReportError(req.Name, "Name", "name", "required", "")
	}

	if req.Content != nil && strings.TrimSpace(*req.Content) == "" {
		sl.ReportError(req.Content, "Content", "content", "required", "")
	}

	if req.Channel != nil {
		switch *req.Channel {
		case model.ChannelSMS, model.ChannelEmail, model.ChannelPush:
		default:
			sl.ReportError(req.Channel, "Channel", "channel", "oneof", "")
		}
	}
}

func listNotificationsRequestValidation(sl validator.StructLevel) {
	req := sl.Current().Interface().(requestdto.ListNotificationsRequest)

	var (
		dateFrom time.Time
		dateTo   time.Time
		err      error
	)

	if req.DateFrom != "" {
		dateFrom, err = time.Parse(time.RFC3339, req.DateFrom)
		if err != nil {
			sl.ReportError(req.DateFrom, "DateFrom", "date_from", "invalid_date_range", "")
			return
		}
	}

	if req.DateTo != "" {
		dateTo, err = time.Parse(time.RFC3339, req.DateTo)
		if err != nil {
			sl.ReportError(req.DateTo, "DateTo", "date_to", "invalid_date_range", "")
			return
		}
	}

	if !dateFrom.IsZero() && !dateTo.IsZero() && dateFrom.After(dateTo) {
		sl.ReportError(req, "ListNotificationsRequest", "list_notifications_request", "invalid_date_range", "")
	}
}

func toSnakeCase(value string) string {
	var builder strings.Builder
	for idx, r := range value {
		if idx > 0 && r >= 'A' && r <= 'Z' {
			builder.WriteByte('_')
		}
		builder.WriteRune(r)
	}

	return strings.ToLower(builder.String())
}
