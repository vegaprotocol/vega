package staking

import (
	"context"
	"errors"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/validators"
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
//go:generate go run github.com/golang/mock/mockgen -destination mocks/time_ticker_mock.go -package mocks code.vegaprotocol.io/vega/staking TimeTicker
type TimeTicker interface {
	NotifyOnTick(func(context.Context, time.Time))
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/eth_confirmations_mock.go -package mocks code.vegaprotocol.io/vega/staking EthConfirmations
type EthConfirmations interface {
	Check(uint64) error
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/eth_on_chain_verifier_mock.go -package mocks code.vegaprotocol.io/vega/staking EthOnChainVerifier
type EthOnChainVerifier interface {
	CheckStakeDeposited(*types.StakeDeposited) error
	CheckStakeRemoved(*types.StakeRemoved) error
}

// Witness provide foreign chain resources validations
//go:generate go run github.com/golang/mock/mockgen -destination mocks/witness_mock.go -package mocks code.vegaprotocol.io/vega/staking Witness
type Witness interface {
	StartCheck(validators.Resource, func(interface{}, bool), time.Time) error
	RestoreResource(validators.Resource, func(interface{}, bool)) error
}

type StakeVerifier struct {
	log *logging.Logger
	cfg Config

	accs    *Accounting
	witness Witness
	broker  Broker

	ocv EthOnChainVerifier

	currentTime time.Time

	pendingSDs      []*pendingSD
	pendingSRs      []*pendingSR
	finalizedEvents []*types.StakeLinking

	mu     sync.Mutex
	ids    map[string]struct{}
	hashes map[string]struct{}

	// snapshot data
	svss            *stakeVerifierSnapshotState
	keyToSerialiser map[string]func() ([]byte, error)
}

type pendingSD struct {
	*types.StakeDeposited
	check func() error
}

func (p pendingSD) GetID() string { return p.ID }
func (p *pendingSD) Check() error { return p.check() }

type pendingSR struct {
	*types.StakeRemoved
	check func() error
}

func (p pendingSR) GetID() string { return p.ID }
func (p *pendingSR) Check() error { return p.check() }

func NewStakeVerifier(
	log *logging.Logger,
	cfg Config,
	accs *Accounting,
	tt TimeTicker,
	witness Witness,
	broker Broker,
	onChainVerifier EthOnChainVerifier,
) (sv *StakeVerifier) {
	defer func() {
		tt.NotifyOnTick(sv.onTick)
	}()
	log = log.Named("stake-verifier")
	s := &StakeVerifier{
		log:     log,
		cfg:     cfg,
		accs:    accs,
		witness: witness,
		ocv:     onChainVerifier,
		broker:  broker,
		ids:     map[string]struct{}{},
		hashes:  map[string]struct{}{},
		svss: &stakeVerifierSnapshotState{
			changed:    map[string]bool{depositedKey: true, removedKey: true},
			serialised: map[string][]byte{},
			hash:       map[string][]byte{},
		},
		keyToSerialiser: map[string]func() ([]byte, error){},
	}

	s.keyToSerialiser[depositedKey] = s.serialisePendingSD
	s.keyToSerialiser[removedKey] = s.serialisePendingSR

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
	s.svss.changed[removedKey] = true

	s.pendingSRs = append(s.pendingSRs, pending)
	evt := pending.IntoStakeLinking()
	evt.Status = types.StakeLinkingStatusPending
	s.broker.Send(events.NewStakeLinking(ctx, *evt))

	s.log.Info("stake removed event received, starting validation",
		logging.String("event", event.String()))

	return s.witness.StartCheck(
		pending, s.onEventVerified, s.currentTime.Add(timeTilCancel))
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
	s.svss.changed[depositedKey] = true

	evt := pending.IntoStakeLinking()
	evt.Status = types.StakeLinkingStatusPending
	s.broker.Send(events.NewStakeLinking(ctx, *evt))

	s.log.Info("stake deposited event received, starting validation",
		logging.String("event", event.String()))

	return s.witness.StartCheck(
		pending, s.onEventVerified, s.currentTime.Add(timeTilCancel))
}

func (s *StakeVerifier) removePendingStakeDeposited(id string) error {
	for i, v := range s.pendingSDs {
		if v.ID == id {
			s.pendingSDs = s.pendingSDs[:i+copy(s.pendingSDs[i:], s.pendingSDs[i+1:])]
			s.svss.changed[depositedKey] = true
			return nil
		}
	}
	return ErrInvalidStakeDepositedEventID
}

func (s *StakeVerifier) removePendingStakeRemoved(id string) error {
	for i, v := range s.pendingSRs {
		if v.ID == id {
			s.pendingSRs = s.pendingSRs[:i+copy(s.pendingSRs[i:], s.pendingSRs[i+1:])]
			s.svss.changed[removedKey] = true
			return nil
		}
	}
	return ErrInvalidStakeRemovedEventID
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
	evt.FinalizedAt = s.currentTime.UnixNano()
	s.finalizedEvents = append(s.finalizedEvents, evt)
}

func (s *StakeVerifier) onTick(ctx context.Context, t time.Time) {
	s.currentTime = t
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
