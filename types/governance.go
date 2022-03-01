package types

import (
	"errors"

	vegapb "code.vegaprotocol.io/protos/vega"
	commandspb "code.vegaprotocol.io/protos/vega/commands/v1"
	oraclespb "code.vegaprotocol.io/protos/vega/oracles/v1"

	"code.vegaprotocol.io/vega/types/num"
)

type GovernanceData = vegapb.GovernanceData

type VoteValue = vegapb.Vote_Value

const (
	// VoteValueUnspecified Default value, always invalid.
	VoteValueUnspecified VoteValue = vegapb.Vote_VALUE_UNSPECIFIED
	// VoteValueNo represents a vote against the proposal.
	VoteValueNo VoteValue = vegapb.Vote_VALUE_NO
	// VoteValueYes represents a vote in favour of the proposal.
	VoteValueYes VoteValue = vegapb.Vote_VALUE_YES
)

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
)

// Vote represents a governance vote casted by a party for a given proposal.
type Vote struct {
	// PartyID is the party that casted the vote.
	PartyID string
	// ProposalID is the proposal identifier concerned by the vote.
	ProposalID string
	// Value is the actual position of the vote: yes or no.
	Value VoteValue
	// Timestamp is the date and time (in nanoseconds) at which the vote has
	// been casted.
	Timestamp int64
	// TotalGovernanceTokenBalance is the total number of tokens hold by the
	// party that casted the vote.
	TotalGovernanceTokenBalance *num.Uint
	// TotalGovernanceTokenWeight is the weight of the vote compared to the
	// total number of governance token.
	TotalGovernanceTokenWeight num.Decimal
}

type VoteSubmission struct {
	// The ID of the proposal to vote for.
	ProposalID string
	// The actual value of the vote
	Value VoteValue
}

func NewVoteSubmissionFromProto(p *commandspb.VoteSubmission) *VoteSubmission {
	return &VoteSubmission{
		ProposalID: p.ProposalId,
		Value:      p.Value,
	}
}

func (v VoteSubmission) IntoProto() *commandspb.VoteSubmission {
	return &commandspb.VoteSubmission{
		ProposalId: v.ProposalID,
		Value:      v.Value,
	}
}

func (v VoteSubmission) String() string {
	return v.IntoProto().String()
}

type ProposalSubmission struct {
	// Proposal reference
	Reference string
	// Proposal configuration and the actual change that is meant to be executed when proposal is enacted
	Terms *ProposalTerms
}

func ProposalSubmissionFromProposal(p *Proposal) *ProposalSubmission {
	return &ProposalSubmission{
		Reference: p.Reference,
		Terms:     p.Terms,
	}
}

func NewProposalSubmissionFromProto(p *commandspb.ProposalSubmission) *ProposalSubmission {
	var pterms *ProposalTerms
	if p.Terms != nil {
		pterms = ProposalTermsFromProto(p.Terms)
	}
	return &ProposalSubmission{
		Reference: p.Reference,
		Terms:     pterms,
	}
}

func (p ProposalSubmission) IntoProto() *commandspb.ProposalSubmission {
	var terms *vegapb.ProposalTerms
	if p.Terms != nil {
		terms = p.Terms.IntoProto()
	}
	return &commandspb.ProposalSubmission{
		Reference: p.Reference,
		Terms:     terms,
	}
}

type Proposal struct {
	ID           string
	Reference    string
	Party        string
	State        ProposalState
	Timestamp    int64
	Terms        *ProposalTerms
	Reason       ProposalError
	ErrorDetails string
}

func (p Proposal) DeepClone() *Proposal {
	cpy := p
	if p.Terms != nil {
		cpy.Terms = p.Terms.DeepClone()
	}
	return &cpy
}

func ProposalFromProto(pp *vegapb.Proposal) *Proposal {
	return &Proposal{
		ID:           pp.Id,
		Reference:    pp.Reference,
		Party:        pp.PartyId,
		State:        pp.State,
		Timestamp:    pp.Timestamp,
		Terms:        ProposalTermsFromProto(pp.Terms),
		Reason:       pp.Reason,
		ErrorDetails: pp.ErrorDetails,
	}
}

