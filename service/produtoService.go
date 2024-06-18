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

type ProdutoService struct {
	ProdutoRepo *repository.ProdutoRepository
	Tracer      trace.Tracer
	Metrics     telemetry.Prometheus
}

func NewProdutoService(repo *repository.ProdutoRepository, tracer trace.Tracer, metrics telemetry.Prometheus) *ProdutoService {
	return &ProdutoService{
		ProdutoRepo: repo,
		Tracer:      tracer,
		Metrics:     metrics,
	}
}

func (s *ProdutoService) GetProduct(ctx context.Context, productID string) (string, error) {
	ctx, span := s.Tracer.Start(ctx, "Service.GetProduct")
	defer span.End()

	productData, err := s.ProdutoRepo.FetchProductData(ctx, productID)
	if err != nil {
		return "", err
	}
	time.Sleep(time.Millisecond * 300)
	return productData, nil
}

func (s *ProdutoService) callExternalAPI(ctx context.Context, url string) (string, error) {
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
