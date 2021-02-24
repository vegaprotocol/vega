package logging

import (
	"errors"
	"fmt"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	// ErrInvalidLogLevel signal that the log level used is not valid
	// as cannot be unmarshal from a string, or not one of the level
	// provided in this package
	ErrInvalidLogLevel = errors.New("invalid log level")
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

// ParseLevel parse a log level from a string
func ParseLevel(l string) (Level, error) {
	l = strings.ToLower(l)
	switch l {
	case "debug":
		return DebugLevel, nil
	case "info":
		return InfoLevel, nil
	case "warning":
		return WarnLevel, nil
	case "error":
		return ErrorLevel, nil
	case "panic":
		return PanicLevel, nil
	case "fatal":
		return FatalLevel, nil
	default:
		return Level(100), ErrInvalidLogLevel
	}
}

// String marshal a log level to a string representation
func (l Level) String() string {
	switch l {
	case DebugLevel:
		return "Debug"
	case InfoLevel:
		return "Info"
	case WarnLevel:
		return "Warning"
	case ErrorLevel:
		return "Error"
	case PanicLevel:
		return "Panic"
	case FatalLevel:
		return "Fatal"
	default:
		return "Unknow"
	}
}

// ZapLevel return the log level of internal zap level
func (l *Level) ZapLevel() zapcore.Level {
	return zapcore.Level(*l)
}

// Logger is an abstraction on to of the zap logger
type Logger struct {
	*zap.Logger
	config      *zap.Config
	environment string
	name        string
}

// Clone will clone the internal logger
func (log *Logger) Clone() *Logger {
	newConfig := cloneConfig(log.config)
	newLogger, err := newConfig.Build()
	if err != nil {
		panic(err)
	}
	return New(newLogger, newConfig, log.environment, log.name)
}

// GetLevel returns the log level
func (log *Logger) GetLevel() Level {
	return (Level)(log.config.Level.Level())
}

// IsDebug returns true if logger level is less or equal to DebugLevel.
func (log *Logger) IsDebug() bool {
	return log.GetLevel() <= DebugLevel
}

// GetLevelString return a string representation of the current
// log level
func (log *Logger) GetLevelString() string {
	return log.config.Level.String()
}

// GetEnvironment returns the current environment name
func (log *Logger) GetEnvironment() string {
	return log.environment
}

// GetName return the name of this logger
func (log *Logger) GetName() string {
	return log.name
}

// Named instantiate a new logger by cloning it first
// and name it with the string specified
func (log *Logger) Named(name string) *Logger {
	c := log.Clone()
	newName := ""
	if log.name == "" {
		newName = name
	} else {
		newName = fmt.Sprintf("%s.%s", log.name, name)
	}
	c.Logger = c.Logger.Named(newName)
	c.name = newName
	return c
}

// New instantiate a new logger
func New(zaplogger *zap.Logger, zapconfig *zap.Config, environment, name string) *Logger {
	return &Logger{
		Logger:      zaplogger,
		config:      zapconfig,
		environment: environment,
		name:        name,
	}
}

// SetLevel change the level of this logger
func (log *Logger) SetLevel(level Level) {
	lvl := (zapcore.Level)(level)
	if log.config.Level.Level() == lvl {
		return
	}
	log.config.Level.SetLevel(lvl)
}

// With will add default field to each logs
func (log *Logger) With(fields ...zap.Field) *Logger {
	c := log.Clone()
	c.Logger = c.Logger.With(fields...)
	return c
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

// newLoggerFromConfig creates a logger according to the given custom config.
func newLoggerFromConfig(config Config) *Logger {
	encoderConfig := zapcore.EncoderConfig{
		CallerKey:  config.Custom.ZapEncoder.CallerKey,
		LevelKey:   config.Custom.ZapEncoder.LevelKey,
		LineEnding: config.Custom.ZapEncoder.LineEnding,
		MessageKey: config.Custom.ZapEncoder.MessageKey,
		NameKey:    config.Custom.ZapEncoder.NameKey,
		TimeKey:    config.Custom.ZapEncoder.TimeKey,
	}

	encoderConfig.EncodeCaller.UnmarshalText([]byte(config.Custom.ZapEncoder.EncodeCaller))
	encoderConfig.EncodeDuration.UnmarshalText([]byte(config.Custom.ZapEncoder.EncodeDuration))
	encoderConfig.EncodeLevel.UnmarshalText([]byte(config.Custom.ZapEncoder.EncodeLevel))
	encoderConfig.EncodeName.UnmarshalText([]byte(config.Custom.ZapEncoder.EncodeName))
	encoderConfig.EncodeTime.UnmarshalText([]byte(config.Custom.ZapEncoder.EncodeTime))

	zapconfig := zap.Config{
		Level:            zap.NewAtomicLevelAt(zapcore.Level(config.Custom.Zap.Level)),
		Development:      config.Custom.Zap.Development,
		Encoding:         config.Custom.Zap.Encoding,
		EncoderConfig:    encoderConfig,
		OutputPaths:      config.Custom.Zap.OutputPaths,
		ErrorOutputPaths: config.Custom.Zap.ErrorOutputPaths,
	}

	zaplogger, err := zapconfig.Build()
	if err != nil {
		panic(err)
	}
	return New(zaplogger, &zapconfig, config.Environment, "")
}

// NewDevLogger creates a new logger suitable for development environments.
func NewDevLogger() *Logger {
	config := Config{
		Environment: "dev",
		Custom: &Custom{
			Zap: &Zap{
				Development:      true,
				Encoding:         "console",
				Level:            DebugLevel,
				OutputPaths:      []string{"stdout"},
				ErrorOutputPaths: []string{"stderr"},
			},
			ZapEncoder: &ZapEncoder{
				CallerKey:      "C",
				EncodeCaller:   "short",
				EncodeDuration: "string",
				EncodeLevel:    "capital",
				EncodeName:     "full",
				EncodeTime:     "iso8601",
				LevelKey:       "L",
				LineEnding:     "\n",
				MessageKey:     "M",
				NameKey:        "N",
				TimeKey:        "T",
			},
		},
	}
	return newLoggerFromConfig(config)
}

// NewTestLogger creates a new logger suitable for golang unit test
// environments, ie when running "go test ./..."
func NewTestLogger() *Logger {
	config := Config{
		Environment: "test",
		Custom: &Custom{
			Zap: &Zap{
				Development:      true,
				Encoding:         "console",
				Level:            DebugLevel,
				OutputPaths:      []string{"stdout"},
				ErrorOutputPaths: []string{"stderr"},
			},
			ZapEncoder: &ZapEncoder{
				CallerKey:      "C",
				EncodeCaller:   "short",
				EncodeDuration: "string",
				EncodeLevel:    "capital",
				EncodeName:     "full",
				EncodeTime:     "iso8601",
				LevelKey:       "L",
				LineEnding:     "\n",
				MessageKey:     "M",
				NameKey:        "N",
				TimeKey:        "T",
			},
		},
	}
	return newLoggerFromConfig(config)
}

// NewProdLogger creates a new logger suitable for production environments,
// including sending logs to ElasticSearch.
func NewProdLogger() *Logger {
	config := Config{
		Environment: "prod",
		Custom: &Custom{
			Zap: &Zap{
				Development:      false,
				Encoding:         "json",
				Level:            InfoLevel,
				OutputPaths:      []string{"stdout"},
				ErrorOutputPaths: []string{"stderr"},
			},
			ZapEncoder: &ZapEncoder{
				CallerKey:      "caller",
				EncodeCaller:   "short",
				EncodeDuration: "string",
				EncodeLevel:    "lowercase",
				EncodeName:     "full",
				EncodeTime:     "iso8601",
				LevelKey:       "level",
				LineEnding:     "\n",
				MessageKey:     "message",
				NameKey:        "logger",
				TimeKey:        "@timestamp",
			},
		},
	}
	return newLoggerFromConfig(config)
}

// NewLoggerFromConfig creates a standard or custom logger.
func NewLoggerFromConfig(config Config) *Logger {
	switch config.Environment {
	case "dev":
		return NewDevLogger()
	case "test":
		return NewTestLogger()
	case "prod":
		return NewProdLogger()
	case "custom":
		return newLoggerFromConfig(config)
	}

	// Default:
	return NewDevLogger()
}

// Check helps avoid spending CPU time on log entries that will never be printed.
func (log *Logger) Check(l Level) bool {
	return log.Logger.Check(l.ZapLevel(), "") != nil
}

// Errorf implement badger interface
func (log *Logger) Errorf(s string, args ...interface{}) {
	if ce := log.Logger.Check(zap.ErrorLevel, ""); ce != nil {
		log.Logger.WithOptions(zap.AddCallerSkip(2)).Sugar().Errorf(strings.TrimSpace(s), args...)
	}
}

// Warningf implement badger interface
func (log *Logger) Warningf(s string, args ...interface{}) {
	if ce := log.Logger.Check(zap.WarnLevel, ""); ce != nil {
		log.Logger.WithOptions(zap.AddCallerSkip(2)).Sugar().Warnf(strings.TrimSpace(s), args...)
	}
}

// Infof implement badger interface
func (log *Logger) Infof(s string, args ...interface{}) {
	if ce := log.Logger.Check(zap.InfoLevel, ""); ce != nil {
		log.Logger.WithOptions(zap.AddCallerSkip(2)).Sugar().Infof(strings.TrimSpace(s), args...)
	}
}

// Debugf implement badger interface
func (log *Logger) Debugf(s string, args ...interface{}) {
	if ce := log.Logger.Check(zap.DebugLevel, ""); ce != nil {
		log.Logger.WithOptions(zap.AddCallerSkip(2)).Sugar().Debugf(strings.TrimSpace(s), args...)
	}
}
