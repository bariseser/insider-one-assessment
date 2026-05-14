package tracing

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func Init(ctx context.Context, serviceName, endpoint string) (func(context.Context) error, error) {
	otel.SetTextMapPropagator(propagation.TraceContext{})

	if endpoint == "" {
		return func(context.Context) error { return nil }, nil
	}

	exporter, err := otlptracehttp.New(
		ctx,
		otlptracehttp.WithEndpoint(strings.TrimPrefix(strings.TrimPrefix(endpoint, "http://"), "https://")),
		otlptracehttp.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("create otlp trace exporter: %w", err)
	}

	res, err := resource.New(
		ctx,
		resource.WithAttributes(attribute.String("service.name", serviceName)),
	)
	if err != nil {
		return nil, fmt.Errorf("build otel resource: %w", err)
	}

	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	otel.SetTracerProvider(provider)

	return provider.Shutdown, nil
}

func HTTPTransport(base httpRoundTripper) *otelhttp.Transport {
	return otelhttp.NewTransport(base)
}

type httpRoundTripper interface {
	RoundTrip(*http.Request) (*http.Response, error)
}

func InjectCarrier(ctx context.Context) map[string]string {
	carrier := propagation.MapCarrier{}
	otel.GetTextMapPropagator().Inject(ctx, carrier)
	if len(carrier) == 0 {
		return nil
	}

	out := make(map[string]string, len(carrier))
	for key, value := range carrier {
		out[key] = value
	}
	return out
}

func ContextWithCarrier(ctx context.Context, carrier map[string]string) context.Context {
	if len(carrier) == 0 {
		return ctx
	}

	return otel.GetTextMapPropagator().Extract(ctx, propagation.MapCarrier(carrier))
}

func AMQPHeadersFromCarrier(carrier map[string]string) amqp.Table {
	if len(carrier) == 0 {
		return nil
	}

	headers := amqp.Table{}
	for key, value := range carrier {
		headers[key] = value
	}
	return headers
}

func ContextFromAMQPHeaders(ctx context.Context, headers amqp.Table) context.Context {
	if len(headers) == 0 {
		return ctx
	}

	carrier := propagation.MapCarrier{}
	for key, value := range headers {
		if text, ok := value.(string); ok {
			carrier[key] = text
		}
	}

	if len(carrier) == 0 {
		return ctx
	}

	return otel.GetTextMapPropagator().Extract(ctx, carrier)
}
