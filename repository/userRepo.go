package repository

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/trace"
)

type UserRepository struct {
	Tracer trace.Tracer
}

func NewUserRepository(tracer trace.Tracer) *UserRepository {
	return &UserRepository{
		Tracer: tracer,
	}
}

func (repo *UserRepository) FetchUserData(ctx context.Context, userID string) (string, error) {
	ctx, span := repo.Tracer.Start(ctx, "UserRepository.FetchUserData")
	defer span.End()

	// Simulando uma busca no banco de dados
	time.Sleep(150 * time.Millisecond)
	return "UserData for " + userID, nil
}
