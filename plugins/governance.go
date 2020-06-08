package plugins

import (
	"context"
	"sync"

	types "code.vegaprotocol.io/vega/proto"

	"github.com/pkg/errors"
)

var (
	// ErrProposalNotFound is return if proposal has not been found
	ErrProposalNotFound = errors.New("proposal not found")
)

// PropBuffer ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/prop_buffer_mock.go -package mocks code.vegaprotocol.io/vega/plugins PropBuffer
type PropBuffer interface {
	Subscribe() (<-chan []types.Proposal, int)
	Unsubscribe(int)
}

// VoteBuffer ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/vote_buffer_mock.go -package mocks code.vegaprotocol.io/vega/plugins VoteBuffer
type VoteBuffer interface {
	Subscribe() (<-chan []types.Vote, int)
	Unsubscribe(int)
}

// Governance stores governance data
// threaded contention points:
// - Proposal - can be updated if state changes
// - Yes - updated on receiving a yes vote
// - No - updated on receiving a no vote
// each vote itself may be deleted
type Governance struct {
	mu sync.RWMutex

	props      PropBuffer
	votes      VoteBuffer
	pref, vref int
	pch        <-chan []types.Proposal
	vch        <-chan []types.Vote

	proposalsData map[string]*types.Proposal // Proposal.ID : Proposal
	votesData     map[string]*proposalVotes  // Proposal.ID : Votes

	views searchViews
	subs  streams
}

// NewGovernance - return a new governance plugin
func NewGovernance(p PropBuffer, v VoteBuffer) *Governance {
	return &Governance{
		props:         p,
		votes:         v,
		proposalsData: map[string]*types.Proposal{},
		votesData:     map[string]*proposalVotes{},
		views:         newViews(),
		subs:          newStreams(),
	}
}

// Start - start running the consume loop for the plugin
func (g *Governance) Start(ctx context.Context) {
	g.mu.Lock()
	running := true
	if g.pch == nil {
		g.pch, g.pref = g.props.Subscribe()
		running = false
	}
	if g.vch == nil {
		g.vch, g.vref = g.votes.Subscribe()
		running = false
	}
	if !running {
		go g.consume(ctx)
	}
	g.mu.Unlock()
}

// Stop - stop running the plugin. Does not set channels to nil to avoid data-race in consume loop
func (g *Governance) Stop() {
	g.mu.Lock()
	if g.pref != 0 {
		g.props.Unsubscribe(g.pref)
		g.pref = 0
	}
	if g.vref != 0 {
		g.votes.Unsubscribe(g.vref)
		g.vref = 0
	}
	g.mu.Unlock()
}

// SubscribeAll streams all governance data
func (g *Governance) SubscribeAll() (<-chan []types.GovernanceData, int64) {
	return g.subs.subscribeAll()
}

// UnsubscribeAll removes governance data stream
func (g *Governance) UnsubscribeAll(k int64) {
	g.subs.unsubscribeAll(k)
}

// SubscribePartyProposals streams proposals authored by the specific party
func (g *Governance) SubscribePartyProposals(partyID string) (<-chan []types.GovernanceData, int64) {
	return g.subs.subscribePartyProposals(partyID)
}

// UnsubscribePartyProposals removes stream of proposals for authored by the party
func (g *Governance) UnsubscribePartyProposals(partyID string, k int64) {
	g.subs.unsubscribePartyProposals(partyID, k)
}

// SubscribePartyVotes streams all votes cast by the specific party
func (g *Governance) SubscribePartyVotes(partyID string) (<-chan []types.Vote, int64) {
	return g.subs.subscribePartyVotes(partyID)
}

// UnsubscribePartyVotes removes stream of votes for the specific party
func (g *Governance) UnsubscribePartyVotes(partyID string, k int64) {
	g.subs.unsubscribePartyVotes(partyID, k)
}

// SubscribeProposalVotes streams all votes cast for the specific proposal
func (g *Governance) SubscribeProposalVotes(proposalID string) (<-chan []types.Vote, int64) {
	return g.subs.subscribeProposalVotes(proposalID)
}

// UnsubscribeProposalVotes removes stream of votes for the proposal
func (g *Governance) UnsubscribeProposalVotes(proposalID string, k int64) {
	g.subs.unsubscribeProposalVotes(proposalID, k)
}

