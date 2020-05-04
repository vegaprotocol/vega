package plugins

import (
	"sync"
	"sync/atomic"

	types "code.vegaprotocol.io/vega/proto"
)

type streams struct {
	overall      map[int64]chan []types.GovernanceData
	overallCount int64
	overallMu    sync.RWMutex //TODO: measure if sync.Map is better here

	partyProposals      map[string]map[int64]chan []types.GovernanceData
	partyProposalsCount int64 // flat counter across all parties
	partyProposalsMu    sync.RWMutex

	partyVotes      map[string]map[int64]chan []types.Vote
	partyVotesCount int64 // flat counter across all parties

	proposalVotes      map[string]map[int64]chan []types.Vote
	proposalVotesCount int64 // flat counter across all proposals
	votesMu            sync.RWMutex
}

func newStreams() streams {
	return streams{
		overall:        map[int64]chan []types.GovernanceData{},
		partyProposals: map[string]map[int64]chan []types.GovernanceData{},
		partyVotes:     map[string]map[int64]chan []types.Vote{},
		proposalVotes:  map[string]map[int64]chan []types.Vote{},
	}
}

// notifications for all updates
func (s *streams) notifyAll(proposals []types.GovernanceData) {
	if len(proposals) > 0 { // disallow empty notifications
		s.overallMu.RLock()
		for _, ch := range s.overall {
			// push onto channel, but don't wait for consumer to read the data
			// the channel is buffered to 1, so we can write if the channel is empty
			select {
			case ch <- proposals:
				continue
			default:
				continue
			}
		}
		s.overallMu.RUnlock()
	}
}

func (s *streams) subscribeAll() (<-chan []types.GovernanceData, int64) {
	ch := make(chan []types.GovernanceData, 1)
	//TODO: measure, these atomic operations are likely meaningless here
	k := atomic.AddInt64(&s.overallCount, 1)

	s.overallMu.Lock()
	s.overall[k] = ch
	s.overallMu.Unlock()

	return ch, k
}

func (s *streams) unsubscribeAll(k int64) {
	s.overallMu.Lock()
	if ch, ok := s.overall[k]; ok {
		close(ch)
		delete(s.overall, k)
	}
	s.overallMu.Unlock()
}

func partitionProposalsByParty(proposals []types.GovernanceData) map[string][]types.GovernanceData {
	result := map[string][]types.GovernanceData{}
	for _, v := range proposals {
		result[v.Proposal.PartyID] = append(result[v.Proposal.PartyID], v)
	}
	return result
}

// notifications for proposal updates (no votes)
func (s *streams) notifyProposals(data []types.GovernanceData) {
	if len(data) > 0 {
		byParty := partitionProposalsByParty(data)

		s.partyProposalsMu.RLock()
		// the assumption here is that there is likely to be less per party proposal
		// subscriptions than new proposals received by the node
		// if this assumption is incorrect, next two lines have to be inverted
		for partyID, subs := range s.partyProposals {
			if proposals, exists := byParty[partyID]; exists {
				for _, ch := range subs {
					select {
					case ch <- proposals:
						continue
					default:
						continue
					}
				}
			}
		}
		s.partyProposalsMu.RUnlock()
	}
}

func (s *streams) subscribePartyProposals(partyID string) (<-chan []types.GovernanceData, int64) {
	ch := make(chan []types.GovernanceData, 1)
	k := atomic.AddInt64(&s.partyProposalsCount, 1)

	s.partyProposalsMu.Lock()

	if byPartySubs, exists := s.partyProposals[partyID]; exists {
		byPartySubs[k] = ch
	} else {
		s.partyProposals[partyID] = map[int64]chan []types.GovernanceData{k: ch}
	}
	s.partyProposalsMu.Unlock()

	return ch, k
}

func (s *streams) unsubscribePartyProposals(partyID string, k int64) {
	s.partyProposalsMu.Lock()
	if subs, exists := s.partyProposals[partyID]; exists {
		if ch, ok := subs[k]; ok {
			close(ch)
			delete(subs, k)
		}
	}
	s.partyProposalsMu.Unlock()
}

func partitionVotesByParty(votes []types.Vote) map[string][]types.Vote {
	result := map[string][]types.Vote{}
	for _, v := range votes {
		result[v.PartyID] = append(result[v.PartyID], v)
	}
	return result
}

func partitionVotesByProposalID(votes []types.Vote) map[string][]types.Vote {
	result := map[string][]types.Vote{}
	for _, v := range votes {
		result[v.ProposalID] = append(result[v.ProposalID], v)
	}
	return result
}

// notifications for vote casts (no other proposal updates otherwise)
func (s *streams) notifyVotes(votes []types.Vote) {
	if len(votes) > 0 {
		byParty := partitionVotesByParty(votes)
		byProposal := partitionVotesByProposalID(votes)

		s.votesMu.RLock()

		// the assumption here is that there is likely to be less per party vote
		// subscriptions than new votes received by the node
		for partyID, subs := range s.partyVotes {
			if votes, exists := byParty[partyID]; exists {
				for _, ch := range subs {
					select {
					case ch <- votes:
						continue
					default:
						continue
					}
				}
			}
		}
		for proposalID, subs := range s.proposalVotes {
			if votes, exists := byProposal[proposalID]; exists {
				for _, ch := range subs {
					select {
					case ch <- votes:
						continue
					default:
						continue
					}
				}
			}
		}
		s.votesMu.RUnlock()
	}
}

func (s *streams) subscribePartyVotes(partyID string) (<-chan []types.Vote, int64) {
	k := atomic.AddInt64(&s.partyVotesCount, 1)
	ch := make(chan []types.Vote, 1)

	s.votesMu.Lock()
	if byPartySubs, exists := s.partyVotes[partyID]; exists {
		byPartySubs[k] = ch
	} else {
		s.partyVotes[partyID] = map[int64]chan []types.Vote{k: ch}
	}
	s.votesMu.Unlock()

	return ch, k
}

func (s *streams) unsubscribePartyVotes(partyID string, k int64) {
	s.votesMu.Lock()
	if subs, exists := s.partyVotes[partyID]; exists {
		if ch, ok := subs[k]; ok {
			close(ch)
			delete(subs, k)
		}
	}
	s.votesMu.Unlock()
}

func (s *streams) subscribeProposalVotes(proposalID string) (<-chan []types.Vote, int64) {
	ch := make(chan []types.Vote, 1)
	k := atomic.AddInt64(&s.proposalVotesCount, 1)

	s.votesMu.Lock()
	if byProposalSubs, exists := s.proposalVotes[proposalID]; exists {
		byProposalSubs[k] = ch
	} else {
		s.proposalVotes[proposalID] = map[int64]chan []types.Vote{k: ch}
	}
	s.votesMu.Unlock()

	return ch, k
}

func (s *streams) unsubscribeProposalVotes(proposalID string, k int64) {
	s.votesMu.Lock()
	if subs, exists := s.proposalVotes[proposalID]; exists {
		if ch, ok := subs[k]; ok {
			close(ch)
			delete(subs, k)
		}
	}
	s.votesMu.Unlock()
}
