package biu

import (
	"io"
	"os"

	"github.com/rs/zerolog"
)

// LogEvt is alias of zerolog.Event
type LogEvt = zerolog.Event

var logger = zerolog.New(os.Stderr).With().Timestamp().Logger()

// SetLoggerOutput sets the output of logger.
func SetLoggerOutput(w io.Writer) {
	logger = logger.Output(w)
}

// SetLoggerLevel sets the log level of logger.
func SetLoggerLevel(level zerolog.Level) {
	logger = logger.Level(level)
}

// Log returns *zerolog.Event.
func Log() *LogEvt {
	return zerolog.Dict()
}

// Logger get the default logger of biu.
func Logger() zerolog.Logger {
	return logger
}

// Debug starts a new message with info level.
func Debug(msg string, evt *LogEvt) {
	logger.Debug().Dict("fields", evt).Msg(msg)
}

// Info starts a new message with info level.
func Info(msg string, evt *LogEvt) {
	logger.Info().Dict("fields", evt).Msg(msg)
}

// Warn starts a new message with warn level.
func Warn(msg string, evt *LogEvt) {
	logger.Warn().Dict("fields", evt).Msg(msg)
}

// Error starts a new message with error level.
func Error(msg string, evt *LogEvt) {
	logger.Error().Dict("fields", evt).Msg(msg)
}

// Fatal starts a new message with fatal level.
// The os.Exit(1) function is called by the Msg method.
func Fatal(msg string, evt *LogEvt) {
	logger.Fatal().Dict("fields", evt).Msg(msg)
}

// Panic starts a new message with panic level.
// The message is also sent to the panic function.
func Panic(msg string, evt *LogEvt) {
	logger.Panic().Dict("fields", evt).Msg(msg)
}
