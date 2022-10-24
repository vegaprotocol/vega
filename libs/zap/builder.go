package zap

import (
	"fmt"

	"go.uber.org/zap"
)

func Build(cfg zap.Config) (*zap.Logger, error) {
	log, err := cfg.Build()
	if err != nil {
		return nil, fmt.Errorf("couldn't create logger: %w", err)
	}
	return log, nil
}

func BuildJSONFileLogger(level, filePath string) (*zap.Logger, error) {
	return Build(WithJSONFormat(WithFileOutput(WithLevel(DefaultConfig(), level), filePath)))
}

func BuildStandardConsoleLogger(level string) (*zap.Logger, error) {
	return Build(WithStandardOutput(WithConsoleFormat(WithLevel(DefaultConfig(), level))))
}

func BuildStandardJSONLogger(level string) (*zap.Logger, error) {
	return Build(WithStandardOutput(WithJSONFormat(WithLevel(DefaultConfig(), level))))
}
