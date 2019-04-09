package logging

// Config contains the configurable items for this package
type Config struct {
	Environment string
}

// NewDefaultConfig creates an instance of the package-specific configuration, given a
// pointer to a logger instance to be used for logging within the package.
func NewDefaultConfig() *Config {
	return &Config{
		Environment: "dev",
	}
}
