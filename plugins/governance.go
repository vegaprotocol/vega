package plugins

import (
	"context"
	"sync"
	"sync/atomic"

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

// threaded contention points:
// - Proposal - can be updated if votes are recorded before the proposal
// - Yes - updated on receiving a yes vote
// - No - updated on receiving a no vote
type governanceData types.GovernanceData

func newProposal(proposal *types.Proposal) *governanceData {
	return &governanceData{
		Proposal: proposal,
	}
}

// used for the case when vote arrives bofore the proposal
func newDanglingVote(vote *types.Vote) *governanceData {
	result := &governanceData{}
	result.addVote(vote)
	return result
}

func (d *governanceData) addVote(vote *types.Vote) {
	if vote.Value == types.Vote_YES {
		d.Yes = append(d.Yes, vote)
	} else if vote.Value == types.Vote_NO {
		d.No = append(d.No, vote)
	}
}

func cloneVotes(from []*types.Vote) []*types.Vote {
	result := make([]*types.Vote, len(from))
	for i, v := range from {
		result[i] = v // vote itself considered immutable
	}
	return result
}

func (d *governanceData) isDangling() bool {
	return d.Proposal == nil
}

func (d *governanceData) clone() *types.GovernanceData {
	if d.Proposal != nil {
		proposal := *d.Proposal
		return &types.GovernanceData{
			Proposal: &proposal,
			Yes:      cloneVotes(d.Yes),
			No:       cloneVotes(d.No),
		}
	}
	return nil
}

type filterProposals = func(*governanceData) bool

func selectInState(inState *types.Proposal_State) *filterProposals {
	if inState == nil {
		return nil
	}
	impl := func(proposal *governanceData) bool {
		return proposal.Proposal.State != *inState
	}
	return &impl
}

func cloneProposals(data []*governanceData, skip *filterProposals) []*types.GovernanceData {
	var result []*types.GovernanceData
	if skip == nil {
		result = make([]*types.GovernanceData, 0, len(data))
	}
	for _, v := range data {
		if !v.isDangling() && (skip == nil || !(*skip)(v)) {
			result = append(result, v.clone())
		}
	}
	return result
}

type searchViews struct {
	// party id to votes view
	partyVotes map[string][]*types.Vote
	// typed slices
	newMarkets     []*governanceData
	marketUpdates  map[string][]*governanceData
	networkUpdates []*governanceData
	newAssets      []*governanceData
}

func newViews() searchViews {
	return searchViews{
		partyVotes:    map[string][]*types.Vote{},
		marketUpdates: map[string][]*governanceData{},
	}
}

func (s searchViews) addProposal(data *governanceData) {
	switch data.Proposal.Terms.Change.(type) {
	case *types.ProposalTerms_NewMarket:
		s.newMarkets = append(s.newMarkets, data)
	//case *types.ProposalTerms_UpdateMarket:
	//TODO:
	case *types.ProposalTerms_UpdateNetwork:
		s.networkUpdates = append(s.networkUpdates, data)
	case *types.ProposalTerms_NewAsset:
		s.newAssets = append(s.newAssets, data)
	}
}

func (s searchViews) addVote(vote *types.Vote) {
	s.partyVotes[vote.PartyID] = append(s.partyVotes[vote.PartyID], vote)
}

type streams struct {
	overall      map[int64]chan []types.GovernanceData
	overallCount int64

	partyProposals      map[string]map[int64]chan []types.GovernanceData
	partyProposalsCount int64 // flat counter across all parties

	partyVotes      map[string]map[int64]chan []types.Vote
	partyVotesCount int64 // flat counter across all parties

	proposalVotes      map[string]map[int64]chan []types.Vote
	proposalVotesCount int64 // flat counter across all proposals
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
func (s streams) notifyAll(proposals []types.GovernanceData) {
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
}

func partitionProposalsByParty(proposals []types.GovernanceData) map[string][]types.GovernanceData {
	result := map[string][]types.GovernanceData{}
	for _, v := range proposals {
		result[v.Proposal.ID] = append(result[v.Proposal.ID], v)
	}
	return result
}

// notifications for proposal updates (no votes)
func (s streams) notifyProposals(data []types.GovernanceData) {
	byParty := partitionProposalsByParty(data)

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
func (s streams) notifyVotes(votes []types.Vote) {
	byParty := partitionVotesByParty(votes)
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
	byProposal := partitionVotesByProposalID(votes)
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
}

// Governance stores governance data
type Governance struct {
	mu         sync.RWMutex
	props      PropBuffer
	votes      VoteBuffer
	pref, vref int
	pch        <-chan []types.Proposal
	vch        <-chan []types.Vote

	data  map[string]*governanceData
	views searchViews

	subs streams
}

// NewGovernance - return a new governance plugin
func NewGovernance(p PropBuffer, v VoteBuffer) *Governance {
	return &Governance{
		props: p,
		votes: v,
		data:  map[string]*governanceData{},
		views: newViews(),
		subs:  newStreams(),
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

func (g *Governance) storeProposals(proposals []types.Proposal) {
	added := make([]types.GovernanceData, len(proposals))

	g.mu.Lock()
	for i, v := range proposals {
		v := v
		datum, exists := g.data[v.ID]
		if exists {
			datum.Proposal = &v
		} else {
			datum = newProposal(&v)
			g.data[v.ID] = datum
		}
		g.views.addProposal(datum)
		added[i] = *datum.clone()
	}
	g.mu.Unlock()

	// notify the proposals have been stored
	go func() {
		g.mu.Lock()
		g.subs.notifyAll(added)
		g.subs.notifyProposals(added)
		g.mu.Unlock()
	}()
}

func (g *Governance) storeVotes(votes []types.Vote) {
	generalUpdate := make([]types.GovernanceData, len(votes))

	g.mu.Lock()
	for i, v := range votes {
		v := v
		datum, exists := g.data[v.ProposalID]
		if !exists { // create if does not exist
			datum = newDanglingVote(&v)
			g.data[v.ProposalID] = datum
		} else {
			datum.addVote(&v)
		}
		g.views.addVote(&v)

		// notify general channel only about votes that are received after the proposal (ignore dangling)
		// channel dedicated to votes will receive everything
		if !datum.isDangling() {
			generalUpdate[i] = *datum.clone()
		}
	}
	g.mu.Unlock()

	// notify the votes have been stored
	go func() {
		g.mu.Lock()
		g.subs.notifyAll(generalUpdate)
		g.subs.notifyVotes(votes)
		g.mu.Unlock()
	}()
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
			break
		case proposals, ok := <-g.pch:
			if !ok { // channel is closed
				return
			}
			if len(proposals) > 0 { // allow empty slices for testing
				g.storeProposals(proposals)
			}
		case votes, ok := <-g.vch:
			if !ok {
				return
			}
			if len(votes) > 0 { // allow empty slices for testing
				g.storeVotes(votes)
			}
		}
	}
}

// SubscribeAll streams all governance data
func (g *Governance) SubscribeAll() (<-chan []types.GovernanceData, int64) {
	k := atomic.AddInt64(&g.subs.overallCount, 1)

	g.mu.Lock()
	ch := make(chan []types.GovernanceData, 1)
	g.subs.overall[k] = ch
	g.mu.Unlock()

	return ch, k
}

// UnsubscribeAll removes governance data stream
func (g *Governance) UnsubscribeAll(k int64) {
	g.mu.Lock()
	if ch, ok := g.subs.overall[k]; ok {
		close(ch)
		delete(g.subs.overall, k)
	}
	g.mu.Unlock()
}

// SubscribePartyProposals streams proposals authored by the specific party
func (g *Governance) SubscribePartyProposals(partyID string) (<-chan []types.GovernanceData, int64) {
	k := atomic.AddInt64(&g.subs.partyProposalsCount, 1)

	g.mu.Lock()
	byPartySubs, exists := g.subs.partyProposals[partyID]
	if !exists {
		byPartySubs = map[int64]chan []types.GovernanceData{}
		g.subs.partyProposals[partyID] = byPartySubs
	}
	ch := make(chan []types.GovernanceData, 1)
	byPartySubs[k] = ch
	g.mu.Unlock()

	return ch, k
}

// UnsubscribePartyProposals removes stream of proposals for authored by the party
func (g *Governance) UnsubscribePartyProposals(partyID string, k int64) {
	g.mu.Lock()
	if subs, exists := g.subs.partyProposals[partyID]; exists {
		if ch, ok := subs[k]; ok {
			close(ch)
			delete(subs, k)
		}
	}
	g.mu.Unlock()
}

// SubscribePartyVotes streams all votes cast by the specific party
func (g *Governance) SubscribePartyVotes(partyID string) (<-chan []types.Vote, int64) {
	k := atomic.AddInt64(&g.subs.partyVotesCount, 1)

	g.mu.Lock()
	byPartySubs, exists := g.subs.partyVotes[partyID]
	if !exists {
		byPartySubs = map[int64]chan []types.Vote{}
		g.subs.partyVotes[partyID] = byPartySubs
	}
	ch := make(chan []types.Vote, 1)
	byPartySubs[k] = ch
	g.mu.Unlock()

	return ch, k
}

// UnsubscribePartyVotes removes stream of votes for the specific party
func (g *Governance) UnsubscribePartyVotes(partyID string, k int64) {
	g.mu.Lock()
	if subs, exists := g.subs.partyVotes[partyID]; exists {
		if ch, ok := subs[k]; ok {
			close(ch)
			delete(subs, k)
		}
	}
	g.mu.Unlock()
}

// SubscribeProposalVotes streams all votes cast for the specific proposal
func (g *Governance) SubscribeProposalVotes(proposalID string) (<-chan []types.Vote, int64) {
	k := atomic.AddInt64(&g.subs.proposalVotesCount, 1)

	g.mu.Lock()
	byProposalSubs, exists := g.subs.proposalVotes[proposalID]
	if !exists {
		byProposalSubs = map[int64]chan []types.Vote{}
		g.subs.proposalVotes[proposalID] = byProposalSubs
	}
	ch := make(chan []types.Vote, 1)
	byProposalSubs[k] = ch
	g.mu.Unlock()

	return ch, k
}

// UnsubscribeProposalVotes removes stream of votes for the proposal
func (g *Governance) UnsubscribeProposalVotes(proposalID string, k int64) {
	g.mu.Lock()
	if subs, exists := g.subs.proposalVotes[proposalID]; exists {
		if ch, ok := subs[k]; ok {
			close(ch)
			delete(subs, k)
		}
	}
	g.mu.Unlock()
}

// GetAllGovernanceData get all proposals and votes filtered by the state if specified
func (g *Governance) GetAllGovernanceData(inState *types.Proposal_State) []*types.GovernanceData {
	g.mu.RLock()
	result := make([]*types.GovernanceData, 0, len(g.data))
	for _, v := range g.data {
		if !v.isDangling() && (inState == nil || v.Proposal.State == *inState) {
			result = append(result, v.clone())
		}
	}
	g.mu.RUnlock()
	return result
}

// GetProposalsByParty returns proposals (+votes) by the party that authoring them
func (g *Governance) GetProposalsByParty(partyID string, inState *types.Proposal_State) []*types.GovernanceData {
	var result []*types.GovernanceData
	g.mu.RLock()
	for _, v := range g.data {
		if !v.isDangling() && v.Proposal.PartyID == partyID && (inState == nil || v.Proposal.State == *inState) {
			result = append(result, v.clone())
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

	if datum, exists := g.data[id]; exists && !datum.isDangling() {
		return datum.clone(), nil
	}
	return nil, ErrProposalNotFound
}

// GetProposalByReference returns proposal and votes by reference
func (g *Governance) GetProposalByReference(ref string) (*types.GovernanceData, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	for _, v := range g.data {
		if !v.isDangling() && v.Proposal.Reference == ref {
			return v.clone(), nil
		}
	}
	return nil, ErrProposalNotFound
}

// GetNewMarketProposals returns proposals aiming to create new markets
func (g *Governance) GetNewMarketProposals(inState *types.Proposal_State) []*types.GovernanceData {
	skipper := selectInState(inState)

	g.mu.RLock()
	result := cloneProposals(g.views.newMarkets, skipper)
	g.mu.RUnlock()

	return result
}

// GetUpdateMarketProposals returns proposals aiming to update existing market
func (g *Governance) GetUpdateMarketProposals(marketID string, inState *types.Proposal_State) []*types.GovernanceData {
	var result []*types.GovernanceData
	g.mu.RLock()

	if len(marketID) == 0 { // all market updates
		for _, updates := range g.views.marketUpdates {
			for _, v := range updates {
				if !v.isDangling() && (inState == nil || v.Proposal.State == *inState) {
					result = append(result, v.clone())
				}
			}
		}
	} else if updates, exists := g.views.marketUpdates[marketID]; exists {
		result = cloneProposals(updates, selectInState(inState))
	}
	g.mu.RUnlock()
	return result
}

// GetNetworkParametersProposals returns proposals aiming to update network
func (g *Governance) GetNetworkParametersProposals(inState *types.Proposal_State) []*types.GovernanceData {
	skipper := selectInState(inState)

	g.mu.RLock()
	result := cloneProposals(g.views.networkUpdates, skipper)
	g.mu.RUnlock()

	return result
}
