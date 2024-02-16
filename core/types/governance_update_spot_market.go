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

	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/stringer"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
)

type ProposalTermsUpdateSpotMarket struct {
	BatchProposalID    string
	EnactmentTimestamp int64
	UpdateSpotMarket   *UpdateSpotMarket
}

func (a ProposalTermsUpdateSpotMarket) String() string {
	return fmt.Sprintf(
		"updateSpotMarket(%s)",
		stringer.PtrToString(a.UpdateSpotMarket),
	)
}

func (a ProposalTermsUpdateSpotMarket) IntoProto() *vegapb.ProposalTerms_UpdateSpotMarket {
	return &vegapb.ProposalTerms_UpdateSpotMarket{
		UpdateSpotMarket: a.UpdateSpotMarket.IntoProto(),
	}
}

func (a ProposalTermsUpdateSpotMarket) isPTerm() {}

func (a ProposalTermsUpdateSpotMarket) oneOfSingleProto() vegapb.ProposalOneOffTermChangeType {
	return &vegapb.ProposalTerms_UpdateSpotMarket{
		UpdateSpotMarket: a.UpdateSpotMarket.IntoProto(),
	}
}

func (a ProposalTermsUpdateSpotMarket) oneOfBatchProto() vegapb.ProposalOneOffTermBatchChangeType {
	return &vegapb.BatchProposalTermsChange_UpdateSpotMarket{
		UpdateSpotMarket: a.UpdateSpotMarket.IntoProto(),
	}
}

func (a ProposalTermsUpdateSpotMarket) GetTermType() ProposalTermsType {
	return ProposalTermsTypeUpdateSpotMarket
}

func (a ProposalTermsUpdateSpotMarket) DeepClone() ProposalTerm {
	if a.UpdateSpotMarket == nil {
		return &ProposalTermsUpdateSpotMarket{}
	}
	return &ProposalTermsUpdateSpotMarket{
		UpdateSpotMarket: a.UpdateSpotMarket.DeepClone(),
	}
}

func UpdateSpotMarketFromProto(updateSpotMarketProto *vegapb.UpdateSpotMarket) (*ProposalTermsUpdateSpotMarket, error) {
	var updateSpotMarket *UpdateSpotMarket
	if updateSpotMarketProto != nil {
		updateSpotMarket = &UpdateSpotMarket{}
		updateSpotMarket.MarketID = updateSpotMarketProto.MarketId
		if updateSpotMarketProto.Changes != nil {
			var err error
			updateSpotMarket.Changes, err = UpdateSpotMarketConfigurationFromProto(updateSpotMarketProto.Changes)
			if err != nil {
				return nil, err
			}
		}
	}
	return &ProposalTermsUpdateSpotMarket{
		UpdateSpotMarket: updateSpotMarket,
	}, nil
}

type UpdateSpotMarket struct {
	MarketID string
	Changes  *UpdateSpotMarketConfiguration
}

func (n UpdateSpotMarket) String() string {
	return fmt.Sprintf(
		"marketID(%s) changes(%s)",
		n.MarketID,
		stringer.PtrToString(n.Changes),
	)
}

func (n UpdateSpotMarket) IntoProto() *vegapb.UpdateSpotMarket {
	var changes *vegapb.UpdateSpotMarketConfiguration
	if n.Changes != nil {
		changes = n.Changes.IntoProto()
	}
	return &vegapb.UpdateSpotMarket{
		MarketId: n.MarketID,
		Changes:  changes,
	}
}

func (n UpdateSpotMarket) DeepClone() *UpdateSpotMarket {
	cpy := UpdateSpotMarket{
		MarketID: n.MarketID,
	}
	if n.Changes != nil {
		cpy.Changes = n.Changes.DeepClone()
	}
	return &cpy
}

type UpdateSpotMarketConfiguration struct {
	Metadata                  []string
	PriceMonitoringParameters *PriceMonitoringParameters
	TargetStakeParameters     *TargetStakeParameters
	RiskParameters            updateRiskParams
	SLAParams                 *LiquiditySLAParams
}

func (n UpdateSpotMarketConfiguration) String() string {
	return fmt.Sprintf(
		"metadata(%v) priceMonitoring(%s) targetStakeParameters(%s) risk(%s) slaParams(%s)",
		MetadataList(n.Metadata).String(),
		stringer.PtrToString(n.PriceMonitoringParameters),
		stringer.PtrToString(n.TargetStakeParameters),
		stringer.ObjToString(n.RiskParameters),
		stringer.PtrToString(n.SLAParams),
	)
}

