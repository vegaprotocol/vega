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

package types

import (
	"fmt"
	"time"

	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/ptr"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
)

type GovernanceData = vegapb.GovernanceData

type ProposalError = vegapb.ProposalError

const (
	// ProposalErrorUnspecified Default value, always invalid.
	ProposalErrorUnspecified ProposalError = vegapb.ProposalError_PROPOSAL_ERROR_UNSPECIFIED
	// ProposalErrorCloseTimeTooSoon The specified close time is too early base on network parameters.
	ProposalErrorCloseTimeTooSoon ProposalError = vegapb.ProposalError_PROPOSAL_ERROR_CLOSE_TIME_TOO_SOON
	// ProposalErrorCloseTimeTooLate The specified close time is too late based on network parameters.
	ProposalErrorCloseTimeTooLate ProposalError = vegapb.ProposalError_PROPOSAL_ERROR_CLOSE_TIME_TOO_LATE
	// ProposalErrorEnactTimeTooSoon The specified enact time is too early based on network parameters.
	ProposalErrorEnactTimeTooSoon ProposalError = vegapb.ProposalError_PROPOSAL_ERROR_ENACT_TIME_TOO_SOON
	// ProposalErrorEnactTimeTooLate The specified enact time is too late based on network parameters.
	ProposalErrorEnactTimeTooLate ProposalError = vegapb.ProposalError_PROPOSAL_ERROR_ENACT_TIME_TOO_LATE
	// ProposalErrorInsufficientTokens The proposer for this proposal as insufficient tokens.
	ProposalErrorInsufficientTokens ProposalError = vegapb.ProposalError_PROPOSAL_ERROR_INSUFFICIENT_TOKENS
	// ProposalErrorNoProduct The proposal has no product.
	ProposalErrorNoProduct ProposalError = vegapb.ProposalError_PROPOSAL_ERROR_NO_PRODUCT
	// ProposalErrorUnsupportedProduct The specified product is not supported.
	ProposalErrorUnsupportedProduct ProposalError = vegapb.ProposalError_PROPOSAL_ERROR_UNSUPPORTED_PRODUCT
	// ProposalErrorNodeValidationFailed The proposal failed node validation.
	ProposalErrorNodeValidationFailed ProposalError = vegapb.ProposalError_PROPOSAL_ERROR_NODE_VALIDATION_FAILED
	// ProposalErrorMissingBuiltinAssetField A field is missing in a builtin asset source.
	ProposalErrorMissingBuiltinAssetField ProposalError = vegapb.ProposalError_PROPOSAL_ERROR_MISSING_BUILTIN_ASSET_FIELD
	// ProposalErrorMissingErc20ContractAddress The contract address is missing in the ERC20 asset source.
	ProposalErrorMissingErc20ContractAddress ProposalError = vegapb.ProposalError_PROPOSAL_ERROR_MISSING_ERC20_CONTRACT_ADDRESS
	// ProposalErrorInvalidAsset The asset identifier is invalid or does not exist on the Vega network.
	ProposalErrorInvalidAsset ProposalError = vegapb.ProposalError_PROPOSAL_ERROR_INVALID_ASSET
	// ProposalErrorIncompatibleTimestamps Proposal terms timestamps are not compatible (Validation < Closing < Enactment).
	ProposalErrorIncompatibleTimestamps ProposalError = vegapb.ProposalError_PROPOSAL_ERROR_INCOMPATIBLE_TIMESTAMPS
	// ProposalErrorNoRiskParameters No risk parameters were specified.
	ProposalErrorNoRiskParameters ProposalError = vegapb.ProposalError_PROPOSAL_ERROR_NO_RISK_PARAMETERS
	// ProposalErrorNetworkParameterInvalidKey Invalid key in update network parameter proposal.
	ProposalErrorNetworkParameterInvalidKey ProposalError = vegapb.ProposalError_PROPOSAL_ERROR_NETWORK_PARAMETER_INVALID_KEY
	// ProposalErrorNetworkParameterInvalidValue Invalid valid in update network parameter proposal.
	ProposalErrorNetworkParameterInvalidValue ProposalError = vegapb.ProposalError_PROPOSAL_ERROR_NETWORK_PARAMETER_INVALID_VALUE
	// ProposalErrorNetworkParameterValidationFailed Validation failed for network parameter proposal.
	ProposalErrorNetworkParameterValidationFailed ProposalError = vegapb.ProposalError_PROPOSAL_ERROR_NETWORK_PARAMETER_VALIDATION_FAILED
	// ProposalErrorOpeningAuctionDurationTooSmall Opening auction duration is less than the network minimum opening auction time.
	ProposalErrorOpeningAuctionDurationTooSmall ProposalError = vegapb.ProposalError_PROPOSAL_ERROR_OPENING_AUCTION_DURATION_TOO_SMALL
	// ProposalErrorOpeningAuctionDurationTooLarge Opening auction duration is more than the network minimum opening auction time.
	ProposalErrorOpeningAuctionDurationTooLarge ProposalError = vegapb.ProposalError_PROPOSAL_ERROR_OPENING_AUCTION_DURATION_TOO_LARGE
	// ProposalErrorCouldNotInstantiateMarket Market proposal market could not be instantiated during execution.
	ProposalErrorCouldNotInstantiateMarket ProposalError = vegapb.ProposalError_PROPOSAL_ERROR_COULD_NOT_INSTANTIATE_MARKET
	// ProposalErrorInvalidFutureProduct Market proposal market contained invalid product definition.
	ProposalErrorInvalidFutureProduct ProposalError = vegapb.ProposalError_PROPOSAL_ERROR_INVALID_FUTURE_PRODUCT
	// ProposalErrorInvalidRiskParameter Market proposal invalid risk parameter.
	ProposalErrorInvalidRiskParameter ProposalError = vegapb.ProposalError_PROPOSAL_ERROR_INVALID_RISK_PARAMETER
	// ProposalErrorMajorityThresholdNotReached Proposal was declined because vote didn't reach the majority threshold required.
	ProposalErrorMajorityThresholdNotReached ProposalError = vegapb.ProposalError_PROPOSAL_ERROR_MAJORITY_THRESHOLD_NOT_REACHED
	// ProposalErrorParticipationThresholdNotReached Proposal declined because the participation threshold was not reached.
	ProposalErrorParticipationThresholdNotReached ProposalError = vegapb.ProposalError_PROPOSAL_ERROR_PARTICIPATION_THRESHOLD_NOT_REACHED
	// ProposalErrorInvalidAssetDetails Asset proposal invalid asset details.
	ProposalErrorInvalidAssetDetails ProposalError = vegapb.ProposalError_PROPOSAL_ERROR_INVALID_ASSET_DETAILS
	// ProposalErrorUnknownType Proposal is an unknown type.
	ProposalErrorUnknownType ProposalError = vegapb.ProposalError_PROPOSAL_ERROR_UNKNOWN_TYPE
	// ProposalErrorUnknownRiskParameterType Proposal has an unknown risk parameter type.
	ProposalErrorUnknownRiskParameterType ProposalError = vegapb.ProposalError_PROPOSAL_ERROR_UNKNOWN_RISK_PARAMETER_TYPE
	// ProposalErrorInvalidFreeform Validation failed for freeform proposal.
	ProposalErrorInvalidFreeform ProposalError = vegapb.ProposalError_PROPOSAL_ERROR_INVALID_FREEFORM
	// ProposalErrorInsufficientEquityLikeShare The party doesn't have enough equity-like share to propose an update on the market
	// targeted by the proposal.
	ProposalErrorInsufficientEquityLikeShare ProposalError = vegapb.ProposalError_PROPOSAL_ERROR_INSUFFICIENT_EQUITY_LIKE_SHARE
	// ProposalErrorInvalidMarket The market targeted by the proposal does not exist or is not eligible to modification.
	ProposalErrorInvalidMarket ProposalError = vegapb.ProposalError_PROPOSAL_ERROR_INVALID_MARKET
	// ProposalErrorTooManyMarketDecimalPlaces the market uses more decimal places than the settlement asset.
	ProposalErrorTooManyMarketDecimalPlaces ProposalError = vegapb.ProposalError_PROPOSAL_ERROR_TOO_MANY_MARKET_DECIMAL_PLACES
	// ProposalErrorTooManyPriceMonitoringTriggers the market price monitoring setting uses too many triggers.
	ProposalErrorTooManyPriceMonitoringTriggers ProposalError = vegapb.ProposalError_PROPOSAL_ERROR_TOO_MANY_PRICE_MONITORING_TRIGGERS
	// ProposalErrorERC20AddressAlreadyInUse the proposal uses a erc20 address already used by another asset.
	ProposalErrorERC20AddressAlreadyInUse ProposalError = vegapb.ProposalError_PROPOSAL_ERROR_ERC20_ADDRESS_ALREADY_IN_USE
	// ProposalErrorLinearSlippageOutOfRange linear slippage factor is negative or too large.
	ProposalErrorLinearSlippageOutOfRange ProposalError = vegapb.ProposalError_PROPOSAL_ERROR_LINEAR_SLIPPAGE_FACTOR_OUT_OF_RANGE
	// ProposalErrorSquaredSlippageOutOfRange squared slippage factor is negative or too large.
	ProposalErrorQuadraticSlippageOutOfRange ProposalError = vegapb.ProposalError_PROPOSAL_ERROR_QUADRATIC_SLIPPAGE_FACTOR_OUT_OF_RANGE
	// ProporsalErrorInvalidGovernanceTransfer governance transfer invalid.
	ProporsalErrorInvalidGovernanceTransfer ProposalError = vegapb.ProposalError_PROPOSAL_ERROR_GOVERNANCE_TRANSFER_PROPOSAL_INVALID
	// ProporsalErrorFailedGovernanceTransfer governance transfer failed.
	ProporsalErrorFailedGovernanceTransfer ProposalError = vegapb.ProposalError_PROPOSAL_ERROR_GOVERNANCE_TRANSFER_PROPOSAL_FAILED
	// ProporsalErrorFailedGovernanceTransferCancel governance transfer cancellation is invalid.
	ProporsalErrorFailedGovernanceTransferCancel ProposalError = vegapb.ProposalError_PROPOSAL_ERROR_GOVERNANCE_CANCEL_TRANSFER_PROPOSAL_INVALID
	// ProposalErrorInvalidFreeform Validation failed for spot proposal.
	ProposalErrorInvalidSpot ProposalError = vegapb.ProposalError_PROPOSAL_ERROR_INVALID_SPOT
	// ProposalErrorSpotNotEnabled is returned when spots are not enabled.
	ProposalErrorSpotNotEnabled ProposalError = vegapb.ProposalError_PROPOSAL_ERROR_SPOT_PRODUCT_DISABLED
	// ProposalErrorInvalidSuccessorMarket indicates the successor market parameters were invalid.
	ProposalErrorInvalidSuccessorMarket ProposalError = vegapb.ProposalError_PROPOSAL_ERROR_INVALID_SUCCESSOR_MARKET
	// ProposalErrorInvalidStateUpdate indicates that a market state update has failed.
	ProposalErrorInvalidStateUpdate ProposalError = vegapb.ProposalError_PROPOSAL_ERROR_INVALID_MARKET_STATE_UPDATE
	// ProposalErrorInvalidSLAParams indicates that liquidity provision SLA params are invalid.
	ProposalErrorMissingSLAParams ProposalError = vegapb.ProposalError_PROPOSAL_ERROR_MISSING_SLA_PARAMS
	// ProposalErrorMissingSLAParams indicates that mandatory SLA params for a new or update spot market is missing.
	ProposalErrorInvalidSLAParams ProposalError = vegapb.ProposalError_PROPOSAL_ERROR_INVALID_SLA_PARAMS
	// ProposalErrorInvalidPerpsProduct Market proposal market contained invalid product definition.
	ProposalErrorInvalidPerpsProduct ProposalError = vegapb.ProposalError_PROPOSAL_ERROR_INVALID_PERPETUAL_PRODUCT
	// ProposalErrorInvalidReferralProgram is returned when the referral program proposal is not valid.
	ProposalErrorInvalidReferralProgram ProposalError = vegapb.ProposalError_PROPOSAL_ERROR_INVALID_REFERRAL_PROGRAM
	// ProposalErrorInvalidVolumeDiscountProgram is returned when the volume discount program proposal is not valid.
	ProposalErrorInvalidVolumeDiscountProgram ProposalError = vegapb.ProposalError_PROPOSAL_ERROR_INVALID_VOLUME_DISCOUNT_PROGRAM
	// ProposalErrorProposalInBatchRejected is returned when one or more proposals in the batch are rejected.
	ProposalErrorProposalInBatchRejected ProposalError = vegapb.ProposalError_PROPOSAL_ERROR_PROPOSAL_IN_BATCH_REJECTED
	// ProposalErrorProposalInBatchDeclined is returned when one or more proposals in the batch are rejected.
	ProposalErrorProposalInBatchDeclined ProposalError = vegapb.ProposalError_PROPOSAL_ERROR_PROPOSAL_IN_BATCH_DECLINED
	// ProposalErrorInvalidSizeDecimalPlaces is returned in spot market when the proposed position decimal places is > base asset decimal places.
	ProposalErrorInvalidSizeDecimalPlaces = vegapb.ProposalError_PROPOSAL_ERROR_INVALID_SIZE_DECIMAL_PLACES
	// ProposalErrorInvalidVolumeRebateProgram is returned when the volume rebate program proposal is not valid.
	ProposalErrorInvalidVolumeRebateProgram ProposalError = vegapb.ProposalError_PROPOSAL_ERROR_INVALID_VOLUME_REBATE_PROGRAM
)

