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

	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/ptr"
	"code.vegaprotocol.io/vega/libs/stringer"
	proto "code.vegaprotocol.io/vega/protos/vega"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
)

type LiquidityFeeMethod = proto.LiquidityFeeSettings_Method

const (
	LiquidityFeeMethodUnspecified     LiquidityFeeMethod = proto.LiquidityFeeSettings_METHOD_UNSPECIFIED
	LiquidityFeeMethodMarginalCost    LiquidityFeeMethod = proto.LiquidityFeeSettings_METHOD_MARGINAL_COST
	LiquidityFeeMethodWeightedAverage LiquidityFeeMethod = proto.LiquidityFeeSettings_METHOD_WEIGHTED_AVERAGE
	LiquidityFeeMethodConstant        LiquidityFeeMethod = proto.LiquidityFeeSettings_METHOD_CONSTANT
)

type LiquidityProvisionStatus = proto.LiquidityProvision_Status

const (
	// LiquidityProvisionUnspecified The default value.
	LiquidityProvisionUnspecified LiquidityProvisionStatus = proto.LiquidityProvision_STATUS_UNSPECIFIED
	// LiquidityProvisionStatusActive The liquidity provision is active.
	LiquidityProvisionStatusActive LiquidityProvisionStatus = proto.LiquidityProvision_STATUS_ACTIVE
	// LiquidityProvisionStatusStopped The liquidity provision was stopped by the network.
	LiquidityProvisionStatusStopped LiquidityProvisionStatus = proto.LiquidityProvision_STATUS_STOPPED
	// LiquidityProvisionStatusCancelled The liquidity provision was cancelled by the liquidity provider.
	LiquidityProvisionStatusCancelled LiquidityProvisionStatus = proto.LiquidityProvision_STATUS_CANCELLED
	// LiquidityProvisionStatusRejected The liquidity provision was invalid and got rejected.
	LiquidityProvisionStatusRejected LiquidityProvisionStatus = proto.LiquidityProvision_STATUS_REJECTED
	// LiquidityProvisionStatusUndeployed The liquidity provision is valid and accepted by network, but orders aren't deployed.
	LiquidityProvisionStatusUndeployed LiquidityProvisionStatus = proto.LiquidityProvision_STATUS_UNDEPLOYED
	// LiquidityProvisionStatusPending The liquidity provision is valid and accepted by network
	// but have never been deployed. I when it's possible to deploy them for the first time
	// margin check fails, then they will be cancelled without any penalties.
	LiquidityProvisionStatusPending LiquidityProvisionStatus = proto.LiquidityProvision_STATUS_PENDING
)

type LiquiditySLAParams struct {
	PriceRange                  num.Decimal
	CommitmentMinTimeFraction   num.Decimal
	PerformanceHysteresisEpochs uint64
	SlaCompetitionFactor        num.Decimal
}

func (l LiquiditySLAParams) IntoProto() *proto.LiquiditySLAParameters {
	return &proto.LiquiditySLAParameters{
		PriceRange:                  l.PriceRange.String(),
		CommitmentMinTimeFraction:   l.CommitmentMinTimeFraction.String(),
		PerformanceHysteresisEpochs: l.PerformanceHysteresisEpochs,
		SlaCompetitionFactor:        l.SlaCompetitionFactor.String(),
	}
}

func LiquiditySLAParamsFromProto(l *proto.LiquiditySLAParameters) *LiquiditySLAParams {
	if l == nil {
		return nil
	}
	return &LiquiditySLAParams{
		PriceRange:                  num.MustDecimalFromString(l.PriceRange),
		CommitmentMinTimeFraction:   num.MustDecimalFromString(l.CommitmentMinTimeFraction),
		PerformanceHysteresisEpochs: l.PerformanceHysteresisEpochs,
		SlaCompetitionFactor:        num.MustDecimalFromString(l.SlaCompetitionFactor),
	}
}

func (l LiquiditySLAParams) String() string {
	return fmt.Sprintf(
		"priceRange(%s) commitmentMinTimeFraction(%s) performanceHysteresisEpochs(%v) slaCompetitionFactor(%s)",
		l.PriceRange.String(),
		l.CommitmentMinTimeFraction.String(),
		l.PerformanceHysteresisEpochs,
		l.SlaCompetitionFactor.String(),
	)
}