func (p Proposal) IntoProto() *vegapb.Proposal {
	var terms *vegapb.ProposalTerms
	if p.Terms != nil {
		terms = p.Terms.IntoProto()
	}
	return &vegapb.Proposal{
		Id:           p.ID,
		Reference:    p.Reference,
		PartyId:      p.Party,
		State:        p.State,
		Timestamp:    p.Timestamp,
		Terms:        terms,
		Reason:       p.Reason,
		ErrorDetails: p.ErrorDetails,
	}
}

func (v Vote) IntoProto() *vegapb.Vote {
	return &vegapb.Vote{
		PartyId:                     v.PartyID,
		Value:                       v.Value,
		ProposalId:                  v.ProposalID,
		Timestamp:                   v.Timestamp,
		TotalGovernanceTokenBalance: num.UintToString(v.TotalGovernanceTokenBalance),
		TotalGovernanceTokenWeight:  v.TotalGovernanceTokenWeight.String(),
	}
}

func VoteFromProto(v *vegapb.Vote) (*Vote, error) {
	ret := Vote{
		PartyID:    v.PartyId,
		Value:      v.Value,
		ProposalID: v.ProposalId,
		Timestamp:  v.Timestamp,
	}
	if len(v.TotalGovernanceTokenBalance) > 0 {
		ret.TotalGovernanceTokenBalance, _ = num.UintFromString(v.TotalGovernanceTokenBalance, 10)
	}
	if len(v.TotalGovernanceTokenWeight) > 0 {
		w, err := num.DecimalFromString(v.TotalGovernanceTokenWeight)
		if err != nil {
			return nil, err
		}
		ret.TotalGovernanceTokenWeight = w
	}
	return &ret, nil
}

type NewMarketCommitment struct {
	CommitmentAmount *num.Uint
	Fee              num.Decimal
	Sells            []*LiquidityOrder
	Buys             []*LiquidityOrder
	Reference        string
}

func NewMarketCommitmentFromProto(p *vegapb.NewMarketCommitment) (*NewMarketCommitment, error) {
	fee, err := num.DecimalFromString(p.Fee)
	if err != nil {
		return nil, err
	}
	commitmentAmount, overflowed := num.UintFromString(p.CommitmentAmount, 10)
	if overflowed {
		return nil, errors.New("invalid commitment amount")
	}

	l := NewMarketCommitment{
		CommitmentAmount: commitmentAmount,
		Fee:              fee,
		Sells:            make([]*LiquidityOrder, 0, len(p.Sells)),
		Buys:             make([]*LiquidityOrder, 0, len(p.Buys)),
		Reference:        p.Reference,
	}

	for _, sell := range p.Sells {
		order, err := LiquidityOrderFromProto(sell)
		if err != nil {
			return nil, err
		}

		l.Sells = append(l.Sells, order)
	}

	for _, buy := range p.Buys {
		order, err := LiquidityOrderFromProto(buy)
		if err != nil {
			return nil, err
		}

		l.Buys = append(l.Buys, order)
	}

	return &l, nil
}

