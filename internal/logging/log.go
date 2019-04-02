package logging

import (
	"context"
	"fmt"
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// A Level is a logging priority. Higher levels are more important.
type Level int8

// Logging levels (matching zap core internals).
const (
	// DebugLevel logs are typically voluminous, and are usually disabled in
	// production.
	DebugLevel Level = -1
	// InfoLevel is the default logging priority.
	InfoLevel Level = 0
	// WarnLevel logs are more important than Info, but don't need individual
	// human review.
	WarnLevel Level = 1
	// ErrorLevel logs are high-priority. If an application is running smoothly,
	// it shouldn't generate any error-level logs.
	ErrorLevel Level = 2
	// PanicLevel logs a message, then panics.
	PanicLevel Level = 4
	// FatalLevel logs a message, then calls os.Exit(1).
	FatalLevel Level = 5
)

func (l *Level) ZapLevel() zapcore.Level {
	return zapcore.Level(*l)
}

type Logger struct {
	*zap.Logger
	config *zap.Config
	name   string
}

func (log *Logger) Clone() *Logger {
	newConfig := cloneConfig(log.config)
	newLogger, err := newConfig.Build()
	if err != nil {
		panic(err)
	}
	return &Logger{
		Logger: newLogger,
		config: newConfig,
	}
}

func (log *Logger) GetLevel() Level {
	return (Level)(log.config.Level.Level())
}

func (log *Logger) GetLevelString() string {
	return log.config.Level.String()
}

func (log *Logger) GetName() string {
	return log.name
}

func (log *Logger) Named(name string) *Logger {
	c := log.Clone()
	newName := ""
	if log.name == "" {
		newName = name
	} else {
		newName = fmt.Sprintf("%s.%s", log.name, name)
	}
	return &Logger{
		Logger: c.Logger.Named(newName),
		config: c.config,
		name:   newName,
	}
}

func New(core *zapcore.Core, cfg *zap.Config) *Logger {
	logger := Logger{
		Logger: zap.New(*core),
		config: cfg,
		name:   "",
	}
	return &logger
}

func (log *Logger) SetLevel(level Level) {
	lvl := (zapcore.Level)(level)
	if log.config.Level.Level() == lvl {
		return
	}
	log.config.Level.SetLevel(lvl)
}

func (log *Logger) With(fields ...zap.Field) *Logger {
	c := log.Clone()
	return &Logger{
		Logger: c.Logger.With(fields...),
		config: c.config,
	}
}

// AtExit flushes the logs before exiting the process. Useful when an
// app shuts down so we store all logging possible. This is meant to be used
// with defer when initializing your logger
func (log *Logger) AtExit() {
	if log.Logger != nil {
		log.Logger.Sync()
	}
}

func cloneConfig(cfg *zap.Config) *zap.Config {
	c := zap.Config{
		Level:             zap.NewAtomicLevelAt(cfg.Level.Level()),
		Development:       cfg.Development,
		DisableCaller:     cfg.DisableCaller,
		DisableStacktrace: cfg.DisableStacktrace,
		Sampling:          nil,
		Encoding:          cfg.Encoding,
		EncoderConfig:     cfg.EncoderConfig,
		OutputPaths:       cfg.OutputPaths,
		ErrorOutputPaths:  cfg.ErrorOutputPaths,
		InitialFields:     make(map[string]interface{}),
	}
	for k, v := range cfg.InitialFields {
		c.InitialFields[k] = v
	}
	if cfg.Sampling != nil {
		c.Sampling = &zap.SamplingConfig{
			Initial:    cfg.Sampling.Initial,
			Thereafter: cfg.Sampling.Thereafter,
		}
	}
	return &c
}

func NewLoggerFromEnv(env string) *Logger {
	var encoderConfig zapcore.EncoderConfig
	var encoder zapcore.Encoder
	var config zap.Config
	var level zapcore.Level
	/*
		Choices: (with "*" for default)
		CallerEncoder: full*
		DurationEncoder: nanos, seconds*, string
		LevelEncoder: capital, capitalColor, color, lowercase*
		NameEncoder: full*
		TimeEncoder: epoch*, iso8601, millis, nanos
	*/
	switch env {
	case "dev":
		encoderConfig = zapcore.EncoderConfig{
			CallerKey:      "C",
			EncodeCaller:   zapcore.ShortCallerEncoder,
			EncodeDuration: zapcore.StringDurationEncoder,
			EncodeLevel:    zapcore.CapitalLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			LevelKey:       "L",
			LineEnding:     "\n",
			MessageKey:     "M",
			NameKey:        "N",
			TimeKey:        "T",
		}
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
		level = zapcore.Level(DebugLevel)
		config = zap.Config{
			Level:            zap.NewAtomicLevelAt(level),
			Development:      true,
			Encoding:         "console",
			EncoderConfig:    encoderConfig,
			OutputPaths:      []string{"stdout"},
			ErrorOutputPaths: []string{"stderr"},
		}
	default:
		encoderConfig = zapcore.EncoderConfig{
			CallerKey:      "caller",
			EncodeCaller:   zapcore.ShortCallerEncoder,
			EncodeDuration: zapcore.SecondsDurationEncoder,
			EncodeLevel:    zapcore.LowercaseLevelEncoder,
			EncodeName:     zapcore.FullNameEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			LevelKey:       "level",
			LineEnding:     "\n",
			MessageKey:     "message",
			NameKey:        "logger",
			StacktraceKey:  "stacktrace",
			TimeKey:        "@timestamp",
		}
		encoder = zapcore.NewJSONEncoder(encoderConfig)
		level = zapcore.Level(InfoLevel)
		config = zap.Config{
			Level:            zap.NewAtomicLevelAt(level),
			Development:      false,
			Encoding:         "json",
			EncoderConfig:    encoderConfig,
			OutputPaths:      []string{"stdout"},
			ErrorOutputPaths: []string{"stderr"},
		}
	}

	core := zapcore.NewCore(encoder, os.Stdout, level)
	return New(&core, &config)
}

// IPAddressFromContext will attempt to access the 'remote-ip-addr' value
// that we inject into a calling context via a pipelined handlers. Only
// GraphQL API supported at present.
func IPAddressFromContext(ctx context.Context) string {
	if ctx.Value("remote-ip-addr") != nil {
		return ctx.Value("remote-ip-addr").(string)
	}
	return ""
}

// Errorf implement badger interface
func (log *Logger) Errorf(s string, args ...interface{}) {
	log.Logger.WithOptions(zap.AddCallerSkip(2)).Sugar().Errorf(strings.TrimSpace(s), args...)
}

// Warningf implement badger interface
func (log *Logger) Warningf(s string, args ...interface{}) {
	log.Logger.WithOptions(zap.AddCallerSkip(2)).Sugar().Warnf(strings.TrimSpace(s), args...)
}

// Infof implement badger interface
func (log *Logger) Infof(s string, args ...interface{}) {
	log.Logger.WithOptions(zap.AddCallerSkip(2)).Sugar().Infof(strings.TrimSpace(s), args...)
}

// Debugf implement badger interface
func (log *Logger) Debugf(s string, args ...interface{}) {
	log.Logger.WithOptions(zap.AddCallerSkip(2)).Sugar().Debugf(strings.TrimSpace(s), args...)
}
