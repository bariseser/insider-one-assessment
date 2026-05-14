package request

import "insider-one-assessment/internal/model"

type CreateTemplateRequest struct {
	Name    string                    `json:"name" validate:"required"`
	Channel model.NotificationChannel `json:"channel" validate:"required,oneof=sms email push"`
	Content string                    `json:"content" validate:"required"`
}

type UpdateTemplateRequest struct {
	Name    *string                    `json:"name,omitempty"`
	Channel *model.NotificationChannel `json:"channel,omitempty" validate:"omitempty,oneof=sms email push"`
	Content *string                    `json:"content,omitempty"`
}