func (n NewMarketCommitment) DeepClone() *NewMarketCommitment {
	cpy := &NewMarketCommitment{
		Fee:       n.Fee,
		Sells:     make([]*LiquidityOrder, 0, len(n.Sells)),
		Buys:      make([]*LiquidityOrder, 0, len(n.Buys)),
		Reference: n.Reference,
	}
	if n.CommitmentAmount != nil {
		cpy.CommitmentAmount = n.CommitmentAmount.Clone()
	} else {
		cpy.CommitmentAmount = num.Zero()
	}
	for _, s := range n.Sells {
		cpy.Sells = append(cpy.Sells, s.DeepClone())
	}
	for _, b := range n.Buys {
		cpy.Buys = append(cpy.Buys, b.DeepClone())
	}
	return cpy
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

type NewMarket struct {
	Changes             *NewMarketConfiguration
	LiquidityCommitment *NewMarketCommitment
}

type NewMarketConfiguration struct {
	Instrument                    *InstrumentConfiguration
	DecimalPlaces                 uint64
	Metadata                      []string
	PriceMonitoringParameters     *PriceMonitoringParameters
	LiquidityMonitoringParameters *LiquidityMonitoringParameters
	RiskParameters                riskParams
	TradingMode                   tradingMode
	// New market risk model parameters
	//
	// Types that are valid to be assigned to RiskParameters:
	//	*NewMarketConfigurationSimple
	//	*NewMarketConfigurationLogNormal
	// RiskParameters isNewMarketConfiguration_RiskParameters
	// Trading mode for the new market
	//
	// Types that are valid to be assigned to TradingMode:
	//	*NewMarketConfiguration_Continuous
	//	*NewMarketConfiguration_Discrete
	// TradingMode          isNewMarketConfiguration_TradingMode `protobuf_oneof:"trading_mode"`
}

type riskParams interface {
	isNMCRP()
	rpIntoProto() interface{}
	DeepClone() riskParams
}

type tradingMode interface {
	isTradingMode()
	tmIntoProto() interface{}
	DeepClone() tradingMode
}

type ProposalTermsNewMarket struct {
	NewMarket *NewMarket
}

type UpdateMarket = vegapb.UpdateMarket

type ProposalTermsUpdateMarket struct {
	UpdateMarket *UpdateMarket
}

type UpdateNetworkParameter struct {
	Changes *NetworkParameter
}

type ProposalTermsUpdateNetworkParameter struct {
	UpdateNetworkParameter *UpdateNetworkParameter
}

type NewAsset struct {
	Changes *AssetDetails
}

type ProposalTermsNewAsset struct {
	NewAsset *NewAsset
}

type NewFreeformDetails struct {
	URL         string
	Description string
	Hash        string
}

type NewFreeform struct {
	Changes *NewFreeformDetails
}

type ProposalTerms_NewFreeform struct {
	NewFreeform *NewFreeform
}

type proposalTerm interface {
	isPTerm()
	oneOfProto() interface{} // calls IntoProto
	DeepClone() proposalTerm
	GetTermType() ProposalTermsType
}

func (n *NewAsset) GetChanges() *AssetDetails {
	if n != nil {
		return n.Changes
	}
	return nil
}

func (n NewMarket) IntoProto() *vegapb.NewMarket {
	var changes *vegapb.NewMarketConfiguration
	if n.Changes != nil {
		changes = n.Changes.IntoProto()
	}
	var commitment *vegapb.NewMarketCommitment
	if n.LiquidityCommitment != nil {
		commitment = n.LiquidityCommitment.IntoProto()
	}
	return &vegapb.NewMarket{
		Changes:             changes,
		LiquidityCommitment: commitment,
	}
}

func (n NewMarket) DeepClone() *NewMarket {
	cpy := NewMarket{}
	if n.LiquidityCommitment != nil {
		cpy.LiquidityCommitment = n.LiquidityCommitment.DeepClone()
	}
	if n.Changes != nil {
		cpy.Changes = n.Changes.DeepClone()
	}
	return &cpy
}

func (n NewMarketConfiguration) IntoProto() *vegapb.NewMarketConfiguration {
	riskParams := n.RiskParameters.rpIntoProto()
	md := make([]string, 0, len(n.Metadata))
	md = append(md, n.Metadata...)

	var instrument *vegapb.InstrumentConfiguration
	if n.Instrument != nil {
		instrument = n.Instrument.IntoProto()
	}
	var priceMonitoring *vegapb.PriceMonitoringParameters
	if n.PriceMonitoringParameters != nil {
		priceMonitoring = n.PriceMonitoringParameters.IntoProto()
	}
	var liquidityMonitoring *vegapb.LiquidityMonitoringParameters
	if n.LiquidityMonitoringParameters != nil {
		liquidityMonitoring = n.LiquidityMonitoringParameters.IntoProto()
	}

	r := &vegapb.NewMarketConfiguration{
		Instrument:                    instrument,
		DecimalPlaces:                 n.DecimalPlaces,
		Metadata:                      md,
		PriceMonitoringParameters:     priceMonitoring,
		LiquidityMonitoringParameters: liquidityMonitoring,
	}
	switch rp := riskParams.(type) {
	case *vegapb.NewMarketConfiguration_Simple:
		r.RiskParameters = rp
	case *vegapb.NewMarketConfiguration_LogNormal:
		r.RiskParameters = rp
	}
	return r
}

func (n NewMarketConfiguration) DeepClone() *NewMarketConfiguration {
	cpy := &NewMarketConfiguration{
		DecimalPlaces: n.DecimalPlaces,
		Metadata:      make([]string, len(n.Metadata)),
	}
	cpy.Metadata = append(cpy.Metadata, n.Metadata...)
	if n.Instrument != nil {
		cpy.Instrument = n.Instrument.DeepClone()
	}
	if n.PriceMonitoringParameters != nil {
		cpy.PriceMonitoringParameters = n.PriceMonitoringParameters.DeepClone()
	}
	if n.LiquidityMonitoringParameters != nil {
		cpy.LiquidityMonitoringParameters = n.LiquidityMonitoringParameters.DeepClone()
	}
	if n.RiskParameters != nil {
		cpy.RiskParameters = n.RiskParameters.DeepClone()
	}
	if n.TradingMode != nil {
		cpy.TradingMode = n.TradingMode.DeepClone()
	}
	return cpy
}

func NewMarketConfigurationFromProto(p *vegapb.NewMarketConfiguration) *NewMarketConfiguration {
	md := make([]string, 0, len(p.Metadata))
	md = append(md, p.Metadata...)

	var instrument *InstrumentConfiguration
	if p.Instrument != nil {
		instrument = InstrumentConfigurationFromProto(p.Instrument)
	}

	var priceMonitoring *PriceMonitoringParameters
	if p.PriceMonitoringParameters != nil {
		priceMonitoring = PriceMonitoringParametersFromProto(p.PriceMonitoringParameters)
	}
	var liquidityMonitoring *LiquidityMonitoringParameters
	if p.LiquidityMonitoringParameters != nil {
		liquidityMonitoring = LiquidityMonitoringParametersFromProto(p.LiquidityMonitoringParameters)
	}

	r := &NewMarketConfiguration{
		Instrument:                    instrument,
		DecimalPlaces:                 p.DecimalPlaces,
		Metadata:                      md,
		PriceMonitoringParameters:     priceMonitoring,
		LiquidityMonitoringParameters: liquidityMonitoring,
	}
	if p.RiskParameters != nil {
		switch rp := p.RiskParameters.(type) {
		case *vegapb.NewMarketConfiguration_Simple:
			r.RiskParameters = NewMarketConfigurationSimpleFromProto(rp)
		case *vegapb.NewMarketConfiguration_LogNormal:
			r.RiskParameters = NewMarketConfigurationLogNormalFromProto(rp)
		}
	}
	return r
}

func ProposalTermsFromProto(p *vegapb.ProposalTerms) *ProposalTerms {
	var change proposalTerm
	if p.Change != nil {
		switch ch := p.Change.(type) {
		case *vegapb.ProposalTerms_NewMarket:
			change = NewNewMarketFromProto(ch)
		case *vegapb.ProposalTerms_UpdateMarket:
			change = NewUpdateMarketFromProto(ch)
		case *vegapb.ProposalTerms_UpdateNetworkParameter:
			change = NewUpdateNetworkParameterFromProto(ch)
		case *vegapb.ProposalTerms_NewAsset:
			change = NewNewAssetFromProto(ch)
		case *vegapb.ProposalTerms_NewFreeform:
			change = NewNewFreeformFromProto(ch)
		}
	}

	return &ProposalTerms{
		ClosingTimestamp:    p.ClosingTimestamp,
		EnactmentTimestamp:  p.EnactmentTimestamp,
		ValidationTimestamp: p.ValidationTimestamp,
		Change:              change,
	}
}

func NewNewMarketFromProto(p *vegapb.ProposalTerms_NewMarket) *ProposalTermsNewMarket {
	var newMarket *NewMarket
	if p.NewMarket != nil {
		newMarket = &NewMarket{}

		if p.NewMarket.Changes != nil {
			newMarket.Changes = NewMarketConfigurationFromProto(p.NewMarket.Changes)
		}
		if p.NewMarket.LiquidityCommitment != nil {
			newMarket.LiquidityCommitment, _ = NewMarketCommitmentFromProto(p.NewMarket.LiquidityCommitment)
		}
	}

	return &ProposalTermsNewMarket{
		NewMarket: newMarket,
	}
}

func NewUpdateMarketFromProto(p *vegapb.ProposalTerms_UpdateMarket) *ProposalTermsUpdateMarket {
	panic("unimplemented")
}

func NewUpdateNetworkParameterFromProto(
	p *vegapb.ProposalTerms_UpdateNetworkParameter,
) *ProposalTermsUpdateNetworkParameter {
	var updateNP *UpdateNetworkParameter
	if p.UpdateNetworkParameter != nil {
		updateNP = &UpdateNetworkParameter{}

		if p.UpdateNetworkParameter.Changes != nil {
			updateNP.Changes = NetworkParameterFromProto(p.UpdateNetworkParameter.Changes)
		}
	}

	return &ProposalTermsUpdateNetworkParameter{
		UpdateNetworkParameter: updateNP,
	}
}

func NewNewAssetFromProto(p *vegapb.ProposalTerms_NewAsset) *ProposalTermsNewAsset {
	var newAsset *NewAsset
	if p.NewAsset != nil {
		newAsset = &NewAsset{}

		if p.NewAsset.Changes != nil {
			newAsset.Changes = AssetDetailsFromProto(p.NewAsset.Changes)
		}
	}

	return &ProposalTermsNewAsset{
		NewAsset: newAsset,
	}
}

func NewNewFreeformFromProto(p *vegapb.ProposalTerms_NewFreeform) *ProposalTerms_NewFreeform {
	var newFreeform *NewFreeform
	if p.NewFreeform != nil && p.NewFreeform.Changes != nil {
		newFreeform = &NewFreeform{
			Changes: &NewFreeformDetails{
				URL:         p.NewFreeform.Changes.Url,
				Description: p.NewFreeform.Changes.Description,
				Hash:        p.NewFreeform.Changes.Hash,
			},
		}
	}

	return &ProposalTerms_NewFreeform{
		NewFreeform: newFreeform,
	}
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
	return p.IntoProto().String()
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
	case *ProposalTerms_NewFreeform:
		return c.NewFreeform
	default:
		return nil
	}
}

