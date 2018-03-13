package core

import (
	"vega/src/matching"
)

type Vega struct {
	config  Config
	markets map[string]*matching.OrderBook
	orders  map[string]*matching.OrderEntry
}

func New(config Config) *Vega {
	return &Vega{
		config:  config,
		markets: make(map[string]*matching.OrderBook),
		orders:  make(map[string]*matching.OrderEntry),
	}
}
