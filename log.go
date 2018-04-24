package biu

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/json-iterator/go"
	"github.com/rs/zerolog"
)

// LogEvt is alias of zerolog.Event
type LogEvt = zerolog.Event

var logger = zerolog.New(os.Stderr).With().Timestamp().Logger()

const (
	cReset    = 0
	cBold     = 1
	cRed      = 31
	cGreen    = 32
	cYellow   = 33
	cBlue     = 34
	cMagenta  = 35
	cCyan     = 36
	cGray     = 37
	cDarkGray = 90
)

var consoleBufPool = sync.Pool{
	New: func() interface{} {
		return bytes.NewBuffer(make([]byte, 0, 100))
	},
}

type ColorWriter struct{}

func (w ColorWriter) Write(p []byte) (n int, err error) {
	var event map[string]interface{}
	err = jsoniter.Unmarshal(p, &event)
	if err != nil {
		return
	}
	buf := consoleBufPool.Get().(*bytes.Buffer)
	defer consoleBufPool.Put(buf)
	lvlColor := cReset
	level := "????"
	if l, ok := event[zerolog.LevelFieldName].(string); ok {
		level = strings.ToUpper(l)[0:4]
	}
	fmt.Fprintf(buf, "%s |%s| %s",
		colorize(event[zerolog.TimestampFieldName], cDarkGray),
		colorize(level, lvlColor),
		colorize(event[zerolog.MessageFieldName], cReset))
	fields := make([]string, 0, len(event))
	for field := range event {
		switch field {
		case zerolog.LevelFieldName, zerolog.TimestampFieldName, zerolog.MessageFieldName:
			continue
		}
		fields = append(fields, field)
	}
	sort.Strings(fields)
	for _, field := range fields {
		fmt.Fprintf(buf, " %s: ", colorize(field, cCyan))
		switch value := event[field].(type) {
		case string:
			if needsQuote(value) {
				buf.WriteString(strconv.Quote(value))
			} else {
				buf.WriteString(value)
			}
		case map[string]interface{}:
			if len(value) == 0 {
				fmt.Fprintf(buf, "%s", colorize("NONE", cMagenta))
				continue
			}
			for k, v := range value {
				fmt.Fprintf(buf, "%s=", colorize(k, cYellow))
				fmt.Fprint(buf, v)
				fmt.Fprint(buf, " ")
			}
		default:
			fmt.Fprintf(buf, "value %T", value)
			fmt.Fprint(buf, value)
		}
	}
	buf.WriteByte('\n')
	buf.WriteTo(os.Stderr)
	n = len(p)
	return
}

func colorize(s interface{}, color int) string {
	return fmt.Sprintf("\x1b[%dm%v\x1b[0m", color, s)
}

func needsQuote(s string) bool {
	for i := range s {
		if s[i] < 0x20 || s[i] > 0x7e || s[i] == ' ' || s[i] == '\\' || s[i] == '"' {
			return true
		}
	}
	return false
}

func UseColorLogger() {
	SetLoggerOutput(ColorWriter{})
}

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