func (a ProposalTermsNewMarket) IntoProto() *vegapb.ProposalTerms_NewMarket {
	return &vegapb.ProposalTerms_NewMarket{
		NewMarket: a.NewMarket.IntoProto(),
	}
}

func (a ProposalTermsNewMarket) isPTerm() {}
func (a ProposalTermsNewMarket) oneOfProto() interface{} {
	return a.IntoProto()
}

func (a ProposalTermsNewMarket) GetTermType() ProposalTermsType {
	return ProposalTermsTypeNewMarket
}

func (a ProposalTermsNewMarket) DeepClone() proposalTerm {
	if a.NewMarket == nil {
		return &ProposalTermsNewMarket{}
	}
	return &ProposalTermsNewMarket{
		NewMarket: a.NewMarket.DeepClone(),
	}
}

func (a ProposalTermsUpdateMarket) IntoProto() *vegapb.ProposalTerms_UpdateMarket {
	return &vegapb.ProposalTerms_UpdateMarket{
		UpdateMarket: a.UpdateMarket,
	}
}

func (a ProposalTermsUpdateMarket) isPTerm() {}
func (a ProposalTermsUpdateMarket) oneOfProto() interface{} {
	return a.IntoProto()
}

func (a ProposalTermsUpdateMarket) GetTermType() ProposalTermsType {
	return ProposalTermsTypeUpdateMarket
}

