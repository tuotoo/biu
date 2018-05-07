package biu_test

import (
	"os"

	"github.com/rs/zerolog"
	"github.com/tuotoo/biu"
)

func ExampleLogger() {
	biu.SetLoggerOutput(os.Stdout)
	biu.Debug().Msg("hello")
	biu.SetLoggerLevel(zerolog.InfoLevel)
	biu.Debug().Msg("hello")
	biu.Info().Int("int", 1).Msg("hello")
}
