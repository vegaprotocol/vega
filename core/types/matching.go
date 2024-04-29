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
	"time"

	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/ptr"
	"code.vegaprotocol.io/vega/libs/stringer"
	proto "code.vegaprotocol.io/vega/protos/vega"
)

type Order struct {
	ID               string
	MarketID         string
	Party            string
	Side             Side
	Price            *num.Uint
	OriginalPrice    *num.Uint
	Size             uint64
	Remaining        uint64
	TimeInForce      OrderTimeInForce
	Type             OrderType
	CreatedAt        int64
	Status           OrderStatus
	ExpiresAt        int64
	Reference        string
	Reason           OrderError
	UpdatedAt        int64
	Version          uint64
	BatchID          uint64
	PeggedOrder      *PeggedOrder
	PostOnly         bool
	ReduceOnly       bool
	extraRemaining   uint64
	IcebergOrder     *IcebergOrder
	GeneratedOffbook bool
}

func (o *Order) ReduceOnlyAdjustRemaining(extraSize uint64) {
	if !o.ReduceOnly {
		panic("order.ReduceOnlyAdjustSize shall be call only on reduce-only orders")
	}

	o.extraRemaining = extraSize
	o.Remaining -= extraSize
}

func (o *Order) ClearUpExtraRemaining() {
	// ignore for non reduce only
	if !o.ReduceOnly {
		return
	}

	o.Remaining += o.extraRemaining
	o.extraRemaining = 0
}

// TrueRemaining is the full remaining size of an order. If this is an iceberg order
// it will return the visible peak + the hidden volume.
func (o *Order) TrueRemaining() uint64 {
	rem := o.Remaining
	if o.IcebergOrder != nil {
		rem += o.IcebergOrder.ReservedRemaining
	}
	return rem
}

// IcebergNeedsRefresh returns whether the given iceberg order's visible peak has
// dropped below the minimum visible size, and there is hidden volume available to
// restore it.
func (o *Order) IcebergNeedsRefresh() bool {
	if o.IcebergOrder == nil {
		// not an iceberg
		return false
	}

	if o.IcebergOrder.ReservedRemaining == 0 {
		// nothing to refresh with
		return false
	}

	if o.Remaining >= o.IcebergOrder.MinimumVisibleSize {
		// not under the minimum
		return false
	}

	return true
}

// SetIcebergPeaks will restore the given iceberg orders visible size with
// some of its hidden volume.
func (o *Order) SetIcebergPeaks() {
	if o.IcebergOrder == nil {
		return
	}

	if o.Remaining > o.IcebergOrder.PeakSize && o.IcebergOrder.ReservedRemaining == 0 {
		// iceberg is at full volume and so set its visible amount to its peak size
		peak := num.MinV(o.Remaining, o.IcebergOrder.PeakSize)
		o.IcebergOrder.ReservedRemaining = o.Remaining - peak
		o.Remaining = peak
		return
	}

	// calculate the refill amount
	refill := o.IcebergOrder.PeakSize - o.Remaining
	refill = num.MinV(refill, o.IcebergOrder.ReservedRemaining)

	o.Remaining += refill
	o.IcebergOrder.ReservedRemaining -= refill
}

func (o Order) IntoSubmission() *OrderSubmission {
	sub := &OrderSubmission{
		MarketID:    o.MarketID,
		Size:        o.Size,
		Side:        o.Side,
		TimeInForce: o.TimeInForce,
		ExpiresAt:   o.ExpiresAt,
		Type:        o.Type,
		Reference:   o.Reference,
		PostOnly:    o.PostOnly,
		ReduceOnly:  o.ReduceOnly,
	}
	if o.IcebergOrder != nil {
		sub.IcebergOrder = &IcebergOrder{
			PeakSize:           o.IcebergOrder.PeakSize,
			MinimumVisibleSize: o.IcebergOrder.MinimumVisibleSize,
		}
	}
	if o.Price != nil {
		sub.Price = o.Price.Clone()
	}
	if o.PeggedOrder != nil {
		sub.PeggedOrder = o.PeggedOrder.Clone()
	}

	return sub
}

func (o Order) Clone() *Order {
	cpy := o
	if o.Price != nil {
		cpy.Price = o.Price.Clone()
	} else {
		cpy.Price = num.UintZero()
	}
	// this isn't really needed, to original order is about to be replaced, or the original price is getting reassinged
	// but in case something goes wrong, we don't want a pointer to this field in 2 places
	if o.OriginalPrice != nil {
		cpy.OriginalPrice = o.OriginalPrice.Clone()
	}
	if o.PeggedOrder != nil {
		cpy.PeggedOrder = o.PeggedOrder.Clone()
	}
	if o.IcebergOrder != nil {
		cpy.IcebergOrder = o.IcebergOrder.Clone()
	}
	return &cpy
}

