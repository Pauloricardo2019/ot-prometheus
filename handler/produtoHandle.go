package handler

import (
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"ot-prometheus/models"
	"ot-prometheus/service"
	"ot-prometheus/telemetria"

	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/otel/trace"
)

type ProdutoHandle struct {
	Service *service.ProdutoService
	Metrics telemetria.Prometheus
	Tracer  trace.Tracer
}

func NewProdutoHandle(service *service.ProdutoService, metrics telemetria.Prometheus, tracer trace.Tracer) *ProdutoHandle {
	return &ProdutoHandle{
		Service: service,
		Metrics: metrics,
		Tracer:  tracer,
	}
}

func (h *ProdutoHandle) GetProduct(c echo.Context) error {
	start := time.Now()

	ctx := c.Request().Context()
	ctx, span := h.Tracer.Start(ctx, "Handler.GetProduct")
	defer span.End()

	var status string
	defer func() {
		h.Metrics.HTTP_StartRequestCounter.WithLabelValues(telemetria.LABEL_PREFIXO+"product_api", status).Inc()
	}()

	mr := models.Product{}
	if err := c.Bind(&mr); err != nil {
		status = "4xx"
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request payload")
	}

	h.Metrics.API_ActiveRequestGauge.Inc()
	defer h.Metrics.API_ActiveRequestGauge.Dec()

	result, err := h.Service.GetProduct(ctx, mr.Product)
	if err != nil {
		status = "5xx"
		return echo.NewHTTPError(http.StatusInternalServerError, "Internal Server Error")
	}

	if rand.Float32() > 0.8 {
		status = "4xx"
	} else {
		status = "2xx"
	}

	h.Metrics.HTTP_RequestCounter.WithLabelValues(telemetria.LABEL_PREFIXO + "product_api_increment").Inc()

	duration := time.Since(start)
	h.Metrics.API_CreateRequestDuration.WithLabelValues(telemetria.LABEL_PREFIXO+"product_api_duration", strconv.Itoa(int(duration.Milliseconds()))).Observe(duration.Seconds())

	return c.String(http.StatusOK, result)
}
