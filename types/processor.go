//lint:file-ignore ST1003 Ignore underscores in names, this is straigh copied from the proto package to ease introducing the domain types

package types

import (
	"code.vegaprotocol.io/vega/proto"
	commandspb "code.vegaprotocol.io/vega/proto/commands/v1"
	"code.vegaprotocol.io/vega/types/num"
	"github.com/golang/protobuf/ptypes/wrappers"
)

type OrderCancellation = commandspb.OrderCancellation
type OracleDataSubmission = commandspb.OracleDataSubmission
type NodeRegistration = commandspb.NodeRegistration
type NodeVote = commandspb.NodeVote
type Transaction = proto.Transaction

type ChainEvent = commandspb.ChainEvent
type SignedBundle = proto.SignedBundle
type NetworkParameter = proto.NetworkParameter
type Signature = proto.Signature
type Transaction_PubKey = proto.Transaction_PubKey

type OrderSubmission struct {
	// Market identifier for the order, required field
	MarketId string
	// Price for the order, the price is an integer, for example `123456` is a correctly
	// formatted price of `1.23456` assuming market configured to 5 decimal places,
	// , required field for limit orders, however it is not required for market orders
	Price *num.Uint
	// Size for the order, for example, in a futures market the size equals the number of contracts, cannot be negative
	Size uint64
	// Side for the order, e.g. SIDE_BUY or SIDE_SELL, required field - See [`Side`](#vega.Side)
	Side proto.Side
	// Time in force indicates how long an order will remain active before it is executed or expires, required field
	// - See [`Order.TimeInForce`](#vega.Order.TimeInForce)
	TimeInForce proto.Order_TimeInForce
	// Timestamp for when the order will expire, in nanoseconds since the epoch,
	// required field only for [`Order.TimeInForce`](#vega.Order.TimeInForce)`.TIME_IN_FORCE_GTT`
	// - See [`VegaTimeResponse`](#api.VegaTimeResponse).`timestamp`
	ExpiresAt int64
	// Type for the order, required field - See [`Order.Type`](#vega.Order.Type)
	Type proto.Order_Type
	// Reference given for the order, this is typically used to retrieve an order submitted through consensus, currently
	// set internally by the node to return a unique reference identifier for the order submission
	Reference string
	// Used to specify the details for a pegged order
	// - See [`PeggedOrder`](#vega.PeggedOrder)
	PeggedOrder *PeggedOrder
}

func (o OrderSubmission) IntoProto() *commandspb.OrderSubmission {
	p := &commandspb.OrderSubmission{
		MarketId: o.MarketId,
		// Need to update protobuf to use string TODO UINT
		Price:       o.Price.Uint64(),
		Size:        o.Size,
		Side:        o.Side,
		TimeInForce: o.TimeInForce,
		ExpiresAt:   o.ExpiresAt,
		Type:        o.Type,
		Reference:   o.Reference,
		PeggedOrder: o.PeggedOrder.IntoProto(),
	}
	return p
}

func NewOrderSubmissionFromProto(p *commandspb.OrderSubmission) (*OrderSubmission, error) {
	o := OrderSubmission{}
	o.MarketId = p.MarketId
	// Need to update protobuf to use string TODO UINT
	o.Price = num.NewUint(p.Price)
	o.Size = p.Size
	o.Side = p.Side
	o.TimeInForce = p.TimeInForce
	o.ExpiresAt = p.ExpiresAt
	o.Type = p.Type
	o.Reference = p.Reference
	o.PeggedOrder = NewPeggedOrderFromProto(p.PeggedOrder)
	return &o, nil
}

func (o OrderSubmission) String() string {
	return o.IntoProto().String()
}

func (o OrderSubmission) IntoOrder(party string) *Order {
	order := &Order{
		MarketId:    o.MarketId,
		PartyId:     party,
		Side:        o.Side,
		Price:       o.Price,
		Size:        o.Size,
		Remaining:   o.Size,
		TimeInForce: o.TimeInForce,
		Type:        o.Type,
		Status:      proto.Order_STATUS_ACTIVE,
		ExpiresAt:   o.ExpiresAt,
		Reference:   o.Reference,
		PeggedOrder: o.PeggedOrder,
	}
	return order
}