func (o Order) String() string {
	return fmt.Sprintf(
		"ID(%s) marketID(%s) party(%s) side(%s) price(%s) size(%v) remaining(%v) timeInForce(%s) type(%s) status(%s) reference(%s) reason(%s) version(%v) batchID(%v) createdAt(%v) updatedAt(%v) expiresAt(%v) originalPrice(%s) peggedOrder(%s) postOnly(%v) reduceOnly(%v) iceberg(%s)",
		o.ID,
		o.MarketID,
		o.Party,
		o.Side.String(),
		num.UintToString(o.Price),
		o.Size,
		o.Remaining,
		o.TimeInForce.String(),
		o.Type.String(),
		o.Status.String(),
		o.Reference,
		o.Reason.String(),
		o.Version,
		o.BatchID,
		o.CreatedAt,
		o.UpdatedAt,
		o.ExpiresAt,
		stringer.PtrToString(o.OriginalPrice),
		stringer.PtrToString(o.PeggedOrder),
		o.PostOnly,
		o.ReduceOnly,
		stringer.PtrToString(o.IcebergOrder),
	)
}

type Orders []*Order

func (o Orders) IntoProto() []*proto.Order {
	out := make([]*proto.Order, 0, len(o))
	for _, v := range o {
		out = append(out, v.IntoProto())
	}
	return out
}

func (o *Order) IntoProto() *proto.Order {
	var pegged *proto.PeggedOrder
	if o.PeggedOrder != nil {
		pegged = o.PeggedOrder.IntoProto()
	}
	var reason *OrderError
	if o.Reason != OrderErrorUnspecified {
		reason = ptr.From(o.Reason)
	}

	var iceberg *proto.IcebergOrder
	if o.IcebergOrder != nil {
		iceberg = o.IcebergOrder.IntoProto()
	}

	return &proto.Order{
		Id:           o.ID,
		MarketId:     o.MarketID,
		PartyId:      o.Party,
		Side:         o.Side,
		Price:        num.UintToString(o.Price),
		Size:         o.Size,
		Remaining:    o.Remaining,
		TimeInForce:  o.TimeInForce,
		Type:         o.Type,
		CreatedAt:    o.CreatedAt,
		Status:       o.Status,
		ExpiresAt:    o.ExpiresAt,
		Reference:    o.Reference,
		Reason:       reason,
		UpdatedAt:    o.UpdatedAt,
		Version:      o.Version,
		BatchId:      o.BatchID,
		PeggedOrder:  pegged,
		PostOnly:     o.PostOnly,
		ReduceOnly:   o.ReduceOnly,
		IcebergOrder: iceberg,
	}
}

func OrderFromProto(o *proto.Order) (*Order, error) {
	var iceberg *IcebergOrder
	if o.IcebergOrder != nil {
		var err error
		iceberg, err = NewIcebergOrderFromProto(o.IcebergOrder)
		if err != nil {
			return nil, err
		}
	}
	var pegged *PeggedOrder
	if o.PeggedOrder != nil {
		var err error
		pegged, err = NewPeggedOrderFromProto(o.PeggedOrder)
		if err != nil {
			return nil, err
		}
	}
	price := num.UintZero()
	if len(o.Price) > 0 {
		var overflowed bool
		price, overflowed = num.UintFromString(o.Price, 10)
		if overflowed {
			return nil, errors.New("invalid price")
		}
	}
	reason := OrderErrorUnspecified
	if o.Reason != nil {
		reason = *o.Reason
	}
	return &Order{
		ID:           o.Id,
		MarketID:     o.MarketId,
		Party:        o.PartyId,
		Side:         o.Side,
		Price:        price,
		Size:         o.Size,
		Remaining:    o.Remaining,
		TimeInForce:  o.TimeInForce,
		Type:         o.Type,
		CreatedAt:    o.CreatedAt,
		Status:       o.Status,
		ExpiresAt:    o.ExpiresAt,
		Reference:    o.Reference,
		Reason:       reason,
		UpdatedAt:    o.UpdatedAt,
		Version:      o.Version,
		BatchID:      o.BatchId,
		PeggedOrder:  pegged,
		PostOnly:     o.PostOnly,
		ReduceOnly:   o.ReduceOnly,
		IcebergOrder: iceberg,
	}, nil
}

// Create sets the creation time (CreatedAt) to t and returns the
// updated order.
func (o *Order) Create(t int64) *Order {
	o.CreatedAt = t
	return o
}

// Update sets the modification time (UpdatedAt) to t and returns the
// updated order.
func (o *Order) Update(t time.Time) *Order {
	o.UpdatedAt = t.UnixNano()
	return o
}

// IsPersistent returns true if the order is persistent.
// A persistent order is a Limit type order that might be
// matched in the future.
func (o *Order) IsPersistent() bool {
	return (o.TimeInForce == OrderTimeInForceGTC ||
		o.TimeInForce == OrderTimeInForceGTT ||
		o.TimeInForce == OrderTimeInForceGFN ||
		o.TimeInForce == OrderTimeInForceGFA) &&
		o.Type == OrderTypeLimit &&
		o.Remaining > 0
}

func (o *Order) IsExpireable() bool {
	return (o.TimeInForce == OrderTimeInForceGFN ||
		o.TimeInForce == OrderTimeInForceGTT ||
		o.TimeInForce == OrderTimeInForceGFA) &&
		o.ExpiresAt > 0
}

// IsFinished returns true if an order
// is in any state different to ACTIVE and PARKED
// Basically any order which is never gonna
// trade anymore.
func (o *Order) IsFinished() bool {
	return o.Status != OrderStatusActive && o.Status != OrderStatusParked
}

