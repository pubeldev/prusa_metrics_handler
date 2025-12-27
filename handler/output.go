package handler

import (
	"context"
	"pstrobl96/prusa_metrics_handler/prometheus"
	"sync"

	"github.com/InfluxCommunity/influxdb3-go/v2/influxdb3"
	"github.com/rs/zerolog/log"
)

var (
	client *influxdb3.Client
)

type InfluxClient interface {
	Write(ctx context.Context, data []byte, opts ...influxdb3.WriteOption) error
}

func sentToInflux(message []string, client InfluxClient) (result bool, err error) {
	log.Trace().Msg("Sending to InfluxDB")
	var wg sync.WaitGroup

	for _, line := range message {
		wg.Add(1)
		go func(line string) {
			defer wg.Done()
			log.Trace().Msgf("InfluxDB line: %s", line)
			err = client.Write(context.Background(), []byte(line))
			if err != nil {
				log.Debug().Err(err).Msg("Error while sending to InfluxDB")
				prometheus.MetricsHandlerErrorsTotal.Inc()
			}
		}(line)
	}

	wg.Wait()

	return false, nil
}
