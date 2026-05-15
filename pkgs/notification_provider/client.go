package notification_provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"insider-one-assessment/internal/tracing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

type Request struct {
	To      string `json:"to"`
	Channel string `json:"channel"`
	Content string `json:"content"`
}

type Response struct {
	MessageID string `json:"messageId"`
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
}

type Result struct {
	StatusCode   int
	ResponseBody string
	Response     Response
}

type INotificationClient interface {
	Send(ctx context.Context, req Request) (Result, error)
}

type notificationClient struct {
	baseURL          string
	httpClient       *http.Client
	mockResponseCode string
}

var tracer = otel.Tracer("insider-one-assessment/provider")

func NewNotificationClient(baseURL string, timeout time.Duration, mockResponseCode string) INotificationClient {
	return &notificationClient{
		baseURL:          baseURL,
		mockResponseCode: mockResponseCode,
		httpClient: &http.Client{
			Timeout:   timeout,
			Transport: tracing.HTTPTransport(http.DefaultTransport),
		},
	}
}

func (c *notificationClient) Send(ctx context.Context, req Request) (Result, error) {
	ctx, span := tracer.Start(ctx, "provider.send")
	defer span.End()

	span.SetAttributes(
		attribute.String("provider.channel", req.Channel),
		attribute.String("provider.recipient", req.To),
	)

	if c.mockResponseCode != "" {
		statusCode, err := strconv.Atoi(c.mockResponseCode)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return Result{}, fmt.Errorf("parse mock provider response code: %w", err)
		}

		result := Result{
			StatusCode:   statusCode,
			ResponseBody: fmt.Sprintf(`{"messageId":"mock-provider-message-id","status":"mocked","timestamp":"2026-01-01T00:00:00Z","statusCode":%d}`, statusCode),
			Response: Response{
				MessageID: "mock-provider-message-id",
				Status:    "mocked",
				Timestamp: "2026-01-01T00:00:00Z",
			},
		}
		span.SetAttributes(attribute.Int("http.response.status_code", statusCode))
		return result, nil
	}

	payload, err := json.Marshal(req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return Result{}, fmt.Errorf("marshal provider request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, bytes.NewReader(payload))
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return Result{}, fmt.Errorf("build provider request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return Result{}, fmt.Errorf("send provider request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return Result{}, fmt.Errorf("read provider response: %w", err)
	}
	span.SetAttributes(attribute.Int("http.response.status_code", resp.StatusCode))

	result := Result{
		StatusCode:   resp.StatusCode,
		ResponseBody: string(responseBody),
	}

	if err := json.Unmarshal(responseBody, &result.Response); err != nil {
		return result, nil
	}

	return result, nil
}
