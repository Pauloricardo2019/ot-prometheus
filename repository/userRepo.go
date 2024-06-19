package repository

import (
	"context"
	"fmt"
	"math/rand"
	"ot-prometheus/telemetria"
	"strconv"
	"time"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
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
	start := time.Now()
	id, _ := strconv.Atoi(userID)
	defer func() {
		duration := time.Since(start)
		// logger := telemetria.LoggerFromContext(ctx)
		logger := telemetria.LoggerFromContext(ctx)
		logger.Info("stone FetchUserData executed",
			zap.Int("userID", id),
			zap.Duration("duration", duration),
		)
	}()
	// Simulating fetching user data from a database or external service
	// Here you can add your implementation to fetch real user data
	time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)
	return fmt.Sprintf("User data for ID: %s", userID), nil
}
