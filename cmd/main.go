package cmd

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"pubeldev/prusa_metrics_handler/config"
	"pubeldev/prusa_metrics_handler/handler"
	"strconv"
	"sync"
	"syscall"

	"github.com/alecthomas/kingpin/v2"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
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

	logLevel = kingpin.Flag("log.level", "Log level for prusa_metrics_handler.").Default("info").String()
	prefix   = kingpin.Flag("prefix", "Prefix for metrics").Default("prusa_").String()

	metricsPath = kingpin.Flag("exporter.metrics-path", "Path where to expose metrics.").Default("/metrics").String()
	metricsPort = kingpin.Flag("exporter.metrics-port", "Port where to expose metrics.").Default("10011").Int()
)

// Run function to start the metrics handler
func Run() {
	kingpin.Parse()
	log.Info().Msg("// pubel.dev prusa_metrics_handler starting")

	logLevel, err := zerolog.ParseLevel(*logLevel)

	if err != nil {
		logLevel = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(logLevel)

	client := influxdb2.NewClient(*influxHost, *influxToken)
	defer client.Close()

	ready, err := client.Ready(context.Background())
	if err != nil {
		log.Printf("Warning: InfluxDB not ready: %v", err)
	} else {
		log.Printf("InfluxDB Ready: %s", ready)
	}

	cfg := config.LoadConfig(*influxListenHost, *influxListenHostPort, *influxHost, *influxToken, *influxOrg, *influxBucket, *prefix)

	app := handler.NewApplication(client, cfg)

	ctx, cancel := context.WithCancel(context.Background())

	// Handle signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	log.Info().Msg("Syslog logs server starting at: " + *influxListenHost)

	var wg sync.WaitGroup

	// Start Workers
	wg.Add(1)
	go func() {
		defer wg.Done()
		app.RunSyslogServer(ctx)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		app.RunWriter(ctx)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		app.RunReporter(ctx)
	}()

	http.Handle(*metricsPath, promhttp.Handler())
	http.ListenAndServe(":"+strconv.Itoa(*metricsPort), nil)

	// Wait for signal
	<-sigChan
	log.Info().Msg("Shutting down...")
	cancel()
	wg.Wait()
	log.Info().Msg("Shutdown complete.")
}
