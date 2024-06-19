package telemetria

import "github.com/prometheus/client_golang/prometheus"

type options struct {
	register         prometheus.Registerer
	additionalLabels []string
}

func defaultOptions() options {
	return options{
		register:         prometheus.DefaultRegisterer,
		additionalLabels: []string{},
	}
}

type Option func(*options)

func WithRegisterer(r prometheus.Registerer) Option {
	return func(o *options) {
		o.register = r
	}
}

func WithAdditionalLabels(additionalLabels []string) Option {
	return func(o *options) {
		o.additionalLabels = additionalLabels
	}
}
