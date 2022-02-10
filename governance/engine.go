package governance

import (
	"context"
	"encoding/binary"
	"fmt"
	"sort"
	"time"

	proto "code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/assets"
	"code.vegaprotocol.io/vega/events"
	vgcrypto "code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/netparams"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
	"code.vegaprotocol.io/vega/validators"

	"github.com/pkg/errors"
)

var (
	ErrProposalNotFound        = errors.New("proposal not found")
	ErrProposalIsDuplicate     = errors.New("proposal with given ID already exists")
	ErrVoterInsufficientTokens = errors.New("vote requires more tokens than party has")
	ErrProposalPassed          = errors.New("proposal has passed and can no longer be voted on")
	ErrUnsupportedProposalType = errors.New("unsupported proposal type")
	ErrProposalDoesNotExists   = errors.New("proposal does not exists")
)

// Broker - event bus.
type Broker interface {
	Send(e events.Event)
	SendBatch(es []events.Event)
}

// StakingAccounts ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/staking_accounts_mock.go -package mocks code.vegaprotocol.io/vega/governance StakingAccounts
type StakingAccounts interface {
	GetAvailableBalance(party string) (*num.Uint, error)
	GetStakingAssetTotalSupply() *num.Uint
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/assets_mock.go -package mocks code.vegaprotocol.io/vega/governance Assets
type Assets interface {
	NewAsset(ref string, assetDetails *types.AssetDetails) (string, error)
	Get(assetID string) (*assets.Asset, error)
	IsEnabled(string) bool
}

// TimeService ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/time_service_mock.go -package mocks code.vegaprotocol.io/vega/governance TimeService
type TimeService interface {
	GetTimeNow() (time.Time, error)
}

// Witness ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/witness_mock.go -package mocks code.vegaprotocol.io/vega/governance Witness
type Witness interface {
	StartCheck(validators.Resource, func(interface{}, bool), time.Time) error
	RestoreResource(validators.Resource, func(interface{}, bool)) error
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/netparams_mock.go -package mocks code.vegaprotocol.io/vega/governance NetParams
type NetParams interface {
	Validate(string, string) error
	Update(context.Context, string, string) error
	GetDecimal(string) (num.Decimal, error)
	GetInt(string) (int64, error)
	GetUint(string) (*num.Uint, error)
	GetDuration(string) (time.Duration, error)
	GetJSONStruct(string, netparams.Reset) error
	Get(string) (string, error)
}

// Engine is the governance engine that handles proposal and vote lifecycle.
type Engine struct {
	Config
	log         *logging.Logger
	accs        StakingAccounts
	currentTime time.Time
	// we store proposals in slice
	// not as easy to access them directly, but by doing this we can keep
	// them in order of arrival, which makes their processing deterministic
	activeProposals        []*proposal
	enactedProposals       []*proposal
	nodeProposalValidation *NodeValidation
	broker                 Broker
	assets                 Assets
	netp                   NetParams

	// snapshot state
	gss             *governanceSnapshotState
	keyToSerialiser map[string]func() ([]byte, error)
}

func NewEngine(
	log *logging.Logger,
	cfg Config,
	accs StakingAccounts,
	broker Broker,
	assets Assets,
	witness Witness,
	netp NetParams,
	now time.Time,
) *Engine {
	log = log.Named(namedLogger)
	log.SetLevel(cfg.Level.Level)

	e := &Engine{
		Config:                 cfg,
		accs:                   accs,
		log:                    log,
		currentTime:            now,
		activeProposals:        []*proposal{},
		enactedProposals:       []*proposal{},
		nodeProposalValidation: NewNodeValidation(log, assets, now, witness),
		broker:                 broker,
		assets:                 assets,
		netp:                   netp,
		gss: &governanceSnapshotState{
			changed:    map[string]bool{activeKey: true, enactedKey: true, nodeValidationKey: true},
			hash:       map[string][]byte{},
			serialised: map[string][]byte{},
		},
		keyToSerialiser: map[string]func() ([]byte, error){},
	}

	e.keyToSerialiser[activeKey] = e.serialiseActiveProposals
	e.keyToSerialiser[enactedKey] = e.serialiseEnactedProposals
	e.keyToSerialiser[nodeValidationKey] = e.serialiseNodeProposals
	return e
}

func (e *Engine) Hash() []byte {
	// get the node proposal hash first
	npHash := e.nodeProposalValidation.Hash()

	// Create the slice for this state
	// 32 -> len(proposal.ID) = 32 bytes pubkey
	// vote counts = 3*uint64
	// 32 -> len of enactedProposal.ID
	// len of the np hash
	output := make(
		[]byte,
		(len(e.activeProposals)*(32+8*3) +
			len(e.enactedProposals)*32 +
			len(npHash)),
	)

	var i int
	for _, k := range e.activeProposals {
		idbytes := []byte(k.ID)
		copy(output[i:], idbytes[:])
		i += 32
		binary.BigEndian.PutUint64(output[i:], uint64(len(k.yes)))
		i += 8
		binary.BigEndian.PutUint64(output[i:], uint64(len(k.no)))
		i += 8
		binary.BigEndian.PutUint64(output[i:], uint64(len(k.invalidVotes)))
		i += 8
	}
	for _, k := range e.enactedProposals {
		idbytes := []byte(k.ID)
		copy(output[i:], idbytes[:])
		i += 32
	}
	// now add the hash of the nodeProposals
	copy(output[i:], npHash[:])

	return vgcrypto.Hash(output)
}

// ReloadConf updates the internal configuration of the governance engine.
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

func (e *Engine) preEnactProposal(p *proposal) (te *ToEnact, perr types.ProposalError, err error) {
	te = &ToEnact{
		p: p,
	}
	defer func() {
		if err != nil {
			p.State = proto.Proposal_STATE_FAILED
			p.Reason = perr
		}
	}()

	switch p.Terms.Change.GetTermType() {
	case types.ProposalTerms_NEW_MARKET:
		te.m = &ToEnactMarket{}
	case types.ProposalTerms_UPDATE_NETWORK_PARAMETER:
		unp := p.Terms.GetUpdateNetworkParameter()
		if unp != nil {
			te.n = unp.Changes
		}
	case types.ProposalTerms_NEW_ASSET:
		asset, err := e.assets.Get(p.ID)
		if err != nil {
			return nil, proto.ProposalError_PROPOSAL_ERROR_UNSPECIFIED, err
		}
		te.a = asset.Type()
	case types.ProposalTerms_NEW_FREEFORM:
		te.f = &ToEnactFreeform{}
	}
	return
}

func (e *Engine) preVoteClosedProposal(p *proposal) *VoteClosed {
	vc := &VoteClosed{
		p: p.Proposal,
	}
	switch p.Terms.Change.GetTermType() {
	case types.ProposalTerms_NEW_MARKET:
		startAuction := true
		if p.State != proto.Proposal_STATE_PASSED {
			startAuction = false
		} else {
			// this proposal needs to be included in the checkpoint but we don't need to copy
			// the proposal here, as it may reach the enacted state shortly
			e.enactedProposals = append(e.enactedProposals, p)

			e.gss.changed[enactedKey] = true
		}
		vc.m = &NewMarketVoteClosed{
			startAuction: startAuction,
		}
	}
	return vc
}

func (e *Engine) removeProposal(id string) {
	for i, p := range e.activeProposals {
		if p.ID == id {
			copy(e.activeProposals[i:], e.activeProposals[i+1:])
			e.activeProposals[len(e.activeProposals)-1] = nil
			e.activeProposals = e.activeProposals[:len(e.activeProposals)-1]

			e.gss.changed[activeKey] = true
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

		for _, proposal := range e.activeProposals {
			// only enter this if the proposal state is OPEN
			// or we would return many times the voteClosed eventually
			if proposal.State == proto.Proposal_STATE_OPEN && proposal.Terms.ClosingTimestamp < now {
				e.closeProposal(ctx, proposal)
				voteClosed = append(voteClosed, e.preVoteClosedProposal(proposal))
			}

			if proposal.State != proto.Proposal_STATE_OPEN && proposal.State != proto.Proposal_STATE_PASSED {
				toBeRemoved = append(toBeRemoved, proposal.ID)
			} else if proposal.State == proto.Proposal_STATE_PASSED &&
				(e.isAutoEnactableProposal(proposal.Proposal) || proposal.Terms.EnactmentTimestamp < now) {
				enact, _, err := e.preEnactProposal(proposal)
				if err != nil {
					e.broker.Send(events.NewProposalEvent(ctx, *proposal.Proposal))
					e.log.Error("proposal enactment has failed",
						logging.String("proposal-id", proposal.ID),
						logging.Error(err))
				} else {
					toBeRemoved = append(toBeRemoved, proposal.ID)
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
			logging.String("proposal-id", p.ID))
		p.State = proto.Proposal_STATE_OPEN
		e.broker.Send(events.NewProposalEvent(ctx, *p.Proposal))
		e.startValidatedProposal(p) // can't fail, and proposal has been validated at an ulterior time
	}
	for _, p := range rejected {
		e.log.Info("proposal has not been validated by nodes",
			logging.String("proposal-id", p.ID))
		p.State = proto.Proposal_STATE_REJECTED
		p.Reason = proto.ProposalError_PROPOSAL_ERROR_NODE_VALIDATION_FAILED
		e.broker.Send(events.NewProposalEvent(ctx, *p.Proposal))
	}

	if len(accepted) != 0 || len(rejected) != 0 {
		e.gss.changed[nodeValidationKey] = true
	}

	for _, ep := range toBeEnacted {
		// this is the new market proposal, and should already be in the slice
		prop := *ep.ProposalData()
		if prop.Terms.Change.GetTermType() == types.ProposalTerms_NEW_MARKET {
			// just in case the proposal wasn't added for whatever reason (shouldn't be possible)
			found := false
			for i, p := range e.enactedProposals {
				if p.ID == prop.ID {
					e.enactedProposals[i] = &prop // replace with pointer to copy
					found = true
					break
				}
			}
			// no need to append
			if found {
				continue
			}
		}
		// take a copy in the state just before the proposal was enacted
		e.enactedProposals = append(e.enactedProposals, &prop)
	}

	if len(toBeEnacted) != 0 {
		e.gss.changed[enactedKey] = true
	}

	// flush here for now
	return toBeEnacted, voteClosed
}

func (e *Engine) getProposal(id string) (*proposal, bool) {
	for _, v := range e.activeProposals {
		if v.ID == id {
			return v, true
		}
	}

	p, ok := e.nodeProposalValidation.getProposal(id)
	if !ok {
		return nil, false
	}

	return p.proposal, true
}

// SubmitProposal submits new proposal to the governance engine so it can be voted on, passed and enacted.
// Only open can be submitted and validated at this point. No further validation happens.
func (e *Engine) SubmitProposal(
	ctx context.Context,
	psub types.ProposalSubmission,
	id, party string,
) (ts *ToSubmit, err error) {
	if _, ok := e.getProposal(id); ok {
		return nil, ErrProposalIsDuplicate // state is not allowed to change externally
	}

	p := types.Proposal{
		ID:        id,
		Timestamp: e.currentTime.UnixNano(),
		Party:     party,
		State:     proto.Proposal_STATE_OPEN,
		Terms:     psub.Terms,
		Reference: psub.Reference,
	}

	defer func() {
		e.broker.Send(events.NewProposalEvent(ctx, p))
	}()
	perr, err := e.validateOpenProposal(p)
	if err != nil {
		p.State = proto.Proposal_STATE_REJECTED
		p.Reason = perr
		if e.log.GetLevel() == logging.DebugLevel {
			e.log.Debug("Proposal rejected", logging.String("proposal-id", p.ID),
				logging.String("proposal details", p.IntoProto().String()))
		}
		return nil, err
	}

	// now if it's a 2 steps proposal, start the node votes
	if e.isTwoStepsProposal(&p) {
		p.State = proto.Proposal_STATE_WAITING_FOR_NODE_VOTE
		err = e.startTwoStepsProposal(&p)
	} else {
		e.startProposal(&p)
	}
	if err != nil {
		return nil, err
	}
	return e.intoToSubmit(&p)
}

func (e *Engine) RejectProposal(
	ctx context.Context, p *types.Proposal, r types.ProposalError, errorDetails error,
) error {
	if _, ok := e.getProposal(p.ID); !ok {
		return ErrProposalDoesNotExists
	}

	e.rejectProposal(p, r, errorDetails)
	e.broker.Send(events.NewProposalEvent(ctx, *p))
	return nil
}

// FinaliseEnactment receives the enact proposal and updates the state in out enactedProposal
// list to have the current state of the proposals. This is entirely so that when we restore
// from a snapshot we can propagate the proposal with the latest state back into the apiservice.
func (e *Engine) FinaliseEnactment(ctx context.Context, prop *types.Proposal) {
	// find the proposal so we can update the state after enactment
	for _, enacted := range e.enactedProposals {
		if enacted.ID == prop.ID {
			enacted.State = prop.State
			break
		}
	}
	e.broker.Send(events.NewProposalEvent(ctx, *prop))
}

func (e *Engine) rejectProposal(p *types.Proposal, r types.ProposalError, errorDetails error) {
	e.removeProposal(p.ID)
	p.ErrorDetails = errorDetails.Error()
	p.Reason = r
	p.State = types.ProposalStateRejected
}

// toSubmit build the return response for the SubmitProposal
// method.
func (e *Engine) intoToSubmit(p *types.Proposal) (*ToSubmit, error) {
	tsb := &ToSubmit{p: p}

	switch p.Terms.Change.GetTermType() {
	case types.ProposalTerms_NEW_MARKET:
		// use to calculate the auction duration
		// which is basically enacttime - closetime
		// FIXME(): normally we should use the closetime
		// but this would not play well with the MarketAuctionState stuff
		// for now we start the auction as of now.
		closeTime := e.currentTime
		enactTime := time.Unix(p.Terms.EnactmentTimestamp, 0)
		newMarket := p.Terms.GetNewMarket()
		mkt, perr, err := createMarket(p.ID, newMarket, e.netp, e.currentTime, e.assets, enactTime.Sub(closeTime))
		if err != nil {
			e.rejectProposal(p, perr, err)
			return nil, fmt.Errorf("%w, %v", err, perr)
		}
		tsb.m = &ToSubmitNewMarket{
			m: mkt,
		}
		tsb.m.l = types.LiquidityProvisionSubmissionFromMarketCommitment(
			newMarket.LiquidityCommitment, p.ID)
	}

	return tsb, nil
}

func (e *Engine) startProposal(p *types.Proposal) {
	e.activeProposals = append(e.activeProposals, &proposal{
		Proposal:     p,
		yes:          map[string]*types.Vote{},
		no:           map[string]*types.Vote{},
		invalidVotes: map[string]*types.Vote{},
	})

	e.gss.changed[activeKey] = true
}

func (e *Engine) startValidatedProposal(p *proposal) {
	e.activeProposals = append(e.activeProposals, p)
	e.gss.changed[activeKey] = true
}

func (e *Engine) startTwoStepsProposal(p *types.Proposal) error {
	e.gss.changed[nodeValidationKey] = true
	return e.nodeProposalValidation.Start(p)
}

func (e *Engine) isTwoStepsProposal(p *types.Proposal) bool {
	return e.nodeProposalValidation.IsNodeValidationRequired(p)
}

// isAutoEnactableProposal returns true if the proposal is of a type that has no on-chain enactment
// and so can be automatically enacted without needing to care for the enactment timestamps.
func (e *Engine) isAutoEnactableProposal(p *types.Proposal) bool {
	switch p.Terms.Change.GetTermType() {
	case types.ProposalTerms_NEW_FREEFORM:
		return true
	}
	return false
}

func (e *Engine) getProposalParams(terms *types.ProposalTerms) (*ProposalParameters, error) {
	switch terms.Change.GetTermType() {
	case types.ProposalTerms_NEW_MARKET:
		return e.getNewMarketProposalParameters(), nil
	case types.ProposalTerms_NEW_ASSET:
		return e.getNewAssetProposalParameters(), nil
	case types.ProposalTerms_UPDATE_NETWORK_PARAMETER:
		return e.getUpdateNetworkParameterProposalParameters(), nil
	case types.ProposalTerms_NEW_FREEFORM:
		return e.getNewFreeformProposalarameters(), nil
	default:
		return nil, ErrUnsupportedProposalType
	}
}

// validates proposals read from the chain.
func (e *Engine) validateOpenProposal(proposal types.Proposal) (types.ProposalError, error) {
	params, err := e.getProposalParams(proposal.Terms)
	if err != nil {
		return types.ProposalError_PROPOSAL_ERROR_UNKNOWN_TYPE, err
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
	if !e.isAutoEnactableProposal(&proposal) && enactTime.Before(minEnactTime) {
		e.log.Debug("proposal enact time is too soon",
			logging.Time("expected-min", minEnactTime),
			logging.Time("provided", enactTime),
			logging.String("id", proposal.ID))
		return types.ProposalError_PROPOSAL_ERROR_ENACT_TIME_TOO_SOON,
			fmt.Errorf("proposal enactment time too soon, expected > %v, got %v", minEnactTime, enactTime)
	}

	maxEnactTime := e.currentTime.Add(params.MaxEnact)
	if !e.isAutoEnactableProposal(&proposal) && enactTime.After(maxEnactTime) {
		e.log.Debug("proposal enact time is too late",
			logging.Time("expected-max", maxEnactTime),
			logging.Time("provided", enactTime),
			logging.String("id", proposal.ID))
		return types.ProposalError_PROPOSAL_ERROR_ENACT_TIME_TOO_LATE,
			fmt.Errorf("proposal enactment time too late, expected < %v, got %v", maxEnactTime, enactTime)
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

	if !e.isAutoEnactableProposal(&proposal) && enactTime.Before(closeTime) {
		e.log.Debug("proposal enactment time can't be smaller than closing time",
			logging.Time("enactment-time", enactTime),
			logging.Time("closing-time", closeTime),
			logging.String("id", proposal.ID))
		return types.ProposalError_PROPOSAL_ERROR_INCOMPATIBLE_TIMESTAMPS,
			fmt.Errorf("proposal enactment time cannot be before closing time, expected > %v got %v", closeTime, enactTime)
	}

	proposerTokens, err := getGovernanceTokens(e.accs, proposal.Party)
	if err != nil {
		e.log.Debug("proposer have no governance token",
			logging.PartyID(proposal.Party),
			logging.ProposalID(proposal.ID))
		return types.ProposalError_PROPOSAL_ERROR_INSUFFICIENT_TOKENS, err
	}
	if proposerTokens.LT(params.MinProposerBalance) {
		e.log.Debug("proposer have insufficient governance token",
			logging.BigUint("expect-balance", params.MinProposerBalance),
			logging.String("proposer-balance", proposerTokens.String()),
			logging.PartyID(proposal.Party),
			logging.ProposalID(proposal.ID))
		return types.ProposalError_PROPOSAL_ERROR_INSUFFICIENT_TOKENS,
			fmt.Errorf("proposer have insufficient governance token, expected >= %v got %v", params.MinProposerBalance, proposerTokens)
	}
	return e.validateChange(proposal.Terms)
}

// validates proposed change.
func (e *Engine) validateChange(terms *types.ProposalTerms) (types.ProposalError, error) {
	switch terms.Change.GetTermType() {
	case types.ProposalTerms_NEW_MARKET:
		closeTime := time.Unix(terms.ClosingTimestamp, 0)
		enactTime := time.Unix(terms.EnactmentTimestamp, 0)

		perr, err := validateNewMarket(e.currentTime, terms.GetNewMarket(), e.assets, true, e.netp, enactTime.Sub(closeTime))
		if err != nil {
			return perr, err
		}
		return types.ProposalError_PROPOSAL_ERROR_UNSPECIFIED, nil
	case types.ProposalTerms_NEW_ASSET:
		return validateNewAsset(terms.GetNewAsset().Changes)
	case types.ProposalTerms_UPDATE_NETWORK_PARAMETER:
		return validateNetworkParameterUpdate(e.netp, terms.GetUpdateNetworkParameter().Changes)
	case types.ProposalTerms_NEW_FREEFORM:
		return validateNewFreeform(terms.GetNewFreeform())
	}
	return types.ProposalError_PROPOSAL_ERROR_UNSPECIFIED, nil
}

// AddVote adds vote onto an existing active proposal (if found) so the proposal could pass and be enacted.
func (e *Engine) AddVote(ctx context.Context, cmd types.VoteSubmission, party string) error {
	proposal, err := e.validateVote(cmd, party)
	if err != nil {
		e.log.Debug("invalid vote submission",
			logging.PartyID(party),
			logging.String("vote", cmd.String()),
			logging.Error(err),
		)
		return err
	}

	if e.log.GetLevel() <= logging.DebugLevel {
		e.log.Debug("vote submission accepted",
			logging.PartyID(party),
			logging.String("vote", cmd.String()),
		)
	}

	vote := types.Vote{
		PartyID:                     party,
		ProposalID:                  cmd.ProposalID,
		Value:                       cmd.Value,
		Timestamp:                   e.currentTime.UnixNano(),
		TotalGovernanceTokenBalance: num.Zero(),
		TotalGovernanceTokenWeight:  num.DecimalZero(),
	}

	// we only want to count the last vote, so add to yes/no map, delete from the other
	// if the party hasn't cast a vote yet, the delete is just a noop
	if vote.Value == types.VoteValueYes {
		delete(proposal.no, vote.PartyID)
		proposal.yes[vote.PartyID] = &vote
	} else {
		delete(proposal.yes, vote.PartyID)
		proposal.no[vote.PartyID] = &vote
	}
	e.broker.Send(events.NewVoteEvent(ctx, vote))
	return nil
}

func (e *Engine) validateVote(vote types.VoteSubmission, party string) (*proposal, error) {
	proposal, found := e.getProposal(vote.ProposalID)
	if !found {
		return nil, ErrProposalNotFound
	} else if proposal.State == types.ProposalStatePassed {
		return nil, ErrProposalPassed
	}

	params, err := e.getProposalParams(proposal.Terms)
	if err != nil {
		return nil, err
	}

	voterTokens, err := getGovernanceTokens(e.accs, party)
	if err != nil {
		return nil, err
	}
	if voterTokens.LT(params.MinVoterBalance) {
		return nil, ErrVoterInsufficientTokens
	}

	return proposal, nil
}

// closeProposal determines the state of the proposal, passed or declined. I
// also computes the vote balance and weight for the API.
func (e *Engine) closeProposal(ctx context.Context, proposal *proposal) {
	if !proposal.IsOpen() {
		return
	}

	params := e.mustGetProposalParams(proposal)

	finalState := proposal.Close(params, e.accs)

	if finalState == types.ProposalStatePassed {
		e.log.Debug("Proposal passed", logging.ProposalID(proposal.ID))
	} else if finalState == types.ProposalStateDeclined {
		e.log.Debug("Proposal declined", logging.ProposalID(proposal.ID))
	}

	e.broker.Send(events.NewProposalEvent(ctx, *proposal.Proposal))
	e.broker.SendBatch(newUpdatedProposalEvents(ctx, proposal))
}

func newUpdatedProposalEvents(ctx context.Context, proposal *proposal) []events.Event {
	votes := []*events.Vote{}

	for _, y := range proposal.yes {
		votes = append(votes, events.NewVoteEvent(ctx, *y))
	}
	for _, n := range proposal.no {
		votes = append(votes, events.NewVoteEvent(ctx, *n))
	}
	for _, n := range proposal.invalidVotes {
		votes = append(votes, events.NewVoteEvent(ctx, *n))
	}

	sort.Slice(votes, func(i, j int) bool {
		return votes[i].Proto().Timestamp < votes[j].Proto().Timestamp
	})

	evts := make([]events.Event, 0, len(votes))
	for _, e := range votes {
		evts = append(evts, e)
	}

	return evts
}

func (e *Engine) mustGetProposalParams(proposal *proposal) *ProposalParameters {
	params, err := e.getProposalParams(proposal.Terms)
	if err != nil {
		e.log.Panic("failed to get the proposal parameters from the terms",
			logging.Error(err),
		)
	}
	return params
}

type proposal struct {
	*types.Proposal
	yes          map[string]*types.Vote
	no           map[string]*types.Vote
	invalidVotes map[string]*types.Vote
}

func (p *proposal) IsOpen() bool {
	return p.State == types.ProposalStateOpen
}

func (p *proposal) Close(params *ProposalParameters, accounts StakingAccounts) types.ProposalState {
	if !p.IsOpen() {
		return p.State
	}

	totalStake := accounts.GetStakingAssetTotalSupply()

	yes := p.countVotes(p.yes, accounts)
	yesDec := num.DecimalFromUint(yes)
	no := p.countVotes(p.no, accounts)
	totalVotes := num.Sum(yes, no)
	totalVotesDec := num.DecimalFromUint(totalVotes)
	p.weightVotes(p.yes, totalVotesDec)
	p.weightVotes(p.no, totalVotesDec)
	majorityThreshold := totalVotesDec.Mul(params.RequiredMajority)
	totalStakeDec := num.DecimalFromUint(totalStake)
	participationThreshold := totalStakeDec.Mul(params.RequiredParticipation)

	if yesDec.GreaterThan(majorityThreshold) && totalVotesDec.GreaterThanOrEqual(participationThreshold) {
		p.State = proto.Proposal_STATE_PASSED
	} else {
		p.Reason = proto.ProposalError_PROPOSAL_ERROR_MAJORITY_THRESHOLD_NOT_REACHED
		if totalVotesDec.LessThan(participationThreshold) {
			p.Reason = proto.ProposalError_PROPOSAL_ERROR_PARTICIPATION_THRESHOLD_NOT_REACHED
		}
		p.State = proto.Proposal_STATE_DECLINED
	}

	return p.State
}

func (p *proposal) countVotes(votes map[string]*types.Vote, accounts StakingAccounts) *num.Uint {
	tally := num.Zero()
	for k, v := range votes {
		v.TotalGovernanceTokenBalance = getTokensBalance(accounts, v.PartyID)
		// the user may have withdrawn their governance token
		// before the end of the vote. We will then remove them from the map if it's the case.
		if v.TotalGovernanceTokenBalance.IsZero() {
			p.invalidVotes[k] = v
			delete(votes, k)
			continue
		}
		tally.AddSum(v.TotalGovernanceTokenBalance)
	}

	return tally
}

func (p *proposal) weightVotes(votes map[string]*types.Vote, totalVotes num.Decimal) {
	for _, v := range votes {
		balanceDec := num.DecimalFromUint(v.TotalGovernanceTokenBalance)
		v.TotalGovernanceTokenWeight = balanceDec.Div(totalVotes)
	}
}

func getTokensBalance(accounts StakingAccounts, partyID string) *num.Uint {
	balance, _ := getGovernanceTokens(accounts, partyID)
	return balance
}

func getGovernanceTokens(accounts StakingAccounts, party string) (*num.Uint, error) {
	balance, err := accounts.GetAvailableBalance(party)
	if err != nil {
		return nil, err
	}

	return balance, err
}

func (e *Engine) updateValidatorKey(ctx context.Context, m map[string]*types.Vote, oldKey, newKey string) {
	if vote, ok := m[oldKey]; ok {
		delete(m, oldKey)
		vote.PartyID = newKey
		e.broker.Send(events.NewVoteEvent(ctx, *vote))
		m[newKey] = vote
	}
}

func (e *Engine) ValidatorKeyChanged(ctx context.Context, oldKey, newKey string) {
	for _, p := range e.activeProposals {
		e.updateValidatorKey(ctx, p.yes, oldKey, newKey)
		e.updateValidatorKey(ctx, p.no, oldKey, newKey)
		e.updateValidatorKey(ctx, p.invalidVotes, oldKey, newKey)
	}
}
