package core

import (
	"vega/api/ssee"
	"vega/matching"
)

type Vega struct {
	config  Config
	markets map[string]*matching.OrderBook
	orders  map[string]*matching.OrderEntry
	sse     ssee.SseServer // heheheh there's totally a better way to do this.
}

type Config struct {
	Matching matching.Config
}

func New(config Config, sseServer ssee.SseServer) *Vega {
	return &Vega{
		config:  config,
		markets: make(map[string]*matching.OrderBook),
		orders:  make(map[string]*matching.OrderEntry),
		sse:     sseServer,
	}
}

func DefaultConfig() Config {
	return Config{
		Matching: matching.DefaultConfig(),
	}
}