func (a ProposalTermsUpdateMarket) DeepClone() proposalTerm {
	if a.UpdateMarket == nil {
		return &ProposalTermsUpdateMarket{}
	}
	return &ProposalTermsUpdateMarket{
		UpdateMarket: a.UpdateMarket.DeepClone(),
	}
}

func (a ProposalTermsUpdateNetworkParameter) IntoProto() *vegapb.ProposalTerms_UpdateNetworkParameter {
	return &vegapb.ProposalTerms_UpdateNetworkParameter{
		UpdateNetworkParameter: a.UpdateNetworkParameter.IntoProto(),
	}
}

func (a ProposalTermsUpdateNetworkParameter) isPTerm() {}
func (a ProposalTermsUpdateNetworkParameter) oneOfProto() interface{} {
	return a.IntoProto()
}

func (a ProposalTermsUpdateNetworkParameter) GetTermType() ProposalTermsType {
	return ProposalTermsTypeUpdateNetworkParameter
}

func (a ProposalTermsUpdateNetworkParameter) DeepClone() proposalTerm {
	if a.UpdateNetworkParameter == nil {
		return &ProposalTermsUpdateNetworkParameter{}
	}
	return &ProposalTermsUpdateNetworkParameter{
		UpdateNetworkParameter: a.UpdateNetworkParameter.DeepClone(),
	}
}