func (o *Order) HasTraded() bool {
	return o.Size != o.Remaining
}

func (o *Order) applyOrderAmendmentSizeIceberg(delta int64) {
	// handle increase in size
	if delta > 0 {
		o.Size += uint64(delta)
		o.IcebergOrder.ReservedRemaining += uint64(delta)
		return
	}

	// handle decrease in size
	dec := uint64(-delta)
	o.Size -= dec

	if o.IcebergOrder.ReservedRemaining >= dec {
		o.IcebergOrder.ReservedRemaining -= dec
		return
	}

	diff := dec - o.IcebergOrder.ReservedRemaining
	if o.Remaining > diff {
		o.Remaining -= dec - o.IcebergOrder.ReservedRemaining
	} else {
		o.Remaining = 0
	}
	o.IcebergOrder.ReservedRemaining = 0
}

func (o *Order) amendSize(size uint64) {
	o.amendSizeWithDelta(int64(size) - int64(o.Size))
}

// amendSizeWithDelta update the orders size/remaining fields based on the size
// an direction of the given delta.
func (o *Order) amendSizeWithDelta(delta int64) {
	if o.IcebergOrder != nil {
		o.applyOrderAmendmentSizeIceberg(delta)
		return
	}

	// handle size increase
	if delta > 0 {
		o.Size += uint64(delta)
		o.Remaining += uint64(delta)
		return
	}

	// handle size decrease
	dec := uint64(-delta)
	o.Size -= dec
	if o.Remaining > dec {
		o.Remaining -= dec
	} else {
		o.Remaining = 0
	}
}

// ApplyOrderAmendment assumes the amendment have been validated before.
func (o *Order) ApplyOrderAmendment(amendment *OrderAmendment, updatedAtNano int64, priceFactor num.Decimal) (order *Order, err error) {
	order = o.Clone()
	order.UpdatedAt = updatedAtNano
	order.Version++

	if o.PeggedOrder != nil {
		order.PeggedOrder = &PeggedOrder{
			Reference: o.PeggedOrder.Reference,
			Offset:    o.PeggedOrder.Offset,
		}
	}

	var amendPrice *num.Uint
	if amendment.Price != nil {
		amendPrice, _ = num.UintFromDecimal(amendment.Price.ToDecimal().Mul(priceFactor))
	}
	// apply price changes
	if amendment.Price != nil && o.Price.NEQ(amendPrice) {
		order.Price = amendPrice.Clone()
		order.OriginalPrice = amendment.Price.Clone()
	}

	if amendment.Size != nil {
		order.amendSize(*amendment.Size)
	}

	if delta := amendment.SizeDelta; delta != 0 {
		order.amendSizeWithDelta(delta)
	}

	// apply tif
	if amendment.TimeInForce != OrderTimeInForceUnspecified {
		order.TimeInForce = amendment.TimeInForce
		if amendment.TimeInForce != OrderTimeInForceGTT {
			order.ExpiresAt = 0
		}
	}
	if amendment.ExpiresAt != nil {
		order.ExpiresAt = *amendment.ExpiresAt
	}

	// apply pegged order values
	if order.PeggedOrder != nil {
		if amendment.PeggedOffset != nil {
			order.PeggedOrder.Offset = amendment.PeggedOffset.Clone()
		}

		if amendment.PeggedReference != PeggedReferenceUnspecified {
			order.PeggedOrder.Reference = amendment.PeggedReference
		}
		if verr := order.ValidatePeggedOrder(); verr != OrderErrorUnspecified {
			err = verr
		}
	}

	return order, err
}

func (order *Order) ValidatePeggedOrder() OrderError {
	if order.Type != OrderTypeLimit {
		// All pegged orders must be LIMIT orders
		return ErrPeggedOrderMustBeLimitOrder
	}

	if order.TimeInForce != OrderTimeInForceGTT && order.TimeInForce != OrderTimeInForceGTC && order.TimeInForce != OrderTimeInForceGFN {
		// Pegged orders can only be GTC or GTT
		return ErrPeggedOrderMustBeGTTOrGTC
	}

	if order.PeggedOrder.Reference == PeggedReferenceUnspecified {
		// We must specify a valid reference
		return ErrPeggedOrderWithoutReferencePrice
	}

	if order.Side == SideBuy {
		switch order.PeggedOrder.Reference {
		case PeggedReferenceBestAsk:
			return ErrPeggedOrderBuyCannotReferenceBestAskPrice
		case PeggedReferenceMid:
			if order.PeggedOrder.Offset.IsZero() {
				return ErrPeggedOrderOffsetMustBeGreaterThanZero
			}
		}
	} else {
		switch order.PeggedOrder.Reference {
		case PeggedReferenceBestBid:
			return ErrPeggedOrderSellCannotReferenceBestBidPrice
		case PeggedReferenceMid:
			if order.PeggedOrder.Offset.IsZero() {
				return ErrPeggedOrderOffsetMustBeGreaterThanZero
			}
		}
	}
	return OrderErrorUnspecified
}

type PeggedOrder struct {
	Reference PeggedReference
	Offset    *num.Uint
}

