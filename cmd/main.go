package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"ot-prometheus/handler"
	"ot-prometheus/producer"
	"ot-prometheus/repository"
	"ot-prometheus/service"
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

	metricas := telemetry.NewPrometheusMetrics()
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

	produtorepo := repository.NewProdutoRepository(tracer.OTelTracer)
	produtoservice := service.NewProdutoService(produtorepo, tracer.OTelTracer, metricas)
	produtoHandle := handler.NewProdutoHandle(produtoservice, metricas, tracer)

	userrepo := repository.NewUserRepository(tracer.OTelTracer)
	userservice := service.NewUserService(userrepo, tracer.OTelTracer, metricas)
	userHandle := handler.NewUserHandle(userservice, metricas, tracer)

	router := NewServer(logger, tracer.OTelTracer)

	router.Post("/user", userHandle.GetUser())
	router.Post("/product", produtoHandle.GetProduct())

	router.Handle("/metrics", promhttp.Handler())

	server := http.Server{
		Addr:    ":8989",
		Handler: router,
	}

	// Iniciando a coleta de métricas de memória e CPU
	initMetricsCollector(metricas)

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
		producer.ProducerProduct()
	}()

	wg.Add(1)
	go func() {
		time.Sleep(time.Second)
		producer.ProducerUser()
	}()

	wg.Wait()

}

// ///////////////////////////////////////////////////////////////////////////////////////////////////
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

// ///////////////////////////////////////////////////////////////////////////////////////////////////
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
			appMetrics.CPUUsageGauge.Set(cpuUsage)

			// time.Sleep(time.Second * 5)
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
