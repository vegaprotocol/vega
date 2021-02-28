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
	ErrProposalNotFound        = errors.New("proposal not found")
	ErrProposalIsDuplicate     = errors.New("proposal with given ID already exists")
	ErrVoterInsufficientTokens = errors.New("vote requires more tokens than party has")
	ErrProposalInvalidState    = errors.New("proposal state not valid, only open can be submitted")
	ErrProposalPassed          = errors.New("proposal has passed and can no longer be voted on")
	ErrUnsupportedProposalType = errors.New("unsupported proposal type")
	ErrProposalDoesNotExists   = errors.New("proposal does not exists")
)

// Broker - event bus
//go:generate go run github.com/golang/mock/mockgen -destination mocks/broker_mock.go -package mocks code.vegaprotocol.io/vega/governance Broker
type Broker interface {
	Send(e events.Event)
}

// Accounts ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/accounts_mock.go -package mocks code.vegaprotocol.io/vega/governance Accounts
type Accounts interface {
	GetPartyGeneralAccount(party, asset string) (*types.Account, error)
	GetAssetTotalSupply(asset string) (uint64, error)
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
	log         *logging.Logger
	accs        Accounts
	currentTime time.Time
	// we store proposals in slice
	// not as easy to access them directly, but by doing this we can keep
	// them in order of arrival, which makes their processing deterministic
	activeProposals        []*proposalData
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
		activeProposals:        []*proposalData{},
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
		te.m = &ToEnactMarket{}
	case *types.ProposalTerms_UpdateNetworkParameter:
		te.n = change.UpdateNetworkParameter.Changes
	case *types.ProposalTerms_NewAsset:
		asset, err := e.assets.Get(p.GetId())
		if err != nil {
			return nil, types.ProposalError_PROPOSAL_ERROR_UNSPECIFIED, err
		}
		te.a = asset.ProtoAsset()
	}
	return
}

func (e *Engine) preVoteClosedProposal(p *types.Proposal) *VoteClosed {
	vc := &VoteClosed{
		p: p,
	}
	switch p.Terms.Change.(type) {
	case *types.ProposalTerms_NewMarket:
		startAuction := true
		if p.State != types.Proposal_STATE_PASSED {
			startAuction = false
		}
		vc.m = &NewMarketVoteClosed{
			startAuction: startAuction,
		}
	}
	return vc
}

func (e *Engine) removeProposal(id string) {
	for i, p := range e.activeProposals {
		if p.Id == id {
			copy(e.activeProposals[i:], e.activeProposals[i+1:])
			e.activeProposals[len(e.activeProposals)-1] = nil
			e.activeProposals = e.activeProposals[:len(e.activeProposals)-1]
			return
		}
	}
}

// OnChainTimeUpdate triggers time bound state changes.
func (e *Engine) OnChainTimeUpdate(ctx context.Context, t time.Time) ([]*ToEnact, []*VoteClosed) {
	e.currentTime = t
	var (
		toBeEnacted []*ToEnact
		voteClosed  []*VoteClosed
		toBeRemoved []string // ids
	)

	if len(e.activeProposals) > 0 {
		now := t.Unix()

		// we can't reach a point where the vote asset would not
		// so we can panic here if it were to happen
		voteAsset, err := e.netp.Get(netparams.GovernanceVoteAsset)
		if err != nil {
			e.log.Panic("error trying to get the vote asset from network parameters",
				logging.Error(err))
		}

		totalStake, err := e.accs.GetAssetTotalSupply(voteAsset)
		if err != nil {
			// an error here mean the asset has not been enabled,
			// which might happen eventually, we can just continue
			// return and do nothinbg for now I suppose
			e.log.Debug("governance asset is not enabled yet",
				logging.Error(err))
			return nil, nil
		}
		counter := newStakeCounter(e.log, e.accs, voteAsset)

		for _, proposal := range e.activeProposals {
			// only enter this if the proposal state is OPEN
			// or we would return many times the voteClosed eventually
			if proposal.State == types.Proposal_STATE_OPEN && proposal.Terms.ClosingTimestamp < now {
				e.closeProposal(ctx, proposal, counter, totalStake)
				voteClosed = append(voteClosed, e.preVoteClosedProposal(proposal.Proposal))
			}

			if proposal.State != types.Proposal_STATE_OPEN && proposal.State != types.Proposal_STATE_PASSED {
				toBeRemoved = append(toBeRemoved, proposal.Id)
			} else if proposal.State == types.Proposal_STATE_PASSED && proposal.Terms.EnactmentTimestamp < now {
				enact, _, err := e.preEnactProposal(proposal.Proposal)
				if err != nil {
					e.broker.Send(events.NewProposalEvent(ctx, *proposal.Proposal))
					e.log.Error("proposal enactment has failed",
						logging.String("proposal-id", proposal.Id),
						logging.Error(err))
				} else {
					toBeRemoved = append(toBeRemoved, proposal.Id)
					toBeEnacted = append(toBeEnacted, enact)
				}
			}
		}
	}

	// now we iterate over all proposal ids to remove them from the list
	for _, id := range toBeRemoved {
		e.removeProposal(id)
	}

	// then get all proposal accepted through node validation, and start their vote time.
	accepted, rejected := e.nodeProposalValidation.OnChainTimeUpdate(t)
	for _, p := range accepted {
		e.log.Info("proposal has been validated by nodes, starting now",
			logging.String("proposal-id", p.Id))
		p.State = types.Proposal_STATE_OPEN
		e.broker.Send(events.NewProposalEvent(ctx, *p))
		e.startProposal(p) // can't fail, and proposal has been validated at an ulterior time
	}
	for _, p := range rejected {
		e.log.Info("proposal has not been validated by nodes",
			logging.String("proposal-id", p.Id))
		p.State = types.Proposal_STATE_REJECTED
		p.Reason = types.ProposalError_PROPOSAL_ERROR_NODE_VALIDATION_FAILED
		e.broker.Send(events.NewProposalEvent(ctx, *p))
	}

	// flush here for now
	return toBeEnacted, voteClosed
}

