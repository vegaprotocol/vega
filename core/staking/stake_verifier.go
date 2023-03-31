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
	"errors"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/core/validators"
	"code.vegaprotocol.io/vega/logging"
)

const (
	// 3 weeks, duration of the whole network at first?
	timeTilCancel = 24 * 21 * time.Hour
)

var (
	ErrNoStakeDepositedEventFound    = errors.New("no stake deposited event found")
	ErrNoStakeRemovedEventFound      = errors.New("no stake removed event found")
	ErrMissingConfirmations          = errors.New("not enough confirmations")
	ErrInvalidStakeRemovedEventID    = errors.New("invalid stake removed event ID")
	ErrInvalidStakeDepositedEventID  = errors.New("invalid stake deposited event ID")
	ErrDuplicatedStakeDepositedEvent = errors.New("duplicated stake deposited event")
	ErrDuplicatedStakeRemovedEvent   = errors.New("duplicated stake deposited event")
)

// Witness provide foreign chain resources validations

type TimeService interface {
	GetTimeNow() time.Time
}

type EthConfirmations interface {
	Check(uint64) error
}

type EthOnChainVerifier interface {
	CheckStakeDeposited(*types.StakeDeposited) error
	CheckStakeRemoved(*types.StakeRemoved) error
}

// Witness provide foreign chain resources validations.
type Witness interface {
	StartCheck(validators.Resource, func(interface{}, bool), time.Time) error
	RestoreResource(validators.Resource, func(interface{}, bool)) error
}

type StakeVerifier struct {
	log *logging.Logger
	cfg Config

	accs        *Accounting
	witness     Witness
	timeService TimeService
	broker      Broker

	ocv            EthOnChainVerifier
	ethEventSource EthereumEventSource

	pendingSDs      []*pendingSD
	pendingSRs      []*pendingSR
	finalizedEvents []*types.StakeLinking

	mu     sync.Mutex
	ids    map[string]struct{}
	hashes map[string]struct{}

	// snapshot data
	svss *stakeVerifierSnapshotState
}

type pendingSD struct {
	*types.StakeDeposited
	check func() error
}

func (p pendingSD) GetID() string               { return p.ID }
func (p pendingSD) GetType() types.NodeVoteType { return types.NodeVoteTypeStakeDeposited }
func (p *pendingSD) Check() error               { return p.check() }

type pendingSR struct {
	*types.StakeRemoved
	check func() error
}

func (p pendingSR) GetID() string               { return p.ID }
func (p pendingSR) GetType() types.NodeVoteType { return types.NodeVoteTypeStakeRemoved }
func (p *pendingSR) Check() error               { return p.check() }

func NewStakeVerifier(
	log *logging.Logger,
	cfg Config,
	accs *Accounting,
	witness Witness,
	ts TimeService,

	broker Broker,
	onChainVerifier EthOnChainVerifier,
	ethEventSource EthereumEventSource,
) (sv *StakeVerifier) {
	log = log.Named("stake-verifier")
	s := &StakeVerifier{
		log:            log,
		cfg:            cfg,
		accs:           accs,
		witness:        witness,
		ocv:            onChainVerifier,
		timeService:    ts,
		broker:         broker,
		ethEventSource: ethEventSource,
		ids:            map[string]struct{}{},
		hashes:         map[string]struct{}{},
		svss:           &stakeVerifierSnapshotState{},
	}
	return s
}

func (s *StakeVerifier) ensureNotDuplicate(id, h string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.ids[id]; ok {
		return false
	}
	if _, ok := s.hashes[h]; ok {
		return false
	}

	s.ids[id] = struct{}{}
	s.hashes[h] = struct{}{}

	return true
}

// TODO: address this as the ID/hash map will grow forever now
// func (s *StakeVerifier) removeEvent(id string) {
// 	delete(s.ids, id)
// }

