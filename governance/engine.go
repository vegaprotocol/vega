package governance

import (
	"context"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/assets"
	"code.vegaprotocol.io/vega/blockchain"
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/nodewallet"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
)

var (
	ErrProposalNotFound                        = errors.New("proposal not found")
	ErrProposalIsDuplicate                     = errors.New("proposal with given ID already exists")
	ErrProposalCloseTimeInvalid                = errors.New("proposal closes too soon or too late")
	ErrProposalEnactTimeInvalid                = errors.New("proposal enactment times too soon or late")
	ErrVoterInsufficientTokens                 = errors.New("vote requires more tokens than party has")
	ErrVotePeriodExpired                       = errors.New("proposal voting has been closed")
	ErrAssetProposalReferenceDuplicate         = errors.New("duplicate asset proposal for reference")
	ErrProposalInvalidState                    = errors.New("proposal state not valid, only open can be submitted")
	ErrProposalCloseTimeTooSoon                = errors.New("proposal closes too soon")
	ErrProposalCloseTimeTooLate                = errors.New("proposal closes too late")
	ErrProposalEnactTimeTooSoon                = errors.New("proposal enactment time is too soon")
	ErrProposalEnactTimeTooLate                = errors.New("proposal enactment time is too late")
	ErrProposalInsufficientTokens              = errors.New("party requires more tokens to submit a proposal")
	ErrProposalMinPaticipationStakeTooLow      = errors.New("proposal minimum participation stake is too low")
	ErrProposalMinPaticipationStakeInvalid     = errors.New("proposal minimum participation stake is out of bounds [0..1]")
	ErrProposalMinRequiredMajorityStakeTooLow  = errors.New("proposal minimum required majority stake is too low")
	ErrProposalMinRequiredMajorityStakeInvalid = errors.New("proposal minimum required majority stake is out of bounds [0.5..1]")
	ErrProposalPassed                          = errors.New("proposal has passed and can no longer be voted on")
	ErrNoNetworkParams                         = errors.New("network parameters were not configured for this proposal type")
)

// Broker - event bus
//go:generate go run github.com/golang/mock/mockgen -destination mocks/broker_mock.go -package mocks code.vegaprotocol.io/vega/governance Broker
type Broker interface {
	Send(e events.Event)
}

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
	Flush()
}

// VoteBuf...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/vote_buffer_mock.go -package mocks code.vegaprotocol.io/vega/governance VoteBuf
type VoteBuf interface {
	Add(types.Vote)
	Flush()
}

