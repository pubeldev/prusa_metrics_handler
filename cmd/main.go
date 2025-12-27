package cmd

import (
	"net/http"
	"pstrobl96/prusa_metrics_handler/handler"
	"strconv"

	"github.com/alecthomas/kingpin/v2"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	influxListenHost     = kingpin.Flag("influx.listen.hostname", "UDP address to listen on for syslog messages.").Default("0.0.0.0").String()
	influxListenHostPort = kingpin.Flag("influx.listen.port", "UDP port to listen on for syslog messages.").Default("8514").Int()
	influxHost           = kingpin.Flag("influx.url", "url for influx").Default("http://localhost:8181").String()
	influxBucket         = kingpin.Flag("influx.bucket", "Bucket for influx (or database for v1)").Default("prusa").String()
	influxOrg            = kingpin.Flag("influx.org", "Influx organization").Default("prusa").String()
	influxToken          = kingpin.Flag("influx.token", "Token for influx").Default("loremipsumdolorsitamet").String()
	logLevel             = kingpin.Flag("log.level", "Log level for prusa_metrics_handler.").Default("info").String()
	prefix               = kingpin.Flag("prefix", "Prefix for metrics").Default("prusa_").String()
	metricsPath          = kingpin.Flag("exporter.metrics-path", "Path where to expose metrics.").Default("/metrics").String()
	metricsPort          = kingpin.Flag("exporter.metrics-port", "Port where to expose metrics.").Default("10011").Int()
)

// Run function to start the metrics handler
func Run() {
	kingpin.Parse()
	log.Info().Msg("Prusa metrics handler starting")

	logLevel, err := zerolog.ParseLevel(*logLevel)

	if err != nil {
		logLevel = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(logLevel)
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnixNano

	log.Info().Msg("Syslog logs server starting at: " + *syslogListenAddress)
	go handler.MetricsListener(*syslogListenAddress, *influxURL, *influxToken, *influxBucket, *influxOrg, *prefix)

	http.Handle(*metricsPath, promhttp.Handler())
	http.ListenAndServe(":"+strconv.Itoa(*metricsPort), nil)

}
