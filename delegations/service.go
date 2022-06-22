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

package delegations

import (
	"context"
	"sync/atomic"
	"time"

	"code.vegaprotocol.io/data-node/contextutil"
	"code.vegaprotocol.io/data-node/logging"
	pb "code.vegaprotocol.io/protos/vega"
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/delegation_store_mock.go -package mocks code.vegaprotocol.io/data-node/delegations DelegationStore
type DelegationStore interface {
	GetAllDelegations(skip, limit uint64, descending bool) ([]*pb.Delegation, error)
	GetAllDelegationsOnEpoch(epochSeq string, skip, limit uint64, descending bool) ([]*pb.Delegation, error)
	GetPartyDelegations(party string, skip, limit uint64, descending bool) ([]*pb.Delegation, error)
	GetPartyDelegationsOnEpoch(party string, epochSeq string, skip, limit uint64, descending bool) ([]*pb.Delegation, error)
	GetPartyNodeDelegations(party string, node string, skip, limit uint64, descending bool) ([]*pb.Delegation, error)
	GetPartyNodeDelegationsOnEpoch(party string, node string, epochSeq string) ([]*pb.Delegation, error)
	GetNodeDelegations(nodeID string, skip, limit uint64, descending bool) ([]*pb.Delegation, error)
	GetNodeDelegationsOnEpoch(nodeID string, epochSeq string, skip, limit uint64, descending bool) ([]*pb.Delegation, error)
	Subscribe(updates chan pb.Delegation) uint64
	Unsubscribe(id uint64) error
}

// Service represent the epoch service
type Service struct {
	Config
	log             *logging.Logger
	delegationStore DelegationStore
	subscriberCnt   int32
}

// NewService creates an validators service with the necessary dependencies
func NewService(
	log *logging.Logger,
	config Config,
	delegationStore DelegationStore,
) *Service {
	// setup logger
	log = log.Named(namedLogger)
	log.SetLevel(config.Level.Get())

	return &Service{
		log:             log,
		Config:          config,
		delegationStore: delegationStore,
	}
}

// ReloadConf update the market service internal configuration
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

// GetDelegationSubscribersCount returns the total number of active subscribers for ObserveDelegations.
func (s *Service) GetDelegationSubscribersCount() int32 {
	return atomic.LoadInt32(&s.subscriberCnt)
}

//ObserveDelegations returns a channel for subscribing to delegation updates.
func (s *Service) ObserveDelegations(ctx context.Context, retries int, party, nodeID string) (delegationsCh <-chan pb.Delegation, ref uint64) {
	delegations := make(chan pb.Delegation, 10)
	internal := make(chan pb.Delegation, 10)
	ref = s.delegationStore.Subscribe(internal)

	var cancel func()
	ctx, cancel = context.WithCancel(ctx)
	go func() {
		atomic.AddInt32(&s.subscriberCnt, 1)
		defer atomic.AddInt32(&s.subscriberCnt, -1)
		ip, _ := contextutil.RemoteIPAddrFromContext(ctx)
		defer cancel()
		for {
			select {
			case <-ctx.Done():
				s.log.Debug(
					"Delegations subscriber closed connection",
					logging.Uint64("id", ref),
					logging.String("ip-address", ip),
				)
				// this error only happens when the subscriber reference doesn't exist
				// so we can still safely close the channels
				if err := s.delegationStore.Unsubscribe(ref); err != nil {
					s.log.Error(
						"Failure un-subscribing delegations subscriber when context.Done()",
						logging.Uint64("id", ref),
						logging.String("ip-address", ip),
						logging.Error(err),
					)
				}
				close(internal)
				close(delegations)
				return
			case dl := <-internal:
				// if it's not required by the filter, we're done here, otherwise try to push it into the outer channel
				success := true
				if (len(party) <= 0 || party == dl.Party) &&
					(len(nodeID) <= 0 || nodeID == dl.NodeId) {
					success = false
				}
				retryCount := retries

				for !success && retryCount >= 0 {
					select {
					case delegations <- dl:
						retryCount = retries
						s.log.Debug(
							"Delegations for subscriber sent successfully",
							logging.Uint64("ref", ref),
							logging.String("ip-address", ip),
						)
						success = true
					default:
						retryCount--
						if retryCount > 0 {
							s.log.Debug(
								"Delegations for subscriber not sent",
								logging.Uint64("ref", ref),
								logging.String("ip-address", ip))
						}
						time.Sleep(time.Duration(10) * time.Millisecond)
					}
				}
				if !success && retryCount <= 0 {
					s.log.Warn(
						"Delegations subscriber has hit the retry limit",
						logging.Uint64("ref", ref),
						logging.String("ip-address", ip),
						logging.Int("retries", retries))
					cancel()
				}

			}
		}
	}()

	return delegations, ref

}

func (s *Service) GetAllDelegations(skip, limit uint64, descending bool) ([]*pb.Delegation, error) {
	return s.delegationStore.GetAllDelegations(skip, limit, descending)
}
func (s *Service) GetAllDelegationsOnEpoch(epochSeq string, skip, limit uint64, descending bool) ([]*pb.Delegation, error) {
	return s.delegationStore.GetAllDelegationsOnEpoch(epochSeq, skip, limit, descending)
}
func (s *Service) GetPartyDelegations(party string, skip, limit uint64, descending bool) ([]*pb.Delegation, error) {
	return s.delegationStore.GetPartyDelegations(party, skip, limit, descending)
}
func (s *Service) GetPartyDelegationsOnEpoch(party string, epochSeq string, skip, limit uint64, descending bool) ([]*pb.Delegation, error) {
	return s.delegationStore.GetPartyDelegationsOnEpoch(party, epochSeq, skip, limit, descending)
}
func (s *Service) GetPartyNodeDelegations(party string, node string, skip, limit uint64, descending bool) ([]*pb.Delegation, error) {
	return s.delegationStore.GetPartyNodeDelegations(party, node, skip, limit, descending)
}
func (s *Service) GetPartyNodeDelegationsOnEpoch(party string, node string, epochSeq string) ([]*pb.Delegation, error) {
	return s.delegationStore.GetPartyNodeDelegationsOnEpoch(party, node, epochSeq)
}
func (s *Service) GetNodeDelegations(nodeID string, skip, limit uint64, descending bool) ([]*pb.Delegation, error) {
	return s.delegationStore.GetNodeDelegations(nodeID, skip, limit, descending)
}
func (s *Service) GetNodeDelegationsOnEpoch(nodeID string, epochSeq string, skip, limit uint64, descending bool) ([]*pb.Delegation, error) {
	return s.delegationStore.GetNodeDelegationsOnEpoch(nodeID, epochSeq, skip, limit, descending)
}
