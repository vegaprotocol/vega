package storage

import (
	"fmt"
	"math"
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
	subscribers             map[uint64]chan pb.Delegation
	subscriberID            uint64
	minEpoch                *uint64
}

func NewDelegations(log *logging.Logger, c Config) *Delegations {
	// setup logger
	log = log.Named(namedLogger)
	log.SetLevel(c.Level.Get())

	d := &Delegations{
		epochToPartyDelegations: map[string]map[string]map[string]string{},
		log:                     log,
		Config:                  c,
		subscribers:             map[uint64]chan pb.Delegation{},
		minEpoch:                new(uint64),
	}

	*d.minEpoch = math.MaxUint64
	return d
}

// ReloadConf update the internal conf of the market
func (s *Delegations) ReloadConf(cfg Config) {
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

// Subscribe allows a client to register for updates of the delegations
func (s *Delegations) Subscribe(updates chan pb.Delegation) uint64 {
	s.mut.Lock()
	defer s.mut.Unlock()

	s.subscriberID++
	s.subscribers[s.subscriberID] = updates

	s.log.Debug("Delegations subscriber added in delegations store",
		logging.Uint64("subscriber-id", s.subscriberID))

	return s.subscriberID
}

// Unsubscribe allows the client to unregister interest in delegation
func (s *Delegations) Unsubscribe(id uint64) error {
	s.mut.Lock()
	defer s.mut.Unlock()

	if len(s.subscribers) == 0 {
		s.log.Debug("Un-subscribe called in delegations store, no subscribers connected",
			logging.Uint64("subscriber-id", id))
		return nil
	}

	if _, exists := s.subscribers[id]; exists {
		delete(s.subscribers, id)
		return nil
	}

	return fmt.Errorf("subscriber to delegation updates does not exist with id: %d", id)
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
		clearOldEpochsDelegations(de.EpochSeq, s.minEpoch, func(epochSeq string) { delete(s.epochToPartyDelegations, epochSeq) })
	}

	party, ok := epoch[de.Party]
	if !ok {
		party = map[string]string{}
		epoch[de.Party] = party
	}
	party[de.NodeId] = de.Amount

	s.notifyWithLock(de)
}

//notifyWithLock notifies registered subscribers - assumes lock has already been acquired
func (s *Delegations) notifyWithLock(de pb.Delegation) {
	if len(s.subscribers) == 0 {
		return
	}
	var ok bool
	for id, sub := range s.subscribers {
		select {
		case sub <- de:
			ok = true
		default:
			ok = false
		}
		if ok {
			s.log.Debug("Delegations channel updated for subscriber successfully",
				logging.Uint64("id", id))
		} else {
			s.log.Debug("Delegations channel could not be updated for subscriber",
				logging.Uint64("id", id))
		}
	}
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

//GetNodeDelegationsOnEpoch returns the delegations to a node by all parties at a given epoch
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

//GetPartyDelegationsOnEpoch returns all delegation by party on a given epoch
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

//GetPartyNodeDelegationsOnEpoch returns the delegations from party to node at epoch
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
