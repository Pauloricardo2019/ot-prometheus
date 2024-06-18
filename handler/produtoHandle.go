package handler

import (
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"ot-prometheus/models"
	"ot-prometheus/service"
	"ot-prometheus/telemetry"
	"ot-prometheus/telemetryfs"
	"strconv"
	"time"
)

type ProdutoHandle struct {
	Service *service.ProdutoService
	Metrics telemetry.Prometheus
	Tracer  telemetryfs.Tracer
}

func NewProdutoHandle(service *service.ProdutoService, metrics telemetry.Prometheus, tracer telemetryfs.Tracer) *ProdutoHandle {
	return &ProdutoHandle{
		Service: service,
		Metrics: metrics,
		Tracer:  tracer,
	}
}

func (a *ProdutoHandle) GetProduct() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		ctx := r.Context()

		ctx, span := a.Tracer.OTelTracer.Start(r.Context(), "Handler.GetProduct")
		defer span.End()

		var status string
		defer func() {
			a.Metrics.HTTP_StartRequestCounter.WithLabelValues("x_stone_balance_product_api", status).Inc()
		}()

		mr := models.Product{}
		if err := json.NewDecoder(r.Body).Decode(&mr); err != nil {
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			status = "4xx"
			return
		}

		a.Metrics.API_ActiveRequestGauge.Inc()
		defer a.Metrics.API_ActiveRequestGauge.Dec()

		result, err := a.Service.GetProduct(ctx, mr.Product)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			status = "5xx"
			return
		}

		if rand.Float32() > 0.8 {
			status = "4xx"
		} else {
			status = "2xx"
		}
		log.Println(result, status)

		a.Metrics.HTTP_RequestCounter.WithLabelValues("x_stone_balance_product_api_increment").Inc() // Increment the counter

		duration := time.Since(start)
		a.Metrics.API_CreateRequestDuration.WithLabelValues("x_stone_balance_product_api_duration", strconv.Itoa(int(duration.Milliseconds()))).Observe(duration.Seconds())

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(result))
	}
}
