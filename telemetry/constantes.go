package telemetry

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

const (
	PREFIX_METRIC       = "saczuck_"
	PREFIX_PARAM_METRIC = "x_saczuck_"
)
