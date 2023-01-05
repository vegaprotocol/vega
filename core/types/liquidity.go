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
	proto "code.vegaprotocol.io/vega/protos/vega"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
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
	// A set of liquidity sell orders to meet the liquidity provision obligation
	Sells []*LiquidityOrder
	// A set of liquidity buy orders to meet the liquidity provision obligation
	Buys []*LiquidityOrder
	// A reference to be added to every order created out of this liquidityProvisionSubmission
	Reference string
}

func (l LiquidityProvisionSubmission) IntoProto() *commandspb.LiquidityProvisionSubmission {
	lps := &commandspb.LiquidityProvisionSubmission{
		MarketId:         l.MarketID,
		CommitmentAmount: num.UintToString(l.CommitmentAmount),
		Fee:              l.Fee.String(),
		Sells:            make([]*proto.LiquidityOrder, 0, len(l.Sells)),
		Buys:             make([]*proto.LiquidityOrder, 0, len(l.Buys)),
		Reference:        l.Reference,
	}

	for _, sell := range l.Sells {
		lps.Sells = append(lps.Sells, sell.IntoProto())
	}

	for _, buy := range l.Buys {
		lps.Buys = append(lps.Buys, buy.IntoProto())
	}
	return lps
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

func (l LiquidityProvisionSubmission) String() string {
	return fmt.Sprintf(
		"marketID(%s) reference(%s) commitmentAmount(%s) fee(%s) sells(%s) buys(%s)",
		l.MarketID,
		l.Reference,
		uintPointerToString(l.CommitmentAmount),
		l.Fee.String(),
		LiquidityOrders(l.Sells).String(),
		LiquidityOrders(l.Buys).String(),
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
	// A set of liquidity sell orders to meet the liquidity provision obligation
	Sells []*LiquidityOrderReference
	// A set of liquidity buy orders to meet the liquidity provision obligation
	Buys []*LiquidityOrderReference
	// Version of this liquidity provision order
	Version uint64
	// Status of this liquidity provision order
	Status LiquidityProvisionStatus
	// A reference shared between this liquidity provision and all it's orders
	Reference string
}

func (l LiquidityProvision) String() string {
	return fmt.Sprintf(
		"ID(%s) marketID(%s) party(%s) status(%s) reference(%s) commitmentAmount(%s) fee(%s) sells(%s) buys(%s) version(%v) createdAt(%v) updatedAt(%v)",
		l.ID,
		l.MarketID,
		l.Party,
		l.Status.String(),
		l.Reference,
		uintPointerToString(l.CommitmentAmount),
		l.Fee.String(),
		LiquidityOrderReferences(l.Sells).String(),
		LiquidityOrderReferences(l.Buys).String(),
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
		Sells:            make([]*proto.LiquidityOrderReference, 0, len(l.Sells)),
		Buys:             make([]*proto.LiquidityOrderReference, 0, len(l.Buys)),
	}

	for _, sell := range l.Sells {
		lp.Sells = append(lp.Sells, sell.IntoProto())
	}

	for _, buy := range l.Buys {
		lp.Buys = append(lp.Buys, buy.IntoProto())
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
		Sells:            make([]*LiquidityOrderReference, 0, len(p.Sells)),
		Buys:             make([]*LiquidityOrderReference, 0, len(p.Buys)),
	}

	for _, sell := range p.Sells {
		lor, err := LiquidityOrderReferenceFromProto(sell)
		if err != nil {
			return nil, err
		}
		l.Sells = append(l.Sells, lor)
	}

	for _, buy := range p.Buys {
		lor, err := LiquidityOrderReferenceFromProto(buy)
		if err != nil {
			return nil, err
		}
		l.Buys = append(l.Buys, lor)
	}

	return &l, nil
}

type LiquidityOrderReferences []*LiquidityOrderReference

func (ls LiquidityOrderReferences) String() string {
	if ls == nil {
		return "[]"
	}
	strs := make([]string, 0, len(ls))
	for _, l := range ls {
		strs = append(strs, l.String())
	}
	return "[" + strings.Join(strs, ", ") + "]"
}

type LiquidityOrderReference struct {
	// Unique identifier of the pegged order generated by the core to fulfil this liquidity order
	OrderID string
	// The liquidity order from the original submission
	LiquidityOrder *LiquidityOrder
}

func (l LiquidityOrderReference) String() string {
	return fmt.Sprintf(
		"orderID(%s) liquidityOrder(%s)",
		l.OrderID,
		reflectPointerToString(l.LiquidityOrder),
	)
}

func (l LiquidityOrderReference) IntoProto() *proto.LiquidityOrderReference {
	var order *proto.LiquidityOrder
	if l.LiquidityOrder != nil {
		order = l.LiquidityOrder.IntoProto()
	}
	return &proto.LiquidityOrderReference{
		OrderId:        l.OrderID,
		LiquidityOrder: order,
	}
}

func LiquidityOrderReferenceFromProto(p *proto.LiquidityOrderReference) (*LiquidityOrderReference, error) {
	lo, err := LiquidityOrderFromProto(p.LiquidityOrder)
	if err != nil {
		return nil, err
	}

	return &LiquidityOrderReference{
		OrderID:        p.OrderId,
		LiquidityOrder: lo,
	}, nil
}

type LiquidityOrders []*LiquidityOrder

func (ls LiquidityOrders) String() string {
	if ls == nil {
		return "[]"
	}
	strs := make([]string, 0, len(ls))
	for _, l := range ls {
		strs = append(strs, l.String())
	}
	return "[" + strings.Join(strs, ", ") + "]"
}

type LiquidityOrder struct {
	// The pegged reference point for the order
	Reference PeggedReference
	// The relative proportion of the commitment to be allocated at a price level
	Proportion uint32
	// The offset/amount of units away for the order
	Offset *num.Uint
}

func (l LiquidityOrder) String() string {
	return fmt.Sprintf(
		"reference(%s) proportion(%v) offset(%s)",
		l.Reference.String(),
		l.Proportion,
		uintPointerToString(l.Offset),
	)
}

func (l LiquidityOrder) DeepClone() *LiquidityOrder {
	return &LiquidityOrder{
		Reference:  l.Reference,
		Proportion: l.Proportion,
		Offset:     l.Offset,
	}
}

func (l LiquidityOrder) IntoProto() *proto.LiquidityOrder {
	return &proto.LiquidityOrder{
		Reference:  l.Reference,
		Proportion: l.Proportion,
		Offset:     l.Offset.String(),
	}
}

func LiquidityOrderFromProto(p *proto.LiquidityOrder) (*LiquidityOrder, error) {
	offset, overflow := num.UintFromString(p.Offset, 10)
	if overflow {
		return nil, errors.New("invalid offset")
	}

	return &LiquidityOrder{
		Offset:     offset,
		Proportion: p.Proportion,
		Reference:  p.Reference,
	}, nil
}

type LiquidityMonitoringParameters struct {
	// Specifies parameters related to target stake calculation
	TargetStakeParameters *TargetStakeParameters
	// Specifies the triggering ratio for entering liquidity auction
	TriggeringRatio num.Decimal
	// Specifies by how many seconds an auction should be extended if leaving the auction were to trigger a liquidity auction
	AuctionExtension int64
}

func (l LiquidityMonitoringParameters) IntoProto() *proto.LiquidityMonitoringParameters {
	var params *proto.TargetStakeParameters
	if l.TargetStakeParameters != nil {
		params = l.TargetStakeParameters.IntoProto()
	}
	return &proto.LiquidityMonitoringParameters{
		TargetStakeParameters: params,
		TriggeringRatio:       l.TriggeringRatio.String(),
		AuctionExtension:      l.AuctionExtension,
	}
}

func (l LiquidityMonitoringParameters) DeepClone() *LiquidityMonitoringParameters {
	var params *TargetStakeParameters
	if l.TargetStakeParameters != nil {
		params = l.TargetStakeParameters.DeepClone()
	}
	return &LiquidityMonitoringParameters{
		TriggeringRatio:       l.TriggeringRatio,
		AuctionExtension:      l.AuctionExtension,
		TargetStakeParameters: params,
	}
}

func (l LiquidityMonitoringParameters) String() string {
	return fmt.Sprintf(
		"auctionExtension(%v) trigerringRatio(%s) targetStake(%s)",
		l.AuctionExtension,
		l.TriggeringRatio.String(),
		reflectPointerToString(l.TargetStakeParameters),
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

	tr, err := num.DecimalFromString(p.TriggeringRatio)
	if err != nil {
		return nil, fmt.Errorf("error getting trigerring ratio value from proto: %s", err)
	}

	return &LiquidityMonitoringParameters{
		TargetStakeParameters: params,
		AuctionExtension:      p.AuctionExtension,
		TriggeringRatio:       tr,
	}, nil
}

type LiquidityProvisionAmendment struct {
	// Market identifier for the order, required field
	MarketID string
	// Specified as a unitless number that represents the amount of settlement asset of the market
	CommitmentAmount *num.Uint
	// Nominated liquidity fee factor, which is an input to the calculation of taker fees on the market, as per setting fees and rewarding liquidity providers
	Fee num.Decimal
	// A set of liquidity sell orders to meet the liquidity provision obligation
	Sells []*LiquidityOrder
	// A set of liquidity buy orders to meet the liquidity provision obligation
	Buys []*LiquidityOrder
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

	l := LiquidityProvisionAmendment{
		Fee:              fee,
		MarketID:         p.MarketId,
		CommitmentAmount: commitmentAmount,
		Sells:            make([]*LiquidityOrder, 0, len(p.Sells)),
		Buys:             make([]*LiquidityOrder, 0, len(p.Buys)),
		Reference:        p.Reference,
	}

	for _, sell := range p.Sells {
		offset := num.UintZero()

		if len(p.CommitmentAmount) > 0 {
			var overflowed bool
			offset, overflowed = num.UintFromString(sell.Offset, 10)
			if overflowed {
				return nil, errors.New("invalid sell side offset")
			}
		}

		order := &LiquidityOrder{
			Reference:  sell.Reference,
			Proportion: sell.Proportion,
			Offset:     offset,
		}
		l.Sells = append(l.Sells, order)
	}

	for _, buy := range p.Buys {
		offset := num.UintZero()

		if len(p.CommitmentAmount) > 0 {
			var overflowed bool
			offset, overflowed = num.UintFromString(buy.Offset, 10)
			if overflowed {
				return nil, errors.New("invalid buy side offset")
			}
		}

		order := &LiquidityOrder{
			Reference:  buy.Reference,
			Proportion: buy.Proportion,
			Offset:     offset,
		}
		l.Buys = append(l.Buys, order)
	}
	return &l, nil
}

func (a LiquidityProvisionAmendment) IntoProto() *commandspb.LiquidityProvisionAmendment {
	lps := &commandspb.LiquidityProvisionAmendment{
		MarketId:         a.MarketID,
		CommitmentAmount: num.UintToString(a.CommitmentAmount),
		Fee:              a.Fee.String(),
		Sells:            make([]*proto.LiquidityOrder, 0, len(a.Sells)),
		Buys:             make([]*proto.LiquidityOrder, 0, len(a.Buys)),
		Reference:        a.Reference,
	}

	for _, sell := range a.Sells {
		order := &proto.LiquidityOrder{
			Reference:  sell.Reference,
			Proportion: sell.Proportion,
			Offset:     sell.Offset.String(),
		}
		lps.Sells = append(lps.Sells, order)
	}

	for _, buy := range a.Buys {
		order := &proto.LiquidityOrder{
			Reference:  buy.Reference,
			Proportion: buy.Proportion,
			Offset:     buy.Offset.String(),
		}
		lps.Buys = append(lps.Buys, order)
	}
	return lps
}

func (a LiquidityProvisionAmendment) String() string {
	return fmt.Sprintf(
		"marketID(%s) reference(%s) commitmentAmount(%s) fee(%s) sells(%v) buys(%v)",
		a.MarketID,
		a.Reference,
		uintPointerToString(a.CommitmentAmount),
		a.Fee.String(),
		LiquidityOrders(a.Sells).String(),
		LiquidityOrders(a.Buys).String(),
	)
}

func (a LiquidityProvisionAmendment) GetMarketID() string {
	return a.MarketID
}

func (a LiquidityProvisionAmendment) ContainsOrders() bool {
	return len(a.Sells) > 0 || len(a.Buys) > 0
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
