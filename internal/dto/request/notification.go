package request

import (
	"insider-one-assessment/internal/model"
	"time"
)

type CreateNotificationsRequest struct {
	IdempotencyKey string             `json:"-"`
	Notifications  []NotificationItem `json:"notifications" validate:"required,min=1,max=1000,dive"`
}

type CreateNotificationRequest struct {
	IdempotencyKey string           `json:"-"`
	Notification   NotificationItem `json:"notification" validate:"required"`
}

type NotificationItem struct {
	Recipient   string                     `json:"recipient" validate:"required" example:"+905551234567"`
	Channel     model.NotificationChannel  `json:"channel" validate:"required,oneof=sms email push" example:"sms"`
	Content     string                     `json:"content" example:"Hello {{name}}, your code is {{code}}"`
	Priority    model.NotificationPriority `json:"priority" validate:"omitempty,oneof=high normal low" example:"high"`
	TemplateID  *string                    `json:"template_id,omitempty" example:"995a9705-226b-4d8b-bc42-5b77a08ac7bd"`
	Variables   map[string]string          `json:"variables,omitempty" swaggertype:"object" example:"{\"name\":\"Baris\",\"code\":\"123456\"}"`
	ScheduledAt *time.Time                 `json:"scheduled_at,omitempty" swaggertype:"string" example:"2026-05-14T22:22:00+03:00"`
}

type ListNotificationsRequest struct {
	Status   string `validate:"omitempty,oneof=pending queued processing sent failed cancelled"`
	Channel  string `validate:"omitempty,oneof=sms email push"`
	DateFrom string
	DateTo   string
	Page     int `validate:"omitempty,min=1"`
	PageSize int `validate:"omitempty,min=1,max=100"`
}