func (n UpdateNetworkParameter) IntoProto() *vegapb.UpdateNetworkParameter {
	return &vegapb.UpdateNetworkParameter{
		Changes: n.Changes.IntoProto(),
	}
}

func (n UpdateNetworkParameter) String() string {
	return n.IntoProto().String()
}

func (n UpdateNetworkParameter) DeepClone() *UpdateNetworkParameter {
	if n.Changes == nil {
		return &UpdateNetworkParameter{}
	}
	return &UpdateNetworkParameter{
		Changes: n.Changes.DeepClone(),
	}
}

func (a ProposalTermsNewAsset) IntoProto() *vegapb.ProposalTerms_NewAsset {
	var newAsset *vegapb.NewAsset
	if a.NewAsset != nil {
		newAsset = a.NewAsset.IntoProto()
	}
	return &vegapb.ProposalTerms_NewAsset{
		NewAsset: newAsset,
	}
}

func (a ProposalTermsNewAsset) isPTerm() {}
func (a ProposalTermsNewAsset) oneOfProto() interface{} {
	return a.IntoProto()
}

func (a ProposalTermsNewAsset) GetTermType() ProposalTermsType {
	return ProposalTermsTypeNewAsset
}

func (a ProposalTermsNewAsset) DeepClone() proposalTerm {
	if a.NewAsset == nil {
		return &ProposalTermsNewAsset{}
	}
	return &ProposalTermsNewAsset{
		NewAsset: a.NewAsset.DeepClone(),
	}
}

func (n NewAsset) IntoProto() *vegapb.NewAsset {
	var changes *vegapb.AssetDetails
	if n.Changes != nil {
		changes = n.Changes.IntoProto()
	}
	return &vegapb.NewAsset{
		Changes: changes,
	}
}

func (n NewAsset) String() string {
	return n.IntoProto().String()
}

func (n NewAsset) DeepClone() *NewAsset {
	if n.Changes == nil {
		return &NewAsset{}
	}
	return &NewAsset{
		Changes: n.Changes.DeepClone(),
	}
}

func (n NewMarketCommitment) IntoProto() *vegapb.NewMarketCommitment {
	r := &vegapb.NewMarketCommitment{
		CommitmentAmount: num.UintToString(n.CommitmentAmount),
		Fee:              n.Fee.String(),
		Sells:            make([]*vegapb.LiquidityOrder, 0, len(n.Sells)),
		Buys:             make([]*vegapb.LiquidityOrder, 0, len(n.Buys)),
		Reference:        n.Reference,
	}
	for _, s := range n.Sells {
		r.Sells = append(r.Sells, s.IntoProto())
	}
	for _, b := range n.Buys {
		r.Buys = append(r.Buys, b.IntoProto())
	}
	return r
}

func (n NewMarketCommitment) String() string {
	return n.IntoProto().String()
}

type NewMarketConfigurationLogNormal struct {
	LogNormal *LogNormalRiskModel
}

type NewMarketConfigurationSimple struct {
	Simple *SimpleModelParams
}