type ProposalState = vegapb.Proposal_State

const (
	// ProposalStateUnspecified Default value, always invalid.
	ProposalStateUnspecified ProposalState = vegapb.Proposal_STATE_UNSPECIFIED
	// ProposalStateFailed Proposal enactment has failed - even though proposal has passed, its execution could not be performed.
	ProposalStateFailed ProposalState = vegapb.Proposal_STATE_FAILED
	// ProposalStateOpen Proposal is open for voting.
	ProposalStateOpen ProposalState = vegapb.Proposal_STATE_OPEN
	// ProposalStatePassed Proposal has gained enough support to be executed.
	ProposalStatePassed ProposalState = vegapb.Proposal_STATE_PASSED
	// ProposalStateRejected Proposal wasn't accepted (proposal terms failed validation due to wrong configuration or failing to meet network requirements).
	ProposalStateRejected ProposalState = vegapb.Proposal_STATE_REJECTED
	// ProposalStateDeclined Proposal didn't get enough votes (either failing to gain required participation or majority level).
	ProposalStateDeclined ProposalState = vegapb.Proposal_STATE_DECLINED
	// ProposalStateEnacted Proposal enacted.
	ProposalStateEnacted ProposalState = vegapb.Proposal_STATE_ENACTED
	// ProposalStateWaitingForNodeVote Waiting for node validation of the proposal.
	ProposalStateWaitingForNodeVote ProposalState = vegapb.Proposal_STATE_WAITING_FOR_NODE_VOTE
)

