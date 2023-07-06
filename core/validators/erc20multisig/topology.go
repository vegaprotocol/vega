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

package erc20multisig

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/core/broker"
	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/core/validators"
	"code.vegaprotocol.io/vega/logging"
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/mocks.go -package mocks code.vegaprotocol.io/vega/core/validators/erc20multisig Witness,MultiSigOnChainVerifier,EthConfirmations,EthereumEventSource

const (
	// 3 weeks, duration of the whole network at first?
	timeTilCancel = 24 * 21 * time.Hour
)

var (
	ErrDuplicatedSignerEvent    = errors.New("duplicated signer event")
	ErrDuplicatedThresholdEvent = errors.New("duplicated threshold event")
)

// Witness provide foreign chain resources validations.
type Witness interface {
	StartCheck(validators.Resource, func(interface{}, bool), time.Time) error
	RestoreResource(validators.Resource, func(interface{}, bool)) error
}

type MultiSigOnChainVerifier interface {
	CheckSignerEvent(*types.SignerEvent) error
	CheckThresholdSetEvent(*types.SignerThresholdSetEvent) error
}

type EthereumEventSource interface {
	UpdateMultisigControlStartingBlock(uint64)
}

// Topology keeps track of all the validators
// registered in the erc20 bridge.
type Topology struct {
	config Config
	log    *logging.Logger

	currentTime time.Time

	witness Witness
	broker  broker.Interface
	ocv     MultiSigOnChainVerifier

	// use to access both the pendingEvents and pendingThresholds maps
	mu sync.Mutex

	// the current map of all the signer on the bridge
	signers map[string]struct{}
	// signer address to list of all events related to it
	// order by block time.
	eventsPerAddress map[string][]*types.SignerEvent
	// a map of all pending events waiting to be processed
	pendingSigners map[string]*pendingSigner

	// the signer required treshold
	// last one is always kept
	threshold         *types.SignerThresholdSetEvent
	pendingThresholds map[string]*pendingThresholdSet

	// a map of all seen events
	seen map[string]struct{}

	witnessedThresholds map[string]struct{}
	witnessedSigners    map[string]struct{}

	// snapshot state
	tss            *topologySnapshotState
	ethEventSource EthereumEventSource
}

type pendingSigner struct {
	*types.SignerEvent

	check func() error
}

func (p pendingSigner) GetID() string { return p.ID }
func (p pendingSigner) GetType() types.NodeVoteType {
	var ty types.NodeVoteType
	switch p.Kind {
	case types.SignerEventKindAdded:
		ty = types.NodeVoteTypeSignerAdded
	case types.SignerEventKindRemoved:
		ty = types.NodeVoteTypeSignerRemoved
	}

	return ty
}

func (p *pendingSigner) Check(ctx context.Context) error { return p.check() }

type pendingThresholdSet struct {
	*types.SignerThresholdSetEvent
	check func() error
}

func (p pendingThresholdSet) GetID() string { return p.ID }
func (p pendingThresholdSet) GetType() types.NodeVoteType {
	return types.NodeVoteTypeSignerThresholdSet
}
func (p *pendingThresholdSet) Check(ctx context.Context) error { return p.check() }

func NewTopology(
	config Config,
	log *logging.Logger,
	witness Witness,
	ocv MultiSigOnChainVerifier,
	broker broker.Interface,
) *Topology {
	log = log.Named(namedLogger + ".topology")
	log.SetLevel(config.Level.Get())
	t := &Topology{
		config:              config,
		log:                 log,
		witness:             witness,
		ocv:                 ocv,
		broker:              broker,
		signers:             map[string]struct{}{},
		eventsPerAddress:    map[string][]*types.SignerEvent{},
		pendingSigners:      map[string]*pendingSigner{},
		pendingThresholds:   map[string]*pendingThresholdSet{},
		seen:                map[string]struct{}{},
		witnessedThresholds: map[string]struct{}{},
		witnessedSigners:    map[string]struct{}{},
		tss:                 &topologySnapshotState{},
	}
	return t
}

func (t *Topology) SetWitness(w Witness) {
	t.witness = w
}

func (t *Topology) SetEthereumEventSource(e EthereumEventSource) {
	t.ethEventSource = e
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
	t.pendingSigners[event.ID] = pending
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
		if !ok {
			// invalid, just delete from the map
			delete(t.pendingSigners, e.ID)
			return
		}
		t.witnessedSigners[e.ID] = struct{}{}
	case *pendingThresholdSet:
		if !ok {
			// invalid, just delete from the map
			delete(t.pendingThresholds, e.ID)
			return
		}
		t.witnessedThresholds[e.ID] = struct{}{}
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
	for k := range t.witnessedThresholds {
		ids = append(ids, k)
		delete(t.witnessedThresholds, k)
	}
	sort.Strings(ids)

	// now iterate over all events and update the
	// threshold if we get an event with a more recent
	// block time.
	for _, v := range ids {
		event := t.pendingThresholds[v]
		t.setThresholdSetEvent(ctx, event.SignerThresholdSetEvent)
		delete(t.pendingThresholds, v)
	}
}

func (t *Topology) setThresholdSetEvent(
	ctx context.Context, event *types.SignerThresholdSetEvent,
) {
	// if it's out first time here
	if t.threshold == nil || event.BlockTime > t.threshold.BlockTime {
		t.threshold = event
	}

	// send the event anyway so APIs can be aware of past thresholds
	t.broker.Send(events.NewERC20MultiSigThresholdSet(ctx, *event))
}

func (t *Topology) updateSigners(ctx context.Context) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if len(t.witnessedSigners) <= 0 {
		return
	}
	// sort all IDs to access pendings events in order
	ids := []string{}
	for k := range t.witnessedSigners {
		ids = append(ids, k)
		delete(t.witnessedSigners, k)
	}
	sort.Strings(ids)

	// first add all events to the map of events per addresses
	for _, id := range ids {
		// get the event
		event := t.pendingSigners[id]

		t.addSignerEvent(ctx, event.SignerEvent)

		// delete from pending then
		delete(t.pendingSigners, id)
	}
}

func (t *Topology) addSignerEvent(ctx context.Context, event *types.SignerEvent) {
	epa, ok := t.eventsPerAddress[event.Address]
	if !ok {
		epa = []*types.SignerEvent{}
	}

	// now add the event to the list for this address
	epa = append(epa, event)
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
	t.broker.Send(events.NewERC20MultiSigSigner(ctx, *event))
}
