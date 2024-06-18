package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"runtime"
	"strconv"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

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

	productRepo := NewProdutoRepository(tracer.OTelTracer)
	productService := NewProdutoService(productRepo, tracer.OTelTracer, metrics)
	productHandle := NewProdutoHandle(productService, metrics, tracer.OTelTracer)

	userRepo := NewUserRepository(tracer.OTelTracer)
	userService := NewUserService(userRepo, tracer.OTelTracer, metrics)
	userHandle := NewUserHandle(userService, metrics, tracer.OTelTracer)

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

type ProdutoHandle struct {
	Service *ProdutoService
	Metrics telemetry.Prometheus
	Tracer  trace.Tracer
}

func NewProdutoHandle(service *ProdutoService, metrics telemetry.Prometheus, tracer trace.Tracer) *ProdutoHandle {
	return &ProdutoHandle{
		Service: service,
		Metrics: metrics,
		Tracer:  tracer,
	}
}

func (h *ProdutoHandle) GetProduct() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		ctx := r.Context()
		ctx, span := h.Tracer.Start(r.Context(), "Handler.GetProduct")
		defer span.End()

		var status string
		defer func() {
			h.Metrics.HTTP_StartRequestCounter.WithLabelValues("x_stone_balance_product_api", status).Inc()
		}()

		mr := Product{}
		if err := json.NewDecoder(r.Body).Decode(&mr); err != nil {
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			status = "4xx"
			return
		}

		h.Metrics.API_ActiveRequestGauge.Inc()
		defer h.Metrics.API_ActiveRequestGauge.Dec()

		result, err := h.Service.GetProduct(ctx, mr.Product)
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

		h.Metrics.HTTP_RequestCounter.WithLabelValues("x_stone_balance_product_api_increment").Inc()

		duration := time.Since(start)
		h.Metrics.API_CreateRequestDuration.WithLabelValues("x_stone_balance_product_api_duration", strconv.Itoa(int(duration.Milliseconds()))).Observe(duration.Seconds())

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(result))
	}
}

type UserHandle struct {
	Service *UserService
	Metrics telemetry.Prometheus
	Tracer  trace.Tracer
}

func NewUserHandle(service *UserService, metrics telemetry.Prometheus, tracer trace.Tracer) *UserHandle {
	return &UserHandle{
		Service: service,
		Metrics: metrics,
		Tracer:  tracer,
	}
}

func (h *UserHandle) GetUser() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ctx := r.Context()
		ctx, span := h.Tracer.Start(ctx, "Handler.GetUser")
		defer span.End()

		logger := telemetryfs.Logger(ctx)

		var status string
		defer func() {
			h.Metrics.HTTP_StartRequestCounter.WithLabelValues("x_stone_balance_user_api", status).Inc()
		}()

		var mr User
		if err := json.NewDecoder(r.Body).Decode(&mr); err != nil {
			logger.Error("error on bind json", zap.Error(err))
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			status = "4xx"
			return
		}

		h.Metrics.API_ActiveRequestGauge.Inc()
		defer h.Metrics.API_ActiveRequestGauge.Dec()

		span.SetAttributes(attribute.String("user", mr.User))

		result, err := h.Service.GetUser(ctx, mr.User)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			status = strconv.Itoa(http.StatusInternalServerError)
			return
		}

		logger.Info("user data", zap.String("user", mr.User), zap.String("data", result))

		if rand.Float32() > 0.8 {
			status = "4xx"
		} else {
			status = "2xx"
		}

		log.Println(result, status)

		h.Metrics.HTTP_RequestCounter.WithLabelValues("x_stone_balance_user_api_increment").Inc()

		duration := time.Since(start)
		h.Metrics.API_CreateRequestDuration.WithLabelValues("x_stone_balance_user_api_duration", strconv.Itoa(int(duration.Milliseconds()))).Observe(duration.Seconds())

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(result))
	}
}

type ProdutoService struct {
	Repository *ProdutoRepository
	Tracer     trace.Tracer
	Metrics    telemetry.Prometheus
}

func NewProdutoService(repo *ProdutoRepository, tracer trace.Tracer, metrics telemetry.Prometheus) *ProdutoService {
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

type UserService struct {
	UserRepo *UserRepository
	Tracer   trace.Tracer
	Metrics  telemetry.Prometheus
}

func NewUserService(repo *UserRepository, tracer trace.Tracer, metrics telemetry.Prometheus) *UserService {
	return &UserService{
		UserRepo: repo,
		Tracer:   tracer,
		Metrics:  metrics,
	}
}

func (s *UserService) GetUser(ctx context.Context, userID string) (string, error) {
	ctx, span := s.Tracer.Start(ctx, "Service.GetUser")
	defer span.End()

	s.Metrics.API_ActiveRequestGauge.Inc()
	defer s.Metrics.API_ActiveRequestGauge.Dec()

	userData, err := s.UserRepo.FetchUserData(ctx, userID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return "", err
	}

	return userData, nil
}

type ProdutoRepository struct {
	Tracer trace.Tracer
}

func NewProdutoRepository(tracer trace.Tracer) *ProdutoRepository {
	return &ProdutoRepository{
		Tracer: tracer,
	}
}

func (r *ProdutoRepository) FetchProductData(ctx context.Context, productID string) (string, error) {
	// Simulating fetching product data from a database or external service
	// Here you can add your implementation to fetch real product data
	time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)
	return fmt.Sprintf("Product data for ID: %s", productID), nil
}

type UserRepository struct {
	Tracer trace.Tracer
}

func NewUserRepository(tracer trace.Tracer) *UserRepository {
	return &UserRepository{
		Tracer: tracer,
	}
}

func (r *UserRepository) FetchUserData(ctx context.Context, userID string) (string, error) {
	// Simulating fetching user data from a database or external service
	// Here you can add your implementation to fetch real user data
	time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)
	return fmt.Sprintf("User data for ID: %s", userID), nil
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

type User struct {
	User string `json:"user"`
}

type Product struct {
	Product string `json:"product"`
}
