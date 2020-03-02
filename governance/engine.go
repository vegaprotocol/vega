package governance

import (
	"sync"
	"time"

	"code.vegaprotocol.io/vega/logging"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/pkg/errors"
)

var (
	ErrProposalNotFound    = errors.New("proposal not found")
	ErrProposalIsDuplicate = errors.New("proposal with given ID already exists")
)

const (
	StatusOpen     ProposalStatus = "open"
	StatusPassed   ProposalStatus = "passed"
	StatusRejected ProposalStatus = "rejected"
	StatusEnacted  ProposalStatus = "enacted"
	StatusFailed   ProposalStatus = "failed"
)

// ProposalStatus ...
type ProposalStatus string

// Accounts ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/accounts_mock.go -package mocks code.vegaprotocol.io/vega/governance Accounts
type Accounts interface {
	GetPartyTokenAccount(id string) (*types.Account, error)
	GetTotalTokens() int64
}

// Buffer ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/proposal_buffer_mock.go -package mocks code.vegaprotocol.io/vega/governance Buffer
type Buffer interface {
	Add(Proposal)
	Flush()
}

type Engine struct {
	Config
	accs         Accounts
	buf          Buffer
	log          *logging.Logger
	mu           sync.Mutex
	currentTime  int64
	proposals    map[string]*Proposal
	proposalRefs map[string]*Proposal
}

type Vote struct {
	id, party string
	yes       bool
}

// Proposal placeholder type
type Proposal struct {
	id, reference string
	percentage    float64
	yes, no       []Vote // when no votes reaches 100 - percentage + 1 or yes reaches %+1, we know what to do
	ttl           int64
	validUntil    int64
	status        ProposalStatus
	approved      bool // this will be a special type
	err           error
}

func NewEngine(log *logging.Logger, cfg Config, accs Accounts, buf Buffer, now time.Time) *Engine {
	return &Engine{
		Config:      cfg,
		accs:        accs,
		buf:         buf,
		log:         log,
		currentTime: now.UnixNano(),
		proposals:   map[string]*Proposal{},
	}
}

// ReloadConf updates the internal configuration of the collateral engine
func (e *Engine) ReloadConf(cfg Config) {
	e.log.Info("reloading configuration")
	if e.log.GetLevel() != cfg.Level.Get() {
		e.log.Info("updating log level",
			logging.String("old", e.log.GetLevel().String()),
			logging.String("new", cfg.Level.String()),
		)
		e.log.SetLevel(cfg.Level.Get())
	}

	e.mu.Lock()
	e.Config = cfg
	e.mu.Unlock()
}

// OnChainUpdate - update curtime, expire proposals
func (e *Engine) OnChainTimeUpdate(t time.Time) []*Proposal {
	e.currentTime = t.UnixNano()
	expired := []*Proposal{}
	for k, p := range e.proposals {
		// only if we're passed the valid unitl, letting in the last of the votes
		if p.validUntil < e.currentTime {
			expired = append(expired, p)
			delete(e.proposals, k)
			delete(e.proposalRefs, p.reference) // remove from ref map, too
		}
	}
	return e.checkProposals(expired)
}

func (e *Engine) AddProposal(p Proposal) error {
	// @TODO -> we probably should keep proposals in memory here
	if cp, ok := e.proposals[p.id]; ok && cp.status == p.status {
		return ErrProposalIsDuplicate
	}
	if len(p.yes) == 0 {
		// ensure slice exists
		p.yes = []Vote{}
	}
	if len(p.no) == 0 {
		p.no = []Vote{}
	}
	// if the proposal is no longer open, remove from active pool
	if p.status != StatusOpen {
		delete(e.proposals, p.id)
		delete(e.proposalRefs, p.reference)
	} else {
		e.proposals[p.id] = &p
		e.proposalRefs[p.reference] = &p
	}
	e.buf.Add(p)
	return nil
}

func (e *Engine) AddVote(v Vote) error {
	p, ok := e.proposals[v.id]
	if !ok {
		return ErrProposalNotFound
	}
	if v.yes {
		p.yes = append(p.yes, v)
	} else {
		p.no = append(p.no, v)
	}
	e.buf.Add(*p)
	return nil
}

func (e *Engine) checkProposals(proposals []*Proposal) []*Proposal {
	accepted := make([]*Proposal, 0, len(proposals))
	buf := map[string]*types.Account{}
	for _, p := range proposals {
		totalYES := int64(0)
		for _, v := range p.yes {
			tok, ok := buf[v.party]
			if !ok {
				tok, p.err = e.accs.GetPartyTokenAccount(v.party)
				if p.err != nil {
					e.log.Error(
						"Failed to get account for party",
						logging.String("party-id", v.party),
						logging.Error(p.err),
					)
					break
				}
			}
			totalYES += tok.Balance
		}
		if p.err == nil {
			req := float64(e.accs.GetTotalTokens()) * p.percentage
			// percentage should be N/100 so we can multiply the total by this value and get the answer
			p.approved = (req <= float64(totalYES)) // N% of total votes should be reached
		}
		p.status = StatusRejected
		if p.approved {
			accepted = append(accepted, p)
			p.status = StatusPassed
		}
		e.buf.Add(*p)
	}
	return accepted
}
