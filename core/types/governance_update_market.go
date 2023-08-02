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

	"code.vegaprotocol.io/vega/core/datasource"
	dsdefinition "code.vegaprotocol.io/vega/core/datasource/definition"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/stringer"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
)

type ProposalTermsUpdateMarket struct {
	UpdateMarket *UpdateMarket
}

func (a ProposalTermsUpdateMarket) String() string {
	return fmt.Sprintf(
		"updateMarket(%s)",
		stringer.ReflectPointerToString(a.UpdateMarket),
	)
}

func (a ProposalTermsUpdateMarket) IntoProto() *vegapb.ProposalTerms_UpdateMarket {
	return &vegapb.ProposalTerms_UpdateMarket{
		UpdateMarket: a.UpdateMarket.IntoProto(),
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

func UpdateMarketFromProto(p *vegapb.ProposalTerms_UpdateMarket) (*ProposalTermsUpdateMarket, error) {
	var updateMarket *UpdateMarket
	if p.UpdateMarket != nil {
		updateMarket = &UpdateMarket{}

		updateMarket.MarketID = p.UpdateMarket.MarketId

		if p.UpdateMarket.Changes != nil {
			var err error
			updateMarket.Changes, err = UpdateMarketConfigurationFromProto(p.UpdateMarket.Changes)
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
		stringer.ReflectPointerToString(n.Changes),
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
	RiskParameters                updateRiskParams
	LpPriceRange                  num.Decimal
	LinearSlippageFactor          num.Decimal
	QuadraticSlippageFactor       num.Decimal
}

func (n UpdateMarketConfiguration) String() string {
	return fmt.Sprintf(
		"instrument(%s) metadata(%v) priceMonitoring(%s) liquidityMonitoring(%s) risk(%s) lpPriceRange(%s) linearSlippageFactor(%s) quadraticSlippageFactor(%s)",
		stringer.ReflectPointerToString(n.Instrument),
		MetadataList(n.Metadata).String(),
		stringer.ReflectPointerToString(n.PriceMonitoringParameters),
		stringer.ReflectPointerToString(n.LiquidityMonitoringParameters),
		stringer.ReflectPointerToString(n.RiskParameters),
		n.LpPriceRange.String(),
		n.LinearSlippageFactor.String(),
		n.QuadraticSlippageFactor.String(),
	)
}

func (n UpdateMarketConfiguration) DeepClone() *UpdateMarketConfiguration {
	cpy := &UpdateMarketConfiguration{
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

	r := &vegapb.UpdateMarketConfiguration{
		Instrument:                    instrument,
		Metadata:                      md,
		PriceMonitoringParameters:     priceMonitoring,
		LiquidityMonitoringParameters: liquidityMonitoring,
		LpPriceRange:                  n.LpPriceRange.String(),
		LinearSlippageFactor:          n.LinearSlippageFactor.String(),
		QuadraticSlippageFactor:       n.QuadraticSlippageFactor.String(),
	}
	switch rp := riskParams.(type) {
	case *vegapb.UpdateMarketConfiguration_Simple:
		r.RiskParameters = rp
	case *vegapb.UpdateMarketConfiguration_LogNormal:
		r.RiskParameters = rp
	}
	return r
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

	r := &UpdateMarketConfiguration{
		Instrument:                    instrument,
		Metadata:                      md,
		PriceMonitoringParameters:     priceMonitoring,
		LiquidityMonitoringParameters: liquidityMonitoring,
		LpPriceRange:                  lppr,
		LinearSlippageFactor:          linearSlippageFactor,
		QuadraticSlippageFactor:       quadraticSlippageFactor,
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
	// *UpdateInstrumentConfigurationFuture
	Product updateInstrumentConfigurationProduct
}

func (i UpdateInstrumentConfiguration) DeepClone() *UpdateInstrumentConfiguration {
	cpy := UpdateInstrumentConfiguration{
		Code: i.Code,
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
		"code(%s) product(%s)",
		i.Code,
		stringer.ReflectPointerToString(i.Product),
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
		stringer.ReflectPointerToString(i.Future),
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
		stringer.ReflectPointerToString(i.Perps),
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

		r.Product = &UpdateInstrumentConfigurationPerps{
			Perps: &UpdatePerpsProduct{
				QuoteName:                           pr.Perpetual.QuoteName,
				MarginFundingFactor:                 marginFundingFactor,
				InterestRate:                        interestRate,
				ClampLowerBound:                     clampLowerBound,
				ClampUpperBound:                     clampUpperBound,
				DataSourceSpecForSettlementData:     *datasource.NewDefinitionWith(settlement),
				DataSourceSpecForSettlementSchedule: *datasource.NewDefinitionWith(settlementSchedule),
				DataSourceSpecBinding:               datasource.SpecBindingForPerpsFromProto(pr.Perpetual.DataSourceSpecBinding),
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
		stringer.ReflectPointerToString(f.DataSourceSpecForSettlementData),
		stringer.ReflectPointerToString(f.DataSourceSpecForTradingTermination),
		stringer.ReflectPointerToString(f.DataSourceSpecBinding),
	)
}

type UpdatePerpsProduct struct {
	QuoteName string

	MarginFundingFactor num.Decimal
	InterestRate        num.Decimal
	ClampLowerBound     num.Decimal
	ClampUpperBound     num.Decimal

	DataSourceSpecForSettlementData     dsdefinition.Definition
	DataSourceSpecForSettlementSchedule dsdefinition.Definition
	DataSourceSpecBinding               *datasource.SpecBindingForPerps
}

func (p UpdatePerpsProduct) IntoProto() *vegapb.UpdatePerpetualProduct {
	return &vegapb.UpdatePerpetualProduct{
		QuoteName:                           p.QuoteName,
		MarginFundingFactor:                 p.MarginFundingFactor.String(),
		InterestRate:                        p.InterestRate.String(),
		ClampLowerBound:                     p.ClampLowerBound.String(),
		ClampUpperBound:                     p.ClampUpperBound.String(),
		DataSourceSpecForSettlementData:     p.DataSourceSpecForSettlementData.IntoProto(),
		DataSourceSpecForSettlementSchedule: p.DataSourceSpecForSettlementSchedule.IntoProto(),
		DataSourceSpecBinding:               p.DataSourceSpecBinding.IntoProto(),
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
		stringer.ReflectPointerToString(p.DataSourceSpecForSettlementData),
		stringer.ReflectPointerToString(p.DataSourceSpecForSettlementSchedule),
		stringer.ReflectPointerToString(p.DataSourceSpecBinding),
	)
}

type UpdateMarketConfigurationSimple struct {
	Simple *SimpleModelParams
}

func (n UpdateMarketConfigurationSimple) String() string {
	return fmt.Sprintf(
		"simple(%s)",
		stringer.ReflectPointerToString(n.Simple),
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
		stringer.ReflectPointerToString(n.LogNormal),
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
