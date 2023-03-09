// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package staking

import (
	"context"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/logging"

	"code.vegaprotocol.io/vega/libs/proto"
)

var (
	depositedKey = (&types.PayloadStakeVerifierDeposited{}).Key()
	removedKey   = (&types.PayloadStakeVerifierRemoved{}).Key()

	hashKeys = []string{
		depositedKey,
		removedKey,
	}
)

type stakeVerifierSnapshotState struct {
	serialisedDeposited []byte
	serialisedRemoved   []byte
}

func (s *StakeVerifier) serialisePendingSD() ([]byte, error) {
	s.log.Info("serialising pending SD", logging.Int("n", len(s.pendingSDs)))
	deposited := make([]*types.StakeDeposited, 0, len(s.pendingSDs))

	for _, p := range s.pendingSDs {
		deposited = append(deposited, p.StakeDeposited)
	}

	pl := types.Payload{
		Data: &types.PayloadStakeVerifierDeposited{
			StakeVerifierDeposited: deposited,
		},
	}
	return proto.Marshal(pl.IntoProto())
}

func (s *StakeVerifier) serialisePendingSR() ([]byte, error) {
	s.log.Info("serialising pending SR", logging.Int("n", len(s.pendingSRs)))
	removed := make([]*types.StakeRemoved, 0, len(s.pendingSRs))

	for _, p := range s.pendingSRs {
		removed = append(removed, p.StakeRemoved)
	}

	pl := types.Payload{
		Data: &types.PayloadStakeVerifierRemoved{
			StakeVerifierRemoved: removed,
		},
	}

	return proto.Marshal(pl.IntoProto())
}

func (s *StakeVerifier) serialiseK(serialFunc func() ([]byte, error), dataField *[]byte) ([]byte, error) {
	data, err := serialFunc()
	if err != nil {
		return nil, err
	}
	*dataField = data
	return data, nil
}

// get the serialised form and hash of the given key.
func (s *StakeVerifier) serialise(k string) ([]byte, error) {
	switch k {
	case depositedKey:
		return s.serialiseK(s.serialisePendingSD, &s.svss.serialisedDeposited)
	case removedKey:
		return s.serialiseK(s.serialisePendingSR, &s.svss.serialisedRemoved)
	default:
		return nil, types.ErrSnapshotKeyDoesNotExist
	}
}

func (s *StakeVerifier) Namespace() types.SnapshotNamespace {
	return types.StakeVerifierSnapshot
}

func (s *StakeVerifier) Keys() []string {
	return hashKeys
}

func (s *StakeVerifier) Stopped() bool {
	return false
}

func (s *StakeVerifier) GetState(k string) ([]byte, []types.StateProvider, error) {
	data, err := s.serialise(k)
	return data, nil, err
}

func (s *StakeVerifier) LoadState(ctx context.Context, payload *types.Payload) ([]types.StateProvider, error) {
	if s.Namespace() != payload.Data.Namespace() {
		return nil, types.ErrInvalidSnapshotNamespace
	}

	switch pl := payload.Data.(type) {
	case *types.PayloadStakeVerifierDeposited:
		return nil, s.restorePendingSD(ctx, pl.StakeVerifierDeposited, payload)
	case *types.PayloadStakeVerifierRemoved:
		return nil, s.restorePendingSR(ctx, pl.StakeVerifierRemoved, payload)
	default:
		return nil, types.ErrUnknownSnapshotType
	}
}

func (s *StakeVerifier) restorePendingSD(ctx context.Context, deposited []*types.StakeDeposited, p *types.Payload) error {
	s.log.Debug("restoring pendingSDs snapshot", logging.Int("n_pending", len(deposited)))
	s.pendingSDs = make([]*pendingSD, 0, len(deposited))
	evts := []events.Event{}
	for _, d := range deposited {
		// this populates the id/hash structs
		if !s.ensureNotDuplicate(d.ID, d.IntoStakeLinking().Hash()) {
			s.log.Panic("pendingSD's unexpectedly pre-populated when restoring from snapshot")
		}

		pending := &pendingSD{
			StakeDeposited: d,
			check:          func() error { return s.ocv.CheckStakeDeposited(d) },
		}

		s.pendingSDs = append(s.pendingSDs, pending)
		s.log.Debug("restoring witness resource")
		if err := s.witness.RestoreResource(pending, s.onEventVerified); err != nil {
			s.log.Panic("unable to restore pending stake deposited resource", logging.String("id", pending.ID), logging.Error(err))
		}
		evts = append(evts, events.NewStakeLinking(ctx, *pending.IntoStakeLinking()))
	}
	var err error
	s.svss.serialisedDeposited, err = proto.Marshal(p.IntoProto())
	s.broker.SendBatch(evts)

	// now populate "seen" map with finalised events from accounting
	for _, acc := range s.accs.hashableAccounts {
		for _, evt := range acc.Events {
			sl := types.StakeLinkingFromProto(evt.IntoProto())
			if !s.ensureNotDuplicate(sl.ID, sl.Hash()) {
				s.log.Panic("finalised events unexpectedly pre-populated when restoring from snapshot")
			}
		}
	}

	return err
}

func (s *StakeVerifier) OnStateLoaded(ctx context.Context) error {
	// tell the internal EEF where it got up to so we do not resend events we're already seen
	lastBlockSeen := s.getLastBlockSeen()
	if lastBlockSeen == 0 {
		lastBlockSeen = s.accs.getLastBlockSeen()
	}
	if lastBlockSeen != 0 {
		s.log.Info("restoring staking bridge starting block", logging.Uint64("block", lastBlockSeen))
		s.ethEventSource.UpdateStakingStartingBlock(lastBlockSeen)
	}
	return nil
}

func (s *StakeVerifier) restorePendingSR(ctx context.Context, removed []*types.StakeRemoved, p *types.Payload) error {
	s.log.Debug("restoring pendingSRs snapshot", logging.Int("n_pending", len(removed)))
	s.pendingSRs = make([]*pendingSR, 0, len(removed))
	evts := []events.Event{}
	for _, r := range removed {
		// this populates the id/hash structs
		if !s.ensureNotDuplicate(r.ID, r.IntoStakeLinking().Hash()) {
			s.log.Panic("pendingSR's unexpectedly pre-populated when restoring from snapshot")
		}

		pending := &pendingSR{
			StakeRemoved: r,
			check:        func() error { return s.ocv.CheckStakeRemoved(r) },
		}

		s.pendingSRs = append(s.pendingSRs, pending)
		if err := s.witness.RestoreResource(pending, s.onEventVerified); err != nil {
			s.log.Panic("unable to restore pending stake removed resource", logging.String("id", pending.ID), logging.Error(err))
		}
		evts = append(evts, events.NewStakeLinking(ctx, *pending.IntoStakeLinking()))
	}

	var err error
	s.svss.serialisedRemoved, err = proto.Marshal(p.IntoProto())
	s.broker.SendBatch(evts)
	return err
}
