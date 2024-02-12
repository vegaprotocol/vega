// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package governance

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"

	"code.vegaprotocol.io/vega/core/assets"
	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/netparams"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/core/validators"
	vgcrypto "code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"

	"github.com/pkg/errors"
)

var (
	ErrProposalDoesNotExist                      = errors.New("proposal does not exist")
	ErrMarketDoesNotExist                        = errors.New("market does not exist")
	ErrMarketStateUpdateNotAllowed               = errors.New("market state does not allow for state update")
	ErrMarketProposalStillOpen                   = errors.New("original market proposal is still open")
	ErrProposalNotOpenForVotes                   = errors.New("proposal is not open for votes")
	ErrProposalIsDuplicate                       = errors.New("proposal with given ID already exists")
	ErrVoterInsufficientTokensAndEquityLikeShare = errors.New("vote requires tokens or equity-like share")
	ErrVoterInsufficientTokens                   = errors.New("vote requires more tokens than the party has")
	ErrUnsupportedProposalType                   = errors.New("unsupported proposal type")
	ErrUnsupportedAssetSourceType                = errors.New("unsupported asset source type")
	ErrExpectedERC20Asset                        = errors.New("expected an ERC20 asset but was not")
	ErrErc20AddressAlreadyInUse                  = errors.New("erc20 address already in use")
	ErrParentMarketDoesNotExist                  = errors.New("market to succeed does not exist")
	ErrParentMarketAlreadySucceeded              = errors.New("the parent market was already succeeded by a prior proposal")
	ErrParentMarketSucceededByCompeting          = errors.New("the parent market has been succeeded by a competing propsal")
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/mocks.go -package mocks code.vegaprotocol.io/vega/core/governance Markets,StakingAccounts,Assets,TimeService,Witness,NetParams,Banking

// Broker - event bus.
type Broker interface {
	Send(e events.Event)
	SendBatch(es []events.Event)
}

// Markets allows to get the market data for use in the market update proposal
// computation.
type Markets interface {
	MarketExists(market string) bool
	GetMarket(market string, settled bool) (types.Market, bool)
	GetMarketState(market string) (types.MarketState, error)
	GetEquityLikeShareForMarketAndParty(market, party string) (num.Decimal, bool)
	RestoreMarket(ctx context.Context, marketConfig *types.Market) error
	StartOpeningAuction(ctx context.Context, marketID string) error
	UpdateMarket(ctx context.Context, marketConfig *types.Market) error
	IsSucceeded(mktID string) bool
}

// StakingAccounts ...
type StakingAccounts interface {
	GetAvailableBalance(party string) (*num.Uint, error)
	GetStakingAssetTotalSupply() *num.Uint
}

type Assets interface {
	NewAsset(ctx context.Context, ref string, assetDetails *types.AssetDetails) (string, error)
	Get(assetID string) (*assets.Asset, error)
	IsEnabled(string) bool
	SetRejected(ctx context.Context, assetID string) error
	SetPendingListing(ctx context.Context, assetID string) error
	ValidateAsset(assetID string) error
	ExistsForEthereumAddress(address string) bool
}

type Banking interface {
	VerifyGovernanceTransfer(transfer *types.NewTransferConfiguration) error
	VerifyCancelGovernanceTransfer(transferID string) error
}

// TimeService ...
type TimeService interface {
	GetTimeNow() time.Time
}

// Witness ...
type Witness interface {
	StartCheck(validators.Resource, func(interface{}, bool), time.Time) error
	RestoreResource(validators.Resource, func(interface{}, bool)) error
}

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
	log *logging.Logger

	nodeProposalValidation *NodeValidation
	accs                   StakingAccounts
	markets                Markets
	timeService            TimeService
	broker                 Broker
	assets                 Assets
	netp                   NetParams
	banking                Banking

	// we store proposals in slice
	// not as easy to access them directly, but by doing this we can keep
	// them in order of arrival, which makes their processing deterministic
	activeBatchProposals map[string]*batchProposal
	activeProposals      []*proposal
	enactedProposals     []*proposal

	// snapshot state
	gss *governanceSnapshotState
	// main chain ID
	chainID uint64
}

