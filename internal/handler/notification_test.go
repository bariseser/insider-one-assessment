//go:build integration

package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"insider-one-assessment/internal/dto/request"
	"insider-one-assessment/internal/dto/resource"
	"insider-one-assessment/internal/model"
	"insider-one-assessment/internal/repository"
	"insider-one-assessment/internal/utils"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Notification Handler Integration", func() {
	Describe("Template CRUD", func() {
		It("should create, list, get, and update templates", func() {
			createResp := doJSONEnvelope[resource.TemplateResponse](http.MethodPost, "/templates", request.CreateTemplateRequest{
				Name:    "welcome-push",
				Channel: model.ChannelPush,
				Content: "hello from push template",
			}, http.StatusCreated)

			Expect(createResp.Template.ID).NotTo(BeEmpty())
			Expect(createResp.Template.Name).To(Equal("welcome-push"))

			listResp := doGetJSONEnvelope[resource.TemplatesResponse]("/templates?channel=push", http.StatusOK)
			Expect(listResp.Templates).To(HaveLen(1))
			Expect(listResp.Templates[0].ID).To(Equal(createResp.Template.ID))
			Expect(listResp.Templates[0].Channel).To(Equal(model.ChannelPush))

			getResp := doGetJSONEnvelope[resource.TemplateResponse]("/templates/"+createResp.Template.ID, http.StatusOK)
			Expect(getResp.Template.ID).To(Equal(createResp.Template.ID))

			updatedContent := "updated push template"
			updateResp := doJSONEnvelope[resource.TemplateResponse](http.MethodPatch, "/templates/"+createResp.Template.ID, request.UpdateTemplateRequest{
				Content: &updatedContent,
			}, http.StatusOK)
			Expect(updateResp.Template.Content).To(Equal(updatedContent))
		})
	})

	Describe("GET /health", func() {
		It("should return ok", func() {
			resp, err := http.Get(baseURL + "/health")
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			var payload map[string]string
			Expect(json.NewDecoder(resp.Body).Decode(&payload)).To(Succeed())
			Expect(payload["status"]).To(Equal("ok"))
		})
	})

	Describe("POST /notifications and GET /notifications/{id}", func() {
		It("should create a single notification and fetch it back", func() {
			scheduledAt := time.Now().UTC().Add(time.Hour).Truncate(time.Second)
			body := request.CreateNotificationRequest{
				Notification: request.NotificationItem{
					Recipient:   "+905551234567",
					Channel:     model.ChannelSMS,
					Content:     "hello sms",
					Priority:    model.PriorityHigh,
					ScheduledAt: &scheduledAt,
				},
			}

			createResp := doJSONEnvelopeWithHeaders[resource.NotificationResponse](http.MethodPost, "/notifications", body, map[string]string{
				"Idempotency-Key": "handler-create-1",
			}, http.StatusAccepted)
			Expect(createResp.Notification.ID).NotTo(BeEmpty())
			Expect(createResp.Notification.BatchID).To(BeNil())
			Expect(createResp.Notification.Status).To(Equal(model.StatusPending))

			notificationResp := doGetJSONEnvelope[resource.NotificationResponse]("/notifications/"+createResp.Notification.ID, http.StatusOK)
			Expect(notificationResp.Notification.ID).To(Equal(createResp.Notification.ID))
			Expect(notificationResp.Notification.BatchID).To(BeNil())
			Expect(notificationResp.Notification.Status).To(Equal(model.StatusPending))
			Expect(notificationResp.Notification.ScheduledAt).NotTo(BeNil())
			Expect(notificationResp.Notification.ScheduledAt.UTC()).To(Equal(scheduledAt))
		})

		It("should snapshot template content when creating a notification", func() {
			templateResp := doJSONEnvelope[resource.TemplateResponse](http.MethodPost, "/templates", request.CreateTemplateRequest{
				Name:    "welcome-email",
				Channel: model.ChannelEmail,
				Content: "Hello {{name}}, your code is {{code}}",
			}, http.StatusCreated)

			createResp := doJSONEnvelope[resource.NotificationResponse](http.MethodPost, "/notifications", request.CreateNotificationRequest{
				Notification: request.NotificationItem{
					Recipient:  "user@example.com",
					Channel:    model.ChannelEmail,
					TemplateID: &templateResp.Template.ID,
					Variables: map[string]string{
						"name": "Baris",
						"code": "123456",
					},
				},
			}, http.StatusAccepted)

			notificationResp := doGetJSONEnvelope[resource.NotificationResponse]("/notifications/"+createResp.Notification.ID, http.StatusOK)
			Expect(notificationResp.Notification.TemplateID).NotTo(BeNil())
			Expect(*notificationResp.Notification.TemplateID).To(Equal(templateResp.Template.ID))
			Expect(notificationResp.Notification.Content).To(Equal("Hello Baris, your code is 123456"))

			updatedContent := "new template content"
			_ = doJSONEnvelope[resource.TemplateResponse](http.MethodPatch, "/templates/"+templateResp.Template.ID, request.UpdateTemplateRequest{
				Content: &updatedContent,
			}, http.StatusOK)

			notificationResp = doGetJSONEnvelope[resource.NotificationResponse]("/notifications/"+createResp.Notification.ID, http.StatusOK)
			Expect(notificationResp.Notification.Content).To(Equal("Hello Baris, your code is 123456"))
		})
	})

	Describe("POST /batches and GET /batches/{id}", func() {
		It("should create a batch and fetch it back", func() {
			body := request.CreateNotificationsRequest{
				Notifications: []request.NotificationItem{
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
						Priority:  model.PriorityNormal,
					},
				},
			}

			createResp := doJSONEnvelopeWithHeaders[resource.BatchNotificationsCreateResponse](http.MethodPost, "/batches", body, map[string]string{
				"Idempotency-Key": "handler-batch-create-1",
			}, http.StatusAccepted)
			Expect(createResp.Batch.BatchID).NotTo(BeEmpty())
			Expect(createResp.Batch.NotificationIDs).To(HaveLen(2))
			Expect(createResp.Batch.Status).To(Equal(model.StatusPending))

			batchResp := doGetJSONEnvelope[resource.BatchNotificationsResponse]("/batches/"+createResp.Batch.BatchID, http.StatusOK)
			Expect(batchResp.Batch.ID).To(Equal(createResp.Batch.BatchID))
			Expect(batchResp.Batch.Notifications).To(HaveLen(2))
		})
	})

	Describe("GET /notifications", func() {
		It("should list notifications with pagination and filters", func() {
			body := request.CreateNotificationsRequest{
				Notifications: []request.NotificationItem{
					{
						Recipient: "+905551234567",
						Channel:   model.ChannelSMS,
						Content:   "hello sms",
					},
					{
						Recipient: "user@example.com",
						Channel:   model.ChannelEmail,
						Content:   "hello email",
					},
				},
			}

			_ = doJSONEnvelope[resource.BatchNotificationsCreateResponse](http.MethodPost, "/batches", body, http.StatusAccepted)

			listResp := doGetJSONEnvelope[resource.ListNotificationsResponse]("/notifications?channel=sms&page=1&page_size=20", http.StatusOK)

			Expect(listResp.Total).To(Equal(1))
			Expect(listResp.Notifications).To(HaveLen(1))
			Expect(listResp.Notifications[0].Channel).To(Equal(model.ChannelSMS))
		})
	})

	Describe("POST /notifications/{id}/cancel", func() {
		It("should cancel a pending notification", func() {
			createResp := doJSONEnvelope[resource.NotificationResponse](http.MethodPost, "/notifications", request.CreateNotificationRequest{
				Notification: request.NotificationItem{
					Recipient: "+905551234567",
					Channel:   model.ChannelSMS,
					Content:   "cancel me",
				},
			}, http.StatusAccepted)

			cancelResp := doJSONEnvelope[resource.NotificationResponse](http.MethodPost, "/notifications/"+createResp.Notification.ID+"/cancel", nil, http.StatusOK)

			Expect(cancelResp.Notification.ID).To(Equal(createResp.Notification.ID))
			Expect(cancelResp.Notification.Status).To(Equal(model.StatusCancelled))
			Expect(cancelResp.Notification.CancelledAt).NotTo(BeNil())
		})
	})

	Describe("GET /metrics", func() {
		It("should return a real-time snapshot", func() {
			_ = doJSONEnvelope[resource.NotificationResponse](http.MethodPost, "/notifications", request.CreateNotificationRequest{
				Notification: request.NotificationItem{
					Recipient: "+905551234567",
					Channel:   model.ChannelSMS,
					Content:   "metrics",
				},
			}, http.StatusAccepted)

			resp := doGetJSON[repository.MetricSnapshot]("/metrics", http.StatusOK)

			Expect(resp.GeneratedAt.IsZero()).To(BeFalse())
			Expect(resp.PendingCount).To(Equal(1))
			Expect(resp.QueueDepth).To(Equal(1))
			Expect(resp.CountsByChannel["sms"]).To(Equal(1))
		})
	})

	Describe("validation failures", func() {
		It("should return bad request for invalid json body", func() {
			resp, err := http.Post(baseURL+"/notifications", "application/json", bytes.NewBufferString("{"))
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))

			var payload utils.HTTPErrorResponse
			Expect(json.NewDecoder(resp.Body).Decode(&payload)).To(Succeed())
			Expect(payload.Error.Code).To(Equal("body_parser_failed"))
		})

		It("should return bad request for invalid batch id format", func() {
			resp, err := http.Get(baseURL + "/batches/string")
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))

			var payload utils.HTTPErrorResponse
			Expect(json.NewDecoder(resp.Body).Decode(&payload)).To(Succeed())
			Expect(payload.Error.Message).To(Equal("invalid request: batch id must be a valid uuid"))
		})

		It("should return bad request for sms content longer than 160 characters", func() {
			resp, err := http.Post(baseURL+"/notifications", "application/json", bytes.NewBufferString(`{
				"notification": {
					"recipient": "+905551234567",
					"channel": "sms",
					"content": "`+strings.Repeat("a", 161)+`"
				}
			}`))
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))

			var payload utils.HTTPErrorResponse
			Expect(json.NewDecoder(resp.Body).Decode(&payload)).To(Succeed())
			Expect(payload.Error.Message).To(Equal("sms content exceeds 160 characters"))
		})

		It("should return conflict for duplicate template names", func() {
			_ = doJSONEnvelope[resource.TemplateResponse](http.MethodPost, "/templates", request.CreateTemplateRequest{
				Name:    "duplicate-name",
				Channel: model.ChannelSMS,
				Content: "first",
			}, http.StatusCreated)

			resp, err := http.Post(baseURL+"/templates", "application/json", bytes.NewBufferString(`{
				"name":"duplicate-name",
				"channel":"email",
				"content":"second"
			}`))
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(http.StatusConflict))

			var payload utils.HTTPErrorResponse
			Expect(json.NewDecoder(resp.Body).Decode(&payload)).To(Succeed())
			Expect(payload.Error.Code).To(Equal("conflict"))
			Expect(payload.Error.Message).To(Equal("template already exists"))
		})
	})
})