func (p PeggedOrder) Clone() *PeggedOrder {
	cpy := p
	return &cpy
}

func NewPeggedOrderFromProto(p *proto.PeggedOrder) (*PeggedOrder, error) {
	if p == nil {
		return nil, nil
	}

	offset, overflowed := num.UintFromString(p.Offset, 10)
	if overflowed {
		return nil, errors.New("invalid offset")
	}

	return &PeggedOrder{
		Reference: p.Reference,
		Offset:    offset,
	}, nil
}

func (p PeggedOrder) IntoProto() *proto.PeggedOrder {
	return &proto.PeggedOrder{
		Reference: p.Reference,
		Offset:    p.Offset.String(),
	}
}

func (p PeggedOrder) String() string {
	return fmt.Sprintf(
		"reference(%s) offset(%s)",
		p.Reference.String(),
		stringer.PtrToString(p.Offset),
	)
}

type IcebergOrder struct {
	ReservedRemaining  uint64
	PeakSize           uint64
	MinimumVisibleSize uint64
}

func (i IcebergOrder) Clone() *IcebergOrder {
	cpy := i
	return &cpy
}

func NewIcebergOrderFromProto(i *proto.IcebergOrder) (*IcebergOrder, error) {
	if i == nil {
		return nil, nil
	}
	return &IcebergOrder{
		ReservedRemaining:  i.ReservedRemaining,
		PeakSize:           i.PeakSize,
		MinimumVisibleSize: i.MinimumVisibleSize,
	}, nil
}

func (i IcebergOrder) IntoProto() *proto.IcebergOrder {
	return &proto.IcebergOrder{
		ReservedRemaining:  i.ReservedRemaining,
		PeakSize:           i.PeakSize,
		MinimumVisibleSize: i.MinimumVisibleSize,
	}
}

func (i IcebergOrder) String() string {
	return fmt.Sprintf(
		"reserved-remaining(%d) initial-peak-size(%d) minimum-peak-size(%d)",
		i.ReservedRemaining,
		i.PeakSize,
		i.MinimumVisibleSize,
	)
}

type OrderConfirmation struct {
	Order                 *Order
	Trades                []*Trade
	PassiveOrdersAffected []*Order
}

func (o *OrderConfirmation) IntoProto() *proto.OrderConfirmation {
	return &proto.OrderConfirmation{
		Order:                 o.Order.IntoProto(),
		Trades:                Trades(o.Trades).IntoProto(),
		PassiveOrdersAffected: Orders(o.PassiveOrdersAffected).IntoProto(),
	}
}

func (o OrderConfirmation) TradedValue() *num.Uint {
	total := num.UintZero()
	for _, t := range o.Trades {
		size := num.NewUint(t.Size)
		total.AddSum(size.Mul(size, t.Price))
	}
	return total
}

type OrderCancellationConfirmation struct {
	Order *Order
}

func (o *OrderCancellationConfirmation) IntoProto() *proto.OrderCancellationConfirmation {
	return &proto.OrderCancellationConfirmation{
		Order: o.Order.IntoProto(),
	}
}

type Trade struct {
	ID                 string
	MarketID           string
	Price              *num.Uint
	MarketPrice        *num.Uint
	Size               uint64
	Buyer              string
	Seller             string
	Aggressor          Side
	BuyOrder           string
	SellOrder          string
	Timestamp          int64
	Type               TradeType
	BuyerFee           *Fee
	SellerFee          *Fee
	BuyerAuctionBatch  uint64
	SellerAuctionBatch uint64
}

func (t *Trade) SetIDs(tradeID string, aggressive, passive *Order) {
	t.ID = tradeID
	if aggressive.Side == SideBuy {
		t.BuyOrder = aggressive.ID
		t.SellOrder = passive.ID
		return
	}
	t.SellOrder = aggressive.ID
	t.BuyOrder = passive.ID
}

func (t *Trade) IntoProto() *proto.Trade {
	var buyerFee, sellerFee *proto.Fee
	if t.BuyerFee != nil {
		buyerFee = t.BuyerFee.IntoProto()
	}
	if t.SellerFee != nil {
		sellerFee = t.SellerFee.IntoProto()
	}
	return &proto.Trade{
		Id:                 t.ID,
		MarketId:           t.MarketID,
		Price:              num.UintToString(t.Price),
		Size:               t.Size,
		Buyer:              t.Buyer,
		Seller:             t.Seller,
		Aggressor:          t.Aggressor,
		BuyOrder:           t.BuyOrder,
		SellOrder:          t.SellOrder,
		Timestamp:          t.Timestamp,
		Type:               t.Type,
		BuyerFee:           buyerFee,
		SellerFee:          sellerFee,
		BuyerAuctionBatch:  t.BuyerAuctionBatch,
		SellerAuctionBatch: t.SellerAuctionBatch,
	}
}

