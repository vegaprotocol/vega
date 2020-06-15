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
	// Validation errors

	ErrProposalCloseTimeTooSoon   = errors.New("proposal closes too soon")
	ErrProposalCloseTimeTooLate   = errors.New("proposal closes too late")
	ErrProposalEnactTimeTooSoon   = errors.New("proposal enactment times too soon")
	ErrProposalEnactTimeTooLate   = errors.New("proposal enactment times too late")
	ErrProposalInsufficientTokens = errors.New("proposal requires more tokens than party has")

	ErrVoterInsufficientTokens = errors.New("vote requires more tokens than party has")
	ErrVotePeriodExpired       = errors.New("proposal voting has been closed")
)

// Accounts ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/accounts_mock.go -package mocks code.vegaprotocol.io/vega/governance Accounts
type Accounts interface {
	GetPartyTokenAccount(id string) (*types.Account, error)
	GetTotalTokens() uint64
}

// Buffer ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/proposal_buffer_mock.go -package mocks code.vegaprotocol.io/vega/governance Buffer
type Buffer interface {
	Add(types.Proposal)
}

// VoteBuf...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/vote_buffer_mock.go -package mocks code.vegaprotocol.io/vega/governance VoteBuf
type VoteBuf interface {
	Add(types.Vote)
}

type Engine struct {
	Config
	accs          Accounts
	buf           Buffer
	vbuf          VoteBuf
	log           *logging.Logger
	mu            sync.Mutex
	currentTime   time.Time
	proposals     map[string]*governanceData
	proposalRefs  map[string]*governanceData
	networkParams NetworkParameters
}

type governanceData struct {
	*types.Proposal
	yes map[string]*types.Vote
	no  map[string]*types.Vote
}

func NewEngine(log *logging.Logger, cfg Config, params *NetworkParameters, accs Accounts, buf Buffer, vbuf VoteBuf, now time.Time) *Engine {
	log.Debug("Governance parameters",
		logging.String("MinClose", params.minClose.String()),
		logging.String("MaxClose", params.maxClose.String()),
		logging.String("MinEnact", params.minEnact.String()),
		logging.String("MaxEnact", params.maxEnact.String()),
		logging.Uint64("MinParticipationStake", params.minParticipationStake),
	)
	return &Engine{
		Config:        cfg,
		accs:          accs,
		buf:           buf,
		vbuf:          vbuf,
		log:           log,
		currentTime:   now,
		proposals:     map[string]*governanceData{},
		proposalRefs:  map[string]*governanceData{},
		networkParams: *params,
	}
}

// ReloadConf updates the internal configuration of the governance engine
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

// OnChainTimeUpdate - update curtime, expire proposals
func (e *Engine) OnChainTimeUpdate(t time.Time) []*types.Proposal {
	e.currentTime = t
	now := t.Unix()

	totalStake := e.accs.GetTotalTokens()
	counter := newStakeCounter(e.log, e.accs)

	var toBeEnacted []*types.Proposal
	for k, p := range e.proposals {
		if p.Terms.ClosingTimestamp < now {
			e.closeProposal(p, counter, totalStake)
		}
		if p.State == types.Proposal_STATE_PASSED && p.Terms.EnactmentTimestamp < now {
			toBeEnacted = append(toBeEnacted, p.Proposal)
			delete(e.proposals, k)
			delete(e.proposalRefs, p.Reference)
		}
	}
	return toBeEnacted
}

func (e *Engine) AddProposal(p types.Proposal) error {
	// @TODO -> we probably should keep proposals in memory here
	if cp, ok := e.proposals[p.ID]; ok && cp.State == p.State {
		return ErrProposalIsDuplicate
	}
	if _, ok := e.proposalRefs[p.Reference]; ok {
		return ErrProposalIsDuplicate
	}
	var err error
	if err = e.validateProposal(p); err != nil {
		p.State = types.Proposal_STATE_REJECTED
	}
	if p.State != types.Proposal_STATE_OPEN {
		delete(e.proposals, p.ID)
		delete(e.proposalRefs, p.Reference)
	} else {
		pv := governanceData{
			Proposal: &p,
			yes:      map[string]*types.Vote{},
			no:       map[string]*types.Vote{},
		}
		e.proposals[p.ID] = &pv
		e.proposalRefs[p.Reference] = &pv
	}
	e.buf.Add(p)
	return err
}