func (s *StakeVerifier) ProcessStakeRemoved(
	ctx context.Context, event *types.StakeRemoved,
) error {
	if ok := s.ensureNotDuplicate(event.ID, event.IntoStakeLinking().Hash()); !ok {
		s.log.Error("stake removed event already exists",
			logging.String("event", event.String()))
		return ErrDuplicatedStakeRemovedEvent
	}

	pending := &pendingSR{
		StakeRemoved: event,
		check:        func() error { return s.ocv.CheckStakeRemoved(event) },
	}
	s.pendingSRs = append(s.pendingSRs, pending)
	evt := pending.IntoStakeLinking()
	evt.Status = types.StakeLinkingStatusPending
	s.broker.Send(events.NewStakeLinking(ctx, *evt))

	s.log.Info("stake removed event received, starting validation",
		logging.String("event", event.String()))

	return s.witness.StartCheck(
		pending, s.onEventVerified, s.timeService.GetTimeNow().Add(timeTilCancel))
}

func (s *StakeVerifier) ProcessStakeDeposited(
	ctx context.Context, event *types.StakeDeposited,
) error {
	if ok := s.ensureNotDuplicate(event.ID, event.IntoStakeLinking().Hash()); !ok {
		s.log.Error("stake deposited event already exists",
			logging.String("event", event.String()))
		return ErrDuplicatedStakeDepositedEvent
	}

	pending := &pendingSD{
		StakeDeposited: event,
		check:          func() error { return s.ocv.CheckStakeDeposited(event) },
	}

	s.pendingSDs = append(s.pendingSDs, pending)

	evt := pending.IntoStakeLinking()
	evt.Status = types.StakeLinkingStatusPending
	s.broker.Send(events.NewStakeLinking(ctx, *evt))

	s.log.Info("stake deposited event received, starting validation",
		logging.String("event", event.String()))

	return s.witness.StartCheck(
		pending, s.onEventVerified, s.timeService.GetTimeNow().Add(timeTilCancel))
}

func (s *StakeVerifier) removePendingStakeDeposited(id string) error {
	for i, v := range s.pendingSDs {
		if v.ID == id {
			s.pendingSDs = s.pendingSDs[:i+copy(s.pendingSDs[i:], s.pendingSDs[i+1:])]
			return nil
		}
	}
	return ErrInvalidStakeDepositedEventID
}

func (s *StakeVerifier) removePendingStakeRemoved(id string) error {
	for i, v := range s.pendingSRs {
		if v.ID == id {
			s.pendingSRs = s.pendingSRs[:i+copy(s.pendingSRs[i:], s.pendingSRs[i+1:])]
			return nil
		}
	}
	return ErrInvalidStakeRemovedEventID
}

func (s *StakeVerifier) getLastBlockSeen() uint64 {
	var block uint64
	for _, p := range s.pendingSDs {
		if block == 0 {
			block = p.BlockNumber
			continue
		}

		if p.BlockNumber < block {
			block = p.BlockNumber
		}
	}

	for _, p := range s.pendingSRs {
		if block == 0 {
			block = p.BlockNumber
			continue
		}

		if p.BlockNumber < block {
			block = p.BlockNumber
		}
	}

	return block
}

func (s *StakeVerifier) onEventVerified(event interface{}, ok bool) {
	var evt *types.StakeLinking
	switch pending := event.(type) {
	case *pendingSD:
		evt = pending.IntoStakeLinking()
		if err := s.removePendingStakeDeposited(evt.ID); err != nil {
			s.log.Error("could not remove pending stake deposited event", logging.Error(err))
		}
	case *pendingSR:
		evt = pending.IntoStakeLinking()
		if err := s.removePendingStakeRemoved(evt.ID); err != nil {
			s.log.Error("could not remove pending stake removed event", logging.Error(err))
		}
	default:
		s.log.Error("stake verifier received invalid event")
		return
	}

	evt.Status = types.StakeLinkingStatusRejected
	if ok {
		evt.Status = types.StakeLinkingStatusAccepted
	}
	evt.FinalizedAt = s.timeService.GetTimeNow().UnixNano()
	s.finalizedEvents = append(s.finalizedEvents, evt)
}

func (s *StakeVerifier) OnTick(ctx context.Context, t time.Time) {
	for _, evt := range s.finalizedEvents {
		// s.removeEvent(evt.ID)
		if evt.Status == types.StakeLinkingStatusAccepted {
			s.accs.AddEvent(ctx, evt)
		}
		s.log.Info("stake linking finalized",
			logging.String("status", evt.Status.String()),
			logging.String("event", evt.String()))
		s.broker.Send(events.NewStakeLinking(ctx, *evt))
	}

	s.finalizedEvents = nil
}
