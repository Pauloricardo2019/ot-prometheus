package service

import (
	"context"
	"ot-prometheus/repository"
	"ot-prometheus/telemetria"
	"strconv"
	"time"

	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type UserService struct {
	UserRepo *repository.UserRepository
	Tracer   trace.Tracer
	Metrics  telemetria.Prometheus
}

func NewUserService(repo *repository.UserRepository, tracer trace.Tracer, metrics telemetria.Prometheus) *UserService {
	return &UserService{
		UserRepo: repo,
		Tracer:   tracer,
		Metrics:  metrics,
	}
}

func (s *UserService) GetUser(ctx context.Context, userID string) (string, error) {
	start := time.Now()
	id, _ := strconv.Atoi(userID)
	defer func() {
		duration := time.Since(start)
		// logger := telemetria.LoggerFromContext(ctx)
		logger := telemetria.LoggerFromContext(ctx)
		logger.Info("stone GetUser executed",
			zap.Int("user_id", id),
			zap.Duration("duration", duration),
		)
	}()
	ctx, span := s.Tracer.Start(ctx, "Service.GetUser")
	defer span.End()

	s.Metrics.API_ActiveRequestGauge.Inc()
	defer s.Metrics.API_ActiveRequestGauge.Dec()

	userData, err := s.UserRepo.FetchUserData(ctx, userID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return "", err
	}

	return userData, nil
}
