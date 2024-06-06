package telemetryfs

import (
	"context"
	"fmt"
	"github.com/gofrs/uuid/v5"
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
	"net/http"
)

var tracer trace.Tracer

var (
	SERVICE_NAME           = "myapp"
	SERVICE_NAMESPACE      = "mynamespace"
	COLLECTOR_ENDPOINT     = "172.28.0.4:4317"
	DEPLOYMENT_ENVIRONMENT = "production"
	SAMPLING_RATIO         = 1.0
)

// TracerConfig represents the configuration for the tracer.
type TracerConfig struct {
	ServiceName      string `conf:"env:SERVICE_NAME"`
	ServiceNamespace string `conf:"env:SERVICE_NAMESPACE"`
	Endpoint         string `conf:"env:COLLECTOR_ENDPOINT"`
	Environment      string `conf:"env:DEPLOYMENT_ENVIRONMENT"`
	// The ratio of samples sent by TraceID. See more on TraceIDRatioBased.
	// NOTE: The sampling in production is always 1% (100:1). So just values lesser than 1% will make an effect.
	SamplingRatio float64 `conf:"env:SAMPLING_RATIO"`
}

type Tracer struct {
	OTelTracer trace.Tracer

	tracerProvider *sdkTrace.TracerProvider
	exporter       *otlptrace.Exporter
}

// Shutdown shuts down the tracer and exporter.
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

func NewTracer(ctx context.Context, prefix, appVersion string) (Tracer, error) {
	var cfg TracerConfig

	cfg.ServiceName = SERVICE_NAME
	cfg.ServiceNamespace = SERVICE_NAMESPACE
	cfg.Endpoint = COLLECTOR_ENDPOINT
	cfg.Environment = DEPLOYMENT_ENVIRONMENT
	cfg.SamplingRatio = SAMPLING_RATIO

	client := otlptracegrpc.NewClient(otlptracegrpc.WithEndpoint(cfg.Endpoint), otlptracegrpc.WithInsecure())

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

func newResource(cfg TracerConfig, appVersion string) *resource.Resource {
	return resource.NewWithAttributes(
		semconv.SchemaURL,
		// the service name used to display traces in backends
		semconv.ServiceNamespaceKey.String(cfg.ServiceNamespace),
		semconv.ServiceNameKey.String(cfg.ServiceName),
		semconv.DeploymentEnvironmentKey.String(cfg.Environment),
		semconv.ServiceInstanceIDKey.String(uuid.Must(uuid.NewV4()).String()),
		semconv.ServiceVersionKey.String(appVersion),
	)
}

type ctxKey struct{}

// FromContext returns the OTel Tracer associated with the given context.
// If there is no tracer, it will panic.
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

func Start(ctx context.Context, spanName string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	return FromContext(ctx).Start(ctx, spanName, trace.WithAttributes(attrs...))
}

// WithTracer returns a new context derived from ctx that
// is associated with the given tracer.
func WithTracer(parent context.Context, t trace.Tracer) context.Context {
	return context.WithValue(parent, ctxKey{}, t)
}

// HandleUnexpectedError adds the information regarding the error on the current span and logs.
func HandleUnexpectedError(ctx context.Context, err error, fields ...zap.Field) {
	trace.SpanFromContext(ctx).RecordError(err)
	Logger(ctx).With(append(fields, zap.Error(err))...).Error("unexpected error")
}

// TracerToContextMiddleware associates a tracer with the current context.
func TracerToContextMiddleware(tracer trace.Tracer) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			r = r.WithContext(WithTracer(r.Context(), tracer))
			next.ServeHTTP(w, r)
		}

		return http.HandlerFunc(fn)
	}
}
