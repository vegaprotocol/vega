package risk

import "vega/internal/logging"

type Config struct {
	log logging.Logger
	level logging.Level
	
	// If set to true, all python risk model files will be loaded via an absolute path.
	// If set to false (default) all python risk model files will be loaded via relative path to the vega binary.
	PyRiskModelAbsolutePath bool

	PyRiskModelDefaultFileName string
	PyRiskModelShortIndex int
	PyRiskModelLongIndex int
}

func NewConfig() *Config {
	level := logging.DebugLevel
	logger := logging.NewLogger()
	logger.InitConsoleLogger(level)
	logger.AddExitHandler()
	return &Config{
		log: logger,
		level: level,
		PyRiskModelDefaultFileName: "/risk-model.py",
		PyRiskModelShortIndex: 0,
		PyRiskModelLongIndex: 1,
		PyRiskModelAbsolutePath: false,
	}
}