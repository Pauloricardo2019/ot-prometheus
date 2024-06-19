package handler

import (
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"ot-prometheus/models"
	"ot-prometheus/service"
	"ot-prometheus/telemetry"
	"ot-prometheus/telemetryfs"

	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type UserHandle struct {
	Service *service.UserService
	Metrics telemetry.Prometheus
	Tracer  trace.Tracer
}

func NewUserHandle(service *service.UserService, metrics telemetry.Prometheus, tracer trace.Tracer) *UserHandle {
	return &UserHandle{
		Service: service,
		Metrics: metrics,
		Tracer:  tracer,
	}
}

func (h *UserHandle) GetUser(c echo.Context) error {
	start := time.Now()

	ctx := c.Request().Context()
	ctx, span := h.Tracer.Start(ctx, "Handler.GetUser")
	defer span.End()

	logger := telemetryfs.Logger(ctx)

	var status string
	defer func() {
		h.Metrics.HTTP_StartRequestCounter.WithLabelValues("x_stone_balance_user_api", status).Inc()
	}()

	var mr models.User
	if err := c.Bind(&mr); err != nil {
		logger.Error("error on bind json", zap.Error(err))
		status = "4xx"
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request payload")
	}

	h.Metrics.API_ActiveRequestGauge.Inc()
	defer h.Metrics.API_ActiveRequestGauge.Dec()

	span.SetAttributes(attribute.String("user", mr.User))

	result, err := h.Service.GetUser(ctx, mr.User)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.Error("failed to get user", zap.Error(err))
		status = strconv.Itoa(http.StatusInternalServerError)
		return echo.NewHTTPError(http.StatusInternalServerError, "Internal Server Error")
	}

	logger.Info("user data", zap.String("user", mr.User), zap.String("data", result))

	if rand.Float32() > 0.8 {
		status = "4xx"
	} else {
		status = "2xx"
	}

	h.Metrics.HTTP_RequestCounter.WithLabelValues("x_stone_balance_user_api_increment").Inc()

	duration := time.Since(start)
	h.Metrics.API_CreateRequestDuration.WithLabelValues("x_stone_balance_user_api_duration", strconv.Itoa(int(duration.Milliseconds()))).Observe(duration.Seconds())

	return c.String(http.StatusOK, result)
}
