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

//go:generate go run github.com/golang/mock/mockgen -destination mocks/accounts_mock.go -package mocks code.vegaprotocol.io/vega/governance Accounts
type Accounts interface {
	GetPartyTokenAccount(id string) (*types.Account, error)
	GetTotalTokens() int64
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/prop_buf_mock.go -package mocks code.vegaprotocol.io/vega/governance ProposalBuf
type ProposalBuf interface {
	Add(Proposal)
	Flush()
}

type Engine struct {
	Config
	accs        Accounts
	buf         ProposalBuf
	log         *logging.Logger
	mu          sync.Mutex
	currentTime int64
	proposals   map[string]*Proposal
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
	approved      bool // this will be a special type
	err           error
}

func NewEngine(log *logging.Logger, cfg Config, accs Accounts, buf ProposalBuf, now time.Time) *Engine {
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
func (e *Engine) OnChainTimeUpdate(t time.Time) {
	e.currentTime = t.UnixNano()
	expired := []*Proposal{}
	for k, p := range e.proposals {
		// only if we're passed the valid unitl, letting in the last of the votes
		if p.validUntil < e.currentTime {
			expired = append(expired, p)
			delete(e.proposals, k)
		}
	}
	// @TODO this is just a hack, we should make sure we're not going to flush at the wrong time
	go func() {
		e.checkProposals(expired)
		e.buf.Flush()
	}()
}

func (e *Engine) AddProposal(p Proposal) error {
	_, ok := e.proposals[p.id]
	if ok {
		return ErrProposalIsDuplicate
	}
	if len(p.yes) == 0 {
		// ensure slice exists
		p.yes = []Vote{}
	}
	if len(p.no) == 0 {
		p.no = []Vote{}
	}
	e.proposals[p.id] = &p
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
	return nil
}

func (e *Engine) checkProposals(proposals []*Proposal) {
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
		e.buf.Add(*p)
	}
}
