package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/shirou/gopsutil/cpu"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"ot-prometheus/telemetry"
	"ot-prometheus/telemetryfs"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

var (
	BuildCommit = "undefined"
	BuildTag    = "undefined"
	BuildTime   = "undefined"
)

type ApiRest struct {
	Service *Service
	Metrics telemetry.Prometheus
	Tracer  telemetryfs.Tracer
}

func NewApiRest(service *Service, metrics telemetry.Prometheus, tracer telemetryfs.Tracer) *ApiRest {
	return &ApiRest{
		Service: service,
		Metrics: metrics,
		Tracer:  tracer,
	}
}

type Service struct {
	Repo    *Repository
	Tracer  trace.Tracer
	Metrics telemetry.Prometheus
}

func NewService(repo *Repository, tracer trace.Tracer, metrics telemetry.Prometheus) *Service {
	return &Service{
		Repo:    repo,
		Tracer:  tracer,
		Metrics: metrics,
	}
}

type Repository struct {
	Tracer trace.Tracer
}

func NewRepository(tracer trace.Tracer) *Repository {
	return &Repository{
		Tracer: tracer,
	}
}

func (repo *Repository) FetchUserData(ctx context.Context, userID string) (string, error) {
	_, span := repo.Tracer.Start(ctx, "Repository.FetchUserData")
	defer span.End()

	// Simulando uma busca no banco de dados
	time.Sleep(50 * time.Millisecond)
	return "UserData for " + userID, nil
}

func (repo *Repository) FetchProductData(ctx context.Context, productID string) (string, error) {
	_, span := repo.Tracer.Start(ctx, "Repository.FetchProductData")
	defer span.End()

	// Simulando uma busca no banco de dados
	time.Sleep(50 * time.Millisecond)
	return "ProductData for " + productID, nil
}

func (s *Service) GetUser(ctx context.Context, userID string) (string, error) {
	_, span := s.Tracer.Start(ctx, "Service.GetUser")
	defer span.End()

	userData, err := s.Repo.FetchUserData(ctx, userID)
	if err != nil {
		return "", err
	}

	// Simulando uma chamada de API externa
	// resp, err := s.callExternalAPI(ctx, "http://0.0.0.0:8989/user/"+userID)
	resp, err := s.callExternalAPI(ctx, "http://0.0.0.0:8989/user")
	if err != nil {
		return "", err
	}

	return userData + " and " + resp, nil
}

func (s *Service) GetProduct(ctx context.Context, productID string) (string, error) {
	_, span := s.Tracer.Start(ctx, "Service.GetProduct")
	defer span.End()

	productData, err := s.Repo.FetchProductData(ctx, productID)
	if err != nil {
		return "", err
	}

	// Simulando uma chamada de API externa
	// resp, err := s.callExternalAPI(ctx, "http://0.0.0.0:8989/product"+productID)
	resp, err := s.callExternalAPI(ctx, "http://0.0.0.0:8989/product")
	if err != nil {
		return "", err
	}

	return productData + " and " + resp, nil
}

func (s *Service) callExternalAPI(ctx context.Context, url string) (string, error) {
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

	appMetrics := telemetry.NewPrometheusMetrics()

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
	metricsServer, err := telemetryfs.NewMetricsServer()
	if err != nil {
		logger.Error("creating metrics server", zap.Error(err))
		return
	}

	repo := NewRepository(tracer.OTelTracer)
	service := NewService(repo, tracer.OTelTracer, appMetrics)
	apiRest := NewApiRest(service, appMetrics, tracer)

	router := NewServer(logger, tracer.OTelTracer)
	router.Post("/user", apiRest.GetUser())
	router.Post("/product", apiRest.GetProduct())
	router.Handle("/metrics", promhttp.Handler())

	server := http.Server{
		Addr:    ":8989",
		Handler: router,
	}

	// Iniciando a coleta de métricas de memória e CPU
	initMetricsCollector(appMetrics)

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
		producerProduct()
		producerUser()
	}()
	wg.Wait()
}

func (a *ApiRest) GetUser() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		_, span := a.Tracer.OTelTracer.Start(r.Context(), "Handler.GetUser")
		defer span.End()

		var status string
		defer func() {

			a.Metrics.UserStartRequestCounter.WithLabelValues("stone_balance_user_api", status).Inc()
		}()

		var mr User
		if err := json.NewDecoder(r.Body).Decode(&mr); err != nil {
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			status = "4xx"
			return
		}

		a.Metrics.ActiveRequestGauge.Inc()
		defer a.Metrics.ActiveRequestGauge.Dec()

		result, err := a.Service.GetUser(r.Context(), mr.User)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			status = "5xx"
			return
		}

		status = "2xx"
		log.Println(result, status)

		a.Metrics.RequestCounter.WithLabelValues("stone_balance_user_api_increment").Inc() // Increment the counter

		duration := time.Since(start)
		a.Metrics.CreateRequestDuration.WithLabelValues("stone_balance_user_api_duration", strconv.Itoa(int(duration.Milliseconds()))).Observe(duration.Seconds())

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(result))
	}
}

func (a *ApiRest) GetProduct() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		_, span := a.Tracer.OTelTracer.Start(r.Context(), "Handler.GetProduct")
		defer span.End()

		var status string
		defer func() {
			a.Metrics.ProductStartRequestCounter.WithLabelValues("stone_balance_product_api", status).Inc()
		}()

		mr := Product{}
		if err := json.NewDecoder(r.Body).Decode(&mr); err != nil {
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			status = "4xx"
			return
		}

		a.Metrics.ActiveRequestGauge.Inc()
		defer a.Metrics.ActiveRequestGauge.Dec()

		result, err := a.Service.GetProduct(r.Context(), mr.Product)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			status = "5xx"
			return
		}

		status = "2xx"
		log.Println(result, status)

		a.Metrics.RequestCounter.WithLabelValues("stone_balance_product_api_increment").Inc() // Increment the counter

		duration := time.Since(start)
		a.Metrics.CreateRequestDuration.WithLabelValues("stone_balance_product_api_duration", strconv.Itoa(int(duration.Milliseconds()))).Observe(duration.Seconds())

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(result))
	}
}

