package plugins

import (
	"context"
	"sync"
	"sync/atomic"

	types "code.vegaprotocol.io/vega/proto"

	"github.com/pkg/errors"
)

var (
	ErrProposalNotFound = errors.New("proposal not found")
)

// PropBuffer...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/prop_buffer_mock.go -package mocks code.vegaprotocol.io/vega/plugins PropBuffer
type PropBuffer interface {
	Subscribe() (<-chan []types.Proposal, int)
	Unsubscribe(int)
}

// VoteBuffer...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/vote_buffer_mock.go -package mocks code.vegaprotocol.io/vega/plugins VoteBuffer
type VoteBuffer interface {
	Subscribe() (<-chan []types.Vote, int)
	Unsubscribe(int)
}

type Proposals struct {
	mu             sync.RWMutex
	props          PropBuffer
	votes          VoteBuffer
	pref, vref     int
	pch            <-chan []types.Proposal
	vch            <-chan []types.Vote
	pData          map[string]*types.Proposal
	pByRef         map[string]*types.Proposal
	vData          map[string]map[types.Vote_Value]map[string]*types.Vote // nested map by proposal -> vote value -> party
	pByPartyID     map[string][]*types.Proposal
	vByPartyID     map[string][]*types.Vote
	newMarkets     map[string]*types.Proposal
	marketUpdates  map[string][]*types.Proposal
	networkUpdates []*types.Proposal

	// stream subscriptions
	subs     map[int64]chan []types.GovernanceData
	subCount int64
}

// NewProposals - return a new proposal plugin
func NewProposals(p PropBuffer, v VoteBuffer) *Proposals {
	return &Proposals{
		props:         p,
		votes:         v,
		pData:         map[string]*types.Proposal{},
		pByRef:        map[string]*types.Proposal{},
		vData:         map[string]map[types.Vote_Value]map[string]*types.Vote{},
		pByPartyID:    map[string][]*types.Proposal{},
		vByPartyID:    map[string][]*types.Vote{},
		newMarkets:    map[string]*types.Proposal{},
		marketUpdates: map[string][]*types.Proposal{},
		subs:          map[int64]chan []types.GovernanceData{},
	}
}

// Start - start running the consume loop for the plugin
func (p *Proposals) Start(ctx context.Context) {
	p.mu.Lock()
	running := true
	if p.pch == nil {
		p.pch, p.pref = p.props.Subscribe()
		running = false
	}
	if p.vch == nil {
		p.vch, p.vref = p.votes.Subscribe()
		running = false
	}
	if !running {
		go p.consume(ctx)
	}
	p.mu.Unlock()
}

// Stop - stop running the plugin. Does not set channels to nil to avoid data-race in consume loop
func (p *Proposals) Stop() {
	p.mu.Lock()
	if p.pref != 0 {
		p.props.Unsubscribe(p.pref)
		p.pref = 0
	}
	if p.vref != 0 {
		p.votes.Unsubscribe(p.vref)
		p.vref = 0
	}
	p.mu.Unlock()
}

func (p *Proposals) storeProposal(proposal *types.Proposal) {
	p.pData[proposal.ID] = proposal
	p.pByRef[proposal.Reference] = proposal
	p.pByPartyID[proposal.PartyID] = append(p.pByPartyID[proposal.PartyID], proposal)
	if _, ok := p.vData[proposal.ID]; !ok {
		p.vData[proposal.ID] = map[types.Vote_Value]map[string]*types.Vote{
			types.Vote_YES: map[string]*types.Vote{},
			types.Vote_NO:  map[string]*types.Vote{},
		}
	}
	switch proposal.Terms.Change.(type) {
	case *types.ProposalTerms_NewMarket:
		p.newMarkets[proposal.Terms.GetNewMarket().Changes.Id] = proposal // each market has unique id
	case *types.ProposalTerms_UpdateMarket:
		//id := proposal.Terms.GetUpdateMarket().Changes.Id
		//p.marketUpdates[id] = append(p.marketUpdates[id], proposal)
	case *types.ProposalTerms_UpdateNetwork:
		p.networkUpdates = append(p.networkUpdates, proposal)
	}
}

