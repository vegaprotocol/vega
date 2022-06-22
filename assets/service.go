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

package assets

import (
	"code.vegaprotocol.io/data-node/logging"
	types "code.vegaprotocol.io/protos/vega"
)

// Plugin Exports functions for fetching assets
//go:generate go run github.com/golang/mock/mockgen -destination mocks/plugin_mock.go -package mocks code.vegaprotocol.io/data-node/assets Plugin
type Plugin interface {
	GetByID(string) (*types.Asset, error)
	GetAll() []types.Asset
}

// Svc is governance service, responsible for managing proposals and votes.
type Svc struct {
	cfg Config
	log *logging.Logger
	p   Plugin
}

// NewService creates new governance service instance
func NewService(log *logging.Logger, cfg Config, plugin Plugin) *Svc {
	log = log.Named(namedLogger)
	log.SetLevel(cfg.Level.Get())

	return &Svc{
		cfg: cfg,
		log: log,
		p:   plugin,
	}
}

// ReloadConf updates the internal configuration of the collateral engine
func (s *Svc) ReloadConf(cfg Config) {
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

func (s *Svc) GetByID(id string) (*types.Asset, error) {
	return s.p.GetByID(id)
}

func (s *Svc) GetAll() ([]types.Asset, error) {
	return s.p.GetAll(), nil
}
