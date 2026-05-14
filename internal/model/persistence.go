package model

import (
	"time"

	"github.com/google/uuid"
)

type NotificationAttempt struct {
	ID                 uuid.UUID `gorm:"column:id;type:uuid;primaryKey"`
	NotificationID     uuid.UUID `gorm:"column:notification_id;type:uuid;not null"`
	AttemptNumber      int       `gorm:"column:attempt_number;not null"`
	ResponseStatusCode *int      `gorm:"column:response_status_code"`
	ResponseBody       *string   `gorm:"column:response_body;type:text"`
	ErrorMessage       *string   `gorm:"column:error_message;type:text"`
	CreatedAt          time.Time `gorm:"column:created_at;not null"`
}

type IdempotencyKey struct {
	ID             uuid.UUID  `gorm:"column:id;type:uuid;primaryKey"`
	IdempotencyKey string     `gorm:"column:idempotency_key;type:text;not null"`
	BatchID        *uuid.UUID `gorm:"column:batch_id;type:uuid"`
	NotificationID *uuid.UUID `gorm:"column:notification_id;type:uuid"`
	RequestHash    string     `gorm:"column:request_hash;type:text;not null"`
	CreatedAt      time.Time  `gorm:"column:created_at;not null"`
}

type OutboxEvent struct {
	ID            uuid.UUID  `gorm:"column:id;type:uuid;primaryKey"`
	AggregateType string     `gorm:"column:aggregate_type;type:text;not null"`
	AggregateID   uuid.UUID  `gorm:"column:aggregate_id;type:uuid;not null"`
	EventType     string     `gorm:"column:event_type;type:text;not null"`
	Payload       []byte     `gorm:"column:payload;type:jsonb;not null"`
	PublishedAt   *time.Time `gorm:"column:published_at"`
	AvailableAt   time.Time  `gorm:"column:available_at;not null"`
	CreatedAt     time.Time  `gorm:"column:created_at;not null"`
}

func (NotificationAttempt) TableName() string {
	return "notification_attempts"
}

func (IdempotencyKey) TableName() string {
	return "idempotency_keys"
}

func (OutboxEvent) TableName() string {
	return "outbox_events"
}