type ProposalTermsType int

const (
	ProposalTermsTypeUpdateMarket ProposalTermsType = iota
	ProposalTermsTypeNewMarket
	ProposalTermsTypeUpdateNetworkParameter
	ProposalTermsTypeNewAsset
	ProposalTermsTypeNewFreeform
	ProposalTermsTypeUpdateAsset
	ProposalTermsTypeNewTransfer
	ProposalTermsTypeNewSpotMarket
	ProposalTermsTypeUpdateSpotMarket
	ProposalTermsTypeCancelTransfer
	ProposalTermsTypeUpdateMarketState
	ProposalTermsTypeUpdateReferralProgram
	ProposalTermsTypeUpdateVolumeDiscountProgram
	ProposalTermsTypeUpdateVolumeRebateProgram
)

type ProposalSubmission struct {
	// Proposal reference
	Reference string
	// Proposal configuration and the actual change that is meant to be executed when proposal is enacted
	Terms *ProposalTerms
	// Rationale behind the proposal change.
	Rationale *ProposalRationale
}

func (p ProposalSubmission) IntoProto() *commandspb.ProposalSubmission {
	var terms *vegapb.ProposalTerms
	if p.Terms != nil {
		terms = p.Terms.IntoProto()
	}
	return &commandspb.ProposalSubmission{
		Reference: p.Reference,
		Terms:     terms,
		Rationale: &vegapb.ProposalRationale{
			Description: p.Rationale.Description,
			Title:       p.Rationale.Title,
		},
	}
}

