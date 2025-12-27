package main

import (
	"pubeldev/prusa_metrics_handler/cmd"

	"github.com/rs/zerolog"
)

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnixNano
	cmd.Run()
}