func (n UpdateSpotMarketConfiguration) DeepClone() *UpdateSpotMarketConfiguration {
	cpy := &UpdateSpotMarketConfiguration{
		Metadata:  make([]string, len(n.Metadata)),
		SLAParams: n.SLAParams.DeepClone(),
	}
	cpy.Metadata = append(cpy.Metadata, n.Metadata...)
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

func (n UpdateSpotMarketConfiguration) IntoProto() *vegapb.UpdateSpotMarketConfiguration {
	riskParams := n.RiskParameters.updateRiskParamsIntoProto()
	md := make([]string, 0, len(n.Metadata))
	md = append(md, n.Metadata...)

	var priceMonitoring *vegapb.PriceMonitoringParameters
	if n.PriceMonitoringParameters != nil {
		priceMonitoring = n.PriceMonitoringParameters.IntoProto()
	}
	targetStakeParameters := n.TargetStakeParameters.IntoProto()

	r := &vegapb.UpdateSpotMarketConfiguration{
		Metadata:                  md,
		PriceMonitoringParameters: priceMonitoring,
		TargetStakeParameters:     targetStakeParameters,
		SlaParams:                 n.SLAParams.IntoProto(),
	}
	switch rp := riskParams.(type) {
	case *vegapb.UpdateSpotMarketConfiguration_Simple:
		r.RiskParameters = rp
	case *vegapb.UpdateSpotMarketConfiguration_LogNormal:
		r.RiskParameters = rp
	}
	return r
}

func UpdateSpotMarketConfigurationFromProto(p *vegapb.UpdateSpotMarketConfiguration) (*UpdateSpotMarketConfiguration, error) {
	md := make([]string, 0, len(p.Metadata))
	md = append(md, p.Metadata...)
	var priceMonitoring *PriceMonitoringParameters
	if p.PriceMonitoringParameters != nil {
		priceMonitoring = PriceMonitoringParametersFromProto(p.PriceMonitoringParameters)
	}
	targetStakeParameters := TargetStakeParametersFromProto(p.TargetStakeParameters)

	var slaParams *LiquiditySLAParams
	if p.SlaParams != nil {
		slaParams = LiquiditySLAParamsFromProto(p.SlaParams)
	}

	r := &UpdateSpotMarketConfiguration{
		Metadata:                  md,
		PriceMonitoringParameters: priceMonitoring,
		TargetStakeParameters:     targetStakeParameters,
		SLAParams:                 slaParams,
	}
	if p.RiskParameters != nil {
		switch rp := p.RiskParameters.(type) {
		case *vegapb.UpdateSpotMarketConfiguration_Simple:
			r.RiskParameters = UpdateSpotMarketConfigurationSimpleFromProto(rp)
		case *vegapb.UpdateSpotMarketConfiguration_LogNormal:
			r.RiskParameters = UpdateSpotMarketConfigurationLogNormalFromProto(rp)
		}
	}
	return r, nil
}

type UpdateSpotMarketConfigurationSimple struct {
	Simple *SimpleModelParams
}

func (n UpdateSpotMarketConfigurationSimple) String() string {
	return fmt.Sprintf(
		"simple(%s)",
		stringer.PtrToString(n.Simple),
	)
}

func (n UpdateSpotMarketConfigurationSimple) updateRiskParamsIntoProto() interface{} {
	return n.IntoProto()
}

func (n UpdateSpotMarketConfigurationSimple) DeepClone() updateRiskParams {
	if n.Simple == nil {
		return &UpdateSpotMarketConfigurationSimple{}
	}
	return &UpdateSpotMarketConfigurationSimple{
		Simple: n.Simple.DeepClone(),
	}
}

func (n UpdateSpotMarketConfigurationSimple) IntoProto() *vegapb.UpdateSpotMarketConfiguration_Simple {
	return &vegapb.UpdateSpotMarketConfiguration_Simple{
		Simple: n.Simple.IntoProto(),
	}
}

func UpdateSpotMarketConfigurationSimpleFromProto(p *vegapb.UpdateSpotMarketConfiguration_Simple) *UpdateSpotMarketConfigurationSimple {
	return &UpdateSpotMarketConfigurationSimple{
		Simple: SimpleModelParamsFromProto(p.Simple),
	}
}

type UpdateSpotMarketConfigurationLogNormal struct {
	LogNormal *LogNormalRiskModel
}

func (n UpdateSpotMarketConfigurationLogNormal) String() string {
	return fmt.Sprintf(
		"logNormal(%s)",
		stringer.PtrToString(n.LogNormal),
	)
}

func (n UpdateSpotMarketConfigurationLogNormal) updateRiskParamsIntoProto() interface{} {
	return n.IntoProto()
}

func (n UpdateSpotMarketConfigurationLogNormal) DeepClone() updateRiskParams {
	if n.LogNormal == nil {
		return &UpdateSpotMarketConfigurationLogNormal{}
	}
	return &UpdateSpotMarketConfigurationLogNormal{
		LogNormal: n.LogNormal.DeepClone(),
	}
}

func (n UpdateSpotMarketConfigurationLogNormal) IntoProto() *vegapb.UpdateSpotMarketConfiguration_LogNormal {
	return &vegapb.UpdateSpotMarketConfiguration_LogNormal{
		LogNormal: n.LogNormal.IntoProto(),
	}
}

func UpdateSpotMarketConfigurationLogNormalFromProto(p *vegapb.UpdateSpotMarketConfiguration_LogNormal) *UpdateSpotMarketConfigurationLogNormal {
	return &UpdateSpotMarketConfigurationLogNormal{
		LogNormal: &LogNormalRiskModel{
			RiskAversionParameter: num.DecimalFromFloat(p.LogNormal.RiskAversionParameter),
			Tau:                   num.DecimalFromFloat(p.LogNormal.Tau),
			Params:                LogNormalParamsFromProto(p.LogNormal.Params),
		},
	}
}
