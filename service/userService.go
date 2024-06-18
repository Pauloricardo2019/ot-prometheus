package service

import (
	"bytes"
	"context"
	"net/http"
	"ot-prometheus/repository"
	"ot-prometheus/telemetry"
	"time"

	"go.opentelemetry.io/otel/trace"
)

type UserService struct {
	UserRepo *repository.UserRepository
	Tracer   trace.Tracer
	Metrics  telemetry.Prometheus
}

func NewUserService(repo *repository.UserRepository, tracer trace.Tracer, metrics telemetry.Prometheus) *UserService {
	return &UserService{
		UserRepo: repo,
		Tracer:   tracer,
		Metrics:  metrics,
	}
}

func (s *UserService) GetUser(ctx context.Context, userID string) (string, error) {
	ctx, span := s.Tracer.Start(ctx, "Service.GetUser")
	defer span.End()

	userData, err := s.UserRepo.FetchUserData(ctx, userID)
	if err != nil {
		return "", err
	}
	time.Sleep(time.Millisecond * 198)
	return userData, nil
}

func (s *UserService) callExternalAPI(ctx context.Context, url string) (string, error) {
	_, span := s.Tracer.Start(ctx, "Service.callExternalAPI")
	defer span.End()

	req, _ := http.NewRequestWithContext(ctx, "POST", url, nil)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	return buf.String(), nil
}
