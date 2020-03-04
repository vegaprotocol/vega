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
	Add(types.Proposal)
	Flush()
}

type network struct {
	minClose, maxClose, minEnact, maxEnact int64
	participation                          uint64
}

type Engine struct {
	Config
	accs         Accounts
	buf          Buffer
	log          *logging.Logger
	mu           sync.Mutex
	currentTime  int64
	proposals    map[string]*proposalVote
	proposalRefs map[string]*proposalVote
	net          Network
}

type proposalVote struct {
	*types.Proposal
	yes []*types.Vote
	no  []*types.Vote
}

func NewEngine(log *logging.Logger, cfg Config, accs Accounts, buf Buffer, now time.Time) *Engine {
	return &Engine{
		Config:       cfg,
		accs:         accs,
		buf:          buf,
		log:          log,
		currentTime:  now.UnixNano(),
		proposals:    map[string]*proposalVote{},
		proposalRefs: map[string]*proposalVote{},
		net: Network{
			minClose:      cfg.DefaultMinClose,
			maxClose:      cfg.DefaultMaxClose,
			minEnact:      cfg.DefaultMinEnact,
			maxEnact:      cfg.DefaultMaxEnact,
			participation: cfg.DefaultMinParticipation,
		},
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
func (e *Engine) OnChainTimeUpdate(t time.Time) []*types.Proposal {
	e.currentTime = t.Unix()
	expired := []*types.Proposal{}
	for k, p := range e.proposals {
		// only if we're passed the valid unitl, letting in the last of the votes
		if p.Terms.ClosingTimestamp < e.currentTime {
			expired = append(expired, p.Proposal)
			delete(e.proposals, k)
			delete(e.proposalRefs, p.Reference) // remove from ref map, Foo
		}
	}
	return e.checkProposals(expired)
}

func (e *Engine) AddProposal(p types.Proposal) error {
	ts := types.ProposalTerms{}
	// @TODO -> we probably should keep proposals in memory here
	if cp, ok := e.proposals[p.ID]; ok && cp.State == p.State {
		return ErrProposalIsDuplicate
	}
	if p.State != types.Proposal_OPEN {
		delete(e.proposals, p.ID)
		delete(e.proposalRefs, p.Reference)
	} else {
		pv := proposalVote{
			Proposal: p,
			yes:      []*types.Vote{},
			no:       []*types.Vote{},
		}
		e.proposals[p.ID] = &pv
		e.proposalRefs[p.Reference] = &pv
	}
	e.buf.Add(p)
	return nil
}

func (e *Engine) AddVote(v types.Vote) error {
	p, ok := e.proposals[v.ProposalID]
	if !ok {
		return ErrProposalNotFound
	}
	if v.Value == types.Vote_YES {
		p.yes = append(p.yes, v)
	} else {
		p.no = append(p.no, v)
	}
	return nil
}

func (e *Engine) checkProposals(proposals []*proposalVote) []*types.Proposal {
	accepted := make([]*types.Proposal, 0, len(proposals))
	buf := map[string]*types.Account{}
	var err error
	for _, pw := range proposals {
		p := pw.Proposal
		totalYES := int64(0)
		for _, v := range pw.yes {
			tok, ok := buf[v.Voter]
			if !ok {
				tok, err = e.accs.GetPartyTokenAccount(v.Voter)
				if err != nil {
					e.log.Error(
						"Failed to get account for party",
						logging.String("party-id", v.Voter),
						logging.Error(err),
					)
					break
				}
			}
			totalYES += tok.Balance
		}
		p.State = types.Proposal_DECLINED
		if p.Terms.MinParticipationStake >= totalYES {
			p.State = types.Proposal_PASSED
			accepted = append(accepted, p)
		}
		e.buf.Add(*p)
	}
	return accepted
}
