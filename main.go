package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"runtime"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"ot-prometheus/handler"
	"ot-prometheus/repository"
	"ot-prometheus/service"
	"ot-prometheus/telemetry"
	"ot-prometheus/telemetryfs"
)

var (
	BuildCommit = "undefined"
	BuildTag    = "1.0.0"
	BuildTime   = "undefined"
)

func main() {
	logger, err := telemetryfs.NewLogger()
	if err != nil {
		panic(fmt.Errorf("creating logger: %w", err))
	}

	defer func() {
		_ = logger.Sync()
	}()

	logger = logger.With(
		zap.String("build_commit", BuildCommit),
		zap.String("build_tag", BuildTag),
		zap.String("build_time", BuildTime),
		zap.Int("go_max_procs", runtime.GOMAXPROCS(0)),
		zap.Int("runtime_num_cpu", runtime.NumCPU()),
	)

	ctx := telemetryfs.WithLogger(context.Background(), logger)

	metrics := telemetry.NewPrometheusMetrics()
	tracer, err := telemetryfs.NewTracer(ctx, "OTEL", BuildTag)
	if err != nil {
		logger.Error("error creating the tracer", zap.Error(err))
		return
	}

	defer func() {
		if err = tracer.Shutdown(ctx); err != nil {
			logger.Error("error flushing tracer", zap.Error(err))
		}
	}()

	ctx = telemetryfs.WithTracer(ctx, tracer.OTelTracer)

	productRepo := repository.NewProdutoRepository(tracer.OTelTracer)
	productService := service.NewProdutoService(productRepo, tracer.OTelTracer, metrics)
	productHandle := handler.NewProdutoHandle(productService, metrics, tracer.OTelTracer)

	userRepo := repository.NewUserRepository(tracer.OTelTracer)
	userService := service.NewUserService(userRepo, tracer.OTelTracer, metrics)
	userHandle := handler.NewUserHandle(userService, metrics, tracer.OTelTracer)

	router := NewServer(logger, tracer.OTelTracer)
	router.Use(ZapMiddleware(logger))
	router.Use(TracerMiddleware(tracer.OTelTracer))
	router.Post("/user", userHandle.GetUser())
	router.Post("/product", productHandle.GetProduct())
	router.Handle("/metrics", promhttp.Handler())

	server := http.Server{
		Addr:    ":8989",
		Handler: router,
	}

	initMetricsCollector(metrics)

	go func() {
		logger.Info("server started",
			zap.String("address", server.Addr),
		)

		if serverErr := server.ListenAndServe(); serverErr != nil && !errors.Is(serverErr, http.ErrServerClosed) {
			logger.Error("failed to listen and serve server", zap.Error(serverErr))
		}
	}()

	go func() {
		logger.Info("metrics server started",
			zap.String("address", ":8080"),
		)

		metricsServer := &http.Server{
			Addr:    ":8080",
			Handler: promhttp.Handler(),
		}

		if err := metricsServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("failed to listen and serve metric server", zap.Error(err))
		}
	}()

	select {}
}

func NewServer(logger *zap.Logger, otTracer trace.Tracer) chi.Router {
	r := chi.NewRouter()

	r.Use(ZapMiddleware(logger))
	r.Use(TracerMiddleware(otTracer))

	return r
}

func ZapMiddleware(logger *zap.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			ctx = telemetryfs.WithLogger(ctx, logger)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func TracerMiddleware(otTracer trace.Tracer) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			ctx = telemetryfs.WithTracer(ctx, otTracer)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func initMetricsCollector(metrics telemetry.Prometheus) {
	metrics.HTTP_RequestCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_request_total",
			Help: "Total number of HTTP requests made.",
		},
		[]string{"handler", "status"},
	)
	prometheus.MustRegister(metrics.HTTP_RequestCounter)

	metrics.HTTP_StartRequestCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_start_request_total",
			Help: "Total number of HTTP start requests made.",
		},
		[]string{"handler", "status"},
	)
	prometheus.MustRegister(metrics.HTTP_StartRequestCounter)

	metrics.API_ActiveRequestGauge = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "api_active_requests",
			Help: "Number of active API requests.",
		},
	)
	prometheus.MustRegister(metrics.API_ActiveRequestGauge)

	metrics.API_CreateRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "api_request_duration_seconds",
			Help: "Duration of API requests in seconds.",
			Buckets: []float64{
				0.1, 0.3, 1.2, 5.0,
			},
		},
		[]string{"handler", "duration"},
	)
	prometheus.MustRegister(metrics.API_CreateRequestDuration)
}
