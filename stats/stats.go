// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package stats

import (
	"time"

	"code.vegaprotocol.io/vega/logging"
)

// Stats ties together all other package level application stats types.
type Stats struct {
	log          *logging.Logger
	cfg          Config
	Blockchain   *Blockchain
	version      string
	versionHash  string
	chainVersion string
	uptime       time.Time
}

// New instantiates a new Stats.
func New(log *logging.Logger, cfg Config, version string, versionHash string) *Stats {
	log = log.Named(namedLogger)
	log.SetLevel(cfg.Level.Get())
	return &Stats{
		log:         log,
		cfg:         cfg,
		Blockchain:  &Blockchain{},
		version:     version,
		versionHash: versionHash,
		uptime:      time.Now(),
	}
}

// ReloadConf updates the internal configuration.
func (s *Stats) ReloadConf(cfg Config) {
	s.log.Info("reloading configuration")
	if s.log.GetLevel() != cfg.Level.Get() {
		s.log.Info("updating log level",
			logging.String("old", s.log.GetLevel().String()),
			logging.String("new", cfg.Level.String()),
		)
		s.log.SetLevel(cfg.Level.Get())
	}

	s.cfg = cfg
}

// SetChainVersion sets the version of the chain in use by vega.
func (s *Stats) SetChainVersion(v string) {
	s.chainVersion = v
}

// GetChainVersion returns the version of the chain in use by vega.
func (s *Stats) GetChainVersion() string {
	return s.chainVersion
}

// GetVersion return the version of vega which is currently running.
func (s *Stats) GetVersion() string {
	return s.version
}

// GetVersionHash return the hash of the commit this vega
// binary was compiled from.
func (s *Stats) GetVersionHash() string {
	return s.versionHash
}

func (s *Stats) GetUptime() time.Time {
	return s.uptime
}

func (s Stats) Height() uint64 {
	return s.Blockchain.Height()
}

func (s Stats) BlockHash() string {
	return s.Blockchain.Hash()
}