func TradeFromProto(t *proto.Trade) *Trade {
	if t == nil {
		return nil
	}

	return &Trade{
		ID:                 t.Id,
		MarketID:           t.MarketId,
		Price:              num.MustUintFromString(t.Price, 10),
		Size:               t.Size,
		Buyer:              t.Buyer,
		Seller:             t.Seller,
		Aggressor:          t.Aggressor,
		BuyOrder:           t.BuyOrder,
		SellOrder:          t.SellOrder,
		Timestamp:          t.Timestamp,
		Type:               t.Type,
		BuyerFee:           FeeFromProto(t.BuyerFee),
		SellerFee:          FeeFromProto(t.SellerFee),
		BuyerAuctionBatch:  t.BuyerAuctionBatch,
		SellerAuctionBatch: t.SellerAuctionBatch,
	}
}

func (t Trade) String() string {
	return fmt.Sprintf(
		"ID(%s) marketID(%s) price(%s) marketPrice(%s) size(%v) buyer(%s) seller(%s) aggressor(%s) buyOrder(%s) sellOrder(%s) timestamp(%v) type(%s) buyerAuctionBatch(%v) sellerAuctionBatch(%v) buyerFee(%s) sellerFee(%s)",
		t.ID,
		t.MarketID,
		stringer.PtrToString(t.Price),
		stringer.PtrToString(t.MarketPrice),
		t.Size,
		t.Buyer,
		t.Seller,
		t.Aggressor.String(),
		t.BuyOrder,
		t.SellOrder,
		t.Timestamp,
		t.Type.String(),
		t.BuyerAuctionBatch,
		t.SellerAuctionBatch,
		stringer.PtrToString(t.SellerFee),
		stringer.PtrToString(t.BuyerFee),
	)
}

type Trades []*Trade

func (t Trades) IntoProto() []*proto.Trade {
	out := make([]*proto.Trade, 0, len(t))
	for _, v := range t {
		out = append(out, v.IntoProto())
	}
	return out
}

type TradeType = proto.Trade_Type

const (
	// Default value, always invalid.
	TradeTypeUnspecified TradeType = proto.Trade_TYPE_UNSPECIFIED
	// Normal trading between two parties.
	TradeTypeDefault TradeType = proto.Trade_TYPE_DEFAULT
	// Trading initiated by the network with another party on the book,
	// which helps to zero-out the positions of one or more distressed parties.
	TradeTypeNetworkCloseOutGood TradeType = proto.Trade_TYPE_NETWORK_CLOSE_OUT_GOOD
	// Trading initiated by the network with another party off the book,
	// with a distressed party in order to zero-out the position of the party.
	TradeTypeNetworkCloseOutBad TradeType = proto.Trade_TYPE_NETWORK_CLOSE_OUT_BAD
)

type PeggedReference = proto.PeggedReference

const (
	// Default value for PeggedReference, no reference given.
	PeggedReferenceUnspecified PeggedReference = proto.PeggedReference_PEGGED_REFERENCE_UNSPECIFIED
	// Mid price reference.
	PeggedReferenceMid PeggedReference = proto.PeggedReference_PEGGED_REFERENCE_MID
	// Best bid price reference.
	PeggedReferenceBestBid PeggedReference = proto.PeggedReference_PEGGED_REFERENCE_BEST_BID
	// Best ask price reference.
	PeggedReferenceBestAsk PeggedReference = proto.PeggedReference_PEGGED_REFERENCE_BEST_ASK
)

type OrderStatus = proto.Order_Status

const (
	// Default value, always invalid.
	OrderStatusUnspecified OrderStatus = proto.Order_STATUS_UNSPECIFIED
	// Used for active unfilled or partially filled orders.
	OrderStatusActive OrderStatus = proto.Order_STATUS_ACTIVE
	// Used for expired GTT orders.
	OrderStatusExpired OrderStatus = proto.Order_STATUS_EXPIRED
	// Used for orders cancelled by the party that created the order.
	OrderStatusCancelled OrderStatus = proto.Order_STATUS_CANCELLED
	// Used for unfilled FOK or IOC orders, and for orders that were stopped by the network.
	OrderStatusStopped OrderStatus = proto.Order_STATUS_STOPPED
	// Used for closed fully filled orders.
	OrderStatusFilled OrderStatus = proto.Order_STATUS_FILLED
	// Used for orders when not enough collateral was available to fill the margin requirements.
	OrderStatusRejected OrderStatus = proto.Order_STATUS_REJECTED
	// Used for closed partially filled IOC orders.
	OrderStatusPartiallyFilled OrderStatus = proto.Order_STATUS_PARTIALLY_FILLED
	// Order has been removed from the order book and has been parked, this applies to pegged orders only.
	OrderStatusParked OrderStatus = proto.Order_STATUS_PARKED
)

type Side = proto.Side

const (
	// Default value, always invalid.
	SideUnspecified Side = proto.Side_SIDE_UNSPECIFIED
	// Buy order.
	SideBuy Side = proto.Side_SIDE_BUY
	// Sell order.
	SideSell Side = proto.Side_SIDE_SELL
)

type OrderType = proto.Order_Type

const (
	// Default value, always invalid.
	OrderTypeUnspecified OrderType = proto.Order_TYPE_UNSPECIFIED
	// Used for Limit orders.
	OrderTypeLimit OrderType = proto.Order_TYPE_LIMIT
	// Used for Market orders.
	OrderTypeMarket OrderType = proto.Order_TYPE_MARKET
	// Used for orders where the initiating party is the network (with distressed traders).
	OrderTypeNetwork OrderType = proto.Order_TYPE_NETWORK
)

