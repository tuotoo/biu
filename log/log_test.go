package log_test

import (
	"github.com/rs/zerolog"
	"github.com/tuotoo/biu/log"
)

func ExampleUseConsoleLogger() {
	log.UseConsoleLogger()
	log.Debug().Msg("hello")
	log.SetLoggerLevel(zerolog.InfoLevel)
	log.Debug().Msg("hello")
	log.Info().Int("int", 1).Msg("hello")
}
