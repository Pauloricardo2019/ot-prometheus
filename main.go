package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"log"
	"math/rand"
	"net/http"
	"ot-prometheus/telemetry"
	"ot-prometheus/telemetryfs"
	"runtime"
	"strconv"
	"sync"
	"time"
)

var (
	BuildCommit = "undefined"
	BuildTag    = "undefined"
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

	_ = logger.With(
		zap.String("build_commit", BuildCommit),
		zap.String("build_tag", BuildTag),
		zap.String("build_time", BuildTime),
		zap.Int("go_max_procs", runtime.GOMAXPROCS(0)),
		zap.Int("runtime_num_cpu", runtime.NumCPU()),
	)

	ctx := telemetryfs.WithLogger(context.Background(), logger)

	appMetrics := telemetry.NewPrometheusMetrics()

	tracer, err := telemetryfs.NewTracer(ctx, "OTEL", BuildTag)
	if err != nil {
		fmt.Errorf("error creating the tracer: %w", err)
		return
	}

	defer func() {
		if err = tracer.Shutdown(ctx); err != nil {
			logger.Error("error flushing tracer", zap.Error(err))
		}
	}()

	ctx = telemetryfs.WithTracer(ctx, tracer.OTelTracer)
	metricsServer, err := telemetryfs.NewMetricsServer()
	if err != nil {
		fmt.Errorf("creating metrics server: %w", err)
		return
	}

	router := NewServer(logger, tracer.OTelTracer)
	router.Post("/", func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		var status string
		var user string
		defer func() {
			appMetrics.UserStartRequestCounter.WithLabelValues(user, status).Inc()
		}()

		var mr MyRequest
		json.NewDecoder(r.Body).Decode(&mr)

		appMetrics.ActiveRequestGauge.Inc()
		defer appMetrics.ActiveRequestGauge.Dec()

		_, span := tracer.OTelTracer.Start(r.Context(), "handler")
		defer span.End()

		if rand.Float32() > 0.8 {
			status = "4xx"
		} else {
			status = "2xx"
		}

		user = mr.User
		log.Println(user, status)

		appMetrics.RequestCounter.Inc() // Increment the counter

		rand.Seed(time.Now().UnixNano())
		n := rand.Intn(7) + 1

		timeDuration := time.Duration(n) * time.Second

		time.Sleep(timeDuration)

		duration := time.Since(start)

		appMetrics.CreateRequestDuration.WithLabelValues(strconv.Itoa(int(duration.Seconds()))).Observe(duration.Seconds())
		w.Write([]byte(status))
	})

	router.Handle("/metrics", promhttp.Handler()) // Expose the metrics endpoint

	server := http.Server{
		Addr:    ":8989",
		Handler: router,
	}

	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()

		logger.Info("server started",
			zap.String("address", server.Addr),
		)

		if serverErr := server.ListenAndServe(); serverErr != nil && !errors.Is(serverErr, http.ErrServerClosed) {
			logger.Error("failed to listen and serve server", zap.Error(serverErr))
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		logger.Info("metrics server started",
			zap.String("address", metricsServer.Addr),
		)

		if err := metricsServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("failed to listen and serve metric server", zap.Error(err))
		}
	}()

	wg.Add(1)
	go func() {
		producer()
	}()
	wg.Wait()
}

type MyRequest struct {
	User string
}

func producer() {
	userPool := []string{"bob", "alice", "jack"}
	for {
		postBody, _ := json.Marshal(MyRequest{
			User: userPool[rand.Intn(len(userPool))],
		})
		requestBody := bytes.NewBuffer(postBody)
		http.Post("http://api:8989", "application/json", requestBody)
		time.Sleep(time.Second * 2)
	}
}

func NewServer(logger *zap.Logger, tracer trace.Tracer) *chi.Mux {
	redMetricsMiddleware := telemetryfs.NewRedMetricsMiddleware()
	router := chi.NewRouter()
	router.Use(
		telemetryfs.LoggerToContextMiddleware(logger),
		telemetryfs.TracerToContextMiddleware(tracer),
		redMetricsMiddleware.Handle(),
	)

	return router
}
