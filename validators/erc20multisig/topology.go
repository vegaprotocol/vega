package erc20multisig

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/broker"
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
	ErrDuplicatedSignerEvent    = errors.New("duplicated signer event")
	ErrDuplicatedThresholdEvent = errors.New("duplicated threshold event")
)

// Witness provide foreign chain resources validations
//go:generate go run github.com/golang/mock/mockgen -destination mocks/witness_mock.go -package mocks code.vegaprotocol.io/vega/validators/erc20multisig Witness
type Witness interface {
	StartCheck(validators.Resource, func(interface{}, bool), time.Time) error
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/multisig_on_chain_verifier_mock.go -package mocks code.vegaprotocol.io/vega/validators/erc20multisig MultiSigOnChainVerifier
type MultiSigOnChainVerifier interface {
	CheckSignerEvent(*types.SignerEvent) error
	CheckThresholdSetEvent(*types.SignerThresholdSetEvent) error
}

// Topology keeps track of all the validators
// registered in the erc20 bridge.
type Topology struct {
	config Config
	log    *logging.Logger

	currentTime time.Time

	witness Witness
	broker  broker.BrokerI
	ocv     MultiSigOnChainVerifier

	// use to access both the pendingEvents and pendingThresholds maps
	mu sync.Mutex

	// the current map of all the signer on the bridge
	signers map[string]struct{}
	// signer address to list of all events related to it
	// order by block time.
	eventsPerAddress map[string][]*types.SignerEvent
	// a map of all pending events waiting to be processed
	pendingEvents map[string]*pendingSigner

	// the signer required treshold
	// last one is always kept
	threshold         *types.SignerThresholdSetEvent
	pendingThresholds map[string]*pendingThresholdSet

	// a map of all seen events
	seen map[string]struct{}

	witnessedThresholds map[string]bool
	witnessedSigners    map[string]bool
}

type pendingSigner struct {
	*types.SignerEvent
	check func() error
}

func (p pendingSigner) GetID() string { return p.ID }
func (p *pendingSigner) Check() error { return p.check() }

type pendingThresholdSet struct {
	*types.SignerThresholdSetEvent
	check func() error
}

func (p pendingThresholdSet) GetID() string { return p.ID }
func (p *pendingThresholdSet) Check() error { return p.check() }

func NewTopology(
	config Config,
	log *logging.Logger,
	witness Witness,
	ocv MultiSigOnChainVerifier,
	broker broker.BrokerI,
) *Topology {
	log = log.Named(namedLogger + ".topology")
	log.SetLevel(config.Level.Get())
	return &Topology{
		config:              config,
		log:                 log,
		witness:             witness,
		ocv:                 ocv,
		broker:              broker,
		signers:             map[string]struct{}{},
		eventsPerAddress:    map[string][]*types.SignerEvent{},
		pendingEvents:       map[string]*pendingSigner{},
		pendingThresholds:   map[string]*pendingThresholdSet{},
		seen:                map[string]struct{}{},
		witnessedThresholds: map[string]bool{},
		witnessedSigners:    map[string]bool{},
	}
}

func (t *Topology) SetWitness(w Witness) {
	t.witness = w
}

func (t *Topology) ExcessSigners(addresses []string) bool {
	addressesMap := map[string]struct{}{}
	for _, v := range addresses {
		addressesMap[v] = struct{}{}
	}

	for k := range t.signers {
		if _, ok := addressesMap[k]; !ok {
			return true
		}
	}

	return false
}

func (t *Topology) GetSigners() []string {
	t.mu.Lock()
	defer t.mu.Unlock()

	out := make([]string, 0, len(t.signers))
	for k := range t.signers {
		out = append(out, k)
	}
	sort.Strings(out)

	return out
}

func (t *Topology) IsSigner(address string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	_, ok := t.signers[address]
	return ok
}

func (t *Topology) GetThreshold() uint32 {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.threshold != nil {
		return t.threshold.Threshold
	}
	return 0
}

func (t *Topology) ProcessSignerEvent(event *types.SignerEvent) error {
	if ok := t.ensureNotDuplicate(event.Hash()); !ok {
		t.log.Error("signer event already exists",
			logging.String("event", event.String()))
		return ErrDuplicatedSignerEvent
	}

	pending := &pendingSigner{
		SignerEvent: event,
		check:       func() error { return t.ocv.CheckSignerEvent(event) },
	}
	t.pendingEvents[event.ID] = pending
	// s.svss.changed[removedKey] = true

	t.log.Info("signer event received, starting validation",
		logging.String("event", event.String()))

	return t.witness.StartCheck(
		pending, t.onEventVerified, t.currentTime.Add(timeTilCancel))
}

func (t *Topology) ProcessThresholdEvent(event *types.SignerThresholdSetEvent) error {
	if ok := t.ensureNotDuplicate(event.Hash()); !ok {
		t.log.Error("threshold event already exists",
			logging.String("event", event.String()))
		return ErrDuplicatedThresholdEvent
	}

	pending := &pendingThresholdSet{
		SignerThresholdSetEvent: event,
		check:                   func() error { return t.ocv.CheckThresholdSetEvent(event) },
	}
	t.pendingThresholds[event.ID] = pending
	// s.svss.changed[removedKey] = true

	t.log.Info("signer threshold set event received, starting validation",
		logging.String("event", event.String()))

	return t.witness.StartCheck(
		pending, t.onEventVerified, t.currentTime.Add(timeTilCancel))
}

func (t *Topology) ensureNotDuplicate(h string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	if _, ok := t.seen[h]; ok {
		return false
	}
	t.seen[h] = struct{}{}

	return true
}

func (t *Topology) onEventVerified(event interface{}, ok bool) {
	switch e := event.(type) {
	case *pendingSigner:
		t.witnessedSigners[e.ID] = ok
	case *pendingThresholdSet:
		t.witnessedThresholds[e.ID] = ok
	default:
		t.log.Error("stake verifier received invalid event")
		return
	}
}

func (t *Topology) OnTick(ctx context.Context, ct time.Time) {
	t.currentTime = ct
	t.updateThreshold(ctx)
	t.updateSigners(ctx)
}

func (t *Topology) updateThreshold(ctx context.Context) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if len(t.witnessedThresholds) <= 0 {
		return
	}

	// sort all IDs to access pendings events in order
	ids := []string{}
	for k, v := range t.witnessedThresholds {
		// only account for events which were validated
		// by the network, meaning v == true
		if v {
			ids = append(ids, k)
		} else {
			// just deleting invalid ones
			delete(t.pendingThresholds, k)
		}
		delete(t.witnessedThresholds, k)
	}
	sort.Strings(ids)

	// now iterate over all events and update the
	// threshold if we get an event with a more recent
	// block time.
	for _, v := range ids {
		event := t.pendingThresholds[v]

		// if it's out first time here
		if t.threshold == nil {
			t.threshold = event.SignerThresholdSetEvent
			continue
		} else if event.BlockTime > t.threshold.BlockTime {
			// this event is more recent, we can replace our internal
			// event for treshold
			t.threshold = event.SignerThresholdSetEvent
		}

		// send the event anyway so APIs can be aware of past thresholds
		t.broker.Send(events.NewERC20MultiSigThresholdSet(ctx, *event.SignerThresholdSetEvent))

		delete(t.pendingThresholds, v)
	}
}