func ProposalSubmissionFromProposal(p *Proposal) *ProposalSubmission {
	return &ProposalSubmission{
		Reference: p.Reference,
		Terms:     p.Terms,
		Rationale: p.Rationale,
	}
}

func NewProposalSubmissionFromProto(p *commandspb.ProposalSubmission) (*ProposalSubmission, error) {
	var pterms *ProposalTerms
	if p.Terms != nil {
		var err error
		pterms, err = ProposalTermsFromProto(p.Terms)
		if err != nil {
			return nil, err
		}
	}
	return &ProposalSubmission{
		Reference: p.Reference,
		Terms:     pterms,
		Rationale: &ProposalRationale{
			Description: p.Rationale.Description,
			Title:       p.Rationale.Title,
		},
	}, nil
}

// ProposalParameters stores proposal specific parameters.
type ProposalParameters struct {
	MinClose                time.Duration
	MaxClose                time.Duration
	MinEnact                time.Duration
	MaxEnact                time.Duration
	RequiredParticipation   num.Decimal
	RequiredMajority        num.Decimal
	MinProposerBalance      *num.Uint
	MinVoterBalance         *num.Uint
	RequiredParticipationLP num.Decimal
	RequiredMajorityLP      num.Decimal
	MinEquityLikeShare      num.Decimal
}