func (n NewMarketConfigurationLogNormal) IntoProto() *vegapb.NewMarketConfiguration_LogNormal {
	return &vegapb.NewMarketConfiguration_LogNormal{
		LogNormal: n.LogNormal.IntoProto(),
	}
}

func NewMarketConfigurationLogNormalFromProto(p *vegapb.NewMarketConfiguration_LogNormal) *NewMarketConfigurationLogNormal {
	return &NewMarketConfigurationLogNormal{
		LogNormal: &LogNormalRiskModel{
			RiskAversionParameter: num.DecimalFromFloat(p.LogNormal.RiskAversionParameter),
			Tau:                   num.DecimalFromFloat(p.LogNormal.Tau),
			Params:                LogNormalParamsFromProto(p.LogNormal.Params),
		},
	}
}

func (n NewMarketConfigurationLogNormal) DeepClone() riskParams {
	if n.LogNormal == nil {
		return &NewMarketConfigurationLogNormal{}
	}
	return &NewMarketConfigurationLogNormal{
		LogNormal: n.LogNormal.DeepClone(),
	}
}

func (*NewMarketConfigurationLogNormal) isNMCRP() {}

func (n NewMarketConfigurationLogNormal) rpIntoProto() interface{} {
	return n.IntoProto()
}

func (n NewMarketConfigurationSimple) IntoProto() *vegapb.NewMarketConfiguration_Simple {
	return &vegapb.NewMarketConfiguration_Simple{
		Simple: n.Simple.IntoProto(),
	}
}

func NewMarketConfigurationSimpleFromProto(p *vegapb.NewMarketConfiguration_Simple) *NewMarketConfigurationSimple {
	return &NewMarketConfigurationSimple{
		Simple: SimpleModelParamsFromProto(p.Simple),
	}
}

func (*NewMarketConfigurationSimple) isNMCRP() {}

func (n NewMarketConfigurationSimple) DeepClone() riskParams {
	if n.Simple == nil {
		return &NewMarketConfigurationSimple{}
	}
	return &NewMarketConfigurationSimple{
		Simple: n.Simple.DeepClone(),
	}
}

func (n NewMarketConfigurationSimple) rpIntoProto() interface{} {
	return n.IntoProto()
}

type InstrumentConfiguration struct {
	Name string
	Code string
	// *InstrumentConfigurationFuture
	Product instrumentConfigurationProduct
}

type instrumentConfigurationProduct interface {
	isInstrumentConfigurationProduct()
	icpIntoProto() interface{}
	Asset() string
	DeepClone() instrumentConfigurationProduct
}

type InstrumentConfigurationFuture struct {
	Future *FutureProduct
}

type FutureProduct struct {
	SettlementAsset                 string
	QuoteName                       string
	OracleSpecForSettlementPrice    *oraclespb.OracleSpecConfiguration
	OracleSpecForTradingTermination *oraclespb.OracleSpecConfiguration
	OracleSpecBinding               *OracleSpecToFutureBinding
}

func (i InstrumentConfigurationFuture) DeepClone() instrumentConfigurationProduct {
	if i.Future == nil {
		return &InstrumentConfigurationFuture{}
	}
	return &InstrumentConfigurationFuture{
		Future: i.Future.DeepClone(),
	}
}

func (i InstrumentConfigurationFuture) Asset() string {
	return i.Future.SettlementAsset
}

func (i InstrumentConfiguration) DeepClone() *InstrumentConfiguration {
	cpy := InstrumentConfiguration{
		Name: i.Name,
		Code: i.Code,
	}
	if i.Product != nil {
		cpy.Product = i.Product.DeepClone()
	}
	return &cpy
}

func (i InstrumentConfiguration) IntoProto() *vegapb.InstrumentConfiguration {
	p := i.Product.icpIntoProto()
	r := &vegapb.InstrumentConfiguration{
		Name: i.Name,
		Code: i.Code,
	}
	switch pr := p.(type) {
	case *vegapb.InstrumentConfiguration_Future:
		r.Product = pr
	}
	return r
}

