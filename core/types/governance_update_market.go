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
	vegapb "code.vegaprotocol.io/vega/protos/vega"
)

type ProposalTermsUpdateMarket struct {
	UpdateMarket *UpdateMarket
}

func (a ProposalTermsUpdateMarket) String() string {
	return fmt.Sprintf(
		"updateMarket(%s)",
		reflectPointerToString(a.UpdateMarket),
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
		reflectPointerToString(n.Changes),
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
		reflectPointerToString(n.Instrument),
		MetadataList(n.Metadata).String(),
		reflectPointerToString(n.PriceMonitoringParameters),
		reflectPointerToString(n.LiquidityMonitoringParameters),
		reflectPointerToString(n.RiskParameters),
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

	var instrument *UpdateInstrumentConfiguration
	if p.Instrument != nil {
		instrument = UpdateInstrumentConfigurationFromProto(p.Instrument)
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
	}
	return r
}

func (i UpdateInstrumentConfiguration) String() string {
	return fmt.Sprintf(
		"code(%s) product(%s)",
		i.Code,
		reflectPointerToString(i.Product),
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
		reflectPointerToString(i.Future),
	)
}

func (i UpdateInstrumentConfigurationFuture) IntoProto() *vegapb.UpdateInstrumentConfiguration_Future {
	return &vegapb.UpdateInstrumentConfiguration_Future{
		Future: i.Future.IntoProto(),
	}
}

func UpdateInstrumentConfigurationFromProto(p *vegapb.UpdateInstrumentConfiguration) *UpdateInstrumentConfiguration {
	r := &UpdateInstrumentConfiguration{
		Code: p.Code,
	}

	switch pr := p.Product.(type) {
	case *vegapb.UpdateInstrumentConfiguration_Future:
		r.Product = &UpdateInstrumentConfigurationFuture{
			Future: &UpdateFutureProduct{
				QuoteName:                           pr.Future.QuoteName,
				DataSourceSpecForSettlementData:     *DataSourceDefinitionFromProto(pr.Future.DataSourceSpecForSettlementData),
				DataSourceSpecForTradingTermination: *DataSourceDefinitionFromProto(pr.Future.DataSourceSpecForTradingTermination),
				DataSourceSpecBinding:               DataSourceSpecBindingForFutureFromProto(pr.Future.DataSourceSpecBinding),
			},
		}
	}
	return r
}

type UpdateFutureProduct struct {
	QuoteName                           string
	DataSourceSpecForSettlementData     DataSourceDefinition
	DataSourceSpecForTradingTermination DataSourceDefinition
	DataSourceSpecBinding               *DataSourceSpecBindingForFuture
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
		DataSourceSpecForSettlementData:     f.DataSourceSpecForSettlementData.DeepClone(),
		DataSourceSpecForTradingTermination: f.DataSourceSpecForTradingTermination.DeepClone(),
		DataSourceSpecBinding:               f.DataSourceSpecBinding.DeepClone(),
	}
}

func (f UpdateFutureProduct) String() string {
	return fmt.Sprintf(
		"quoteName(%s) oracleSpec(settlementData(%s) tradingTermination(%s) binding(%s))",
		f.QuoteName,
		reflectPointerToString(f.DataSourceSpecForSettlementData),
		reflectPointerToString(f.DataSourceSpecForTradingTermination),
		reflectPointerToString(f.DataSourceSpecBinding),
	)
}

type UpdateMarketConfigurationSimple struct {
	Simple *SimpleModelParams
}

func (n UpdateMarketConfigurationSimple) String() string {
	return fmt.Sprintf(
		"simple(%s)",
		reflectPointerToString(n.Simple),
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
		reflectPointerToString(n.LogNormal),
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