type OrderAmendment struct {
	// Order identifier, this is required to find the order and will not be updated, required field
	OrderId string
	// Market identifier, this is required to find the order and will not be updated
	MarketId string
	// Amend the price for the order, if the Price value is set, otherwise price will remain unchanged - See [`Price`](#vega.Price)
	Price *num.Uint
	// Amend the size for the order by the delta specified:
	// - To reduce the size from the current value set a negative integer value
	// - To increase the size from the current value, set a positive integer value
	// - To leave the size unchanged set a value of zero
	SizeDelta int64
	// Amend the expiry time for the order, if the Timestamp value is set, otherwise expiry time will remain unchanged
	// - See [`VegaTimeResponse`](#api.VegaTimeResponse).`timestamp`
	ExpiresAt int64
	// Amend the time in force for the order, set to TIME_IN_FORCE_UNSPECIFIED to remain unchanged
	// - See [`TimeInForce`](#api.VegaTimeResponse).`timestamp`
	TimeInForce proto.Order_TimeInForce
	// Amend the pegged order offset for the order
	PeggedOffset         *num.Uint
	PeggedOffsetPositive bool
	// Amend the pegged order reference for the order
	// - See [`PeggedReference`](#vega.PeggedReference)
	PeggedReference proto.PeggedReference
}

func NewOrderAmendmentFromProto(p *commandspb.OrderAmendment) (*OrderAmendment, error) {
	o := OrderAmendment{}
	o.OrderId = p.OrderId
	o.MarketId = p.MarketId
	// Needs to update the protobuf definition TODO UINT
	if p.Price != nil {
		o.Price = num.NewUint(p.Price.Value)
	}
	o.SizeDelta = p.SizeDelta
	if p.ExpiresAt != nil {
		o.ExpiresAt = p.ExpiresAt.Value
	}
	o.TimeInForce = p.TimeInForce
	if p.PeggedOffset != nil {
		var offset uint64
		if p.PeggedOffset.Value < 0 {
			offset = uint64(-p.PeggedOffset.Value)
			o.PeggedOffsetPositive = false
		} else {
			offset = uint64(p.PeggedOffset.Value)
			o.PeggedOffsetPositive = true
		}
		o.PeggedOffset = num.NewUint(offset)
	}
	o.PeggedReference = p.PeggedReference
	return &o, nil
}

func (o OrderAmendment) IntoProto() *commandspb.OrderAmendment {
	oa := &commandspb.OrderAmendment{
		OrderId:         o.OrderId,
		MarketId:        o.MarketId,
		SizeDelta:       o.SizeDelta,
		TimeInForce:     o.TimeInForce,
		PeggedReference: o.PeggedReference,
	}
	if !o.Price.IsZero() {
		oa.Price = &proto.Price{Value: o.Price.Uint64()}
	}
	if o.PeggedOffset != nil {
		var offset int64
		if o.PeggedOffsetPositive {
			offset = int64(o.PeggedOffset.Uint64())
		} else {
			offset = -int64(o.PeggedOffset.Uint64())
		}
		oa.PeggedOffset = &wrappers.Int64Value{Value: offset}
	}

	if o.ExpiresAt != 0 {
		oa.ExpiresAt = &proto.Timestamp{Value: o.ExpiresAt}
	}
	return oa
}

func (o OrderAmendment) String() string {
	return o.IntoProto().String()
}

type WithdrawSubmission struct {
	// The amount to be withdrawn
	Amount *num.Uint
	// The asset we want to withdraw
	Asset string
	// Foreign chain specifics
	Ext *WithdrawExt
}

func NewWithdrawSubmissionFromProto(p *commandspb.WithdrawSubmission) (*WithdrawSubmission, error) {
	w := WithdrawSubmission{
		Amount: num.NewUint(p.Amount),
		Asset:  p.Asset,
		Ext:    p.Ext,
	}
	return &w, nil
}

func (w WithdrawSubmission) IntoProto() *commandspb.WithdrawSubmission {
	ws := &commandspb.WithdrawSubmission{
		// Update once the protobuf changes TODO UINT
		Amount: w.Amount.Uint64(),
		Asset:  w.Asset,
		Ext:    w.Ext,
	}
	return ws
}

func (w WithdrawSubmission) String() string {
	return w.IntoProto().String()
}
