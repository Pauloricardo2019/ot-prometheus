package telemetryfs

import (
	"net/http"
	"strconv"
	"time"

	"ot-prometheus/telemetry"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func NewMetricsServer(options ...Option) (*http.Server, error) {
	opts := defaultOptions()
	for _, option := range options {
		option(&opts)
	}

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.InstrumentMetricHandler(
		opts.register,
		promhttp.HandlerFor(prometheus.DefaultGatherer, promhttp.HandlerOpts{}),
	))

	server := &http.Server{
		Handler: mux,
		Addr:    ":9191",
	}
	return server, nil
}

type RedMetricsMiddleware struct {
	httpServerRequestDuration *prometheus.HistogramVec
	additionalLabels          []string
}

func NewRedMetricsMiddleware(options ...Option) *RedMetricsMiddleware {
	opts := defaultOptions()
	for _, option := range options {
		option(&opts)
	}

	defaultLabels := []string{"http_response_status_code", "http_request_method", "http_route"}

	metrics := RedMetricsMiddleware{
		additionalLabels: opts.additionalLabels,
		httpServerRequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    telemetry.PREFIX_METRIC + "http_server_request_duration_ms",
				Help:    "Request duration histogram for HTTP server in milliseconds",
				Buckets: []float64{1, 5, 10, 25, 50, 100, 300, 500, 1000, 5000, 10000},
			},
			append(defaultLabels, opts.additionalLabels...),
		),
	}

	opts.register.MustRegister(metrics.httpServerRequestDuration)

	return &metrics
}

func (m *RedMetricsMiddleware) Handle() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			dw := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			start := time.Now()
			next.ServeHTTP(dw, r)

			ctx := r.Context()
			pathPattern := chi.RouteContext(ctx).RoutePattern()

			labels := prometheus.Labels{
				"http_response_status_code": strconv.Itoa(dw.Status()),
				"http_request_method":       r.Method,
				"http_route":                pathPattern,
			}

			q := r.URL.Query()

			for _, label := range m.additionalLabels {
				if q.Has(label) {
					labels[label] = telemetry.StatusTrue
					continue
				}

				labels[label] = telemetry.StatusFalse
			}

			m.httpServerRequestDuration.With(labels).Observe(time.Since(start).Seconds() * 1000)
		})
	}
}
