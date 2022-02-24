package services

import (
	"context"
	"math"
	"strconv"
	"sync"

	pb "code.vegaprotocol.io/protos/vega"
	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/subscribers"
)

// next, current and last two.
const maxEpochsToKeep = uint64(4)

type delegationE interface {
	events.Event
	Proto() eventspb.DelegationBalanceEvent
}

// Delegations is a storage for keeping track of delegation state from delegation balance update events.
type Delegations struct {
	*subscribers.Base
	ctx context.Context

	mut                     sync.RWMutex
	epochToPartyDelegations map[string]map[string]map[string]string // epoch -> party -> node -> amount
	ch                      chan eventspb.DelegationBalanceEvent
	minEpoch                uint64
}

func NewDelegations(ctx context.Context) (delegations *Delegations) {
	defer func() { go delegations.consume() }()

	return &Delegations{
		Base:                    subscribers.NewBase(ctx, 1000, true),
		ctx:                     ctx,
		epochToPartyDelegations: map[string]map[string]map[string]string{},
		ch:                      make(chan eventspb.DelegationBalanceEvent, 100),
		minEpoch:                math.MaxUint64,
	}
}

func (d *Delegations) consume() {
	defer func() { close(d.ch) }()
	for {
		select {
		case <-d.Closed():
			return
		case de, ok := <-d.ch:
			if !ok {
				// cleanup base
				d.Halt()
				// channel is closed
				return
			}
			d.mut.Lock()
			d.addDelegation(pb.Delegation{
				NodeId:   de.NodeId,
				Party:    de.Party,
				EpochSeq: de.EpochSeq,
				Amount:   de.Amount,
			})
			d.mut.Unlock()
		}
	}
}

func (d *Delegations) Push(evts ...events.Event) {
	for _, e := range evts {
		if ae, ok := e.(delegationE); ok {
			d.ch <- ae.Proto()
		}
	}
}

func (d *Delegations) List(party, node, epoch string) []*pb.Delegation {
	d.mut.RLock()
	defer d.mut.RUnlock()

	var delegations []*pb.Delegation
	if epoch == "" && party == "" && node == "" { // all delegations for all parties all nodes across all epochs
		delegations = d.getAllDelegations()
	} else if epoch == "" && party == "" && node != "" { // all delegations for node from all parties across all epochs
		delegations = d.getNodeDelegations(node)
	} else if epoch == "" && party != "" && node == "" { // all delegations by a given party to all nodes across all epochs
		delegations = d.getPartyDelegations(party)
	} else if epoch == "" && party != "" && node != "" { // all delegations by a given party to a given node across all epochs
		delegations = d.getPartyNodeDelegations(party, node)
	} else if epoch != "" && party == "" && node == "" { // all delegations by all parties for all nodes in a given epoch
		delegations = d.getAllDelegationsOnEpoch(epoch)
	} else if epoch != "" && party == "" && node != "" { // all delegations to a given node on a given epoch
		delegations = d.getNodeDelegationsOnEpoch(node, epoch)
	} else if epoch != "" && party != "" && node == "" { // all delegations by a given party on a given epoch
		delegations = d.getPartyDelegationsOnEpoch(party, epoch)
	} else if epoch != "" && party != "" && node != "" { // all delegations by a given party to a given node on a given epoch
		delegations = d.getPartyNodeDelegationsOnEpoch(party, node, epoch)
	}

	return delegations
}

// clearOldDelegations makes sure we only keep as many as <maxEpochsToKeep> epoch delegations.
func (d *Delegations) clearOldDelegations(epochSeq string) {
	epochSeqUint, err := strconv.ParseUint(epochSeq, 10, 64)
	if err != nil {
		return
	}
	// if we see an epoch younger than we've seen before - update the min epoch
	if epochSeqUint <= d.minEpoch {
		d.minEpoch = epochSeqUint
	}
	// if we haven't seen yet <maxEpochsToKeep> or we have no more than the required number of epochs - we don't have anything to do here
	if epochSeqUint < maxEpochsToKeep || d.minEpoch >= (epochSeqUint-maxEpochsToKeep+1) {
		return
	}

	// cleanup enough epochs such that we have at most <maxEpochsToKeep> epochs
	for i := d.minEpoch; i < (epochSeqUint - maxEpochsToKeep + 1); i++ {
		delete(d.epochToPartyDelegations, strconv.FormatUint(i, 10))
	}
	d.minEpoch = epochSeqUint - maxEpochsToKeep + 1
}

// AddDelegation is updated with new delegation update from the subscriber.
func (d *Delegations) addDelegation(de pb.Delegation) {
	// update party delegations
	epoch, ok := d.epochToPartyDelegations[de.EpochSeq]
	if !ok {
		epoch = map[string]map[string]string{}
		d.epochToPartyDelegations[de.EpochSeq] = epoch
		d.clearOldDelegations(de.EpochSeq)
	}

	party, ok := epoch[de.Party]
	if !ok {
		party = map[string]string{}
		epoch[de.Party] = party
	}
	party[de.NodeId] = de.Amount
}