func ProposalParametersFromProto(pp *vegapb.ProposalParameters) *ProposalParameters {
	return &ProposalParameters{
		MinClose:                time.Duration(pp.MinClose),
		MaxClose:                time.Duration(pp.MaxClose),
		MinEnact:                time.Duration(pp.MinEnact),
		MaxEnact:                time.Duration(pp.MaxEnact),
		RequiredParticipation:   num.MustDecimalFromString(pp.RequiredParticipation),
		RequiredMajority:        num.MustDecimalFromString(pp.RequiredMajority),
		MinProposerBalance:      num.MustUintFromString(pp.MinProposerBalance, 10),
		MinVoterBalance:         num.MustUintFromString(pp.MinVoterBalance, 10),
		RequiredParticipationLP: num.MustDecimalFromString(pp.RequiredParticipationLp),
		RequiredMajorityLP:      num.MustDecimalFromString(pp.RequiredMajorityLp),
		MinEquityLikeShare:      num.MustDecimalFromString(pp.MinEquityLikeShare),
	}
}

func (pp *ProposalParameters) Clone() ProposalParameters {
	copy := *pp

	copy.MinProposerBalance = pp.MinProposerBalance.Clone()
	copy.MinVoterBalance = pp.MinVoterBalance.Clone()
	return copy
}

func (pp *ProposalParameters) ToProto() *vegapb.ProposalParameters {
	return &vegapb.ProposalParameters{
		MinClose:                int64(pp.MinClose),
		MaxClose:                int64(pp.MaxClose),
		MinEnact:                int64(pp.MinEnact),
		MaxEnact:                int64(pp.MaxEnact),
		RequiredParticipation:   pp.RequiredParticipation.String(),
		RequiredMajority:        pp.RequiredMajority.String(),
		MinProposerBalance:      pp.MinProposerBalance.String(),
		MinVoterBalance:         pp.MinVoterBalance.String(),
		RequiredParticipationLp: pp.RequiredParticipationLP.String(),
		RequiredMajorityLp:      pp.RequiredMajorityLP.String(),
		MinEquityLikeShare:      pp.MinEquityLikeShare.String(),
	}
}

type BatchProposal struct {
	ID                 string
	Reference          string
	Party              string
	State              ProposalState
	Timestamp          int64
	ClosingTimestamp   int64
	Proposals          []*Proposal
	Rationale          *ProposalRationale
	Reason             ProposalError
	ErrorDetails       string
	ProposalParameters *ProposalParameters
}

func BatchProposalFromSnapshotProto(bp *vegapb.Proposal, pps []*vegapb.Proposal) *BatchProposal {
	proposals := make([]*Proposal, 0, len(pps))

	for _, pp := range pps {
		proposal, _ := ProposalFromProto(pp)
		proposals = append(proposals, proposal)
	}

	return &BatchProposal{
		ID:                 bp.Id,
		Reference:          bp.Reference,
		Party:              bp.PartyId,
		State:              bp.State,
		Timestamp:          bp.Timestamp,
		ClosingTimestamp:   bp.BatchTerms.ClosingTimestamp,
		Rationale:          ProposalRationaleFromProto(bp.Rationale),
		Reason:             *bp.Reason,
		ProposalParameters: ProposalParametersFromProto(bp.BatchTerms.ProposalParams),
		Proposals:          proposals,
	}
}

