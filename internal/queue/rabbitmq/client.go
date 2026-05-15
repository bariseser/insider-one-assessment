package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"

	"insider-one-assessment/internal/tracing"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

const (
	ExchangeName = "notifications"
	QueueName    = "notifications.dispatch"
)

type DeliveryMessage struct {
	NotificationID string            `json:"notification_id"`
	BatchID        string            `json:"batch_id"`
	Channel        string            `json:"channel"`
	Priority       string            `json:"priority"`
	TraceCarrier   map[string]string `json:"trace_carrier,omitempty"`
}

type IClient interface {
	Publish(ctx context.Context, routingKey string, message DeliveryMessage) error
	Consume() (<-chan amqp.Delivery, error)
	Close() error
}

type client struct {
	connection *amqp.Connection
	channel    *amqp.Channel
}

var tracer = otel.Tracer("insider-one-assessment/rabbitmq")

func NewClient(amqpURL string) (IClient, error) {
	connection, err := amqp.Dial(amqpURL)
	if err != nil {
		return nil, fmt.Errorf("dial rabbitmq: %w", err)
	}

	channel, err := connection.Channel()
	if err != nil {
		_ = connection.Close()
		return nil, fmt.Errorf("open rabbitmq channel: %w", err)
	}

	if err := channel.ExchangeDeclare(ExchangeName, "direct", true, false, false, false, nil); err != nil {
		_ = channel.Close()
		_ = connection.Close()
		return nil, fmt.Errorf("declare exchange: %w", err)
	}

	queue, err := channel.QueueDeclare(QueueName, true, false, false, false, nil)
	if err != nil {
		_ = channel.Close()
		_ = connection.Close()
		return nil, fmt.Errorf("declare queue: %w", err)
	}

	for _, routingKey := range []string{"notification.sms", "notification.email", "notification.push"} {
		if err := channel.QueueBind(queue.Name, routingKey, ExchangeName, false, nil); err != nil {
			_ = channel.Close()
			_ = connection.Close()
			return nil, fmt.Errorf("bind queue to %s: %w", routingKey, err)
		}
	}

	if err := channel.Qos(100, 0, false); err != nil {
		_ = channel.Close()
		_ = connection.Close()
		return nil, fmt.Errorf("configure qos: %w", err)
	}

	return &client{
		connection: connection,
		channel:    channel,
	}, nil
}

func (c *client) Publish(ctx context.Context, routingKey string, message DeliveryMessage) error {
	ctx, span := tracer.Start(ctx, "rabbitmq.publish")
	defer span.End()

	span.SetAttributes(
		attribute.String("messaging.system", "rabbitmq"),
		attribute.String("messaging.destination.name", ExchangeName),
		attribute.String("messaging.rabbitmq.destination.routing_key", routingKey),
		attribute.String("notification.id", message.NotificationID),
		attribute.String("notification.channel", message.Channel),
	)

	body, err := json.Marshal(message)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("marshal delivery message: %w", err)
	}

	err = c.channel.PublishWithContext(ctx, ExchangeName, routingKey, false, false, amqp.Publishing{
		ContentType:  "application/json",
		Body:         body,
		DeliveryMode: amqp.Persistent,
		Headers:      tracing.AMQPHeadersFromCarrier(message.TraceCarrier),
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("publish delivery message: %w", err)
	}

	return nil
}

func (c *client) Consume() (<-chan amqp.Delivery, error) {
	deliveries, err := c.channel.Consume(QueueName, "", false, false, false, false, nil)
	if err != nil {
		return nil, fmt.Errorf("consume queue: %w", err)
	}

	return deliveries, nil
}

func (c *client) Close() error {
	if err := c.channel.Close(); err != nil {
		_ = c.connection.Close()
		return err
	}

	return c.connection.Close()
}