// GetAllDelegations returns all delegations across all epochs, all parties, all nodes.
func (d *Delegations) getAllDelegations() []*pb.Delegation {
	delegations := []*pb.Delegation{}

	for epoch, epochDelegations := range d.epochToPartyDelegations {
		for party, partyDelegations := range epochDelegations {
			for node, amount := range partyDelegations {
				delegations = append(delegations, &pb.Delegation{
					Party:    party,
					NodeId:   node,
					Amount:   amount,
					EpochSeq: epoch,
				})
			}
		}
	}
	return delegations
}

// GetAllDelegationsOnEpoch returns all delegation for the given epoch.
func (d *Delegations) getAllDelegationsOnEpoch(epochSeq string) []*pb.Delegation {
	delegations := []*pb.Delegation{}

	epochDelegations, ok := d.epochToPartyDelegations[epochSeq]
	if !ok {
		return delegations
	}
	for party, partyDelegations := range epochDelegations {
		for node, amount := range partyDelegations {
			delegations = append(delegations, &pb.Delegation{
				Party:    party,
				NodeId:   node,
				Amount:   amount,
				EpochSeq: epochSeq,
			})
		}
	}
	return delegations
}

// GetNodeDelegations returns all the delegations made to a node across all epochs.
func (d *Delegations) getNodeDelegations(nodeID string) []*pb.Delegation {
	delegations := []*pb.Delegation{}

	for epoch, epochDelegations := range d.epochToPartyDelegations {
		for party, partyDelegations := range epochDelegations {
			for node, amount := range partyDelegations {
				if node != nodeID {
					continue
				}
				delegations = append(delegations, &pb.Delegation{
					Party:    party,
					NodeId:   node,
					Amount:   amount,
					EpochSeq: epoch,
				})
			}
		}
	}
	return delegations
}

// GetNodeDelegationsOnEpoch returns the delegations to a node by all parties at a given epoch.
func (d *Delegations) getNodeDelegationsOnEpoch(nodeID string, epochSeq string) []*pb.Delegation {
	delegations := []*pb.Delegation{}

	epochDelegations, ok := d.epochToPartyDelegations[epochSeq]
	if !ok {
		return delegations
	}

	for party, partyDelegations := range epochDelegations {
		for node, amount := range partyDelegations {
			if node != nodeID {
				continue
			}
			delegations = append(delegations, &pb.Delegation{
				Party:    party,
				NodeId:   node,
				Amount:   amount,
				EpochSeq: epochSeq,
			})
		}
	}
	return delegations
}

// GetPartyDelegations returns all the delegations by a party across all epochs.
func (d *Delegations) getPartyDelegations(party string) []*pb.Delegation {
	delegations := []*pb.Delegation{}

	for epoch, epochDelegations := range d.epochToPartyDelegations {
		partyDelAtEpoch, ok := epochDelegations[party]
		if !ok {
			continue
		}

		for node, amount := range partyDelAtEpoch {
			delegations = append(delegations, &pb.Delegation{
				Party:    party,
				NodeId:   node,
				Amount:   amount,
				EpochSeq: epoch,
			})
		}
	}
	return delegations
}

// GetPartyDelegationsOnEpoch returns all delegation by party on a given epoch.
func (d *Delegations) getPartyDelegationsOnEpoch(party string, epochSeq string) []*pb.Delegation {
	delegations := []*pb.Delegation{}

	epochDelegations, ok := d.epochToPartyDelegations[epochSeq]
	if !ok {
		return delegations
	}

	partyDelegations, ok := epochDelegations[party]
	if !ok {
		return delegations
	}

	for node, amount := range partyDelegations {
		delegations = append(delegations, &pb.Delegation{
			Party:    party,
			NodeId:   node,
			Amount:   amount,
			EpochSeq: epochSeq,
		})
	}

	return delegations
}

// GetPartyNodeDelegations returns the delegations from party to node across all epochs.
func (d *Delegations) getPartyNodeDelegations(party string, node string) []*pb.Delegation {
	delegations := []*pb.Delegation{}

	for epoch, epochDelegations := range d.epochToPartyDelegations {
		partyDelegations, ok := epochDelegations[party]
		if !ok {
			continue
		}

		nodeDelegations, ok := partyDelegations[node]
		if !ok {
			continue
		}

		delegations = append(delegations, &pb.Delegation{
			Party:    party,
			NodeId:   node,
			Amount:   nodeDelegations,
			EpochSeq: epoch,
		})
	}

	return delegations
}

// GetPartyNodeDelegationsOnEpoch returns the delegations from party to node at epoch.
func (d *Delegations) getPartyNodeDelegationsOnEpoch(party, node, epochSeq string) []*pb.Delegation {
	delegations := []*pb.Delegation{}

	epochDelegations, ok := d.epochToPartyDelegations[epochSeq]
	if !ok {
		return delegations
	}

	partyDelegations, ok := epochDelegations[party]
	if !ok {
		return delegations
	}

	nodeDelegations, ok := partyDelegations[node]
	if !ok {
		return delegations
	}

	delegations = append(delegations, &pb.Delegation{
		Party:    party,
		NodeId:   node,
		Amount:   nodeDelegations,
		EpochSeq: epochSeq,
	})

	return delegations
}

func (d *Delegations) Types() []events.Type {
	return []events.Type{
		events.DelegationBalanceEvent,
	}
}
