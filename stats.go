package internal

import (
	"code.vegaprotocol.io/vega/blockchain"
	"code.vegaprotocol.io/vega/logging"
)

// Stats ties together all other package level application stats types.
type Stats struct {
	log          *logging.Logger
	Blockchain   *blockchain.Stats
	version      string
	versionHash  string
	chainVersion string
}

// NewStats instanciates a new Stats
func NewStats(logger *logging.Logger, version string, versionHash string) *Stats {
	return &Stats{
		log:         logger,
		Blockchain:  blockchain.NewStats(),
		version:     version,
		versionHash: versionHash,
	}
}

// SetChainVersion sets the version of the chain in use by vega
func (s *Stats) SetChainVersion(v string) {
	s.chainVersion = v
}

// GetChainVersion returns the version of the chain in use by vega
func (s *Stats) GetChainVersion() string {
	return s.chainVersion
}

// GetVersion return the version of vega which is currently running
func (s *Stats) GetVersion() string {
	return s.version
}

// GetVersionHash return the hash of the commit this vega
// binary was compiled from.
func (s *Stats) GetVersionHash() string {
	return s.versionHash
}