func (p *Proposals) consume(ctx context.Context) {
	defer func() {
		p.Stop()
		p.pch = nil
		p.vch = nil
	}()
	for {
		select {
		case <-ctx.Done():
			return
		case proposals, ok := <-p.pch:
			if !ok {
				// channel is closed
				return
			}
			// support empty slices for testing
			if len(proposals) == 0 {
				continue
			}
			updates := make([]string, 0, len(proposals))
			p.mu.Lock()
			for _, v := range proposals {
				p.storeProposal(&v)
				updates = append(updates, v.ID)
			}
			go p.notify(updates)
			p.mu.Unlock()
		case votes, ok := <-p.vch:
			if !ok {
				return
			}
			// empty slices are used for testing
			if len(votes) == 0 {
				continue
			}
			// alloc assuming worst case scenario
			updates := make([]string, 0, len(votes))
			p.mu.Lock()
			for _, v := range votes {
				pvotes, ok := p.vData[v.ProposalID]
				if !ok {
					pvotes = map[types.Vote_Value]map[string]*types.Vote{
						types.Vote_YES: map[string]*types.Vote{},
						types.Vote_NO:  map[string]*types.Vote{},
					}
				}
				// value maps always exist
				pvotes[v.Value][v.PartyID] = &v
				oppositeValue := types.Vote_NO
				if v.Value == oppositeValue {
					oppositeValue = types.Vote_YES
				}
				delete(pvotes[oppositeValue], v.PartyID)
				p.vData[v.ProposalID] = pvotes
				p.vByPartyID[v.PartyID] = append(p.vByPartyID[v.PartyID], &v)
				updates = append(updates, v.ProposalID)
			}
			go p.notify(updates)
			p.mu.Unlock()
		}
	}
}

func (p *Proposals) notify(ids []string) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	data := make([]types.GovernanceData, 0, len(ids))
	for _, id := range ids {
		if prop, ok := p.pData[id]; ok {
			data = append(data, *p.getGovernanceData(*prop))
		}
	}
	for _, ch := range p.subs {
		// push onto channel, but don't wait for consumer to read the data
		// the channel is buffered to 1, so we can write if the channel is empty
		select {
		case ch <- data:
			continue
		default:
			continue
		}
	}
}

// Subscribe - get all changes to proposals/votes
func (p *Proposals) Subscribe() (<-chan []types.GovernanceData, int64) {
	p.mu.Lock()
	k := atomic.AddInt64(&p.subCount, 1)
	ch := make(chan []types.GovernanceData, 1)
	p.subs[k] = ch
	p.mu.Unlock()
	return ch, k
}

// Unsubscribe - remove stream of proposal updates
func (p *Proposals) Unsubscribe(k int64) {
	p.mu.Lock()
	if ch, ok := p.subs[k]; ok {
		close(ch)
		delete(p.subs, k)
	}
	p.mu.Unlock()
}

// GetAllGovernanceData get all proposals and votes
func (p *Proposals) GetAllGovernanceData() []*types.GovernanceData {
	return p.getProposals(nil)
}

// GetProposalsInState returns proposals + current votes in the specified state
func (p *Proposals) GetProposalsInState(includeState types.Proposal_State) []*types.GovernanceData {
	var inState proposalFilter = func(proposal *types.Proposal) bool {
		return proposal.State != includeState
	}
	return p.getProposals(&inState)
}

// GetProposalsNotInState returns proposals + current votes NOT in the specified state
func (p *Proposals) GetProposalsNotInState(excludeState types.Proposal_State) []*types.GovernanceData {
	var notInState proposalFilter = func(proposal *types.Proposal) bool {
		return proposal.State == excludeState
	}
	return p.getProposals(&notInState)
}

// GetProposalsByMarket returns proposals + current votes by market that is affected by these proposals
func (p *Proposals) GetProposalsByMarket(marketID string) []*types.GovernanceData {
	p.mu.RLock()
	defer p.mu.RUnlock()

	updated := p.marketUpdates[marketID]
	total := len(updated)

	added, ok := p.newMarkets[marketID]
	if ok {
		total++
	}
	result := make([]*types.GovernanceData, 0, total)
	result = append(result, p.getGovernanceData(*added))
	for _, prop := range updated {
		result = append(result, p.getGovernanceData(*prop))
	}
	return result
}

