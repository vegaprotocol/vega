package types

import (
	"errors"
	"fmt"
	"strings"

	vegapb "code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/types/num"
)

var ErrInvalidCommitmentAmount = errors.New("invalid commitment amount")

type ProposalTermsNewMarket struct {
	NewMarket *NewMarket
}

func (a ProposalTermsNewMarket) String() string {
	return fmt.Sprintf(
		"newMarket(%s)",
		reflectPointerToString(a.NewMarket),
	)
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

func NewNewMarketFromProto(p *vegapb.ProposalTerms_NewMarket) (*ProposalTermsNewMarket, error) {
	var newMarket *NewMarket
	if p.NewMarket != nil {
		newMarket = &NewMarket{}

		if p.NewMarket.Changes != nil {
			newMarket.Changes = NewMarketConfigurationFromProto(p.NewMarket.Changes)
		}
		if p.NewMarket.LiquidityCommitment != nil {
			var err error
			newMarket.LiquidityCommitment, err = NewMarketCommitmentFromProto(p.NewMarket.LiquidityCommitment)
			if err != nil {
				return nil, err
			}
		}
	}

	return &ProposalTermsNewMarket{
		NewMarket: newMarket,
	}, nil
}

type NewMarket struct {
	Changes             *NewMarketConfiguration
	LiquidityCommitment *NewMarketCommitment
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

func (n NewMarket) String() string {
	return fmt.Sprintf(
		"changes(%s) liquidityCommitment(%s)",
		reflectPointerToString(n.Changes),
		reflectPointerToString(n.LiquidityCommitment),
	)
}

type NewMarketConfiguration struct {
	Instrument                    *InstrumentConfiguration
	DecimalPlaces                 uint64
	PositionDecimalPlaces         uint64
	Metadata                      []string
	PriceMonitoringParameters     *PriceMonitoringParameters
	LiquidityMonitoringParameters *LiquidityMonitoringParameters
	RiskParameters                newRiskParams
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

func (n NewMarketConfiguration) IntoProto() *vegapb.NewMarketConfiguration {
	riskParams := n.RiskParameters.newRiskParamsIntoProto()
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
		PositionDecimalPlaces:         n.PositionDecimalPlaces,
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
		DecimalPlaces:         n.DecimalPlaces,
		PositionDecimalPlaces: n.PositionDecimalPlaces,
		Metadata:              make([]string, len(n.Metadata)),
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
	return cpy
}

func (n NewMarketConfiguration) String() string {
	return fmt.Sprintf(
		"decimalPlaces(%v) positionDecimalPlaces(%v) metadata(%v) instrument(%s) priceMonitoring(%s) liquidityMonitoring(%s) risk(%s)",
		n.Metadata,
		n.DecimalPlaces,
		n.PositionDecimalPlaces,
		reflectPointerToString(n.Instrument),
		reflectPointerToString(n.PriceMonitoringParameters),
		reflectPointerToString(n.LiquidityMonitoringParameters),
		reflectPointerToString(n.RiskParameters),
	)
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
		PositionDecimalPlaces:         p.PositionDecimalPlaces,
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
	return fmt.Sprintf(
		"reference(%s) commitmentAmount(%s) fee(%s) sells(%v) buys(%v)",
		n.Reference,
		uintPointerToString(n.CommitmentAmount),
		n.Fee.String(),
		LiquidityOrders(n.Sells).String(),
		LiquidityOrders(n.Buys).String(),
	)
}

type newRiskParams interface {
	newRiskParamsIntoProto() interface{}
	DeepClone() newRiskParams
	String() string
}

type NewMarketConfigurationSimple struct {
	Simple *SimpleModelParams
}

func (n NewMarketConfigurationSimple) String() string {
	return fmt.Sprintf(
		"simple(%s)",
		reflectPointerToString(n.Simple),
	)
}

func (n NewMarketConfigurationSimple) IntoProto() *vegapb.NewMarketConfiguration_Simple {
	return &vegapb.NewMarketConfiguration_Simple{
		Simple: n.Simple.IntoProto(),
	}
}

func (n NewMarketConfigurationSimple) DeepClone() newRiskParams {
	if n.Simple == nil {
		return &NewMarketConfigurationSimple{}
	}
	return &NewMarketConfigurationSimple{
		Simple: n.Simple.DeepClone(),
	}
}

func (n NewMarketConfigurationSimple) newRiskParamsIntoProto() interface{} {
	return n.IntoProto()
}

func NewMarketConfigurationSimpleFromProto(p *vegapb.NewMarketConfiguration_Simple) *NewMarketConfigurationSimple {
	return &NewMarketConfigurationSimple{
		Simple: SimpleModelParamsFromProto(p.Simple),
	}
}

type NewMarketConfigurationLogNormal struct {
	LogNormal *LogNormalRiskModel
}

func (n NewMarketConfigurationLogNormal) IntoProto() *vegapb.NewMarketConfiguration_LogNormal {
	return &vegapb.NewMarketConfiguration_LogNormal{
		LogNormal: n.LogNormal.IntoProto(),
	}
}

func (n NewMarketConfigurationLogNormal) DeepClone() newRiskParams {
	if n.LogNormal == nil {
		return &NewMarketConfigurationLogNormal{}
	}
	return &NewMarketConfigurationLogNormal{
		LogNormal: n.LogNormal.DeepClone(),
	}
}

func (n NewMarketConfigurationLogNormal) newRiskParamsIntoProto() interface{} {
	return n.IntoProto()
}

func (n NewMarketConfigurationLogNormal) String() string {
	return fmt.Sprintf(
		"logNormal(%s)",
		reflectPointerToString(n.LogNormal),
	)
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

type instrumentConfigurationProduct interface {
	isInstrumentConfigurationProduct()
	icpIntoProto() interface{}
	Asset() string
	DeepClone() instrumentConfigurationProduct
	String() string
}

type InstrumentConfigurationFuture struct {
	Future *FutureProduct
}

func (i InstrumentConfigurationFuture) String() string {
	return fmt.Sprintf(
		"future(%s)",
		reflectPointerToString(i.Future),
	)
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

type InstrumentConfiguration struct {
	Name string
	Code string
	// *InstrumentConfigurationFuture
	Product instrumentConfigurationProduct
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

func (i InstrumentConfiguration) String() string {
	return fmt.Sprintf(
		"name(%s) code(%s) product(%s)",
		i.Name,
		i.Code,
		reflectPointerToString(i.Product),
	)
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
				OracleSpecForSettlementPrice:    OracleSpecConfigurationFromProto(pr.Future.OracleSpecForSettlementPrice),
				OracleSpecForTradingTermination: OracleSpecConfigurationFromProto(pr.Future.OracleSpecForTradingTermination),
				SettlementPriceDecimalPlaces:    pr.Future.SettlementPriceDecimals,
				OracleSpecBinding:               OracleSpecBindingForFutureFromProto(pr.Future.OracleSpecBinding),
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

type FutureProduct struct {
	SettlementAsset                 string
	QuoteName                       string
	OracleSpecForSettlementPrice    *OracleSpecConfiguration
	OracleSpecForTradingTermination *OracleSpecConfiguration
	OracleSpecBinding               *OracleSpecBindingForFuture
	SettlementPriceDecimalPlaces    uint32
}

func (f FutureProduct) IntoProto() *vegapb.FutureProduct {
	return &vegapb.FutureProduct{
		SettlementAsset:                 f.SettlementAsset,
		QuoteName:                       f.QuoteName,
		OracleSpecForSettlementPrice:    f.OracleSpecForSettlementPrice.IntoProto(),
		OracleSpecForTradingTermination: f.OracleSpecForTradingTermination.IntoProto(),
		SettlementPriceDecimals:         f.SettlementPriceDecimalPlaces,
		OracleSpecBinding:               f.OracleSpecBinding.IntoProto(),
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
	return fmt.Sprintf(
		"quote(%s) settlementAsset(%s) settlementPriceDecimalPlaces(%v) oracleSpec(settlementPrice(%s) tradingTermination(%s) binding(%s))",
		f.QuoteName,
		f.SettlementAsset,
		f.SettlementPriceDecimalPlaces,
		reflectPointerToString(f.OracleSpecForSettlementPrice),
		reflectPointerToString(f.OracleSpecForTradingTermination),
		reflectPointerToString(f.OracleSpecBinding),
	)
}

func (f FutureProduct) Asset() string {
	return f.SettlementAsset
}

type MetadataList []string

func (m MetadataList) String() string {
	if m == nil {
		return "[]"
	}
	return "[" + strings.Join(m, ", ") + "]"
}

type NewMarketCommitment struct {
	CommitmentAmount *num.Uint
	Fee              num.Decimal
	Sells            []*LiquidityOrder
	Buys             []*LiquidityOrder
	Reference        string
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

func NewMarketCommitmentFromProto(p *vegapb.NewMarketCommitment) (*NewMarketCommitment, error) {
	fee, err := num.DecimalFromString(p.Fee)
	if err != nil {
		return nil, err
	}
	commitmentAmount, overflowed := num.UintFromString(p.CommitmentAmount, 10)
	if overflowed {
		return nil, ErrInvalidCommitmentAmount
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
