package logger

import (
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
)

type Level string

const (
	LevelDebug Level = "debug"
	LevelInfo  Level = "info"
	LevelWarn  Level = "warn"
	LevelError Level = "error"
)

type Format string

const (
	FormatJSON   Format = "json"
	FormatPretty Format = "pretty"
)

type Config struct {
	Level  Level
	Format Format
	Output io.Writer
}

type Logger struct {
	zl zerolog.Logger
}

func New(cfg Config) *Logger {
	if cfg.Output == nil {
		cfg.Output = os.Stdout
	}

	var output io.Writer = cfg.Output

	if cfg.Format == FormatPretty {
		output = zerolog.ConsoleWriter{
			Out:        cfg.Output,
			TimeFormat: "2006-01-02 15:04:05.000",
		}
	}

	zl := zerolog.New(output).With().Timestamp().Logger()

	switch cfg.Level {
	case LevelDebug:
		zl = zl.Level(zerolog.DebugLevel)
	case LevelWarn:
		zl = zl.Level(zerolog.WarnLevel)
	case LevelError:
		zl = zl.Level(zerolog.ErrorLevel)
	default:
		zl = zl.Level(zerolog.InfoLevel)
	}

	return &Logger{zl: zl}
}

func (l *Logger) Debug(msg string) {
	l.zl.Debug().Msg(msg)
}

func (l *Logger) Info(msg string) {
	l.zl.Info().Msg(msg)
}

func (l *Logger) Warn(msg string) {
	l.zl.Warn().Msg(msg)
}

func (l *Logger) Error(msg string) {
	l.zl.Error().Msg(msg)
}

func (l *Logger) WithField(key string, value interface{}) *Logger {
	return &Logger{zl: l.zl.With().Interface(key, value).Logger()}
}

func (l *Logger) WithError(err error) *Logger {
	return &Logger{zl: l.zl.With().Err(err).Logger()}
}

func (l *Logger) Zerolog() *zerolog.Logger {
	return &l.zl
}

func init() {
	zerolog.TimeFieldFormat = time.RFC3339Nano
}
