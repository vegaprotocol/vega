// Package log provides an interface to a global logger.
package log

import (
	"time"

	"github.com/tav/golly/process"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var root *zap.Logger

// A Level is a logging priority. Higher levels are more important.
type Level int8

// Logging levels.
const (
	// DebugLevel logs are typically voluminous, and are usually disabled in
	// production.
	DebugLevel Level = -1
	// InfoLevel is the default logging priority.
	InfoLevel Level = 0
	// ErrorLevel logs are high-priority. If an application is running smoothly,
	// it shouldn't generate any error-level logs.
	ErrorLevel Level = 2
	// FatalLevel logs a message, then calls os.Exit(1).
	FatalLevel Level = 5
)

func buildCfg(path string, lvl Level) (*zap.Logger, error) {
	enc := zapcore.EncoderConfig{
		CallerKey:      "",
		EncodeCaller:   zapcore.ShortCallerEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeLevel:    zapcore.CapitalColorLevelEncoder,
		EncodeTime:     encTime,
		LevelKey:       "L",
		LineEnding:     zapcore.DefaultLineEnding,
		MessageKey:     "M",
		NameKey:        "N",
		StacktraceKey:  "",
		TimeKey:        "T",
	}
	cfg := zap.Config{
		Development:      true,
		Encoding:         "console",
		EncoderConfig:    enc,
		ErrorOutputPaths: []string{path},
		Level:            zap.NewAtomicLevelAt(zapcore.Level(lvl)),
		OutputPaths:      []string{path},
	}
	return cfg.Build()
}

func encTime(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Format("[2006-01-02 15:04:05]"))
}

func setLogger(path string, lvl Level) error {
	l, err := buildCfg(path, lvl)
	if err != nil {
		return err
	}
	if root == nil {
		root = l
		return nil
	}
	c := l.Core()
	wrap := zap.WrapCore(func(r zapcore.Core) zapcore.Core {
		return zapcore.NewTee(r, c)
	})
	root = root.WithOptions(wrap)
	return nil
}

// Debugf formats the given arguments and outputs them to the global logger at
// the DebugLevel.
func Debugf(format string, args ...interface{}) {
	root.Sugar().Debugf(format, args...)
}

// Errorf formats the given arguments and outputs them to the global logger at
// the ErrorLevel.
func Errorf(format string, args ...interface{}) {
	root.Sugar().Errorf(format, args...)
}

// Fatalf formats the given arguments and outputs them to the global logger,
// before calling os.Exit(1).
func Fatalf(format string, args ...interface{}) {
	root.Sugar().Fatalf(format, args...)
}

// Infof formats the given arguments and outputs them to the global logger at
// the InfoLevel.
func Infof(format string, args ...interface{}) {
	root.Sugar().Infof(format, args...)
}

// InitConsoleLogger initialises a console logger configured with defaults for
// use as the root logger. If a root logger already exists, it will be tee-d
// together with the new console logger.
func InitConsoleLogger(lvl Level) error {
	return setLogger("stderr", lvl)
}

// InitFileLogger initialises a file logger configured with defaults for use as
// the root logger. If a root logger already exists, it will be tee-d together
// with the new file logger.
func InitFileLogger(path string, lvl Level) error {
	return setLogger(path, lvl)
}

func init() {
	// Flush the logs before exiting the process.
	process.SetExitHandler(func() {
		if root != nil {
			root.Sync()
		}
	})
}
