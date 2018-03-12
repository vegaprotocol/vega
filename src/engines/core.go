package engines

import (
	"vega/src/matching"
)

type Vega struct {
	markets map[string]*matching.OrderBook
	orders  map[string]*matching.OrderEntry
}

func New() *Vega {
	return &Vega{
		markets: make(map[string]*matching.OrderBook),
		orders:  make(map[string]*matching.OrderEntry),
	}
}

