package biu_test

import (
	"github.com/rs/zerolog"
	"github.com/tuotoo/biu"
)

func ExampleLogger() {
	biu.Logger().With().Timestamp()
	biu.Debug("hello", biu.Log())
	biu.SetLoggerLevel(zerolog.InfoLevel)
	biu.Debug("hello", biu.Log())
	biu.Info("hello", biu.Log().Int("int", 1))
}
