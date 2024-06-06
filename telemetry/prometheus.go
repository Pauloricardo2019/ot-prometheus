package telemetry

import (
	"github.com/prometheus/client_golang/prometheus"
)

const (
	// StatusOK is a label for successful metrics in Prometheus.
	StatusOK = "ok"
	// StatusError is a label for error-related metrics in Prometheus.
	StatusError = "error"
	// StatusHit is label for hit cache metrics in Prometheus.
	StatusHit = "hit"
	// StatusMiss is a label for miss cache metrics in Prometheus.
	StatusMiss = "miss"
	// StatusTrue is a label for the presence of a value.
	StatusTrue = "true"
	// StatusFalse is a label for the absence of a value.
	StatusFalse = "false"
	// StatusSuccess is a label for successful metrics in Prometheus.
	StatusSuccess = "success"
	// StatusFailure is a label for failure metrics in Prometheus.
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
		apiMetrics.CpuUsageGauge,
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
	CpuUsageGauge             prometheus.Gauge
}

func NewApiMetrics() ApiMetrics {
	return ApiMetrics{
		HTTP_RequestCounter: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "stone_test_counter",
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
				Name: "stone_http_request_get_user_status_count", // metric name
				Help: "Count of status returned by user.",
			},
			[]string{"user", "status"}, // labels
		),

		MemoryUsageGauge: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "stone_app_memory_usage_bytes",
			Help: "Current memory usage of the application in bytes.",
		}),
		CpuUsageGauge: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "stone_app_cpu_usage_percent",
			Help: "Current CPU usage of the application as a percentage.",
		}),
	}
}
