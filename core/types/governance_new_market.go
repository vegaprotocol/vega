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
	"errors"
	"fmt"
	"strings"

	"code.vegaprotocol.io/vega/libs/num"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
)

var (
	ErrInvalidCommitmentAmount = errors.New("invalid commitment amount")
	ErrMissingSlippageFactor   = errors.New("slippage factor not specified")
)

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
			var err error
			newMarket.Changes, err = NewMarketConfigurationFromProto(p.NewMarket.Changes)
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
	Changes *NewMarketConfiguration
}

func (n NewMarket) ParentMarketID() (string, bool) {
	if n.Changes.Successor == nil {
		return "", false
	}
	return n.Changes.Successor.ParentID, true
}

func (n NewMarket) Successor() *SuccessorConfig {
	if n.Changes.Successor == nil {
		return nil
	}
	cpy := *n.Changes.Successor
	return &cpy
}

func (n NewMarket) IntoProto() *vegapb.NewMarket {
	var changes *vegapb.NewMarketConfiguration
	if n.Changes != nil {
		changes = n.Changes.IntoProto()
	}
	return &vegapb.NewMarket{
		Changes: changes,
	}
}

func (n NewMarket) DeepClone() *NewMarket {
	cpy := NewMarket{}
	if n.Changes != nil {
		cpy.Changes = n.Changes.DeepClone()
	}
	return &cpy
}

func (n NewMarket) String() string {
	return fmt.Sprintf(
		"changes(%s)",
		reflectPointerToString(n.Changes),
	)
}

type SuccessorConfig struct {
	ParentID              string
	InsurancePoolFraction num.Decimal
}

type NewMarketConfiguration struct {
	Instrument                    *InstrumentConfiguration
	DecimalPlaces                 uint64
	PositionDecimalPlaces         int64
	Metadata                      []string
	PriceMonitoringParameters     *PriceMonitoringParameters
	LiquidityMonitoringParameters *LiquidityMonitoringParameters
	RiskParameters                newRiskParams
	LpPriceRange                  num.Decimal
	LinearSlippageFactor          num.Decimal
	QuadraticSlippageFactor       num.Decimal
	Successor                     *SuccessorConfig
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
		LpPriceRange:                  n.LpPriceRange.String(),
		LinearSlippageFactor:          n.LinearSlippageFactor.String(),
		QuadraticSlippageFactor:       n.QuadraticSlippageFactor.String(),
	}
	if n.Successor != nil {
		r.Successor = n.Successor.IntoProto()
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
		DecimalPlaces:           n.DecimalPlaces,
		PositionDecimalPlaces:   n.PositionDecimalPlaces,
		Metadata:                make([]string, len(n.Metadata)),
		LpPriceRange:            n.LpPriceRange.Copy(),
		LinearSlippageFactor:    n.LinearSlippageFactor.Copy(),
		QuadraticSlippageFactor: n.QuadraticSlippageFactor.Copy(),
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
	if n.Successor != nil {
		cs := *n.Successor
		cpy.Successor = &cs
	}
	return cpy
}

func (n NewMarketConfiguration) String() string {
	return fmt.Sprintf(
		"decimalPlaces(%v) positionDecimalPlaces(%v) metadata(%v) instrument(%s) priceMonitoring(%s) liquidityMonitoring(%s) risk(%s) lpPriceRange(%s) linearSlippageFactor(%s) quadraticSlippageFactor(%s)",
		n.Metadata,
		n.DecimalPlaces,
		n.PositionDecimalPlaces,
		reflectPointerToString(n.Instrument),
		reflectPointerToString(n.PriceMonitoringParameters),
		reflectPointerToString(n.LiquidityMonitoringParameters),
		reflectPointerToString(n.RiskParameters),
		n.LpPriceRange.String(),
		n.LinearSlippageFactor.String(),
		n.QuadraticSlippageFactor.String(),
	)
}

