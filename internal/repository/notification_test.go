package repository_test

import (
	"database/sql"
	"encoding/json"
	"time"

	requestdto "insider-one-assessment/internal/dto/request"
	"insider-one-assessment/internal/model"
	"insider-one-assessment/internal/repository"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Notification Repository", func() {
	Describe("CreateNotification", func() {
		It("should create a single notification without creating a batch row", func() {
			notification, replayed, err := notificationRepository.CreateNotification(ctx, "idem-single-1", "hash-single-1", requestdto.NotificationItem{
				Recipient: "+905551234567",
				Channel:   model.ChannelSMS,
				Content:   "hello single",
				Priority:  model.PriorityNormal,
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(replayed).To(BeFalse())
			Expect(notification.ID).NotTo(BeEmpty())
			Expect(notification.BatchID).To(BeNil())
			Expect(notification.Status).To(Equal(model.StatusPending))

			var batchCount int
			Expect(db.QueryRowContext(ctx, `SELECT COUNT(*) FROM notification_batches`).Scan(&batchCount)).To(Succeed())
			Expect(batchCount).To(Equal(0))

			var persistedBatchID sql.NullString
			Expect(db.QueryRowContext(ctx, `SELECT batch_id FROM notifications WHERE id = $1`, notification.ID).Scan(&persistedBatchID)).To(Succeed())
			Expect(persistedBatchID.Valid).To(BeFalse())

			var idempotencyBatchID sql.NullString
			var idempotencyNotificationID sql.NullString
			Expect(db.QueryRowContext(ctx, `SELECT batch_id, notification_id FROM idempotency_keys WHERE idempotency_key = $1`, "idem-single-1").Scan(&idempotencyBatchID, &idempotencyNotificationID)).To(Succeed())
			Expect(idempotencyBatchID.Valid).To(BeFalse())
			Expect(idempotencyNotificationID.Valid).To(BeTrue())
			Expect(idempotencyNotificationID.String).To(Equal(notification.ID.String()))

			var payload []byte
			Expect(db.QueryRowContext(ctx, `SELECT payload FROM outbox_events WHERE aggregate_id = $1`, notification.ID).Scan(&payload)).To(Succeed())

			var decoded map[string]any
			Expect(json.Unmarshal(payload, &decoded)).To(Succeed())
			Expect(decoded["notification_id"]).To(Equal(notification.ID.String()))
			Expect(decoded).NotTo(HaveKey("batch_id"))
		})
	})

	Describe("CreateNotifications", func() {
		It("should create a batch, notifications, idempotency key, and outbox events", func() {
			params := repository.CreateNotificationsParams{
				IdempotencyKey: "idem-create-1",
				RequestHash:    "hash-create-1",
				Notifications: []requestdto.NotificationItem{
					{
						Recipient: "+905551234567",
						Channel:   model.ChannelSMS,
						Content:   "hello sms",
						Priority:  model.PriorityHigh,
					},
					{
						Recipient: "user@example.com",
						Channel:   model.ChannelEmail,
						Content:   "hello email",
						Priority:  model.PriorityLow,
					},
				},
			}

			batch, notifications, replayed, err := notificationRepository.CreateNotifications(ctx, params)

			Expect(err).NotTo(HaveOccurred())
			Expect(replayed).To(BeFalse())
			Expect(batch.ID).NotTo(BeEmpty())
			Expect(batch.TotalCount).To(Equal(2))
			Expect(notifications).To(HaveLen(2))
			Expect(notifications[0].BatchID).NotTo(BeNil())
			Expect(*notifications[0].BatchID).To(Equal(batch.ID))
			Expect(notifications[0].Status).To(Equal(model.StatusPending))
			Expect(notifications[1].Status).To(Equal(model.StatusPending))

			var notificationCount int
			Expect(db.QueryRowContext(ctx, `SELECT COUNT(*) FROM notifications WHERE batch_id = $1`, batch.ID).Scan(&notificationCount)).To(Succeed())
			Expect(notificationCount).To(Equal(2))

			var idempotencyCount int
			Expect(db.QueryRowContext(ctx, `SELECT COUNT(*) FROM idempotency_keys WHERE batch_id = $1`, batch.ID).Scan(&idempotencyCount)).To(Succeed())
			Expect(idempotencyCount).To(Equal(1))

			var outboxCount int
			Expect(db.QueryRowContext(ctx, `SELECT COUNT(*) FROM outbox_events WHERE aggregate_id IN ($1, $2)`, notifications[0].ID, notifications[1].ID).Scan(&outboxCount)).To(Succeed())
			Expect(outboxCount).To(Equal(2))

			var payload []byte
			Expect(db.QueryRowContext(ctx, `SELECT payload FROM outbox_events WHERE aggregate_id = $1`, notifications[0].ID).Scan(&payload)).To(Succeed())

			var decoded map[string]any
			Expect(json.Unmarshal(payload, &decoded)).To(Succeed())
			Expect(decoded["notification_id"]).To(Equal(notifications[0].ID.String()))
			Expect(decoded["batch_id"]).To(Equal(batch.ID.String()))
			Expect(decoded["channel"]).To(Equal(string(model.ChannelSMS)))
			Expect(decoded["priority"]).To(Equal(string(model.PriorityHigh)))
		})

		It("should replay the existing batch for the same idempotency key and request hash", func() {
			params := repository.CreateNotificationsParams{
				IdempotencyKey: "idem-replay-1",
				RequestHash:    "hash-replay-1",
				Notifications: []requestdto.NotificationItem{
					{
						Recipient: "+905551234567",
						Channel:   model.ChannelSMS,
						Content:   "hello replay",
					},
				},
			}

			firstBatch, firstNotifications, replayed, err := notificationRepository.CreateNotifications(ctx, params)
			Expect(err).NotTo(HaveOccurred())
			Expect(replayed).To(BeFalse())

			secondBatch, secondNotifications, replayed, err := notificationRepository.CreateNotifications(ctx, params)

			Expect(err).NotTo(HaveOccurred())
			Expect(replayed).To(BeTrue())
			Expect(secondBatch.ID).To(Equal(firstBatch.ID))
			Expect(secondNotifications).To(HaveLen(1))
			Expect(secondNotifications[0].ID).To(Equal(firstNotifications[0].ID))

			var batchCount int
			Expect(db.QueryRowContext(ctx, `SELECT COUNT(*) FROM notification_batches`).Scan(&batchCount)).To(Succeed())
			Expect(batchCount).To(Equal(1))

			var notificationCount int
			Expect(db.QueryRowContext(ctx, `SELECT COUNT(*) FROM notifications`).Scan(&notificationCount)).To(Succeed())
			Expect(notificationCount).To(Equal(1))
		})

		It("should schedule outbox availability from scheduled_at", func() {
			scheduledAt := time.Now().UTC().Add(2 * time.Hour).Truncate(time.Second)
			params := repository.CreateNotificationsParams{
				RequestHash: "hash-scheduled-1",
				Notifications: []requestdto.NotificationItem{
					{
						Recipient:   "user@example.com",
						Channel:     model.ChannelEmail,
						Content:     "scheduled email",
						Priority:    model.PriorityNormal,
						ScheduledAt: &scheduledAt,
					},
				},
			}

			_, notifications, replayed, err := notificationRepository.CreateNotifications(ctx, params)

			Expect(err).NotTo(HaveOccurred())
			Expect(replayed).To(BeFalse())
			Expect(notifications).To(HaveLen(1))
			Expect(notifications[0].ScheduledAt).NotTo(BeNil())
			Expect(notifications[0].ScheduledAt.UTC()).To(Equal(scheduledAt))

			var availableAt time.Time
			Expect(db.QueryRowContext(ctx, `SELECT available_at FROM outbox_events WHERE aggregate_id = $1`, notifications[0].ID).Scan(&availableAt)).To(Succeed())
			Expect(availableAt.UTC()).To(Equal(scheduledAt))
		})

		It("should return idempotency conflict for the same key with a different request hash", func() {
			firstParams := repository.CreateNotificationsParams{
				IdempotencyKey: "idem-conflict-1",
				RequestHash:    "hash-a",
				Notifications: []requestdto.NotificationItem{
					{
						Recipient: "+905551234567",
						Channel:   model.ChannelSMS,
						Content:   "first payload",
					},
				},
			}
			secondParams := repository.CreateNotificationsParams{
				IdempotencyKey: "idem-conflict-1",
				RequestHash:    "hash-b",
				Notifications: []requestdto.NotificationItem{
					{
						Recipient: "user@example.com",
						Channel:   model.ChannelEmail,
						Content:   "second payload",
					},
				},
			}

			_, _, _, err := notificationRepository.CreateNotifications(ctx, firstParams)
			Expect(err).NotTo(HaveOccurred())

			_, _, _, err = notificationRepository.CreateNotifications(ctx, secondParams)

			Expect(err).To(MatchError(repository.ErrIdempotencyConflict))
		})
	})
})
