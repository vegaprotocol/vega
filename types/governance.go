//lint:file-ignore ST1003 Ignore underscores in names, this is straigh copied from the proto package to ease introducing the domain types

package types

import (
	"code.vegaprotocol.io/vega/proto"
	v1 "code.vegaprotocol.io/vega/proto/oracles/v1"
	"code.vegaprotocol.io/vega/types/num"
)

type Vote_Value = proto.Vote_Value

const (
	// Default value, always invalid
	Vote_VALUE_UNSPECIFIED Vote_Value = 0
	// A vote against the proposal
	Vote_VALUE_NO Vote_Value = 1
	// A vote in favour of the proposal
	Vote_VALUE_YES Vote_Value = 2
)

type ProposalError = proto.ProposalError

const (
	// Default value
	ProposalError_PROPOSAL_ERROR_UNSPECIFIED ProposalError = 0
	// The specified close time is too early base on network parameters
	ProposalError_PROPOSAL_ERROR_CLOSE_TIME_TOO_SOON ProposalError = 1
	// The specified close time is too late based on network parameters
	ProposalError_PROPOSAL_ERROR_CLOSE_TIME_TOO_LATE ProposalError = 2
	// The specified enact time is too early based on network parameters
	ProposalError_PROPOSAL_ERROR_ENACT_TIME_TOO_SOON ProposalError = 3
	// The specified enact time is too late based on network parameters
	ProposalError_PROPOSAL_ERROR_ENACT_TIME_TOO_LATE ProposalError = 4
	// The proposer for this proposal as insufficient tokens
	ProposalError_PROPOSAL_ERROR_INSUFFICIENT_TOKENS ProposalError = 5
	// The instrument quote name and base name were the same
	ProposalError_PROPOSAL_ERROR_INVALID_INSTRUMENT_SECURITY ProposalError = 6
	// The proposal has no product
	ProposalError_PROPOSAL_ERROR_NO_PRODUCT ProposalError = 7
	// The specified product is not supported
	ProposalError_PROPOSAL_ERROR_UNSUPPORTED_PRODUCT ProposalError = 8
	// Invalid future maturity timestamp (expect RFC3339)
	ProposalError_PROPOSAL_ERROR_INVALID_FUTURE_PRODUCT_TIMESTAMP ProposalError = 9
	// The product maturity is past
	ProposalError_PROPOSAL_ERROR_PRODUCT_MATURITY_IS_PASSED ProposalError = 10
	// The proposal has no trading mode
	ProposalError_PROPOSAL_ERROR_NO_TRADING_MODE ProposalError = 11
	// The proposal has an unsupported trading mode
	ProposalError_PROPOSAL_ERROR_UNSUPPORTED_TRADING_MODE ProposalError = 12
	// The proposal failed node validation
	ProposalError_PROPOSAL_ERROR_NODE_VALIDATION_FAILED ProposalError = 13
	// A field is missing in a builtin asset source
	ProposalError_PROPOSAL_ERROR_MISSING_BUILTIN_ASSET_FIELD ProposalError = 14
	// The contract address is missing in the ERC20 asset source
	ProposalError_PROPOSAL_ERROR_MISSING_ERC20_CONTRACT_ADDRESS ProposalError = 15
	// The asset identifier is invalid or does not exist on the Vega network
	ProposalError_PROPOSAL_ERROR_INVALID_ASSET ProposalError = 16
	// Proposal terms timestamps are not compatible (Validation < Closing < Enactment)
	ProposalError_PROPOSAL_ERROR_INCOMPATIBLE_TIMESTAMPS ProposalError = 17
	// No risk parameters were specified
	ProposalError_PROPOSAL_ERROR_NO_RISK_PARAMETERS ProposalError = 18
	// Invalid key in update network parameter proposal
	ProposalError_PROPOSAL_ERROR_NETWORK_PARAMETER_INVALID_KEY ProposalError = 19
	// Invalid valid in update network parameter proposal
	ProposalError_PROPOSAL_ERROR_NETWORK_PARAMETER_INVALID_VALUE ProposalError = 20
	// Validation failed for network parameter proposal
	ProposalError_PROPOSAL_ERROR_NETWORK_PARAMETER_VALIDATION_FAILED ProposalError = 21
	// Opening auction duration is less than the network minimum opening auction time
	ProposalError_PROPOSAL_ERROR_OPENING_AUCTION_DURATION_TOO_SMALL ProposalError = 22
	// Opening auction duration is more than the network minimum opening auction time
	ProposalError_PROPOSAL_ERROR_OPENING_AUCTION_DURATION_TOO_LARGE ProposalError = 23
	// Market proposal is missing a liquidity commitment
	ProposalError_PROPOSAL_ERROR_MARKET_MISSING_LIQUIDITY_COMMITMENT ProposalError = 24
	// Market proposal market could not be instantiate in execution
	ProposalError_PROPOSAL_ERROR_COULD_NOT_INSTANTIATE_MARKET ProposalError = 25
	// Market proposal market contained invalid product definition
	ProposalError_PROPOSAL_ERROR_INVALID_FUTURE_PRODUCT ProposalError = 26
)