func (t *Topology) updateSigners(ctx context.Context) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if len(t.witnessedSigners) <= 0 {
		return
	}

	// sort all IDs to access pendings events in order
	ids := []string{}
	for k, v := range t.witnessedSigners {
		// only account for events which were validated
		// by the network, meaning v == true
		if v {
			ids = append(ids, k)
		} else {
			delete(t.pendingEvents, k)
		}
		delete(t.witnessedSigners, k)
	}
	sort.Strings(ids)

	// first add all events to the map of events per addresses
	for _, id := range ids {
		// get the event
		event := t.pendingEvents[id]
		epa, ok := t.eventsPerAddress[event.Address]
		if !ok {
			epa = []*types.SignerEvent{}
		}

		// now add the event to the list for this address
		epa = append(epa, event.SignerEvent)
		// sort them in arrival order
		sort.Slice(epa, func(i, j int) bool {
			return epa[i].BlockTime < epa[j].BlockTime
		})

		t.eventsPerAddress[event.Address] = epa

		// now depending of the last event received,
		// we add or remove from the list of signers
		switch epa[len(epa)-1].Kind {
		case types.SignerEventKindRemoved:
			delete(t.signers, event.Address)
		case types.SignerEventKindAdded:
			t.signers[event.Address] = struct{}{}
		}

		// send the event anyway so APIs can be aware of past thresholds
		t.broker.Send(events.NewERC20MultiSigSigner(ctx, *event.SignerEvent))
		// delete from pending then
		delete(t.pendingEvents, id)
	}
}
