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

type ProdutoService struct {
	Repository *repository.ProdutoRepository
	Tracer     trace.Tracer
	Metrics    telemetria.Prometheus
}

func NewProdutoService(repo *repository.ProdutoRepository, tracer trace.Tracer, metrics telemetria.Prometheus) *ProdutoService {
	return &ProdutoService{
		Repository: repo,
		Tracer:     tracer,
		Metrics:    metrics,
	}
}

func (s *ProdutoService) GetProduct(ctx context.Context, product string) (string, error) {
	start := time.Now()
	id, _ := strconv.Atoi(product)
	defer func() {
		duration := time.Since(start)
		// logger := telemetria.LoggerFromContext(ctx)
		logger := telemetria.LoggerFromContext(ctx)
		logger.Info("stone GetProductService executed",
			zap.Int("product_id", id),
			zap.Duration("duration", duration),
		)
	}()

	ctx, span := s.Tracer.Start(ctx, "Service.GetProduct")
	defer span.End()

	s.Metrics.API_ActiveRequestGauge.Inc()
	defer s.Metrics.API_ActiveRequestGauge.Dec()

	productData, err := s.Repository.FetchProductData(ctx, product)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return "", err
	}

	return productData, nil
}