func InstrumentConfigurationFromProto(
	p *vegapb.InstrumentConfiguration,
) *InstrumentConfiguration {
	r := &InstrumentConfiguration{
		Name: p.Name,
		Code: p.Code,
	}

	switch pr := p.Product.(type) {
	case *vegapb.InstrumentConfiguration_Future:
		r.Product = &InstrumentConfigurationFuture{
			Future: &FutureProduct{
				SettlementAsset:                 pr.Future.SettlementAsset,
				QuoteName:                       pr.Future.QuoteName,
				OracleSpecForSettlementPrice:    pr.Future.OracleSpecForSettlementPrice.DeepClone(),
				OracleSpecForTradingTermination: pr.Future.OracleSpecForTradingTermination.DeepClone(),
				OracleSpecBinding: OracleSpecToFutureBindingFromProto(
					pr.Future.OracleSpecBinding),
			},
		}
	}
	return r
}

func (i InstrumentConfigurationFuture) IntoProto() *vegapb.InstrumentConfiguration_Future {
	return &vegapb.InstrumentConfiguration_Future{
		Future: i.Future.IntoProto(),
	}
}

func (i InstrumentConfigurationFuture) icpIntoProto() interface{} {
	return i.IntoProto()
}

func (InstrumentConfigurationFuture) isInstrumentConfigurationProduct() {}

func (f FutureProduct) IntoProto() *vegapb.FutureProduct {
	return &vegapb.FutureProduct{
		SettlementAsset:                 f.SettlementAsset,
		QuoteName:                       f.QuoteName,
		OracleSpecForSettlementPrice:    f.OracleSpecForSettlementPrice.DeepClone(),
		OracleSpecForTradingTermination: f.OracleSpecForTradingTermination.DeepClone(),

		OracleSpecBinding: f.OracleSpecBinding.IntoProto(),
	}
}

func (f FutureProduct) DeepClone() *FutureProduct {
	return &FutureProduct{
		SettlementAsset:                 f.SettlementAsset,
		QuoteName:                       f.QuoteName,
		OracleSpecForSettlementPrice:    f.OracleSpecForSettlementPrice.DeepClone(),
		OracleSpecForTradingTermination: f.OracleSpecForTradingTermination.DeepClone(),
		OracleSpecBinding:               f.OracleSpecBinding.DeepClone(),
	}
}

func (f FutureProduct) String() string {
	return f.IntoProto().String()
}

func (f FutureProduct) Asset() string {
	return f.SettlementAsset
}

func (f ProposalTerms_NewFreeform) IntoProto() *vegapb.ProposalTerms_NewFreeform {
	var newFreeform *vegapb.NewFreeform
	if f.NewFreeform != nil {
		newFreeform = f.NewFreeform.IntoProto()
	}
	return &vegapb.ProposalTerms_NewFreeform{
		NewFreeform: newFreeform,
	}
}

func (f ProposalTerms_NewFreeform) isPTerm() {}
func (f ProposalTerms_NewFreeform) oneOfProto() interface{} {
	return f.IntoProto()
}

func (f ProposalTerms_NewFreeform) GetTermType() ProposalTermsType {
	return ProposalTermsTypeNewFreeform
}

func (f ProposalTerms_NewFreeform) DeepClone() proposalTerm {
	if f.NewFreeform == nil {
		return &ProposalTerms_NewFreeform{}
	}
	return &ProposalTerms_NewFreeform{
		NewFreeform: f.NewFreeform.DeepClone(),
	}
}

func (n NewFreeform) IntoProto() *vegapb.NewFreeform {
	return &vegapb.NewFreeform{
		Changes: &vegapb.NewFreeformDetails{
			Url:         n.Changes.URL,
			Description: n.Changes.Description,
			Hash:        n.Changes.Hash,
		},
	}
}

func (n NewFreeform) String() string {
	return n.IntoProto().String()
}

func (n NewFreeform) DeepClone() *NewFreeform {
	return &NewFreeform{
		Changes: &NewFreeformDetails{
			URL:         n.Changes.URL,
			Description: n.Changes.Description,
			Hash:        n.Changes.Hash,
		},
	}
}
