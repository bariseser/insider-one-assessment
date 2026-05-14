package model_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"insider-one-assessment/internal/model"
)

var _ = Describe("Notification Model", func() {
	Describe("NotificationChannel", func() {
		It("should expose the expected channel values", func() {
			Expect(model.ChannelSMS).To(Equal(model.NotificationChannel("sms")))
			Expect(model.ChannelEmail).To(Equal(model.NotificationChannel("email")))
			Expect(model.ChannelPush).To(Equal(model.NotificationChannel("push")))
		})
	})

	Describe("NotificationPriority", func() {
		It("should expose the expected priority values", func() {
			Expect(model.PriorityHigh).To(Equal(model.NotificationPriority("high")))
			Expect(model.PriorityNormal).To(Equal(model.NotificationPriority("normal")))
			Expect(model.PriorityLow).To(Equal(model.NotificationPriority("low")))
		})
	})

	Describe("NotificationStatus", func() {
		It("should expose the expected status values", func() {
			Expect(model.StatusPending).To(Equal(model.NotificationStatus("pending")))
			Expect(model.StatusQueued).To(Equal(model.NotificationStatus("queued")))
			Expect(model.StatusProcessing).To(Equal(model.NotificationStatus("processing")))
			Expect(model.StatusSent).To(Equal(model.NotificationStatus("sent")))
			Expect(model.StatusFailed).To(Equal(model.NotificationStatus("failed")))
			Expect(model.StatusCancelled).To(Equal(model.NotificationStatus("cancelled")))
		})
	})

	Describe("GORM table names", func() {
		It("should expose the expected table names", func() {
			Expect(model.Notification{}.TableName()).To(Equal("notifications"))
			Expect(model.NotificationBatch{}.TableName()).To(Equal("notification_batches"))
			Expect(model.MessageTemplate{}.TableName()).To(Equal("message_templates"))
			Expect(model.NotificationAttempt{}.TableName()).To(Equal("notification_attempts"))
			Expect(model.IdempotencyKey{}.TableName()).To(Equal("idempotency_keys"))
			Expect(model.OutboxEvent{}.TableName()).To(Equal("outbox_events"))
		})
	})
})
