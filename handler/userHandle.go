package handler

import (
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"ot-prometheus/models"
	"ot-prometheus/service"
	"ot-prometheus/telemetry"
	"ot-prometheus/telemetryfs"
	"strconv"
	"time"

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

func (h *UserHandle) GetUser() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ctx := r.Context()
		ctx, span := h.Tracer.Start(ctx, "Handler.GetUser")
		defer span.End()

		logger := telemetryfs.Logger(ctx)

		var status string
		defer func() {
			h.Metrics.HTTP_StartRequestCounter.WithLabelValues("x_stone_balance_user_api", status).Inc()
		}()

		var mr models.User
		if err := json.NewDecoder(r.Body).Decode(&mr); err != nil {
			logger.Error("error on bind json", zap.Error(err))
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			status = "4xx"
			return
		}

		h.Metrics.API_ActiveRequestGauge.Inc()
		defer h.Metrics.API_ActiveRequestGauge.Dec()

		span.SetAttributes(attribute.String("user", mr.User))

		result, err := h.Service.GetUser(ctx, mr.User)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			status = strconv.Itoa(http.StatusInternalServerError)
			return
		}

		logger.Info("user data", zap.String("user", mr.User), zap.String("data", result))

		if rand.Float32() > 0.8 {
			status = "4xx"
		} else {
			status = "2xx"
		}

		log.Println(result, status)

		h.Metrics.HTTP_RequestCounter.WithLabelValues("x_stone_balance_user_api_increment").Inc()

		duration := time.Since(start)
		h.Metrics.API_CreateRequestDuration.WithLabelValues("x_stone_balance_user_api_duration", strconv.Itoa(int(duration.Milliseconds()))).Observe(duration.Seconds())

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(result))
	}
}
