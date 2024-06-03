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
		apiMetrics.CreateRequestDuration,
		apiMetrics.RequestCounter,
		apiMetrics.ActiveRequestGauge,
		apiMetrics.UserStartRequestCounter,
	)

	return Prometheus{
		ApiMetrics: apiMetrics,
	}
}

type ApiMetrics struct {
	RequestCounter             *prometheus.CounterVec
	CreateRequestDuration      *prometheus.HistogramVec
	ActiveRequestGauge         prometheus.Gauge
	UserStartRequestCounter    *prometheus.CounterVec
	ProductStartRequestCounter *prometheus.CounterVec
}

func NewApiMetrics() ApiMetrics {
	return ApiMetrics{
		RequestCounter: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "test_counter",
			Help: "Count how many report statements have been added",
		},
			[]string{"handler_name"},
		),
		CreateRequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "response_time_seconds",
				Help:    "Histogram of response times for handler in seconds",
				Buckets: []float64{0.1, 0.5, 1, 2, 5, 10, 30, 60, 120},
			},
			[]string{"handler_name", "status"},
		),
		ActiveRequestGauge: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "active_requests",
			Help: "Current number of active requests being handled",
		}),
		UserStartRequestCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_request_get_user_status_count", // metric name
				Help: "Count of status returned by user.",
			},
			[]string{"user", "status"}, // labels
		),
		ProductStartRequestCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_request_get_user_status_count", // metric name
				Help: "Count of status returned by user.",
			},
			[]string{"product", "status"}, // labels
		),
	}
}
