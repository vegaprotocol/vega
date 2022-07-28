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
	"code.vegaprotocol.io/vega/core/types/num"
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

func UpdateMarketFromProto(p *vegapb.ProposalTerms_UpdateMarket) *ProposalTermsUpdateMarket {
	var updateMarket *UpdateMarket
	if p.UpdateMarket != nil {
		updateMarket = &UpdateMarket{}

		updateMarket.MarketID = p.UpdateMarket.MarketId

		if p.UpdateMarket.Changes != nil {
			updateMarket.Changes = UpdateMarketConfigurationFromProto(p.UpdateMarket.Changes)
		}
	}

	return &ProposalTermsUpdateMarket{
		UpdateMarket: updateMarket,
	}
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
}

func (n UpdateMarketConfiguration) String() string {
	return fmt.Sprintf(
		"instrument(%s) metadata(%v) priceMonitoring(%s) liquidityMonitoring(%s) risk(%s)",
		reflectPointerToString(n.Instrument),
		MetadataList(n.Metadata).String(),
		reflectPointerToString(n.PriceMonitoringParameters),
		reflectPointerToString(n.LiquidityMonitoringParameters),
		reflectPointerToString(n.RiskParameters),
	)
}

func (n UpdateMarketConfiguration) DeepClone() *UpdateMarketConfiguration {
	cpy := &UpdateMarketConfiguration{
		Metadata: make([]string, len(n.Metadata)),
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
	}
	switch rp := riskParams.(type) {
	case *vegapb.UpdateMarketConfiguration_Simple:
		r.RiskParameters = rp
	case *vegapb.UpdateMarketConfiguration_LogNormal:
		r.RiskParameters = rp
	}
	return r
}

func UpdateMarketConfigurationFromProto(p *vegapb.UpdateMarketConfiguration) *UpdateMarketConfiguration {
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
		liquidityMonitoring = LiquidityMonitoringParametersFromProto(p.LiquidityMonitoringParameters)
	}

	r := &UpdateMarketConfiguration{
		Instrument:                    instrument,
		Metadata:                      md,
		PriceMonitoringParameters:     priceMonitoring,
		LiquidityMonitoringParameters: liquidityMonitoring,
	}
	if p.RiskParameters != nil {
		switch rp := p.RiskParameters.(type) {
		case *vegapb.UpdateMarketConfiguration_Simple:
			r.RiskParameters = UpdateMarketConfigurationSimpleFromProto(rp)
		case *vegapb.UpdateMarketConfiguration_LogNormal:
			r.RiskParameters = UpdateMarketConfigurationLogNormalFromProto(rp)
		}
	}
	return r
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
				QuoteName:                       pr.Future.QuoteName,
				OracleSpecForSettlementPrice:    OracleSpecConfigurationFromProto(pr.Future.OracleSpecForSettlementPrice),
				OracleSpecForTradingTermination: OracleSpecConfigurationFromProto(pr.Future.OracleSpecForTradingTermination),
				SettlementPriceDecimals:         pr.Future.SettlementPriceDecimals,
				OracleSpecBinding:               OracleSpecBindingForFutureFromProto(pr.Future.OracleSpecBinding),
			},
		}
	}
	return r
}

type UpdateFutureProduct struct {
	QuoteName                       string
	OracleSpecForSettlementPrice    *OracleSpecConfiguration
	OracleSpecForTradingTermination *OracleSpecConfiguration
	OracleSpecBinding               *OracleSpecBindingForFuture
	SettlementPriceDecimals         uint32
}

func (f UpdateFutureProduct) IntoProto() *vegapb.UpdateFutureProduct {
	return &vegapb.UpdateFutureProduct{
		QuoteName:                       f.QuoteName,
		OracleSpecForSettlementPrice:    f.OracleSpecForSettlementPrice.IntoProto(),
		OracleSpecForTradingTermination: f.OracleSpecForTradingTermination.IntoProto(),
		OracleSpecBinding:               f.OracleSpecBinding.IntoProto(),
		SettlementPriceDecimals:         f.SettlementPriceDecimals,
	}
}

func (f UpdateFutureProduct) DeepClone() *UpdateFutureProduct {
	return &UpdateFutureProduct{
		QuoteName:                       f.QuoteName,
		OracleSpecForSettlementPrice:    f.OracleSpecForSettlementPrice.DeepClone(),
		OracleSpecForTradingTermination: f.OracleSpecForTradingTermination.DeepClone(),
		OracleSpecBinding:               f.OracleSpecBinding.DeepClone(),
		SettlementPriceDecimals:         f.SettlementPriceDecimals,
	}
}

func (f UpdateFutureProduct) String() string {
	return fmt.Sprintf(
		"quoteName(%s) oracleSpec(settlementPrice(%s) tradingTermination(%s) binding(%s))",
		f.QuoteName,
		reflectPointerToString(f.OracleSpecForSettlementPrice),
		reflectPointerToString(f.OracleSpecForTradingTermination),
		reflectPointerToString(f.OracleSpecBinding),
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