type OrderTimeInForce = proto.Order_TimeInForce

const (
	// Default value for TimeInForce, can be valid for an amend.
	OrderTimeInForceUnspecified OrderTimeInForce = proto.Order_TIME_IN_FORCE_UNSPECIFIED
	// Good until cancelled.
	OrderTimeInForceGTC OrderTimeInForce = proto.Order_TIME_IN_FORCE_GTC
	// Good until specified time.
	OrderTimeInForceGTT OrderTimeInForce = proto.Order_TIME_IN_FORCE_GTT
	// Immediate or cancel.
	OrderTimeInForceIOC OrderTimeInForce = proto.Order_TIME_IN_FORCE_IOC
	// Fill or kill.
	OrderTimeInForceFOK OrderTimeInForce = proto.Order_TIME_IN_FORCE_FOK
	// Good for auction.
	OrderTimeInForceGFA OrderTimeInForce = proto.Order_TIME_IN_FORCE_GFA
	// Good for normal.
	OrderTimeInForceGFN OrderTimeInForce = proto.Order_TIME_IN_FORCE_GFN
)

type OrderError = proto.OrderError

const (
	// Default value, no error reported.
	OrderErrorUnspecified OrderError = proto.OrderError_ORDER_ERROR_UNSPECIFIED
	// Order was submitted for a market that does not exist.
	OrderErrorInvalidMarketID OrderError = proto.OrderError_ORDER_ERROR_INVALID_MARKET_ID
	// Order was submitted with an invalid identifier.
	OrderErrorInvalidOrderID OrderError = proto.OrderError_ORDER_ERROR_INVALID_ORDER_ID
	// Order was amended with a sequence number that was not previous version + 1.
	OrderErrorOutOfSequence OrderError = proto.OrderError_ORDER_ERROR_OUT_OF_SEQUENCE
	// Order was amended with an invalid remaining size (e.g. remaining greater than total size).
	OrderErrorInvalidRemainingSize OrderError = proto.OrderError_ORDER_ERROR_INVALID_REMAINING_SIZE
	// Node was unable to get Vega (blockchain) time.
	OrderErrorTimeFailure OrderError = proto.OrderError_ORDER_ERROR_TIME_FAILURE
	// Failed to remove an order from the book.
	OrderErrorRemovalFailure OrderError = proto.OrderError_ORDER_ERROR_REMOVAL_FAILURE
	// An order with `TimeInForce.TIME_IN_FORCE_GTT` was submitted or amended
	// with an expiration that was badly formatted or otherwise invalid.
	OrderErrorInvalidExpirationDatetime OrderError = proto.OrderError_ORDER_ERROR_INVALID_EXPIRATION_DATETIME
	// Order was submitted or amended with an invalid reference field.
	OrderErrorInvalidOrderReference OrderError = proto.OrderError_ORDER_ERROR_INVALID_ORDER_REFERENCE
	// Order amend was submitted for an order field that cannot not be amended (e.g. order identifier).
	OrderErrorEditNotAllowed OrderError = proto.OrderError_ORDER_ERROR_EDIT_NOT_ALLOWED
	// Amend failure because amend details do not match original order.
	OrderErrorAmendFailure OrderError = proto.OrderError_ORDER_ERROR_AMEND_FAILURE
	// Order not found in an order book or store.
	OrderErrorNotFound OrderError = proto.OrderError_ORDER_ERROR_NOT_FOUND
	// Order was submitted with an invalid or missing party identifier.
	OrderErrorInvalidParty OrderError = proto.OrderError_ORDER_ERROR_INVALID_PARTY_ID
	// Order was submitted for a market that has closed.
	OrderErrorMarketClosed OrderError = proto.OrderError_ORDER_ERROR_MARKET_CLOSED
	// Order was submitted, but the party did not have enough collateral to cover the order.
	OrderErrorMarginCheckFailed OrderError = proto.OrderError_ORDER_ERROR_MARGIN_CHECK_FAILED
	// Order was submitted, but the party did not have an account for this asset.
	OrderErrorMissingGeneralAccount OrderError = proto.OrderError_ORDER_ERROR_MISSING_GENERAL_ACCOUNT
	// Unspecified internal error.
	OrderErrorInternalError OrderError = proto.OrderError_ORDER_ERROR_INTERNAL_ERROR
	// Order was submitted with an invalid or missing size (e.g. 0).
	OrderErrorInvalidSize OrderError = proto.OrderError_ORDER_ERROR_INVALID_SIZE
	// Order was submitted with an invalid persistence for its type.
	OrderErrorInvalidPersistance OrderError = proto.OrderError_ORDER_ERROR_INVALID_PERSISTENCE
	// Order was submitted with an invalid type field.
	OrderErrorInvalidType OrderError = proto.OrderError_ORDER_ERROR_INVALID_TYPE
	// Order was stopped as it would have traded with another order submitted from the same party.
	OrderErrorSelfTrading OrderError = proto.OrderError_ORDER_ERROR_SELF_TRADING
	// Order was submitted, but the party did not have enough collateral to cover the fees for the order.
	OrderErrorInsufficientFundsToPayFees OrderError = proto.OrderError_ORDER_ERROR_INSUFFICIENT_FUNDS_TO_PAY_FEES
	// Order was submitted with an incorrect or invalid market type.
	OrderErrorIncorrectMarketType OrderError = proto.OrderError_ORDER_ERROR_INCORRECT_MARKET_TYPE
	// Order was submitted with invalid time in force.
	OrderErrorInvalidTimeInForce OrderError = proto.OrderError_ORDER_ERROR_INVALID_TIME_IN_FORCE
	// A GFN order has got to the market when it is in auction mode.
	OrderErrorCannotSendGFNOrderDuringAnAuction OrderError = proto.OrderError_ORDER_ERROR_CANNOT_SEND_GFN_ORDER_DURING_AN_AUCTION
	// A GFA order has got to the market when it is in continuous trading mode.
	OrderErrorGFAOrderDuringContinuousTrading OrderError = proto.OrderError_ORDER_ERROR_CANNOT_SEND_GFA_ORDER_DURING_CONTINUOUS_TRADING
	// Attempt to amend order to GTT without ExpiryAt.
	OrderErrorCannotAmendToGTTWithoutExpiryAt OrderError = proto.OrderError_ORDER_ERROR_CANNOT_AMEND_TO_GTT_WITHOUT_EXPIRYAT
	// Attempt to amend ExpiryAt to a value before CreatedAt.
	OrderErrorExpiryAtBeforeCreatedAt OrderError = proto.OrderError_ORDER_ERROR_EXPIRYAT_BEFORE_CREATEDAT
	// Attempt to amend to GTC without an ExpiryAt value.
	OrderErrorCannotHaveGTCAndExpiryAt OrderError = proto.OrderError_ORDER_ERROR_CANNOT_HAVE_GTC_AND_EXPIRYAT
	// Amending to FOK or IOC is invalid.
	OrderErrorCannotAmendToFOKOrIOC OrderError = proto.OrderError_ORDER_ERROR_CANNOT_AMEND_TO_FOK_OR_IOC
	// Amending to GFA or GFN is invalid.
	OrderErrorCannotAmendToGFAOrGFN OrderError = proto.OrderError_ORDER_ERROR_CANNOT_AMEND_TO_GFA_OR_GFN
	// Amending from GFA or GFN is invalid.
	OrderErrorCannotAmendFromGFAOrGFN OrderError = proto.OrderError_ORDER_ERROR_CANNOT_AMEND_FROM_GFA_OR_GFN
	// IOC orders are not allowed during auction.
	OrderErrorCannotSendIOCOrderDuringAuction OrderError = proto.OrderError_ORDER_ERROR_CANNOT_SEND_IOC_ORDER_DURING_AUCTION
	// FOK orders are not allowed during auction.
	OrderErrorCannotSendFOKOrderDurinAuction OrderError = proto.OrderError_ORDER_ERROR_CANNOT_SEND_FOK_ORDER_DURING_AUCTION
	// Pegged orders must be LIMIT orders.
	OrderErrorMustBeLimitOrder OrderError = proto.OrderError_ORDER_ERROR_MUST_BE_LIMIT_ORDER
	// Pegged orders can only have TIF GTC or GTT.
	OrderErrorMustBeGTTOrGTC OrderError = proto.OrderError_ORDER_ERROR_MUST_BE_GTT_OR_GTC
	// Pegged order must have a reference price.
	OrderErrorWithoutReferencePrice OrderError = proto.OrderError_ORDER_ERROR_WITHOUT_REFERENCE_PRICE
	// Buy pegged order cannot reference best ask price.
	OrderErrorBuyCannotReferenceBestAskPrice OrderError = proto.OrderError_ORDER_ERROR_BUY_CANNOT_REFERENCE_BEST_ASK_PRICE
	// Pegged order offset must be >= 0.
	OrderErrorOffsetMustBeGreaterOrEqualToZero OrderError = proto.OrderError_ORDER_ERROR_OFFSET_MUST_BE_GREATER_OR_EQUAL_TO_ZERO
	// Sell pegged order cannot reference best bid price.
	OrderErrorSellCannotReferenceBestBidPrice OrderError = proto.OrderError_ORDER_ERROR_SELL_CANNOT_REFERENCE_BEST_BID_PRICE
	// Pegged order offset must be > zero.
	OrderErrorOffsetMustBeGreaterThanZero OrderError = proto.OrderError_ORDER_ERROR_OFFSET_MUST_BE_GREATER_THAN_ZERO
	// The party has an insufficient balance, or does not have
	// a general account to submit the order (no deposits made
	// for the required asset).
	OrderErrorInsufficientAssetBalance OrderError = proto.OrderError_ORDER_ERROR_INSUFFICIENT_ASSET_BALANCE
	// Cannot amend a non pegged orders details.
	OrderErrorCannotAmendPeggedOrderDetailsOnNonPeggedOrder OrderError = proto.OrderError_ORDER_ERROR_CANNOT_AMEND_PEGGED_ORDER_DETAILS_ON_NON_PEGGED_ORDER
	// We are unable to re-price a pegged order because a market price is unavailable.
	OrderErrorUnableToRepricePeggedOrder OrderError = proto.OrderError_ORDER_ERROR_UNABLE_TO_REPRICE_PEGGED_ORDER
	// It is not possible to amend the price of an existing pegged order.
	OrderErrorUnableToAmendPriceOnPeggedOrder OrderError = proto.OrderError_ORDER_ERROR_UNABLE_TO_AMEND_PRICE_ON_PEGGED_ORDER
	// An FOK, IOC, or GFN order was rejected because it resulted in trades outside the price bounds.
	OrderErrorNonPersistentOrderOutOfPriceBounds OrderError = proto.OrderError_ORDER_ERROR_NON_PERSISTENT_ORDER_OUT_OF_PRICE_BOUNDS
	// Unable to submit pegged order, temporarily too many pegged orders across all markets.
	OrderErrorTooManyPeggedOrders                    OrderError = proto.OrderError_ORDER_ERROR_TOO_MANY_PEGGED_ORDERS
	OrderErrorPostOnlyOrderWouldTrade                OrderError = proto.OrderError_ORDER_ERROR_POST_ONLY_ORDER_WOULD_TRADE
	OrderErrorReduceOnlyOrderWouldNotReducePosition  OrderError = proto.OrderError_ORDER_ERROR_REDUCE_ONLY_ORDER_WOULD_NOT_REDUCE_POSITION
	OrderErrorIsolatedMarginCheckFailed              OrderError = proto.OrderError_ORDER_ERROR_ISOLATED_MARGIN_CHECK_FAILED
	OrderErrorPeggedOrdersNotAllowedInIsolatedMargin OrderError = proto.OrderError_ORDER_ERROR_PEGGED_ORDERS_NOT_ALLOWED_IN_ISOLATED_MARGIN_MODE
	OrderErrorPriceNotInTickSize                     OrderError = proto.OrderError_ORDER_ERROR_PRICE_NOT_IN_TICK_SIZE
)

