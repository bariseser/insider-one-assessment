package repository

import (
	"context"
	"time"

	"insider-one-assessment/internal/dto/request"
	"insider-one-assessment/internal/model"
)

type INotificationReadRepository interface {
	GetNotificationByID(ctx context.Context, id string) (*model.Notification, error)
	GetBatchByID(ctx context.Context, id string) (*model.NotificationBatch, []model.Notification, error)
	ListNotifications(ctx context.Context, params ListNotificationsParams) ([]model.Notification, int, error)
}

type INotificationWriteRepository interface {
	CreateNotification(ctx context.Context, idempotencyKey, requestHash string, item request.NotificationItem) (*model.Notification, bool, error)
	CreateNotifications(ctx context.Context, params CreateNotificationsParams) (model.NotificationBatch, []model.Notification, bool, error)
	CancelNotification(ctx context.Context, id string, cancelledAt time.Time) (*model.Notification, error)
}

type IDeliveryRepository interface {
	StartProcessingNotification(ctx context.Context, id string, startedAt time.Time) (*model.Notification, error)
	MarkNotificationSent(ctx context.Context, id string, attemptResult NotificationAttemptResult, providerMessageID string, sentAt time.Time) (*model.Notification, error)
	MarkNotificationFailed(ctx context.Context, id string, attemptResult NotificationAttemptResult, failedAt time.Time) (*model.Notification, error)
	ScheduleNotificationRetry(ctx context.Context, id string, attemptResult NotificationAttemptResult, nextAttemptAt time.Time) (*model.Notification, error)
}

type IOutboxRepository interface {
	ClaimDueOutboxEvents(ctx context.Context, limit int) ([]OutboxEvent, error)
	MarkOutboxEventPublished(ctx context.Context, eventID, notificationID string, publishedAt time.Time) error
}

type INotificationRepository interface {
	INotificationReadRepository
	INotificationWriteRepository
	IDeliveryRepository
	IOutboxRepository
}