// GetProposals get all proposals and votes filtered by the state if specified
func (g *Governance) GetProposals(inState *types.Proposal_State) []*types.GovernanceData {
	g.mu.RLock()

	result := make([]*types.GovernanceData, 0, len(g.proposalsData))
	for _, p := range g.proposalsData {
		if inState == nil || p.State == *inState {
			result = append(result, makeGovernanceData(p, g.votesData[p.ID]))
		}
	}
	g.mu.RUnlock()
	return result
}

// GetProposalsByParty returns proposals (+votes) by the party that authoring them
func (g *Governance) GetProposalsByParty(partyID string, inState *types.Proposal_State) []*types.GovernanceData {
	var result []*types.GovernanceData
	g.mu.RLock()
	for _, p := range g.proposalsData {
		if p.PartyID == partyID && (inState == nil || p.State == *inState) {
			result = append(result, makeGovernanceData(p, g.votesData[p.ID]))
		}
	}
	g.mu.RUnlock()
	return result
}

// GetVotesByParty returns votes cast by the party
func (g *Governance) GetVotesByParty(partyID string) []*types.Vote {
	var result []*types.Vote
	g.mu.RLock()
	if data, exists := g.views.partyVotes[partyID]; exists {
		result = make([]*types.Vote, len(data))
		for i, v := range data {
			result[i] = v
		}
	}
	g.mu.RUnlock()
	return result
}

// GetProposalByID returns proposal and votes by proposal ID
func (g *Governance) GetProposalByID(id string) (*types.GovernanceData, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if datum, exists := g.proposalsData[id]; exists {
		return makeGovernanceData(datum, g.votesData[id]), nil
	}
	return nil, ErrProposalNotFound
}

// GetProposalByReference returns proposal and votes by reference
func (g *Governance) GetProposalByReference(ref string) (*types.GovernanceData, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	for _, p := range g.proposalsData {
		if p.Reference == ref {
			return makeGovernanceData(p, g.votesData[p.ID]), nil
		}
	}
	return nil, ErrProposalNotFound
}

// GetNewMarketProposals returns proposals aiming to create new markets
func (g *Governance) GetNewMarketProposals(inState *types.Proposal_State) []*types.GovernanceData {
	skipper := selectInState(inState)

	g.mu.RLock()
	result := collectProposals(g.views.newMarkets, g.votesData, skipper)
	g.mu.RUnlock()

	return result
}

// GetUpdateMarketProposals returns proposals aiming to update existing market
func (g *Governance) GetUpdateMarketProposals(marketID string, inState *types.Proposal_State) []*types.GovernanceData {
	var result []*types.GovernanceData
	g.mu.RLock()

	if len(marketID) == 0 { // all market updates
		for _, updates := range g.views.marketUpdates {
			for _, p := range updates {
				if inState == nil || p.State == *inState {
					result = append(result, makeGovernanceData(p, g.votesData[p.ID]))
				}
			}
		}
	} else if updates, exists := g.views.marketUpdates[marketID]; exists {
		result = collectProposals(updates, g.votesData, selectInState(inState))
	}
	g.mu.RUnlock()
	return result
}

// GetNetworkParametersProposals returns proposals aiming to update network
func (g *Governance) GetNetworkParametersProposals(inState *types.Proposal_State) []*types.GovernanceData {
	skipper := selectInState(inState)

	g.mu.RLock()
	result := collectProposals(g.views.networkUpdates, g.votesData, skipper)
	g.mu.RUnlock()

	return result
}

// GetNewAssetProposals returns proposals aiming to create new assets
func (g *Governance) GetNewAssetProposals(inState *types.Proposal_State) []*types.GovernanceData {
	skipper := selectInState(inState)

	g.mu.RLock()
	result := collectProposals(g.views.newAssets, g.votesData, skipper)
	g.mu.RUnlock()

	return result
}

func (g *Governance) consume(ctx context.Context) {
	defer func() {
		g.Stop()
		g.pch = nil
		g.vch = nil
	}()
	for {
		select {
		case <-ctx.Done():
			return
		case proposals, ok := <-g.pch:
			if !ok { // channel is closed
				return
			}
			if len(proposals) > 0 { // expect empty slices in testing
				g.storeProposals(proposals)
			}
		case votes, ok := <-g.vch:
			if !ok {
				return
			}
			if len(votes) > 0 { // expect empty slices in testing
				g.storeVotes(votes)
			}
		}
	}
}

func (g *Governance) storeProposals(proposals []types.Proposal) {
	added := make([]types.GovernanceData, len(proposals))

	g.mu.Lock()

	for i, p := range proposals {
		p := p
		g.proposalsData[p.ID] = &p

		g.views.addProposal(&p)
		added[i] = *makeGovernanceData(&p, g.votesData[p.ID])
	}

	g.mu.Unlock()

	go g.subs.notifyAll(added)
	go g.subs.notifyProposals(added)
}

