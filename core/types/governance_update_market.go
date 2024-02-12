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

	"code.vegaprotocol.io/vega/core/datasource"
	dsdefinition "code.vegaprotocol.io/vega/core/datasource/definition"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/ptr"
	"code.vegaprotocol.io/vega/libs/stringer"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
)

type ProposalTermsUpdateMarket struct {
	UpdateMarket *UpdateMarket
}

func (a ProposalTermsUpdateMarket) String() string {
	return fmt.Sprintf(
		"updateMarket(%s)",
		stringer.PtrToString(a.UpdateMarket),
	)
}

func (a ProposalTermsUpdateMarket) isPTerm() {}

func (a ProposalTermsUpdateMarket) oneOfSingleProto() vegapb.ProposalOneOffTermChangeType {
	return &vegapb.ProposalTerms_UpdateMarket{
		UpdateMarket: a.UpdateMarket.IntoProto(),
	}
}

func (a ProposalTermsUpdateMarket) oneOfBatchProto() vegapb.ProposalOneOffTermBatchChangeType {
	return &vegapb.BatchProposalTermsChange_UpdateMarket{
		UpdateMarket: a.UpdateMarket.IntoProto(),
	}
}

func (a ProposalTermsUpdateMarket) GetTermType() ProposalTermsType {
	return ProposalTermsTypeUpdateMarket
}

func (a ProposalTermsUpdateMarket) DeepClone() ProposalTerm {
	if a.UpdateMarket == nil {
		return &ProposalTermsUpdateMarket{}
	}
	return &ProposalTermsUpdateMarket{
		UpdateMarket: a.UpdateMarket.DeepClone(),
	}
}

func UpdateMarketFromProto(updateMarketProto *vegapb.UpdateMarket) (*ProposalTermsUpdateMarket, error) {
	var updateMarket *UpdateMarket
	if updateMarketProto != nil {
		updateMarket = &UpdateMarket{}

		updateMarket.MarketID = updateMarketProto.MarketId

		if updateMarketProto.Changes != nil {
			var err error
			updateMarket.Changes, err = UpdateMarketConfigurationFromProto(updateMarketProto.Changes)
			if err != nil {
				return nil, err
			}
		}
	}

	return &ProposalTermsUpdateMarket{
		UpdateMarket: updateMarket,
	}, nil
}

type UpdateMarket struct {
	MarketID string
	Changes  *UpdateMarketConfiguration
}

func (n UpdateMarket) String() string {
	return fmt.Sprintf(
		"marketID(%s) changes(%s)",
		n.MarketID,
		stringer.PtrToString(n.Changes),
	)
}

func (n UpdateMarket) IntoProto() *vegapb.UpdateMarket {
	var changes *vegapb.UpdateMarketConfiguration
	if n.Changes != nil {
		changes = n.Changes.IntoProto()
	}
	return &vegapb.UpdateMarket{
		MarketId: n.MarketID,
		Changes:  changes,
	}
}

func (n UpdateMarket) DeepClone() *UpdateMarket {
	cpy := UpdateMarket{
		MarketID: n.MarketID,
	}
	if n.Changes != nil {
		cpy.Changes = n.Changes.DeepClone()
	}
	return &cpy
}

type updateRiskParams interface {
	updateRiskParamsIntoProto() interface{}
	DeepClone() updateRiskParams
	String() string
}

type UpdateMarketConfiguration struct {
	Instrument                    *UpdateInstrumentConfiguration
	Metadata                      []string
	PriceMonitoringParameters     *PriceMonitoringParameters
	LiquidityMonitoringParameters *LiquidityMonitoringParameters
	LiquiditySLAParameters        *LiquiditySLAParams
	RiskParameters                updateRiskParams
	LinearSlippageFactor          num.Decimal
	QuadraticSlippageFactor       num.Decimal
	LiquidityFeeSettings          *LiquidityFeeSettings
	LiquidationStrategy           *LiquidationStrategy
	MarkPriceConfiguration        *CompositePriceConfiguration
}

func (n UpdateMarketConfiguration) String() string {
	return fmt.Sprintf(
		"instrument(%s) metadata(%v) priceMonitoring(%s) liquidityMonitoring(%s) risk(%s) linearSlippageFactor(%s) quadraticSlippageFactor(%s), markPriceConfiguration(%s)",
		stringer.PtrToString(n.Instrument),
		MetadataList(n.Metadata).String(),
		stringer.PtrToString(n.PriceMonitoringParameters),
		stringer.PtrToString(n.LiquidityMonitoringParameters),
		stringer.ObjToString(n.RiskParameters),
		n.LinearSlippageFactor.String(),
		n.QuadraticSlippageFactor.String(),
		stringer.PtrToString(n.MarkPriceConfiguration),
	)
}

