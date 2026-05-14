package service_test

import (
	"context"
	"time"

	requestdto "insider-one-assessment/internal/dto/request"
	"insider-one-assessment/internal/model"
	"insider-one-assessment/internal/repository"
	repositorymocks "insider-one-assessment/internal/repository/mocks"
	. "insider-one-assessment/internal/service"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func mustUUID(value string) uuid.UUID {
	return uuid.MustParse(value)
}

func mustUUIDPtr(value string) *uuid.UUID {
	parsed := mustUUID(value)
	return &parsed
}

var _ = Describe("Notification Service", func() {
	var (
		ctx                  context.Context
		ctrl                 *gomock.Controller
		notificationRepoMock *repositorymocks.MockINotificationRepository
		templateRepoMock     *repositorymocks.MockITemplateRepository
		svc                  INotificationService
		templateSvc          ITemplateService
	)

	BeforeEach(func() {
		ctx = context.Background()
		ctrl = gomock.NewController(GinkgoT())
		notificationRepoMock = repositorymocks.NewMockINotificationRepository(ctrl)
		templateRepoMock = repositorymocks.NewMockITemplateRepository(ctrl)
		svc = NewNotificationService(notificationRepoMock, notificationRepoMock, templateRepoMock)
		templateSvc = NewTemplateService(templateRepoMock)
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Describe("CreateNotificationBatch", func() {
		It("should return batch size validation error", func() {
			req := requestdto.CreateNotificationsRequest{
				Notifications: make([]requestdto.NotificationItem, 1001),
			}

			resp, err := svc.CreateBatchNotification(ctx, req)

			Expect(err).To(MatchError("batch size exceeds 1000"))
			Expect(resp).To(BeNil())
		})

		It("should call repository and map the create response", func() {
			req := requestdto.CreateNotificationsRequest{
				IdempotencyKey: "idem-1",
				Notifications: []requestdto.NotificationItem{
					{
						Recipient: "+905551234567",
						Channel:   model.ChannelSMS,
						Content:   "hello",
					},
				},
			}

			notificationRepoMock.EXPECT().
				CreateNotifications(gomock.Any(), gomock.AssignableToTypeOf(repository.CreateNotificationsParams{})).
				DoAndReturn(func(_ context.Context, params repository.CreateNotificationsParams) (model.NotificationBatch, []model.Notification, bool, error) {
					Expect(params.IdempotencyKey).To(Equal("idem-1"))
					Expect(params.RequestHash).NotTo(BeEmpty())
					Expect(params.Notifications).To(HaveLen(1))

					return model.NotificationBatch{
							ID: mustUUID("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
						}, []model.Notification{
							{ID: mustUUID("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"), Status: model.StatusPending},
						}, true, nil
				})

			resp, err := svc.CreateBatchNotification(ctx, req)

			Expect(err).NotTo(HaveOccurred())
			Expect(resp.BatchID).To(Equal("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"))
			Expect(resp.NotificationIDs).To(Equal([]string{"bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"}))
			Expect(resp.Status).To(Equal(model.StatusPending))
			Expect(resp.Replayed).To(BeTrue())
		})

		It("should resolve content from template snapshot", func() {
			templateID := "11111111-1111-1111-1111-111111111111"
			templateRepoMock.EXPECT().
				GetTemplateByID(gomock.Any(), templateID).
				Return(&model.MessageTemplate{
					ID:      mustUUID(templateID),
					Name:    "welcome-push",
					Channel: model.ChannelPush,
					Content: "Hello {{name}}, your code is {{code}}",
				}, nil)
			notificationRepoMock.EXPECT().
				CreateNotifications(gomock.Any(), gomock.AssignableToTypeOf(repository.CreateNotificationsParams{})).
				DoAndReturn(func(_ context.Context, params repository.CreateNotificationsParams) (model.NotificationBatch, []model.Notification, bool, error) {
					Expect(params.Notifications).To(HaveLen(1))
					Expect(params.Notifications[0].TemplateID).NotTo(BeNil())
					Expect(*params.Notifications[0].TemplateID).To(Equal(templateID))
					Expect(params.Notifications[0].Content).To(Equal("Hello Baris, your code is 123456"))

					return model.NotificationBatch{ID: mustUUID("cccccccc-cccc-cccc-cccc-cccccccccccc")}, []model.Notification{{ID: mustUUID("dddddddd-dddd-dddd-dddd-dddddddddddd")}}, false, nil
				})

			resp, err := svc.CreateBatchNotification(ctx, requestdto.CreateNotificationsRequest{
				Notifications: []requestdto.NotificationItem{
					{
						Recipient:  "push-token",
						Channel:    model.ChannelPush,
						TemplateID: &templateID,
						Variables: map[string]string{
							"name": "Baris",
							"code": "123456",
						},
					},
				},
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(resp.BatchID).To(Equal("cccccccc-cccc-cccc-cccc-cccccccccccc"))
			Expect(resp.NotificationIDs).To(Equal([]string{"dddddddd-dddd-dddd-dddd-dddddddddddd"}))
		})

		It("should return validation error when a template variable is missing", func() {
			templateID := "11111111-1111-1111-1111-111111111111"
			templateRepoMock.EXPECT().
				GetTemplateByID(gomock.Any(), templateID).
				Return(&model.MessageTemplate{
					ID:      mustUUID(templateID),
					Name:    "welcome-push",
					Channel: model.ChannelPush,
					Content: "Hello {{name}}, your code is {{code}}",
				}, nil)

			resp, err := svc.CreateBatchNotification(ctx, requestdto.CreateNotificationsRequest{
				Notifications: []requestdto.NotificationItem{
					{
						Recipient:  "push-token",
						Channel:    model.ChannelPush,
						TemplateID: &templateID,
						Variables: map[string]string{
							"name": "Baris",
						},
					},
				},
			})

			Expect(resp).To(BeNil())
			Expect(err).To(MatchError("missing template variable: code"))
		})
	})

	Describe("CreateNotification", func() {
		It("should create a single notification without batch-size semantics", func() {
			notificationRepoMock.EXPECT().
				CreateNotification(gomock.Any(), "", gomock.Any(), gomock.AssignableToTypeOf(requestdto.NotificationItem{})).
				DoAndReturn(func(_ context.Context, idempotencyKey, requestHash string, item requestdto.NotificationItem) (*model.Notification, bool, error) {
					Expect(idempotencyKey).To(BeEmpty())
					Expect(requestHash).NotTo(BeEmpty())
					Expect(item.Recipient).To(Equal("+905551234567"))
					return &model.Notification{
						ID:           mustUUID("22222222-2222-2222-2222-222222222222"),
						BatchID:      nil,
						Recipient:    "+905551234567",
						Channel:      model.ChannelSMS,
						Content:      "hello",
						Priority:     model.PriorityNormal,
						Status:       model.StatusPending,
						AttemptCount: 0,
						MaxAttempts:  5,
					}, false, nil
				})

			resp, err := svc.CreateNotification(ctx, requestdto.CreateNotificationRequest{
				Notification: requestdto.NotificationItem{
					Recipient: "+905551234567",
					Channel:   model.ChannelSMS,
					Content:   "hello",
				},
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(resp.ID).To(Equal("22222222-2222-2222-2222-222222222222"))
			Expect(resp.BatchID).To(BeNil())
			Expect(resp.Status).To(Equal(model.StatusPending))
		})
	})

	Describe("Template CRUD", func() {
		It("should create template", func() {
			templateRepoMock.EXPECT().
				CreateTemplate(gomock.Any(), gomock.AssignableToTypeOf(&model.MessageTemplate{})).
				DoAndReturn(func(_ context.Context, template *model.MessageTemplate) error {
					Expect(template.Name).To(Equal("welcome-email"))
					Expect(template.Channel).To(Equal(model.ChannelEmail))
					Expect(template.Content).To(Equal("hello email"))
					template.ID = mustUUID("11111111-1111-1111-1111-111111111111")
					return nil
				})

			resp, err := templateSvc.CreateTemplate(ctx, requestdto.CreateTemplateRequest{
				Name:    "welcome-email",
				Channel: model.ChannelEmail,
				Content: "hello email",
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(resp.ID).To(Equal("11111111-1111-1111-1111-111111111111"))
			Expect(resp.Name).To(Equal("welcome-email"))
		})

		It("should pass channel filter to repository", func() {
			channel := model.ChannelPush
			templateRepoMock.EXPECT().
				ListTemplates(gomock.Any()).
				Return([]model.MessageTemplate{
					{
						ID:      mustUUID("11111111-1111-1111-1111-111111111111"),
						Name:    "welcome-push",
						Channel: model.ChannelPush,
						Content: "hello push",
					},
				}, nil)

			resp, err := templateSvc.ListTemplates(ctx)

			Expect(err).NotTo(HaveOccurred())
			Expect(resp).To(HaveLen(1))
			Expect(resp[0].Channel).To(Equal(model.ChannelPush))
			Expect(resp[0].Channel).To(Equal(channel))
		})

		It("should update template", func() {
			updatedContent := "updated content"
			templateID := "11111111-1111-1111-1111-111111111111"
			templateRepoMock.EXPECT().
				GetTemplateByID(gomock.Any(), templateID).
				Return(&model.MessageTemplate{
					ID:      mustUUID(templateID),
					Name:    "welcome-email",
					Channel: model.ChannelEmail,
					Content: "hello email",
				}, nil)
			templateRepoMock.EXPECT().
				UpdateTemplate(gomock.Any(), gomock.AssignableToTypeOf(&model.MessageTemplate{})).
				DoAndReturn(func(_ context.Context, template *model.MessageTemplate) error {
					Expect(template.ID).To(Equal(mustUUID(templateID)))
					Expect(template.Content).To(Equal(updatedContent))
					return nil
				})

			resp, err := templateSvc.UpdateTemplate(ctx, templateID, requestdto.UpdateTemplateRequest{
				Content: &updatedContent,
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(resp.Content).To(Equal(updatedContent))
		})
	})

	Describe("GetNotificationByID", func() {
		It("should map repository not found error", func() {
			notificationID := "22222222-2222-2222-2222-222222222222"
			notificationRepoMock.EXPECT().GetNotificationByID(gomock.Any(), notificationID).Return(nil, repository.ErrNotificationNotFound)

			resp, err := svc.GetNotificationByID(ctx, notificationID)

			Expect(err).To(MatchError("notification not found"))
			Expect(resp).To(BeNil())
		})

		It("should map the notification resource on success", func() {
			now := time.Now().UTC().Truncate(time.Second)
			notificationID := "22222222-2222-2222-2222-222222222222"
			batchID := "33333333-3333-3333-3333-333333333333"
			notificationRepoMock.EXPECT().GetNotificationByID(gomock.Any(), notificationID).Return(&model.Notification{
				ID:           mustUUID(notificationID),
				BatchID:      mustUUIDPtr(batchID),
				Recipient:    "user@example.com",
				Channel:      model.ChannelEmail,
				Content:      "hello",
				Priority:     model.PriorityNormal,
				Status:       model.StatusSent,
				AttemptCount: 1,
				MaxAttempts:  5,
				CreatedAt:    now,
				UpdatedAt:    now,
			}, nil)

			resp, err := svc.GetNotificationByID(ctx, notificationID)

			Expect(err).NotTo(HaveOccurred())
			Expect(resp.ID).To(Equal(notificationID))
			Expect(resp.BatchID).NotTo(BeNil())
			Expect(*resp.BatchID).To(Equal(batchID))
			Expect(resp.Channel).To(Equal(model.ChannelEmail))
			Expect(resp.Status).To(Equal(model.StatusSent))
		})
	})

	Describe("GetBatchByID", func() {
	})

	Describe("ListNotifications", func() {
		It("should apply default pagination when request values are zero", func() {
			notificationRepoMock.EXPECT().
				ListNotifications(gomock.Any(), gomock.AssignableToTypeOf(repository.ListNotificationsParams{})).
				DoAndReturn(func(_ context.Context, params repository.ListNotificationsParams) ([]model.Notification, int, error) {
					Expect(params.Limit).To(Equal(20))
					Expect(params.Offset).To(Equal(0))
					return nil, 0, nil
				})

			resp, err := svc.ListNotifications(ctx, requestdto.ListNotificationsRequest{
				Page:     0,
				PageSize: 0,
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(resp.Total).To(BeZero())
			Expect(resp.Notifications).To(BeEmpty())
			Expect(resp.Page).To(Equal(1))
			Expect(resp.PageSize).To(Equal(20))
		})

		It("should map listed notifications", func() {
			now := time.Now().UTC().Truncate(time.Second)
			notificationRepoMock.EXPECT().
				ListNotifications(gomock.Any(), gomock.AssignableToTypeOf(repository.ListNotificationsParams{})).
				Return([]model.Notification{
					{
						ID:           mustUUID("eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee"),
						Recipient:    "+905551234567",
						Channel:      model.ChannelSMS,
						Content:      "hello",
						Priority:     model.PriorityHigh,
						Status:       model.StatusQueued,
						AttemptCount: 0,
						MaxAttempts:  5,
						CreatedAt:    now,
						UpdatedAt:    now,
					},
				}, 1, nil)

			resp, err := svc.ListNotifications(ctx, requestdto.ListNotificationsRequest{
				Page:     1,
				PageSize: 20,
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(resp.Total).To(Equal(1))
			Expect(resp.Notifications).To(HaveLen(1))
			Expect(resp.Notifications[0].ID).To(Equal("eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee"))
			Expect(resp.Notifications[0].Status).To(Equal(model.StatusQueued))
		})
	})

	Describe("CancelNotification", func() {
		It("should map repository not cancellable error", func() {
			notificationID := "22222222-2222-2222-2222-222222222222"
			notificationRepoMock.EXPECT().
				CancelNotification(gomock.Any(), notificationID, gomock.AssignableToTypeOf(time.Time{})).
				Return(nil, repository.ErrNotificationNotCancellable)

			resp, err := svc.CancelNotification(ctx, notificationID)

			Expect(err).To(MatchError("notification is not cancellable"))
			Expect(resp).To(BeNil())
		})

		It("should map cancelled notification resource", func() {
			now := time.Now().UTC().Truncate(time.Second)
			notificationID := "22222222-2222-2222-2222-222222222222"
			notificationRepoMock.EXPECT().
				CancelNotification(gomock.Any(), notificationID, gomock.AssignableToTypeOf(time.Time{})).
				Return(&model.Notification{
					ID:           mustUUID(notificationID),
					Recipient:    "+905551234567",
					Channel:      model.ChannelSMS,
					Content:      "hello",
					Priority:     model.PriorityNormal,
					Status:       model.StatusCancelled,
					AttemptCount: 0,
					MaxAttempts:  5,
					CancelledAt:  &now,
					CreatedAt:    now,
					UpdatedAt:    now,
				}, nil)

			resp, err := svc.CancelNotification(ctx, notificationID)

			Expect(err).NotTo(HaveOccurred())
			Expect(resp.ID).To(Equal(notificationID))
			Expect(resp.Status).To(Equal(model.StatusCancelled))
			Expect(resp.CancelledAt).NotTo(BeNil())
		})
	})
})