func (g *Governance) storeVotes(votes []types.Vote) {
	var proposals []types.GovernanceData

	g.mu.Lock()
	for _, v := range votes {
		v := v
		datum, exists := g.votesData[v.ProposalID]
		if !exists {
			datum = newVotes()
			g.votesData[v.ProposalID] = datum
		}
		datum.store(v)
		g.views.addVote(&v)

		if p, exists := g.proposalsData[v.ProposalID]; exists {
			proposals = append(proposals, *makeGovernanceData(p, g.votesData[v.ProposalID]))
		}
	}
	g.mu.Unlock()

	go g.subs.notifyAll(proposals)
	go g.subs.notifyVotes(votes)
}

type filterProposals = func(*types.Proposal) bool

func selectInState(inState *types.Proposal_State) *filterProposals {
	if inState == nil {
		return nil
	}
	impl := func(proposal *types.Proposal) bool {
		return proposal.State != *inState
	}
	return &impl
}

func collectProposals(p []*types.Proposal, v map[string]*proposalVotes, skip *filterProposals) []*types.GovernanceData {
	var result []*types.GovernanceData
	if skip == nil {
		result = make([]*types.GovernanceData, 0, len(p))
	}
	for _, i := range p {
		if skip == nil || !(*skip)(i) {
			result = append(result, makeGovernanceData(i, v[i.ID]))
		}
	}
	return result
}

type searchViews struct {
	partyVotes map[string][]*types.Vote // partyId : votes
	// typed slices
	newMarkets     []*types.Proposal
	marketUpdates  map[string][]*types.Proposal // marketID : []proposals
	networkUpdates []*types.Proposal
	newAssets      []*types.Proposal
}

func newViews() searchViews {
	return searchViews{
		partyVotes:    map[string][]*types.Vote{},
		marketUpdates: map[string][]*types.Proposal{},
	}
}

func (s *searchViews) addProposal(proposal *types.Proposal) {
	switch proposal.Terms.Change.(type) {
	case *types.ProposalTerms_NewMarket:
		s.newMarkets = append(s.newMarkets, proposal)
	case *types.ProposalTerms_UpdateMarket:
		//TODO: add real market id once the update proposals are implemented
		s.marketUpdates[""] = append(s.marketUpdates[""], proposal)
	case *types.ProposalTerms_UpdateNetwork:
		s.networkUpdates = append(s.networkUpdates, proposal)
	case *types.ProposalTerms_NewAsset:
		s.newAssets = append(s.newAssets, proposal)
	}
}

func (s *searchViews) addVote(vote *types.Vote) {
	// all votes are tracked and never removed even if party submitted opposing vote
	// this is different to the votes stored for proposals
	s.partyVotes[vote.PartyID] = append(s.partyVotes[vote.PartyID], vote)
}

type proposalVotes map[types.Vote_Value]map[string]types.Vote

func newVotes() *proposalVotes {
	return &proposalVotes{
		types.Vote_VALUE_YES: map[string]types.Vote{},
		types.Vote_VALUE_NO:  map[string]types.Vote{},
	}
}

// since proposalVotes can hold one of two values,
// the function will only attempt removing opposite value
func (v *proposalVotes) removeOld(partyID string, newValue types.Vote_Value) {
	opposite := types.Vote_VALUE_NO
	if newValue == opposite {
		opposite = types.Vote_VALUE_YES
	}
	delete((*v)[opposite], partyID)
}

func (v *proposalVotes) store(vote types.Vote) {
	(*v)[vote.Value][vote.PartyID] = vote
	v.removeOld(vote.PartyID, vote.Value)
}

func (v *proposalVotes) getVotes(proposalID string, value types.Vote_Value) []*types.Vote {
	if v == nil || len(*v) == 0 {
		return nil
	}
	result := make([]*types.Vote, 0, len((*v)[value]))
	for _, vote := range (*v)[value] {
		vote := vote
		result = append(result, &vote)
	}
	return result
}

func makeGovernanceData(proposal *types.Proposal, v *proposalVotes) *types.GovernanceData {
	// copy whole proposal to avoid data races
	copy := *proposal
	return &types.GovernanceData{
		Proposal: &copy,
		Yes:      v.getVotes(proposal.ID, types.Vote_VALUE_YES),
		No:       v.getVotes(proposal.ID, types.Vote_VALUE_NO),
	}
}
