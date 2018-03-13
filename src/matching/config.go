package matching

type Config struct {
	Quiet bool
}

func DefaultConfig() Config {
	return Config{
		Quiet: false,
	}
}