var (
	ErrInvalidMarketID                             = OrderErrorInvalidMarketID
	ErrInvalidOrderID                              = OrderErrorInvalidOrderID
	ErrOrderOutOfSequence                          = OrderErrorOutOfSequence
	ErrInvalidRemainingSize                        = OrderErrorInvalidRemainingSize
	ErrOrderRemovalFailure                         = OrderErrorRemovalFailure
	ErrInvalidExpirationDatetime                   = OrderErrorInvalidExpirationDatetime
	ErrEditNotAllowed                              = OrderErrorEditNotAllowed
	ErrOrderAmendFailure                           = OrderErrorAmendFailure
	ErrOrderNotFound                               = OrderErrorNotFound
	ErrInvalidPartyID                              = OrderErrorInvalidParty
	ErrInvalidSize                                 = OrderErrorInvalidSize
	ErrInvalidPersistence                          = OrderErrorInvalidPersistance
	ErrInvalidType                                 = OrderErrorInvalidType
	ErrInvalidTimeInForce                          = OrderErrorInvalidTimeInForce
	ErrPeggedOrderMustBeLimitOrder                 = OrderErrorMustBeLimitOrder
	ErrPeggedOrderMustBeGTTOrGTC                   = OrderErrorMustBeGTTOrGTC
	ErrPeggedOrderWithoutReferencePrice            = OrderErrorWithoutReferencePrice
	ErrPeggedOrderBuyCannotReferenceBestAskPrice   = OrderErrorBuyCannotReferenceBestAskPrice
	ErrPeggedOrderOffsetMustBeGreaterOrEqualToZero = OrderErrorOffsetMustBeGreaterOrEqualToZero
	ErrPeggedOrderSellCannotReferenceBestBidPrice  = OrderErrorSellCannotReferenceBestBidPrice
	ErrPeggedOrderOffsetMustBeGreaterThanZero      = OrderErrorOffsetMustBeGreaterThanZero
	ErrTooManyPeggedOrders                         = OrderErrorTooManyPeggedOrders
	ErrPostOnlyOrderWouldTrade                     = OrderErrorPostOnlyOrderWouldTrade
	ErrReduceOnlyOrderWouldNotReducePosition       = OrderErrorReduceOnlyOrderWouldNotReducePosition
	ErrPeggedOrdersNotAllowedInIsolatedMargin      = OrderErrorPeggedOrdersNotAllowedInIsolatedMargin
	ErrOrderNotInTickSize                          = OrderErrorPriceNotInTickSize
)

func OtherSide(s Side) Side {
	switch s {
	case SideBuy:
		return SideSell
	case SideSell:
		return SideBuy
	}
	return SideUnspecified
}

func IsOrderError(err error) (OrderError, bool) {
	oerr, ok := err.(OrderError)
	return oerr, ok
}

func IsStoppingOrder(o OrderError) bool {
	return o == OrderErrorNonPersistentOrderOutOfPriceBounds ||
		o == ErrPostOnlyOrderWouldTrade ||
		o == ErrReduceOnlyOrderWouldNotReducePosition ||
		o == OrderErrorIsolatedMarginCheckFailed ||
		o == OrderErrorPeggedOrdersNotAllowedInIsolatedMargin
}
