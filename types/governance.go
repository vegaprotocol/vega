//lint:file-ignore ST1003 Ignore underscores in names, this is straigh copied from the proto package to ease introducing the domain types

package types

import (
	"errors"

	proto "code.vegaprotocol.io/protos/vega"
	commandspb "code.vegaprotocol.io/protos/vega/commands/v1"
	v1 "code.vegaprotocol.io/protos/vega/oracles/v1"
	"code.vegaprotocol.io/vega/types/num"
)

type GovernanceData = proto.GovernanceData

type VoteValue = proto.Vote_Value

const (
	// Default value, always invalid.
	VoteValueUnspecified VoteValue = proto.Vote_VALUE_UNSPECIFIED
	// A vote against the proposal.
	VoteValueNo VoteValue = proto.Vote_VALUE_NO
	// A vote in favour of the proposal.
	VoteValueYes VoteValue = proto.Vote_VALUE_YES
)

type ProposalError = proto.ProposalError

const (
	// Default value.
	ProposalError_PROPOSAL_ERROR_UNSPECIFIED ProposalError = 0
	// The specified close time is too early base on network parameters.
	ProposalError_PROPOSAL_ERROR_CLOSE_TIME_TOO_SOON ProposalError = 1
	// The specified close time is too late based on network parameters.
	ProposalError_PROPOSAL_ERROR_CLOSE_TIME_TOO_LATE ProposalError = 2
	// The specified enact time is too early based on network parameters.
	ProposalError_PROPOSAL_ERROR_ENACT_TIME_TOO_SOON ProposalError = 3
	// The specified enact time is too late based on network parameters.
	ProposalError_PROPOSAL_ERROR_ENACT_TIME_TOO_LATE ProposalError = 4
	// The proposer for this proposal as insufficient tokens.
	ProposalError_PROPOSAL_ERROR_INSUFFICIENT_TOKENS ProposalError = 5
	// The instrument quote name and base name were the same.
	ProposalError_PROPOSAL_ERROR_INVALID_INSTRUMENT_SECURITY ProposalError = 6
	// The proposal has no product.
	ProposalError_PROPOSAL_ERROR_NO_PRODUCT ProposalError = 7
	// The specified product is not supported.
	ProposalError_PROPOSAL_ERROR_UNSUPPORTED_PRODUCT ProposalError = 8
	// Invalid future maturity timestamp (expect RFC3339).
	ProposalError_PROPOSAL_ERROR_INVALID_FUTURE_PRODUCT_TIMESTAMP ProposalError = 9
	// The product maturity is past.
	ProposalError_PROPOSAL_ERROR_PRODUCT_MATURITY_IS_PASSED ProposalError = 10
	// The proposal has no trading mode.
	ProposalError_PROPOSAL_ERROR_NO_TRADING_MODE ProposalError = 11
	// The proposal has an unsupported trading mode.
	ProposalError_PROPOSAL_ERROR_UNSUPPORTED_TRADING_MODE ProposalError = 12
	// The proposal failed node validation.
	ProposalError_PROPOSAL_ERROR_NODE_VALIDATION_FAILED ProposalError = 13
	// A field is missing in a builtin asset source.
	ProposalError_PROPOSAL_ERROR_MISSING_BUILTIN_ASSET_FIELD ProposalError = 14
	// The contract address is missing in the ERC20 asset source.
	ProposalError_PROPOSAL_ERROR_MISSING_ERC20_CONTRACT_ADDRESS ProposalError = 15
	// The asset identifier is invalid or does not exist on the Vega network.
	ProposalError_PROPOSAL_ERROR_INVALID_ASSET ProposalError = 16
	// Proposal terms timestamps are not compatible (Validation < Closing < Enactment).
	ProposalError_PROPOSAL_ERROR_INCOMPATIBLE_TIMESTAMPS ProposalError = 17
	// No risk parameters were specified.
	ProposalError_PROPOSAL_ERROR_NO_RISK_PARAMETERS ProposalError = 18
	// Invalid key in update network parameter proposal.
	ProposalError_PROPOSAL_ERROR_NETWORK_PARAMETER_INVALID_KEY ProposalError = 19
	// Invalid valid in update network parameter proposal.
	ProposalError_PROPOSAL_ERROR_NETWORK_PARAMETER_INVALID_VALUE ProposalError = 20
	// Validation failed for network parameter proposal.
	ProposalError_PROPOSAL_ERROR_NETWORK_PARAMETER_VALIDATION_FAILED ProposalError = 21
	// Opening auction duration is less than the network minimum opening auction time.
	ProposalError_PROPOSAL_ERROR_OPENING_AUCTION_DURATION_TOO_SMALL ProposalError = 22
	// Opening auction duration is more than the network minimum opening auction time.
	ProposalError_PROPOSAL_ERROR_OPENING_AUCTION_DURATION_TOO_LARGE ProposalError = 23
	// Market proposal is missing a liquidity commitment.
	ProposalError_PROPOSAL_ERROR_MARKET_MISSING_LIQUIDITY_COMMITMENT ProposalError = 24
	// Market proposal market could not be instantiate in execution.
	ProposalError_PROPOSAL_ERROR_COULD_NOT_INSTANTIATE_MARKET ProposalError = 25
	// Market proposal market contained invalid product definition.
	ProposalError_PROPOSAL_ERROR_INVALID_FUTURE_PRODUCT ProposalError = 26
	// Proposal is an unknown type.
	ProposalError_PROPOSAL_ERROR_UNKNOWN_TYPE ProposalError = 27
	// Proposal has an unknown risk parameter type.
	ProposalError_PROPOSAL_ERROR_UNKNOWN_RISK_PARAMETER_TYPE ProposalError = 28
	// Validation failed for freeform proposal.
	ProposalError_PROPOSAL_ERROR_INVALID_FREEFORM ProposalError = 29
)

