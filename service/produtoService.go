package service

import (
	"context"
	"ot-prometheus/repository"
	"ot-prometheus/telemetry"

	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type ProdutoService struct {
	Repository *repository.ProdutoRepository
	Tracer     trace.Tracer
	Metrics    telemetry.Prometheus
}

func NewProdutoService(repo *repository.ProdutoRepository, tracer trace.Tracer, metrics telemetry.Prometheus) *ProdutoService {
	return &ProdutoService{
		Repository: repo,
		Tracer:     tracer,
		Metrics:    metrics,
	}
}

func (s *ProdutoService) GetProduct(ctx context.Context, product string) (string, error) {
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
