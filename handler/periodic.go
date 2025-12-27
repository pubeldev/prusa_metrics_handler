package handler

import (
	"context"
	"net"
	"pubeldev/prusa_metrics_handler/prometheus"
	"time"

	"github.com/influxdata/influxdb-client-go/v2/api/write"
	"github.com/rs/zerolog/log"
)

func (app *Application) RunReporter(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			app.PrintersMu.RLock()
			for _, printer := range app.Printers {
				report := printer.CreateReport(3.0)
				if report != nil {
					prometheus.PrinterExpectedMessages.WithLabelValues(printer.MacAddress).Set(float64(report["expected_number_of_messages"].(int)))
					prometheus.PrinterReceivedMessages.WithLabelValues(printer.MacAddress).Set(float64(report["received_number_of_messages"].(int)))
					prometheus.PrinterDropRate.WithLabelValues(printer.MacAddress).Set(report["drop_rate"].(float64))
				}
			}
			app.PrintersMu.RUnlock()
		}
	}
}

func (app *Application) RunWriter(ctx context.Context) {
	// InfluxDB Client Go handles batching internally
	for {
		select {
		case <-ctx.Done():
			app.WriteAPI.Flush()
			return
		case p := <-app.PointsChan:
			prometheus.DataPointsWrittenTotal.Inc()

			// Convert to InfluxDB Point
			ip := write.NewPoint(
				p.Measurement,
				p.Tags,
				p.Values,
				p.Timestamp,
			)
			app.WriteAPI.WritePoint(ip)
		}
	}
}

func (app *Application) RunSyslogServer(ctx context.Context) {
	addr, err := net.ResolveUDPAddr("udp", app.SyslogAddr)
	if err != nil {
		log.Fatal().Msgf("Failed to resolve UDP: %v", err)
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		log.Fatal().Msgf("Failed to listen UDP: %v", err)
	}
	defer conn.Close()

	log.Info().Msgf("Listening for Syslog on %s", app.SyslogAddr)

	buf := make([]byte, 65535)

	// Create a sub-routine to handle context cancellation closing the socket
	go func() {
		<-ctx.Done()
		conn.Close()
	}()

	for {
		n, _, err := conn.ReadFromUDP(buf)
		if err != nil {
			prometheus.MetricsHandlerErrorsTotal.Inc()

			if ctx.Err() != nil {
				return // Context cancelled
			}
			log.Info().Msgf("UDP Read Error: %v", err)
			continue
		}

		prometheus.MetricsHandlerTotal.Inc()

		// Copy data to avoid buffer races in goroutines if we parallelize processing
		data := make([]byte, n)
		copy(data, buf[:n])

		// Process synchronously (or fire goroutine if load is high)
		app.ProcessMessage(data)
	}
}
