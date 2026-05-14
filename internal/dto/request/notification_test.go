package request_test

import (
	"encoding/json"
	"time"

	requestdto "insider-one-assessment/internal/dto/request"
	"insider-one-assessment/internal/model"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Notification Request DTOs", func() {
	Describe("CreateNotificationRequest JSON contract", func() {
		It("should marshal a single notification with expected field names", func() {
			scheduledAt := time.Date(2026, time.May, 13, 18, 0, 0, 0, time.UTC)
			req := requestdto.CreateNotificationRequest{
				IdempotencyKey: "idem-1",
				Notification: requestdto.NotificationItem{
					Recipient:   "+905551234567",
					Channel:     model.ChannelSMS,
					Content:     "hello",
					Priority:    model.PriorityHigh,
					ScheduledAt: &scheduledAt,
				},
			}

			payload, err := json.Marshal(req)

			Expect(err).NotTo(HaveOccurred())
			Expect(string(payload)).NotTo(ContainSubstring(`"idempotency_key"`))
			Expect(string(payload)).To(ContainSubstring(`"notification"`))
			Expect(string(payload)).To(ContainSubstring(`"recipient":"+905551234567"`))
		})
	})

	Describe("CreateNotificationsRequest JSON contract", func() {
		It("should marshal notifications with expected field names", func() {
			scheduledAt := time.Date(2026, time.May, 13, 18, 0, 0, 0, time.UTC)
			req := requestdto.CreateNotificationsRequest{
				IdempotencyKey: "idem-1",
				Notifications: []requestdto.NotificationItem{
					{
						Recipient:   "+905551234567",
						Channel:     model.ChannelSMS,
						Content:     "hello",
						Priority:    model.PriorityHigh,
						ScheduledAt: &scheduledAt,
					},
				},
			}

			payload, err := json.Marshal(req)

			Expect(err).NotTo(HaveOccurred())
			Expect(string(payload)).NotTo(ContainSubstring(`"idempotency_key"`))
			Expect(string(payload)).To(ContainSubstring(`"notifications"`))
			Expect(string(payload)).To(ContainSubstring(`"recipient":"+905551234567"`))
			Expect(string(payload)).To(ContainSubstring(`"channel":"sms"`))
			Expect(string(payload)).To(ContainSubstring(`"content":"hello"`))
			Expect(string(payload)).To(ContainSubstring(`"priority":"high"`))
			Expect(string(payload)).To(ContainSubstring(`"scheduled_at":"2026-05-13T18:00:00Z"`))
		})

		It("should ignore idempotency_key in the request body", func() {
			raw := []byte(`{
				"idempotency_key": "idem-2",
				"notifications": [
					{
						"recipient": "user@example.com",
						"channel": "email",
						"content": "welcome",
						"priority": "normal"
					}
				]
			}`)

			var req requestdto.CreateNotificationsRequest
			err := json.Unmarshal(raw, &req)

			Expect(err).NotTo(HaveOccurred())
			Expect(req.IdempotencyKey).To(BeEmpty())
			Expect(req.Notifications).To(HaveLen(1))
			Expect(req.Notifications[0].Recipient).To(Equal("user@example.com"))
			Expect(req.Notifications[0].Channel).To(Equal(model.ChannelEmail))
			Expect(req.Notifications[0].Content).To(Equal("welcome"))
			Expect(req.Notifications[0].Priority).To(Equal(model.PriorityNormal))
		})
	})

	Describe("ListNotificationsRequest", func() {
		It("should hold filtering and pagination inputs", func() {
			req := requestdto.ListNotificationsRequest{
				Status:   "pending",
				Channel:  "sms",
				DateFrom: "2026-05-13T10:00:00Z",
				DateTo:   "2026-05-14T10:00:00Z",
				Page:     2,
				PageSize: 50,
			}

			Expect(req.Status).To(Equal("pending"))
			Expect(req.Channel).To(Equal("sms"))
			Expect(req.DateFrom).To(Equal("2026-05-13T10:00:00Z"))
			Expect(req.DateTo).To(Equal("2026-05-14T10:00:00Z"))
			Expect(req.Page).To(Equal(2))
			Expect(req.PageSize).To(Equal(50))
		})
	})

})
