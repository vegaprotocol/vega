// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package types

import (
	"fmt"

	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/ptr"
	"code.vegaprotocol.io/vega/libs/stringer"
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
	ProposalErrorInvalidSLAParams ProposalError = vegapb.ProposalError_PROPOSAL_ERROR_MISSING_SLA_PARAMS
	// ProposalErrorMissingSLAParams indicates that mandatory SLA params for a new or update spot market is missing.
	ProposalErrorMissingSLAParams ProposalError = vegapb.ProposalError_PROPOSAL_ERROR_INVALID_SLA_PARAMS
	// ProposalErrorInvalidPerpsProduct Market proposal market contained invalid product definition.
	ProposalErrorInvalidPerpsProduct ProposalError = vegapb.ProposalError_PROPOSAL_ERROR_INVALID_PERPS_PRODUCT
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

type Proposal struct {
	ID                      string
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

func (p *Proposal) IsMarketStateUpdate() bool {
	switch p.Terms.Change.(type) {
	case *ProposalTermsUpdateMarketState:
		return true
	default:
		return false
	}
}

func (p *Proposal) IsMarketUpdate() bool {
	switch p.Terms.Change.(type) {
	case *ProposalTermsUpdateMarket:
		return true
	default:
		return false
	}
}

func (p *Proposal) IsSpotMarketUpdate() bool {
	switch p.Terms.Change.(type) {
	case *ProposalTermsUpdateSpotMarket:
		return true
	default:
		return false
	}
}

func (p *Proposal) MarketUpdate() *UpdateMarket {
	switch terms := p.Terms.Change.(type) {
	case *ProposalTermsUpdateMarket:
		return terms.UpdateMarket
	default:
		return nil
	}
}

func (p *Proposal) UpdateMarketState() *UpdateMarketState {
	switch terms := p.Terms.Change.(type) {
	case *ProposalTermsUpdateMarketState:
		return terms.UpdateMarketState
	default:
		return nil
	}
}

func (p *Proposal) SpotMarketUpdate() *UpdateSpotMarket {
	switch terms := p.Terms.Change.(type) {
	case *ProposalTermsUpdateSpotMarket:
		return terms.UpdateSpotMarket
	default:
		return nil
	}
}

func (p *Proposal) IsNewMarket() bool {
	return p.Terms.Change.GetTermType() == ProposalTermsTypeNewMarket
}

func (p *Proposal) NewMarket() *NewMarket {
	switch terms := p.Terms.Change.(type) {
	case *ProposalTermsNewMarket:
		return terms.NewMarket
	default:
		return nil
	}
}

func (p *Proposal) IsSuccessorMarket() bool {
	if p.Terms == nil || p.Terms.Change == nil {
		return false
	}
	if nm := p.NewMarket(); nm != nil {
		return nm.Changes.Successor != nil
	}
	return false
}

func (p *Proposal) WaitForNodeVote() {
	p.State = ProposalStateWaitingForNodeVote
}

func (p *Proposal) Reject(reason ProposalError) {
	p.State = ProposalStateRejected
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
	return fmt.Sprintf(
		"id(%s) reference(%s) party(%s) state(%s) timestamp(%v) terms(%s) reason(%s) errorDetails(%s) requireMajority(%s) requiredParticiption(%s) requireLPMajority(%s) requiredLPParticiption(%s)",
		p.ID,
		p.Reference,
		p.Party,
		p.State.String(),
		p.Timestamp,
		stringer.ReflectPointerToString(p.Terms),
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

func ProposalRationaleFromProto(p *vegapb.ProposalRationale) *ProposalRationale {
	if p == nil {
		return nil
	}
	return &ProposalRationale{
		Description: p.Description,
		Title:       p.Title,
	}
}

type ProposalTerms struct {
	ClosingTimestamp    int64
	EnactmentTimestamp  int64
	ValidationTimestamp int64
	// *ProposalTermsUpdateMarket
	// *ProposalTermsNewMarket
	// *ProposalTermsUpdateNetworkParameter
	// *ProposalTermsNewAsset
	Change proposalTerm
}

func (p ProposalTerms) IntoProto() *vegapb.ProposalTerms {
	change := p.Change.oneOfProto()
	r := &vegapb.ProposalTerms{
		ClosingTimestamp:    p.ClosingTimestamp,
		EnactmentTimestamp:  p.EnactmentTimestamp,
		ValidationTimestamp: p.ValidationTimestamp,
	}

	switch ch := change.(type) {
	case *vegapb.ProposalTerms_NewMarket:
		r.Change = ch
	case *vegapb.ProposalTerms_UpdateMarket:
		r.Change = ch
	case *vegapb.ProposalTerms_UpdateNetworkParameter:
		r.Change = ch
	case *vegapb.ProposalTerms_NewAsset:
		r.Change = ch
	case *vegapb.ProposalTerms_UpdateAsset:
		r.Change = ch
	case *vegapb.ProposalTerms_NewFreeform:
		r.Change = ch
	case *vegapb.ProposalTerms_NewTransfer:
		r.Change = ch
	case *vegapb.ProposalTerms_CancelTransfer:
		r.Change = ch
	case *vegapb.ProposalTerms_NewSpotMarket:
		r.Change = ch
	case *vegapb.ProposalTerms_UpdateSpotMarket:
		r.Change = ch
	case *vegapb.ProposalTerms_UpdateMarketState:
		r.Change = ch
	}
	return r
}

func (p ProposalTerms) DeepClone() *ProposalTerms {
	cpy := p
	cpy.Change = p.Change.DeepClone()
	return &cpy
}

func (p ProposalTerms) String() string {
	return fmt.Sprintf(
		"validationTs(%v) closingTs(%v) enactmentTs(%v) change(%s)",
		p.ValidationTimestamp,
		p.ClosingTimestamp,
		p.EnactmentTimestamp,
		stringer.ReflectPointerToString(p.Change),
	)
}

func (p *ProposalTerms) GetNewTransfer() *NewTransfer {
	switch c := p.Change.(type) {
	case *ProposalTermsNewTransfer:
		return c.NewTransfer
	default:
		return nil
	}
}

func (p *ProposalTerms) GetCancelTransfer() *CancelTransfer {
	switch c := p.Change.(type) {
	case *ProposalTermsCancelTransfer:
		return c.CancelTransfer
	default:
		return nil
	}
}

func (p *ProposalTerms) GetMarketStateUpdate() *UpdateMarketState {
	switch c := p.Change.(type) {
	case *ProposalTermsUpdateMarketState:
		return c.UpdateMarketState
	default:
		return nil
	}
}

func (p *ProposalTerms) GetNewAsset() *NewAsset {
	switch c := p.Change.(type) {
	case *ProposalTermsNewAsset:
		return c.NewAsset
	default:
		return nil
	}
}

func (p *ProposalTerms) GetUpdateAsset() *UpdateAsset {
	switch c := p.Change.(type) {
	case *ProposalTermsUpdateAsset:
		return c.UpdateAsset
	default:
		return nil
	}
}

func (p *ProposalTerms) GetNewMarket() *NewMarket {
	switch c := p.Change.(type) {
	case *ProposalTermsNewMarket:
		return c.NewMarket
	default:
		return nil
	}
}

func (p *ProposalTerms) GetUpdateMarket() *UpdateMarket {
	switch c := p.Change.(type) {
	case *ProposalTermsUpdateMarket:
		return c.UpdateMarket
	default:
		return nil
	}
}

func (p *ProposalTerms) GetNewSpotMarket() *NewSpotMarket {
	switch c := p.Change.(type) {
	case *ProposalTermsNewSpotMarket:
		return c.NewSpotMarket
	default:
		return nil
	}
}

func (p *ProposalTerms) GetUpdateSpotMarket() *UpdateSpotMarket {
	switch c := p.Change.(type) {
	case *ProposalTermsUpdateSpotMarket:
		return c.UpdateSpotMarket
	default:
		return nil
	}
}

func (p *ProposalTerms) GetUpdateNetworkParameter() *UpdateNetworkParameter {
	switch c := p.Change.(type) {
	case *ProposalTermsUpdateNetworkParameter:
		return c.UpdateNetworkParameter
	default:
		return nil
	}
}

func (p *ProposalTerms) GetNewFreeform() *NewFreeform {
	switch c := p.Change.(type) {
	case *ProposalTermsNewFreeform:
		return c.NewFreeform
	default:
		return nil
	}
}

func ProposalTermsFromProto(p *vegapb.ProposalTerms) (*ProposalTerms, error) {
	var (
		change proposalTerm
		err    error
	)
	if p.Change != nil {
		switch ch := p.Change.(type) {
		case *vegapb.ProposalTerms_NewMarket:
			change, err = NewNewMarketFromProto(ch)
		case *vegapb.ProposalTerms_UpdateMarket:
			change, err = UpdateMarketFromProto(ch)
		case *vegapb.ProposalTerms_UpdateNetworkParameter:
			change = NewUpdateNetworkParameterFromProto(ch)
		case *vegapb.ProposalTerms_NewAsset:
			change, err = NewNewAssetFromProto(ch)
		case *vegapb.ProposalTerms_UpdateAsset:
			change, err = NewUpdateAssetFromProto(ch)
		case *vegapb.ProposalTerms_NewFreeform:
			change = NewNewFreeformFromProto(ch)
		case *vegapb.ProposalTerms_NewSpotMarket:
			change, err = NewNewSpotMarketFromProto(ch)
		case *vegapb.ProposalTerms_UpdateSpotMarket:
			change, err = UpdateSpotMarketFromProto(ch)
		case *vegapb.ProposalTerms_NewTransfer:
			change, err = NewNewTransferFromProto(ch)
		case *vegapb.ProposalTerms_CancelTransfer:
			change, err = NewCancelGovernanceTransferFromProto(ch)
		case *vegapb.ProposalTerms_UpdateMarketState:
			change, err = NewTerminateMarketFromProto(ch)
		}
	}
	if err != nil {
		return nil, err
	}

	return &ProposalTerms{
		ClosingTimestamp:    p.ClosingTimestamp,
		EnactmentTimestamp:  p.EnactmentTimestamp,
		ValidationTimestamp: p.ValidationTimestamp,
		Change:              change,
	}, nil
}

type proposalTerm interface {
	isPTerm()
	oneOfProto() interface{} // calls IntoProto
	DeepClone() proposalTerm
	GetTermType() ProposalTermsType
	String() string
}
