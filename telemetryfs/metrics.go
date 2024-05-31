package telemetryfs

import (
	"github.com/go-chi/chi/v5"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"ot-prometheus/telemetry"
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

	return &http.Server{
		Handler: mux,
		Addr:    ":1616",
	}, nil
}

type RedMetricsMiddleware struct {
	// httpServerRequestDuration collects the metric for http server request duration in milliseconds.
	httpServerRequestDuration *prometheus.HistogramVec
	// additionalLabels contains additional labels to provide context to httpServerRequestDuration metric.
	additionalLabels []string
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
				Name:    "http_server_request_duration_ms",
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
		fn := func(w http.ResponseWriter, r *http.Request) {
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

			m.httpServerRequestDuration.With(labels).Observe(time.Now().Sub(start).Seconds() * 1000)
		}

		return http.HandlerFunc(fn)
	}
}
