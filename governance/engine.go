package governance

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"sort"
	"time"

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
	ErrProposalDoesNotExist                      = errors.New("proposal does not exist")
	ErrMarketDoesNotExist                        = errors.New("market does not exist")
	ErrProposalNotOpenForVotes                   = errors.New("proposal is not open for votes")
	ErrProposalIsDuplicate                       = errors.New("proposal with given ID already exists")
	ErrVoterInsufficientTokensAndEquityLikeShare = errors.New("vote requires tokens or equity-like share")
	ErrVoterInsufficientTokens                   = errors.New("vote requires more tokens than the party has")
	ErrUnsupportedProposalType                   = errors.New("unsupported proposal type")
)

// Broker - event bus.
type Broker interface {
	Send(e events.Event)
	SendBatch(es []events.Event)
}

// Markets allows to get the market data for use in the market update proposal
// computation.
//go:generate go run github.com/golang/mock/mockgen -destination mocks/markets_mock.go -package mocks code.vegaprotocol.io/vega/governance Markets
type Markets interface {
	MarketExists(market string) bool
	GetEquityLikeShareForMarketAndParty(market, party string) (num.Decimal, bool)
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
	markets     Markets
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
	markets Markets,
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
		markets:                markets,
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
		len(e.activeProposals)*(32+8*3)+len(e.enactedProposals)*32+len(npHash),
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
	h := vgcrypto.Hash(output)
	e.log.Debug("governance state hash", logging.String("hash", hex.EncodeToString(h)))
	return h
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
			p.State = types.ProposalStateFailed
			p.Reason = perr
		}
	}()

	switch p.Terms.Change.GetTermType() {
	case types.ProposalTermsTypeNewMarket:
		te.m = &ToEnactMarket{}
	case types.ProposalTermsTypeUpdateNetworkParameter:
		unp := p.Terms.GetUpdateNetworkParameter()
		if unp != nil {
			te.n = unp.Changes
		}
	case types.ProposalTermsTypeNewAsset:
		asset, err := e.assets.Get(p.ID)
		if err != nil {
			return nil, types.ProposalErrorUnspecified, err
		}
		te.a = asset.Type()
	case types.ProposalTermsTypeNewFreeform:
		te.f = &ToEnactFreeform{}
	}
	return
}

