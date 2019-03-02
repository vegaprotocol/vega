package internal

import (
	"vega/internal/blockchain"
	"vega/internal/logging"
)

// Stats ties together all other package level application stats types.
type Stats struct {
	log         *logging.Logger
	Blockchain  *blockchain.Stats
	version     string
	versionHash string
}

func NewStats(logger *logging.Logger, version string, versionHash string) *Stats {
	return &Stats{
		log:         logger,
		Blockchain:  blockchain.NewStats(),
		version:     version,
		versionHash: versionHash,
	}
}

func (s *Stats) GetVersion() string {
   return s.version
}

func (s *Stats) GetVersionHash() string {
	return s.versionHash
}