func (l LiquiditySLAParams) DeepClone() *LiquiditySLAParams {
	return &LiquiditySLAParams{
		PriceRange:                  l.PriceRange,
		CommitmentMinTimeFraction:   l.CommitmentMinTimeFraction,
		PerformanceHysteresisEpochs: l.PerformanceHysteresisEpochs,
		SlaCompetitionFactor:        l.SlaCompetitionFactor,
	}
}

type TargetStakeParameters struct {
	TimeWindow    int64
	ScalingFactor num.Decimal
}

func (t TargetStakeParameters) IntoProto() *proto.TargetStakeParameters {
	sf, _ := t.ScalingFactor.Float64()
	return &proto.TargetStakeParameters{
		TimeWindow:    t.TimeWindow,
		ScalingFactor: sf,
	}
}

func TargetStakeParametersFromProto(p *proto.TargetStakeParameters) *TargetStakeParameters {
	return &TargetStakeParameters{
		TimeWindow:    p.TimeWindow,
		ScalingFactor: num.DecimalFromFloat(p.ScalingFactor),
	}
}

func (t TargetStakeParameters) String() string {
	return fmt.Sprintf(
		"timeWindows(%v) scalingFactor(%s)",
		t.TimeWindow,
		t.ScalingFactor.String(),
	)
}

func (t TargetStakeParameters) DeepClone() *TargetStakeParameters {
	return &TargetStakeParameters{
		TimeWindow:    t.TimeWindow,
		ScalingFactor: t.ScalingFactor,
	}
}

type LiquidityProvisionSubmission struct {
	// Market identifier for the order, required field
	MarketID string
	// Specified as a unitless number that represents the amount of settlement asset of the market
	CommitmentAmount *num.Uint
	// Nominated liquidity fee factor, which is an input to the calculation of taker fees on the market, as per setting fees and rewarding liquidity providers
	Fee num.Decimal
	// A reference to be added to every order created out of this liquidityProvisionSubmission
	Reference string
}

func (l LiquidityProvisionSubmission) IntoProto() *commandspb.LiquidityProvisionSubmission {
	return &commandspb.LiquidityProvisionSubmission{
		MarketId:         l.MarketID,
		CommitmentAmount: num.UintToString(l.CommitmentAmount),
		Fee:              l.Fee.String(),
		Reference:        l.Reference,
	}
}

func LiquidityProvisionSubmissionFromProto(p *commandspb.LiquidityProvisionSubmission) (*LiquidityProvisionSubmission, error) {
	fee, err := num.DecimalFromString(p.Fee)
	if err != nil {
		return nil, err
	}

	commitmentAmount := num.UintZero()
	if len(p.CommitmentAmount) > 0 {
		var overflowed bool
		commitmentAmount, overflowed = num.UintFromString(p.CommitmentAmount, 10)
		if overflowed {
			return nil, errors.New("invalid commitment amount")
		}
	}

	l := LiquidityProvisionSubmission{
		Fee:              fee,
		MarketID:         p.MarketId,
		CommitmentAmount: commitmentAmount,
		Reference:        p.Reference,
	}

	return &l, nil
}

func (l LiquidityProvisionSubmission) String() string {
	return fmt.Sprintf(
		"marketID(%s) reference(%s) commitmentAmount(%s) fee(%s)",
		l.MarketID,
		l.Reference,
		stringer.PtrToString(l.CommitmentAmount),
		l.Fee.String(),
	)
}

