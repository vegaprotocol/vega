package zap

import (
	"fmt"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var SupportedLogLevels = []string{
	zapcore.DebugLevel.String(),
	zapcore.InfoLevel.String(),
	zapcore.WarnLevel.String(),
	zapcore.ErrorLevel.String(),
}

func IsSupportedLogLevel(level string) bool {
	for _, supported := range SupportedLogLevels {
		if level == supported {
			return true
		}
	}
	return false
}

func EnsureIsSupportedLogLevel(level string) error {
	if !IsSupportedLogLevel(level) {
		return fmt.Errorf("unsupported log level %q, supported levels: %s", level, strings.Join(SupportedLogLevels, ", "))
	}
	return nil
}

func parseLevel(level string) (zap.AtomicLevel, error) {
	if err := EnsureIsSupportedLogLevel(level); err != nil {
		return zap.AtomicLevel{}, err
	}

	l := new(zapcore.Level)

	if err := l.UnmarshalText([]byte(level)); err != nil {
		return zap.AtomicLevel{}, fmt.Errorf("couldn't parse log level: %w", err)
	}

	return zap.NewAtomicLevelAt(*l), nil
}