type ProposalState = proto.Proposal_State

const (
	// Default value, always invalid.
	ProposalStateUnspecified ProposalState = proto.Proposal_STATE_UNSPECIFIED
	// Proposal enactment has failed - even though proposal has passed, its execution could not be performed.
	ProposalStateFailed ProposalState = proto.Proposal_STATE_FAILED
	// Proposal is open for voting.
	ProposalStateOpen ProposalState = proto.Proposal_STATE_OPEN
	// Proposal has gained enough support to be executed.
	ProposalStatePassed ProposalState = proto.Proposal_STATE_PASSED
	// Proposal wasn't accepted (proposal terms failed validation due to wrong configuration or failing to meet network requirements).
	ProposalStateRejected ProposalState = proto.Proposal_STATE_REJECTED
	// Proposal didn't get enough votes (either failing to gain required participation or majority level).
	ProposalStateDeclined ProposalState = proto.Proposal_STATE_DECLINED
	// Proposal enacted.
	ProposalStateEnacted ProposalState = proto.Proposal_STATE_ENACTED
	// Waiting for node validation of the proposal.
	ProposalStateWaitingForNodeVote ProposalState = proto.Proposal_STATE_WAITING_FOR_NODE_VOTE
)

type Proposal_Terms_TYPE int

const (
	ProposalTerms_UPDATE_MARKET Proposal_Terms_TYPE = iota
	ProposalTerms_NEW_MARKET
	ProposalTerms_UPDATE_NETWORK_PARAMETER
	ProposalTerms_NEW_ASSET
	ProposalTerms_NEW_FREEFORM
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
	var terms *proto.ProposalTerms
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

func ProposalFromProto(pp *proto.Proposal) *Proposal {
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

func (p Proposal) IntoProto() *proto.Proposal {
	var terms *proto.ProposalTerms
	if p.Terms != nil {
		terms = p.Terms.IntoProto()
	}
	return &proto.Proposal{
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

func (v Vote) IntoProto() *proto.Vote {
	return &proto.Vote{
		PartyId:                     v.PartyID,
		Value:                       v.Value,
		ProposalId:                  v.ProposalID,
		Timestamp:                   v.Timestamp,
		TotalGovernanceTokenBalance: num.UintToString(v.TotalGovernanceTokenBalance),
		TotalGovernanceTokenWeight:  v.TotalGovernanceTokenWeight.String(),
	}
}

func VoteFromProto(v *proto.Vote) (*Vote, error) {
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

func NewMarketCommitmentFromProto(p *proto.NewMarketCommitment) (*NewMarketCommitment, error) {
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
	// *ProposalTerms_UpdateMarket
	// *ProposalTerms_NewMarket
	// *ProposalTerms_UpdateNetworkParameter
	// *ProposalTerms_NewAsset
	Change pterms
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
	//	*NewMarketConfiguration_Simple
	//	*NewMarketConfiguration_LogNormal
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

type ProposalTerms_NewMarket struct {
	NewMarket *NewMarket
}

type (
	UpdateMarket               = proto.UpdateMarket
	ProposalTerms_UpdateMarket struct {
		UpdateMarket *UpdateMarket
	}
)

type UpdateNetworkParameter struct {
	Changes *NetworkParameter
}

type ProposalTerms_UpdateNetworkParameter struct {
	UpdateNetworkParameter *UpdateNetworkParameter
}

type NewAsset struct {
	Changes *AssetDetails
}

type ProposalTerms_NewAsset struct {
	NewAsset *NewAsset
}

type NewFreeform struct {
	URL         string
	Description string
	Hash        string
}

type ProposalTerms_NewFreeform struct {
	NewFreeform *NewFreeform
}

type pterms interface {
	isPTerm()
	oneOfProto() interface{} // calls IntoProto
	DeepClone() pterms
	GetTermType() Proposal_Terms_TYPE
}

func (n *NewAsset) GetChanges() *AssetDetails {
	if n != nil {
		return n.Changes
	}
	return nil
}

func (n NewMarket) IntoProto() *proto.NewMarket {
	var changes *proto.NewMarketConfiguration
	if n.Changes != nil {
		changes = n.Changes.IntoProto()
	}
	var commitment *proto.NewMarketCommitment
	if n.LiquidityCommitment != nil {
		commitment = n.LiquidityCommitment.IntoProto()
	}
	return &proto.NewMarket{
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

func (n NewMarketConfiguration) IntoProto() *proto.NewMarketConfiguration {
	riskParams := n.RiskParameters.rpIntoProto()
	tradingMode := n.TradingMode.tmIntoProto()
	md := make([]string, 0, len(n.Metadata))
	md = append(md, n.Metadata...)

	var instrument *proto.InstrumentConfiguration
	if n.Instrument != nil {
		instrument = n.Instrument.IntoProto()
	}
	var priceMonitoring *proto.PriceMonitoringParameters
	if n.PriceMonitoringParameters != nil {
		priceMonitoring = n.PriceMonitoringParameters.IntoProto()
	}
	var liquidityMonitoring *proto.LiquidityMonitoringParameters
	if n.LiquidityMonitoringParameters != nil {
		liquidityMonitoring = n.LiquidityMonitoringParameters.IntoProto()
	}

	r := &proto.NewMarketConfiguration{
		Instrument:                    instrument,
		DecimalPlaces:                 n.DecimalPlaces,
		Metadata:                      md,
		PriceMonitoringParameters:     priceMonitoring,
		LiquidityMonitoringParameters: liquidityMonitoring,
	}
	switch rp := riskParams.(type) {
	case *proto.NewMarketConfiguration_Simple:
		r.RiskParameters = rp
	case *proto.NewMarketConfiguration_LogNormal:
		r.RiskParameters = rp
	}
	switch tm := tradingMode.(type) {
	case *proto.NewMarketConfiguration_Continuous:
		r.TradingMode = tm
	case *proto.NewMarketConfiguration_Discrete:
		r.TradingMode = tm
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

func NewMarketConfigurationFromProto(p *proto.NewMarketConfiguration) *NewMarketConfiguration {
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
		case *proto.NewMarketConfiguration_Simple:
			r.RiskParameters = NewMarketConfiguration_SimpleFromProto(rp)
		case *proto.NewMarketConfiguration_LogNormal:
			r.RiskParameters = NewMarketConfiguration_LogNormalFromProto(rp)
		}
	}
	if p.TradingMode != nil {
		switch tm := p.TradingMode.(type) {
		case *proto.NewMarketConfiguration_Continuous:
			r.TradingMode = NewMarketConfiguration_ContinuousFromProto(tm)
		case *proto.NewMarketConfiguration_Discrete:
			r.TradingMode = NewMarketConfiguration_DiscreteFromProto(tm)
		}
	}

	return r
}

func (n *NewMarketConfiguration) GetTradingMode() tradingMode {
	if n != nil {
		return n.TradingMode
	}
	return nil
}

func (n *NewMarketConfiguration) GetContinuous() *ContinuousTrading {
	if x, ok := n.GetTradingMode().(*NewMarketConfiguration_Continuous); ok {
		return x.Continuous
	}
	return nil
}

func ProposalTermsFromProto(p *proto.ProposalTerms) *ProposalTerms {
	var change pterms
	if p.Change != nil {
		switch ch := p.Change.(type) {
		case *proto.ProposalTerms_NewMarket:
			change = NewNewMarketFromProto(ch)
		case *proto.ProposalTerms_UpdateMarket:
			change = NewUpdateMarketFromProto(ch)
		case *proto.ProposalTerms_UpdateNetworkParameter:
			change = NewUpdateNetworkParameterFromProto(ch)
		case *proto.ProposalTerms_NewAsset:
			change = NewNewAssetFromProto(ch)
		case *proto.ProposalTerms_NewFreeform:
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

func NewNewMarketFromProto(p *proto.ProposalTerms_NewMarket) *ProposalTerms_NewMarket {
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

	return &ProposalTerms_NewMarket{
		NewMarket: newMarket,
	}
}

func NewUpdateMarketFromProto(p *proto.ProposalTerms_UpdateMarket) *ProposalTerms_UpdateMarket {
	panic("unimplemented")
}

func NewUpdateNetworkParameterFromProto(
	p *proto.ProposalTerms_UpdateNetworkParameter,
) *ProposalTerms_UpdateNetworkParameter {
	var updateNP *UpdateNetworkParameter
	if p.UpdateNetworkParameter != nil {
		updateNP = &UpdateNetworkParameter{}

		if p.UpdateNetworkParameter.Changes != nil {
			updateNP.Changes = NetworkParameterFromProto(p.UpdateNetworkParameter.Changes)
		}
	}

	return &ProposalTerms_UpdateNetworkParameter{
		UpdateNetworkParameter: updateNP,
	}
}

func NewNewAssetFromProto(p *proto.ProposalTerms_NewAsset) *ProposalTerms_NewAsset {
	var newAsset *NewAsset
	if p.NewAsset != nil {
		newAsset = &NewAsset{}

		if p.NewAsset.Changes != nil {
			newAsset.Changes = AssetDetailsFromProto(p.NewAsset.Changes)
		}
	}

	return &ProposalTerms_NewAsset{
		NewAsset: newAsset,
	}
}

func NewNewFreeformFromProto(p *proto.ProposalTerms_NewFreeform) *ProposalTerms_NewFreeform {
	var newFreeform *NewFreeform
	if p.NewFreeform != nil {
		newFreeform = &NewFreeform{
			URL:         p.NewFreeform.Url,
			Description: p.NewFreeform.Description,
			Hash:        p.NewFreeform.Hash,
		}
	}

	return &ProposalTerms_NewFreeform{
		NewFreeform: newFreeform,
	}
}

func (p ProposalTerms) IntoProto() *proto.ProposalTerms {
	change := p.Change.oneOfProto()
	r := &proto.ProposalTerms{
		ClosingTimestamp:    p.ClosingTimestamp,
		EnactmentTimestamp:  p.EnactmentTimestamp,
		ValidationTimestamp: p.ValidationTimestamp,
	}
	switch ch := change.(type) {
	case *proto.ProposalTerms_NewMarket:
		r.Change = ch
	case *proto.ProposalTerms_UpdateMarket:
		r.Change = ch
	case *proto.ProposalTerms_UpdateNetworkParameter:
		r.Change = ch
	case *proto.ProposalTerms_NewAsset:
		r.Change = ch
	case *proto.ProposalTerms_NewFreeform:
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
	case *ProposalTerms_NewAsset:
		return c.NewAsset
	default:
		return nil
	}
}

func (p *ProposalTerms) GetNewMarket() *NewMarket {
	switch c := p.Change.(type) {
	case *ProposalTerms_NewMarket:
		return c.NewMarket
	default:
		return nil
	}
}

func (p *ProposalTerms) GetUpdateNetworkParameter() *UpdateNetworkParameter {
	switch c := p.Change.(type) {
	case *ProposalTerms_UpdateNetworkParameter:
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

func (a ProposalTerms_NewMarket) IntoProto() *proto.ProposalTerms_NewMarket {
	return &proto.ProposalTerms_NewMarket{
		NewMarket: a.NewMarket.IntoProto(),
	}
}

func (a ProposalTerms_NewMarket) isPTerm() {}
func (a ProposalTerms_NewMarket) oneOfProto() interface{} {
	return a.IntoProto()
}

func (a ProposalTerms_NewMarket) GetTermType() Proposal_Terms_TYPE {
	return ProposalTerms_NEW_MARKET
}

func (a ProposalTerms_NewMarket) DeepClone() pterms {
	if a.NewMarket == nil {
		return &ProposalTerms_NewMarket{}
	}
	return &ProposalTerms_NewMarket{
		NewMarket: a.NewMarket.DeepClone(),
	}
}

func (a ProposalTerms_UpdateMarket) IntoProto() *proto.ProposalTerms_UpdateMarket {
	return &proto.ProposalTerms_UpdateMarket{
		UpdateMarket: a.UpdateMarket,
	}
}

func (a ProposalTerms_UpdateMarket) isPTerm() {}
func (a ProposalTerms_UpdateMarket) oneOfProto() interface{} {
	return a.IntoProto()
}

func (a ProposalTerms_UpdateMarket) GetTermType() Proposal_Terms_TYPE {
	return ProposalTerms_UPDATE_MARKET
}

func (a ProposalTerms_UpdateMarket) DeepClone() pterms {
	if a.UpdateMarket == nil {
		return &ProposalTerms_UpdateMarket{}
	}
	return &ProposalTerms_UpdateMarket{
		UpdateMarket: a.UpdateMarket.DeepClone(),
	}
}

func (a ProposalTerms_UpdateNetworkParameter) IntoProto() *proto.ProposalTerms_UpdateNetworkParameter {
	return &proto.ProposalTerms_UpdateNetworkParameter{
		UpdateNetworkParameter: a.UpdateNetworkParameter.IntoProto(),
	}
}

func (a ProposalTerms_UpdateNetworkParameter) isPTerm() {}
func (a ProposalTerms_UpdateNetworkParameter) oneOfProto() interface{} {
	return a.IntoProto()
}

func (a ProposalTerms_UpdateNetworkParameter) GetTermType() Proposal_Terms_TYPE {
	return ProposalTerms_UPDATE_NETWORK_PARAMETER
}

func (a ProposalTerms_UpdateNetworkParameter) DeepClone() pterms {
	if a.UpdateNetworkParameter == nil {
		return &ProposalTerms_UpdateNetworkParameter{}
	}
	return &ProposalTerms_UpdateNetworkParameter{
		UpdateNetworkParameter: a.UpdateNetworkParameter.DeepClone(),
	}
}

func (n UpdateNetworkParameter) IntoProto() *proto.UpdateNetworkParameter {
	return &proto.UpdateNetworkParameter{
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

func (a ProposalTerms_NewAsset) IntoProto() *proto.ProposalTerms_NewAsset {
	var newAsset *proto.NewAsset
	if a.NewAsset != nil {
		newAsset = a.NewAsset.IntoProto()
	}
	return &proto.ProposalTerms_NewAsset{
		NewAsset: newAsset,
	}
}

func (a ProposalTerms_NewAsset) isPTerm() {}
func (a ProposalTerms_NewAsset) oneOfProto() interface{} {
	return a.IntoProto()
}

func (a ProposalTerms_NewAsset) GetTermType() Proposal_Terms_TYPE {
	return ProposalTerms_NEW_ASSET
}

func (a ProposalTerms_NewAsset) DeepClone() pterms {
	if a.NewAsset == nil {
		return &ProposalTerms_NewAsset{}
	}
	return &ProposalTerms_NewAsset{
		NewAsset: a.NewAsset.DeepClone(),
	}
}

func (n NewAsset) IntoProto() *proto.NewAsset {
	var changes *proto.AssetDetails
	if n.Changes != nil {
		changes = n.Changes.IntoProto()
	}
	return &proto.NewAsset{
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

func (n NewMarketCommitment) IntoProto() *proto.NewMarketCommitment {
	r := &proto.NewMarketCommitment{
		CommitmentAmount: num.UintToString(n.CommitmentAmount),
		Fee:              n.Fee.String(),
		Sells:            make([]*proto.LiquidityOrder, 0, len(n.Sells)),
		Buys:             make([]*proto.LiquidityOrder, 0, len(n.Buys)),
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

type NewMarketConfiguration_LogNormal struct {
	LogNormal *LogNormalRiskModel
}

type NewMarketConfiguration_Simple struct {
	Simple *SimpleModelParams
}

func (n NewMarketConfiguration_LogNormal) IntoProto() *proto.NewMarketConfiguration_LogNormal {
	return &proto.NewMarketConfiguration_LogNormal{
		LogNormal: n.LogNormal.IntoProto(),
	}
}

func NewMarketConfiguration_LogNormalFromProto(p *proto.NewMarketConfiguration_LogNormal) *NewMarketConfiguration_LogNormal {
	return &NewMarketConfiguration_LogNormal{
		LogNormal: &LogNormalRiskModel{
			RiskAversionParameter: num.DecimalFromFloat(p.LogNormal.RiskAversionParameter),
			Tau:                   num.DecimalFromFloat(p.LogNormal.Tau),
			Params:                LogNormalParamsFromProto(p.LogNormal.Params),
		},
	}
}

func (n NewMarketConfiguration_LogNormal) DeepClone() riskParams {
	if n.LogNormal == nil {
		return &NewMarketConfiguration_LogNormal{}
	}
	return &NewMarketConfiguration_LogNormal{
		LogNormal: n.LogNormal.DeepClone(),
	}
}

func (*NewMarketConfiguration_LogNormal) isNMCRP() {}

func (n NewMarketConfiguration_LogNormal) rpIntoProto() interface{} {
	return n.IntoProto()
}

func (n NewMarketConfiguration_Simple) IntoProto() *proto.NewMarketConfiguration_Simple {
	return &proto.NewMarketConfiguration_Simple{
		Simple: n.Simple.IntoProto(),
	}
}

func NewMarketConfiguration_SimpleFromProto(p *proto.NewMarketConfiguration_Simple) *NewMarketConfiguration_Simple {
	return &NewMarketConfiguration_Simple{
		Simple: SimpleModelParamsFromProto(p.Simple),
	}
}

func (*NewMarketConfiguration_Simple) isNMCRP() {}

func (n NewMarketConfiguration_Simple) DeepClone() riskParams {
	if n.Simple == nil {
		return &NewMarketConfiguration_Simple{}
	}
	return &NewMarketConfiguration_Simple{
		Simple: n.Simple.DeepClone(),
	}
}

func (n NewMarketConfiguration_Simple) rpIntoProto() interface{} {
	return n.IntoProto()
}

type InstrumentConfiguration struct {
	Name string
	Code string
	// *InstrumentConfiguration_Future
	Product icProd
}

type icProd interface {
	isInstrumentConfiguration_Product()
	icpIntoProto() interface{}
	Asset() string
	DeepClone() icProd
}

type InstrumentConfiguration_Future struct {
	Future *FutureProduct
}

type FutureProduct struct {
	Maturity                        string
	SettlementAsset                 string
	QuoteName                       string
	OracleSpecForSettlementPrice    *v1.OracleSpecConfiguration
	OracleSpecForTradingTermination *v1.OracleSpecConfiguration
	OracleSpecBinding               *OracleSpecToFutureBinding
}

func (i InstrumentConfiguration_Future) DeepClone() icProd {
	if i.Future == nil {
		return &InstrumentConfiguration_Future{}
	}
	return &InstrumentConfiguration_Future{
		Future: i.Future.DeepClone(),
	}
}

func (i InstrumentConfiguration_Future) Asset() string {
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

func (i InstrumentConfiguration) IntoProto() *proto.InstrumentConfiguration {
	p := i.Product.icpIntoProto()
	r := &proto.InstrumentConfiguration{
		Name: i.Name,
		Code: i.Code,
	}
	switch pr := p.(type) {
	case *proto.InstrumentConfiguration_Future:
		r.Product = pr
	}
	return r
}

func InstrumentConfigurationFromProto(
	p *proto.InstrumentConfiguration,
) *InstrumentConfiguration {
	r := &InstrumentConfiguration{
		Name: p.Name,
		Code: p.Code,
	}

	switch pr := p.Product.(type) {
	case *proto.InstrumentConfiguration_Future:
		r.Product = &InstrumentConfiguration_Future{
			Future: &FutureProduct{
				Maturity:                        pr.Future.Maturity,
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

func (i InstrumentConfiguration_Future) IntoProto() *proto.InstrumentConfiguration_Future {
	return &proto.InstrumentConfiguration_Future{
		Future: i.Future.IntoProto(),
	}
}

func (i InstrumentConfiguration_Future) icpIntoProto() interface{} {
	return i.IntoProto()
}

func (InstrumentConfiguration_Future) isInstrumentConfiguration_Product() {}

func (f FutureProduct) IntoProto() *proto.FutureProduct {
	return &proto.FutureProduct{
		Maturity:                        f.Maturity,
		SettlementAsset:                 f.SettlementAsset,
		QuoteName:                       f.QuoteName,
		OracleSpecForSettlementPrice:    f.OracleSpecForSettlementPrice.DeepClone(),
		OracleSpecForTradingTermination: f.OracleSpecForTradingTermination.DeepClone(),

		OracleSpecBinding: f.OracleSpecBinding.IntoProto(),
	}
}

func (f FutureProduct) DeepClone() *FutureProduct {
	return &FutureProduct{
		Maturity:                        f.Maturity,
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

type ContinuousTrading struct {
	TickSize string
}

func ContinuousTradingFromProto(c *proto.ContinuousTrading) *ContinuousTrading {
	return &ContinuousTrading{
		TickSize: c.TickSize,
	}
}

func (c ContinuousTrading) IntoProto() *proto.ContinuousTrading {
	return &proto.ContinuousTrading{
		TickSize: c.TickSize,
	}
}

func (c ContinuousTrading) DeepClone() *ContinuousTrading {
	return &ContinuousTrading{
		TickSize: c.TickSize,
	}
}

func (c ContinuousTrading) String() string {
	return c.IntoProto().String()
}

type NewMarketConfiguration_Continuous struct {
	Continuous *ContinuousTrading
}

func (n NewMarketConfiguration_Continuous) IntoProto() *proto.NewMarketConfiguration_Continuous {
	return &proto.NewMarketConfiguration_Continuous{
		Continuous: n.Continuous.IntoProto(),
	}
}

func NewMarketConfiguration_ContinuousFromProto(p *proto.NewMarketConfiguration_Continuous) *NewMarketConfiguration_Continuous {
	return &NewMarketConfiguration_Continuous{
		Continuous: &ContinuousTrading{
			TickSize: p.Continuous.TickSize,
		},
	}
}

func (*NewMarketConfiguration_Continuous) isTradingMode() {}

func (n NewMarketConfiguration_Continuous) tmIntoProto() interface{} {
	return n.IntoProto()
}

func (n NewMarketConfiguration_Continuous) DeepClone() tradingMode {
	if n.Continuous == nil {
		return &NewMarketConfiguration_Continuous{}
	}
	return &NewMarketConfiguration_Continuous{
		Continuous: n.Continuous.DeepClone(),
	}
}

type NewMarketConfiguration_Discrete struct {
	Discrete *DiscreteTrading
}

func (n NewMarketConfiguration_Discrete) IntoProto() *proto.NewMarketConfiguration_Discrete {
	return &proto.NewMarketConfiguration_Discrete{
		Discrete: n.Discrete.IntoProto(),
	}
}

func NewMarketConfiguration_DiscreteFromProto(p *proto.NewMarketConfiguration_Discrete) *NewMarketConfiguration_Discrete {
	return &NewMarketConfiguration_Discrete{
		Discrete: &DiscreteTrading{
			DurationNs: p.Discrete.DurationNs,
			TickSize:   p.Discrete.TickSize,
		},
	}
}

func (*NewMarketConfiguration_Discrete) isTradingMode() {}

func (n NewMarketConfiguration_Discrete) tmIntoProto() interface{} {
	return n.IntoProto()
}

func (n NewMarketConfiguration_Discrete) DeepClone() tradingMode {
	if n.Discrete == nil {
		return &NewMarketConfiguration_Discrete{}
	}
	return &NewMarketConfiguration_Discrete{
		Discrete: n.Discrete.DeepClone(),
	}
}

type DiscreteTrading struct {
	DurationNs int64
	TickSize   string
}

func DiscreteTradingFromProto(d *proto.DiscreteTrading) *DiscreteTrading {
	return &DiscreteTrading{
		DurationNs: d.DurationNs,
		TickSize:   d.TickSize,
	}
}

func (d DiscreteTrading) DeepClone() *DiscreteTrading {
	return &DiscreteTrading{
		DurationNs: d.DurationNs,
		TickSize:   d.TickSize,
	}
}

func (d DiscreteTrading) IntoProto() *proto.DiscreteTrading {
	return &proto.DiscreteTrading{
		DurationNs: d.DurationNs,
		TickSize:   d.TickSize,
	}
}

func (f ProposalTerms_NewFreeform) IntoProto() *proto.ProposalTerms_NewFreeform {
	var newFreeform *proto.NewFreeform
	if f.NewFreeform != nil {
		newFreeform = f.NewFreeform.IntoProto()
	}
	return &proto.ProposalTerms_NewFreeform{
		NewFreeform: newFreeform,
	}
}

func (f ProposalTerms_NewFreeform) isPTerm() {}
func (f ProposalTerms_NewFreeform) oneOfProto() interface{} {
	return f.IntoProto()
}

func (f ProposalTerms_NewFreeform) GetTermType() Proposal_Terms_TYPE {
	return ProposalTerms_NEW_FREEFORM
}

func (f ProposalTerms_NewFreeform) DeepClone() pterms {
	if f.NewFreeform == nil {
		return &ProposalTerms_NewFreeform{}
	}
	return &ProposalTerms_NewFreeform{
		NewFreeform: f.NewFreeform.DeepClone(),
	}
}

func (n NewFreeform) IntoProto() *proto.NewFreeform {
	return &proto.NewFreeform{
		Url:         n.URL,
		Description: n.Description,
		Hash:        n.Hash,
	}
}

func (n NewFreeform) String() string {
	return n.IntoProto().String()
}

func (n NewFreeform) DeepClone() *NewFreeform {
	return &NewFreeform{
		URL:         n.URL,
		Description: n.Description,
		Hash:        n.Hash,
	}
}