func NewEngine(
	log *logging.Logger,
	cfg Config,
	accs StakingAccounts,
	tm TimeService,
	broker Broker,
	assets Assets,
	witness Witness,
	markets Markets,
	netp NetParams,
	banking Banking,
) *Engine {
	log = log.Named(namedLogger)
	log.SetLevel(cfg.Level.Level)

	e := &Engine{
		Config:                 cfg,
		accs:                   accs,
		log:                    log,
		activeProposals:        []*proposal{},
		enactedProposals:       []*proposal{},
		activeBatchProposals:   map[string]*batchProposal{},
		nodeProposalValidation: NewNodeValidation(log, assets, tm.GetTimeNow(), witness),
		timeService:            tm,
		broker:                 broker,
		assets:                 assets,
		markets:                markets,
		netp:                   netp,
		gss:                    &governanceSnapshotState{},
		banking:                banking,
	}
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

func (e *Engine) preEnactProposal(ctx context.Context, p *proposal) (te *ToEnact, perr types.ProposalError, err error) {
	te = &ToEnact{
		p: p,
	}
	defer func() {
		if err != nil {
			p.FailWithErr(perr, err)
		}
	}()

	switch p.Terms.Change.GetTermType() {
	case types.ProposalTermsTypeNewMarket:
		te.m = &ToEnactNewMarket{}
	case types.ProposalTermsTypeNewSpotMarket:
		te.s = &ToEnactNewSpotMarket{}
	case types.ProposalTermsTypeUpdateMarket:
		mkt, perr, err := e.updatedMarketFromProposal(p)
		if err != nil {
			return nil, perr, err
		}
		te.updatedMarket = mkt
	case types.ProposalTermsTypeUpdateSpotMarket:
		mkt, perr, err := e.updatedSpotMarketFromProposal(p)
		if err != nil {
			return nil, perr, err
		}
		te.updatedSpotMarket = mkt
	case types.ProposalTermsTypeUpdateNetworkParameter:
		unp := p.Terms.GetUpdateNetworkParameter()
		if unp != nil {
			te.n = unp.Changes
		}
		if err := e.netp.Validate(unp.Changes.Key, unp.Changes.Value); err != nil {
			return nil, types.ProposalErrorNetworkParameterInvalidValue, err
		}
	case types.ProposalTermsTypeNewAsset:
		asset, err := e.assets.Get(p.ID)
		if err != nil {
			return nil, types.ProposalErrorUnspecified, err
		}
		te.newAsset = asset.Type()
		// notify the asset engine that the proposal was passed
		// and the asset is not pending for listing on the bridge
		e.assets.SetPendingListing(ctx, p.ID)
	case types.ProposalTermsTypeUpdateAsset:
		asset, perr, err := e.updatedAssetFromProposal(p)
		if err != nil {
			return nil, perr, err
		}
		te.updatedAsset = asset
	case types.ProposalTermsTypeNewFreeform:
		te.f = &ToEnactFreeform{}
	case types.ProposalTermsTypeNewTransfer:
		te.t = &ToEnactTransfer{}
	case types.ProposalTermsTypeCancelTransfer:
		te.c = &ToEnactCancelTransfer{}
	case types.ProposalTermsTypeUpdateMarketState:
		te.msu = &ToEnactMarketStateUpdate{}
	case types.ProposalTermsTypeUpdateReferralProgram:
		te.referralProgramChanges = updatedReferralProgramFromProposal(p)
	case types.ProposalTermsTypeUpdateVolumeDiscountProgram:
		te.volumeDiscountProgram = updatedVolumeDiscountProgramFromProposal(p)
	}
	return //nolint:nakedret
}

func (e *Engine) preVoteClosedProposal(p *proposal) *VoteClosed {
	vc := &VoteClosed{
		p: p.Proposal,
	}
	switch p.Terms.Change.GetTermType() {
	case types.ProposalTermsTypeNewMarket, types.ProposalTermsTypeNewSpotMarket:
		startAuction := true
		if p.State != types.ProposalStatePassed {
			startAuction = false
		}

		vc.m = &NewMarketVoteClosed{
			startAuction: startAuction,
		}
	}
	return vc
}

func (e *Engine) removeActiveProposalByID(ctx context.Context, id string) {
	for i, p := range e.activeProposals {
		if p.ID == id {
			copy(e.activeProposals[i:], e.activeProposals[i+1:])
			e.activeProposals[len(e.activeProposals)-1] = nil
			e.activeProposals = e.activeProposals[:len(e.activeProposals)-1]

			if p.State == types.ProposalStateDeclined || p.State == types.ProposalStateFailed || p.State == types.ProposalStateRejected {
				// if it's an asset proposal we need to update it's
				// state in the asset engine
				switch p.Terms.Change.GetTermType() {
				case types.ProposalTermsTypeNewAsset:
					e.assets.SetRejected(ctx, p.ID)
				}
			}
			return
		}
	}
}

func (e *Engine) OnTick(ctx context.Context, t time.Time) ([]*ToEnact, []*VoteClosed) {
	now := t.Unix()

	voteClosed, addToActiveProposals := e.evaluateBatchProposals(ctx, now)

	var (
		preparedToEnact []*ToEnact
		toBeRemoved     []string // ids
	)

	for _, proposal := range e.activeProposals {
		// check if the market for successor proposals still exists, if not, reject the proposal
		if nm := proposal.Terms.GetNewMarket(); nm != nil && nm.Successor() != nil {
			if _, err := e.markets.GetMarketState(proposal.ID); err != nil {
				proposal.RejectWithErr(types.ProposalErrorInvalidSuccessorMarket, ErrParentMarketSucceededByCompeting)
				e.broker.Send(events.NewProposalEvent(ctx, *proposal.Proposal))
				toBeRemoved = append(toBeRemoved, proposal.ID)
				continue
			}
		}

		// do not check parent market, the market was either rejected when the parent was succeeded
		// or, if the parent market state is gone (ie succession window has expired), the proposal simply
		// loses its parent market reference
		if proposal.ShouldClose(now) {
			e.closeProposal(ctx, proposal)
			voteClosed = append(voteClosed, e.preVoteClosedProposal(proposal))
		}

		if !proposal.IsOpen() && !proposal.IsPassed() {
			toBeRemoved = append(toBeRemoved, proposal.ID)
		} else if proposal.IsPassed() && (e.isAutoEnactableProposal(proposal.Proposal) || proposal.IsTimeToEnact(now)) {
			enact, perr, err := e.preEnactProposal(ctx, proposal)
			if err != nil {
				e.broker.Send(events.NewProposalEvent(ctx, *proposal.Proposal))
				toBeRemoved = append(toBeRemoved, proposal.ID)
				e.log.Error("proposal enactment has failed",
					logging.String("proposal-id", proposal.ID),
					logging.String("proposal-error", perr.String()),
					logging.Error(err))
			} else {
				toBeRemoved = append(toBeRemoved, proposal.ID)
				preparedToEnact = append(preparedToEnact, enact)
			}
		}
	}

	// then get all proposal accepted through node validation, and start their vote time.
	accepted, rejected := e.nodeProposalValidation.OnTick(t)
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

		// if it's an asset proposal we need to update it's
		// state in the asset engine
		switch p.Terms.Change.GetTermType() {
		case types.ProposalTermsTypeNewAsset:
			e.assets.SetRejected(ctx, p.ID)
		}
	}

	toBeEnacted := []*ToEnact{}
	for i, ep := range preparedToEnact {
		// this is the new market proposal, and should already be in the slice
		prop := *ep.ProposalData()

		propType := prop.Terms.Change.GetTermType()
		id := prop.ID
		if propType == types.ProposalTermsTypeNewMarket || propType == types.ProposalTermsTypeUpdateMarket {
			if propType == types.ProposalTermsTypeUpdateMarket {
				id = prop.Terms.GetUpdateMarket().MarketID
			}

			// before trying to enact a successor market proposal, check if the market wasn't rejected due
			// to another successor leaving opening auction
			if _, err := e.markets.GetMarketState(id); err != nil {
				if nm := prop.Terms.GetNewMarket(); nm != nil && nm.Successor() != nil {
					prop.RejectWithErr(types.ProposalErrorInvalidSuccessorMarket, ErrParentMarketSucceededByCompeting)
					e.broker.Send(events.NewProposalEvent(ctx, *prop.Proposal))
					continue
				}
				e.log.Error("could not get state of market %s", logging.String("market-id", id))
				continue
			}
		}

		e.enactedProposals = append(e.enactedProposals, &prop)
		toBeEnacted = append(toBeEnacted, preparedToEnact[i])
	}

	// now we iterate over all proposal ids to remove them from the list
	for _, id := range toBeRemoved {
		e.removeActiveProposalByID(ctx, id)
	}

	// add these passed proposals from batch to prepare them for enactment
	e.activeProposals = append(e.activeProposals, addToActiveProposals...)

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
		ID:                      id,
		Timestamp:               e.timeService.GetTimeNow().UnixNano(),
		Party:                   party,
		State:                   types.ProposalStateOpen,
		Terms:                   psub.Terms,
		Reference:               psub.Reference,
		Rationale:               psub.Rationale,
		RequiredMajority:        num.DecimalZero(),
		RequiredParticipation:   num.DecimalZero(),
		RequiredLPMajority:      num.DecimalZero(),
		RequiredLPParticipation: num.DecimalZero(),
	}

	defer func() {
		e.broker.Send(events.NewProposalEvent(ctx, *p))
	}()

	params, err := e.getProposalParams(p.Terms.Change)
	if err != nil {
		p.RejectWithErr(types.ProposalErrorUnknownType, err)
		return nil, err
	}

	if perr, err := e.validateOpenProposal(p, params); err != nil {
		p.RejectWithErr(perr, err)
		if e.log.IsDebug() {
			e.log.Debug("Proposal rejected",
				logging.String("proposal-id", p.ID),
				logging.String("proposal details", p.String()),
			)
		}
		return nil, err
	}

	// now if it's a 2 steps proposal, start the node votes
	if e.isTwoStepsProposal(p) {
		p.WaitForNodeVote()
		if err := e.startTwoStepsProposal(ctx, p); err != nil {
			p.RejectWithErr(types.ProposalErrorNodeValidationFailed, err)
			if e.log.IsDebug() {
				e.log.Debug("Proposal rejected",
					logging.String("proposal-id", p.ID),
					logging.String("proposal details", p.String()),
				)
			}
			return nil, err
		}
	} else {
		e.startProposal(p)
	}

	return e.intoToSubmit(ctx, p, &enactmentTime{current: p.Terms.EnactmentTimestamp}, false)
}

