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
	"code.vegaprotocol.io/vega/libs/stringer"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
)

type ProposalTermsNewSpotMarket struct {
	NewSpotMarket *NewSpotMarket
}

func (a ProposalTermsNewSpotMarket) String() string {
	return fmt.Sprintf(
		"newSpotMarket(%s)",
		stringer.ReflectPointerToString(a.NewSpotMarket),
	)
}

func (a ProposalTermsNewSpotMarket) IntoProto() *vegapb.ProposalTerms_NewSpotMarket {
	return &vegapb.ProposalTerms_NewSpotMarket{
		NewSpotMarket: a.NewSpotMarket.IntoProto(),
	}
}

func (a ProposalTermsNewSpotMarket) isPTerm() {}

func (a ProposalTermsNewSpotMarket) oneOfProto() interface{} {
	return a.IntoProto()
}

func (a ProposalTermsNewSpotMarket) GetTermType() ProposalTermsType {
	return ProposalTermsTypeNewSpotMarket
}

func (a ProposalTermsNewSpotMarket) DeepClone() proposalTerm {
	if a.NewSpotMarket == nil {
		return &ProposalTermsNewSpotMarket{}
	}
	return &ProposalTermsNewSpotMarket{
		NewSpotMarket: a.NewSpotMarket.DeepClone(),
	}
}

func NewNewSpotMarketFromProto(p *vegapb.ProposalTerms_NewSpotMarket) (*ProposalTermsNewSpotMarket, error) {
	var newSpotMarket *NewSpotMarket
	if p.NewSpotMarket != nil {
		newSpotMarket = &NewSpotMarket{}

		if p.NewSpotMarket.Changes != nil {
			var err error
			newSpotMarket.Changes, err = NewSpotMarketConfigurationFromProto(p.NewSpotMarket.Changes)
			if err != nil {
				return nil, err
			}
		}
	}

	return &ProposalTermsNewSpotMarket{
		NewSpotMarket: newSpotMarket,
	}, nil
}

type NewSpotMarket struct {
	Changes *NewSpotMarketConfiguration
}

func (n NewSpotMarket) IntoProto() *vegapb.NewSpotMarket {
	var changes *vegapb.NewSpotMarketConfiguration
	if n.Changes != nil {
		changes = n.Changes.IntoProto()
	}
	return &vegapb.NewSpotMarket{
		Changes: changes,
	}
}

func (n NewSpotMarket) DeepClone() *NewSpotMarket {
	cpy := NewSpotMarket{}
	if n.Changes != nil {
		cpy.Changes = n.Changes.DeepClone()
	}
	return &cpy
}

func (n NewSpotMarket) String() string {
	return fmt.Sprintf(
		"changes(%s)",
		stringer.ReflectPointerToString(n.Changes),
	)
}

