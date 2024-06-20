package telemetria

import (
	"github.com/prometheus/client_golang/prometheus"
)

const (
	StatusOK      = "ok"
	StatusError   = "error"
	StatusHit     = "hit"
	StatusMiss    = "miss"
	StatusTrue    = "true"
	StatusFalse   = "false"
	StatusSuccess = "success"
	StatusFailure = "failure"
)

// Prometheus encapsulates all the API metrics.
type Prometheus struct {
	ApiMetrics
}

// NewPrometheusMetrics initializes and registers Prometheus metrics, returning a Prometheus struct.
func NewPrometheusMetrics() Prometheus {
	apiMetrics := NewApiMetrics()

	prometheus.MustRegister(
		apiMetrics.API_CreateRequestDuration,
		apiMetrics.HTTP_RequestCounter,
		apiMetrics.API_ActiveRequestGauge,
		apiMetrics.HTTP_StartRequestCounter,
		apiMetrics.MemoryUsageGauge,
		apiMetrics.MemoryAllocGauge,
		apiMetrics.MemorySysGauge,
		apiMetrics.CPUUsageGauge,
	)

	return Prometheus{
		ApiMetrics: apiMetrics,
	}
}

// ApiMetrics defines all the metrics for the API.
type ApiMetrics struct {
	HTTP_RequestCounter       *prometheus.CounterVec
	API_CreateRequestDuration *prometheus.HistogramVec
	API_ActiveRequestGauge    prometheus.Gauge
	HTTP_StartRequestCounter  *prometheus.CounterVec
	MemoryUsageGauge          prometheus.Gauge
	MemoryAllocGauge          prometheus.Gauge
	MemorySysGauge            prometheus.Gauge
	CPUUsageGauge             prometheus.Gauge
}

// NewApiMetrics initializes and returns an ApiMetrics struct with all the defined metrics.
func NewApiMetrics() ApiMetrics {
	return ApiMetrics{
		HTTP_RequestCounter: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: PREFIXO + "http_request_counter" + COUNTER,
			Help: "Count how many report statements have been added",
		},
			[]string{"handler_name"},
		),
		API_CreateRequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    PREFIXO + "response_time_seconds" + HISTO,
				Help:    "Histogram of response times for handler in seconds",
				Buckets: []float64{0.1, 0.5, 1, 2, 5, 10, 30, 60, 120},
			},
			[]string{"handler_name", "response_time"},
		),
		API_ActiveRequestGauge: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: PREFIXO + "stone_active_requests" + GAUGE,
			Help: "Current number of active requests being handled",
		}),
		HTTP_StartRequestCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: PREFIXO + "http_request_status_count" + COUNTER, // metric name
				Help: "Count of status returned by user.",
			},
			[]string{"user", "status"}, // labels
		),
		MemoryUsageGauge: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: PREFIXO + "app_memory_usage_bytes" + GAUGE,
			Help: "Current memory usage of the application in bytes.",
		}),
		MemoryAllocGauge: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: PREFIXO + "app_memory_alloc_bytes" + GAUGE,
			Help: "Current memory allocated by the application in bytes.",
		}),
		MemorySysGauge: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: PREFIXO + "app_memory_sys_bytes" + GAUGE,
			Help: "Current memory usage by the system in bytes.",
		}),
		CPUUsageGauge: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: PREFIXO + "app_cpu_usage_percent" + GAUGE,
			Help: "Current CPU usage of the application as a percentage.",
		}),
	}
}

// InitMetricsCollector initializes and registers additional Prometheus metrics.
func InitMetricsCollector(metrics Prometheus) {
	metrics.HTTP_RequestCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: PREFIXO + "http_request_total" + COUNTER,
			Help: "Total number of HTTP requests made.",
		},
		[]string{"handler", "status"},
	)
	prometheus.MustRegister(metrics.HTTP_RequestCounter)

	metrics.HTTP_StartRequestCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: PREFIXO + "http_start_request_total" + COUNTER,
			Help: "Total number of HTTP start requests made.",
		},
		[]string{"handler", "status"},
	)
	prometheus.MustRegister(metrics.HTTP_StartRequestCounter)

	metrics.API_ActiveRequestGauge = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: PREFIXO + "api_active_requests" + GAUGE,
			Help: "Number of active API requests.",
		},
	)
	prometheus.MustRegister(metrics.API_ActiveRequestGauge)

	metrics.API_CreateRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: PREFIXO + "api_request_duration_seconds" + HISTO,
			Help: "Duration of API requests in seconds.",
			Buckets: []float64{
				0.1, 0.3, 1.2, 5.0,
			},
		},
		[]string{"handler", "duration"},
	)
	prometheus.MustRegister(metrics.API_CreateRequestDuration)
}
