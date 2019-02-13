package risk

import "vega/internal/logging"

// namedLogger is the identifier for package and should ideally match the package name
// this is simply emitted as a hierarchical label e.g. 'api.grpc'.
const namedLogger = "risk"

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

func NewConfig(logger logging.Logger) *Config {
	logger = logger.Named(namedLogger)
	level := logging.DebugLevel
	return &Config{
		log: logger,
		level: level,
		PyRiskModelDefaultFileName: "/risk-model.py",
		PyRiskModelShortIndex: 0,
		PyRiskModelLongIndex: 1,
		PyRiskModelAbsolutePath: false,
	}
}