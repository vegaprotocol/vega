package plugins

import (
	"context"
	"sync"

	types "code.vegaprotocol.io/vega/proto"

	"github.com/pkg/errors"
)

var (
	ErrProposalNotFound = errors.New("proposal not found")
)

// PropVote combines proposal and votes
type PropVote struct {
	types.Proposal
	Votes map[types.Vote_Value][]types.Vote
}

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
	vData      map[string]map[types.Vote_Value][]types.Vote
}

// NewProposal - return a new proposal plugin
func NewProposals(p PropBuffer, v VoteBuffer) *Proposals {
	return &Proposals{
		props:  p,
		votes:  v,
		pData:  map[string]*types.Proposal{},
		pByRef: map[string]*types.Proposal{},
		vData:  map[string]map[types.Vote_Value][]types.Vote{},
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
			p.mu.Lock()
			for _, v := range proposals {
				p.pData[v.ID] = &v
				p.pByRef[v.Reference] = &v
				if _, ok := p.vData[v.ID]; !ok {
					p.vData[v.ID] = map[types.Vote_Value][]types.Vote{}
				}
			}
			p.mu.Unlock()
		case votes, ok := <-p.vch:
			if !ok {
				return
			}
			// empty slices are used for testing
			if len(votes) == 0 {
				continue
			}
			p.mu.Lock()
			for _, v := range votes {
				pvotes, ok := p.vData[v.ProposalID]
				if !ok {
					pvotes = map[types.Vote_Value][]types.Vote{}
				}
				vSlice, ok := pvotes[v.Value]
				if !ok {
					vSlice = []types.Vote{}
				}
				pvotes[v.Value] = append(vSlice, v)
				p.vData[v.ProposalID] = pvotes
			}
			p.mu.Unlock()
		}
	}
}

// GetProposalByReference returns proposal and votes by reference (or error if proposal not found)
func (p *Proposals) GetProposalByReference(ref string) (*PropVote, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if v, ok := p.pByRef[ref]; ok {
		ret := PropVote{
			Proposal: *v,
		}
		ret.Votes = p.vData[v.ID]
		return &ret, nil
	}
	return nil, ErrProposalNotFound
}

// GetProposalByID returns proposal and votes by ID
func (p *Proposals) GetProposalByID(id string) (*PropVote, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if v, ok := p.pData[id]; ok {
		ret := PropVote{
			Proposal: *v,
		}
		ret.Votes = p.vData[id]
		return &ret, nil
	}
	return nil, ErrProposalNotFound
}

// GetOpenProposals returns proposals + current votes
func (p *Proposals) GetOpenProposals() []PropVote {
	p.mu.RLock()
	ret := []PropVote{}
	for _, prop := range p.pData {
		if prop.State == types.Proposal_OPEN {
			pv := PropVote{
				Proposal: *prop,
			}
			pv.Votes = p.vData[prop.ID]
			ret = append(ret, pv)
		}
	}
	p.mu.RUnlock()
	return ret
}
