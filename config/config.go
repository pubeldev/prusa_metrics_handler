package config

import (
	"time"

	"github.com/rs/zerolog/log"
)

type Config struct {
	Host              string
	Port              int
	DBHost            string
	DBToken           string
	DBOrg             string
	DBBucket          string
	ReportingInterval time.Duration
	Prefix            string
}

func LoadConfig(host string, port int, dbHost, dbToken, dbOrg, dbBucket, prefix string) Config {
	log.Trace().Msgf("Loading configuration: host=%s, port=%d, dbHost=%s, dbOrg=%s, dbBucket=%s", host, port, dbHost, dbOrg, dbBucket)

	return Config{
		Host:     host,
		Port:     port,
		DBHost:   dbHost,
		DBToken:  dbToken,
		DBOrg:    dbOrg,
		DBBucket: dbBucket,
		Prefix:   prefix,
	}
}