type User struct {
	User string `json:"user"`
}

type Product struct {
	Product string `json:"product"`
}

func producerUser() {
	userPool := []string{"bob", "alice", "jack", "mike", "tiger", "panda", "dog"}
	for {
		postBody, _ := json.Marshal(User{
			User: userPool[rand.Intn(len(userPool))],
		})
		requestBody := bytes.NewBuffer(postBody)
		http.Post("http://0.0.0.0:8989/user", "application/json", requestBody)
		time.Sleep(time.Second * 2)
	}
}

func producerProduct() {
	productPool := []string{"camiseta", "blusa", "calça", "jaqueta", "camisa"}
	for {
		postBody, _ := json.Marshal(Product{
			Product: productPool[rand.Intn(len(productPool))],
		})
		requestBody := bytes.NewBuffer(postBody)
		_, err := http.Post("http://0.0.0.0:8989/product", "application/json", requestBody)
		if err != nil {
			fmt.Println("error on send post product", err)
		}
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

type MemoryUsage struct {
	Alloc        uint64 // Memória alocada e ainda não liberada (bytes)
	TotalAlloc   uint64 // Total de memória alocada (bytes)
	Sys          uint64 // Memória obtida do sistema (bytes)
	Mallocs      uint64 // Número de operações de alocação
	Frees        uint64 // Número de operações de liberação
	HeapAlloc    uint64 // Memória alocada no heap (bytes)
	HeapSys      uint64 // Memória do sistema alocada no heap (bytes)
	HeapIdle     uint64 // Memória no heap, mas não usada (bytes)
	HeapInuse    uint64 // Memória alocada no heap e usada (bytes)
	HeapReleased uint64 // Memória no heap liberada para o sistema (bytes)
	HeapObjects  uint64 // Número de objetos alocados no heap
}

// GetMemoryUsage captura e retorna o uso de memória atual
func getMemoryUsage() MemoryUsage {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	return MemoryUsage{
		Alloc:        memStats.Alloc,
		TotalAlloc:   memStats.TotalAlloc,
		Sys:          memStats.Sys,
		Mallocs:      memStats.Mallocs,
		Frees:        memStats.Frees,
		HeapAlloc:    memStats.HeapAlloc,
		HeapSys:      memStats.HeapSys,
		HeapIdle:     memStats.HeapIdle,
		HeapInuse:    memStats.HeapInuse,
		HeapReleased: memStats.HeapReleased,
		HeapObjects:  memStats.HeapObjects,
	}
}

func initMetricsCollector(appMetrics telemetry.Prometheus) {
	go func() {
		ticker := time.NewTicker(time.Second * 1)
		for {
			select {
			case <-ticker.C:
				memoryUsage := getMemoryUsage()
				appMetrics.MemoryUsageGauge.Set(float64(memoryUsage.Alloc))

				cpuTotal, cpuUsage := getCPUInfo()
				appMetrics.CpuTotalUsageGauge.Set(float64(*cpuTotal))
				appMetrics.CpuUsageGauge.Set(float64(*cpuUsage))
			}
		}
	}()
}

func getCPUInfo() (*int, *int) {
	// Número total de CPUs
	numCPU, err := cpu.Counts(true)
	if err != nil {
		fmt.Printf("Erro ao obter o número de CPUs: %v\n", err)
		return nil, nil
	}
	fmt.Printf("Número total de CPUs: %d\n", numCPU)

	// Uso da CPU
	percent, err := cpu.Percent(0, true)
	if err != nil {
		fmt.Printf("Erro ao obter o uso da CPU: %v\n", err)
		return nil, nil
	}

	inUse := 0
	for i, p := range percent {
		fmt.Printf("Uso da CPU %d: %.2f%%\n", i, p)
		if p > 0 {
			inUse++
		}
	}

	fmt.Printf("Número de CPUs em uso: %d\n", inUse)
	return &numCPU, &inUse
}

func getCpuUsage() float64 {
	var (
		usr1, sys1, idle1 uint64
		usr2, sys2, idle2 uint64
	)

	content, err := ioutil.ReadFile("/proc/stat")
	if err != nil {
		log.Printf("Error reading /proc/stat: %v", err)
		return 0
	}

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		// A primeira linha começa com 'cpu', que é o agregado de todas as CPUs
		if len(fields) > 0 && fields[0] == "cpu" {
			numFields := len(fields)
			if numFields >= 5 {
				usr1, _ = strconv.ParseUint(fields[1], 10, 64)
				sys1, _ = strconv.ParseUint(fields[3], 10, 64)
				idle1, _ = strconv.ParseUint(fields[4], 10, 64)
			}
			if numFields >= 8 {
				usr2, _ = strconv.ParseUint(fields[1], 10, 64)
				sys2, _ = strconv.ParseUint(fields[3], 10, 64)
				idle2, _ = strconv.ParseUint(fields[4], 10, 64)
			}
			break
		}
	}

	delta := float64(usr2 + sys2 - usr1 - sys1)
	total := float64(usr2 + sys2 + idle2 - usr1 - sys1 - idle1)
	cpuUsage := 100 * delta / total

	return cpuUsage
}
