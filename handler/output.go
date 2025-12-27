package handler

import (
	"context"

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

	for _, line := range message {
		err = client.Write(context.Background(), []byte(line))
		if err != nil {
			log.Debug().Err(err).Msg("Error while sending to InfluxDB")
			return false, err
		}
	}

	return false, nil
}