func (e *Engine) getProposal(id string) (*proposalData, bool) {
	for _, v := range e.activeProposals {
		if v.Id == id {
			return v, true
		}
	}
	return nil, false
}

// SubmitProposal submits new proposal to the governance engine so it can be voted on, passed and enacted.
// Only open can be submitted and validated at this point. No further validation happens.
func (e *Engine) SubmitProposal(ctx context.Context, p types.Proposal, id string) (*ToSubmit, error) {
	p.Id = id
	p.Timestamp = e.currentTime.UnixNano()

	if _, ok := e.getProposal(p.Id); ok {
		return nil, ErrProposalIsDuplicate // state is not allowed to change externally
	}
	if p.State == types.Proposal_STATE_OPEN {

		defer func() {
			e.broker.Send(events.NewProposalEvent(ctx, p))
		}()

		perr, err := e.validateOpenProposal(p)
		if err != nil {
			p.State = types.Proposal_STATE_REJECTED
			p.Reason = perr
			if e.log.GetLevel() == logging.DebugLevel {
				e.log.Debug("Proposal rejected", logging.String("proposal-id", p.Id))
			}
			return nil, err
		}

		// now if it's a 2 steps proposal, start the node votes
		if e.isTwoStepsProposal(&p) {
			p.State = types.Proposal_STATE_WAITING_FOR_NODE_VOTE
			err = e.startTwoStepsProposal(&p)
		} else {
			e.startProposal(&p)
		}
		if err != nil {
			return nil, err
		}
		return e.intoToSubmit(&p)
	}
	return nil, ErrProposalInvalidState
}

func (e *Engine) RejectProposal(
	ctx context.Context, p *types.Proposal, r types.ProposalError,
) error {
	if _, ok := e.getProposal(p.Id); !ok {
		return ErrProposalDoesNotExists
	}

	e.rejectProposal(p, r)
	e.broker.Send(events.NewProposalEvent(ctx, *p))
	return nil
}

func (e *Engine) rejectProposal(p *types.Proposal, r types.ProposalError) {
	e.removeProposal(p.Id)
	p.Reason = r
	p.State = types.Proposal_STATE_REJECTED
}

// toSubmit build the return response for the SubmitProposal
// method
func (e *Engine) intoToSubmit(p *types.Proposal) (*ToSubmit, error) {
	tsb := &ToSubmit{p: p}

	switch change := p.Terms.Change.(type) {
	case *types.ProposalTerms_NewMarket:
		// use to calcualte the auction duration
		// which is basically enactime - closetime
		// FIXME(): normaly we should use the closetime
		// but this would not play well with the MarketAcutionState stuff
		// for now we start the auction as of now.
		closeTime := e.currentTime
		enactTime := time.Unix(p.Terms.EnactmentTimestamp, 0)

		mkt, perr, err := createMarket(p.Id, change.NewMarket.Changes, e.netp, e.currentTime, e.assets, enactTime.Sub(closeTime))
		if err != nil {
			e.rejectProposal(p, perr)
			return nil, fmt.Errorf("%w, %v", err, perr)
		}
		tsb.m = &ToSubmitNewMarket{
			m: mkt,
		}
		if change.NewMarket.LiquidityCommitment != nil {
			tsb.m.l = change.NewMarket.LiquidityCommitment.IntoSubmission(p.Id)
		}
	}

	return tsb, nil
}