type Proposal_State = proto.Proposal_State

const (
	// Default value, always invalid
	Proposal_STATE_UNSPECIFIED Proposal_State = 0
	// Proposal enactment has failed - even though proposal has passed, its execution could not be performed
	Proposal_STATE_FAILED Proposal_State = 1
	// Proposal is open for voting
	Proposal_STATE_OPEN Proposal_State = 2
	// Proposal has gained enough support to be executed
	Proposal_STATE_PASSED Proposal_State = 3
	// Proposal wasn't accepted (proposal terms failed validation due to wrong configuration or failing to meet network requirements)
	Proposal_STATE_REJECTED Proposal_State = 4
	// Proposal didn't get enough votes (either failing to gain required participation or majority level)
	Proposal_STATE_DECLINED Proposal_State = 5
	// Proposal enacted
	Proposal_STATE_ENACTED Proposal_State = 6
	// Waiting for node validation of the proposal
	Proposal_STATE_WAITING_FOR_NODE_VOTE Proposal_State = 7
)

type Proposal_Terms_TYPE int

const (
	ProposalTerms_UPDATE_MARKET Proposal_Terms_TYPE = iota
	ProposalTerms_NEW_MARKET
	ProposalTerms_UPDATE_NETWORK_PARAMETER
	ProposalTerms_NEW_ASSET
)

