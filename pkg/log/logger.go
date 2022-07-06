package log

import (
	"context"
	"errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/pkgerrors"
	"net"
	"os"
	"time"
)

const (
	RFC3339 = "2006-01-02T15:04:05.000000Z07:00"
)

type Logger struct {
	*zerolog.Logger
}

type LoggerOpts func(*Logger)

// NewLogger returns a new Logger instance
func NewLogger(opts ...LoggerOpts) Logger {

	zerolog.TimestampFunc = func() time.Time { return time.Now().UTC() }
	zerolog.TimeFieldFormat = RFC3339
	zerolog.TimestampFieldName = "time"

	zerolog.DurationFieldUnit = time.Millisecond

	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack

	//consoleWriter := zerolog.ConsoleWriter{Out: os.Stdout}
	////multi := zerolog.MultiLevelWriter(consoleWriter, os.Stdout)
	//zl := zerolog.New(consoleWriter).With().Timestamp().Logger()

	zl := zerolog.New(os.Stdout)
	l := Logger{&zl}

	for _, opt := range opts {
		opt(&l)
	}
	return l
}

func (l Logger) LogError(iom string, err error) {
	l.ErrorEvent(iom, err).
		Err(err).Msg("")
}

func (l Logger) ErrorEvent(iom string, err error) *zerolog.Event {
	e := l.Log().Str("context", iom)
	if errors.Is(err, context.Canceled) {
		e.Bool("canceled", true)
	}
	if errors.Is(err, context.DeadlineExceeded) {
		e.Bool("deadline", true)
	}

	switch err := err.(type) {
	case net.Error:
		e.Str("class", "net").
			Bool("temporary", err.Temporary()).
			Bool("timeout", err.Timeout())
	default:
		e = l.Log().Err(err)
	}
	return e
}
