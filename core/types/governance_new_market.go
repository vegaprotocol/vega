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
	"errors"
	"fmt"
	"strings"

	"code.vegaprotocol.io/vega/core/datasource"
	dsdefinition "code.vegaprotocol.io/vega/core/datasource/definition"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/ptr"
	"code.vegaprotocol.io/vega/libs/stringer"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
)

var (
	ErrInvalidCommitmentAmount = errors.New("invalid commitment amount")
	ErrMissingSlippageFactor   = errors.New("slippage factor not specified")
)

type ProductType int32

const (
	ProductTypeFuture ProductType = iota
	ProductTypeSpot
	ProductTypePerps
)

type ProposalTermsNewMarket struct {
	NewMarket *NewMarket
}

func (a ProposalTermsNewMarket) String() string {
	return fmt.Sprintf(
		"newMarket(%s)",
		stringer.PtrToString(a.NewMarket),
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
	if n.Changes.Successor == nil || len(n.Changes.Successor.ParentID) == 0 {
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

func (n *NewMarket) ClearSuccessor() {
	n.Changes.Successor = nil
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
		stringer.PtrToString(n.Changes),
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
	LiquiditySLAParameters        *LiquiditySLAParams
	LiquidityFeeSettings          *LiquidityFeeSettings

	RiskParameters          newRiskParams
	LinearSlippageFactor    num.Decimal
	QuadraticSlippageFactor num.Decimal
	Successor               *SuccessorConfig
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
	LiquidationStrategy *LiquidationStrategy
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

	var liquiditySLAParameters *vegapb.LiquiditySLAParameters
	if n.LiquiditySLAParameters != nil {
		liquiditySLAParameters = n.LiquiditySLAParameters.IntoProto()
	}

	var liquidityFeeSettings *vegapb.LiquidityFeeSettings
	if n.LiquidityFeeSettings != nil {
		liquidityFeeSettings = n.LiquidityFeeSettings.IntoProto()
	}
	var liqStrat *vegapb.LiquidationStrategy
	if n.LiquidationStrategy != nil {
		liqStrat = n.LiquidationStrategy.IntoProto()
	}

	r := &vegapb.NewMarketConfiguration{
		Instrument:                    instrument,
		DecimalPlaces:                 n.DecimalPlaces,
		PositionDecimalPlaces:         n.PositionDecimalPlaces,
		Metadata:                      md,
		PriceMonitoringParameters:     priceMonitoring,
		LiquidityMonitoringParameters: liquidityMonitoring,
		LiquiditySlaParameters:        liquiditySLAParameters,
		LinearSlippageFactor:          n.LinearSlippageFactor.String(),
		LiquidityFeeSettings:          liquidityFeeSettings,
		LiquidationStrategy:           liqStrat,
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
	if n.LiquiditySLAParameters != nil {
		cpy.LiquiditySLAParameters = n.LiquiditySLAParameters.DeepClone()
	}
	if n.Successor != nil {
		cs := *n.Successor
		cpy.Successor = &cs
	}
	if n.LiquidationStrategy != nil {
		cpy.LiquidationStrategy = n.LiquidationStrategy.DeepClone()
	}
	return cpy
}

func (n NewMarketConfiguration) String() string {
	return fmt.Sprintf(
		"decimalPlaces(%v) positionDecimalPlaces(%v) metadata(%v) instrument(%s) priceMonitoring(%s) liquidityMonitoring(%s) risk(%s) linearSlippageFactor(%s) quadraticSlippageFactor(%s)",
		n.Metadata,
		n.DecimalPlaces,
		n.PositionDecimalPlaces,
		stringer.PtrToString(n.Instrument),
		stringer.PtrToString(n.PriceMonitoringParameters),
		stringer.PtrToString(n.LiquidityMonitoringParameters),
		stringer.ObjToString(n.RiskParameters),
		n.LinearSlippageFactor.String(),
		n.QuadraticSlippageFactor.String(),
	)
}

func (n NewMarketConfiguration) ProductType() ProductType {
	return n.Instrument.Product.Type()
}

func (n NewMarketConfiguration) GetFuture() *InstrumentConfigurationFuture {
	if n.ProductType() == ProductTypeFuture {
		f, _ := n.Instrument.Product.(*InstrumentConfigurationFuture)
		return f
	}
	return nil
}

func (n NewMarketConfiguration) GetPerps() *InstrumentConfigurationPerps {
	if n.ProductType() == ProductTypePerps {
		p, _ := n.Instrument.Product.(*InstrumentConfigurationPerps)
		return p
	}
	return nil
}

func (n NewMarketConfiguration) GetSpot() *InstrumentConfigurationSpot {
	if n.ProductType() == ProductTypeSpot {
		f, _ := n.Instrument.Product.(*InstrumentConfigurationSpot)
		return f
	}
	return nil
}

func NewMarketConfigurationFromProto(p *vegapb.NewMarketConfiguration) (*NewMarketConfiguration, error) {
	md := make([]string, 0, len(p.Metadata))
	md = append(md, p.Metadata...)

	var err error
	var instrument *InstrumentConfiguration
	if p.Instrument != nil {
		instrument, err = InstrumentConfigurationFromProto(p.Instrument)
		if err != nil {
			return nil, fmt.Errorf("error getting new instrument configuration from proto: %w", err)
		}
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

	var liquiditySLAParameters *LiquiditySLAParams
	if p.LiquiditySlaParameters != nil {
		liquiditySLAParameters = LiquiditySLAParamsFromProto(p.LiquiditySlaParameters)
	}

	if len(p.LinearSlippageFactor) == 0 {
		return nil, ErrMissingSlippageFactor
	}
	linearSlippageFactor, err := num.DecimalFromString(p.LinearSlippageFactor)
	if err != nil {
		return nil, fmt.Errorf("error getting new market configuration from proto: %w", err)
	}
	var liqStrat *LiquidationStrategy
	if p.LiquidationStrategy != nil {
		if liqStrat, err = LiquidationStrategyFromProto(p.LiquidationStrategy); err != nil {
			return nil, fmt.Errorf("error getting the liquidation strategy from proto: %w", err)
		}
	}

	r := &NewMarketConfiguration{
		Instrument:                    instrument,
		DecimalPlaces:                 p.DecimalPlaces,
		PositionDecimalPlaces:         p.PositionDecimalPlaces,
		Metadata:                      md,
		PriceMonitoringParameters:     priceMonitoring,
		LiquidityMonitoringParameters: liquidityMonitoring,
		LiquiditySLAParameters:        liquiditySLAParameters,
		LinearSlippageFactor:          linearSlippageFactor,
		LiquidityFeeSettings:          LiquidityFeeSettingsFromProto(p.LiquidityFeeSettings),
		QuadraticSlippageFactor:       num.DecimalZero(),
		LiquidationStrategy:           liqStrat,
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
		s, err := SuccessorConfigFromProto(p.Successor)
		if err != nil {
			return nil, err
		}
		r.Successor = s
	}
	return r, nil
}

func SuccessorConfigFromProto(p *vegapb.SuccessorConfiguration) (*SuccessorConfig, error) {
	// successor config is optional, but make sure that, if provided, it's not set to empty parent market ID
	if len(p.ParentMarketId) == 0 {
		return nil, nil
	}
	f, err := num.DecimalFromString(p.InsurancePoolFraction)
	if err != nil {
		return nil, err
	}
	return &SuccessorConfig{
		ParentID:              p.ParentMarketId,
		InsurancePoolFraction: f,
	}, nil
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
		stringer.PtrToString(n.Simple),
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
		stringer.PtrToString(n.LogNormal),
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
	Type() ProductType
}

type InstrumentConfigurationFuture struct {
	Future *FutureProduct
}

func (i InstrumentConfigurationFuture) String() string {
	return fmt.Sprintf(
		"future(%s)",
		stringer.PtrToString(i.Future),
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

func (InstrumentConfigurationFuture) Type() ProductType {
	return ProductTypeFuture
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

type InstrumentConfigurationPerps struct {
	Perps *PerpsProduct
}

func (i InstrumentConfigurationPerps) String() string {
	return fmt.Sprintf(
		"perps(%s)",
		stringer.PtrToString(i.Perps),
	)
}

func (i InstrumentConfigurationPerps) DeepClone() instrumentConfigurationProduct {
	if i.Perps == nil {
		return &InstrumentConfigurationPerps{}
	}
	return &InstrumentConfigurationPerps{
		Perps: i.Perps.DeepClone(),
	}
}

func (i InstrumentConfigurationPerps) Assets() []string {
	return i.Perps.Assets()
}

func (InstrumentConfigurationPerps) Type() ProductType {
	return ProductTypePerps
}

func (i InstrumentConfigurationPerps) IntoProto() *vegapb.InstrumentConfiguration_Perpetual {
	return &vegapb.InstrumentConfiguration_Perpetual{
		Perpetual: i.Perps.IntoProto(),
	}
}

func (i InstrumentConfigurationPerps) icpIntoProto() interface{} {
	return i.IntoProto()
}

func (InstrumentConfigurationPerps) isInstrumentConfigurationProduct() {}

type InstrumentConfiguration struct {
	Name string
	Code string
	// *InstrumentConfigurationFuture
	// *InstrumentConfigurationSpot
	// *InstrumentConfigurationPerps
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
	case *vegapb.InstrumentConfiguration_Perpetual:
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
		stringer.ObjToString(i.Product),
	)
}

func InstrumentConfigurationFromProto(
	p *vegapb.InstrumentConfiguration,
) (*InstrumentConfiguration, error) {
	r := &InstrumentConfiguration{
		Name: p.Name,
		Code: p.Code,
	}

	switch pr := p.Product.(type) {
	case *vegapb.InstrumentConfiguration_Future:
		settl, err := datasource.DefinitionFromProto(pr.Future.DataSourceSpecForSettlementData)
		if err != nil {
			return nil, fmt.Errorf("failed to parse settlement data source spec: %w", err)
		}

		term, err := datasource.DefinitionFromProto(pr.Future.DataSourceSpecForTradingTermination)
		if err != nil {
			return nil, fmt.Errorf("failed to parse trading termination data source spec: %w", err)
		}
		r.Product = &InstrumentConfigurationFuture{
			Future: &FutureProduct{
				SettlementAsset:                     pr.Future.SettlementAsset,
				QuoteName:                           pr.Future.QuoteName,
				DataSourceSpecForSettlementData:     *datasource.NewDefinitionWith(settl),
				DataSourceSpecForTradingTermination: *datasource.NewDefinitionWith(term),
				DataSourceSpecBinding:               datasource.SpecBindingForFutureFromProto(pr.Future.DataSourceSpecBinding),
			},
		}
	case *vegapb.InstrumentConfiguration_Perpetual:
		settlement, err := datasource.DefinitionFromProto(pr.Perpetual.DataSourceSpecForSettlementData)
		if err != nil {
			return nil, fmt.Errorf("failed to parse settlement data source spec: %w", err)
		}

		settlementSchedule, err := datasource.DefinitionFromProto(pr.Perpetual.DataSourceSpecForSettlementSchedule)
		if err != nil {
			return nil, fmt.Errorf("failed to parse settlement schedule data source spec: %w", err)
		}

		var marginFundingFactor, interestRate, clampLowerBound, clampUpperBound num.Decimal
		if marginFundingFactor, err = num.DecimalFromString(pr.Perpetual.MarginFundingFactor); err != nil {
			return nil, fmt.Errorf("failed to parse margin funding factor: %w", err)
		}
		if interestRate, err = num.DecimalFromString(pr.Perpetual.InterestRate); err != nil {
			return nil, fmt.Errorf("failed to parse interest rate: %w", err)
		}
		if clampLowerBound, err = num.DecimalFromString(pr.Perpetual.ClampLowerBound); err != nil {
			return nil, fmt.Errorf("failed to parse clamp lower bound: %w", err)
		}
		if clampUpperBound, err = num.DecimalFromString(pr.Perpetual.ClampUpperBound); err != nil {
			return nil, fmt.Errorf("failed to parse clamp upper bound: %w", err)
		}

		var scalingFactor, lowerBound, upperBound *num.Decimal
		if pr.Perpetual.FundingRateScalingFactor != nil {
			d, err := num.DecimalFromString(*pr.Perpetual.FundingRateScalingFactor)
			if err != nil {
				return nil, fmt.Errorf("failed to parse funding rate scaling factor: %w", err)
			}
			scalingFactor = &d
		}

		if pr.Perpetual.FundingRateLowerBound != nil {
			d, err := num.DecimalFromString(*pr.Perpetual.FundingRateLowerBound)
			if err != nil {
				return nil, fmt.Errorf("failed to parse funding rate lower bound: %w", err)
			}
			lowerBound = &d
		}

		if pr.Perpetual.FundingRateUpperBound != nil {
			d, err := num.DecimalFromString(*pr.Perpetual.FundingRateUpperBound)
			if err != nil {
				return nil, fmt.Errorf("failed to parse funding rate lower bound: %w", err)
			}
			upperBound = &d
		}

		r.Product = &InstrumentConfigurationPerps{
			Perps: &PerpsProduct{
				SettlementAsset:                     pr.Perpetual.SettlementAsset,
				QuoteName:                           pr.Perpetual.QuoteName,
				MarginFundingFactor:                 marginFundingFactor,
				InterestRate:                        interestRate,
				ClampLowerBound:                     clampLowerBound,
				ClampUpperBound:                     clampUpperBound,
				FundingRateScalingFactor:            scalingFactor,
				FundingRateLowerBound:               lowerBound,
				FundingRateUpperBound:               upperBound,
				DataSourceSpecForSettlementData:     *datasource.NewDefinitionWith(settlement),
				DataSourceSpecForSettlementSchedule: *datasource.NewDefinitionWith(settlementSchedule),
				DataSourceSpecBinding:               datasource.SpecBindingForPerpsFromProto(pr.Perpetual.DataSourceSpecBinding),
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
	return r, nil
}

type FutureProduct struct {
	SettlementAsset                     string
	QuoteName                           string
	DataSourceSpecForSettlementData     dsdefinition.Definition
	DataSourceSpecForTradingTermination dsdefinition.Definition
	DataSourceSpecBinding               *datasource.SpecBindingForFuture
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
	return &FutureProduct{
		SettlementAsset:                     f.SettlementAsset,
		QuoteName:                           f.QuoteName,
		DataSourceSpecForSettlementData:     *f.DataSourceSpecForSettlementData.DeepClone().(*dsdefinition.Definition),
		DataSourceSpecForTradingTermination: *f.DataSourceSpecForTradingTermination.DeepClone().(*dsdefinition.Definition),
		DataSourceSpecBinding:               f.DataSourceSpecBinding.DeepClone(),
	}
}

func (f FutureProduct) String() string {
	return fmt.Sprintf(
		"quote(%s) settlementAsset(%s) settlementData(%s) tradingTermination(%s) binding(%s)",
		f.QuoteName,
		f.SettlementAsset,
		stringer.ObjToString(f.DataSourceSpecForSettlementData),
		stringer.ObjToString(f.DataSourceSpecForTradingTermination),
		stringer.PtrToString(f.DataSourceSpecBinding),
	)
}

func (f FutureProduct) Assets() []string {
	return []string{f.SettlementAsset}
}

type PerpsProduct struct {
	SettlementAsset string
	QuoteName       string

	MarginFundingFactor num.Decimal
	InterestRate        num.Decimal
	ClampLowerBound     num.Decimal
	ClampUpperBound     num.Decimal

	FundingRateScalingFactor *num.Decimal
	FundingRateLowerBound    *num.Decimal
	FundingRateUpperBound    *num.Decimal

	DataSourceSpecForSettlementData     dsdefinition.Definition
	DataSourceSpecForSettlementSchedule dsdefinition.Definition
	DataSourceSpecBinding               *datasource.SpecBindingForPerps
}

func (p PerpsProduct) IntoProto() *vegapb.PerpetualProduct {
	var scalingFactor *string
	if p.FundingRateScalingFactor != nil {
		scalingFactor = ptr.From(p.FundingRateScalingFactor.String())
	}

	var upperBound *string
	if p.FundingRateUpperBound != nil {
		upperBound = ptr.From(p.FundingRateUpperBound.String())
	}

	var lowerBound *string
	if p.FundingRateLowerBound != nil {
		lowerBound = ptr.From(p.FundingRateLowerBound.String())
	}

	return &vegapb.PerpetualProduct{
		SettlementAsset:                     p.SettlementAsset,
		QuoteName:                           p.QuoteName,
		MarginFundingFactor:                 p.MarginFundingFactor.String(),
		InterestRate:                        p.InterestRate.String(),
		ClampLowerBound:                     p.ClampLowerBound.String(),
		ClampUpperBound:                     p.ClampUpperBound.String(),
		FundingRateScalingFactor:            scalingFactor,
		FundingRateLowerBound:               lowerBound,
		FundingRateUpperBound:               upperBound,
		DataSourceSpecForSettlementData:     p.DataSourceSpecForSettlementData.IntoProto(),
		DataSourceSpecForSettlementSchedule: p.DataSourceSpecForSettlementSchedule.IntoProto(),
		DataSourceSpecBinding:               p.DataSourceSpecBinding.IntoProto(),
	}
}

func (p PerpsProduct) DeepClone() *PerpsProduct {
	return &PerpsProduct{
		SettlementAsset:                     p.SettlementAsset,
		QuoteName:                           p.QuoteName,
		MarginFundingFactor:                 p.MarginFundingFactor,
		InterestRate:                        p.InterestRate,
		ClampLowerBound:                     p.ClampLowerBound,
		ClampUpperBound:                     p.ClampUpperBound,
		FundingRateScalingFactor:            p.FundingRateScalingFactor,
		FundingRateLowerBound:               p.FundingRateLowerBound,
		FundingRateUpperBound:               p.FundingRateUpperBound,
		DataSourceSpecForSettlementData:     *p.DataSourceSpecForSettlementData.DeepClone().(*dsdefinition.Definition),
		DataSourceSpecForSettlementSchedule: *p.DataSourceSpecForSettlementSchedule.DeepClone().(*dsdefinition.Definition),
		DataSourceSpecBinding:               p.DataSourceSpecBinding.DeepClone(),
	}
}

func (p PerpsProduct) String() string {
	return fmt.Sprintf(
		"quote(%s) settlementAsset(%s) marginFundingFactor(%s) interestRate(%s) clampLowerBound(%s) clampUpperBound(%s) settlementData(%s) settlementSchedule(%s) binding(%s)",
		p.QuoteName,
		p.SettlementAsset,
		p.MarginFundingFactor.String(),
		p.InterestRate.String(),
		p.ClampLowerBound.String(),
		p.ClampUpperBound.String(),
		stringer.ObjToString(p.DataSourceSpecForSettlementData),
		stringer.ObjToString(p.DataSourceSpecForSettlementSchedule),
		stringer.PtrToString(p.DataSourceSpecBinding),
	)
}

func (p PerpsProduct) Assets() []string {
	return []string{p.SettlementAsset}
}

type MetadataList []string

func (m MetadataList) String() string {
	if m == nil {
		return "[]"
	}
	return "[" + strings.Join(m, ", ") + "]"
}