func (bp BatchProposal) ToProto() *vegapb.Proposal {
	batchTerms := &vegapb.BatchProposalTerms{
		ClosingTimestamp: bp.ClosingTimestamp,
		ProposalParams:   bp.ProposalParameters.ToProto(),
	}

	for _, proposal := range bp.Proposals {
		batchTerms.Changes = append(batchTerms.Changes, &vegapb.BatchProposalTermsChange{
			Change:             proposal.Terms.Change.oneOfBatchProto(),
			EnactmentTimestamp: proposal.Terms.EnactmentTimestamp,
		})
	}

	return &vegapb.Proposal{
		Id:           bp.ID,
		Reference:    bp.Reference,
		PartyId:      bp.Party,
		State:        bp.State,
		Timestamp:    bp.Timestamp,
		Reason:       &bp.Reason,
		ErrorDetails: &bp.ErrorDetails,
		Rationale:    bp.Rationale.ToProto(),
		BatchTerms:   batchTerms,
	}
}

// SetProposalParams set specific per proposal parameters and chooses the most aggressive ones.
func (bp *BatchProposal) SetProposalParams(params ProposalParameters) {
	if bp.ProposalParameters == nil {
		bp.ProposalParameters = &params
		bp.ProposalParameters.MaxEnact = 0
		bp.ProposalParameters.MinEnact = 0
		return
	}

	if bp.ProposalParameters.MaxClose > params.MaxClose {
		bp.ProposalParameters.MaxClose = params.MaxClose
	}

	if bp.ProposalParameters.MinClose < params.MinClose {
		bp.ProposalParameters.MinClose = params.MinClose
	}

	if bp.ProposalParameters.MinEquityLikeShare.LessThan(params.MinEquityLikeShare) {
		bp.ProposalParameters.MinEquityLikeShare = params.MinEquityLikeShare
	}

	if bp.ProposalParameters.MinProposerBalance.LT(params.MinProposerBalance) {
		bp.ProposalParameters.MinProposerBalance = params.MinProposerBalance.Clone()
	}

	if bp.ProposalParameters.MinVoterBalance.LT(params.MinVoterBalance) {
		bp.ProposalParameters.MinVoterBalance = params.MinVoterBalance.Clone()
	}

	if bp.ProposalParameters.RequiredMajority.LessThan(params.RequiredMajority) {
		bp.ProposalParameters.RequiredMajority = params.RequiredMajority
	}

	if bp.ProposalParameters.RequiredMajorityLP.LessThan(params.RequiredMajorityLP) {
		bp.ProposalParameters.RequiredMajorityLP = params.RequiredMajorityLP
	}

	if bp.ProposalParameters.RequiredParticipation.LessThan(params.RequiredParticipation) {
		bp.ProposalParameters.RequiredParticipation = params.RequiredParticipation
	}

	if bp.ProposalParameters.RequiredParticipationLP.LessThan(params.RequiredParticipationLP) {
		bp.ProposalParameters.RequiredParticipationLP = params.RequiredParticipationLP
	}
}

func (p *BatchProposal) WaitForNodeVote() {
	p.State = ProposalStateWaitingForNodeVote
	for _, v := range p.Proposals {
		v.WaitForNodeVote()
	}
}

func (p *BatchProposal) Open() {
	p.State = ProposalStateOpen
	for _, v := range p.Proposals {
		v.Open()
	}
}

func (p *BatchProposal) Reject(reason ProposalError) {
	p.State = ProposalStateRejected
	p.Reason = reason
	for _, v := range p.Proposals {
		v.Reject(reason)
	}
}

func (bp *BatchProposal) RejectWithErr(reason ProposalError, details error) {
	bp.ErrorDetails = details.Error()
	bp.State = ProposalStateRejected
	bp.Reason = reason

	for _, proposal := range bp.Proposals {
		proposal.ErrorDetails = bp.ErrorDetails
		proposal.State = bp.State
		proposal.Reason = bp.Reason
	}
}

func (bp *BatchProposal) IsRejected() bool {
	return bp.State == ProposalStateRejected
}

type Proposal struct {
	ID                      string
	BatchID                 *string
	Reference               string
	Party                   string
	State                   ProposalState
	Timestamp               int64
	Terms                   *ProposalTerms
	Rationale               *ProposalRationale
	Reason                  ProposalError
	ErrorDetails            string
	RequiredMajority        num.Decimal
	RequiredParticipation   num.Decimal
	RequiredLPMajority      num.Decimal
	RequiredLPParticipation num.Decimal
}

