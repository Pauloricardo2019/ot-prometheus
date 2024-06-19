package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"runtime"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"

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

// LoggerToContextMiddleware associates a logger with the request context.
func LoggerToContextMiddleware(logger *zap.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			req := c.Request()
			req = req.WithContext(telemetryfs.WithLogger(req.Context(), logger))
			c.SetRequest(req)
			return next(c)
		}
	}
}

// TracerToContextMiddleware associates a tracer with the request context.
func TracerToContextMiddleware(tracer trace.Tracer) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			req := c.Request()
			req = req.WithContext(telemetryfs.WithTracer(req.Context(), tracer))
			c.SetRequest(req)
			return next(c)
		}
	}
}

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

	e := echo.New()
	//e.Logger = telemetryfs.EchoLogger(logger)
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(LoggerToContextMiddleware(logger))
	e.Use(TracerToContextMiddleware(tracer.OTelTracer))

	e.POST("/user", userHandle.GetUser)
	e.POST("/product", productHandle.GetProduct)
	e.GET("/metrics", echo.WrapHandler(promhttp.Handler()))

	initMetricsCollector(metrics)

	go func() {
		logger.Info("server started",
			zap.String("address", ":8989"),
		)

		if err := e.Start(":8989"); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("failed to listen and serve server", zap.Error(err))
		}
	}()

	go func() {
		logger.Info("metrics server started",
			zap.String("address", ":8080"),
		)

		metricsServer := echo.New()
		metricsServer.HideBanner = true
		metricsServer.GET("/metrics", echo.WrapHandler(promhttp.Handler()))

		if err := metricsServer.Start(":8080"); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("failed to listen and serve metric server", zap.Error(err))
		}
	}()

	select {}
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
