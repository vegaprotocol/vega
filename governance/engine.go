package governance

import (
	"sync"
	"time"

	"code.vegaprotocol.io/vega/logging"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/pkg/errors"
)

var (
	ErrProposalInvalidState                    = errors.New("proposal state not valid, only open can be submitted")
	ErrProposalNotFound                        = errors.New("found no open proposal with the id")
	ErrProposalIsDuplicate                     = errors.New("proposal with given ID already exists")
	ErrProposalCloseTimeTooSoon                = errors.New("proposal closes too soon")
	ErrProposalCloseTimeTooLate                = errors.New("proposal closes too late")
	ErrProposalEnactTimeTooSoon                = errors.New("proposal enactment time is too soon")
	ErrProposalEnactTimeTooLate                = errors.New("proposal enactment time is too late")
	ErrProposalInsufficientTokens              = errors.New("party requires more tokens to submit a proposal")
	ErrProposalMinPaticipationStakeTooLow      = errors.New("proposal minimum participation stake is too low")
	ErrProposalMinPaticipationStakeInvalid     = errors.New("proposal minimum participation stake is out of bounds [0..1]")
	ErrProposalMinRequiredMajorityStakeTooLow  = errors.New("proposal minimum required majority stake is too low")
	ErrProposalMinRequiredMajorityStakeInvalid = errors.New("proposal minimum required majority stake is out of bounds [0.5..1]")
	ErrVoterInsufficientTokens                 = errors.New("vote requires more tokens than party has")
	ErrProposalPassed                          = errors.New("proposal has passed and can no longer be voted on")
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

// Engine is the governance engine that handles proposal and vote lifecycle.
type Engine struct {
	Config
	mu  sync.Mutex
	log *logging.Logger

	accounts    Accounts
	buf         Buffer
	vbuf        VoteBuf
	currentTime time.Time

	activeProposals map[string]*governanceData
	networkParams   NetworkParameters
}

type governanceData struct {
	*types.Proposal
	yes map[string]*types.Vote
	no  map[string]*types.Vote
}

// NewEngine creates new governance engine instance
func NewEngine(log *logging.Logger, cfg Config, params *NetworkParameters, accs Accounts, buf Buffer, vbuf VoteBuf, now time.Time) *Engine {
	log.Debug("Governance network parameters",
		logging.String("MinClose", params.minClose.String()),
		logging.String("MaxClose", params.maxClose.String()),
		logging.String("MinEnact", params.minEnact.String()),
		logging.String("MaxEnact", params.maxEnact.String()),
		logging.Float32("RequiredParticipation", params.requiredParticipation),
		logging.Float32("RequiredMajority", params.requiredMajority),
		logging.Float32("MinProposerBalance", params.minProposerBalance),
		logging.Float32("MinVoterBalance", params.minVoterBalance),
	)
	return &Engine{
		Config:          cfg,
		accounts:        accs,
		buf:             buf,
		vbuf:            vbuf,
		log:             log,
		currentTime:     now,
		activeProposals: map[string]*governanceData{},
		networkParams:   *params,
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

// OnChainTimeUpdate triggers time bound state changes.
func (e *Engine) OnChainTimeUpdate(t time.Time) []*types.Proposal {
	e.currentTime = t
	now := t.Unix()

	totalStake := e.accounts.GetTotalTokens()
	counter := newStakeCounter(e.log, e.accounts)

	var toBeEnacted []*types.Proposal
	for id, proposal := range e.activeProposals {
		if proposal.Terms.ClosingTimestamp < now {
			e.closeProposal(proposal, counter, totalStake)
		}

		if proposal.State != types.Proposal_STATE_OPEN && proposal.State != types.Proposal_STATE_PASSED {
			delete(e.activeProposals, id)
		} else if proposal.State == types.Proposal_STATE_PASSED && proposal.Terms.EnactmentTimestamp < now {
			toBeEnacted = append(toBeEnacted, proposal.Proposal)
			delete(e.activeProposals, id)
		}
	}
	return toBeEnacted
}

// SubmitProposal submits new proposal to the governance engine so it can be voted on, passed and enacted.
// Only open can be submitted and validated at this point. No further validation happens.
func (e *Engine) SubmitProposal(proposal types.Proposal) error {
	if _, exists := e.activeProposals[proposal.ID]; exists {
		return ErrProposalIsDuplicate // state is not allowed to change externally
	}
	// Proposals ought to be read from the chain only once: when the proposal submission transaction is processed.
	// After that they should be read from the core’s internal state (which can only be updated deterministically by transactions on the chain...
	if proposal.State == types.Proposal_STATE_OPEN {
		err := e.validateOpenProposal(proposal)
		if err != nil {
			proposal.State = types.Proposal_STATE_REJECTED
		} else {
			e.activeProposals[proposal.ID] = &governanceData{
				Proposal: &proposal,
				yes:      map[string]*types.Vote{},
				no:       map[string]*types.Vote{},
			}
		}
		e.buf.Add(proposal)
		return err
	}
	return ErrProposalInvalidState
}

// validates proposals read from the chain
func (e *Engine) validateOpenProposal(proposal types.Proposal) error {
	proposerTokens, err := getGovernanceTokens(e.accounts, proposal.PartyID)
	if err != nil {
		return err
	}
	totalTokens := e.accounts.GetTotalTokens()
	if float32(proposerTokens) < float32(totalTokens)*e.networkParams.minProposerBalance {
		return ErrProposalInsufficientTokens
	}
	if proposal.Terms.ClosingTimestamp < e.currentTime.Add(e.networkParams.minClose).Unix() {
		return ErrProposalCloseTimeTooSoon
	}
	if proposal.Terms.ClosingTimestamp > e.currentTime.Add(e.networkParams.maxClose).Unix() {
		return ErrProposalCloseTimeTooLate
	}
	if proposal.Terms.EnactmentTimestamp < e.currentTime.Add(e.networkParams.minEnact).Unix() {
		return ErrProposalEnactTimeTooSoon
	}
	if proposal.Terms.EnactmentTimestamp > e.currentTime.Add(e.networkParams.maxEnact).Unix() {
		return ErrProposalEnactTimeTooLate
	}
	return nil
}

// AddVote adds vote onto an existing active proposal (if found) so the proposal could pass and be enacted
func (e *Engine) AddVote(vote types.Vote) error {
	proposal, err := e.validateVote(vote)
	if err != nil {
		return err
	}
	// we only want to count the last vote, so add to yes/no map, delete from the other
	// if the party hasn't cast a vote yet, the delete is just a noop
	if vote.Value == types.Vote_VALUE_YES {
		delete(proposal.no, vote.PartyID)
		proposal.yes[vote.PartyID] = &vote
	} else {
		delete(proposal.yes, vote.PartyID)
		proposal.no[vote.PartyID] = &vote
	}
	e.vbuf.Add(vote)
	return nil
}

func (e *Engine) validateVote(vote types.Vote) (*governanceData, error) {
	voterTokens, err := getGovernanceTokens(e.accounts, vote.PartyID)
	if err != nil {
		return nil, err
	}
	totalTokens := e.accounts.GetTotalTokens()
	if float32(voterTokens) < float32(totalTokens)*e.networkParams.minVoterBalance {
		return nil, ErrVoterInsufficientTokens
	}

	proposal, found := e.activeProposals[vote.ProposalID]
	if !found {
		return nil, ErrProposalNotFound
	} else if proposal.State == types.Proposal_STATE_PASSED {
		return nil, ErrProposalPassed
	}
	return proposal, nil
}

// sets proposal in either declined or passed state
func (e *Engine) closeProposal(data *governanceData, counter *stakeCounter, totalStake uint64) {
	data.State = types.Proposal_STATE_DECLINED // declined unless passed

	yes := counter.countVotes(data.yes)
	no := counter.countVotes(data.no)
	totalVotes := float32(yes + no)

	// yes          > (yes + no)* required majority ratio
	if float32(yes) > totalVotes*e.networkParams.requiredMajority &&
		//(yes+no) >= (yes + no + novote)* required participation ratio
		totalVotes >= float32(totalStake)*e.networkParams.requiredParticipation {
		data.State = types.Proposal_STATE_PASSED
		e.log.Debug("Proposal passed", logging.String("proposal-id", data.ID))
	} else if totalVotes == 0 {
		e.log.Info("Proposal declined - no votes", logging.String("proposal-id", data.ID))
	} else {
		e.log.Info(
			"Proposal declined",
			logging.String("proposal-id", data.ID),
			logging.Uint64("yes-votes", yes),
			logging.Float32("min-yes-required", totalVotes*e.networkParams.requiredMajority),
			logging.Float32("total-votes", totalVotes),
			logging.Float32("min-total-votes-required", float32(totalStake)*e.networkParams.requiredParticipation),
		)
	}
	e.buf.Add(*data.Proposal)
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
	balance, err := getGovernanceTokens(s.accounts, partyID)
	if err != nil {
		s.log.Error(
			"Failed to get governance tokens balance for party",
			logging.String("party-id", partyID),
			logging.Error(err),
		)
		// not much we can do with the error as there is nowhere to buble up the error on tick
		return 0
	}
	s.balances[partyID] = balance
	return balance
}

func getGovernanceTokens(accounts Accounts, partyID string) (uint64, error) {
	account, err := accounts.GetPartyTokenAccount(partyID)
	if err != nil {
		return 0, err
	}
	return account.Balance, nil
}