func (p Proposal) IsOpen() bool {
	return p.State == ProposalStateOpen
}

func (p Proposal) IsPassed() bool {
	return p.State == ProposalStatePassed
}

func (p Proposal) IsDeclined() bool {
	return p.State == ProposalStateDeclined
}

func (p Proposal) IsRejected() bool {
	return p.State == ProposalStateRejected
}

func (p Proposal) IsFailed() bool {
	return p.State == ProposalStateFailed
}

func (p Proposal) IsEnacted() bool {
	return p.State == ProposalStateEnacted
}

func (p Proposal) IsMarketUpdate() bool {
	return p.Terms.IsMarketUpdate()
}

func (p Proposal) IsMarketStateUpdate() bool {
	return p.Terms.IsMarketStateUpdate()
}

func (p Proposal) IsSpotMarketUpdate() bool {
	return p.Terms.IsSpotMarketUpdate()
}

func (p Proposal) IsReferralProgramUpdate() bool {
	return p.Terms.IsReferralProgramUpdate()
}

func (p Proposal) IsVolumeDiscountProgramUpdate() bool {
	return p.Terms.IsVolumeDiscountProgramUpdate()
}

func (p Proposal) IsVolumeRebateProgramUpdate() bool {
	return p.Terms.IsVolumeRebateProgramUpdate()
}

func (p Proposal) MarketUpdate() *UpdateMarket {
	return p.Terms.MarketUpdate()
}

func (p Proposal) UpdateMarketState() *UpdateMarketState {
	return p.Terms.UpdateMarketState()
}

func (p Proposal) SpotMarketUpdate() *UpdateSpotMarket {
	return p.Terms.SpotMarketUpdate()
}

func (p Proposal) IsNewMarket() bool {
	return p.Terms.IsNewMarket()
}

func (p Proposal) NewMarket() *NewMarket {
	return p.Terms.NewMarket()
}

func (p *Proposal) IsSuccessorMarket() bool {
	if p.Terms == nil {
		return false
	}
	return p.Terms.IsSuccessorMarket()
}

func (p *Proposal) WaitForNodeVote() {
	p.State = ProposalStateWaitingForNodeVote
}

func (p *Proposal) Open() {
	p.State = ProposalStateOpen
}

func (p *Proposal) Reject(reason ProposalError) {
	p.State = ProposalStateRejected
	p.Reason = reason
}

func (p *Proposal) Decline(reason ProposalError) {
	p.State = ProposalStateDeclined
	p.Reason = reason
}

func (p *Proposal) RejectWithErr(reason ProposalError, details error) {
	p.ErrorDetails = details.Error()
	p.State = ProposalStateRejected
	p.Reason = reason
}

func (p *Proposal) FailWithErr(reason ProposalError, details error) {
	p.ErrorDetails = details.Error()
	p.State = ProposalStateFailed
	p.Reason = reason
}

// FailUnexpectedly marks the proposal as failed. Calling this method should be
// reserved to cases where errors are the result of an internal issue, such as
// bad workflow, or conditions.
func (p *Proposal) FailUnexpectedly(details error) {
	p.State = ProposalStateFailed
	p.ErrorDetails = details.Error()
}

func (p Proposal) DeepClone() *Proposal {
	cpy := p
	if p.Terms != nil {
		cpy.Terms = p.Terms.DeepClone()
	}
	return &cpy
}

func (p Proposal) String() string {
	var batchID string
	if p.BatchID != nil {
		batchID = *p.BatchID
	}
	return fmt.Sprintf(
		"id(%s) batchId(%s) reference(%s) party(%s) state(%s) timestamp(%v) terms(%s) reason(%s) errorDetails(%s) requireMajority(%s) requiredParticiption(%s) requireLPMajority(%s) requiredLPParticiption(%s)",
		p.ID,
		batchID,
		p.Reference,
		p.Party,
		p.State.String(),
		p.Timestamp,
		p.Terms,
		p.Reason.String(),
		p.ErrorDetails,
		p.RequiredMajority.String(),
		p.RequiredParticipation.String(),
		p.RequiredLPMajority.String(),
		p.RequiredLPParticipation.String(),
	)
}