func (e *Engine) RejectProposal(
	ctx context.Context, p *types.Proposal, r types.ProposalError, errorDetails error,
) error {
	if _, ok := e.getProposal(p.ID); !ok {
		return ErrProposalDoesNotExist
	}

	e.rejectProposal(ctx, p, r, errorDetails)
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

func (e *Engine) rejectProposal(ctx context.Context, p *types.Proposal, r types.ProposalError, errorDetails error) {
	e.removeActiveProposalByID(ctx, p.ID)
	p.RejectWithErr(r, errorDetails)
}

// toSubmit build the return response for the SubmitProposal
// method.
func (e *Engine) intoToSubmit(ctx context.Context, p *types.Proposal, enct *enactmentTime, restore bool) (*ToSubmit, error) {
	tsb := &ToSubmit{p: p}

	switch p.Terms.Change.GetTermType() {
	case types.ProposalTermsTypeNewMarket:
		// use to calculate the auction duration
		// which is basically enacttime - closetime
		// FIXME(): normally we should use the closetime
		// but this would not play well with the MarketAuctionState stuff
		// for now we start the auction as of now.
		newMarket := p.Terms.GetNewMarket()
		var parent *types.Market
		if suc := newMarket.Successor(); suc != nil {
			pm, ok := e.markets.GetMarket(suc.ParentID, true)
			if !ok {
				if !restore {
					e.rejectProposal(ctx, p, types.ProposalErrorInvalidSuccessorMarket, ErrParentMarketDoesNotExist)
					return nil, fmt.Errorf("%w, %v", ErrParentMarketDoesNotExist, types.ProposalErrorInvalidSuccessorMarket)
				}
			} else {
				parent = &pm
			}
			// proposal to succeed a market that was already succeeded
			// on restore, the parent market may be succeeded and the restored market may have own state (ie was the successor)
			// So skip this check when restoring markets from checkpoints
			if !restore && e.markets.IsSucceeded(suc.ParentID) {
				e.rejectProposal(ctx, p, types.ProposalErrorInvalidSuccessorMarket, ErrParentMarketAlreadySucceeded)
				return nil, fmt.Errorf("%w, %v", ErrParentMarketAlreadySucceeded, types.ProposalErrorInvalidSuccessorMarket)
			}
			// restore can be true while parent == nil. This is fine, though
		}
		now := e.timeService.GetTimeNow()
		closeTime := time.Unix(p.Terms.ClosingTimestamp, 0)
		enactTime := time.Unix(p.Terms.EnactmentTimestamp, 0)
		auctionDuration := enactTime.Sub(closeTime)
		if perr, err := validateNewMarketChange(newMarket, e.assets, true, e.netp, auctionDuration, enct, parent, now, restore); err != nil {
			e.rejectProposal(ctx, p, perr, err)
			return nil, fmt.Errorf("%w, %v", err, perr)
		}
		// closeTime = e.timeService.GetTimeNow().Round(time.Second)
		// auctionDuration = enactTime.Sub(closeTime)
		mkt, perr, err := buildMarketFromProposal(p.ID, newMarket, e.netp, auctionDuration)
		if err != nil {
			e.rejectProposal(ctx, p, perr, err)
			return nil, fmt.Errorf("%w, %v", err, perr)
		}
		tsb.m = &ToSubmitNewMarket{
			m: mkt,
		}
	case types.ProposalTermsTypeNewSpotMarket:
		closeTime := time.Unix(p.Terms.ClosingTimestamp, 0)
		enactTime := time.Unix(p.Terms.EnactmentTimestamp, 0)
		newMarket := p.Terms.GetNewSpotMarket()
		auctionDuration := enactTime.Sub(closeTime)
		if perr, err := validateNewSpotMarketChange(newMarket, e.assets, true, e.netp, auctionDuration, enct); err != nil {
			e.rejectProposal(ctx, p, perr, err)
			return nil, fmt.Errorf("%w, %v", err, perr)
		}
		mkt, perr, err := buildSpotMarketFromProposal(p.ID, newMarket, e.netp, auctionDuration)
		if err != nil {
			e.rejectProposal(ctx, p, perr, err)
			return nil, fmt.Errorf("%w, %v", err, perr)
		}
		tsb.s = &ToSubmitNewSpotMarket{
			m: mkt,
		}
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
}

func (e *Engine) startValidatedProposal(p *proposal) {
	e.activeProposals = append(e.activeProposals, p)
}

func (e *Engine) startTwoStepsProposal(ctx context.Context, p *types.Proposal) error {
	return e.nodeProposalValidation.Start(ctx, p)
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

func (e *Engine) getProposalParams(proposalTerm types.ProposalTerm) (*types.ProposalParameters, error) {
	switch proposalTerm.GetTermType() {
	case types.ProposalTermsTypeNewMarket:
		return e.getNewMarketProposalParameters(), nil
	case types.ProposalTermsTypeUpdateMarket:
		return e.getUpdateMarketProposalParameters(), nil
	case types.ProposalTermsTypeNewAsset:
		return e.getNewAssetProposalParameters(), nil
	case types.ProposalTermsTypeUpdateAsset:
		return e.getUpdateAssetProposalParameters(), nil
	case types.ProposalTermsTypeUpdateNetworkParameter:
		return e.getUpdateNetworkParameterProposalParameters(), nil
	case types.ProposalTermsTypeNewFreeform:
		return e.getNewFreeformProposalParameters(), nil
	case types.ProposalTermsTypeNewTransfer:
		return e.getNewTransferProposalParameters(), nil
	case types.ProposalTermsTypeCancelTransfer:
		// for governance transfer cancellation reuse the governance transfer proposal params
		return e.getNewTransferProposalParameters(), nil
	case types.ProposalTermsTypeNewSpotMarket:
		return e.getNewSpotMarketProposalParameters(), nil
	case types.ProposalTermsTypeUpdateSpotMarket:
		return e.getUpdateSpotMarketProposalParameters(), nil
	case types.ProposalTermsTypeUpdateMarketState:
		// reusing market update net params
		return e.getUpdateMarketStateProposalParameters(), nil
	case types.ProposalTermsTypeUpdateReferralProgram:
		return e.getReferralProgramNetworkParameters(), nil
	case types.ProposalTermsTypeUpdateVolumeDiscountProgram:
		return e.getVolumeDiscountProgramNetworkParameters(), nil
	default:
		return nil, ErrUnsupportedProposalType
	}
}

func (e *Engine) validateOpenProposal(proposal *types.Proposal, params *types.ProposalParameters) (types.ProposalError, error) {
	// assign all requirement to the proposal itself.
	proposal.RequiredMajority = params.RequiredMajority
	proposal.RequiredParticipation = params.RequiredParticipation
	proposal.RequiredLPMajority = params.RequiredMajorityLP
	proposal.RequiredLPParticipation = params.RequiredParticipationLP

	now := e.timeService.GetTimeNow()
	closeTime := time.Unix(proposal.Terms.ClosingTimestamp, 0)
	minCloseTime := now.Add(params.MinClose)
	if closeTime.Before(minCloseTime) {
		e.log.Debug("proposal close time is too soon",
			logging.Time("expected-min", minCloseTime),
			logging.Time("provided", closeTime),
			logging.String("id", proposal.ID))
		return types.ProposalErrorCloseTimeTooSoon,
			fmt.Errorf("proposal closing time too soon, expected > %v, got %v", minCloseTime.UTC(), closeTime.UTC())
	}

	maxCloseTime := now.Add(params.MaxClose)
	if closeTime.After(maxCloseTime) {
		e.log.Debug("proposal close time is too late",
			logging.Time("expected-max", maxCloseTime),
			logging.Time("provided", closeTime),
			logging.String("id", proposal.ID))
		return types.ProposalErrorCloseTimeTooLate,
			fmt.Errorf("proposal closing time too late, expected < %v, got %v", maxCloseTime.UTC(), closeTime.UTC())
	}

	enactTime := time.Unix(proposal.Terms.EnactmentTimestamp, 0)
	minEnactTime := now.Add(params.MinEnact)
	if !e.isAutoEnactableProposal(proposal) && enactTime.Before(minEnactTime) {
		e.log.Debug("proposal enact time is too soon",
			logging.Time("expected-min", minEnactTime),
			logging.Time("provided", enactTime),
			logging.String("id", proposal.ID))
		return types.ProposalErrorEnactTimeTooSoon,
			fmt.Errorf("proposal enactment time too soon, expected > %v, got %v", minEnactTime.UTC(), enactTime.UTC())
	}

	maxEnactTime := now.Add(params.MaxEnact)
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
		if closeTime.Before(now) {
			e.log.Debug("proposal validation time can't be in the past",
				logging.Time("now", now),
				logging.Time("validation-time", validationTime),
				logging.String("id", proposal.ID))
			return types.ProposalErrorIncompatibleTimestamps,
				fmt.Errorf("proposal validation time cannot be in the past, expected > %v got %v", now.UTC(), validationTime.UTC())
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

	checkProposerToken := true

	if proposal.IsMarketUpdate() || proposal.IsMarketStateUpdate() {
		marketID := ""
		if proposal.Terms.GetMarketStateUpdate() != nil {
			marketID = proposal.Terms.GetMarketStateUpdate().Changes.MarketID
		} else {
			marketID = proposal.MarketUpdate().MarketID
		}
		proposalError, err := e.validateMarketUpdate(proposal.ID, marketID, proposal.Party, params)
		if err != nil && proposalError != types.ProposalErrorInsufficientEquityLikeShare {
			return proposalError, err
		}
		checkProposerToken = proposalError == types.ProposalErrorInsufficientEquityLikeShare
	}

	if proposal.IsSpotMarketUpdate() {
		proposalError, err := e.validateSpotMarketUpdate(proposal, params)
		if err != nil && proposalError != types.ProposalErrorInsufficientEquityLikeShare {
			return proposalError, err
		}
		checkProposerToken = proposalError == types.ProposalErrorInsufficientEquityLikeShare
	}

	if checkProposerToken {
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
	proposal, found := e.getProposal(cmd.ProposalID)
	batchProposal, batchFound := e.getBatchProposal(cmd.ProposalID)

	if found {
		if !proposal.IsOpenForVotes() {
			return ErrProposalNotOpenForVotes
		}
		return e.addVote(ctx, cmd, proposal, party)
	}

	if batchFound {
		if !batchProposal.IsOpenForVotes() {
			return ErrProposalNotOpenForVotes
		}
		return e.addBatchVote(ctx, batchProposal, cmd, party)
	}

	return ErrProposalDoesNotExist
}

func (e *Engine) addVote(ctx context.Context, cmd types.VoteSubmission, proposal *proposal, party string) error {
	params, err := e.getProposalParams(proposal.Terms.Change)
	if err != nil {
		return err
	}

	if err := e.canVote(proposal.Proposal, params, party); err != nil {
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
		Timestamp:                   e.timeService.GetTimeNow().UnixNano(),
		TotalGovernanceTokenBalance: getTokensBalance(e.accs, party),
		TotalGovernanceTokenWeight:  num.DecimalZero(),
		TotalEquityLikeShareWeight:  num.DecimalZero(),
	}
	if proposal.IsMarketUpdate() {
		mID := proposal.MarketUpdate().MarketID
		vote.TotalEquityLikeShareWeight, _ = e.markets.GetEquityLikeShareForMarketAndParty(mID, party)
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

func (e *Engine) canVote(
	proposal *types.Proposal,
	params *types.ProposalParameters,
	party string,
) error {
	voterTokens, err := getGovernanceTokens(e.accs, party)
	if err != nil {
		return err
	}

	if proposal.IsMarketUpdate() {
		partyELS, _ := e.markets.GetEquityLikeShareForMarketAndParty(proposal.MarketUpdate().MarketID, party)
		if partyELS.IsZero() && voterTokens.IsZero() {
			return ErrVoterInsufficientTokensAndEquityLikeShare
		}
		// If he is not voting using his equity-like share, he should at least
		// have enough tokens.
		if partyELS.IsZero() && voterTokens.LT(params.MinVoterBalance) {
			return ErrVoterInsufficientTokens
		}
	} else {
		if voterTokens.LT(params.MinVoterBalance) {
			return ErrVoterInsufficientTokens
		}
	}

	return nil
}

func (e *Engine) validateMarketUpdate(ID, marketID, party string, params *types.ProposalParameters) (types.ProposalError, error) {
	if !e.markets.MarketExists(marketID) {
		e.log.Debug("market does not exist",
			logging.MarketID(marketID),
			logging.PartyID(party),
			logging.ProposalID(ID))
		return types.ProposalErrorInvalidMarket, ErrMarketDoesNotExist
	}
	for _, p := range e.activeProposals {
		if p.ID == marketID && p.IsOpen() {
			return types.ProposalErrorInvalidMarket, ErrMarketProposalStillOpen
		}
	}

	partyELS, _ := e.markets.GetEquityLikeShareForMarketAndParty(marketID, party)
	if partyELS.LessThan(params.MinEquityLikeShare) {
		e.log.Debug("proposer have insufficient equity-like share",
			logging.String("expect-balance", params.MinEquityLikeShare.String()),
			logging.String("proposer-balance", partyELS.String()),
			logging.PartyID(party),
			logging.MarketID(marketID),
			logging.ProposalID(ID))
		return types.ProposalErrorInsufficientEquityLikeShare,
			fmt.Errorf("proposer have insufficient equity-like share, expected >= %v got %v", params.MinEquityLikeShare, partyELS)
	}

	return types.ProposalErrorUnspecified, nil
}

func (e *Engine) validateSpotMarketUpdate(proposal *types.Proposal, params *types.ProposalParameters) (types.ProposalError, error) {
	updateMarket := proposal.SpotMarketUpdate()
	if !e.markets.MarketExists(updateMarket.MarketID) {
		e.log.Debug("market does not exist",
			logging.MarketID(updateMarket.MarketID),
			logging.PartyID(proposal.Party),
			logging.ProposalID(proposal.ID))
		return types.ProposalErrorInvalidMarket, ErrMarketDoesNotExist
	}
	for _, p := range e.activeProposals {
		if p.ID == updateMarket.MarketID && p.IsOpen() {
			return types.ProposalErrorInvalidMarket, ErrMarketProposalStillOpen
		}
	}

	partyELS, _ := e.markets.GetEquityLikeShareForMarketAndParty(updateMarket.MarketID, proposal.Party)
	if partyELS.LessThan(params.MinEquityLikeShare) {
		e.log.Debug("proposer have insufficient equity-like share",
			logging.String("expect-balance", params.MinEquityLikeShare.String()),
			logging.String("proposer-balance", partyELS.String()),
			logging.PartyID(proposal.Party),
			logging.MarketID(updateMarket.MarketID),
			logging.ProposalID(proposal.ID))
		return types.ProposalErrorInsufficientEquityLikeShare,
			fmt.Errorf("proposer have insufficient equity-like share, expected >= %v got %v", params.MinEquityLikeShare, partyELS)
	}

	return types.ProposalErrorUnspecified, nil
}

func (e *Engine) validateChange(terms *types.ProposalTerms) (types.ProposalError, error) {
	enactTime := time.Unix(terms.EnactmentTimestamp, 0)
	enct := &enactmentTime{current: terms.EnactmentTimestamp}

	switch terms.Change.GetTermType() {
	case types.ProposalTermsTypeNewMarket:
		closeTime := time.Unix(terms.ClosingTimestamp, 0)
		newMarket := terms.GetNewMarket()
		var parent *types.Market
		if suc := newMarket.Successor(); suc != nil {
			pm, ok := e.markets.GetMarket(suc.ParentID, true)
			if !ok {
				return types.ProposalErrorInvalidSuccessorMarket, ErrParentMarketDoesNotExist
			}
			parent = &pm
		}
		return validateNewMarketChange(newMarket, e.assets, true, e.netp, enactTime.Sub(closeTime), enct, parent, e.timeService.GetTimeNow(), false)
	case types.ProposalTermsTypeUpdateMarket:
		enct.shouldNotVerify = true
		marketUpdate := terms.GetUpdateMarket()
		mkt, ok := e.markets.GetMarket(marketUpdate.MarketID, false)
		if !ok {
			return types.ProposalErrorInvalidMarket, ErrParentMarketDoesNotExist
		}

		return validateUpdateMarketChange(marketUpdate, mkt, enct, e.timeService.GetTimeNow(), e.netp)
	case types.ProposalTermsTypeNewAsset:
		return e.validateNewAssetProposal(terms.GetNewAsset())
	case types.ProposalTermsTypeUpdateAsset:
		return terms.GetUpdateAsset().Validate()
	case types.ProposalTermsTypeUpdateNetworkParameter:
		return validateNetworkParameterUpdate(e.netp, terms.GetUpdateNetworkParameter().Changes)
	case types.ProposalTermsTypeNewTransfer:
		return e.validateGovernanceTransfer(terms.GetNewTransfer())
	case types.ProposalTermsTypeCancelTransfer:
		return e.validateCancelGovernanceTransfer(terms.GetCancelTransfer().Changes.TransferID)
	case types.ProposalTermsTypeUpdateMarketState:
		return e.validateMarketUpdateState(terms.GetMarketStateUpdate().Changes)
	case types.ProposalTermsTypeNewSpotMarket:
		closeTime := time.Unix(terms.ClosingTimestamp, 0)
		return validateNewSpotMarketChange(terms.GetNewSpotMarket(), e.assets, true, e.netp, enactTime.Sub(closeTime), enct)
	case types.ProposalTermsTypeUpdateSpotMarket:
		enct.shouldNotVerify = true
		return validateUpdateSpotMarketChange(terms.GetUpdateSpotMarket())
	case types.ProposalTermsTypeUpdateReferralProgram:
		return validateUpdateReferralProgram(e.netp, terms.GetUpdateReferralProgram(), terms.EnactmentTimestamp)
	case types.ProposalTermsTypeUpdateVolumeDiscountProgram:
		return validateUpdateVolumeDiscountProgram(e.netp, terms.GetUpdateVolumeDiscountProgram())
	default:
		return types.ProposalErrorUnspecified, nil
	}
}

func (e *Engine) validateGovernanceTransfer(newTransfer *types.NewTransfer) (types.ProposalError, error) {
	if err := e.banking.VerifyGovernanceTransfer(newTransfer.Changes); err != nil {
		return types.ProporsalErrorInvalidGovernanceTransfer, err
	}
	return types.ProposalErrorUnspecified, nil
}

func (e *Engine) validateCancelGovernanceTransfer(transferID string) (types.ProposalError, error) {
	if err := e.banking.VerifyCancelGovernanceTransfer(transferID); err != nil {
		return types.ProporsalErrorFailedGovernanceTransferCancel, err
	}
	return types.ProposalErrorUnspecified, nil
}

func (e *Engine) validateMarketUpdateState(update *types.MarketStateUpdateConfiguration) (types.ProposalError, error) {
	marketID := update.MarketID
	if !e.markets.MarketExists(marketID) {
		e.log.Debug("market does not exist", logging.MarketID(marketID))
		return types.ProposalErrorInvalidMarket, ErrMarketDoesNotExist
	}

	marketState, err := e.markets.GetMarketState(marketID)
	if err != nil {
		return types.ProposalErrorInvalidMarket, err
	}

	// if the market is already terminated or not yet started or settled
	if marketState == types.MarketStateCancelled || marketState == types.MarketStateClosed || marketState == types.MarketStateTradingTerminated || marketState == types.MarketStateSettled || marketState == types.MarketStateProposed {
		return types.ProposalErrorInvalidMarket, ErrMarketStateUpdateNotAllowed
	}

	return types.ProposalErrorUnspecified, nil
}

func (e *Engine) validateNewAssetProposal(newAsset *types.NewAsset) (types.ProposalError, error) {
	if perr, err := newAsset.Validate(); err != nil {
		return perr, err
	}

	erc20 := newAsset.GetChanges().GetERC20()
	if erc20 == nil {
		// not and erc20 asset, nothing todo
		return types.ProposalErrorUnspecified, nil
	}

	// if we are an erc20 proposal
	// now we ensure no other proposal is ongoing for this asset, or that
	// any asset already exists for this address

	for _, p := range e.activeProposals {
		p := p.Terms.GetNewAsset()
		if p == nil {
			continue
		}
		if source := p.Changes.GetERC20(); source != nil {
			if strings.EqualFold(source.ContractAddress, erc20.ContractAddress) {
				return types.ProposalErrorERC20AddressAlreadyInUse, ErrErc20AddressAlreadyInUse
			}
		}
	}

	for _, p := range e.enactedProposals {
		p := p.Terms.GetNewAsset()
		if p == nil {
			continue
		}
		if source := p.Changes.GetERC20(); source != nil {
			if strings.EqualFold(source.ContractAddress, erc20.ContractAddress) {
				return types.ProposalErrorERC20AddressAlreadyInUse, ErrErc20AddressAlreadyInUse
			}
		}
	}

	if e.assets.ExistsForEthereumAddress(erc20.ContractAddress) {
		return types.ProposalErrorERC20AddressAlreadyInUse, ErrErc20AddressAlreadyInUse
	}

	return types.ProposalErrorUnspecified, nil
}

func (e *Engine) closeProposal(ctx context.Context, proposal *proposal) {
	if !proposal.IsOpen() {
		return
	}

	proposal.Close(e.accs, e.markets)
	if proposal.IsPassed() {
		e.log.Debug("Proposal passed", logging.ProposalID(proposal.ID))
	} else if proposal.IsDeclined() {
		e.log.Debug("Proposal declined", logging.ProposalID(proposal.ID), logging.String("details", proposal.ErrorDetails), logging.String("reason", proposal.Reason.String()))
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

	sort.SliceStable(votes, func(i, j int) bool {
		return votes[i].PartyID() < votes[j].PartyID()
	})

	evts := make([]events.Event, 0, len(votes))
	for _, e := range votes {
		evts = append(evts, e)
	}

	return evts
}

func (e *Engine) updateValidatorKey(ctx context.Context, m map[string]*types.Vote, oldKey, newKey string) {
	if vote, ok := m[oldKey]; ok {
		delete(m, oldKey)
		vote.PartyID = newKey
		e.broker.Send(events.NewVoteEvent(ctx, *vote))
		m[newKey] = vote
	}
}

func (e *Engine) updatedSpotMarketFromProposal(p *proposal) (*types.Market, types.ProposalError, error) {
	terms := p.Terms.GetUpdateSpotMarket()
	existingMarket, exists := e.markets.GetMarket(terms.MarketID, false)
	if !exists {
		return nil, types.ProposalErrorInvalidMarket, fmt.Errorf("market \"%s\" doesn't exist anymore", terms.MarketID)
	}

	newMarket := &types.NewSpotMarket{
		Changes: &types.NewSpotMarketConfiguration{
			DecimalPlaces:             existingMarket.DecimalPlaces,
			PositionDecimalPlaces:     existingMarket.PositionDecimalPlaces,
			Metadata:                  terms.Changes.Metadata,
			PriceMonitoringParameters: terms.Changes.PriceMonitoringParameters,
			TargetStakeParameters:     terms.Changes.TargetStakeParameters,
			SLAParams:                 terms.Changes.SLAParams,
		},
	}

	if perr, err := validateUpdateSpotMarketChange(terms); err != nil {
		return nil, perr, err
	}

	previousAuctionDuration := time.Duration(existingMarket.OpeningAuction.Duration) * time.Second
	return buildSpotMarketFromProposal(existingMarket.ID, newMarket, e.netp, previousAuctionDuration)
}

func (e *Engine) updatedMarketFromProposal(p *proposal) (*types.Market, types.ProposalError, error) {
	terms := p.Terms.GetUpdateMarket()
	existingMarket, exists := e.markets.GetMarket(terms.MarketID, false)
	if !exists {
		return nil, types.ProposalErrorInvalidMarket, fmt.Errorf("market \"%s\" doesn't exist anymore", terms.MarketID)
	}

	newMarket := &types.NewMarket{
		Changes: &types.NewMarketConfiguration{
			Instrument: &types.InstrumentConfiguration{
				Name: terms.Changes.Instrument.Name,
				Code: terms.Changes.Instrument.Code,
			},
			DecimalPlaces:                 existingMarket.DecimalPlaces,
			PositionDecimalPlaces:         existingMarket.PositionDecimalPlaces,
			Metadata:                      terms.Changes.Metadata,
			PriceMonitoringParameters:     terms.Changes.PriceMonitoringParameters,
			LiquidityMonitoringParameters: terms.Changes.LiquidityMonitoringParameters,
			LiquiditySLAParameters:        terms.Changes.LiquiditySLAParameters,
			LinearSlippageFactor:          terms.Changes.LinearSlippageFactor,
			QuadraticSlippageFactor:       terms.Changes.QuadraticSlippageFactor,
			LiquidityFeeSettings:          terms.Changes.LiquidityFeeSettings,
			LiquidationStrategy:           terms.Changes.LiquidationStrategy,
			MarkPriceConfiguration:        terms.Changes.MarkPriceConfiguration,
		},
	}

	switch riskModel := terms.Changes.RiskParameters.(type) {
	case nil:
		return nil, types.ProposalErrorNoRiskParameters, ErrMissingRiskParameters
	case *types.UpdateMarketConfigurationSimple:
		newMarket.Changes.RiskParameters = &types.NewMarketConfigurationSimple{
			Simple: riskModel.Simple,
		}
	case *types.UpdateMarketConfigurationLogNormal:
		newMarket.Changes.RiskParameters = &types.NewMarketConfigurationLogNormal{
			LogNormal: riskModel.LogNormal,
		}
	default:
		return nil, types.ProposalErrorUnknownRiskParameterType, ErrUnsupportedRiskParameters
	}

	switch product := terms.Changes.Instrument.Product.(type) {
	case nil:
		return nil, types.ProposalErrorNoProduct, ErrMissingProduct
	case *types.UpdateInstrumentConfigurationFuture:
		assets, _ := existingMarket.GetAssets()
		newMarket.Changes.Instrument.Product = &types.InstrumentConfigurationFuture{
			Future: &types.FutureProduct{
				SettlementAsset:                     assets[0],
				QuoteName:                           product.Future.QuoteName,
				DataSourceSpecForSettlementData:     product.Future.DataSourceSpecForSettlementData,
				DataSourceSpecForTradingTermination: product.Future.DataSourceSpecForTradingTermination,
				DataSourceSpecBinding:               product.Future.DataSourceSpecBinding,
			},
		}
	case *types.UpdateInstrumentConfigurationPerps:
		assets, _ := existingMarket.GetAssets()
		newMarket.Changes.Instrument.Product = &types.InstrumentConfigurationPerps{
			Perps: &types.PerpsProduct{
				SettlementAsset:                     assets[0],
				QuoteName:                           product.Perps.QuoteName,
				MarginFundingFactor:                 product.Perps.MarginFundingFactor,
				InterestRate:                        product.Perps.InterestRate,
				ClampLowerBound:                     product.Perps.ClampLowerBound,
				ClampUpperBound:                     product.Perps.ClampUpperBound,
				FundingRateScalingFactor:            product.Perps.FundingRateScalingFactor,
				FundingRateLowerBound:               product.Perps.FundingRateLowerBound,
				FundingRateUpperBound:               product.Perps.FundingRateUpperBound,
				DataSourceSpecForSettlementData:     product.Perps.DataSourceSpecForSettlementData,
				DataSourceSpecForSettlementSchedule: product.Perps.DataSourceSpecForSettlementSchedule,
				DataSourceSpecBinding:               product.Perps.DataSourceSpecBinding,
				InternalCompositePriceConfig:        product.Perps.InternalCompositePrice,
			},
		}
	default:
		return nil, types.ProposalErrorUnsupportedProduct, ErrUnsupportedProduct
	}
	// assign the current liquidation strategy if none was set on the update proposal
	if newMarket.Changes.LiquidationStrategy == nil {
		newMarket.Changes.LiquidationStrategy = existingMarket.LiquidationStrategy
	}

	if perr, err := validateUpdateMarketChange(terms, existingMarket, &enactmentTime{current: p.Terms.EnactmentTimestamp, shouldNotVerify: true}, e.timeService.GetTimeNow(), e.netp); err != nil {
		return nil, perr, err
	}

	previousAuctionDuration := time.Duration(existingMarket.OpeningAuction.Duration) * time.Second
	return buildMarketFromProposal(existingMarket.ID, newMarket, e.netp, previousAuctionDuration)
}

func (e *Engine) updatedAssetFromProposal(p *proposal) (*types.Asset, types.ProposalError, error) {
	a := p.Terms.GetUpdateAsset()
	existingAsset, err := e.assets.Get(a.AssetID)
	if err != nil {
		return nil, types.ProposalErrorInvalidAsset, err
	}

	newAsset := &types.Asset{
		ID: a.AssetID,
		Details: &types.AssetDetails{
			Name:     existingAsset.ToAssetType().Details.Name,
			Symbol:   existingAsset.ToAssetType().Details.Symbol,
			Quantum:  a.Changes.Quantum,
			Decimals: existingAsset.DecimalPlaces(),
		},
	}

	switch src := a.Changes.Source.(type) {
	case *types.AssetDetailsUpdateERC20:
		erc20, ok := existingAsset.ERC20()
		if !ok {
			return nil, types.ProposalErrorInvalidAsset, ErrExpectedERC20Asset
		}
		newAsset.Details.Source = &types.AssetDetailsErc20{
			ERC20: &types.ERC20{
				ContractAddress:   erc20.Address(),
				LifetimeLimit:     src.ERC20Update.LifetimeLimit.Clone(),
				WithdrawThreshold: src.ERC20Update.WithdrawThreshold.Clone(),
			},
		}
	default:
		return nil, types.ProposalErrorInvalidAsset, ErrUnsupportedAssetSourceType
	}

	return newAsset, types.ProposalErrorUnspecified, nil
}

func (e *Engine) OnChainIDUpdate(cID uint64) error {
	e.chainID = cID
	return nil
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
