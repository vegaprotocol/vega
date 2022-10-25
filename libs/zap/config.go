package zap

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	vgfs "code.vegaprotocol.io/vega/libs/fs"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func DefaultConfig() zap.Config {
	return zap.Config{
		Level:    zap.NewAtomicLevelAt(zapcore.InfoLevel),
		Encoding: "json",
		EncoderConfig: zapcore.EncoderConfig{
			MessageKey:     "message",
			LevelKey:       "level",
			TimeKey:        "@timestamp",
			NameKey:        "logger",
			CallerKey:      "caller",
			StacktraceKey:  "stacktrace",
			LineEnding:     "\n",
			EncodeLevel:    zapcore.LowercaseLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.StringDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
			EncodeName:     zapcore.FullNameEncoder,
		},
		OutputPaths:       []string{"stdout"},
		ErrorOutputPaths:  []string{"stderr"},
		DisableStacktrace: true,
	}
}

func WithLevel(cfg zap.Config, level string) zap.Config {
	parsedLevel, err := parseLevel(level)
	if err != nil {
		parsedLevel = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	}

	cfg.Level = parsedLevel

	return cfg
}

func WithFileOutputForDedicatedProcess(cfg zap.Config, dirPath string) zap.Config {
	date := time.Now().UTC().Format("2006-01-02-15-04-05")
	pid := os.Getpid()
	logFileName := fmt.Sprintf("%s-%d.log", date, pid)
	logFilePath := filepath.Join(dirPath, logFileName)

	return WithFileOutput(cfg, logFilePath)
}

func WithFileOutput(cfg zap.Config, filePath string) zap.Config {
	zapLogPath := toOSFilePath(filePath)

	fileDir, _ := filepath.Split(filePath)
	_ = vgfs.EnsureDir(fileDir)

	cfg.OutputPaths = []string{zapLogPath}
	cfg.ErrorOutputPaths = []string{zapLogPath}

	return cfg
}

func WithStandardOutput(cfg zap.Config) zap.Config {
	cfg.OutputPaths = []string{"stdout"}
	cfg.ErrorOutputPaths = []string{"stderr"}

	return cfg
}

func WithJSONFormat(cfg zap.Config) zap.Config {
	cfg.EncoderConfig.EncodeLevel = zapcore.LowercaseLevelEncoder
	cfg.Encoding = "json"

	return cfg
}

func WithConsoleFormat(cfg zap.Config) zap.Config {
	cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	cfg.Encoding = "console"

	return cfg
}