func (p Proposal) IntoProto() *vegapb.Proposal {
	var terms *vegapb.ProposalTerms
	if p.Terms != nil {
		terms = p.Terms.IntoProto()
	}

	var lpMajority *string
	if !p.RequiredLPMajority.IsZero() {
		lpMajority = toPtr(p.RequiredLPMajority.String())
	}
	var lpParticipation *string
	if !p.RequiredLPParticipation.IsZero() {
		lpParticipation = toPtr(p.RequiredLPParticipation.String())
	}

	proposal := &vegapb.Proposal{
		Id:                                     p.ID,
		BatchId:                                p.BatchID,
		Reference:                              p.Reference,
		PartyId:                                p.Party,
		State:                                  p.State,
		Timestamp:                              p.Timestamp,
		Terms:                                  terms,
		RequiredMajority:                       p.RequiredMajority.String(),
		RequiredParticipation:                  p.RequiredParticipation.String(),
		RequiredLiquidityProviderMajority:      lpMajority,
		RequiredLiquidityProviderParticipation: lpParticipation,
	}
	if p.Reason != ProposalErrorUnspecified {
		proposal.Reason = ptr.From(p.Reason)
	}
	if len(p.ErrorDetails) > 0 {
		proposal.ErrorDetails = ptr.From(p.ErrorDetails)
	}
	if p.Rationale != nil {
		proposal.Rationale = &vegapb.ProposalRationale{
			Description: p.Rationale.Description,
			Title:       p.Rationale.Title,
		}
	}

	return proposal
}

func ProposalFromProto(pp *vegapb.Proposal) (*Proposal, error) {
	terms, err := ProposalTermsFromProto(pp.Terms)
	if err != nil {
		return nil, err
	}

	// we check for all if the len == 0 just at first when reloading from
	// proposal to make sure that proposal without those in handle well.
	// TODO: this is to be removed later

	var majority num.Decimal
	if len(pp.RequiredMajority) <= 0 {
		majority = num.DecimalZero()
	} else if majority, err = num.DecimalFromString(pp.RequiredMajority); err != nil {
		return nil, err
	}

	var participation num.Decimal
	if len(pp.RequiredParticipation) <= 0 {
		participation = num.DecimalZero()
	} else if participation, err = num.DecimalFromString(pp.RequiredParticipation); err != nil {
		return nil, err
	}

	lpMajority := num.DecimalZero()
	if pp.RequiredLiquidityProviderMajority != nil && len(*pp.RequiredLiquidityProviderMajority) > 0 {
		if lpMajority, err = num.DecimalFromString(*pp.RequiredLiquidityProviderMajority); err != nil {
			return nil, err
		}
	}
	lpParticipation := num.DecimalZero()
	if pp.RequiredLiquidityProviderParticipation != nil && len(*pp.RequiredLiquidityProviderParticipation) > 0 {
		if lpParticipation, err = num.DecimalFromString(*pp.RequiredLiquidityProviderParticipation); err != nil {
			return nil, err
		}
	}
	reason := ProposalErrorUnspecified
	if pp.Reason != nil {
		reason = *pp.Reason
	}
	errDetails := ""
	if pp.ErrorDetails != nil {
		errDetails = *pp.ErrorDetails
	}

	return &Proposal{
		ID:                      pp.Id,
		BatchID:                 pp.BatchId,
		Reference:               pp.Reference,
		Party:                   pp.PartyId,
		State:                   pp.State,
		Timestamp:               pp.Timestamp,
		Terms:                   terms,
		Reason:                  reason,
		Rationale:               ProposalRationaleFromProto(pp.Rationale),
		ErrorDetails:            errDetails,
		RequiredMajority:        majority,
		RequiredParticipation:   participation,
		RequiredLPMajority:      lpMajority,
		RequiredLPParticipation: lpParticipation,
	}, nil
}

type ProposalRationale struct {
	Description string
	Title       string
}

func (pr ProposalRationale) ToProto() *vegapb.ProposalRationale {
	return &vegapb.ProposalRationale{
		Description: pr.Description,
		Title:       pr.Title,
	}
}

func ProposalRationaleFromProto(p *vegapb.ProposalRationale) *ProposalRationale {
	if p == nil {
		return nil
	}
	return &ProposalRationale{
		Description: p.Description,
		Title:       p.Title,
	}
}
