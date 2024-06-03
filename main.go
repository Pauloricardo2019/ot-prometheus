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

type ApiRest struct {
	Metrics telemetry.Prometheus
	Tracer  telemetryfs.Tracer
}

func NewApiRest(metrics telemetry.Prometheus, tracer telemetryfs.Tracer) *ApiRest {
	return &ApiRest{
		Metrics: metrics,
		Tracer:  tracer,
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
	apiRest := NewApiRest(appMetrics, tracer)

	router.Post("/user", apiRest.GetUser())
	router.Post("/product", apiRest.GetProduct())
	router.Handle("/metrics", promhttp.Handler())

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

	wg.Add(2)
	go func() {
		producerUser()
	}()
	go func() {
		producerProduct()
	}()
	wg.Wait()
}

func (a *ApiRest) GetUser() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		_, span := a.Tracer.OTelTracer.Start(r.Context(), "GetUser")
		defer span.End()

		var status string
		var user string
		defer func() {
			a.Metrics.UserStartRequestCounter.WithLabelValues(user, status).Inc()
		}()

		var mr User
		if err := json.NewDecoder(r.Body).Decode(&mr); err != nil {
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			status = "4xx"
			return
		}

		a.Metrics.ActiveRequestGauge.Inc()
		defer a.Metrics.ActiveRequestGauge.Dec()

		if rand.Float32() > 0.8 {
			status = "4xx"
		} else {
			status = "2xx"
		}

		user = mr.User
		log.Println(user, status)

		a.Metrics.RequestCounter.WithLabelValues("GetUser").Inc() // Increment the counter

		rand.Seed(time.Now().UnixNano())
		n := rand.Intn(7) + 1

		timeDuration := time.Duration(n) * time.Second
		time.Sleep(timeDuration)

		duration := time.Since(start)
		a.Metrics.CreateRequestDuration.WithLabelValues("GetUser", strconv.Itoa(int(duration.Seconds()))).Observe(duration.Seconds())

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(status))
	}
}

func (a *ApiRest) GetProduct() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		_, span := a.Tracer.OTelTracer.Start(r.Context(), "GetProduct")
		defer span.End()

		var status string
		var product string
		defer func() {
			a.Metrics.ProductStartRequestCounter.WithLabelValues(product, status).Inc()
		}()

		var mr Product
		if err := json.NewDecoder(r.Body).Decode(&mr); err != nil {
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			status = "4xx"
			return
		}

		a.Metrics.ActiveRequestGauge.Inc()
		defer a.Metrics.ActiveRequestGauge.Dec()

		if rand.Float32() > 0.8 {
			status = "4xx"
		} else {
			status = "2xx"
		}

		product = mr.Product
		log.Println(product, status)

		a.Metrics.RequestCounter.WithLabelValues("GetProduct").Inc() // Increment the counter

		rand.Seed(time.Now().UnixNano())
		n := rand.Intn(7) + 1

		timeDuration := time.Duration(n) * time.Second
		time.Sleep(timeDuration)

		duration := time.Since(start)
		a.Metrics.CreateRequestDuration.WithLabelValues("GetProduct", strconv.Itoa(int(duration.Seconds()))).Observe(duration.Seconds())

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(status))
	}
}

type User struct {
	User string
}

type Product struct {
	Product string
}

func producerUser() {
	userPool := []string{"bob", "alice", "jack", "mike", "tiger", "panda", "dog"}
	for {
		postBody, _ := json.Marshal(User{
			User: userPool[rand.Intn(len(userPool))],
		})
		requestBody := bytes.NewBuffer(postBody)
		http.Post("http://api:8989/user", "application/json", requestBody)
		time.Sleep(time.Second * 2)
	}
}

func producerProduct() {
	userPool := []string{"camiseta", "blusa", "calça", "jaqueta", "camisa"}
	for {
		postBody, _ := json.Marshal(Product{
			Product: userPool[rand.Intn(len(userPool))],
		})
		requestBody := bytes.NewBuffer(postBody)
		http.Post("http://api:8989/product", "application/json", requestBody)
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
