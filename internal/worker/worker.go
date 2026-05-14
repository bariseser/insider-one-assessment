package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"sync"
	"time"

	"insider-one-assessment/internal/model"
	"insider-one-assessment/internal/queue/rabbitmq"
	"insider-one-assessment/internal/repository"
	"insider-one-assessment/internal/tracing"
	"insider-one-assessment/pkgs/notification_provider"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"golang.org/x/time/rate"
)

type Worker struct {
	logger        *slog.Logger
	deliveryRepo  repository.IDeliveryRepository
	outboxRepo    repository.IOutboxRepository
	queue         rabbitmq.IClient
	provider      notification_provider.INotificationClient
	publisherTick time.Duration
	limiters      map[string]*rate.Limiter
	mu            sync.Mutex
	rng           *rand.Rand
}

var tracer = otel.Tracer("insider-one-assessment/worker")

func New(logger *slog.Logger, deliveryRepo repository.IDeliveryRepository, outboxRepo repository.IOutboxRepository, queue rabbitmq.IClient, provider notification_provider.INotificationClient, channelRateLimit int) *Worker {
	if channelRateLimit <= 0 {
		channelRateLimit = 100
	}

	return &Worker{
		logger:        logger,
		deliveryRepo:  deliveryRepo,
		outboxRepo:    outboxRepo,
		queue:         queue,
		provider:      provider,
		publisherTick: time.Second,
		limiters: map[string]*rate.Limiter{
			"sms":   rate.NewLimiter(rate.Limit(channelRateLimit), channelRateLimit),
			"email": rate.NewLimiter(rate.Limit(channelRateLimit), channelRateLimit),
			"push":  rate.NewLimiter(rate.Limit(channelRateLimit), channelRateLimit),
		},
		rng: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (w *Worker) Run(ctx context.Context) error {
	w.logger.Info("worker started", "publisher_tick", w.publisherTick.String())

	deliveries, err := w.queue.Consume()
	if err != nil {
		return err
	}

	go w.runOutboxPublisher(ctx)

	for {
		select {
		case <-ctx.Done():
			return nil
		case delivery, ok := <-deliveries:
			if !ok {
				return errors.New("rabbitmq delivery channel closed")
			}

			if err := w.processDelivery(ctx, delivery); err != nil {
				w.logger.Error("process delivery", "error", err)
			}
		}
	}
}

func (w *Worker) runOutboxPublisher(ctx context.Context) {
	ticker := time.NewTicker(w.publisherTick)
	defer ticker.Stop()

	for {
		if err := w.publishReadyOutboxEvents(ctx); err != nil && !errors.Is(err, context.Canceled) {
			w.logger.Error("publish ready outbox events", "error", err)
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (w *Worker) publishReadyOutboxEvents(ctx context.Context) error {
	ctx, span := tracer.Start(ctx, "outbox.publish_ready_events")
	defer span.End()

	events, err := w.outboxRepo.ClaimDueOutboxEvents(ctx, 100)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	span.SetAttributes(attribute.Int("outbox.claimed_count", len(events)))
	if len(events) > 0 {
		w.logger.Info("claimed outbox events", "count", len(events))
	}

	for _, event := range events {
		var message rabbitmq.DeliveryMessage
		if err := json.Unmarshal(event.Payload, &message); err != nil {
			w.logger.Error("unmarshal outbox payload", "event_id", event.ID, "error", err)
			continue
		}

		routingKey := fmt.Sprintf("notification.%s", message.Channel)
		publishCtx := tracing.ContextWithCarrier(ctx, message.TraceCarrier)
		if err := w.queue.Publish(publishCtx, routingKey, message); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return err
		}
		w.logger.Info("published notification to queue",
			"event_id", event.ID,
			"notification_id", message.NotificationID,
			"routing_key", routingKey,
		)

		if err := w.outboxRepo.MarkOutboxEventPublished(ctx, event.ID, message.NotificationID, time.Now().UTC()); err != nil && !errors.Is(err, repository.ErrOutboxEventNotFound) {
			return err
		}
	}

	return nil
}

func (w *Worker) processDelivery(ctx context.Context, delivery amqp.Delivery) error {
	var message rabbitmq.DeliveryMessage
	if err := json.Unmarshal(delivery.Body, &message); err != nil {
		_ = delivery.Nack(false, false)
		return fmt.Errorf("unmarshal delivery body: %w", err)
	}

	ctx = tracing.ContextFromAMQPHeaders(ctx, delivery.Headers)
	ctx, span := tracer.Start(ctx, "worker.process_notification")
	defer span.End()
	span.SetAttributes(
		attribute.String("notification.id", message.NotificationID),
		attribute.String("notification.channel", message.Channel),
	)

	notification, err := w.deliveryRepo.StartProcessingNotification(ctx, message.NotificationID, time.Now().UTC())
	if err != nil {
		if errors.Is(err, repository.ErrNotificationNotReady) {
			w.logger.Warn("notification not ready yet, requeueing delivery",
				"notification_id", message.NotificationID,
			)
			_ = delivery.Nack(false, true)
			return nil
		}
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		_ = delivery.Nack(false, true)
		return err
	}
	w.logger.Info("processing notification",
		"notification_id", notification.ID.String(),
		"batch_id", derefUUID(notification.BatchID),
		"channel", notification.Channel,
		"attempt_count", notification.AttemptCount,
	)

	if err := w.waitForChannelRateLimit(ctx, string(notification.Channel)); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		_ = delivery.Nack(false, true)
		return err
	}

	providerRequest := notification_provider.Request{
		To:      notification.Recipient,
		Channel: string(notification.Channel),
		Content: notification.Content,
	}
	w.logger.Info("sending provider request",
		"notification_id", notification.ID.String(),
		"batch_id", derefUUID(notification.BatchID),
		"provider_payload", providerRequest,
	)

	result, err := w.provider.Send(ctx, providerRequest)
	if err != nil {
		attemptResult := repository.NotificationAttemptResult{
			ErrorMessage: stringPtr(err.Error()),
		}
		if retryErr := w.handleFailure(ctx, notification, attemptResult, retryDecisionRetryableTransportError); retryErr != nil {
			span.RecordError(retryErr)
			span.SetStatus(codes.Error, retryErr.Error())
			_ = delivery.Nack(false, true)
			return retryErr
		}
		_ = delivery.Ack(false)
		return nil
	}

	attemptResult := repository.NotificationAttemptResult{
		ResponseStatusCode: intPtr(result.StatusCode),
		ResponseBody:       stringPtr(result.ResponseBody),
	}

	switch {
	case result.StatusCode == httpStatusAccepted:
		_, err = w.deliveryRepo.MarkNotificationSent(ctx, notification.ID.String(), attemptResult, result.Response.MessageID, time.Now().UTC())
		if err == nil {
			w.logger.Info("notification sent",
				"notification_id", notification.ID.String(),
				"batch_id", derefUUID(notification.BatchID),
				"channel", notification.Channel,
				"status_code", result.StatusCode,
				"provider_message_id", result.Response.MessageID,
			)
		}
	default:
		attemptResult.ErrorMessage = stringPtr(fmt.Sprintf("provider returned %d", result.StatusCode))
		w.logger.Warn("provider returned non-success status",
			"notification_id", notification.ID.String(),
			"batch_id", derefUUID(notification.BatchID),
			"channel", notification.Channel,
			"status_code", result.StatusCode,
		)
		err = w.handleFailure(ctx, notification, attemptResult, classifyProviderStatus(result.StatusCode))
	}
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		_ = delivery.Nack(false, true)
		return err
	}

	return delivery.Ack(false)
}

const httpStatusAccepted = 202

func stringPtr(value string) *string {
	return &value
}

func intPtr(value int) *int {
	return &value
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}

	return *value
}

func derefUUID(value *uuid.UUID) string {
	if value == nil {
		return ""
	}

	return value.String()
}

func (w *Worker) waitForChannelRateLimit(ctx context.Context, channel string) error {
	w.mu.Lock()
	limiter, ok := w.limiters[channel]
	if !ok {
		limiter = rate.NewLimiter(rate.Limit(100), 100)
		w.limiters[channel] = limiter
	}
	w.mu.Unlock()

	return limiter.Wait(ctx)
}

type retryDecision int

const (
	retryDecisionNonRetryable retryDecision = iota
	retryDecisionRetryableTransportError
	retryDecisionRetryableProviderError
)

func classifyProviderStatus(statusCode int) retryDecision {
	switch {
	case statusCode == httpStatusAccepted:
		return retryDecisionNonRetryable
	case statusCode == 429:
		return retryDecisionRetryableProviderError
	case statusCode >= 500:
		return retryDecisionRetryableProviderError
	default:
		return retryDecisionNonRetryable
	}
}

func (w *Worker) handleFailure(
	ctx context.Context,
	notification *model.Notification,
	attemptResult repository.NotificationAttemptResult,
	decision retryDecision,
) error {
	nextAttemptNumber := notification.AttemptCount + 1
	now := time.Now().UTC()

	if decision == retryDecisionNonRetryable || nextAttemptNumber >= notification.MaxAttempts {
		if attemptResult.ErrorMessage == nil {
			attemptResult.ErrorMessage = stringPtr("non-retryable failure")
		}
		_, err := w.deliveryRepo.MarkNotificationFailed(ctx, notification.ID.String(), attemptResult, now)
		if err == nil {
			w.logger.Warn("notification failed",
				"notification_id", notification.ID.String(),
				"batch_id", derefUUID(notification.BatchID),
				"channel", notification.Channel,
				"attempt_count", nextAttemptNumber,
				"error", derefString(attemptResult.ErrorMessage),
			)
		}
		return err
	}

	nextAttemptAt := now.Add(w.computeRetryDelay(nextAttemptNumber))
	_, err := w.deliveryRepo.ScheduleNotificationRetry(ctx, notification.ID.String(), attemptResult, nextAttemptAt)
	if err == nil {
		w.logger.Warn("notification retry scheduled",
			"notification_id", notification.ID.String(),
			"batch_id", derefUUID(notification.BatchID),
			"channel", notification.Channel,
			"attempt_count", nextAttemptNumber,
			"next_attempt_at", nextAttemptAt.Format(time.RFC3339),
			"error", derefString(attemptResult.ErrorMessage),
		)
	}
	return err
}

func (w *Worker) computeRetryDelay(attemptNumber int) time.Duration {
	if attemptNumber < 1 {
		attemptNumber = 1
	}

	baseDelay := 5 * time.Second
	maxDelay := 5 * time.Minute

	delay := baseDelay * time.Duration(1<<(attemptNumber-1))
	if delay > maxDelay {
		delay = maxDelay
	}

	// Add jitter so retries do not synchronize into the same retry window.
	jitterFactor := 0.5 + w.rng.Float64()
	jittered := time.Duration(float64(delay) * jitterFactor)
	if jittered > maxDelay {
		return maxDelay
	}
	if jittered < time.Second {
		return time.Second
	}

	return jittered
}
