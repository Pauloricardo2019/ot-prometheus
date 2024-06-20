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

// LoggerKey is a unique type to avoid context key collisions
type LoggerKey struct{}

// WithLogger returns a new context with the provided logger.
func WithLogger(ctx context.Context, logger *zap.Logger) context.Context {
	return context.WithValue(ctx, LoggerKey{}, logger)
}

// Logger retrieves the zap.Logger from the context, adding trace and span IDs if available.
func Logger(ctx context.Context) *zap.Logger {
	if ctx == nil {
		panic("nil context passed to logger")
	}

	logger, ok := ctx.Value(LoggerKey{}).(*zap.Logger)
	if !ok {
		panic("no zap logger in context")
	}

	if traceID := trace.SpanFromContext(ctx).SpanContext().TraceID(); traceID.IsValid() {
		logger = logger.With(zap.String(traceIDKey, traceID.String()))
	}

	if spanID := trace.SpanFromContext(ctx).SpanContext().SpanID(); spanID.IsValid() {
		logger = logger.With(zap.String(spanIDKey, spanID.String()))
	}

	return logger
}

// Info logs an info level message using the logger from the context.
func Info(ctx context.Context, msg string, fields ...zapcore.Field) {
	Logger(ctx).Info(msg, fields...)
}

// Error logs an error level message using the logger from the context, including the error.
func Error(ctx context.Context, msg string, err error, fields ...zapcore.Field) {
	Logger(ctx).Error(msg, append(fields, zap.Error(err))...)
}

// NewLogger creates and returns a new zap.Logger instance with ISO8601 time encoding.
func NewLogger() (*zap.Logger, error) {
	logConfig := zap.NewProductionConfig()
	logConfig.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	logger, err := logConfig.Build()
	if err != nil {
		return nil, err
	}

	return logger, nil
}

// LoggerToContextMiddleware adds a zap.Logger to the request context for each incoming request.
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

// LoggerFromContext retrieves the logger from the context, or returns a no-op logger if not found.
func LoggerFromContext(ctx context.Context) *zap.Logger {
	logger, ok := ctx.Value(LoggerKey{}).(*zap.Logger)
	if !ok {
		return zap.NewNop() // Return a no-op logger if no logger is found
	}
	return logger
}