type LiquidityProvision struct {
	// Unique identifier
	ID string
	// Unique party identifier for the creator of the provision
	Party string
	// Timestamp for when the order was created at, in nanoseconds since the epoch
	// - See [`VegaTimeResponse`](#api.VegaTimeResponse).`timestamp`
	CreatedAt int64
	// Timestamp for when the order was updated at, in nanoseconds since the epoch
	// - See [`VegaTimeResponse`](#api.VegaTimeResponse).`timestamp`
	UpdatedAt int64
	// Market identifier for the order, required field
	MarketID string
	// Specified as a unitless number that represents the amount of settlement asset of the market
	CommitmentAmount *num.Uint
	// Nominated liquidity fee factor, which is an input to the calculation of taker fees on the market, as per seeting fees and rewarding liquidity providers
	Fee num.Decimal
	// Version of this liquidity provision
	Version uint64
	// Status of this liquidity provision
	Status LiquidityProvisionStatus
	// A reference shared between this liquidity provision and all it's orders
	Reference string
}

func (l LiquidityProvision) String() string {
	return fmt.Sprintf(
		"ID(%s) marketID(%s) party(%s) status(%s) reference(%s) commitmentAmount(%s) fee(%s) version(%v) createdAt(%v) updatedAt(%v)",
		l.ID,
		l.MarketID,
		l.Party,
		l.Status.String(),
		l.Reference,
		stringer.PtrToString(l.CommitmentAmount),
		l.Fee.String(),
		l.Version,
		l.CreatedAt,
		l.UpdatedAt,
	)
}

func (l LiquidityProvision) IntoProto() *proto.LiquidityProvision {
	lp := &proto.LiquidityProvision{
		Id:               l.ID,
		PartyId:          l.Party,
		CreatedAt:        l.CreatedAt,
		UpdatedAt:        l.UpdatedAt,
		MarketId:         l.MarketID,
		CommitmentAmount: num.UintToString(l.CommitmentAmount),
		Fee:              l.Fee.String(),
		Version:          l.Version,
		Status:           l.Status,
		Reference:        l.Reference,
	}

	return lp
}

func LiquidityProvisionFromProto(p *proto.LiquidityProvision) (*LiquidityProvision, error) {
	fee, _ := num.DecimalFromString(p.Fee)
	commitmentAmount := num.UintZero()
	if len(p.CommitmentAmount) > 0 {
		var overflowed bool
		commitmentAmount, overflowed = num.UintFromString(p.CommitmentAmount, 10)
		if overflowed {
			return nil, errors.New("invalid commitment amount")
		}
	}
	l := LiquidityProvision{
		CommitmentAmount: commitmentAmount,
		CreatedAt:        p.CreatedAt,
		ID:               p.Id,
		MarketID:         p.MarketId,
		Party:            p.PartyId,
		Fee:              fee,
		Reference:        p.Reference,
		Status:           p.Status,
		UpdatedAt:        p.UpdatedAt,
		Version:          p.Version,
	}

	return &l, nil
}

type LiquidityMonitoringParameters struct {
	// Specifies parameters related to target stake calculation
	TargetStakeParameters *TargetStakeParameters
}

func (l LiquidityMonitoringParameters) IntoProto() *proto.LiquidityMonitoringParameters {
	var params *proto.TargetStakeParameters
	if l.TargetStakeParameters != nil {
		params = l.TargetStakeParameters.IntoProto()
	}
	return &proto.LiquidityMonitoringParameters{
		TargetStakeParameters: params,
	}
}

func (l LiquidityMonitoringParameters) DeepClone() *LiquidityMonitoringParameters {
	var params *TargetStakeParameters
	if l.TargetStakeParameters != nil {
		params = l.TargetStakeParameters.DeepClone()
	}
	return &LiquidityMonitoringParameters{
		TargetStakeParameters: params,
	}
}

func (l LiquidityMonitoringParameters) String() string {
	return fmt.Sprintf(
		"auctionExtension(%v)",
		stringer.PtrToString(l.TargetStakeParameters),
	)
}

func LiquidityMonitoringParametersFromProto(p *proto.LiquidityMonitoringParameters) (*LiquidityMonitoringParameters, error) {
	if p == nil {
		return nil, nil
	}
	var params *TargetStakeParameters
	if p.TargetStakeParameters != nil {
		params = TargetStakeParametersFromProto(p.TargetStakeParameters)
	}

	return &LiquidityMonitoringParameters{
		TargetStakeParameters: params,
	}, nil
}

