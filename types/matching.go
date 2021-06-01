//lint:file-ignore ST1003 Ignore underscores in names, this is straigh copied from the proto package to ease introducing the domain types

package types

import (
	"fmt"
	"time"

	"code.vegaprotocol.io/vega/proto"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

type Order struct {
	Id                   string
	MarketId             string
	PartyId              string
	Side                 Side
	Price                uint64
	Size                 uint64
	Remaining            uint64
	TimeInForce          Order_TimeInForce
	Type                 Order_Type
	CreatedAt            int64
	Status               Order_Status
	ExpiresAt            int64
	Reference            string
	Reason               OrderError
	UpdatedAt            int64
	Version              uint64
	BatchId              uint64
	PeggedOrder          *PeggedOrder
	LiquidityProvisionId string
}

func (o Order) String() string {
	return o.IntoProto().String()
}

type Orders []*Order

func (o Orders) IntoProto() []*proto.Order {
	out := make([]*proto.Order, 0, len(o))
	for _, v := range o {
		out = append(out, v.IntoProto())
	}
	return out
}

func (o *Order) IsLiquidityOrder() bool {
	return len(o.LiquidityProvisionId) > 0
}

func (o *Order) IntoProto() *proto.Order {
	var pegged *proto.PeggedOrder
	if o.PeggedOrder != nil {
		pegged = o.PeggedOrder.IntoProto()
	}
	return &proto.Order{
		Id:                   o.Id,
		MarketId:             o.MarketId,
		PartyId:              o.PartyId,
		Side:                 o.Side,
		Price:                o.Price,
		Size:                 o.Size,
		Remaining:            o.Remaining,
		TimeInForce:          o.TimeInForce,
		Type:                 o.Type,
		CreatedAt:            o.CreatedAt,
		Status:               o.Status,
		ExpiresAt:            o.ExpiresAt,
		Reference:            o.Reference,
		Reason:               o.Reason,
		UpdatedAt:            o.UpdatedAt,
		Version:              o.Version,
		BatchId:              o.BatchId,
		PeggedOrder:          pegged,
		LiquidityProvisionId: o.LiquidityProvisionId,
	}
}

func OrderFromProto(o *proto.Order) *Order {
	var pegged *PeggedOrder
	if o.PeggedOrder != nil {
		pegged = PeggedOrderFromProto(o.PeggedOrder)
	}
	return &Order{
		Id:                   o.Id,
		MarketId:             o.MarketId,
		PartyId:              o.PartyId,
		Side:                 o.Side,
		Price:                o.Price,
		Size:                 o.Size,
		Remaining:            o.Remaining,
		TimeInForce:          o.TimeInForce,
		Type:                 o.Type,
		CreatedAt:            o.CreatedAt,
		Status:               o.Status,
		ExpiresAt:            o.ExpiresAt,
		Reference:            o.Reference,
		Reason:               o.Reason,
		UpdatedAt:            o.UpdatedAt,
		Version:              o.Version,
		BatchId:              o.BatchId,
		PeggedOrder:          pegged,
		LiquidityProvisionId: o.LiquidityProvisionId,
	}
}

// Create sets the creation time (CreatedAt) to t and returns the
// updated order.
func (o *Order) Create(t time.Time) *Order {
	o.CreatedAt = t.UnixNano()
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
	return (o.TimeInForce == Order_TIME_IN_FORCE_GTC ||
		o.TimeInForce == Order_TIME_IN_FORCE_GTT ||
		o.TimeInForce == Order_TIME_IN_FORCE_GFN ||
		o.TimeInForce == Order_TIME_IN_FORCE_GFA) &&
		o.Type == Order_TYPE_LIMIT &&
		o.Remaining > 0
}

func (o *Order) AmendSize(newSize int64) *OrderAmendment {
	a := &OrderAmendment{
		OrderId:  o.Id,
		MarketId: o.MarketId,

		SizeDelta:   newSize - int64(o.Size),
		TimeInForce: o.TimeInForce,
	}
	if e := o.ExpiresAt; e > 0 {
		a.ExpiresAt = &Timestamp{
			Value: e,
		}
	}

	if p := o.PeggedOrder; p != nil {
		a.PeggedReference = p.Reference
		a.PeggedOffset = &wrapperspb.Int64Value{
			Value: p.Offset,
		}
	} else {
		if p := o.Price; p > 0 {
			a.Price = &Price{
				Value: p,
			}
		}
	}

	return a
}

func (o *Order) IsExpireable() bool {
	return (o.TimeInForce == Order_TIME_IN_FORCE_GFN ||
		o.TimeInForce == Order_TIME_IN_FORCE_GTT ||
		o.TimeInForce == Order_TIME_IN_FORCE_GFA) &&
		o.ExpiresAt > 0
}

// IsFinished returns true if an order
// is in any state different to ACTIVE and PARKED
// Basically any order which is never gonna
// trade anymore
func (o *Order) IsFinished() bool {
	return o.Status != Order_STATUS_ACTIVE && o.Status != Order_STATUS_PARKED
}

func (o *Order) HasTraded() bool {
	return o.Size != o.Remaining
}

type PeggedOrder struct {
	Reference PeggedReference
	Offset    int64
}

func PeggedOrderFromProto(p *proto.PeggedOrder) *PeggedOrder {
	if p == nil {
		return nil
	}
	return &PeggedOrder{
		Reference: p.Reference,
		Offset:    p.Offset,
	}
}

func (p PeggedOrder) IntoProto() *proto.PeggedOrder {
	return &proto.PeggedOrder{
		Reference: p.Reference,
		Offset:    p.Offset,
	}
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

type OrderCancellationConfirmation struct {
	Order *Order
}

func (o *OrderCancellationConfirmation) IntoProto() *proto.OrderCancellationConfirmation {
	return &proto.OrderCancellationConfirmation{
		Order: o.Order.IntoProto(),
	}
}

type Trade struct {
	Id                 string
	MarketId           string
	Price              uint64
	Size               uint64
	Buyer              string
	Seller             string
	Aggressor          Side
	BuyOrder           string
	SellOrder          string
	Timestamp          int64
	Type               Trade_Type
	BuyerFee           *Fee
	SellerFee          *Fee
	BuyerAuctionBatch  uint64
	SellerAuctionBatch uint64
}

func (t *Trade) SetIDs(aggressive, passive *Order, idx int) {
	t.Id = fmt.Sprintf("%s-%010d", aggressive.Id, idx)
	if aggressive.Side == Side_SIDE_BUY {
		t.BuyOrder = aggressive.Id
		t.SellOrder = passive.Id
		return
	}
	t.SellOrder = aggressive.Id
	t.BuyOrder = passive.Id

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
		Id:                 t.Id,
		MarketId:           t.MarketId,
		Price:              t.Price,
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

func (t *Trade) String() string {
	return t.IntoProto().String()
}

type Trades []*Trade

func (t Trades) IntoProto() []*proto.Trade {
	out := make([]*proto.Trade, 0, len(t))
	for _, v := range t {
		out = append(out, v.IntoProto())
	}
	return out
}

type Fee struct {
	MakerFee          uint64
	InfrastructureFee uint64
	LiquidityFee      uint64
}

func (f *Fee) IntoProto() *proto.Fee {
	return &proto.Fee{
		MakerFee:          f.MakerFee,
		InfrastructureFee: f.InfrastructureFee,
		LiquidityFee:      f.LiquidityFee,
	}
}

type Trade_Type = proto.Trade_Type

const (
	// Default value, always invalid
	Trade_TYPE_UNSPECIFIED Trade_Type = 0
	// Normal trading between two parties
	Trade_TYPE_DEFAULT Trade_Type = 1
	// Trading initiated by the network with another party on the book,
	// which helps to zero-out the positions of one or more distressed parties
	Trade_TYPE_NETWORK_CLOSE_OUT_GOOD Trade_Type = 2
	// Trading initiated by the network with another party off the book,
	// with a distressed party in order to zero-out the position of the party
	Trade_TYPE_NETWORK_CLOSE_OUT_BAD Trade_Type = 3
)

type PeggedReference = proto.PeggedReference

const (
	// Default value for PeggedReference, no reference given
	PeggedReference_PEGGED_REFERENCE_UNSPECIFIED PeggedReference = 0
	// Mid price reference
	PeggedReference_PEGGED_REFERENCE_MID PeggedReference = 1
	// Best bid price reference
	PeggedReference_PEGGED_REFERENCE_BEST_BID PeggedReference = 2
	// Best ask price reference
	PeggedReference_PEGGED_REFERENCE_BEST_ASK PeggedReference = 3
)

type Order_Status = proto.Order_Status

const (
	// Default value, always invalid
	Order_STATUS_UNSPECIFIED Order_Status = 0
	// Used for active unfilled or partially filled orders
	Order_STATUS_ACTIVE Order_Status = 1
	// Used for expired GTT orders
	Order_STATUS_EXPIRED Order_Status = 2
	// Used for orders cancelled by the party that created the order
	Order_STATUS_CANCELLED Order_Status = 3
	// Used for unfilled FOK or IOC orders, and for orders that were stopped by the network
	Order_STATUS_STOPPED Order_Status = 4
	// Used for closed fully filled orders
	Order_STATUS_FILLED Order_Status = 5
	// Used for orders when not enough collateral was available to fill the margin requirements
	Order_STATUS_REJECTED Order_Status = 6
	// Used for closed partially filled IOC orders
	Order_STATUS_PARTIALLY_FILLED Order_Status = 7
	// Order has been removed from the order book and has been parked, this applies to pegged orders only
	Order_STATUS_PARKED Order_Status = 8
)

type Side = proto.Side

const (
	// Default value, always invalid
	Side_SIDE_UNSPECIFIED Side = 0
	// Buy order
	Side_SIDE_BUY Side = 1
	// Sell order
	Side_SIDE_SELL Side = 2
)

type Order_Type = proto.Order_Type

const (
	// Default value, always invalid
	Order_TYPE_UNSPECIFIED Order_Type = 0
	// Used for Limit orders
	Order_TYPE_LIMIT Order_Type = 1
	// Used for Market orders
	Order_TYPE_MARKET Order_Type = 2
	// Used for orders where the initiating party is the network (with distressed traders)
	Order_TYPE_NETWORK Order_Type = 3
)

type Order_TimeInForce = proto.Order_TimeInForce

const (
	// Default value for TimeInForce, can be valid for an amend
	Order_TIME_IN_FORCE_UNSPECIFIED Order_TimeInForce = 0
	// Good until cancelled
	Order_TIME_IN_FORCE_GTC Order_TimeInForce = 1
	// Good until specified time
	Order_TIME_IN_FORCE_GTT Order_TimeInForce = 2
	// Immediate or cancel
	Order_TIME_IN_FORCE_IOC Order_TimeInForce = 3
	// Fill or kill
	Order_TIME_IN_FORCE_FOK Order_TimeInForce = 4
	// Good for auction
	Order_TIME_IN_FORCE_GFA Order_TimeInForce = 5
	// Good for normal
	Order_TIME_IN_FORCE_GFN Order_TimeInForce = 6
)

type OrderError = proto.OrderError

const (
	// Default value, no error reported
	OrderError_ORDER_ERROR_UNSPECIFIED OrderError = 0
	// Order was submitted for a market that does not exist
	OrderError_ORDER_ERROR_INVALID_MARKET_ID OrderError = 1
	// Order was submitted with an invalid identifier
	OrderError_ORDER_ERROR_INVALID_ORDER_ID OrderError = 2
	// Order was amended with a sequence number that was not previous version + 1
	OrderError_ORDER_ERROR_OUT_OF_SEQUENCE OrderError = 3
	// Order was amended with an invalid remaining size (e.g. remaining greater than total size)
	OrderError_ORDER_ERROR_INVALID_REMAINING_SIZE OrderError = 4
	// Node was unable to get Vega (blockchain) time
	OrderError_ORDER_ERROR_TIME_FAILURE OrderError = 5
	// Failed to remove an order from the book
	OrderError_ORDER_ERROR_REMOVAL_FAILURE OrderError = 6
	// An order with `TimeInForce.TIME_IN_FORCE_GTT` was submitted or amended
	// with an expiration that was badly formatted or otherwise invalid
	OrderError_ORDER_ERROR_INVALID_EXPIRATION_DATETIME OrderError = 7
	// Order was submitted or amended with an invalid reference field
	OrderError_ORDER_ERROR_INVALID_ORDER_REFERENCE OrderError = 8
	// Order amend was submitted for an order field that cannot not be amended (e.g. order identifier)
	OrderError_ORDER_ERROR_EDIT_NOT_ALLOWED OrderError = 9
	// Amend failure because amend details do not match original order
	OrderError_ORDER_ERROR_AMEND_FAILURE OrderError = 10
	// Order not found in an order book or store
	OrderError_ORDER_ERROR_NOT_FOUND OrderError = 11
	// Order was submitted with an invalid or missing party identifier
	OrderError_ORDER_ERROR_INVALID_PARTY_ID OrderError = 12
	// Order was submitted for a market that has closed
	OrderError_ORDER_ERROR_MARKET_CLOSED OrderError = 13
	// Order was submitted, but the party did not have enough collateral to cover the order
	OrderError_ORDER_ERROR_MARGIN_CHECK_FAILED OrderError = 14
	// Order was submitted, but the party did not have an account for this asset
	OrderError_ORDER_ERROR_MISSING_GENERAL_ACCOUNT OrderError = 15
	// Unspecified internal error
	OrderError_ORDER_ERROR_INTERNAL_ERROR OrderError = 16
	// Order was submitted with an invalid or missing size (e.g. 0)
	OrderError_ORDER_ERROR_INVALID_SIZE OrderError = 17
	// Order was submitted with an invalid persistence for its type
	OrderError_ORDER_ERROR_INVALID_PERSISTENCE OrderError = 18
	// Order was submitted with an invalid type field
	OrderError_ORDER_ERROR_INVALID_TYPE OrderError = 19
	// Order was stopped as it would have traded with another order submitted from the same party
	OrderError_ORDER_ERROR_SELF_TRADING OrderError = 20
	// Order was submitted, but the party did not have enough collateral to cover the fees for the order
	OrderError_ORDER_ERROR_INSUFFICIENT_FUNDS_TO_PAY_FEES OrderError = 21
	// Order was submitted with an incorrect or invalid market type
	OrderError_ORDER_ERROR_INCORRECT_MARKET_TYPE OrderError = 22
	// Order was submitted with invalid time in force
	OrderError_ORDER_ERROR_INVALID_TIME_IN_FORCE OrderError = 23
	// A GFN order has got to the market when it is in auction mode
	OrderError_ORDER_ERROR_GFN_ORDER_DURING_AN_AUCTION OrderError = 24
	// A GFA order has got to the market when it is in continuous trading mode
	OrderError_ORDER_ERROR_GFA_ORDER_DURING_CONTINUOUS_TRADING OrderError = 25
	// Attempt to amend order to GTT without ExpiryAt
	OrderError_ORDER_ERROR_CANNOT_AMEND_TO_GTT_WITHOUT_EXPIRYAT OrderError = 26
	// Attempt to amend ExpiryAt to a value before CreatedAt
	OrderError_ORDER_ERROR_EXPIRYAT_BEFORE_CREATEDAT OrderError = 27
	// Attempt to amend to GTC without an ExpiryAt value
	OrderError_ORDER_ERROR_CANNOT_HAVE_GTC_AND_EXPIRYAT OrderError = 28
	// Amending to FOK or IOC is invalid
	OrderError_ORDER_ERROR_CANNOT_AMEND_TO_FOK_OR_IOC OrderError = 29
	// Amending to GFA or GFN is invalid
	OrderError_ORDER_ERROR_CANNOT_AMEND_TO_GFA_OR_GFN OrderError = 30
	// Amending from GFA or GFN is invalid
	OrderError_ORDER_ERROR_CANNOT_AMEND_FROM_GFA_OR_GFN OrderError = 31
	// IOC orders are not allowed during auction
	OrderError_ORDER_ERROR_CANNOT_SEND_IOC_ORDER_DURING_AUCTION OrderError = 32
	// FOK orders are not allowed during auction
	OrderError_ORDER_ERROR_CANNOT_SEND_FOK_ORDER_DURING_AUCTION OrderError = 33
	// Pegged orders must be LIMIT orders
	OrderError_ORDER_ERROR_MUST_BE_LIMIT_ORDER OrderError = 34
	// Pegged orders can only have TIF GTC or GTT
	OrderError_ORDER_ERROR_MUST_BE_GTT_OR_GTC OrderError = 35
	// Pegged order must have a reference price
	OrderError_ORDER_ERROR_WITHOUT_REFERENCE_PRICE OrderError = 36
	// Buy pegged order cannot reference best ask price
	OrderError_ORDER_ERROR_BUY_CANNOT_REFERENCE_BEST_ASK_PRICE OrderError = 37
	// Pegged order offset must be <= 0
	OrderError_ORDER_ERROR_OFFSET_MUST_BE_LESS_OR_EQUAL_TO_ZERO OrderError = 38
	// Pegged order offset must be < 0
	OrderError_ORDER_ERROR_OFFSET_MUST_BE_LESS_THAN_ZERO OrderError = 39
	// Pegged order offset must be >= 0
	OrderError_ORDER_ERROR_OFFSET_MUST_BE_GREATER_OR_EQUAL_TO_ZERO OrderError = 40
	// Sell pegged order cannot reference best bid price
	OrderError_ORDER_ERROR_SELL_CANNOT_REFERENCE_BEST_BID_PRICE OrderError = 41
	// Pegged order offset must be > zero
	OrderError_ORDER_ERROR_OFFSET_MUST_BE_GREATER_THAN_ZERO OrderError = 42
	// The party has an insufficient balance, or does not have
	// a general account to submit the order (no deposits made
	// for the required asset)
	OrderError_ORDER_ERROR_INSUFFICIENT_ASSET_BALANCE OrderError = 43
	// Cannot amend a non pegged orders details
	OrderError_ORDER_ERROR_CANNOT_AMEND_PEGGED_ORDER_DETAILS_ON_NON_PEGGED_ORDER OrderError = 44
	// We are unable to re-price a pegged order because a market price is unavailable
	OrderError_ORDER_ERROR_UNABLE_TO_REPRICE_PEGGED_ORDER OrderError = 45
	// It is not possible to amend the price of an existing pegged order
	OrderError_ORDER_ERROR_UNABLE_TO_AMEND_PRICE_ON_PEGGED_ORDER OrderError = 46
	// An FOK, IOC, or GFN order was rejected because it resulted in trades outside the price bounds
	OrderError_ORDER_ERROR_NON_PERSISTENT_ORDER_OUT_OF_PRICE_BOUNDS OrderError = 47
)

var (
	ErrInvalidMarketID                             = OrderError_ORDER_ERROR_INVALID_MARKET_ID
	ErrInvalidOrderID                              = OrderError_ORDER_ERROR_INVALID_ORDER_ID
	ErrOrderOutOfSequence                          = OrderError_ORDER_ERROR_OUT_OF_SEQUENCE
	ErrInvalidRemainingSize                        = OrderError_ORDER_ERROR_INVALID_REMAINING_SIZE
	ErrOrderRemovalFailure                         = OrderError_ORDER_ERROR_REMOVAL_FAILURE
	ErrInvalidExpirationDatetime                   = OrderError_ORDER_ERROR_INVALID_EXPIRATION_DATETIME
	ErrEditNotAllowed                              = OrderError_ORDER_ERROR_EDIT_NOT_ALLOWED
	ErrOrderAmendFailure                           = OrderError_ORDER_ERROR_AMEND_FAILURE
	ErrOrderNotFound                               = OrderError_ORDER_ERROR_NOT_FOUND
	ErrInvalidPartyID                              = OrderError_ORDER_ERROR_INVALID_PARTY_ID
	ErrInvalidSize                                 = OrderError_ORDER_ERROR_INVALID_SIZE
	ErrInvalidPersistence                          = OrderError_ORDER_ERROR_INVALID_PERSISTENCE
	ErrInvalidType                                 = OrderError_ORDER_ERROR_INVALID_TYPE
	ErrInvalidTimeInForce                          = OrderError_ORDER_ERROR_INVALID_TIME_IN_FORCE
	ErrPeggedOrderMustBeLimitOrder                 = OrderError_ORDER_ERROR_MUST_BE_LIMIT_ORDER
	ErrPeggedOrderMustBeGTTOrGTC                   = OrderError_ORDER_ERROR_MUST_BE_GTT_OR_GTC
	ErrPeggedOrderWithoutReferencePrice            = OrderError_ORDER_ERROR_WITHOUT_REFERENCE_PRICE
	ErrPeggedOrderBuyCannotReferenceBestAskPrice   = OrderError_ORDER_ERROR_BUY_CANNOT_REFERENCE_BEST_ASK_PRICE
	ErrPeggedOrderOffsetMustBeLessOrEqualToZero    = OrderError_ORDER_ERROR_OFFSET_MUST_BE_LESS_OR_EQUAL_TO_ZERO
	ErrPeggedOrderOffsetMustBeLessThanZero         = OrderError_ORDER_ERROR_OFFSET_MUST_BE_LESS_THAN_ZERO
	ErrPeggedOrderOffsetMustBeGreaterOrEqualToZero = OrderError_ORDER_ERROR_OFFSET_MUST_BE_GREATER_OR_EQUAL_TO_ZERO
	ErrPeggedOrderSellCannotReferenceBestBidPrice  = OrderError_ORDER_ERROR_SELL_CANNOT_REFERENCE_BEST_BID_PRICE
	ErrPeggedOrderOffsetMustBeGreaterThanZero      = OrderError_ORDER_ERROR_OFFSET_MUST_BE_GREATER_THAN_ZERO
)

func IsOrderError(err error) (OrderError, bool) {
	oerr, ok := err.(OrderError)
	return oerr, ok
}
