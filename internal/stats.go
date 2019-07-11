package internal

import (
	"code.vegaprotocol.io/vega/internal/blockchain"
	"code.vegaprotocol.io/vega/internal/logging"
)

// Stats ties together all other package level application stats types.
type Stats struct {
	log          *logging.Logger
	Blockchain   *blockchain.Stats
	version      string
	versionHash  string
	chainVersion string
}

func NewStats(logger *logging.Logger, version string, versionHash string) *Stats {
	return &Stats{
		log:         logger,
		Blockchain:  blockchain.NewStats(),
		version:     version,
		versionHash: versionHash,
	}
}

func (s *Stats) SetChainVersion(v string) {
	s.chainVersion = v
}

func (s *Stats) GetChainVersion() string {
	return s.chainVersion
}

func (s *Stats) GetVersion() string {
	return s.version
}

func (s *Stats) GetVersionHash() string {
	return s.versionHash
}
