package telemetry

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

type Prometheus struct {
	ApiMetrics
}

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

func NewApiMetrics() ApiMetrics {
	return ApiMetrics{
		HTTP_RequestCounter: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "stone_http_request_counter",
			Help: "Count how many report statements have been added",
		},
			[]string{"handler_name"},
		),
		API_CreateRequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "stone_response_time_seconds",
				Help:    "Histogram of response times for handler in seconds",
				Buckets: []float64{0.1, 0.5, 1, 2, 5, 10, 30, 60, 120},
			},
			[]string{"handler_name", "response_time"},
		),
		API_ActiveRequestGauge: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "stone_active_requests",
			Help: "Current number of active requests being handled",
		}),
		HTTP_StartRequestCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "stone_http_request_status_count", // metric name
				Help: "Count of status returned by user.",
			},
			[]string{"user", "status"}, // labels
		),

		MemoryUsageGauge: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "stone_app_memory_usage_bytes",
			Help: "Current memory usage of the application in bytes.",
		}),
		MemoryAllocGauge: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "stone_app_memory_alloc_bytes",
			Help: "Current memory allocated by the application in bytes.",
		}),
		MemorySysGauge: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "stone_app_memory_sys_bytes",
			Help: "Current memory usage by the system in bytes.",
		}),
		CPUUsageGauge: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "stone_app_cpu_usage_percent",
			Help: "Current CPU usage of the application as a percentage.",
		}),
	}
}