func (e *Engine) startProposal(p *types.Proposal) {
	e.activeProposals = append(e.activeProposals, &proposalData{
		Proposal: p,
		yes:      map[string]*types.Vote{},
		no:       map[string]*types.Vote{},
	})
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
			logging.String("id", proposal.Id))
		return types.ProposalError_PROPOSAL_ERROR_CLOSE_TIME_TOO_SOON,
			fmt.Errorf("proposal closing time too soon, expected > %v, got %v", minCloseTime, closeTime)
	}

	maxCloseTime := e.currentTime.Add(params.MaxClose)
	if closeTime.After(maxCloseTime) {
		e.log.Debug("proposal close time is too late",
			logging.Time("expected-max", maxCloseTime),
			logging.Time("provided", closeTime),
			logging.String("id", proposal.Id))
		return types.ProposalError_PROPOSAL_ERROR_CLOSE_TIME_TOO_LATE,
			fmt.Errorf("proposal closing time too late, expected < %v, got %v", maxCloseTime, closeTime)
	}

	enactTime := time.Unix(proposal.Terms.EnactmentTimestamp, 0)
	minEnactTime := e.currentTime.Add(params.MinEnact)
	if enactTime.Before(minEnactTime) {
		e.log.Debug("proposal enact time is too soon",
			logging.Time("expected-min", minEnactTime),
			logging.Time("provided", enactTime),
			logging.String("id", proposal.Id))
		return types.ProposalError_PROPOSAL_ERROR_ENACT_TIME_TOO_SOON,
			fmt.Errorf("proposal enactment time too soon, expected > %v, got %v", minEnactTime, enactTime)
	}

	maxEnactTime := e.currentTime.Add(params.MaxEnact)
	if enactTime.After(maxEnactTime) {
		e.log.Debug("proposal enact time is too late",
			logging.Time("expected-max", maxEnactTime),
			logging.Time("provided", enactTime),
			logging.String("id", proposal.Id))
		return types.ProposalError_PROPOSAL_ERROR_ENACT_TIME_TOO_LATE,
			fmt.Errorf("proposal enactment time too lat, expected < %v, got %v", maxEnactTime, enactTime)
	}

	if e.isTwoStepsProposal(&proposal) {
		validationTime := time.Unix(proposal.Terms.ValidationTimestamp, 0)
		if closeTime.Before(validationTime) {
			e.log.Debug("proposal closing time can't be smaller or equal than validation time",
				logging.Time("closing-time", closeTime),
				logging.Time("validation-time", validationTime),
				logging.String("id", proposal.Id))
			return types.ProposalError_PROPOSAL_ERROR_INCOMPATIBLE_TIMESTAMPS,
				fmt.Errorf("proposal closing time cannot be before validation time, expected > %v got %v", validationTime, closeTime)
		}
	}

	if enactTime.Before(closeTime) {
		e.log.Debug("proposal enactment time can't be smaller than closing time",
			logging.Time("enactment-time", enactTime),
			logging.Time("closing-time", closeTime),
			logging.String("id", proposal.Id))
		return types.ProposalError_PROPOSAL_ERROR_INCOMPATIBLE_TIMESTAMPS,
			fmt.Errorf("proposal enactment time cannot be before closing time, expected > %v got %v", closeTime, enactTime)
	}

	// we can't reach a point where the vote asset would not
	// so we can panic here if it were to happen
	voteAsset, err := e.netp.Get(netparams.GovernanceVoteAsset)
	if err != nil {
		e.log.Panic("error trying to get the vote asset from network parameters",
			logging.Error(err))
	}

	proposerTokens, err := getGovernanceTokens(e.accs, proposal.PartyId, voteAsset)
	if err != nil {
		e.log.Debug("proposer have no governance token",
			logging.String("party-id", proposal.PartyId),
			logging.String("id", proposal.Id))
		return types.ProposalError_PROPOSAL_ERROR_INSUFFICIENT_TOKENS, err
	}
	if proposerTokens < params.MinProposerBalance {
		e.log.Debug("proposer have insufficient governance token",
			logging.Uint64("expect-balance", params.MinProposerBalance),
			logging.Uint64("proposer-balance", proposerTokens),
			logging.String("party-id", proposal.PartyId),
			logging.String("id", proposal.Id))
		return types.ProposalError_PROPOSAL_ERROR_INSUFFICIENT_TOKENS,
			fmt.Errorf("proposer have insufficient governance token, expected >= %v got %v", params.MinProposerBalance, proposerTokens)
	}
	return e.validateChange(proposal.Terms)
}