func doJSON[T any](method, path string, body any, expectedStatus int) T {
	return doJSONWithHeaders[T](method, path, body, nil, expectedStatus)
}

func doJSONWithHeaders[T any](method, path string, body any, headers map[string]string, expectedStatus int) T {
	var payloadBytes []byte
	var err error
	if body != nil {
		payloadBytes, err = json.Marshal(body)
		Expect(err).NotTo(HaveOccurred())
	}

	req, err := http.NewRequest(method, baseURL+path, bytes.NewReader(payloadBytes))
	Expect(err).NotTo(HaveOccurred())
	req.Header.Set("Content-Type", "application/json")
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := http.DefaultClient.Do(req)
	Expect(err).NotTo(HaveOccurred())
	defer resp.Body.Close()

	Expect(resp.StatusCode).To(Equal(expectedStatus))

	var payload T
	Expect(json.NewDecoder(resp.Body).Decode(&payload)).To(Succeed())
	return payload
}

func doGetJSON[T any](path string, expectedStatus int) T {
	resp, err := http.Get(baseURL + path)
	Expect(err).NotTo(HaveOccurred())
	defer resp.Body.Close()

	Expect(resp.StatusCode).To(Equal(expectedStatus))

	var payload T
	Expect(json.NewDecoder(resp.Body).Decode(&payload)).To(Succeed())
	return payload
}

