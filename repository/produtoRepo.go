package repository

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/trace"
)

type ProdutoRepository struct {
	Tracer trace.Tracer
}

func NewProdutoRepository(tracer trace.Tracer) *ProdutoRepository {
	return &ProdutoRepository{
		Tracer: tracer,
	}
}

func (repo *ProdutoRepository) FetchUserData(ctx context.Context, userID string) (string, error) {
	ctx, span := repo.Tracer.Start(ctx, "Repository.FetchUserData")
	defer span.End()

	// Simulando uma busca no banco de dados
	time.Sleep(90 * time.Millisecond)
	return "UserData for " + userID, nil
}

func (repo *ProdutoRepository) FetchProductData(ctx context.Context, productID string) (string, error) {
	ctx, span := repo.Tracer.Start(ctx, "Repository.FetchProductData")
	defer span.End()

	// Simulando uma busca no banco de dados
	time.Sleep(200 * time.Millisecond)
	return "ProductData for " + productID, nil
}