func (e *Engine) validateProposal(p types.Proposal) error {
	tok, err := e.accs.GetPartyTokenAccount(p.PartyID)
	if err != nil {
		return err
	}
	if tok.Balance < 1 {
		return ErrProposalInsufficientTokens
	}
	if p.Terms.ClosingTimestamp < e.currentTime.Add(e.networkParams.minClose).Unix() {
		return ErrProposalCloseTimeTooSoon
	}
	if p.Terms.ClosingTimestamp > e.currentTime.Add(e.networkParams.maxClose).Unix() {
		return ErrProposalCloseTimeTooLate
	}
	if p.Terms.EnactmentTimestamp < e.currentTime.Add(e.networkParams.minEnact).Unix() {
		return ErrProposalEnactTimeTooSoon
	}
	if p.Terms.EnactmentTimestamp > e.currentTime.Add(e.networkParams.maxEnact).Unix() {
		return ErrProposalEnactTimeTooLate
	}

	return nil
}

func (e *Engine) AddVote(v types.Vote) error {
	p, err := e.validateVote(v)
	if err != nil {
		return err
	}
	// we only want to count the last vote, so add to yes/no map, delete from the other
	// if the party hasn't cast a vote yet, the delete is just a noop
	if v.Value == types.Vote_VALUE_YES {
		delete(p.no, v.PartyID)
		p.yes[v.PartyID] = &v
	} else {
		delete(p.yes, v.PartyID)
		p.no[v.PartyID] = &v
	}
	e.vbuf.Add(v)
	return nil
}

func (e *Engine) validateVote(v types.Vote) (*governanceData, error) {
	tacc, err := e.accs.GetPartyTokenAccount(v.PartyID)
	if err != nil {
		return nil, err
	}
	if tacc.Balance == 0 {
		return nil, ErrVoterInsufficientTokens
	}
	p, ok := e.proposals[v.ProposalID]
	if !ok {
		return nil, ErrProposalNotFound
	}
	if p.Terms.ClosingTimestamp < e.currentTime.Unix() {
		return nil, ErrVotePeriodExpired
	}
	return p, nil
}

func (e *Engine) closeProposal(data *governanceData, counter *stakeCounter, totalStake uint64) {
	proposal := data.Proposal

	yes := counter.countVotes(data.yes)
	no := counter.countVotes(data.no)

	proposal.State = types.Proposal_STATE_DECLINED
	if yes > no {
		participationStake := float64(yes + no)
		minParticipationStake := float64(proposal.Terms.MinParticipationStake*totalStake) / 100
		if participationStake >= minParticipationStake {
			proposal.State = types.Proposal_STATE_PASSED
		}
	}
	e.buf.Add(*proposal)
}

// stakeCounter caches token balance per party and counts votes
// reads from accounts on every miss and does not have expiration policy
type stakeCounter struct {
	log      *logging.Logger
	accounts Accounts
	balances map[string]uint64
}

func newStakeCounter(log *logging.Logger, accounts Accounts) *stakeCounter {
	return &stakeCounter{
		log:      log,
		accounts: accounts,
		balances: map[string]uint64{},
	}
}
func (s *stakeCounter) countVotes(votes map[string]*types.Vote) uint64 {
	var tally uint64
	for _, v := range votes {
		tally += s.getTokens(v.PartyID)
	}
	return tally
}

func (s *stakeCounter) getTokens(partyID string) uint64 {
	if balance, found := s.balances[partyID]; found {
		return balance
	}
	account, err := s.accounts.GetPartyTokenAccount(partyID)
	if err != nil {
		s.log.Error(
			"Failed to get account for party",
			logging.String("party-id", partyID),
			logging.Error(err),
		)
		// not much we can do with the error as there is nowhere to buble up the error on tick
		return 0
	}
	s.balances[partyID] = account.Balance
	return account.Balance
}
