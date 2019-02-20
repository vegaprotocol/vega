package risk

import "vega/internal/logging"

// namedLogger is the identifier for package and should ideally match the package name
// this is simply emitted as a hierarchical label e.g. 'api.grpc'.
const namedLogger = "risk"

type Config struct {
	log *logging.Logger
	Level logging.Level
	
	// If set to true, all python risk model files will be loaded via an absolute path.
	// If set to false (default) all python risk model files will be loaded via relative path to the vega binary.
	PyRiskModelAbsolutePath bool            `mapstructure:"absolute_path"`
	PyRiskModelDefaultFileName string       `mapstructure:"default_file_name"`
	PyRiskModelShortIndex int               `mapstructure:"short_index"`
	PyRiskModelLongIndex int                `mapstructure:"long_index"`
}

func NewConfig(logger *logging.Logger) *Config {
	logger = logger.Named(namedLogger)
	return &Config{
		log: logger,
		Level: logging.FatalLevel, //.InfoLevel,
		PyRiskModelDefaultFileName: "/risk-model.py",
		PyRiskModelShortIndex: 0,
		PyRiskModelLongIndex: 1,
		PyRiskModelAbsolutePath: false,
	}
}