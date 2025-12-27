package prometheus

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// MetricsHandlerTotal is the total number of processed messages
	MetricsHandlerTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name:        "prusa_metrics_handler_syslog_messages_total",
		Help:        "The total number of processed events",
		ConstLabels: prometheus.Labels{"type": "syslog"},
	})
	// MetricsHandlerTotal is the total number of processed messages
	MetricsHandlerErrorsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name:        "prusa_metrics_handler_syslog_messages_errors_total",
		Help:        "The total number errors encountered while processing events",
		ConstLabels: prometheus.Labels{"type": "syslog"},
	})
)