// GetProposalsByParty returns proposals + current votes by party authoring them
func (p *Proposals) GetProposalsByParty(partyID string) []*types.GovernanceData {
	p.mu.RLock()
	defer p.mu.RUnlock()

	found := p.pByPartyID[partyID]
	result := make([]*types.GovernanceData, 0, len(found))
	for _, prop := range found {
		result = append(result, p.getGovernanceData(*prop))
	}
	return result
}

// GetVotesByParty returns votes by party
func (p *Proposals) GetVotesByParty(partyID string) []*types.Vote {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.vByPartyID[partyID]
}

// GetProposalByID returns proposal and votes by ID
func (p *Proposals) GetProposalByID(id string) (*types.GovernanceData, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if v, ok := p.pData[id]; ok {
		return p.getGovernanceData(*v), nil
	}
	return nil, ErrProposalNotFound
}

// GetProposalByReference returns proposal by reference (or error if proposal not found)
func (p *Proposals) GetProposalByReference(ref string) (*types.GovernanceData, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if v, ok := p.pByRef[ref]; ok {
		return p.getGovernanceData(*v), nil
	}
	return nil, ErrProposalNotFound
}

// GetNewMarketProposals returns proposals aiming to create new markets
func (p *Proposals) GetNewMarketProposals(marketID string) []*types.GovernanceData {
	if len(marketID) != 0 {
		p.mu.RLock()
		defer p.mu.RUnlock()
		if v, ok := p.newMarkets[marketID]; ok {
			return []*types.GovernanceData{p.getGovernanceData(*v)}
		}
		return nil
	}

	p.mu.RLock()
	result := make([]*types.GovernanceData, 0, len(p.newMarkets))
	for _, v := range p.newMarkets {
		result = append(result, p.getGovernanceData(*v))
	}
	p.mu.RUnlock()
	return result
}

// GetUpdateMarketProposals returns proposals aiming to update markets
func (p *Proposals) GetUpdateMarketProposals(marketID string) []*types.GovernanceData {
	if len(marketID) != 0 {
		p.mu.RLock()
		defer p.mu.RUnlock()
		return p.pickAllProposals(p.marketUpdates[marketID])
	}
	p.mu.RLock()
	result := []*types.GovernanceData{}
	for _, props := range p.marketUpdates {
		result = append(result, p.pickAllProposals(props)...)
	}
	p.mu.RUnlock()
	return result
}

// GetNetworkParametersProposals returns proposals aiming to update network
func (p *Proposals) GetNetworkParametersProposals() []*types.GovernanceData {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.pickAllProposals(p.networkUpdates)
}

func (p *Proposals) pickAllProposals(from []*types.Proposal) []*types.GovernanceData {
	result := make([]*types.GovernanceData, 0, len(from))
	for _, prop := range from {
		result = append(result, p.getGovernanceData(*prop))
	}
	return result
}

type proposalFilter func(p *types.Proposal) bool

func (p *Proposals) getProposals(canSkip *proposalFilter) []*types.GovernanceData {
	var result []*types.GovernanceData
	p.mu.RLock()
	if canSkip == nil {
		result = make([]*types.GovernanceData, 0, len(p.pData))
	}
	for _, prop := range p.pData {
		if canSkip == nil || !(*canSkip)(prop) {
			result = append(result, p.getGovernanceData(*prop))
		}
	}
	p.mu.RUnlock()
	return result
}

func (p *Proposals) getGovernanceData(v types.Proposal) *types.GovernanceData {
	vData := p.vData[v.ID]
	yes := make([]*types.Vote, 0, len(vData[types.Vote_YES]))
	no := make([]*types.Vote, 0, len(vData[types.Vote_NO]))
	for _, vote := range vData[types.Vote_YES] {
		yes = append(yes, vote)
	}
	for _, vote := range vData[types.Vote_NO] {
		no = append(no, vote)
	}
	return &types.GovernanceData{
		Proposal: &v,
		Yes:      yes,
		No:       no,
	}
}
