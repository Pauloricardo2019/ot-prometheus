package telemetria

import (
	"context"

	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	traceIDKey = "trace_id"
	spanIDKey  = "span_id"
)

type LoggerKey struct{}

func WithLogger(ctx context.Context, logger *zap.Logger) context.Context {
	return context.WithValue(ctx, LoggerKey{}, logger)
}

func Logger(ctx context.Context) *zap.Logger {
	if ctx == nil {
		panic("nil context passed to logger")
	}

	if logger, _ := ctx.Value(LoggerKey{}).(*zap.Logger); logger != nil {
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

func LoggerToContextMiddleware(logger *zap.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			req := c.Request()
			req = req.WithContext(WithLogger(req.Context(), logger))
			c.SetRequest(req)
			return next(c)
		}
	}
}

// LoggerFromContext returns the logger from the context.
func LoggerFromContext(ctx context.Context) *zap.Logger {
	logger, ok := ctx.Value(LoggerKey{}).(*zap.Logger)
	if !ok {
		return zap.NewNop() // Return a no-op logger if no logger is found
	}
	return logger
}
