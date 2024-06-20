package telemetria

import "github.com/prometheus/client_golang/prometheus"

// options encapsulates the configuration for metrics registration and additional labels.
type options struct {
	register         prometheus.Registerer
	additionalLabels []string
}

// defaultOptions returns the default configuration options.
func defaultOptions() options {
	return options{
		register:         prometheus.DefaultRegisterer,
		additionalLabels: []string{},
	}
}

// Option defines a function type for modifying options.
type Option func(*options)

// WithRegisterer sets a custom Prometheus registerer in the options.
func WithRegisterer(r prometheus.Registerer) Option {
	return func(o *options) {
		o.register = r
	}
}

// WithAdditionalLabels sets additional labels to be used in metrics.
func WithAdditionalLabels(additionalLabels []string) Option {
	return func(o *options) {
		o.additionalLabels = additionalLabels
	}
}
