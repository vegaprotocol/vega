package logging

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"time"
	"github.com/tav/golly/process"
)

type Logger interface {
	AddExitHandler()

	InitConsoleLogger(lvl Level) error
	InitFileLogger(path string, lvl Level) error

	Debug(args ...interface{})
	Info(args ...interface{})
	Error(args ...interface{})
	Fatal(args ...interface{})

	Debugf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Errorf(format string, args ...interface{})
	Fatalf(format string, args ...interface{})
}

// A Level is a logging priority. Higher levels are more important.
type Level int8

// Logging levels.
const (
	// DebugLevel logs are typically voluminous. Usually disabled in production.
	DebugLevel Level = -1
	// InfoLevel is the default logging priority.
	InfoLevel Level = 0
	// ErrorLevel logs are high-priority. If an application is running smoothly,
	// it shouldn't generate any error-level logs.
	ErrorLevel Level = 2
	// FatalLevel logs a message, then calls os.Exit(1).
	FatalLevel Level = 5
)

type logger struct {
	root *zap.Logger
	id string
}

func NewLogger() Logger {
	return &logger{}
}

// InitConsoleLogger initialises a console logger configured with defaults for
// use as the root logger. If a root logger already exists, it will be tee-d
// together with the new console logger.
func (l *logger) InitConsoleLogger(lvl Level) error {
	return l.setLogger("stderr", lvl)
}

// InitFileLogger initialises a file logger configured with defaults for use as
// the root logger. If a root logger already exists, it will be tee-d together
// with the new file logger.
func (l *logger) InitFileLogger(path string, lvl Level) error {
	return l.setLogger(path, lvl)
}

// AddExitHandler flushes the logs before exiting the process. Useful when an
// app shuts down so we store all logging possible.
func (l *logger) AddExitHandler() {
	// Flush the logs before exiting the process.
	process.SetExitHandler(func() {
		if l.root != nil {
			l.root.Sync()
		}
	})
}

// Debug sends the given arguments to the logger at DebugLevel.
func (l *logger) Debug(args ...interface{}) {
	l.root.Sugar().Debug(args)
}
// Info sends the given arguments to the logger at InfoLevel.
func (l *logger) Info(args ...interface{}) {
	l.root.Sugar().Info(args)
}
// Error sends the given arguments to the logger at ErrorLevel.
func (l *logger) Error(args ...interface{}) {
	l.root.Sugar().Error(args)
}
// Fatal sends the given arguments to the logger at FatalLevel.
func (l *logger) Fatal(args ...interface{}) {
	l.root.Sugar().Fatal(args)
}

// Debugf formats the given arguments and sends them to the logger at DebugLevel.
func (l *logger) Debugf(format string, args ...interface{}) {
	l.root.Sugar().Debugf(format, args...)
}

// Errorf formats the given arguments and sends them to the logger at ErrorLevel.
func (l *logger) Errorf(format string, args ...interface{}) {
	l.root.Sugar().Errorf(format, args...)
}

// Fatalf formats the given arguments and sends them to the logger, before calling os.Exit(1).
func (l *logger) Fatalf(format string, args ...interface{}) {
	l.root.Sugar().Fatalf(format, args...)
}

// Infof formats the given arguments and sends them to the logger at InfoLevel.
func (l *logger) Infof(format string, args ...interface{}) {
	l.root.Sugar().Infof(format, args...)
}


func (l *logger) buildCfg(path string, lvl Level) (*zap.Logger, error) {
	enc := zapcore.EncoderConfig{
		CallerKey:      "",
		EncodeCaller:   zapcore.ShortCallerEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeLevel:    zapcore.CapitalColorLevelEncoder,
		EncodeTime:     l.encTime,
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

func (l *logger) encTime(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Format("[2006-01-02 15:04:05.000]"))
}

func (l *logger) setLogger(path string, lvl Level) error {
	i, err := l.buildCfg(path, lvl)
	if err != nil {
		return err
	}
	if l.root == nil {
		l.root = i
		return nil
	}
	c := i.Core()
	wrap := zap.WrapCore(func(r zapcore.Core) zapcore.Core {
		return zapcore.NewTee(r, c)
	})
	l.root = l.root.WithOptions(wrap)
	return nil
}