type LiquidityProvisionAmendment struct {
	// Market identifier for the order, required field
	MarketID string
	// Specified as a unitless number that represents the amount of settlement asset of the market
	CommitmentAmount *num.Uint
	// Nominated liquidity fee factor, which is an input to the calculation of taker fees on the market, as per setting fees and rewarding liquidity providers
	Fee num.Decimal
	// A reference to be added to every order created out of this liquidityProvisionAmendment
	Reference string
}

func LiquidityProvisionAmendmentFromProto(p *commandspb.LiquidityProvisionAmendment) (*LiquidityProvisionAmendment, error) {
	fee, err := num.DecimalFromString(p.Fee)
	if err != nil {
		return nil, err
	}

	commitmentAmount := num.UintZero()
	if len(p.CommitmentAmount) > 0 {
		var overflowed bool
		commitmentAmount, overflowed = num.UintFromString(p.CommitmentAmount, 10)
		if overflowed {
			return nil, errors.New("invalid commitment amount")
		}
	}

	return &LiquidityProvisionAmendment{
		Fee:              fee,
		MarketID:         p.MarketId,
		CommitmentAmount: commitmentAmount,
		Reference:        p.Reference,
	}, nil
}

func (a LiquidityProvisionAmendment) IntoProto() *commandspb.LiquidityProvisionAmendment {
	return &commandspb.LiquidityProvisionAmendment{
		MarketId:         a.MarketID,
		CommitmentAmount: num.UintToString(a.CommitmentAmount),
		Fee:              a.Fee.String(),
		Reference:        a.Reference,
	}
}

func (a LiquidityProvisionAmendment) String() string {
	return fmt.Sprintf(
		"marketID(%s) reference(%s) commitmentAmount(%s) fee(%s)",
		a.MarketID,
		a.Reference,
		stringer.PtrToString(a.CommitmentAmount),
		a.Fee.String(),
	)
}

func (a LiquidityProvisionAmendment) GetMarketID() string {
	return a.MarketID
}

type LiquidityProvisionCancellation struct {
	// Market identifier for the order, required field
	MarketID string
}

func LiquidityProvisionCancellationFromProto(p *commandspb.LiquidityProvisionCancellation) (*LiquidityProvisionCancellation, error) {
	l := LiquidityProvisionCancellation{
		MarketID: p.MarketId,
	}

	return &l, nil
}

func (l LiquidityProvisionCancellation) IntoProto() *commandspb.LiquidityProvisionCancellation {
	return &commandspb.LiquidityProvisionCancellation{
		MarketId: l.MarketID,
	}
}

func (l LiquidityProvisionCancellation) String() string {
	return fmt.Sprintf("marketID(%s)", l.MarketID)
}

func (l LiquidityProvisionCancellation) GetMarketID() string {
	return l.MarketID
}

type LiquidityFeeSettings struct {
	Method      LiquidityFeeMethod
	FeeConstant num.Decimal
}

func (l *LiquidityFeeSettings) IntoProto() *proto.LiquidityFeeSettings {
	if l == nil {
		return nil
	}

	r := &proto.LiquidityFeeSettings{
		Method: l.Method,
	}

	if l.Method == LiquidityFeeMethodConstant {
		r.FeeConstant = ptr.From(l.FeeConstant.String())
	}

	return r
}

func LiquidityFeeSettingsFromProto(l *proto.LiquidityFeeSettings) *LiquidityFeeSettings {
	if l == nil {
		return nil
	}

	fc := num.DecimalZero()
	if l.Method == LiquidityFeeMethodConstant {
		fc, _ = num.DecimalFromString(*l.FeeConstant)
	}

	return &LiquidityFeeSettings{
		Method:      l.Method,
		FeeConstant: fc,
	}
}

func (l *LiquidityFeeSettings) DeepClone() *LiquidityFeeSettings {
	if l == nil {
		return nil
	}
	return &LiquidityFeeSettings{
		Method:      l.Method,
		FeeConstant: l.FeeConstant,
	}
}

func (l LiquidityFeeSettings) String() string {
	return fmt.Sprintf(
		"method(%s) feeConstant(%s)",
		l.Method.String(),
		l.FeeConstant.String(),
	)
}
