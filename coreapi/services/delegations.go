package services

import (
	"context"
	"fmt"
	"sync"

	pb "code.vegaprotocol.io/protos/vega"
	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/subscribers"
)

type delegationE interface {
	events.Event
	Proto() eventspb.DelegationBalanceEvent
}

//Delegations is a storage for keeping track of delegation state from delegation balance update events
type Delegations struct {
	*subscribers.Base
	ctx context.Context

	mut                     sync.RWMutex
	epochToPartyDelegations map[string]map[string]map[string]string // epoch -> party -> node -> amount
	ch                      chan eventspb.DelegationBalanceEvent
}

func NewDelegations(ctx context.Context) (delegations *Delegations) {
	defer func() { go delegations.consume() }()

	return &Delegations{
		Base:                    subscribers.NewBase(ctx, 1000, true),
		ctx:                     ctx,
		epochToPartyDelegations: map[string]map[string]map[string]string{},
		ch:                      make(chan eventspb.DelegationBalanceEvent, 100),
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
				Amount:   fmt.Sprintf("%d", de.Amount),
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

func (s *Delegations) List(party, node, epoch string) []*pb.Delegation {
	s.mut.RLock()
	defer s.mut.RUnlock()

	var delegations []*pb.Delegation
	if epoch == "" && party == "" && node == "" { // all delegations for all parties all nodes across all epochs
		delegations, _ = s.getAllDelegations()
	} else if epoch == "" && party == "" && node != "" { // all delegations for node from all parties across all epochs
		delegations, _ = s.getNodeDelegations(node)
	} else if epoch == "" && party != "" && node == "" { // all delegations by a given party to all nodes across all epochs
		delegations, _ = s.getPartyDelegations(party)
	} else if epoch == "" && party != "" && node != "" { // all delegations by a given party to a given node across all epochs
		delegations, _ = s.getPartyNodeDelegations(party, node)
	} else if epoch != "" && party == "" && node == "" { // all delegations by all parties for all nodes in a given epoch
		delegations, _ = s.getAllDelegationsOnEpoch(epoch)
	} else if epoch != "" && party == "" && node != "" { // all delegations to a given node on a given epoch
		delegations, _ = s.getNodeDelegationsOnEpoch(node, epoch)
	} else if epoch != "" && party != "" && node == "" { // all delegations by a given party on a given epoch
		delegations, _ = s.getPartyDelegationsOnEpoch(party, epoch)
	} else if epoch != "" && party != "" && node != "" { // all delegations by a given party to a given node on a given epoch
		delegations, _ = s.getPartyNodeDelegationsOnEpoch(party, node, epoch)
	}

	return delegations
}

//AddDelegation is updated with new delegation update from the subscriber
func (s *Delegations) addDelegation(de pb.Delegation) {
	//update party delegations
	epoch, ok := s.epochToPartyDelegations[de.EpochSeq]
	if !ok {
		epoch = map[string]map[string]string{}
		s.epochToPartyDelegations[de.EpochSeq] = epoch
	}

	party, ok := epoch[de.Party]
	if !ok {
		party = map[string]string{}
		epoch[de.Party] = party
	}
	party[de.NodeId] = de.Amount
}

//GetAllDelegations returns all delegations across all epochs, all parties, all nodes
func (s *Delegations) getAllDelegations() ([]*pb.Delegation, error) {
	delegations := []*pb.Delegation{}

	for epoch, epochDelegations := range s.epochToPartyDelegations {
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
	return delegations, nil
}

//GetAllDelegationsOnEpoch returns all delegation for the given epoch
func (s *Delegations) getAllDelegationsOnEpoch(epochSeq string) ([]*pb.Delegation, error) {
	delegations := []*pb.Delegation{}

	epochDelegations, ok := s.epochToPartyDelegations[epochSeq]
	if !ok {
		return delegations, nil
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
	return delegations, nil
}

//GetNodeDelegations returns all the delegations made to a node across all epochs
func (s *Delegations) getNodeDelegations(nodeID string) ([]*pb.Delegation, error) {
	delegations := []*pb.Delegation{}

	for epoch, epochDelegations := range s.epochToPartyDelegations {
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
	return delegations, nil
}

//GetNodeDelegationsOnEpoch returns the delegations to a node by all parties at a given epoch
func (s *Delegations) getNodeDelegationsOnEpoch(nodeID string, epochSeq string) ([]*pb.Delegation, error) {
	delegations := []*pb.Delegation{}

	epochDelegations, ok := s.epochToPartyDelegations[epochSeq]
	if !ok {
		return delegations, nil
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
	return delegations, nil

}

//GetPartyDelegations returns all the delegations by a party across all epochs
func (s *Delegations) getPartyDelegations(party string) ([]*pb.Delegation, error) {
	delegations := []*pb.Delegation{}

	for epoch, epochDelegations := range s.epochToPartyDelegations {
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
	return delegations, nil
}

//GetPartyDelegationsOnEpoch returns all delegation by party on a given epoch
func (s *Delegations) getPartyDelegationsOnEpoch(party string, epochSeq string) ([]*pb.Delegation, error) {
	delegations := []*pb.Delegation{}

	epochDelegations, ok := s.epochToPartyDelegations[epochSeq]
	if !ok {
		return delegations, nil
	}

	partyDelegations, ok := epochDelegations[party]
	if !ok {
		return delegations, nil
	}

	for node, amount := range partyDelegations {
		delegations = append(delegations, &pb.Delegation{
			Party:    party,
			NodeId:   node,
			Amount:   amount,
			EpochSeq: epochSeq,
		})
	}

	return delegations, nil
}

//GetPartyNodeDelegations returns the delegations from party to node across all epochs
func (s *Delegations) getPartyNodeDelegations(party string, node string) ([]*pb.Delegation, error) {
	delegations := []*pb.Delegation{}

	for epoch, epochDelegations := range s.epochToPartyDelegations {
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

	return delegations, nil
}

//GetPartyNodeDelegationsOnEpoch returns the delegations from party to node at epoch
func (s *Delegations) getPartyNodeDelegationsOnEpoch(party, node, epochSeq string) ([]*pb.Delegation, error) {
	delegations := []*pb.Delegation{}

	epochDelegations, ok := s.epochToPartyDelegations[epochSeq]
	if !ok {
		return delegations, nil
	}

	partyDelegations, ok := epochDelegations[party]
	if !ok {
		return delegations, nil
	}

	nodeDelegations, ok := partyDelegations[node]
	if !ok {
		return delegations, nil
	}

	delegations = append(delegations, &pb.Delegation{
		Party:    party,
		NodeId:   node,
		Amount:   nodeDelegations,
		EpochSeq: epochSeq,
	})

	return delegations, nil
}

func (d *Delegations) Types() []events.Type {
	return []events.Type{
		events.DelegationBalanceEvent,
	}
}
