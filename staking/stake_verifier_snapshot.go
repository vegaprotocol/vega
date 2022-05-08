package staking

import (
	"context"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/types"

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
	serialised map[string][]byte
	changed    map[string]bool
}

func (s *StakeVerifier) serialisePendingSD() ([]byte, error) {
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

// get the serialised form and hash of the given key.
func (s *StakeVerifier) serialise(k string) ([]byte, error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	if _, ok := s.keyToSerialiser[k]; !ok {
		return nil, types.ErrSnapshotKeyDoesNotExist
	}

	if !s.svss.changed[k] {
		return s.svss.serialised[k], nil
	}

	data, err := s.keyToSerialiser[k]()
	if err != nil {
		return nil, err
	}

	s.svss.serialised[k] = data
	s.svss.changed[k] = false
	return data, nil
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

func (s *StakeVerifier) HasChanged(k string) bool {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.svss.changed[k]
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
	s.svss.changed[depositedKey] = false
	s.svss.serialised[depositedKey], err = proto.Marshal(p.IntoProto())
	s.broker.SendBatch(evts)
	return err
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
	s.svss.changed[removedKey] = false
	s.svss.serialised[removedKey], err = proto.Marshal(p.IntoProto())
	s.broker.SendBatch(evts)
	return err
}
