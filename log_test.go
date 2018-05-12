package biu_test

import (
	"github.com/rs/zerolog"
	"github.com/tuotoo/biu"
)

func ExampleUseConsoleLogger() {
	biu.UseConsoleLogger()
	biu.Debug().Msg("hello")
	biu.SetLoggerLevel(zerolog.InfoLevel)
	biu.Debug().Msg("hello")
	biu.Info().Int("int", 1).Msg("hello")
}
