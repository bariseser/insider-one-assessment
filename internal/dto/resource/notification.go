package resource

import (
	"time"

	"insider-one-assessment/internal/model"

	"github.com/google/uuid"
)

type NotificationResource struct {
	ID                string                     `json:"id"`
	BatchID           *string                    `json:"batch_id,omitempty"`
	TemplateID        *string                    `json:"template_id,omitempty"`
	Recipient         string                     `json:"recipient"`
	Channel           model.NotificationChannel  `json:"channel"`
	Content           string                     `json:"content"`
	Priority          model.NotificationPriority `json:"priority"`
	Status            model.NotificationStatus   `json:"status"`
	AttemptCount      int                        `json:"attempt_count"`
	MaxAttempts       int                        `json:"max_attempts"`
	ScheduledAt       *time.Time                 `json:"scheduled_at,omitempty"`
	NextAttemptAt     *time.Time                 `json:"next_attempt_at,omitempty"`
	ProviderMessageID *string                    `json:"provider_message_id,omitempty"`
	LastError         *string                    `json:"last_error,omitempty"`
	CancelledAt       *time.Time                 `json:"cancelled_at,omitempty"`
	CreatedAt         time.Time                  `json:"created_at"`
	UpdatedAt         time.Time                  `json:"updated_at"`
}

type ListNotificationsResponse struct {
	Page          int                    `json:"page"`
	PageSize      int                    `json:"page_size"`
	Total         int                    `json:"total"`
	Notifications []NotificationResource `json:"notifications"`
}

type NotificationResponse struct {
	Notification NotificationResource `json:"notification"`
}
type NotificationsResponse struct {
	Notification []NotificationResource `json:"notifications"`
}

type BatchNotificationsCreateResponse struct {
	Batch BatchNotificationsCreateResource `json:"batch"`
}

type BatchNotificationsCreateResource struct {
	BatchID         string                   `json:"id"`
	NotificationIDs []string                 `json:"notification_ids"`
	Status          model.NotificationStatus `json:"status"`
	Replayed        bool                     `json:"replayed"`
}

type BatchNotificationsResponse struct {
	Batch NotificationBatchResource `json:"batch"`
}

type NotificationBatchResource struct {
	ID             string                 `json:"id"`
	IdempotencyKey *string                `json:"idempotency_key,omitempty"`
	TotalCount     int                    `json:"total_count"`
	CreatedAt      time.Time              `json:"created_at"`
	Notifications  []NotificationResource `json:"notifications"`
}

type BatchNotificationsDetailResponse struct {
	Batch BatchNotificationsCreateResource `json:"batch"`
}

type CreateNotificationsResponse struct {
	BatchID         string                   `json:"batch_id"`
	NotificationIDs []string                 `json:"notification_ids"`
	Status          model.NotificationStatus `json:"status"`
	Replayed        bool                     `json:"replayed"`
}

func NewNotificationResource(notification *model.Notification) *NotificationResource {
	r := &NotificationResource{
		ID:                notification.ID.String(),
		BatchID:           uuidPtrToStringPtr(notification.BatchID),
		TemplateID:        uuidPtrToStringPtr(notification.TemplateID),
		Recipient:         notification.Recipient,
		Channel:           notification.Channel,
		Content:           notification.Content,
		Priority:          notification.Priority,
		Status:            notification.Status,
		AttemptCount:      notification.AttemptCount,
		MaxAttempts:       notification.MaxAttempts,
		ScheduledAt:       notification.ScheduledAt,
		NextAttemptAt:     notification.NextAttemptAt,
		ProviderMessageID: notification.ProviderMessageID,
		LastError:         notification.LastError,
		CancelledAt:       notification.CancelledAt,
		CreatedAt:         notification.CreatedAt,
		UpdatedAt:         notification.UpdatedAt,
	}
	return r
}

func NewNotificationsResource(notifications []model.Notification) []NotificationResource {
	var notificationsResource []NotificationResource

	for i := range notifications {
		r := NewNotificationResource(&notifications[i])
		notificationsResource = append(notificationsResource, *r)
	}
	return notificationsResource
}

func uuidPtrToStringPtr(value *uuid.UUID) *string {
	if value == nil {
		return nil
	}

	result := value.String()
	return &result
}
