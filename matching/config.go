package matching

type Config struct {
	ProrataMode bool
	LogPriceLevels bool
}

func DefaultConfig() *Config {
	return &Config{ProrataMode: false}
}

func ProrataModeConfig() *Config {
	return &Config{ProrataMode: true}
}