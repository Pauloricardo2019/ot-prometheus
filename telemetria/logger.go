package telemetria

import (
	"context"
	"net/http"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	traceIDKey = "trace_id"
	spanIDKey  = "span_id"
)

type loggerKey struct{}

func WithLogger(ctx context.Context, logger *zap.Logger) context.Context {
	return context.WithValue(ctx, loggerKey{}, logger)
}

func Logger(ctx context.Context) *zap.Logger {
	if ctx == nil {
		panic("nil context passed to logger")
	}

	if logger, _ := ctx.Value(loggerKey{}).(*zap.Logger); logger != nil {
		if traceID := trace.SpanFromContext(ctx).SpanContext().TraceID(); traceID.IsValid() {
			logger = logger.With(zap.String(traceIDKey, traceID.String()))
		}

		if spanID := trace.SpanFromContext(ctx).SpanContext().SpanID(); spanID.IsValid() {
			logger = logger.With(zap.String(spanIDKey, spanID.String()))
		}

		return logger
	}

	panic("no zap logger in context")
}

func Info(ctx context.Context, msg string, fields ...zapcore.Field) {
	Logger(ctx).Info(msg, fields...)
}

func Error(ctx context.Context, msg string, err error, fields ...zapcore.Field) {
	Logger(ctx).Error(msg, append(fields, zap.Error(err))...)
}

func NewLogger() (*zap.Logger, error) {
	logConfig := zap.NewProductionConfig()
	logConfig.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	logger, err := logConfig.Build()
	if err != nil {
		return nil, err
	}

	return logger, nil
}

func LoggerToContextMiddleware(logger *zap.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			r = r.WithContext(WithLogger(r.Context(), logger))
			next.ServeHTTP(w, r)
		}

		return http.HandlerFunc(fn)
	}
}
