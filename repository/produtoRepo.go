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

type ProdutoRepository struct {
	Tracer trace.Tracer
}

func NewProdutoRepository(tracer trace.Tracer) *ProdutoRepository {
	return &ProdutoRepository{
		Tracer: tracer,
	}
}

func (r *ProdutoRepository) FetchProductData(ctx context.Context, productID string) (string, error) {
	start := time.Now()
	id, _ := strconv.Atoi(productID)
	defer func() {
		duration := time.Since(start)
		logger := telemetria.LoggerFromContext(ctx)
		logger.Info("stone FetchProductData executed",
			zap.Int("product_id", id),
			zap.Duration("duration", duration),
		)
	}()
	// Simulating fetching product data from a database or external service
	// Here you can add your implementation to fetch real product data
	time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)
	return fmt.Sprintf("Product data for ID: %s", productID), nil
}
