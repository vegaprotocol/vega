package storage

import (
	"sync"

	"code.vegaprotocol.io/data-node/logging"
	pb "code.vegaprotocol.io/protos/vega"
)

//Delegations is a storage for keeping track of delegation state from delegation balance update events
type Delegations struct {
	Config

	mut                     sync.RWMutex
	epochToPartyDelegations map[string]map[string]map[string]string // epoch -> party -> node -> amount
	log                     *logging.Logger
}

func NewDelegations(log *logging.Logger, c Config) *Delegations {
	// setup logger
	log = log.Named(namedLogger)
	log.SetLevel(c.Level.Get())

	return &Delegations{
		epochToPartyDelegations: map[string]map[string]map[string]string{},
		log:                     log,
		Config:                  c,
	}
}

// ReloadConf update the internal conf of the market
func (e *Delegations) ReloadConf(cfg Config) {
	e.log.Info("reloading configuration")
	if e.log.GetLevel() != cfg.Level.Get() {
		e.log.Info("updating log level",
			logging.String("old", e.log.GetLevel().String()),
			logging.String("new", cfg.Level.String()),
		)
		e.log.SetLevel(cfg.Level.Get())
	}

	e.Config = cfg
}

//AddDelegation is updated with new delegation update from the subscriber
func (s *Delegations) AddDelegation(de pb.Delegation) {
	s.mut.Lock()
	defer s.mut.Unlock()

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
func (s *Delegations) GetAllDelegations() ([]*pb.Delegation, error) {
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
func (s *Delegations) GetAllDelegationsOnEpoch(epochSeq string) ([]*pb.Delegation, error) {
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
func (s *Delegations) GetNodeDelegations(nodeID string) ([]*pb.Delegation, error) {
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

//GetNodeEpochDelegations returns the delegations to a node by all parties at a given epoch
func (s *Delegations) GetNodeDelegationsOnEpoch(nodeID string, epochSeq string) ([]*pb.Delegation, error) {
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
func (s *Delegations) GetPartyDelegations(party string) ([]*pb.Delegation, error) {
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

//GetPartyEpochDelegations returns all delegation by party on a given epoch
func (s *Delegations) GetPartyDelegationsOnEpoch(party string, epochSeq string) ([]*pb.Delegation, error) {
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
func (s *Delegations) GetPartyNodeDelegations(party string, node string) ([]*pb.Delegation, error) {
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

//GetPartyNodeDelegations returns the delegations from party to node at epoch
func (s *Delegations) GetPartyNodeDelegationsOnEpoch(party, node, epochSeq string) ([]*pb.Delegation, error) {
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