type NewSpotMarketConfiguration struct {
	Instrument                *InstrumentConfiguration
	DecimalPlaces             uint64
	PositionDecimalPlaces     int64
	Metadata                  []string
	PriceMonitoringParameters *PriceMonitoringParameters
	TargetStakeParameters     *TargetStakeParameters
	RiskParameters            newRiskParams
	SLAParams                 *LiquiditySLAParams

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

func (n NewSpotMarketConfiguration) IntoProto() *vegapb.NewSpotMarketConfiguration {
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
	var targetStakeParameters *vegapb.TargetStakeParameters
	if n.TargetStakeParameters != nil {
		targetStakeParameters = n.TargetStakeParameters.IntoProto()
	}

	r := &vegapb.NewSpotMarketConfiguration{
		Instrument:                instrument,
		DecimalPlaces:             n.DecimalPlaces,
		PositionDecimalPlaces:     n.PositionDecimalPlaces,
		Metadata:                  md,
		PriceMonitoringParameters: priceMonitoring,
		TargetStakeParameters:     targetStakeParameters,
		SlaParams:                 n.SLAParams.IntoProto(),
	}
	switch rp := riskParams.(type) {
	case *vegapb.NewSpotMarketConfiguration_Simple:
		r.RiskParameters = rp
	case *vegapb.NewSpotMarketConfiguration_LogNormal:
		r.RiskParameters = rp
	}
	return r
}

func (n NewSpotMarketConfiguration) DeepClone() *NewSpotMarketConfiguration {
	cpy := &NewSpotMarketConfiguration{
		DecimalPlaces:         n.DecimalPlaces,
		PositionDecimalPlaces: n.PositionDecimalPlaces,
		Metadata:              make([]string, len(n.Metadata)),
		SLAParams:             n.SLAParams.DeepClone(),
	}
	cpy.Metadata = append(cpy.Metadata, n.Metadata...)
	if n.Instrument != nil {
		cpy.Instrument = n.Instrument.DeepClone()
	}
	if n.PriceMonitoringParameters != nil {
		cpy.PriceMonitoringParameters = n.PriceMonitoringParameters.DeepClone()
	}
	if n.TargetStakeParameters != nil {
		cpy.TargetStakeParameters = n.TargetStakeParameters.DeepClone()
	}
	if n.RiskParameters != nil {
		cpy.RiskParameters = n.RiskParameters.DeepClone()
	}
	return cpy
}

func (n NewSpotMarketConfiguration) String() string {
	return fmt.Sprintf(
		"decimalPlaces(%v) positionDecimalPlaces(%v) metadata(%v) instrument(%s) priceMonitoring(%s) targetStakeParameters(%s) risk(%s) slaParams(%s)",
		n.Metadata,
		n.DecimalPlaces,
		n.PositionDecimalPlaces,
		stringer.ReflectPointerToString(n.Instrument),
		stringer.ReflectPointerToString(n.PriceMonitoringParameters),
		stringer.ReflectPointerToString(n.TargetStakeParameters),
		stringer.ReflectPointerToString(n.RiskParameters),
		stringer.ReflectPointerToString(n.SLAParams),
	)
}

func NewSpotMarketConfigurationFromProto(p *vegapb.NewSpotMarketConfiguration) (*NewSpotMarketConfiguration, error) {
	md := make([]string, 0, len(p.Metadata))
	md = append(md, p.Metadata...)

	var err error
	var instrument *InstrumentConfiguration
	if p.Instrument != nil {
		instrument, err = InstrumentConfigurationFromProto(p.Instrument)
		if err != nil {
			return nil, fmt.Errorf("failed to parse instrument configuration: %w", err)
		}
	}

	var priceMonitoring *PriceMonitoringParameters
	if p.PriceMonitoringParameters != nil {
		priceMonitoring = PriceMonitoringParametersFromProto(p.PriceMonitoringParameters)
	}
	targetStakeParams := TargetStakeParametersFromProto(p.TargetStakeParameters)

	r := &NewSpotMarketConfiguration{
		Instrument:                instrument,
		DecimalPlaces:             p.DecimalPlaces,
		PositionDecimalPlaces:     p.PositionDecimalPlaces,
		Metadata:                  md,
		PriceMonitoringParameters: priceMonitoring,
		TargetStakeParameters:     targetStakeParams,
		SLAParams:                 LiquiditySLAParamsFromProto(p.SlaParams),
	}
	if p.RiskParameters != nil {
		switch rp := p.RiskParameters.(type) {
		case *vegapb.NewSpotMarketConfiguration_Simple:
			r.RiskParameters = NewSpotMarketConfigurationSimpleFromProto(rp)
		case *vegapb.NewSpotMarketConfiguration_LogNormal:
			r.RiskParameters = NewSpotMarketConfigurationLogNormalFromProto(rp)
		}
	}
	return r, nil
}

type NewSpotMarketConfigurationSimple struct {
	Simple *SimpleModelParams
}

func (n NewSpotMarketConfigurationSimple) String() string {
	return fmt.Sprintf(
		"simple(%s)",
		stringer.ReflectPointerToString(n.Simple),
	)
}

func (n NewSpotMarketConfigurationSimple) IntoProto() *vegapb.NewSpotMarketConfiguration_Simple {
	return &vegapb.NewSpotMarketConfiguration_Simple{
		Simple: n.Simple.IntoProto(),
	}
}

func (n NewSpotMarketConfigurationSimple) DeepClone() newRiskParams {
	if n.Simple == nil {
		return &NewMarketConfigurationSimple{}
	}
	return &NewMarketConfigurationSimple{
		Simple: n.Simple.DeepClone(),
	}
}

func (n NewSpotMarketConfigurationSimple) newRiskParamsIntoProto() interface{} {
	return n.IntoProto()
}

func NewSpotMarketConfigurationSimpleFromProto(p *vegapb.NewSpotMarketConfiguration_Simple) *NewSpotMarketConfigurationSimple {
	return &NewSpotMarketConfigurationSimple{
		Simple: SimpleModelParamsFromProto(p.Simple),
	}
}

type NewSpotMarketConfigurationLogNormal struct {
	LogNormal *LogNormalRiskModel
}

func (n NewSpotMarketConfigurationLogNormal) IntoProto() *vegapb.NewSpotMarketConfiguration_LogNormal {
	return &vegapb.NewSpotMarketConfiguration_LogNormal{
		LogNormal: n.LogNormal.IntoProto(),
	}
}

func (n NewSpotMarketConfigurationLogNormal) DeepClone() newRiskParams {
	if n.LogNormal == nil {
		return &NewSpotMarketConfigurationLogNormal{}
	}
	return &NewSpotMarketConfigurationLogNormal{
		LogNormal: n.LogNormal.DeepClone(),
	}
}

func (n NewSpotMarketConfigurationLogNormal) newRiskParamsIntoProto() interface{} {
	return n.IntoProto()
}

func (n NewSpotMarketConfigurationLogNormal) String() string {
	return fmt.Sprintf(
		"logNormal(%s)",
		stringer.ReflectPointerToString(n.LogNormal),
	)
}

func NewSpotMarketConfigurationLogNormalFromProto(p *vegapb.NewSpotMarketConfiguration_LogNormal) *NewSpotMarketConfigurationLogNormal {
	return &NewSpotMarketConfigurationLogNormal{
		LogNormal: &LogNormalRiskModel{
			RiskAversionParameter: num.DecimalFromFloat(p.LogNormal.RiskAversionParameter),
			Tau:                   num.DecimalFromFloat(p.LogNormal.Tau),
			Params:                LogNormalParamsFromProto(p.LogNormal.Params),
		},
	}
}

type InstrumentConfigurationSpot struct {
	Spot *SpotProduct
}

func (i InstrumentConfigurationSpot) String() string {
	return fmt.Sprintf(
		"spot(%s)",
		stringer.ReflectPointerToString(i.Spot),
	)
}

func (InstrumentConfigurationSpot) Type() ProductType {
	return ProductTypeSpot
}

func (i InstrumentConfigurationSpot) DeepClone() instrumentConfigurationProduct {
	if i.Spot == nil {
		return &InstrumentConfigurationFuture{}
	}
	return &InstrumentConfigurationSpot{
		Spot: i.Spot.DeepClone(),
	}
}

func (i InstrumentConfigurationSpot) Assets() []string {
	return i.Spot.Assets()
}

func (i InstrumentConfigurationSpot) IntoProto() *vegapb.InstrumentConfiguration_Spot {
	return &vegapb.InstrumentConfiguration_Spot{
		Spot: i.Spot.IntoProto(),
	}
}

func (i InstrumentConfigurationSpot) icpIntoProto() interface{} {
	return i.IntoProto()
}

func (InstrumentConfigurationSpot) isInstrumentConfigurationProduct() {}

type SpotProduct struct {
	Name       string
	BaseAsset  string
	QuoteAsset string
}

func (f SpotProduct) IntoProto() *vegapb.SpotProduct {
	return &vegapb.SpotProduct{
		BaseAsset:  f.BaseAsset,
		QuoteAsset: f.QuoteAsset,
	}
}

func (f SpotProduct) DeepClone() *SpotProduct {
	return &SpotProduct{
		BaseAsset:  f.BaseAsset,
		QuoteAsset: f.QuoteAsset,
	}
}

func (f SpotProduct) String() string {
	return fmt.Sprintf(
		"baseAsset(%s) quoteAsset(%s)",
		f.BaseAsset,
		f.QuoteAsset,
	)
}

func (f SpotProduct) Assets() []string {
	return []string{f.BaseAsset, f.QuoteAsset}
}