func (e *Engine) preVoteClosedProposal(p *proposal) *VoteClosed {
	vc := &VoteClosed{
		p: p.Proposal,
	}
	switch p.Terms.Change.GetTermType() {
	case types.ProposalTermsTypeNewMarket:
		startAuction := true
		if p.State != types.ProposalStatePassed {
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

	now := t.Unix()

	for _, proposal := range e.activeProposals {
		if proposal.ShouldClose(now) {
			e.closeProposal(ctx, proposal)
			voteClosed = append(voteClosed, e.preVoteClosedProposal(proposal))
		}

		if !proposal.IsOpen() && !proposal.IsPassed() {
			toBeRemoved = append(toBeRemoved, proposal.ID)
		} else if proposal.IsPassed() && (e.isAutoEnactableProposal(proposal.Proposal) || proposal.IsTimeToEnact(now)) {
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

	// now we iterate over all proposal ids to remove them from the list
	for _, id := range toBeRemoved {
		e.removeProposal(id)
	}

	// then get all proposal accepted through node validation, and start their vote time.
	accepted, rejected := e.nodeProposalValidation.OnChainTimeUpdate(t)
	for _, p := range accepted {
		e.log.Info("proposal has been validated by nodes, starting now",
			logging.String("proposal-id", p.ID))
		p.State = types.ProposalStateOpen
		e.broker.Send(events.NewProposalEvent(ctx, *p.Proposal))
		e.startValidatedProposal(p) // can't fail, and proposal has been validated at an ulterior time
	}
	for _, p := range rejected {
		e.log.Info("proposal has not been validated by nodes",
			logging.String("proposal-id", p.ID))
		p.Reject(types.ProposalErrorNodeValidationFailed)
		e.broker.Send(events.NewProposalEvent(ctx, *p.Proposal))
	}

	if len(accepted) != 0 || len(rejected) != 0 {
		e.gss.changed[nodeValidationKey] = true
	}

	for _, ep := range toBeEnacted {
		// this is the new market proposal, and should already be in the slice
		prop := *ep.ProposalData()
		if prop.Terms.Change.GetTermType() == types.ProposalTermsTypeNewMarket {
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

	p := &types.Proposal{
		ID:        id,
		Timestamp: e.currentTime.UnixNano(),
		Party:     party,
		State:     types.ProposalStateOpen,
		Terms:     psub.Terms,
		Reference: psub.Reference,
	}

	defer func() {
		e.broker.Send(events.NewProposalEvent(ctx, *p))
	}()

	perr, err := e.validateOpenProposal(p)
	if err != nil {
		p.RejectWithErr(perr, err)
		if e.log.GetLevel() == logging.DebugLevel {
			e.log.Debug("Proposal rejected", logging.String("proposal-id", p.ID),
				logging.String("proposal details", p.IntoProto().String()))
		}
		return nil, err
	}

	// now if it's a 2 steps proposal, start the node votes
	if e.isTwoStepsProposal(p) {
		p.WaitForNodeVote()
		err = e.startTwoStepsProposal(p)
	} else {
		e.startProposal(p)
	}
	if err != nil {
		return nil, err
	}
	return e.intoToSubmit(p)
}

func (e *Engine) RejectProposal(
	ctx context.Context, p *types.Proposal, r types.ProposalError, errorDetails error,
) error {
	if _, ok := e.getProposal(p.ID); !ok {
		return ErrProposalDoesNotExist
	}

	e.rejectProposal(p, r, errorDetails)
	e.broker.Send(events.NewProposalEvent(ctx, *p))
	return nil
}

// FinaliseEnactment receives the enact proposal and updates the state in our enactedProposal
// list to have the current state of the proposals. This is entirely so that when we restore
// from a snapshot we can propagate the proposal with the latest state back into the API service.
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
	p.RejectWithErr(r, errorDetails)
}

// toSubmit build the return response for the SubmitProposal
// method.
func (e *Engine) intoToSubmit(p *types.Proposal) (*ToSubmit, error) {
	tsb := &ToSubmit{p: p}

	switch p.Terms.Change.GetTermType() {
	case types.ProposalTermsTypeNewMarket:
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
	case types.ProposalTermsTypeUpdateMarket:
		// TODO Implement
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
	case types.ProposalTermsTypeNewFreeform:
		return true
	}
	return false
}

func (e *Engine) getProposalParams(terms *types.ProposalTerms) (*ProposalParameters, error) {
	switch terms.Change.GetTermType() {
	case types.ProposalTermsTypeNewMarket:
		return e.getNewMarketProposalParameters(), nil
	case types.ProposalTermsTypeUpdateMarket:
		return e.getUpdateMarketProposalParameters(), nil
	case types.ProposalTermsTypeNewAsset:
		return e.getNewAssetProposalParameters(), nil
	case types.ProposalTermsTypeUpdateNetworkParameter:
		return e.getUpdateNetworkParameterProposalParameters(), nil
	case types.ProposalTermsTypeNewFreeform:
		return e.getNewFreeformProposalParameters(), nil
	default:
		return nil, ErrUnsupportedProposalType
	}
}

// validateOpenProposal reads from the chain.
func (e *Engine) validateOpenProposal(proposal *types.Proposal) (types.ProposalError, error) {
	params, err := e.getProposalParams(proposal.Terms)
	if err != nil {
		return types.ProposalErrorUnknownType, err
	}

	closeTime := time.Unix(proposal.Terms.ClosingTimestamp, 0)
	minCloseTime := e.currentTime.Add(params.MinClose)
	if closeTime.Before(minCloseTime) {
		e.log.Debug("proposal close time is too soon",
			logging.Time("expected-min", minCloseTime),
			logging.Time("provided", closeTime),
			logging.String("id", proposal.ID))
		return types.ProposalErrorCloseTimeTooSoon,
			fmt.Errorf("proposal closing time too soon, expected > %v, got %v", minCloseTime.UTC(), closeTime.UTC())
	}

	maxCloseTime := e.currentTime.Add(params.MaxClose)
	if closeTime.After(maxCloseTime) {
		e.log.Debug("proposal close time is too late",
			logging.Time("expected-max", maxCloseTime),
			logging.Time("provided", closeTime),
			logging.String("id", proposal.ID))
		return types.ProposalErrorCloseTimeTooLate,
			fmt.Errorf("proposal closing time too late, expected < %v, got %v", maxCloseTime.UTC(), closeTime.UTC())
	}

	enactTime := time.Unix(proposal.Terms.EnactmentTimestamp, 0)
	minEnactTime := e.currentTime.Add(params.MinEnact)
	if !e.isAutoEnactableProposal(proposal) && enactTime.Before(minEnactTime) {
		e.log.Debug("proposal enact time is too soon",
			logging.Time("expected-min", minEnactTime),
			logging.Time("provided", enactTime),
			logging.String("id", proposal.ID))
		return types.ProposalErrorEnactTimeTooSoon,
			fmt.Errorf("proposal enactment time too soon, expected > %v, got %v", minEnactTime.UTC(), enactTime.UTC())
	}

	maxEnactTime := e.currentTime.Add(params.MaxEnact)
	if !e.isAutoEnactableProposal(proposal) && enactTime.After(maxEnactTime) {
		e.log.Debug("proposal enact time is too late",
			logging.Time("expected-max", maxEnactTime),
			logging.Time("provided", enactTime),
			logging.String("id", proposal.ID))
		return types.ProposalErrorEnactTimeTooLate,
			fmt.Errorf("proposal enactment time too late, expected < %v, got %v", maxEnactTime.UTC(), enactTime.UTC())
	}

	if e.isTwoStepsProposal(proposal) {
		validationTime := time.Unix(proposal.Terms.ValidationTimestamp, 0)
		if closeTime.Before(validationTime) {
			e.log.Debug("proposal closing time can't be smaller or equal than validation time",
				logging.Time("closing-time", closeTime),
				logging.Time("validation-time", validationTime),
				logging.String("id", proposal.ID))
			return types.ProposalErrorIncompatibleTimestamps,
				fmt.Errorf("proposal closing time cannot be before validation time, expected > %v got %v", validationTime.UTC(), closeTime.UTC())
		}
	}

	if !e.isAutoEnactableProposal(proposal) && enactTime.Before(closeTime) {
		e.log.Debug("proposal enactment time can't be smaller than closing time",
			logging.Time("enactment-time", enactTime),
			logging.Time("closing-time", closeTime),
			logging.String("id", proposal.ID))
		return types.ProposalErrorIncompatibleTimestamps,
			fmt.Errorf("proposal enactment time cannot be before closing time, expected > %v got %v", closeTime.UTC(), enactTime.UTC())
	}

	proposerTokens, err := getGovernanceTokens(e.accs, proposal.Party)
	if err != nil {
		e.log.Debug("proposer have no governance token",
			logging.PartyID(proposal.Party),
			logging.ProposalID(proposal.ID))
		return types.ProposalErrorInsufficientTokens, err
	}
	if proposerTokens.LT(params.MinProposerBalance) {
		e.log.Debug("proposer have insufficient governance token",
			logging.BigUint("expect-balance", params.MinProposerBalance),
			logging.String("proposer-balance", proposerTokens.String()),
			logging.PartyID(proposal.Party),
			logging.ProposalID(proposal.ID))
		return types.ProposalErrorInsufficientTokens,
			fmt.Errorf("proposer have insufficient governance token, expected >= %v got %v", params.MinProposerBalance, proposerTokens)
	}

	if proposal.IsMarketUpdate() {
		proposalError, err := e.validateMarketUpdate(proposal, params)
		if err != nil {
			return proposalError, err
		}
	}

	return e.validateChange(proposal.Terms)
}

func (e *Engine) ValidatorKeyChanged(ctx context.Context, oldKey, newKey string) {
	for _, p := range e.activeProposals {
		e.updateValidatorKey(ctx, p.yes, oldKey, newKey)
		e.updateValidatorKey(ctx, p.no, oldKey, newKey)
		e.updateValidatorKey(ctx, p.invalidVotes, oldKey, newKey)
	}
}

// AddVote adds a vote onto an existing active proposal.
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

	vote := types.Vote{
		PartyID:                     party,
		ProposalID:                  cmd.ProposalID,
		Value:                       cmd.Value,
		Timestamp:                   e.currentTime.UnixNano(),
		TotalGovernanceTokenBalance: num.Zero(),
		TotalGovernanceTokenWeight:  num.DecimalZero(),
		TotalEquityLikeShareWeight:  num.DecimalZero(),
	}

	if err := proposal.AddVote(vote); err != nil {
		return fmt.Errorf("couldn't cast the vote: %w", err)
	}

	if e.log.IsDebug() {
		e.log.Debug("vote submission accepted",
			logging.PartyID(party),
			logging.String("vote", cmd.String()),
		)
	}

	e.broker.Send(events.NewVoteEvent(ctx, vote))
	return nil
}

func (e *Engine) validateVote(vote types.VoteSubmission, party string) (*proposal, error) {
	proposal, found := e.getProposal(vote.ProposalID)
	if !found {
		return nil, ErrProposalDoesNotExist
	} else if !proposal.IsOpenForVotes() {
		return nil, ErrProposalNotOpenForVotes
	}

	params, err := e.getProposalParams(proposal.Terms)
	if err != nil {
		return nil, err
	}

	voterTokens, err := getGovernanceTokens(e.accs, party)
	if err != nil {
		return nil, err
	}

	if proposal.IsMarketUpdate() {
		partyELS, _ := e.markets.GetEquityLikeShareForMarketAndParty(proposal.ID, party)
		if partyELS.IsZero() && voterTokens.IsZero() {
			return nil, ErrVoterInsufficientTokensAndEquityLikeShare
		}
		// If he is not voting using his equity-like share, he should at least
		// have enough tokens.
		if partyELS.IsZero() && voterTokens.LT(params.MinVoterBalance) {
			return nil, ErrVoterInsufficientTokens
		}
	} else {
		if voterTokens.LT(params.MinVoterBalance) {
			return nil, ErrVoterInsufficientTokens
		}
	}

	return proposal, nil
}

func (e *Engine) validateMarketUpdate(proposal *types.Proposal, params *ProposalParameters) (types.ProposalError, error) {
	if !e.markets.MarketExists(proposal.ID) {
		e.log.Debug("market does not exist",
			logging.MarketID(proposal.ID),
			logging.PartyID(proposal.Party),
			logging.ProposalID(proposal.ID))
		return types.ProposalErrorInvalidMarket, ErrMarketDoesNotExist
	}
	partyELS, _ := e.markets.GetEquityLikeShareForMarketAndParty(proposal.ID, proposal.Party)
	if partyELS.LessThan(params.MinEquityLikeShare) {
		e.log.Debug("proposer have insufficient equity-like share",
			logging.String("expect-balance", params.MinEquityLikeShare.String()),
			logging.String("proposer-balance", partyELS.String()),
			logging.PartyID(proposal.Party),
			logging.MarketID(proposal.ID),
			logging.ProposalID(proposal.ID))
		return types.ProposalErrorInsufficientEquityLikeShare,
			fmt.Errorf("proposer have insufficient equity-like share, expected >= %v got %v", params.MinEquityLikeShare, partyELS)
	}

	return types.ProposalErrorUnspecified, nil
}

// validates proposed change.
func (e *Engine) validateChange(terms *types.ProposalTerms) (types.ProposalError, error) {
	switch terms.Change.GetTermType() {
	case types.ProposalTermsTypeNewMarket:
		closeTime := time.Unix(terms.ClosingTimestamp, 0)
		enactTime := time.Unix(terms.EnactmentTimestamp, 0)
		perr, err := validateNewMarket(e.currentTime, terms.GetNewMarket(), e.assets, true, e.netp, enactTime.Sub(closeTime))
		if err != nil {
			return perr, err
		}
		return types.ProposalErrorUnspecified, nil
	case types.ProposalTermsTypeNewAsset:
		return validateNewAsset(terms.GetNewAsset().Changes)
	case types.ProposalTermsTypeUpdateNetworkParameter:
		return validateNetworkParameterUpdate(e.netp, terms.GetUpdateNetworkParameter().Changes)
	case types.ProposalTermsTypeNewFreeform:
		return validateNewFreeform(terms.GetNewFreeform())
	}
	return types.ProposalErrorUnspecified, nil
}

func (e *Engine) closeProposal(ctx context.Context, proposal *proposal) {
	if !proposal.IsOpen() {
		return
	}

	params := e.mustGetProposalParams(proposal)

	proposal.Close(params, e.accs, e.markets)

	if proposal.IsPassed() {
		e.log.Debug("Proposal passed", logging.ProposalID(proposal.ID))
	} else if proposal.IsDeclined() {
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

func (e *Engine) updateValidatorKey(ctx context.Context, m map[string]*types.Vote, oldKey, newKey string) {
	if vote, ok := m[oldKey]; ok {
		delete(m, oldKey)
		vote.PartyID = newKey
		e.broker.Send(events.NewVoteEvent(ctx, *vote))
		m[newKey] = vote
	}
}

type proposal struct {
	*types.Proposal
	yes          map[string]*types.Vote
	no           map[string]*types.Vote
	invalidVotes map[string]*types.Vote
}

func (p *proposal) IsTimeToEnact(now int64) bool {
	return p.Proposal.Terms.EnactmentTimestamp < now
}

// ShouldClose tells if the proposal should be closed or not.
// We also check the "open" state, alongside the closing timestamp as solely
// relying on the closing timestamp could lead to call Close() on an
// already-closed proposal.
func (p *proposal) ShouldClose(now int64) bool {
	return p.IsOpen() && p.Proposal.Terms.ClosingTimestamp < now
}

func (p *proposal) IsOpen() bool {
	return p.State == types.ProposalStateOpen
}

func (p *proposal) IsPassed() bool {
	return p.State == types.ProposalStatePassed
}

func (p *proposal) IsDeclined() bool {
	return p.State == types.ProposalStateDeclined
}

func (p *proposal) IsOpenForVotes() bool {
	// It's allowed to vote during the validation of the proposal by the node.
	return p.State == types.ProposalStateOpen || p.State == types.ProposalStateWaitingForNodeVote
}

// AddVote registers the last vote casted by a party. The proposal has to be
// open, it returns an error otherwise.
func (p *proposal) AddVote(vote types.Vote) error {
	if !p.IsOpenForVotes() {
		return ErrProposalNotOpenForVotes
	}

	if vote.Value == types.VoteValueYes {
		delete(p.no, vote.PartyID)
		p.yes[vote.PartyID] = &vote
	} else {
		delete(p.yes, vote.PartyID)
		p.no[vote.PartyID] = &vote
	}

	return nil
}

// Close determines the state of the proposal, passed or declined based on the
// vote balance and weight.
// Warning: this method should only be called once. Use ShouldClose() to know
// when to call.
func (p *proposal) Close(params *ProposalParameters, accounts StakingAccounts, markets Markets) {
	if !p.IsOpen() {
		return
	}

	defer func() {
		p.purgeBlankVotes(p.yes)
		p.purgeBlankVotes(p.no)
	}()

	tokenVoteState, tokenVoteError := p.computeVoteStateUsingTokens(params, accounts)

	p.State = tokenVoteState
	p.Reason = tokenVoteError

	// Proposals, other than market updates, solely relies on votes using the
	// governance tokens. So, only proposals for market update can go beyond this
	// guard.
	if !p.IsMarketUpdate() {
		return
	}

	if tokenVoteState == types.ProposalStateDeclined && tokenVoteError == types.ProposalErrorParticipationThresholdNotReached {
		elsVoteState, elsVoteError := p.computeVoteStateUsingEquityLikeShare(params, markets)
		p.State = elsVoteState
		p.Reason = elsVoteError
	}
}

func (p *proposal) computeVoteStateUsingTokens(params *ProposalParameters, accounts StakingAccounts) (types.ProposalState, types.ProposalError) {
	totalStake := accounts.GetStakingAssetTotalSupply()

	yes := p.countTokens(p.yes, accounts)
	yesDec := num.DecimalFromUint(yes)
	no := p.countTokens(p.no, accounts)
	totalTokens := num.Sum(yes, no)
	totalTokensDec := num.DecimalFromUint(totalTokens)
	p.weightVotesFromToken(p.yes, totalTokensDec)
	p.weightVotesFromToken(p.no, totalTokensDec)
	majorityThreshold := totalTokensDec.Mul(params.RequiredMajority)
	totalStakeDec := num.DecimalFromUint(totalStake)
	participationThreshold := totalStakeDec.Mul(params.RequiredParticipation)

	if yesDec.GreaterThanOrEqual(majorityThreshold) && totalTokensDec.GreaterThanOrEqual(participationThreshold) {
		return types.ProposalStatePassed, types.ProposalErrorUnspecified
	}

	if totalTokensDec.LessThan(participationThreshold) {
		return types.ProposalStateDeclined, types.ProposalErrorParticipationThresholdNotReached
	}

	return types.ProposalStateDeclined, types.ProposalErrorMajorityThresholdNotReached
}

func (p *proposal) computeVoteStateUsingEquityLikeShare(params *ProposalParameters, markets Markets) (types.ProposalState, types.ProposalError) {
	yes := p.countEquityLikeShare(p.yes, markets)
	no := p.countEquityLikeShare(p.no, markets)
	totalEquityLikeShare := yes.Add(no)

	if yes.GreaterThanOrEqual(params.RequiredMajorityLP) && totalEquityLikeShare.GreaterThanOrEqual(params.RequiredParticipationLP) {
		return types.ProposalStatePassed, types.ProposalErrorUnspecified
	}

	if totalEquityLikeShare.LessThan(params.RequiredParticipationLP) {
		return types.ProposalStateDeclined, types.ProposalErrorParticipationThresholdNotReached
	}

	return types.ProposalStateDeclined, types.ProposalErrorMajorityThresholdNotReached
}

func (p *proposal) countTokens(votes map[string]*types.Vote, accounts StakingAccounts) *num.Uint {
	tally := num.Zero()
	for _, v := range votes {
		v.TotalGovernanceTokenBalance = getTokensBalance(accounts, v.PartyID)
		tally.AddSum(v.TotalGovernanceTokenBalance)
	}

	return tally
}

func (p *proposal) countEquityLikeShare(votes map[string]*types.Vote, markets Markets) num.Decimal {
	tally := num.DecimalZero()
	for _, v := range votes {
		v.TotalEquityLikeShareWeight, _ = markets.GetEquityLikeShareForMarketAndParty(p.ID, v.PartyID)
		tally = tally.Add(v.TotalEquityLikeShareWeight)
	}

	return tally
}

func (p *proposal) weightVotesFromToken(votes map[string]*types.Vote, totalVotes num.Decimal) {
	if totalVotes.IsZero() {
		return
	}

	for _, v := range votes {
		tokenBalanceDec := num.DecimalFromUint(v.TotalGovernanceTokenBalance)
		v.TotalGovernanceTokenWeight = tokenBalanceDec.Div(totalVotes)
	}
}

// purgeBlankVotes removes votes that don't have tokens or equity-like share
// associated. The user may have withdrawn their governance token or their
// equity-like share before the end of the vote.
// We will then purge them from the map if it's the case.
func (p *proposal) purgeBlankVotes(votes map[string]*types.Vote) {
	for k, v := range votes {
		if v.TotalGovernanceTokenBalance.IsZero() && v.TotalEquityLikeShareWeight.IsZero() {
			p.invalidVotes[k] = v
			delete(votes, k)
			continue
		}
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