// ValidatorTopology...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/validator_topology_mock.go -package mocks code.vegaprotocol.io/vega/governance ValidatorTopology
type ValidatorTopology interface {
	Exists([]byte) bool
	Len() int
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/wallet_mock.go -package mocks code.vegaprotocol.io/vega/governance Wallet
type Wallet interface {
	Get(chain nodewallet.Blockchain) (nodewallet.Wallet, bool)
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/commander_mock.go -package mocks code.vegaprotocol.io/vega/governance Commander
type Commander interface {
	Command(key nodewallet.Wallet, cmd blockchain.Command, payload proto.Message) error
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/assets_mock.go -package mocks code.vegaprotocol.io/vega/governance Assets
type Assets interface {
	NewAsset(ref string, assetSrc *types.AssetSource) (string, error)
	Get(assetID string) (assets.Asset, error)
}

// Engine is the governance engine that handles proposal and vote lifecycle.
type Engine struct {
	Config
	mu                     sync.Mutex
	log                    *logging.Logger
	accs                   Accounts
	buf                    Buffer
	vbuf                   VoteBuf
	currentTime            time.Time
	activeProposals        map[string]*proposalData
	networkParams          NetworkParameters
	isValidator            bool
	nodeProposalValidation *NodeValidation
	broker                 Broker
}

type proposalData struct {
	*types.Proposal
	yes map[string]*types.Vote
	no  map[string]*types.Vote
}

func NewEngine(log *logging.Logger, cfg Config, params *NetworkParameters, accs Accounts, buf Buffer, vbuf VoteBuf, broker Broker, top ValidatorTopology, wallet Wallet, cmd Commander, assets Assets, now time.Time, isValidator bool) (*Engine, error) {
	log = log.Named(namedLogger)
	// ensure params are set
	nodeValidation, err := NewNodeValidation(log, top, wallet, cmd, assets, now, isValidator)
	if err != nil {
		return nil, err
	}

	return &Engine{
		Config:                 cfg,
		accs:                   accs,
		buf:                    buf,
		vbuf:                   vbuf,
		log:                    log,
		currentTime:            now,
		activeProposals:        map[string]*proposalData{},
		networkParams:          *params,
		nodeProposalValidation: nodeValidation,
		broker:                 broker,
	}, nil
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
	ctx := context.TODO()
	e.currentTime = t
	var toBeEnacted []*types.Proposal
	if len(e.activeProposals) > 0 {
		now := t.Unix()

		totalStake := e.accs.GetTotalTokens()
		counter := newStakeCounter(e.log, e.accs)

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
	}

	// then get all proposal accepted through node validation, and start their vote time.
	accepted, rejected := e.nodeProposalValidation.OnChainTimeUpdate(t)
	for _, p := range accepted {
		e.log.Info("proposal has been validated by nodes, starting now",
			logging.String("proposal-id", p.ID))
		p.State = types.Proposal_STATE_OPEN
		e.broker.Send(events.NewProposalEvent(ctx, *p))
		e.buf.Add(*p)
		e.startProposal(p) // can't fail, and proposal has been validated at an ulterior time
	}
	for _, p := range rejected {
		e.log.Info("proposal has not been validated by nodes",
			logging.String("proposal-id", p.ID))
		p.State = types.Proposal_STATE_REJECTED
		e.broker.Send(events.NewProposalEvent(ctx, *p))
		e.buf.Add(*p)
	}

	// flush here for now
	e.buf.Flush()
	e.vbuf.Flush()
	return toBeEnacted
}

// SubmitProposal submits new proposal to the governance engine so it can be voted on, passed and enacted.
// Only open can be submitted and validated at this point. No further validation happens.
func (e *Engine) SubmitProposal(p types.Proposal) error {
	ctx := context.TODO()
	if _, exists := e.activeProposals[p.ID]; exists {
		return ErrProposalIsDuplicate // state is not allowed to change externally
	}
	if p.State == types.Proposal_STATE_OPEN {
		err := e.validateOpenProposal(p)
		if err != nil {
			p.State = types.Proposal_STATE_REJECTED
		} else {
			// now if it's a 2 steps proposal, start the node votes
			if e.isTwoStepsProposal(&p) {
				p.State = types.Proposal_STATE_WAITING_FOR_NODE_VOTE
				err = e.startTwoStepsProposal(&p)
			} else {
				e.startProposal(&p)
			}
		}
		e.broker.Send(events.NewProposalEvent(ctx, p))
		e.buf.Add(p)
		return err
	}
	return ErrProposalInvalidState
}

func (e *Engine) startProposal(p *types.Proposal) {
	e.activeProposals[p.ID] = &proposalData{
		Proposal: p,
		yes:      map[string]*types.Vote{},
		no:       map[string]*types.Vote{},
	}
}

func (e *Engine) startTwoStepsProposal(p *types.Proposal) error {
	return e.nodeProposalValidation.Start(p)
}

func (e *Engine) isTwoStepsProposal(p *types.Proposal) bool {
	return e.nodeProposalValidation.IsNodeValidationRequired(p)
}

func (e *Engine) getProposalParams(terms *types.ProposalTerms) (*ProposalParameters, error) {
	if terms.GetNewMarket() != nil {
		return &e.networkParams.newMarkets, nil
	}
	return nil, ErrNoNetworkParams
}

// validates proposals read from the chain
func (e *Engine) validateOpenProposal(proposal types.Proposal) error {
	params, err := e.getProposalParams(proposal.Terms)
	if err != nil {
		return err
	}
	if proposal.Terms.ClosingTimestamp < e.currentTime.Add(params.MinClose).Unix() {
		return ErrProposalCloseTimeTooSoon
	}
	if proposal.Terms.ClosingTimestamp > e.currentTime.Add(params.MaxClose).Unix() {
		return ErrProposalCloseTimeTooLate
	}
	if proposal.Terms.EnactmentTimestamp < e.currentTime.Add(params.MinEnact).Unix() {
		return ErrProposalEnactTimeTooSoon
	}
	if proposal.Terms.EnactmentTimestamp > e.currentTime.Add(params.MaxEnact).Unix() {
		return ErrProposalEnactTimeTooLate
	}
	proposerTokens, err := getGovernanceTokens(e.accs, proposal.PartyID)
	if err != nil {
		return err
	}
	totalTokens := e.accs.GetTotalTokens()
	if float32(proposerTokens) < float32(totalTokens)*params.MinProposerBalance {
		return ErrProposalInsufficientTokens
	}
	return nil
}

func (e *Engine) AddNodeVote(v *types.NodeVote) error {
	return e.nodeProposalValidation.AddNodeVote(v)
}

// AddVote adds vote onto an existing active proposal (if found) so the proposal could pass and be enacted
func (e *Engine) AddVote(vote types.Vote) error {
	ctx := context.TODO()
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
	e.broker.Send(events.NewVoteEvent(ctx, vote))
	e.vbuf.Add(vote)
	return nil
}

func (e *Engine) validateVote(vote types.Vote) (*proposalData, error) {
	proposal, found := e.activeProposals[vote.ProposalID]
	if !found {
		return nil, ErrProposalNotFound
	} else if proposal.State == types.Proposal_STATE_PASSED {
		return nil, ErrProposalPassed
	}

	params, err := e.getProposalParams(proposal.Terms)
	if err != nil {
		return nil, err
	}

	voterTokens, err := getGovernanceTokens(e.accs, vote.PartyID)
	if err != nil {
		return nil, err
	}
	totalTokens := e.accs.GetTotalTokens()
	if float32(voterTokens) < float32(totalTokens)*params.MinVoterBalance {
		return nil, ErrVoterInsufficientTokens
	}

	return proposal, nil
}

// sets proposal in either declined or passed state
func (e *Engine) closeProposal(proposal *proposalData, counter *stakeCounter, totalStake uint64) error {
	ctx := context.TODO()
	if proposal.State == types.Proposal_STATE_OPEN {
		proposal.State = types.Proposal_STATE_DECLINED // declined unless passed

		params, err := e.getProposalParams(proposal.Terms)
		if err != nil {
			return err
		}

		yes := counter.countVotes(proposal.yes)
		no := counter.countVotes(proposal.no)
		totalVotes := float32(yes + no)

		// yes          > (yes + no)* required majority ratio
		if float32(yes) > totalVotes*params.RequiredMajority &&
			//(yes+no) >= (yes + no + novote)* required participation ratio
			totalVotes >= float32(totalStake)*params.RequiredParticipation {
			proposal.State = types.Proposal_STATE_PASSED
			e.log.Debug("Proposal passed", logging.String("proposal-id", proposal.ID))
		} else if totalVotes == 0 {
			e.log.Info("Proposal declined - no votes", logging.String("proposal-id", proposal.ID))
		} else {
			e.log.Info(
				"Proposal declined",
				logging.String("proposal-id", proposal.ID),
				logging.Uint64("yes-votes", yes),
				logging.Float32("min-yes-required", totalVotes*params.RequiredMajority),
				logging.Float32("total-votes", totalVotes),
				logging.Float32("min-total-votes-required", float32(totalStake)*params.RequiredParticipation),
				logging.Float32("tokens", float32(totalStake)),
			)
		}
		e.broker.Send(events.NewProposalEvent(ctx, *proposal.Proposal))
		e.buf.Add(*proposal.Proposal)
	}
	return nil
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