func NewMarketConfigurationFromProto(p *vegapb.NewMarketConfiguration) (*NewMarketConfiguration, error) {
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
		var err error
		liquidityMonitoring, err = LiquidityMonitoringParametersFromProto(p.LiquidityMonitoringParameters)
		if err != nil {
			return nil, fmt.Errorf("error getting new market configuration from proto: %w", err)
		}
	}
	lppr, _ := num.DecimalFromString(p.LpPriceRange)

	if len(p.LinearSlippageFactor) == 0 || len(p.QuadraticSlippageFactor) == 0 {
		return nil, ErrMissingSlippageFactor
	}
	linearSlippageFactor, err := num.DecimalFromString(p.LinearSlippageFactor)
	if err != nil {
		return nil, fmt.Errorf("error getting new market configuration from proto: %w", err)
	}
	quadraticSlippageFactor, err := num.DecimalFromString(p.QuadraticSlippageFactor)
	if err != nil {
		return nil, fmt.Errorf("error getting new market configuration from proto: %w", err)
	}

	r := &NewMarketConfiguration{
		Instrument:                    instrument,
		DecimalPlaces:                 p.DecimalPlaces,
		PositionDecimalPlaces:         p.PositionDecimalPlaces,
		Metadata:                      md,
		PriceMonitoringParameters:     priceMonitoring,
		LiquidityMonitoringParameters: liquidityMonitoring,
		LpPriceRange:                  lppr,
		LinearSlippageFactor:          linearSlippageFactor,
		QuadraticSlippageFactor:       quadraticSlippageFactor,
	}
	if p.RiskParameters != nil {
		switch rp := p.RiskParameters.(type) {
		case *vegapb.NewMarketConfiguration_Simple:
			r.RiskParameters = NewMarketConfigurationSimpleFromProto(rp)
		case *vegapb.NewMarketConfiguration_LogNormal:
			r.RiskParameters = NewMarketConfigurationLogNormalFromProto(rp)
		}
	}
	if p.Successor != nil {
		r.Successor = SuccessorConfigFromProto(p.Successor)
	}
	return r, nil
}

func SuccessorConfigFromProto(p *vegapb.SuccessorConfiguration) *SuccessorConfig {
	f, _ := num.DecimalFromString(p.InsurancePoolFraction)
	return &SuccessorConfig{
		ParentID:              p.ParentMarketId,
		InsurancePoolFraction: f,
	}
}

func (s *SuccessorConfig) IntoProto() *vegapb.SuccessorConfiguration {
	return &vegapb.SuccessorConfiguration{
		ParentMarketId:        s.ParentID,
		InsurancePoolFraction: s.InsurancePoolFraction.String(),
	}
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
	Assets() []string
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

func (i InstrumentConfigurationFuture) Assets() []string {
	return i.Future.Assets()
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
	case *vegapb.InstrumentConfiguration_Spot:
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
				SettlementAsset:                     pr.Future.SettlementAsset,
				QuoteName:                           pr.Future.QuoteName,
				DataSourceSpecForSettlementData:     *DataSourceDefinitionFromProto(pr.Future.DataSourceSpecForSettlementData),
				DataSourceSpecForTradingTermination: *DataSourceDefinitionFromProto(pr.Future.DataSourceSpecForTradingTermination),
				DataSourceSpecBinding:               DataSourceSpecBindingForFutureFromProto(pr.Future.DataSourceSpecBinding),
			},
		}
	case *vegapb.InstrumentConfiguration_Spot:
		r.Product = &InstrumentConfigurationSpot{
			Spot: &SpotProduct{
				Name:       pr.Spot.Name,
				BaseAsset:  pr.Spot.BaseAsset,
				QuoteAsset: pr.Spot.QuoteAsset,
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
	SettlementAsset                     string
	QuoteName                           string
	DataSourceSpecForSettlementData     DataSourceDefinition
	DataSourceSpecForTradingTermination DataSourceDefinition
	DataSourceSpecBinding               *DataSourceSpecBindingForFuture
}

func (f FutureProduct) IntoProto() *vegapb.FutureProduct {
	return &vegapb.FutureProduct{
		SettlementAsset:                     f.SettlementAsset,
		QuoteName:                           f.QuoteName,
		DataSourceSpecForSettlementData:     f.DataSourceSpecForSettlementData.IntoProto(),
		DataSourceSpecForTradingTermination: f.DataSourceSpecForTradingTermination.IntoProto(),
		DataSourceSpecBinding:               f.DataSourceSpecBinding.IntoProto(),
	}
}

func (f FutureProduct) DeepClone() *FutureProduct {
	settlData := f.DataSourceSpecForSettlementData.DeepClone()
	termData := f.DataSourceSpecForTradingTermination.DeepClone()
	return &FutureProduct{
		SettlementAsset:                     f.SettlementAsset,
		QuoteName:                           f.QuoteName,
		DataSourceSpecForSettlementData:     settlData,
		DataSourceSpecForTradingTermination: termData,
		DataSourceSpecBinding:               f.DataSourceSpecBinding.DeepClone(),
	}
}

func (f FutureProduct) String() string {
	return fmt.Sprintf(
		"quote(%s) settlementAsset(%s) settlementData(%s) tradingTermination(%s) binding(%s)",
		f.QuoteName,
		f.SettlementAsset,
		reflectPointerToString(f.DataSourceSpecForSettlementData),
		reflectPointerToString(f.DataSourceSpecForTradingTermination),
		reflectPointerToString(f.DataSourceSpecBinding),
	)
}

func (f FutureProduct) Assets() []string {
	return []string{f.SettlementAsset}
}

type MetadataList []string

func (m MetadataList) String() string {
	if m == nil {
		return "[]"
	}
	return "[" + strings.Join(m, ", ") + "]"
}
