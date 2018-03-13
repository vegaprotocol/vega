package core

import "vega/src/matching"

type Config struct {
	Matching matching.Config
}


func DefaultConfig() Config {
	return Config{
		Matching: matching.DefaultConfig(),
	}
}