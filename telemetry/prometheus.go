package telemetry

import (
	"github.com/prometheus/client_golang/prometheus"
)

type ApplicationMetrics struct {
	Metric_RequestCounter        *prometheus.CounterVec
	Metric_CreateRequestDuration *prometheus.HistogramVec
	Metric_ActiveRequestGauge    prometheus.Gauge
	Metric_StartRequestCounter   *prometheus.CounterVec
	Metric_MemoryUsageGauge      prometheus.Gauge
	Metric_CpuUsageGauge         prometheus.Gauge
}

type Prometheus struct {
	ApplicationMetrics
}

func NewPrometheusMetrics() Prometheus {
	appMetrics := NewAppMetrics()

	prometheus.MustRegister(
		appMetrics.Metric_CreateRequestDuration,
		appMetrics.Metric_RequestCounter,
		appMetrics.Metric_ActiveRequestGauge,
		appMetrics.Metric_StartRequestCounter,
		appMetrics.Metric_MemoryUsageGauge,
		appMetrics.Metric_CpuUsageGauge,
	)

	return Prometheus{
		ApplicationMetrics: appMetrics,
	}
}

func NewAppMetrics() ApplicationMetrics {
	return ApplicationMetrics{
		Metric_RequestCounter: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: PREFIX_METRIC + "test_counter",
			Help: "Count how many report statements have been added",
		},
			[]string{"handler_name"},
		),
		Metric_CreateRequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    PREFIX_METRIC + "response_time_seconds",
				Help:    "Histogram of response times for handler in seconds",
				Buckets: []float64{0.1, 0.5, 1, 2, 5, 10, 30, 60, 120},
			},
			[]string{"handler_name", "response_time"},
		),
		Metric_ActiveRequestGauge: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: PREFIX_METRIC + "active_requests",
			Help: "Current number of active requests being handled",
		}),
		Metric_StartRequestCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: PREFIX_METRIC + "http_request_get_user_status_count", // metric name
				Help: "Count of status returned by user.",
			},
			[]string{"user", "status"}, // labels
		),

		Metric_MemoryUsageGauge: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: PREFIX_METRIC + "app_memory_usage_bytes",
			Help: "Current memory usage of the application in bytes.",
		}),
		Metric_CpuUsageGauge: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: PREFIX_METRIC + "app_cpu_usage_percent",
			Help: "Current CPU usage of the application as a percentage.",
		}),
	}
}
