package governance

import (
	"context"
	"fmt"
	"time"

	"code.vegaprotocol.io/vega/assets"
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/netparams"
	types "code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/validators"

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
	ErrIncompatibleTimestamps                  = errors.New("incompatible timestamps")
	ErrUnsupportedProposalType                 = errors.New("unsupported proposal type")
	ErrProposalOpeningAuctionDurationTooShort  = errors.New("proposal opening auction duration is too short")
	ErrProposalOpeningAuctionDurationTooLong   = errors.New("proposal opening auction duration is too long")
	ErrMissingCommandIDFromContext             = errors.New("could not find command id from the context")
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

//go:generate go run github.com/golang/mock/mockgen -destination mocks/assets_mock.go -package mocks code.vegaprotocol.io/vega/governance Assets
type Assets interface {
	NewAsset(ref string, assetSrc *types.AssetSource) (string, error)
	Get(assetID string) (*assets.Asset, error)
	IsEnabled(string) bool
}

// TimeService ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/time_service_mock.go -package mocks code.vegaprotocol.io/vega/governance TimeService
type TimeService interface {
	GetTimeNow() (time.Time, error)
}

// ExtResChecker ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/ext_res_checker_mock.go -package mocks code.vegaprotocol.io/vega/governance ExtResChecker
type ExtResChecker interface {
	StartCheck(validators.Resource, func(interface{}, bool), time.Time) error
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/netparams_mock.go -package mocks code.vegaprotocol.io/vega/governance NetParams
type NetParams interface {
	Validate(string, string) error
	Update(context.Context, string, string) error
	GetFloat(string) (float64, error)
	GetInt(string) (int64, error)
	GetDuration(string) (time.Duration, error)
	GetJSONStruct(string, netparams.Reset) error
	Get(string) (string, error)
}

// Engine is the governance engine that handles proposal and vote lifecycle.
type Engine struct {
	Config
	log                    *logging.Logger
	accs                   Accounts
	currentTime            time.Time
	activeProposals        map[string]*proposalData
	nodeProposalValidation *NodeValidation
	broker                 Broker
	assets                 Assets
	netp                   NetParams
}

type proposalData struct {
	*types.Proposal
	yes map[string]*types.Vote
	no  map[string]*types.Vote
}

func NewEngine(
	log *logging.Logger,
	cfg Config,
	accs Accounts,
	broker Broker,
	assets Assets,
	erc ExtResChecker,
	netp NetParams,
	now time.Time,
) (*Engine, error) {
	log = log.Named(namedLogger)
	log.SetLevel(cfg.Level.Level)
	// ensure params are set
	nodeValidation, err := NewNodeValidation(log, assets, now, erc)
	if err != nil {
		return nil, err
	}

	return &Engine{
		Config:                 cfg,
		accs:                   accs,
		log:                    log,
		currentTime:            now,
		activeProposals:        map[string]*proposalData{},
		nodeProposalValidation: nodeValidation,
		broker:                 broker,
		assets:                 assets,
		netp:                   netp,
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

	e.Config = cfg
}

func (e *Engine) preEnactProposal(p *types.Proposal) (te *ToEnact, perr types.ProposalError, err error) {
	te = &ToEnact{
		p: p,
	}
	defer func() {
		if err != nil {
			p.State = types.Proposal_STATE_FAILED
			p.Reason = perr
		}
	}()
	switch change := p.Terms.Change.(type) {
	case *types.ProposalTerms_NewMarket:
		mkt, perr, err := createMarket(p.ID, change.NewMarket.Changes, e.netp, e.currentTime, e.assets)
		if err != nil {
			return nil, perr, err
		}
		te.m = mkt
	case *types.ProposalTerms_NewAsset:
		asset, err := e.assets.Get(p.GetID())
		if err != nil {
			return nil, types.ProposalError_PROPOSAL_ERROR_UNSPECIFIED, err
		}
		te.a = asset.ProtoAsset()
	}
	return
}

// OnChainTimeUpdate triggers time bound state changes.
func (e *Engine) OnChainTimeUpdate(ctx context.Context, t time.Time) []*ToEnact {
	e.currentTime = t
	var toBeEnacted []*ToEnact
	if len(e.activeProposals) > 0 {
		now := t.Unix()

		totalStake := e.accs.GetTotalTokens()
		counter := newStakeCounter(e.log, e.accs)

		for id, proposal := range e.activeProposals {
			if proposal.Terms.ClosingTimestamp < now {
				e.closeProposal(ctx, proposal, counter, totalStake)
			}

			if proposal.State != types.Proposal_STATE_OPEN && proposal.State != types.Proposal_STATE_PASSED {
				delete(e.activeProposals, id)
			} else if proposal.State == types.Proposal_STATE_PASSED && proposal.Terms.EnactmentTimestamp < now {
				enact, _, err := e.preEnactProposal(proposal.Proposal)
				if err != nil {
					e.broker.Send(events.NewProposalEvent(ctx, *proposal.Proposal))
					e.log.Error("proposal enactment has failed",
						logging.String("proposal-id", proposal.ID),
						logging.Error(err))
				} else {
					toBeEnacted = append(toBeEnacted, enact)
				}
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
		e.startProposal(p) // can't fail, and proposal has been validated at an ulterior time
	}
	for _, p := range rejected {
		e.log.Info("proposal has not been validated by nodes",
			logging.String("proposal-id", p.ID))
		p.State = types.Proposal_STATE_REJECTED
		p.Reason = types.ProposalError_PROPOSAL_ERROR_NODE_VALIDATION_FAILED
		e.broker.Send(events.NewProposalEvent(ctx, *p))
	}

	// flush here for now
	return toBeEnacted
}

// SubmitProposal submits new proposal to the governance engine so it can be voted on, passed and enacted.
// Only open can be submitted and validated at this point. No further validation happens.
func (e *Engine) SubmitProposal(ctx context.Context, p types.Proposal, id string) error {
	p.ID = id
	p.Timestamp = e.currentTime.UnixNano()

	if _, exists := e.activeProposals[p.ID]; exists {
		return ErrProposalIsDuplicate // state is not allowed to change externally
	}
	if p.State == types.Proposal_STATE_OPEN {
		perr, err := e.validateOpenProposal(p)
		if err != nil {
			p.State = types.Proposal_STATE_REJECTED
			p.Reason = perr
			if e.log.GetLevel() == logging.DebugLevel {
				e.log.Debug("Proposal rejected", logging.String("proposal-id", p.ID))
			}
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
	switch terms.Change.(type) {
	case *types.ProposalTerms_NewMarket:
		return e.getNewMarketProposalParameters()
	case *types.ProposalTerms_NewAsset:
		return e.getNewAssetProposalParameters()
	case *types.ProposalTerms_UpdateNetworkParameter:
		return e.getUpdateNetworkParameterProposalParameters()
	default:
		return nil, ErrUnsupportedProposalType
	}
}

// validates proposals read from the chain
func (e *Engine) validateOpenProposal(proposal types.Proposal) (types.ProposalError, error) {
	params, err := e.getProposalParams(proposal.Terms)
	if err != nil {
		// FIXME(): not checking the error here
		// we return unspecified here because not getting proposal
		// params is not possible, the check done before needs to be removed
		return types.ProposalError_PROPOSAL_ERROR_UNSPECIFIED, err
	}

	closeTime := time.Unix(proposal.Terms.ClosingTimestamp, 0)
	minCloseTime := e.currentTime.Add(params.MinClose)
	if closeTime.Before(minCloseTime) {
		e.log.Debug("proposal close time is too soon",
			logging.Time("expected-min", minCloseTime),
			logging.Time("provided", closeTime),
			logging.String("id", proposal.ID))
		return types.ProposalError_PROPOSAL_ERROR_CLOSE_TIME_TOO_SOON,
			fmt.Errorf("proposal closing time too soon, expected > %v, got %v", minCloseTime, closeTime)
	}

	maxCloseTime := e.currentTime.Add(params.MaxClose)
	if closeTime.After(maxCloseTime) {
		e.log.Debug("proposal close time is too late",
			logging.Time("expected-max", maxCloseTime),
			logging.Time("provided", closeTime),
			logging.String("id", proposal.ID))
		return types.ProposalError_PROPOSAL_ERROR_CLOSE_TIME_TOO_LATE,
			fmt.Errorf("proposal closing time too late, expected < %v, got %v", maxCloseTime, closeTime)
	}

	enactTime := time.Unix(proposal.Terms.EnactmentTimestamp, 0)
	minEnactTime := e.currentTime.Add(params.MinEnact)
	if enactTime.Before(minEnactTime) {
		e.log.Debug("proposal enact time is too soon",
			logging.Time("expected-min", minEnactTime),
			logging.Time("provided", enactTime),
			logging.String("id", proposal.ID))
		return types.ProposalError_PROPOSAL_ERROR_ENACT_TIME_TOO_SOON,
			fmt.Errorf("proposal enactment time too soon, expected > %v, got %v", minEnactTime, enactTime)
	}

	maxEnactTime := e.currentTime.Add(params.MaxEnact)
	if enactTime.After(maxEnactTime) {
		e.log.Debug("proposal enact time is too late",
			logging.Time("expected-max", maxEnactTime),
			logging.Time("provided", enactTime),
			logging.String("id", proposal.ID))
		return types.ProposalError_PROPOSAL_ERROR_ENACT_TIME_TOO_LATE,
			fmt.Errorf("proposal enactment time too lat, expected < %v, got %v", maxEnactTime, enactTime)
	}

	if e.isTwoStepsProposal(&proposal) {
		validationTime := time.Unix(proposal.Terms.ValidationTimestamp, 0)
		if closeTime.Before(validationTime) {
			e.log.Debug("proposal closing time can't be smaller or equal than validation time",
				logging.Time("closing-time", closeTime),
				logging.Time("validation-time", validationTime),
				logging.String("id", proposal.ID))
			return types.ProposalError_PROPOSAL_ERROR_INCOMPATIBLE_TIMESTAMPS,
				fmt.Errorf("proposal closing time cannot be before validation time, expected > %v got %v", validationTime, closeTime)
		}
	}

	if enactTime.Before(closeTime) {
		e.log.Debug("proposal enactment time can't be smaller than closing time",
			logging.Time("enactment-time", enactTime),
			logging.Time("closing-time", closeTime),
			logging.String("id", proposal.ID))
		return types.ProposalError_PROPOSAL_ERROR_INCOMPATIBLE_TIMESTAMPS,
			fmt.Errorf("proposal enactment time cannot be before closing time, expected > %v got %v", closeTime, enactTime)
	}

	proposerTokens, err := getGovernanceTokens(e.accs, proposal.PartyID)
	if err != nil {
		e.log.Debug("proposer have no governance token",
			logging.String("party-id", proposal.PartyID),
			logging.String("id", proposal.ID))
		return types.ProposalError_PROPOSAL_ERROR_INSUFFICIENT_TOKENS, err
	}
	if proposerTokens < params.MinProposerBalance {
		e.log.Debug("proposer have insufficient governance token",
			logging.Uint64("expect-balance", params.MinProposerBalance),
			logging.Uint64("proposer-balance", proposerTokens),
			logging.String("party-id", proposal.PartyID),
			logging.String("id", proposal.ID))
		return types.ProposalError_PROPOSAL_ERROR_INSUFFICIENT_TOKENS,
			fmt.Errorf("proposer have insufficient governance token, expected >= %v got %v", params.MinProposerBalance, proposerTokens)
	}
	return e.validateChange(proposal.Terms)
}

// validates proposed change
func (e *Engine) validateChange(terms *types.ProposalTerms) (types.ProposalError, error) {
	switch change := terms.Change.(type) {
	case *types.ProposalTerms_NewMarket:
		return validateNewMarket(e.currentTime, change.NewMarket.Changes, e.assets, true, e.netp)
	case *types.ProposalTerms_NewAsset:
		return validateNewAsset(change.NewAsset.Changes)
	case *types.ProposalTerms_UpdateNetworkParameter:
		return validateNetworkParameterUpdate(e.netp, change.UpdateNetworkParameter.Changes)
	}
	return types.ProposalError_PROPOSAL_ERROR_UNSPECIFIED, nil
}

// AddVote adds vote onto an existing active proposal (if found) so the proposal could pass and be enacted
func (e *Engine) AddVote(ctx context.Context, vote types.Vote) error {
	proposal, err := e.validateVote(vote)
	if err != nil {
		// vote was not created/accepted, send TxErrEvent
		e.broker.Send(events.NewTxErrEvent(ctx, err, vote.PartyID, vote))
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
	if float64(voterTokens) < float64(totalTokens)*params.MinVoterBalance {
		return nil, ErrVoterInsufficientTokens
	}

	return proposal, nil
}

// sets proposal in either declined or passed state
func (e *Engine) closeProposal(ctx context.Context, proposal *proposalData, counter *stakeCounter, totalStake uint64) error {
	if proposal.State == types.Proposal_STATE_OPEN {
		proposal.State = types.Proposal_STATE_DECLINED // declined unless passed

		params, err := e.getProposalParams(proposal.Terms)
		if err != nil {
			return err
		}

		yes := counter.countVotes(proposal.yes)
		no := counter.countVotes(proposal.no)
		totalVotes := float64(yes + no)

		// yes          > (yes + no)* required majority ratio
		if float64(yes) > totalVotes*params.RequiredMajority &&
			//(yes+no) >= (yes + no + novote)* required participation ratio
			totalVotes >= float64(totalStake)*params.RequiredParticipation {
			proposal.State = types.Proposal_STATE_PASSED
			e.log.Debug("Proposal passed", logging.String("proposal-id", proposal.ID))
		} else if totalVotes == 0 {
			e.log.Info("Proposal declined - no votes", logging.String("proposal-id", proposal.ID))
		} else {
			e.log.Info(
				"Proposal declined",
				logging.String("proposal-id", proposal.ID),
				logging.Uint64("yes-votes", yes),
				logging.Float64("min-yes-required", totalVotes*params.RequiredMajority),
				logging.Float64("total-votes", totalVotes),
				logging.Float64("min-total-votes-required", float64(totalStake)*params.RequiredParticipation),
				logging.Float32("tokens", float32(totalStake)),
			)
		}
		e.broker.Send(events.NewProposalEvent(ctx, *proposal.Proposal))
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
