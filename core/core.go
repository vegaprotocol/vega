package core

import (
	"vega/matching"
	"vega/proto"
)

type Vega struct {
	config                 Config
	markets                map[string]*matching.OrderBook
	OrderConfirmationChans []chan msg.OrderConfirmation
}

type Config struct {
	Matching matching.Config
}

func New(config Config) *Vega {
	return &Vega{
		config:                 config,
		markets:                make(map[string]*matching.OrderBook),
		OrderConfirmationChans: []chan msg.OrderConfirmation{},
	}
}

func DefaultConfig() Config {
	return Config{
		Matching: matching.DefaultConfig(),
	}
}
