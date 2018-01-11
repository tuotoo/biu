package biu_test

import (
	"os"

	"github.com/rs/zerolog"
	"github.com/tuotoo/biu"
)

func ExampleLogger() {
	biu.SetLoggerOutput(os.Stdout)
	biu.Debug("hello", biu.Log())
	biu.SetLoggerLevel(zerolog.InfoLevel)
	biu.Debug("hello", biu.Log())
	biu.Info("hello", biu.Log().Int("int", 1))
}
