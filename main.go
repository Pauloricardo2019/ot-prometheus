package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"runtime"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"

	"ot-prometheus/handler"
	"ot-prometheus/repository"
	"ot-prometheus/service"
	"ot-prometheus/telemetria"
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
			req = req.WithContext(telemetria.WithLogger(req.Context(), logger))
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
			req = req.WithContext(telemetria.WithTracer(req.Context(), tracer))
			c.SetRequest(req)
			return next(c)
		}
	}
}

func main() {
	logger, err := telemetria.NewLogger()
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

	ctx := telemetria.WithLogger(context.Background(), logger)

	metrics := telemetria.NewPrometheusMetrics()
	tracer, err := telemetria.NewTracer(ctx, "OTEL", BuildTag)
	if err != nil {
		logger.Error("error creating the tracer", zap.Error(err))
		return
	}

	defer func() {
		if err = tracer.Shutdown(ctx); err != nil {
			logger.Error("error flushing tracer", zap.Error(err))
		}
	}()

	ctx = telemetria.WithTracer(ctx, tracer.OTelTracer)

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

	telemetria.InitMetricsCollector(metrics)

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