func (n UpdateMarketConfiguration) DeepClone() *UpdateMarketConfiguration {
	cpy := &UpdateMarketConfiguration{
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
	if n.LiquidityFeeSettings != nil {
		cpy.LiquidityFeeSettings = n.LiquidityFeeSettings.DeepClone()
	}
	if n.LiquidationStrategy != nil {
		cpy.LiquidationStrategy = n.LiquidationStrategy.DeepClone()
	}
	if n.MarkPriceConfiguration != nil {
		cpy.MarkPriceConfiguration = n.MarkPriceConfiguration.DeepClone()
	}
	return cpy
}

func (n UpdateMarketConfiguration) IntoProto() *vegapb.UpdateMarketConfiguration {
	riskParams := n.RiskParameters.updateRiskParamsIntoProto()
	md := make([]string, 0, len(n.Metadata))
	md = append(md, n.Metadata...)

	var instrument *vegapb.UpdateInstrumentConfiguration
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

	r := &vegapb.UpdateMarketConfiguration{
		Instrument:                    instrument,
		Metadata:                      md,
		PriceMonitoringParameters:     priceMonitoring,
		LiquidityMonitoringParameters: liquidityMonitoring,
		LiquiditySlaParameters:        liquiditySLAParameters,
		LinearSlippageFactor:          n.LinearSlippageFactor.String(),
		LiquidityFeeSettings:          liquidityFeeSettings,
		LiquidationStrategy:           liqStrat,
		MarkPriceConfiguration:        n.MarkPriceConfiguration.IntoProto(),
	}
	switch rp := riskParams.(type) {
	case *vegapb.UpdateMarketConfiguration_Simple:
		r.RiskParameters = rp
	case *vegapb.UpdateMarketConfiguration_LogNormal:
		r.RiskParameters = rp
	}
	return r
}

func (n UpdateMarketConfiguration) GetPerps() *UpdateInstrumentConfigurationPerps {
	if n.GetProductType() == ProductTypePerps {
		ret, _ := n.Instrument.Product.(*UpdateInstrumentConfigurationPerps)
		return ret
	}
	return nil
}

func (n UpdateMarketConfiguration) GetFuture() *UpdateInstrumentConfigurationFuture {
	if n.GetProductType() == ProductTypeFuture {
		ret, _ := n.Instrument.Product.(*UpdateInstrumentConfigurationFuture)
		return ret
	}
	return nil
}

func (n UpdateMarketConfiguration) GetProductType() ProductType {
	if n.Instrument == nil || n.Instrument.Product == nil {
		return ProductTypeUnspecified
	}
	switch n.Instrument.Product.(type) {
	case *UpdateInstrumentConfigurationFuture:
		return ProductTypeFuture
	case *UpdateInstrumentConfigurationPerps:
		return ProductTypePerps
	}
	return ProductTypeUnspecified // maybe spot?
}

func UpdateMarketConfigurationFromProto(p *vegapb.UpdateMarketConfiguration) (*UpdateMarketConfiguration, error) {
	md := make([]string, 0, len(p.Metadata))
	md = append(md, p.Metadata...)

	var err error
	var instrument *UpdateInstrumentConfiguration
	if p.Instrument != nil {
		instrument, err = UpdateInstrumentConfigurationFromProto(p.Instrument)
		if err != nil {
			return nil, fmt.Errorf("error getting update instrument configuration from proto: %s", err)
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
			return nil, fmt.Errorf("error getting update market configuration from proto: %s", err)
		}
	}

	var liquiditySLAParameters *LiquiditySLAParams
	if p.LiquiditySlaParameters != nil {
		liquiditySLAParameters = LiquiditySLAParamsFromProto(p.LiquiditySlaParameters)
	}

	linearSlippageFactor, err := num.DecimalFromString(p.LinearSlippageFactor)
	if err != nil {
		return nil, fmt.Errorf("error getting new market configuration from proto: %w", err)
	}
	var liqStrat *LiquidationStrategy
	if p.LiquidationStrategy != nil {
		if liqStrat, err = LiquidationStrategyFromProto(p.LiquidationStrategy); err != nil {
			return nil, fmt.Errorf("error getting new market configuration from proto: %w", err)
		}
	}

	r := &UpdateMarketConfiguration{
		Instrument:                    instrument,
		Metadata:                      md,
		PriceMonitoringParameters:     priceMonitoring,
		LiquidityMonitoringParameters: liquidityMonitoring,
		LiquiditySLAParameters:        liquiditySLAParameters,
		LinearSlippageFactor:          linearSlippageFactor,
		LiquidityFeeSettings:          LiquidityFeeSettingsFromProto(p.LiquidityFeeSettings),
		QuadraticSlippageFactor:       num.DecimalZero(),
		LiquidationStrategy:           liqStrat,
		MarkPriceConfiguration:        CompositePriceConfigurationFromProto(p.MarkPriceConfiguration),
	}
	if p.RiskParameters != nil {
		switch rp := p.RiskParameters.(type) {
		case *vegapb.UpdateMarketConfiguration_Simple:
			r.RiskParameters = UpdateMarketConfigurationSimpleFromProto(rp)
		case *vegapb.UpdateMarketConfiguration_LogNormal:
			r.RiskParameters = UpdateMarketConfigurationLogNormalFromProto(rp)
		}
	}
	return r, nil
}

type UpdateInstrumentConfiguration struct {
	Code string
	Name string
	// *UpdateInstrumentConfigurationFuture
	// *UpdateInstrumentConfigurationPerps
	Product updateInstrumentConfigurationProduct
}

func (i UpdateInstrumentConfiguration) DeepClone() *UpdateInstrumentConfiguration {
	cpy := UpdateInstrumentConfiguration{
		Code: i.Code,
		Name: i.Name,
	}
	if i.Product != nil {
		cpy.Product = i.Product.DeepClone()
	}
	return &cpy
}

func (i UpdateInstrumentConfiguration) IntoProto() *vegapb.UpdateInstrumentConfiguration {
	p := i.Product.icpIntoProto()
	r := &vegapb.UpdateInstrumentConfiguration{
		Code: i.Code,
		Name: i.Name,
	}
	switch pr := p.(type) {
	case *vegapb.UpdateInstrumentConfiguration_Future:
		r.Product = pr
	case *vegapb.UpdateInstrumentConfiguration_Perpetual:
		r.Product = pr
	}
	return r
}

func (i UpdateInstrumentConfiguration) String() string {
	return fmt.Sprintf(
		"code(%s) name(%s) product(%s)",
		i.Code,
		i.Name,
		stringer.ObjToString(i.Product),
	)
}

type updateInstrumentConfigurationProduct interface {
	isUpdateInstrumentConfigurationProduct()
	icpIntoProto() interface{}
	DeepClone() updateInstrumentConfigurationProduct
	String() string
}

type UpdateInstrumentConfigurationFuture struct {
	Future *UpdateFutureProduct
}

func (i UpdateInstrumentConfigurationFuture) isUpdateInstrumentConfigurationProduct() {}

func (i UpdateInstrumentConfigurationFuture) icpIntoProto() interface{} {
	return i.IntoProto()
}

func (i UpdateInstrumentConfigurationFuture) DeepClone() updateInstrumentConfigurationProduct {
	if i.Future == nil {
		return &UpdateInstrumentConfigurationFuture{}
	}
	return &UpdateInstrumentConfigurationFuture{
		Future: i.Future.DeepClone(),
	}
}

func (i UpdateInstrumentConfigurationFuture) String() string {
	return fmt.Sprintf(
		"future(%s)",
		stringer.PtrToString(i.Future),
	)
}

func (i UpdateInstrumentConfigurationFuture) IntoProto() *vegapb.UpdateInstrumentConfiguration_Future {
	return &vegapb.UpdateInstrumentConfiguration_Future{
		Future: i.Future.IntoProto(),
	}
}

type UpdateInstrumentConfigurationPerps struct {
	Perps *UpdatePerpsProduct
}

func (i UpdateInstrumentConfigurationPerps) isUpdateInstrumentConfigurationProduct() {}

func (i UpdateInstrumentConfigurationPerps) icpIntoProto() interface{} {
	return i.IntoProto()
}

func (i UpdateInstrumentConfigurationPerps) DeepClone() updateInstrumentConfigurationProduct {
	if i.Perps == nil {
		return &UpdateInstrumentConfigurationPerps{}
	}
	return &UpdateInstrumentConfigurationPerps{
		Perps: i.Perps.DeepClone(),
	}
}

func (i UpdateInstrumentConfigurationPerps) String() string {
	return fmt.Sprintf(
		"perps(%s)",
		stringer.PtrToString(i.Perps),
	)
}

func (i UpdateInstrumentConfigurationPerps) IntoProto() *vegapb.UpdateInstrumentConfiguration_Perpetual {
	return &vegapb.UpdateInstrumentConfiguration_Perpetual{
		Perpetual: i.Perps.IntoProto(),
	}
}

func UpdateInstrumentConfigurationFromProto(p *vegapb.UpdateInstrumentConfiguration) (*UpdateInstrumentConfiguration, error) {
	r := &UpdateInstrumentConfiguration{
		Code: p.Code,
		Name: p.Name,
	}

	switch pr := p.Product.(type) {
	case *vegapb.UpdateInstrumentConfiguration_Future:
		settl, err := datasource.DefinitionFromProto(pr.Future.DataSourceSpecForSettlementData)
		if err != nil {
			return nil, fmt.Errorf("failed to parse settlement data source spec: %w", err)
		}
		term, err := datasource.DefinitionFromProto(pr.Future.DataSourceSpecForTradingTermination)
		if err != nil {
			return nil, fmt.Errorf("failed to parse trading termination data source spec: %w", err)
		}
		r.Product = &UpdateInstrumentConfigurationFuture{
			Future: &UpdateFutureProduct{
				QuoteName:                           pr.Future.QuoteName,
				DataSourceSpecForSettlementData:     *datasource.NewDefinitionWith(settl),
				DataSourceSpecForTradingTermination: *datasource.NewDefinitionWith(term),
				DataSourceSpecBinding:               datasource.SpecBindingForFutureFromProto(pr.Future.DataSourceSpecBinding),
			},
		}
	case *vegapb.UpdateInstrumentConfiguration_Perpetual:
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

		var ipc *CompositePriceConfiguration
		if pr.Perpetual.InternalCompositePriceConfiguration != nil {
			ipc = CompositePriceConfigurationFromProto(pr.Perpetual.InternalCompositePriceConfiguration)
		}

		r.Product = &UpdateInstrumentConfigurationPerps{
			Perps: &UpdatePerpsProduct{
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
				InternalCompositePrice:              ipc,
			},
		}
	}
	return r, nil
}

type UpdateFutureProduct struct {
	QuoteName                           string
	DataSourceSpecForSettlementData     dsdefinition.Definition
	DataSourceSpecForTradingTermination dsdefinition.Definition
	DataSourceSpecBinding               *datasource.SpecBindingForFuture
}

func (f UpdateFutureProduct) IntoProto() *vegapb.UpdateFutureProduct {
	return &vegapb.UpdateFutureProduct{
		QuoteName:                           f.QuoteName,
		DataSourceSpecForSettlementData:     f.DataSourceSpecForSettlementData.IntoProto(),
		DataSourceSpecForTradingTermination: f.DataSourceSpecForTradingTermination.IntoProto(),
		DataSourceSpecBinding:               f.DataSourceSpecBinding.IntoProto(),
	}
}

func (f UpdateFutureProduct) DeepClone() *UpdateFutureProduct {
	return &UpdateFutureProduct{
		QuoteName:                           f.QuoteName,
		DataSourceSpecForSettlementData:     *f.DataSourceSpecForSettlementData.DeepClone().(*dsdefinition.Definition),
		DataSourceSpecForTradingTermination: *f.DataSourceSpecForTradingTermination.DeepClone().(*dsdefinition.Definition),
		DataSourceSpecBinding:               f.DataSourceSpecBinding.DeepClone(),
	}
}

func (f UpdateFutureProduct) String() string {
	return fmt.Sprintf(
		"quoteName(%s) settlementData(%s) tradingTermination(%s) binding(%s)",
		f.QuoteName,
		stringer.ObjToString(f.DataSourceSpecForSettlementData),
		stringer.ObjToString(f.DataSourceSpecForTradingTermination),
		stringer.PtrToString(f.DataSourceSpecBinding),
	)
}

type UpdatePerpsProduct struct {
	QuoteName string

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
	InternalCompositePrice              *CompositePriceConfiguration
}

func (p UpdatePerpsProduct) IntoProto() *vegapb.UpdatePerpetualProduct {
	var scalingFactor, upperBound, lowerBound *string
	if p.FundingRateScalingFactor != nil {
		scalingFactor = ptr.From(p.FundingRateScalingFactor.String())
	}
	if p.FundingRateLowerBound != nil {
		lowerBound = ptr.From(p.FundingRateLowerBound.String())
	}
	if p.FundingRateUpperBound != nil {
		upperBound = ptr.From(p.FundingRateUpperBound.String())
	}

	var ipc *vegapb.CompositePriceConfiguration
	if p.InternalCompositePrice != nil {
		ipc = p.InternalCompositePrice.IntoProto()
	}

	return &vegapb.UpdatePerpetualProduct{
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
		InternalCompositePriceConfiguration: ipc,
	}
}

func (p UpdatePerpsProduct) DeepClone() *UpdatePerpsProduct {
	return &UpdatePerpsProduct{
		QuoteName:                           p.QuoteName,
		MarginFundingFactor:                 p.MarginFundingFactor,
		InterestRate:                        p.InterestRate,
		ClampLowerBound:                     p.ClampLowerBound,
		ClampUpperBound:                     p.ClampLowerBound,
		DataSourceSpecForSettlementData:     *p.DataSourceSpecForSettlementData.DeepClone().(*dsdefinition.Definition),
		DataSourceSpecForSettlementSchedule: *p.DataSourceSpecForSettlementSchedule.DeepClone().(*dsdefinition.Definition),
		DataSourceSpecBinding:               p.DataSourceSpecBinding.DeepClone(),
		InternalCompositePrice:              p.InternalCompositePrice.DeepClone(),
	}
}

func (p UpdatePerpsProduct) String() string {
	return fmt.Sprintf(
		"quote(%s)marginFundingFactor(%s) interestRate(%s) clampLowerBound(%s) clampUpperBound(%s) settlementData(%s) settlementSchedule(%s) binding(%s)",
		p.QuoteName,
		p.MarginFundingFactor.String(),
		p.InterestRate.String(),
		p.ClampLowerBound.String(),
		p.ClampUpperBound.String(),
		stringer.ObjToString(p.DataSourceSpecForSettlementData),
		stringer.ObjToString(p.DataSourceSpecForSettlementSchedule),
		stringer.PtrToString(p.DataSourceSpecBinding),
	)
}

type UpdateMarketConfigurationSimple struct {
	Simple *SimpleModelParams
}

func (n UpdateMarketConfigurationSimple) String() string {
	return fmt.Sprintf(
		"simple(%s)",
		stringer.PtrToString(n.Simple),
	)
}

func (n UpdateMarketConfigurationSimple) updateRiskParamsIntoProto() interface{} {
	return n.IntoProto()
}

func (n UpdateMarketConfigurationSimple) DeepClone() updateRiskParams {
	if n.Simple == nil {
		return &UpdateMarketConfigurationSimple{}
	}
	return &UpdateMarketConfigurationSimple{
		Simple: n.Simple.DeepClone(),
	}
}

func (n UpdateMarketConfigurationSimple) IntoProto() *vegapb.UpdateMarketConfiguration_Simple {
	return &vegapb.UpdateMarketConfiguration_Simple{
		Simple: n.Simple.IntoProto(),
	}
}

func UpdateMarketConfigurationSimpleFromProto(p *vegapb.UpdateMarketConfiguration_Simple) *UpdateMarketConfigurationSimple {
	return &UpdateMarketConfigurationSimple{
		Simple: SimpleModelParamsFromProto(p.Simple),
	}
}

type UpdateMarketConfigurationLogNormal struct {
	LogNormal *LogNormalRiskModel
}

func (n UpdateMarketConfigurationLogNormal) String() string {
	return fmt.Sprintf(
		"logNormal(%s)",
		stringer.PtrToString(n.LogNormal),
	)
}

func (n UpdateMarketConfigurationLogNormal) updateRiskParamsIntoProto() interface{} {
	return n.IntoProto()
}

func (n UpdateMarketConfigurationLogNormal) DeepClone() updateRiskParams {
	if n.LogNormal == nil {
		return &UpdateMarketConfigurationLogNormal{}
	}
	return &UpdateMarketConfigurationLogNormal{
		LogNormal: n.LogNormal.DeepClone(),
	}
}

func (n UpdateMarketConfigurationLogNormal) IntoProto() *vegapb.UpdateMarketConfiguration_LogNormal {
	return &vegapb.UpdateMarketConfiguration_LogNormal{
		LogNormal: n.LogNormal.IntoProto(),
	}
}

func UpdateMarketConfigurationLogNormalFromProto(p *vegapb.UpdateMarketConfiguration_LogNormal) *UpdateMarketConfigurationLogNormal {
	return &UpdateMarketConfigurationLogNormal{
		LogNormal: &LogNormalRiskModel{
			RiskAversionParameter: num.DecimalFromFloat(p.LogNormal.RiskAversionParameter),
			Tau:                   num.DecimalFromFloat(p.LogNormal.Tau),
			Params:                LogNormalParamsFromProto(p.LogNormal.Params),
		},
	}
}
