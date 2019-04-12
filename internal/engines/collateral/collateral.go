package collateral

type Engine struct {
	*Config
}

func New(conf *Config) *Engine {
	return &Engine{
		Config: conf,
	}
}
