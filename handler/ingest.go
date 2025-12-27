package handler

import (
	"fmt"
	"time"

	"github.com/InfluxCommunity/influxdb3-go/v2/influxdb3"

	"github.com/rs/zerolog/log"
	"gopkg.in/mcuadros/go-syslog.v2"
)

func startSyslogServer(listenUDP string) (syslog.LogPartsChannel, *syslog.Server) {
	channel := make(syslog.LogPartsChannel)
	handler := syslog.NewChannelHandler(channel)
	server := syslog.NewServer()
	server.SetFormat(syslog.RFC5424)
	server.SetHandler(handler)
	server.ListenUDP(listenUDP)
	server.Boot()
	return channel, server
}

// MetricsListener is a function to handle syslog metrics and sent them to processor
func MetricsListener(listenUDP string, influxURL string, influxToken string, influxBucket string, influxOrg string, prefix string) {
	var err error
	client, err = influxdb3.New(influxdb3.ClientConfig{
		Host:     influxURL,
		Token:    influxToken,
		Database: influxBucket,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create InfluxDB client")
	}
	defer client.Close()

	channel, server := startSyslogServer(listenUDP)

	go func(channel syslog.LogPartsChannel) {
		for logParts := range channel {
			received := time.Now()
			log.Trace().Msg(fmt.Sprintf("%v", logParts))

			process(logParts, received, prefix)
		}
	}(channel)

	server.Wait()

}