// validates proposed change
func (e *Engine) validateChange(terms *types.ProposalTerms) (types.ProposalError, error) {
	switch change := terms.Change.(type) {
	case *types.ProposalTerms_NewMarket:
		closeTime := time.Unix(terms.ClosingTimestamp, 0)
		enactTime := time.Unix(terms.EnactmentTimestamp, 0)

		perr, err := validateNewMarket(e.currentTime, change.NewMarket.Changes, e.assets, true, e.netp, enactTime.Sub(closeTime))
		if err != nil {
			return perr, err
		}
		// TODO: uncomment this once the client support submitting Commitment
		// if change.NewMarket.LiquidityCommitment == nil {
		// 	return types.ProposalError_PROPOSAL_ERROR_MARKET_MISSING_LIQUIDITY_COMMITMENT, ErrMarketMissingLiquidityCommitment
		// }
		// we do not validate the content of the commitment here,
		// the market will do later on
		return types.ProposalError_PROPOSAL_ERROR_UNSPECIFIED, nil
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
		e.broker.Send(events.NewTxErrEvent(ctx, err, vote.PartyId, vote))
		return err
	}
	// we only want to count the last vote, so add to yes/no map, delete from the other
	// if the party hasn't cast a vote yet, the delete is just a noop
	if vote.Value == types.Vote_VALUE_YES {
		delete(proposal.no, vote.PartyId)
		proposal.yes[vote.PartyId] = &vote
	} else {
		delete(proposal.yes, vote.PartyId)
		proposal.no[vote.PartyId] = &vote
	}
	e.broker.Send(events.NewVoteEvent(ctx, vote))
	return nil
}

func (e *Engine) validateVote(vote types.Vote) (*proposalData, error) {
	proposal, found := e.getProposal(vote.ProposalId)
	if !found {
		return nil, ErrProposalNotFound
	} else if proposal.State == types.Proposal_STATE_PASSED {
		return nil, ErrProposalPassed
	}

	params, err := e.getProposalParams(proposal.Terms)
	if err != nil {
		return nil, err
	}

	// we can't reach a point where the vote asset would not
	// so we can panic here if it were to happen
	voteAsset, err := e.netp.Get(netparams.GovernanceVoteAsset)
	if err != nil {
		e.log.Panic("error trying to get the vote asset from network parameters",
			logging.Error(err))
	}

	voterTokens, err := getGovernanceTokens(e.accs, vote.PartyId, voteAsset)
	if err != nil {
		return nil, err
	}
	if voterTokens < params.MinVoterBalance {
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
			e.log.Debug("Proposal passed", logging.String("proposal-id", proposal.Id))
		} else if totalVotes == 0 {
			e.log.Info("Proposal declined - no votes", logging.String("proposal-id", proposal.Id))
		} else {
			e.log.Info(
				"Proposal declined",
				logging.String("proposal-id", proposal.Id),
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
	log       *logging.Logger
	accounts  Accounts
	voteAsset string
	balances  map[string]uint64
}

func newStakeCounter(log *logging.Logger, accounts Accounts, voteAsset string) *stakeCounter {
	return &stakeCounter{
		log:       log,
		accounts:  accounts,
		balances:  map[string]uint64{},
		voteAsset: voteAsset,
	}
}
func (s *stakeCounter) countVotes(votes map[string]*types.Vote) uint64 {
	var tally uint64
	for _, v := range votes {
		tally += s.getTokens(v.PartyId)
	}
	return tally
}

func (s *stakeCounter) getTokens(partyID string) uint64 {
	if balance, found := s.balances[partyID]; found {
		return balance
	}
	balance, err := getGovernanceTokens(s.accounts, partyID, s.voteAsset)
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

func getGovernanceTokens(accounts Accounts, party, voteAsset string) (uint64, error) {
	account, err := accounts.GetPartyGeneralAccount(party, voteAsset)
	if err != nil {
		return 0, err
	}
	return account.Balance, nil
}
