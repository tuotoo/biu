package log

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/json-iterator/go"
	"github.com/rs/zerolog"
)

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

// ColorWriter is a writer for writing pretty log to console
type ColorWriter struct {
	WithColor bool
}

// Write implements io.Writer
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
		if w.WithColor {
			lvlColor = levelColor(l)
		}
		level = strings.ToUpper(l)[0:4]
	}
	fmt.Fprintf(buf, "%s |%s| %s",
		colorize(event[zerolog.TimestampFieldName], cDarkGray, w.WithColor),
		colorize(level, lvlColor, w.WithColor),
		colorize(event[zerolog.MessageFieldName], cReset, w.WithColor))
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
		fmt.Fprintf(buf, " %s: ", colorize(field, cCyan, w.WithColor))
		switch value := event[field].(type) {
		case string:
			if needsQuote(value) {
				buf.WriteString(strconv.Quote(value))
			} else {
				buf.WriteString(value)
			}
		case map[string]interface{}:
			if len(value) == 0 {
				fmt.Fprintf(buf, "%s", colorize("NONE", cMagenta, w.WithColor))
				continue
			}
			for k, v := range value {
				fmt.Fprintf(buf, "%s=", colorize(k, cYellow, w.WithColor))
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

func colorize(s interface{}, color int, withColor bool) string {
	if withColor {
		return fmt.Sprintf("\x1b[%dm%v\x1b[0m", color, s)
	}
	return fmt.Sprintf("%v", s)
}

func levelColor(level string) int {
	switch level {
	case "debug":
		return cMagenta
	case "info":
		return cGreen
	case "warn":
		return cYellow
	case "error", "fatal", "panic":
		return cRed
	default:
		return cReset
	}
}

func needsQuote(s string) bool {
	for i := range s {
		if s[i] < 0x20 || s[i] > 0x7e || s[i] == ' ' || s[i] == '\\' || s[i] == '"' {
			return true
		}
	}
	return false
}

// UseColorLogger will writes the logs to stderr with colorful pretty format.
func UseColorLogger() {
	SetLoggerOutput(ColorWriter{WithColor: true})
}

// UseConsoleLogger will writes the logs to stderr with pretty format without color.
func UseConsoleLogger() {
	SetLoggerOutput(ColorWriter{WithColor: false})
}

// SetLoggerOutput sets the output of logger.
func SetLoggerOutput(w io.Writer) {
	logger = logger.Output(w)
}

// SetLoggerLevel sets the log level of logger.
func SetLoggerLevel(level zerolog.Level) {
	logger = logger.Level(level)
}

type Wrap struct {
	*zerolog.Event
	Level zerolog.Level
}

// Fields is a helper function to use a map to set fields using type assertion.
func (l *Wrap) Fields(fields map[string]interface{}) *Wrap {
	l.Event = l.Event.Fields(fields)
	return l
}

// Array adds the field key with an array to the event context.
// Use zerolog.Arr() to create the array or pass a type that
// implement the LogArrayMarshaler interface.
func (l *Wrap) Array(key string, arr zerolog.LogArrayMarshaler) *Wrap {
	l.Event = l.Event.Array(key, arr)
	return l
}

// Object marshals an object that implement the LogObjectMarshaler interface.
func (l *Wrap) Object(key string, obj zerolog.LogObjectMarshaler) *Wrap {
	l.Event = l.Event.Object(key, obj)
	return l
}

// Str adds the field key with val as a string to the *Event context.
func (l *Wrap) Str(key, val string) *Wrap {
	l.Event = l.Event.Str(key, val)
	return l
}

// Strs adds the field key with vals as a []string to the *Event context.
func (l *Wrap) Strs(key string, vals []string) *Wrap {
	l.Event = l.Event.Strs(key, vals)
	return l
}

// Bytes adds the field key with val as a string to the *Event context.
//
// Runes outside of normal ASCII ranges will be hex-encoded in the resulting
// JSON.
func (l *Wrap) Bytes(key string, val []byte) *Wrap {
	l.Event = l.Event.Bytes(key, val)
	return l
}

// AnErr adds the field key with err as a string to the *Event context.
// If err is nil, no field is added.
func (l *Wrap) AnErr(key string, err error) *Wrap {
	l.Event = l.Event.AnErr(key, err)
	return l
}

// Errs adds the field key with errs as an array of strings to the *Event context.
// If err is nil, no field is added.
func (l *Wrap) Errs(key string, errs []error) *Wrap {
	l.Event = l.Event.Errs(key, errs)
	return l
}

// Err adds the field "error" with err as a string to the *Event context.
// If err is nil, no field is added.
// To customize the key name, change zerolog.ErrorFieldName.
func (l *Wrap) Err(err error) *Wrap {
	l.Event = l.Event.Err(err)
	return l
}

// Bool adds the field key with val as a bool to the *Event context.
func (l *Wrap) Bool(key string, b bool) *Wrap {
	l.Event = l.Event.Bool(key, b)
	return l
}

// Bools adds the field key with val as a []bool to the *Event context.
func (l *Wrap) Bools(key string, b []bool) *Wrap {
	l.Event = l.Event.Bools(key, b)
	return l
}

// Int adds the field key with i as a int to the *Event context.
func (l *Wrap) Int(key string, i int) *Wrap {
	l.Event = l.Event.Int(key, i)
	return l
}

// Ints adds the field key with i as a []int to the *Event context.
func (l *Wrap) Ints(key string, i []int) *Wrap {
	l.Event = l.Event.Ints(key, i)
	return l
}

// Int8 adds the field key with i as a int8 to the *Event context.
func (l *Wrap) Int8(key string, i int8) *Wrap {
	l.Event = l.Event.Int8(key, i)
	return l
}

// Ints8 adds the field key with i as a []int8 to the *Event context.
func (l *Wrap) Ints8(key string, i []int8) *Wrap {
	l.Event = l.Event.Ints8(key, i)
	return l
}

// Int16 adds the field key with i as a int16 to the *Event context.
func (l *Wrap) Int16(key string, i int16) *Wrap {
	l.Event = l.Event.Int16(key, i)
	return l
}

// Ints16 adds the field key with i as a []int16 to the *Event context.
func (l *Wrap) Ints16(key string, i []int16) *Wrap {
	l.Event = l.Event.Ints16(key, i)
	return l
}

// Int32 adds the field key with i as a int32 to the *Event context.
func (l *Wrap) Int32(key string, i int32) *Wrap {
	l.Event = l.Event.Int32(key, i)
	return l
}

// Ints32 adds the field key with i as a []int32 to the *Event context.
func (l *Wrap) Ints32(key string, i []int32) *Wrap {
	l.Event = l.Event.Ints32(key, i)
	return l
}

// Int64 adds the field key with i as a int64 to the *Event context.
func (l *Wrap) Int64(key string, i int64) *Wrap {
	l.Event = l.Event.Int64(key, i)
	return l
}

// Ints64 adds the field key with i as a []int64 to the *Event context.
func (l *Wrap) Ints64(key string, i []int64) *Wrap {
	l.Event = l.Event.Ints64(key, i)
	return l
}

// Uint adds the field key with i as a uint to the *Event context.
func (l *Wrap) Uint(key string, i uint) *Wrap {
	l.Event = l.Event.Uint(key, i)
	return l
}

// Uints adds the field key with i as a []int to the *Event context.
func (l *Wrap) Uints(key string, i []uint) *Wrap {
	l.Event = l.Event.Uints(key, i)
	return l
}

// Uint8 adds the field key with i as a uint8 to the *Event context.
func (l *Wrap) Uint8(key string, i uint8) *Wrap {
	l.Event = l.Event.Uint8(key, i)
	return l
}

// Uints8 adds the field key with i as a []int8 to the *Event context.
func (l *Wrap) Uints8(key string, i []uint8) *Wrap {
	l.Event = l.Event.Uints8(key, i)
	return l
}

// Uint16 adds the field key with i as a uint16 to the *Event context.
func (l *Wrap) Uint16(key string, i uint16) *Wrap {
	l.Event = l.Event.Uint16(key, i)
	return l
}

// Uints16 adds the field key with i as a []int16 to the *Event context.
func (l *Wrap) Uints16(key string, i []uint16) *Wrap {
	l.Event = l.Event.Uints16(key, i)
	return l
}

// Uint32 adds the field key with i as a uint32 to the *Event context.
func (l *Wrap) Uint32(key string, i uint32) *Wrap {
	l.Event = l.Event.Uint32(key, i)
	return l
}

// Uints32 adds the field key with i as a []int32 to the *Event context.
func (l *Wrap) Uints32(key string, i []uint32) *Wrap {
	l.Event = l.Event.Uints32(key, i)
	return l
}

// Uint64 adds the field key with i as a uint64 to the *Event context.
func (l *Wrap) Uint64(key string, i uint64) *Wrap {
	l.Event = l.Event.Uint64(key, i)
	return l
}

// Uints64 adds the field key with i as a []int64 to the *Event context.
func (l *Wrap) Uints64(key string, i []uint64) *Wrap {
	l.Event = l.Event.Uints64(key, i)
	return l
}

// Float32 adds the field key with f as a float32 to the *Event context.
func (l *Wrap) Float32(key string, f float32) *Wrap {
	l.Event = l.Event.Float32(key, f)
	return l
}

// Floats32 adds the field key with f as a []float32 to the *Event context.
func (l *Wrap) Floats32(key string, f []float32) *Wrap {
	l.Event = l.Event.Floats32(key, f)
	return l
}

// Float64 adds the field key with f as a float64 to the *Event context.
func (l *Wrap) Float64(key string, f float64) *Wrap {
	l.Event = l.Event.Float64(key, f)
	return l
}

// Floats64 adds the field key with f as a []float64 to the *Event context.
func (l *Wrap) Floats64(key string, f []float64) *Wrap {
	l.Event = l.Event.Floats64(key, f)
	return l
}

// Timestamp adds the current local time as UNIX timestamp to the *Event context with the "time" key.
// To customize the key name, change zerolog.TimestampFieldName.
func (l *Wrap) Timestamp() *Wrap {
	l.Event = l.Event.Timestamp()
	return l
}

// Time adds the field key with t formated as string using zerolog.TimeFieldFormat.
func (l *Wrap) Time(key string, t time.Time) *Wrap {
	l.Event = l.Event.Time(key, t)
	return l
}

// Times adds the field key with t formated as string using zerolog.TimeFieldFormat.
func (l *Wrap) Times(key string, t []time.Time) *Wrap {
	l.Event = l.Event.Times(key, t)
	return l
}

// Dur adds the field key with duration d stored as zerolog.DurationFieldUnit.
// If zerolog.DurationFieldInteger is true, durations are rendered as integer
// instead of float.
func (l *Wrap) Dur(key string, d time.Duration) *Wrap {
	l.Event = l.Event.Dur(key, d)
	return l
}

// Durs adds the field key with duration d stored as zerolog.DurationFieldUnit.
// If zerolog.DurationFieldInteger is true, durations are rendered as integer
// instead of float.
func (l *Wrap) Durs(key string, d []time.Duration) *Wrap {
	l.Event = l.Event.Durs(key, d)
	return l
}

// TimeDiff adds the field key with positive duration between time t and start.
// If time t is not greater than start, duration will be 0.
// Duration format follows the same principle as Dur().
func (l *Wrap) TimeDiff(key string, t time.Time, start time.Time) *Wrap {
	l.Event = l.Event.TimeDiff(key, t, start)
	return l
}

// Interface adds the field key with i marshaled using reflection.
func (l *Wrap) Interface(key string, i interface{}) *Wrap {
	l.Event = l.Event.Interface(key, i)
	return l
}

// Debug starts a new log with debug level.
func Debug() *Wrap {
	_, filename, line, _ := runtime.Caller(1)
	_, name := filepath.Split(filename)
	return &Wrap{
		Event: zerolog.Dict().
			Str("line", fmt.Sprintf("%s:%d", name, line)),
		Level: zerolog.DebugLevel,
	}
}

// Info starts a new log with info level.
func Info() *Wrap {
	return &Wrap{Event: zerolog.Dict(), Level: zerolog.InfoLevel}
}

// Warn starts a new log with warn level.
func Warn() *Wrap {
	return &Wrap{Event: zerolog.Dict(), Level: zerolog.WarnLevel}
}

// Error starts a new log with error level.
func Error() *Wrap {
	return &Wrap{Event: zerolog.Dict(), Level: zerolog.ErrorLevel}
}

// Fatal starts a new log with fatal level.
func Fatal() *Wrap {
	return &Wrap{Event: zerolog.Dict(), Level: zerolog.FatalLevel}
}

// Panic starts a new log with panic level.
func Panic() *Wrap {
	return &Wrap{Event: zerolog.Dict(), Level: zerolog.PanicLevel}
}

// Msg sends the *Wrap with msg added as the message field if not empty.
//
// NOTICE: once this method is called, the *Wrap should be disposed.
// Calling Msg twice can have unexpected result.
func (l Wrap) Msg(msg string) {
	switch l.Level {
	case zerolog.DebugLevel:
		logger.Debug().Dict("fields", l.Event).Msg(msg)
	case zerolog.InfoLevel:
		logger.Info().Dict("fields", l.Event).Msg(msg)
	case zerolog.WarnLevel:
		logger.Warn().Dict("fields", l.Event).Msg(msg)
	case zerolog.ErrorLevel:
		logger.Error().Dict("fields", l.Event).Msg(msg)
	case zerolog.FatalLevel:
		logger.Fatal().Dict("fields", l.Event).Msg(msg)
	case zerolog.PanicLevel:
		logger.Panic().Dict("fields", l.Event).Msg(msg)
	}
}

func (l Wrap) Msgf(format string, v ...interface{}) {
	l.Msg(fmt.Sprintf(format, v...))
}
