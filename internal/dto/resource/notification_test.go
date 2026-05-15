package resource_test

import (
	"encoding/json"
	"time"

	resourcedto "insider-one-assessment/internal/dto/resource"
	"insider-one-assessment/internal/model"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Notification Resource DTOs", func() {
	Describe("CreateNotificationsResponse JSON contract", func() {
		It("should marshal expected response fields", func() {
			resp := resourcedto.CreateNotificationsResponse{
				BatchID:         "batch-1",
				NotificationIDs: []string{"notification-1", "notification-2"},
				Status:          model.StatusPending,
				Replayed:        true,
			}

			payload, err := json.Marshal(resp)

			Expect(err).NotTo(HaveOccurred())
			Expect(string(payload)).To(ContainSubstring(`"batch_id":"batch-1"`))
			Expect(string(payload)).To(ContainSubstring(`"notification_ids":["notification-1","notification-2"]`))
			Expect(string(payload)).To(ContainSubstring(`"status":"pending"`))
			Expect(string(payload)).To(ContainSubstring(`"replayed":true`))
		})
	})

	Describe("NotificationResource JSON contract", func() {
		It("should omit optional fields when they are nil or empty", func() {
			now := time.Date(2026, time.May, 13, 19, 0, 0, 0, time.UTC)
			resp := resourcedto.NotificationResource{
				ID:           "notification-1",
				Recipient:    "+905551234567",
				Channel:      model.ChannelSMS,
				Content:      "hello",
				Priority:     model.PriorityNormal,
				Status:       model.StatusPending,
				AttemptCount: 0,
				MaxAttempts:  5,
				CreatedAt:    now,
				UpdatedAt:    now,
			}

			payload, err := json.Marshal(resp)

			Expect(err).NotTo(HaveOccurred())
			Expect(string(payload)).NotTo(ContainSubstring(`"batch_id"`))
			Expect(string(payload)).NotTo(ContainSubstring(`"scheduled_at"`))
			Expect(string(payload)).NotTo(ContainSubstring(`"next_attempt_at"`))
			Expect(string(payload)).NotTo(ContainSubstring(`"provider_message_id"`))
			Expect(string(payload)).NotTo(ContainSubstring(`"last_error"`))
			Expect(string(payload)).NotTo(ContainSubstring(`"cancelled_at"`))
		})

		It("should include optional fields when they are present", func() {
			now := time.Date(2026, time.May, 13, 19, 0, 0, 0, time.UTC)
			batchID := "batch-1"
			scheduledAt := now.Add(time.Hour)
			nextAttemptAt := now.Add(2 * time.Hour)
			providerMessageID := "provider-1"
			lastError := "temporary provider failure"
			resp := resourcedto.NotificationResource{
				ID:                "notification-1",
				BatchID:           &batchID,
				Recipient:         "user@example.com",
				Channel:           model.ChannelEmail,
				Content:           "hello",
				Priority:          model.PriorityHigh,
				Status:            model.StatusFailed,
				AttemptCount:      2,
				MaxAttempts:       5,
				ScheduledAt:       &scheduledAt,
				NextAttemptAt:     &nextAttemptAt,
				ProviderMessageID: &providerMessageID,
				LastError:         &lastError,
				CancelledAt:       &now,
				CreatedAt:         now,
				UpdatedAt:         now,
			}

			payload, err := json.Marshal(resp)

			Expect(err).NotTo(HaveOccurred())
			Expect(string(payload)).To(ContainSubstring(`"batch_id":"batch-1"`))
			Expect(string(payload)).To(ContainSubstring(`"scheduled_at":"2026-05-13T20:00:00Z"`))
			Expect(string(payload)).To(ContainSubstring(`"next_attempt_at":"2026-05-13T21:00:00Z"`))
			Expect(string(payload)).To(ContainSubstring(`"provider_message_id":"provider-1"`))
			Expect(string(payload)).To(ContainSubstring(`"last_error":"temporary provider failure"`))
			Expect(string(payload)).To(ContainSubstring(`"cancelled_at":"2026-05-13T19:00:00Z"`))
		})
	})

	Describe("NotificationBatchResource", func() {
		It("should marshal nested notifications", func() {
			now := time.Date(2026, time.May, 13, 19, 0, 0, 0, time.UTC)
			idempotencyKey := "idem-1"
			resp := resourcedto.NotificationBatchResource{
				ID:             "batch-1",
				IdempotencyKey: &idempotencyKey,
				TotalCount:     1,
				CreatedAt:      now,
				Notifications: []resourcedto.NotificationResource{
					{
						ID:           "notification-1",
						Recipient:    "+905551234567",
						Channel:      model.ChannelSMS,
						Content:      "hello",
						Priority:     model.PriorityNormal,
						Status:       model.StatusPending,
						AttemptCount: 0,
						MaxAttempts:  5,
						CreatedAt:    now,
						UpdatedAt:    now,
					},
				},
			}

			payload, err := json.Marshal(resp)

			Expect(err).NotTo(HaveOccurred())
			Expect(string(payload)).To(ContainSubstring(`"id":"batch-1"`))
			Expect(string(payload)).To(ContainSubstring(`"idempotency_key":"idem-1"`))
			Expect(string(payload)).To(ContainSubstring(`"total_count":1`))
			Expect(string(payload)).To(ContainSubstring(`"notifications"`))
			Expect(string(payload)).To(ContainSubstring(`"id":"notification-1"`))
		})
	})

	Describe("ListNotificationsResponse", func() {
		It("should marshal pagination metadata", func() {
			resp := resourcedto.ListNotificationsResponse{
				Notifications: []resourcedto.NotificationResource{
					{ID: "notification-1"},
				},
				Page:     2,
				PageSize: 50,
				Total:    101,
			}

			payload, err := json.Marshal(resp)

			Expect(err).NotTo(HaveOccurred())
			Expect(string(payload)).To(ContainSubstring(`"page":2`))
			Expect(string(payload)).To(ContainSubstring(`"page_size":50`))
			Expect(string(payload)).To(ContainSubstring(`"total":101`))
			Expect(string(payload)).To(ContainSubstring(`"notifications"`))
		})
	})
})