func doJSONEnvelope[T any](method, path string, body any, expectedStatus int) T {
	envelope := doJSON[utils.HTTPSuccessResponse](method, path, body, expectedStatus)

	payloadBytes, err := json.Marshal(envelope.Data)
	Expect(err).NotTo(HaveOccurred())

	var payload T
	Expect(json.Unmarshal(payloadBytes, &payload)).To(Succeed())
	return payload
}

func doJSONEnvelopeWithHeaders[T any](method, path string, body any, headers map[string]string, expectedStatus int) T {
	envelope := doJSONWithHeaders[utils.HTTPSuccessResponse](method, path, body, headers, expectedStatus)

	payloadBytes, err := json.Marshal(envelope.Data)
	Expect(err).NotTo(HaveOccurred())

	var payload T
	Expect(json.Unmarshal(payloadBytes, &payload)).To(Succeed())
	return payload
}

func doGetJSONEnvelope[T any](path string, expectedStatus int) T {
	envelope := doGetJSON[utils.HTTPSuccessResponse](path, expectedStatus)

	payloadBytes, err := json.Marshal(envelope.Data)
	Expect(err).NotTo(HaveOccurred())

	var payload T
	Expect(json.Unmarshal(payloadBytes, &payload)).To(Succeed())
	return payload
}
