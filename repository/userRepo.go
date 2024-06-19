package repository

import (
	"context"
	"fmt"
	"math/rand"
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

func (r *UserRepository) FetchUserData(ctx context.Context, userID string) (string, error) {
	// Simulating fetching user data from a database or external service
	// Here you can add your implementation to fetch real user data
	time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)
	return fmt.Sprintf("User data for ID: %s", userID), nil
}
