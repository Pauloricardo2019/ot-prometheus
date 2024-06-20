package telemetria

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid/v5"
	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/contrib/propagators/b3"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdkTrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.8.0"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// TracerConfig represents the configuration for the tracer.
type TracerConfig struct {
	ServiceName      string  `conf:"env:SERVICE_NAME"`
	ServiceNamespace string  `conf:"env:SERVICE_NAMESPACE"`
	Endpoint         string  `conf:"env:COLLECTOR_ENDPOINT"`
	Environment      string  `conf:"env:DEPLOYMENT_ENVIRONMENT"`
	SamplingRatio    float64 `conf:"env:SAMPLING_RATIO"`
}

// Tracer encapsulates the OpenTelemetry tracer and its components.
type Tracer struct {
	OTelTracer     trace.Tracer
	tracerProvider *sdkTrace.TracerProvider
	exporter       *otlptrace.Exporter
}

var tracer trace.Tracer

var (
	SERVICE_NAME           = "myapp"
	SERVICE_NAMESPACE      = "mynamespace"
	COLLECTOR_ENDPOINT     = "0.0.0.0:4317"
	DEPLOYMENT_ENVIRONMENT = "production"
	SAMPLING_RATIO         = 1.0
)

// NewTracer initializes a new OpenTelemetry tracer with the provided context, prefix, and appVersion.
func NewTracer(ctx context.Context, prefix, appVersion string) (Tracer, error) {
	cfg := TracerConfig{
		ServiceName:      SERVICE_NAME,
		ServiceNamespace: SERVICE_NAMESPACE,
		Endpoint:         COLLECTOR_ENDPOINT,
		Environment:      DEPLOYMENT_ENVIRONMENT,
		SamplingRatio:    SAMPLING_RATIO,
	}

	client := otlptracegrpc.NewClient(
		otlptracegrpc.WithEndpoint(cfg.Endpoint),
		otlptracegrpc.WithInsecure(),
	)

	exporter, err := otlptrace.New(ctx, client)
	if err != nil {
		return Tracer{}, fmt.Errorf("creating exporter: %w", err)
	}

	tracerProvider := sdkTrace.NewTracerProvider(
		sdkTrace.WithSampler(sdkTrace.TraceIDRatioBased(cfg.SamplingRatio)),
		sdkTrace.WithBatcher(exporter),
		sdkTrace.WithResource(newResource(cfg, appVersion)),
	)

	// set global tracer provider & text propagators
	otel.SetTracerProvider(tracerProvider)
	otel.SetTextMapPropagator(b3.New(b3.WithInjectEncoding(b3.B3MultipleHeader)))

	return Tracer{
		OTelTracer:     otel.Tracer(cfg.ServiceName),
		tracerProvider: tracerProvider,
		exporter:       exporter,
	}, nil
}

// newResource creates a new resource for the tracer based on the given configuration and app version.
func newResource(cfg TracerConfig, appVersion string) *resource.Resource {
	return resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceNamespaceKey.String(cfg.ServiceNamespace),
		semconv.ServiceNameKey.String(cfg.ServiceName),
		semconv.DeploymentEnvironmentKey.String(cfg.Environment),
		semconv.ServiceInstanceIDKey.String(uuid.Must(uuid.NewV4()).String()),
		semconv.ServiceVersionKey.String(appVersion),
	)
}

// Shutdown shuts down the tracer provider and exporter.
func (t Tracer) Shutdown(ctx context.Context) error {
	var err error

	if shutdownErr := t.tracerProvider.Shutdown(ctx); shutdownErr != nil {
		err = fmt.Errorf("shutting down otel tracer provider: %w", shutdownErr)
	}

	if shutdownErr := t.exporter.Shutdown(ctx); shutdownErr != nil {
		err = fmt.Errorf("shutting down otel tracer exporter: %w", shutdownErr)
	}

	return err
}

// FromContext returns the OpenTelemetry tracer associated with the given context.
func FromContext(ctx context.Context) trace.Tracer {
	if ctx == nil {
		panic("nil context passed to tracer")
	}

	t, ok := ctx.Value(ctxKey{}).(trace.Tracer)
	if !ok {
		panic("no otel tracer in context")
	}

	return t
}

// Start starts a new span with the given name and attributes.
func Start(ctx context.Context, spanName string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	return FromContext(ctx).Start(ctx, spanName, trace.WithAttributes(attrs...))
}

// WithTracer returns a new context derived from parent that is associated with the given tracer.
func WithTracer(parent context.Context, t trace.Tracer) context.Context {
	return context.WithValue(parent, ctxKey{}, t)
}

// HandleUnexpectedError records an error in the current span and logs it.
func HandleUnexpectedError(ctx context.Context, err error, fields ...zap.Field) {
	trace.SpanFromContext(ctx).RecordError(err)
	Logger(ctx).With(append(fields, zap.Error(err))...).Error("unexpected error")
}

// TracerToContextMiddleware associates a tracer with the current context.
func TracerToContextMiddleware(tracer trace.Tracer) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			req := c.Request()
			req = req.WithContext(WithTracer(req.Context(), tracer))
			c.SetRequest(req)
			return next(c)
		}
	}
}

// ctxKey is the type used for the context key for storing the tracer.
type ctxKey struct{}
