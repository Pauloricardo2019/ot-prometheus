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
	"go.uber.org/zap"
)

type UserHandle struct {
	UserService *service.UserService
	Metrics     telemetry.Prometheus
	Tracer      telemetryfs.Tracer
}

func NewUserHandle(service *service.UserService, metrics telemetry.Prometheus, tracer telemetryfs.Tracer) *UserHandle {
	return &UserHandle{
		UserService: service,
		Metrics:     metrics,
		Tracer:      tracer,
	}
}

func (a *UserHandle) GetUser() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ctx := r.Context()

		//Feito no vídeo assim, (SEM INJEÇÃO DE DEPENDÊNCIA)
		tracer := telemetryfs.FromContext(ctx)
		ctx, span := tracer.Start(r.Context(), "Handler.GetUser")
		defer span.End()

		logger := telemetryfs.Logger(ctx)

		var status string
		defer func() {
			a.Metrics.HTTP_StartRequestCounter.WithLabelValues("x_stone_balance_user_api", status).Inc()
		}()

		var mr models.User
		if err := json.NewDecoder(r.Body).Decode(&mr); err != nil {
			logger.Error("error on bind json", zap.Error(err))
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			status = "4xx"
			return
		}

		a.Metrics.API_ActiveRequestGauge.Inc()
		defer a.Metrics.API_ActiveRequestGauge.Dec()

		span.SetAttributes(attribute.String("user", mr.User))

		result, err := a.UserService.GetUser(ctx, mr.User)
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

		a.Metrics.HTTP_RequestCounter.WithLabelValues("x_stone_balance_user_api_increment").Inc() // Increment the counter

		duration := time.Since(start)
		a.Metrics.API_CreateRequestDuration.WithLabelValues("x_stone_balance_user_api_duration", strconv.Itoa(int(duration.Milliseconds()))).Observe(duration.Seconds())

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(result))
	}
}
