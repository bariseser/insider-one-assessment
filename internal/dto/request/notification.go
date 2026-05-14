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
	Recipient   string                     `json:"recipient" validate:"required"`
	Channel     model.NotificationChannel  `json:"channel" validate:"required,oneof=sms email push"`
	Content     string                     `json:"content"`
	Priority    model.NotificationPriority `json:"priority" validate:"omitempty,oneof=high normal low"`
	TemplateID  *string                    `json:"template_id,omitempty"`
	Variables   map[string]string          `json:"variables,omitempty"`
	ScheduledAt *time.Time                 `json:"scheduled_at,omitempty"`
}

type ListNotificationsRequest struct {
	Status   string `validate:"omitempty,oneof=pending queued processing sent failed cancelled"`
	Channel  string `validate:"omitempty,oneof=sms email push"`
	DateFrom string
	DateTo   string
	Page     int `validate:"omitempty,min=1"`
	PageSize int `validate:"omitempty,min=1,max=100"`
}
