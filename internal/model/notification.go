package model

import (
	"time"

	"github.com/google/uuid"
)

type NotificationChannel string

const (
	ChannelSMS   NotificationChannel = "sms"
	ChannelEmail NotificationChannel = "email"
	ChannelPush  NotificationChannel = "push"
)

type NotificationPriority string

const (
	PriorityHigh   NotificationPriority = "high"
	PriorityNormal NotificationPriority = "normal"
	PriorityLow    NotificationPriority = "low"
)

type NotificationStatus string

const (
	StatusPending    NotificationStatus = "pending"
	StatusQueued     NotificationStatus = "queued"
	StatusProcessing NotificationStatus = "processing"
	StatusSent       NotificationStatus = "sent"
	StatusFailed     NotificationStatus = "failed"
	StatusCancelled  NotificationStatus = "cancelled"
)

type Notification struct {
	ID                uuid.UUID            `gorm:"column:id;type:uuid;primaryKey"`
	BatchID           *uuid.UUID           `gorm:"column:batch_id;type:uuid"`
	TemplateID        *uuid.UUID           `gorm:"column:template_id;type:uuid"`
	Recipient         string               `gorm:"column:recipient;type:text;not null"`
	Channel           NotificationChannel  `gorm:"column:channel;type:notification_channel;not null"`
	Content           string               `gorm:"column:content;type:text;not null"`
	Priority          NotificationPriority `gorm:"column:priority;type:notification_priority;not null;default:normal"`
	Status            NotificationStatus   `gorm:"column:status;type:notification_status;not null;default:pending"`
	AttemptCount      int                  `gorm:"column:attempt_count;not null;default:0"`
	MaxAttempts       int                  `gorm:"column:max_attempts;not null;default:5"`
	ScheduledAt       *time.Time           `gorm:"column:scheduled_at"`
	NextAttemptAt     *time.Time           `gorm:"column:next_attempt_at"`
	ProviderMessageID *string              `gorm:"column:provider_message_id;type:text"`
	LastError         *string              `gorm:"column:last_error;type:text"`
	CancelledAt       *time.Time           `gorm:"column:cancelled_at"`
	CreatedAt         time.Time            `gorm:"column:created_at;not null"`
	UpdatedAt         time.Time            `gorm:"column:updated_at;not null"`
}

type NotificationBatch struct {
	ID             uuid.UUID `gorm:"column:id;type:uuid;primaryKey"`
	IdempotencyKey *string   `gorm:"column:idempotency_key;type:text"`
	TotalCount     int       `gorm:"column:total_count;not null"`
	CreatedAt      time.Time `gorm:"column:created_at;not null"`
}

func (Notification) TableName() string {
	return "notifications"
}

func (NotificationBatch) TableName() string {
	return "notification_batches"
}
