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
	mu         sync.RWMutex
	props      PropBuffer
	votes      VoteBuffer
	pref, vref int
	pch        <-chan []types.Proposal
	vch        <-chan []types.Vote
	pData      map[string]*types.Proposal
	pByRef     map[string]*types.Proposal
	vData      map[string]map[types.Vote_Value]map[string]types.Vote // nested map by proposal -> vote value -> party

	// stream subscriptions
	subs     map[int64]chan []types.ProposalVote
	subCount int64
}

// NewProposal - return a new proposal plugin
func NewProposals(p PropBuffer, v VoteBuffer) *Proposals {
	return &Proposals{
		props:  p,
		votes:  v,
		pData:  map[string]*types.Proposal{},
		pByRef: map[string]*types.Proposal{},
		vData:  map[string]map[types.Vote_Value]map[string]types.Vote{},
		subs:   map[int64]chan []types.ProposalVote{},
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
				p.pData[v.ID] = &v
				p.pByRef[v.Reference] = &v
				if _, ok := p.vData[v.ID]; !ok {
					p.vData[v.ID] = map[types.Vote_Value]map[string]types.Vote{
						types.Vote_YES: map[string]types.Vote{},
						types.Vote_NO:  map[string]types.Vote{},
					}
				}
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
					pvotes = map[types.Vote_Value]map[string]types.Vote{
						types.Vote_YES: map[string]types.Vote{},
						types.Vote_NO:  map[string]types.Vote{},
					}
				}
				// value maps always exist
				pvotes[v.Value][v.PartyID] = v
				oppositeValue := types.Vote_NO
				if v.Value == oppositeValue {
					oppositeValue = types.Vote_YES
				}
				delete(pvotes[oppositeValue], v.PartyID)
				p.vData[v.ProposalID] = pvotes
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
	data := make([]types.ProposalVote, 0, len(ids))
	for _, id := range ids {
		if prop, ok := p.pData[id]; ok {
			data = append(data, p.getPropVote(*prop))
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

// Subscribe- get all changes to proposals/votes
func (p *Proposals) Subscribe() (<-chan []types.ProposalVote, int64) {
	p.mu.Lock()
	k := atomic.AddInt64(&p.subCount, 1)
	ch := make(chan []types.ProposalVote, 1)
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

// GetProposalByReference returns proposal and votes by reference (or error if proposal not found)
func (p *Proposals) GetProposalByReference(ref string) (*types.ProposalVote, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if v, ok := p.pByRef[ref]; ok {
		ret := p.getPropVote(*v)
		return &ret, nil
	}
	return nil, ErrProposalNotFound
}

// GetProposalByID returns proposal and votes by ID
func (p *Proposals) GetProposalByID(id string) (*types.ProposalVote, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if v, ok := p.pData[id]; ok {
		ret := p.getPropVote(*v)
		return &ret, nil
	}
	return nil, ErrProposalNotFound
}

// GetProposals get all proposals
func (p *Proposals) GetProposals() []types.ProposalVote {
	p.mu.RLock()
	ret := make([]types.ProposalVote, 0, len(p.pData))
	for _, prop := range p.pData {
		ret = append(ret, p.getPropVote(*prop))
	}
	p.mu.RUnlock()
	return ret
}

// GetOpenProposals returns proposals + current votes
func (p *Proposals) GetOpenProposals() []types.ProposalVote {
	p.mu.RLock()
	ret := []types.ProposalVote{}
	for _, prop := range p.pData {
		if prop.State == types.Proposal_OPEN {
			ret = append(ret, p.getPropVote(*prop))
		}
	}
	p.mu.RUnlock()
	return ret
}

func (p *Proposals) getPropVote(v types.Proposal) types.ProposalVote {
	vData := p.vData[v.ID]
	yes := make([]*types.Vote, 0, len(vData[types.Vote_YES]))
	no := make([]*types.Vote, 0, len(vData[types.Vote_NO]))
	for _, vote := range vData[types.Vote_YES] {
		cpy := vote
		yes = append(yes, &cpy)
	}
	for _, vote := range vData[types.Vote_NO] {
		cpy := vote
		no = append(no, &cpy)
	}
	return types.ProposalVote{
		Proposal: &v,
		Yes:      yes,
		No:       no,
	}
}
