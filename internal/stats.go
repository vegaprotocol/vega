package internal

import (
	"vega/internal/blockchain"
	"vega/internal/logging"
)

// Stats ties together all other package level application stats types.
type Stats struct {
	log        *logging.Logger
	Blockchain *blockchain.Stats
}

func NewStats(logger *logging.Logger) *Stats {
	return &Stats{
		log:        logger,
		Blockchain: blockchain.NewStats(),
	}
}