// Vote represents a governance vote casted by a party for a given proposal.
type Vote struct {
	// PartyID is the party that casted the vote.
	PartyID string
	// ProposalID is the proposal identifier concerned by the vote.
	ProposalID string
	// Value is the actual position of the vote: yes or no.
	Value Vote_Value
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

type Proposal struct {
	Id           string
	Reference    string
	PartyId      string
	State        Proposal_State
	Timestamp    int64
	Terms        *ProposalTerms
	Reason       ProposalError
	ErrorDetails string
}

func (p Proposal) DeepClone() *Proposal {
	cpy := p
	cpy.Terms = p.Terms.DeepClone()
	return &cpy
}

func (p Proposal) IntoProto() *proto.Proposal {
	return &proto.Proposal{
		Id:           p.Id,
		Reference:    p.Reference,
		PartyId:      p.PartyId,
		State:        p.State,
		Timestamp:    p.Timestamp,
		Terms:        p.Terms.IntoProto(),
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
		TotalGovernanceTokenBalance: v.TotalGovernanceTokenBalance.Uint64(),
		TotalGovernanceTokenWeight:  v.TotalGovernanceTokenWeight.String(),
	}
}

type NewMarketCommitment struct {
	CommitmentAmount *num.Uint
	Fee              num.Decimal
	Sells            []*LiquidityOrder
	Buys             []*LiquidityOrder
	Reference        string
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
}

type tradingMode interface {
	isTradingMode()
	tmIntoProto() interface{}
}

type ProposalTerms_NewMarket struct {
	NewMarket *NewMarket
}

type UpdateMarket = proto.UpdateMarket
type ProposalTerms_UpdateMarket struct {
	UpdateMarket *UpdateMarket
}

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

type pterms interface {
	isPTerm()
	oneOfProto() interface{} // calls IntoProto
	DeepClone() pterms
	GetTermType() Proposal_Terms_TYPE
}

func (m *NewAsset) GetChanges() *AssetDetails {
	if m != nil {
		return m.Changes
	}
	return nil
}

func (n NewMarket) IntoProto() *proto.NewMarket {
	return &proto.NewMarket{
		Changes:             n.Changes.IntoProto(),
		LiquidityCommitment: n.LiquidityCommitment.IntoProto(),
	}
}

func (n NewMarketConfiguration) IntoProto() *proto.NewMarketConfiguration {
	riskParams := n.RiskParameters.rpIntoProto()
	tradingMode := n.TradingMode.tmIntoProto()
	md := make([]string, 0, len(n.Metadata))
	md = append(md, n.Metadata...)
	r := &proto.NewMarketConfiguration{
		Instrument:                    n.Instrument.IntoProto(),
		DecimalPlaces:                 n.DecimalPlaces,
		Metadata:                      md,
		PriceMonitoringParameters:     n.PriceMonitoringParameters.IntoProto(),
		LiquidityMonitoringParameters: n.LiquidityMonitoringParameters.IntoProto(),
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

func ProposalTermsFromProto(p *proto.ProposalTerms) *ProposalTerms {
	r := &ProposalTerms{
		ClosingTimestamp:    p.ClosingTimestamp,
		EnactmentTimestamp:  p.EnactmentTimestamp,
		ValidationTimestamp: p.ValidationTimestamp,
	}
	return r
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

func (m *ProposalTerms) GetNewAsset() *NewAsset {
	switch c := m.Change.(type) {
	case ProposalTerms_NewAsset:
		return c.NewAsset
	default:
		return nil
	}
}

func (m *ProposalTerms) GetNewMarket() *NewMarket {
	switch c := m.Change.(type) {
	case ProposalTerms_NewMarket:
		return c.NewMarket
	default:
		return nil
	}
}

func (m *ProposalTerms) GetUpdateNetworkParameter() *UpdateNetworkParameter {
	switch c := m.Change.(type) {
	case ProposalTerms_UpdateNetworkParameter:
		return c.UpdateNetworkParameter
	default:
		return nil
	}
}

func (a ProposalTerms_NewMarket) IntoProto() *proto.ProposalTerms_NewMarket {
	return &proto.ProposalTerms_NewMarket{
		NewMarket: a.NewMarket.IntoProto(),
	}
}

func ProposalNewMarketFromProto(p *proto.ProposalTerms_NewMarket) *ProposalTerms_NewMarket {
	return &ProposalTerms_NewMarket{
		NewMarket: &NewMarket{}, // @TODO
	}
}

func (a ProposalTerms_NewMarket) isPTerm() {}
func (a ProposalTerms_NewMarket) oneOfProto() interface{} {
	return a.IntoProto()
}
func (a ProposalTerms_NewMarket) GetTermType() Proposal_Terms_TYPE {
	return ProposalTerms_NEW_MARKET
}

// DeepClone @TODO
func (a ProposalTerms_NewMarket) DeepClone() pterms {
	return a
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

// DeepClone @TODO
func (a ProposalTerms_UpdateMarket) DeepClone() pterms {
	return a
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

// DeepClone @TODO
func (a ProposalTerms_UpdateNetworkParameter) DeepClone() pterms {
	return a
}

func (n UpdateNetworkParameter) IntoProto() *proto.UpdateNetworkParameter {
	return &proto.UpdateNetworkParameter{
		Changes: n.Changes.IntoProto(),
	}
}

func (n UpdateNetworkParameter) String() string {
	return n.IntoProto().String()
}

func (a ProposalTerms_NewAsset) IntoProto() *proto.ProposalTerms_NewAsset {
	return &proto.ProposalTerms_NewAsset{
		NewAsset: a.NewAsset.IntoProto(),
	}
}

func (a ProposalTerms_NewAsset) isPTerm() {}
func (a ProposalTerms_NewAsset) oneOfProto() interface{} {
	return a.IntoProto()
}
func (a ProposalTerms_NewAsset) GetTermType() Proposal_Terms_TYPE {
	return ProposalTerms_NEW_ASSET
}

// DeepClone @TODO
func (a ProposalTerms_NewAsset) DeepClone() pterms {
	return a
}

func (n NewAsset) IntoProto() *proto.NewAsset {
	return &proto.NewAsset{
		Changes: n.Changes.IntoProto(),
	}
}

func (n NewAsset) String() string {
	return n.IntoProto().String()
}

func (n NewMarketCommitment) IntoProto() *proto.NewMarketCommitment {
	r := &proto.NewMarketCommitment{
		CommitmentAmount: n.CommitmentAmount.Uint64(),
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

func (*NewMarketConfiguration_LogNormal) isNMCRP() {}

func (n NewMarketConfiguration_LogNormal) rpIntoProto() interface{} {
	return n.IntoProto()
}

func (n NewMarketConfiguration_Simple) IntoProto() *proto.NewMarketConfiguration_Simple {
	return &proto.NewMarketConfiguration_Simple{
		Simple: n.Simple.IntoProto(),
	}
}

func (*NewMarketConfiguration_Simple) isNMCRP() {}

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
}

type InstrumentConfiguration_Future struct {
	Future *FutureProduct
}

type FutureProduct struct {
	Maturity          string
	SettlementAsset   string
	QuoteName         string
	OracleSpec        *v1.OracleSpecConfiguration
	OracleSpecBinding *OracleSpecToFutureBinding
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
		Maturity:          f.Maturity,
		SettlementAsset:   f.SettlementAsset,
		QuoteName:         f.QuoteName,
		OracleSpec:        f.OracleSpec.DeepClone(), // @TODO
		OracleSpecBinding: f.OracleSpecBinding.IntoProto(),
	}
}

func (f FutureProduct) String() string {
	return f.IntoProto().String()
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

func (*NewMarketConfiguration_Continuous) isTradingMode() {}

func (n NewMarketConfiguration_Continuous) tmIntoProto() interface{} {
	return n.IntoProto()
}

type NewMarketConfiguration_Discrete struct {
	Discrete *DiscreteTrading
}

func (n NewMarketConfiguration_Discrete) IntoProto() *proto.NewMarketConfiguration_Discrete {
	return &proto.NewMarketConfiguration_Discrete{
		Discrete: n.Discrete.IntoProto(),
	}
}

func (*NewMarketConfiguration_Discrete) isTradingMode() {}

func (n NewMarketConfiguration_Discrete) tmIntoProto() interface{} {
	return n.IntoProto()
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

func (d DiscreteTrading) IntoProto() *proto.DiscreteTrading {
	return &proto.DiscreteTrading{
		DurationNs: d.DurationNs,
		TickSize:   d.TickSize,
	}
}
