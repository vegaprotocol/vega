// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
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

	vegapb "code.vegaprotocol.io/protos/vega"
	commandspb "code.vegaprotocol.io/protos/vega/commands/v1"
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
	// ProposalErrorMarketMissingLiquidityCommitment Market proposal is missing a liquidity commitment.
	ProposalErrorMarketMissingLiquidityCommitment ProposalError = vegapb.ProposalError_PROPOSAL_ERROR_MARKET_MISSING_LIQUIDITY_COMMITMENT
	// ProposalErrorCouldNotInstantiateMarket Market proposal market could not be instantiated during execution.
	ProposalErrorCouldNotInstantiateMarket ProposalError = vegapb.ProposalError_PROPOSAL_ERROR_COULD_NOT_INSTANTIATE_MARKET
	// ProposalErrorInvalidFutureProduct Market proposal market contained invalid product definition.
	ProposalErrorInvalidFutureProduct ProposalError = vegapb.ProposalError_PROPOSAL_ERROR_INVALID_FUTURE_PRODUCT
	// ProposalErrorMissingCommitmentAmount Market proposal is missing commitment amount.
	ProposalErrorMissingCommitmentAmount ProposalError = vegapb.ProposalError_PROPOSAL_ERROR_MISSING_COMMITMENT_AMOUNT
	// ProposalErrorInvalidFeeAmount Market proposal have invalid fee.
	ProposalErrorInvalidFeeAmount ProposalError = vegapb.ProposalError_PROPOSAL_ERROR_INVALID_FEE_AMOUNT
	// ProposalErrorInvalidShape Market proposal have invalid shape.
	ProposalErrorInvalidShape ProposalError = vegapb.ProposalError_PROPOSAL_ERROR_INVALID_SHAPE
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
			Hash:        p.Rationale.Hash,
			Url:         p.Rationale.URL,
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
			Hash:        p.Rationale.Hash,
			URL:         p.Rationale.Url,
		},
	}, nil
}

type Proposal struct {
	ID           string
	Reference    string
	Party        string
	State        ProposalState
	Timestamp    int64
	Terms        *ProposalTerms
	Rationale    *ProposalRationale
	Reason       ProposalError
	ErrorDetails string
}

func (p *Proposal) IsMarketUpdate() bool {
	switch p.Terms.Change.(type) {
	case *ProposalTermsUpdateMarket:
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
		"id(%s) reference(%s) party(%s) state(%s) timestamp(%v) terms(%s) reason(%s) errorDetails(%s)",
		p.ID,
		p.Reference,
		p.Party,
		p.State.String(),
		p.Timestamp,
		reflectPointerToString(p.Terms),
		p.Reason.String(),
		p.ErrorDetails,
	)
}

func (p Proposal) IntoProto() *vegapb.Proposal {
	var terms *vegapb.ProposalTerms
	if p.Terms != nil {
		terms = p.Terms.IntoProto()
	}
	proposal := &vegapb.Proposal{
		Id:           p.ID,
		Reference:    p.Reference,
		PartyId:      p.Party,
		State:        p.State,
		Timestamp:    p.Timestamp,
		Terms:        terms,
		Reason:       p.Reason,
		ErrorDetails: p.ErrorDetails,
	}

	if p.Rationale != nil {
		proposal.Rationale = &vegapb.ProposalRationale{
			Description: p.Rationale.Description,
			Hash:        p.Rationale.Hash,
			Url:         p.Rationale.URL,
		}
	}

	return proposal
}

func ProposalFromProto(pp *vegapb.Proposal) (*Proposal, error) {
	terms, err := ProposalTermsFromProto(pp.Terms)
	if err != nil {
		return nil, err
	}

	return &Proposal{
		ID:           pp.Id,
		Reference:    pp.Reference,
		Party:        pp.PartyId,
		State:        pp.State,
		Timestamp:    pp.Timestamp,
		Terms:        terms,
		Reason:       pp.Reason,
		Rationale:    ProposalRationaleFromProto(pp.Rationale),
		ErrorDetails: pp.ErrorDetails,
	}, nil
}

type ProposalRationale struct {
	Description string
	Hash        string
	URL         string
}

func ProposalRationaleFromProto(p *vegapb.ProposalRationale) *ProposalRationale {
	if p == nil {
		return nil
	}
	return &ProposalRationale{
		Description: p.Description,
		Hash:        p.Hash,
		URL:         p.Url,
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
	case *vegapb.ProposalTerms_NewFreeform:
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
		reflectPointerToString(p.Change),
	)
}

func (p *ProposalTerms) GetNewAsset() *NewAsset {
	switch c := p.Change.(type) {
	case *ProposalTermsNewAsset:
		return c.NewAsset
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
			change = UpdateMarketFromProto(ch)
		case *vegapb.ProposalTerms_UpdateNetworkParameter:
			change = NewUpdateNetworkParameterFromProto(ch)
		case *vegapb.ProposalTerms_NewAsset:
			change, err = NewNewAssetFromProto(ch)
		case *vegapb.ProposalTerms_NewFreeform:
			change = NewNewFreeformFromProto(ch)
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
