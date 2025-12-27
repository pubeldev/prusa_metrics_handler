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
	// MetricsHandlerErrorsTotal is the total number of processed messages
	MetricsHandlerErrorsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name:        "prusa_metrics_handler_syslog_messages_errors_total",
		Help:        "The total number errors encountered while processing events",
		ConstLabels: prometheus.Labels{"type": "syslog"},
	})

	// PrinterExpectedMessages tracks expected messages per printer
	PrinterExpectedMessages = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "prusa_metrics_handler_printer_expected_messages",
		Help: "Expected number of messages from printer in the last interval",
	}, []string{"mac_address"})

	// PrinterReceivedMessages tracks received messages per printer
	PrinterReceivedMessages = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "prusa_metrics_handler_printer_received_messages",
		Help: "Received number of messages from printer in the last interval",
	}, []string{"mac_address"})

	// PrinterDropRate tracks message drop rate per printer
	PrinterDropRate = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "prusa_metrics_handler_printer_drop_rate",
		Help: "Message drop rate from printer in the last interval",
	}, []string{"mac_address"})

	// DataPointsWrittenTotal tracks total data points written to InfluxDB
	DataPointsWrittenTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "prusa_metrics_handler_datapoints_written_total",
		Help: "Total number of data points written to InfluxDB",
	})
)
