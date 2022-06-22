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

package epochs

import (
	"context"
	"time"

	"code.vegaprotocol.io/data-node/logging"
	pb "code.vegaprotocol.io/protos/vega"
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/epoch_store_mock.go -package mocks code.vegaprotocol.io/data-node/epochs EpochStore
type EpochStore interface {
	GetTotalNodesUptime() time.Duration
	GetEpochByID(id string) (*pb.Epoch, error)
	GetEpoch() (*pb.Epoch, error)
}

// Service represent the epoch service
type Service struct {
	Config
	log        *logging.Logger
	epochStore EpochStore
}

// NewService creates an epoch service with the necessary dependencies
func NewService(
	log *logging.Logger,
	config Config,
	epochStore EpochStore,
) *Service {
	// setup logger
	log = log.Named(namedLogger)
	log.SetLevel(config.Level.Get())

	return &Service{
		log:        log,
		Config:     config,
		epochStore: epochStore,
	}
}

// ReloadConf update the epoch service internal configuration
func (s *Service) ReloadConf(cfg Config) {
	s.log.Info("reloading configuration")
	if s.log.GetLevel() != cfg.Level.Get() {
		s.log.Info("updating log level",
			logging.String("old", s.log.GetLevel().String()),
			logging.String("new", cfg.Level.String()),
		)
		s.log.SetLevel(cfg.Level.Get())
	}

	s.Config = cfg
}

func (s *Service) GetEpochByID(ctx context.Context, id string) (*pb.Epoch, error) {
	return s.epochStore.GetEpochByID(id)
}

func (s *Service) GetEpoch(ctx context.Context) (*pb.Epoch, error) {
	return s.epochStore.GetEpoch()
}
