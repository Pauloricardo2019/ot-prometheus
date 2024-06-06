package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
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
	"golang.org/x/sys/unix"
)

var (
	BuildCommit = "undefined"
	BuildTag    = "1.0.0"
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

	return userData, nil
}

func (s *Service) GetProduct(ctx context.Context, productID string) (string, error) {
	_, span := s.Tracer.Start(ctx, "Service.GetProduct")
	defer span.End()

	productData, err := s.Repo.FetchProductData(ctx, productID)
	if err != nil {
		return "", err
	}

	return productData, nil
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
		time.Sleep(time.Second)
		producerProduct()
	}()

	wg.Add(1)
	go func() {
		time.Sleep(time.Second)
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

func initMetricsCollector(appMetrics telemetry.Prometheus) {
	go func() {
		for {
			// Obtenha a memória alocada pelo programa
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			freeMemory, _ := GetFreeMemory()
			appMetrics.MemoryUsageGauge.Set(freeMemory)

			// Obtenha a utilização atual da CPU
			cpuUsage, _ := GetCPUUsage()
			appMetrics.CpuUsageGauge.Set(cpuUsage)

			time.Sleep(time.Second * 5)
		}
	}()
}

func GetFreeMemory() (float64, error) {
	var stat unix.Sysinfo_t

	// Chama a função Sysinfo que preenche a struct Sysinfo_t
	if err := unix.Sysinfo(&stat); err != nil {
		return 0, fmt.Errorf("erro ao obter informações do sistema: %w", err)
	}

	// A quantidade de memória livre está em stat.Freeram e stat.Bufferram
	// Os valores estão em KB, então multiplicamos por 1024 para obter em bytes
	freeMemory := float64(stat.Freeram) * float64(stat.Unit)
	bufferMemory := float64(stat.Bufferram) * float64(stat.Unit)

	// Memória livre total
	totalFreeMemory := freeMemory + bufferMemory

	return totalFreeMemory, nil
}

func GetCPUUsage() (float64, error) {
	idle1, total1, err := getCPUSample()
	if err != nil {
		return 0, err
	}

	time.Sleep(1 * time.Second)

	idle2, total2, err := getCPUSample()
	if err != nil {
		return 0, err
	}

	idleTicks := float64(idle2 - idle1)
	totalTicks := float64(total2 - total1)

	if totalTicks == 0 {
		return 0, fmt.Errorf("totalTicks é zero, possível erro na leitura das amostras")
	}

	cpuUsage := 100 * (totalTicks - idleTicks) / totalTicks
	fmt.Println("CPU USAGE: ", cpuUsage)
	return cpuUsage, nil
}

// getCPUSample coleta uma amostra dos tempos de CPU.
func getCPUSample() (uint64, uint64, error) {
	file, err := os.Open("/proc/stat")
	if err != nil {
		return 0, 0, fmt.Errorf("erro ao abrir /proc/stat: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "cpu ") {
			fields := strings.Fields(line)
			if len(fields) < 5 {
				return 0, 0, fmt.Errorf("linha /proc/stat malformada: %s", line)
			}

			idle, err := strconv.ParseUint(fields[4], 10, 64)
			if err != nil {
				return 0, 0, fmt.Errorf("erro ao fazer parse de idle: %w", err)
			}

			total := uint64(0)
			for _, field := range fields[1:] {
				value, err := strconv.ParseUint(field, 10, 64)
				if err != nil {
					return 0, 0, fmt.Errorf("erro ao fazer parse de field: %w", err)
				}
				total += value
			}

			return idle, total, nil
		}
	}

	if err := scanner.Err(); err != nil {
		return 0, 0, fmt.Errorf("erro ao ler /proc/stat: %w", err)
	}

	return 0, 0, fmt.Errorf("linha de CPU não encontrada em /proc/stat")
}
