package model

import (
	"time"

	"github.com/google/uuid"
)

type MessageTemplate struct {
	ID        uuid.UUID           `gorm:"column:id;type:uuid;primaryKey"`
	Name      string              `gorm:"column:name;type:text;not null"`
	Channel   NotificationChannel `gorm:"column:channel;type:notification_channel;not null"`
	Content   string              `gorm:"column:content;type:text;not null"`
	CreatedAt time.Time           `gorm:"column:created_at;not null"`
	UpdatedAt time.Time           `gorm:"column:updated_at;not null"`
}

func (MessageTemplate) TableName() string {
	return "message_templates"
}
