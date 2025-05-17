package logger

import (
	"fmt"
	"os"
	"strings"

	"github.com/rs/zerolog"
)

//go:generate go run go.uber.org/mock/mockgen -source=logger.go -destination=mock/logger_mock.go -package=mock github.com/savioruz/goth/pkg/logger Interface

type Interface interface {
	Debug(message interface{}, args ...interface{})
	Info(message interface{}, args ...interface{})
	Warn(message interface{}, args ...interface{})
	Error(message interface{}, args ...interface{})
	Fatal(message interface{}, args ...interface{})
}

type Logger struct {
	logger *zerolog.Logger
}

var _ Interface = (*Logger)(nil)

func New(level string) *Logger {
	var l zerolog.Level

	switch strings.ToLower(level) {
	case "debug":
		l = zerolog.DebugLevel
	case "info":
		l = zerolog.InfoLevel
	case "warn":
		l = zerolog.WarnLevel
	case "error":
		l = zerolog.ErrorLevel
	default:
		l = zerolog.InfoLevel
	}

	zerolog.SetGlobalLevel(l)

	skipFrameCount := 3
	logger := zerolog.New(os.Stdout).With().Timestamp().CallerWithSkipFrameCount(zerolog.CallerSkipFrameCount + skipFrameCount).Logger()

	return &Logger{
		logger: &logger,
	}
}

func (l *Logger) log(message string, args ...interface{}) {
	if len(args) == 0 {
		l.logger.Info().Msg(message)
	} else {
		l.logger.Info().Msgf(message, args...)
	}
}

func (l *Logger) msg(level string, message interface{}, args ...interface{}) {
	switch msg := message.(type) {
	case error:
		l.log(msg.Error(), args...)
	case string:
		l.log(msg, args...)
	default:
		l.log(fmt.Sprintf("%s message %v has an unknown type %v", level, message, msg), args...)
	}
}

func (l *Logger) Debug(message interface{}, args ...interface{}) {
	l.msg("Debug", message, args...)
}

func (l *Logger) Info(message interface{}, args ...interface{}) {
	l.msg("Info", message, args...)
}

func (l *Logger) Warn(message interface{}, args ...interface{}) {
	l.msg("Warn", message, args...)
}

func (l *Logger) Error(message interface{}, args ...interface{}) {
	l.msg("Error", message, args...)
}

func (l *Logger) Fatal(message interface{}, args ...interface{}) {
	l.msg("Fatal", message, args...)
}
