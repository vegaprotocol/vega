//lint:file-ignore ST1003 Ignore underscores in names, this is straigh copied from the proto package to ease introducing the domain types

package types

import (
	proto "code.vegaprotocol.io/protos/vega"
	commandspb "code.vegaprotocol.io/protos/vega/commands/v1"
	"code.vegaprotocol.io/data-node/types/num"
)

type OracleDataSubmission = commandspb.OracleDataSubmission
type NodeRegistration = commandspb.NodeRegistration
type NodeVote = commandspb.NodeVote
type Transaction = proto.Transaction

type ChainEvent = commandspb.ChainEvent
type SignedBundle = proto.SignedBundle
type Signature = proto.Signature
type Transaction_PubKey = proto.Transaction_PubKey

type OrderCancellation struct {
	OrderId  string
	MarketId string
}

func OrderCancellationFromProto(p *commandspb.OrderCancellation) *OrderCancellation {
	return &OrderCancellation{
		OrderId:  p.OrderId,
		MarketId: p.MarketId,
	}
}

func (o OrderCancellation) IntoProto() *commandspb.OrderCancellation {
	return &commandspb.OrderCancellation{
		OrderId:  o.OrderId,
		MarketId: o.MarketId,
	}
}

func (o OrderCancellation) String() string {
	return o.IntoProto().String()
}

type OrderSubmission struct {
	// Market identifier for the order, required field
	MarketId string
	// Price for the order, the price is an integer, for example `123456` is a correctly
	// formatted price of `1.23456` assuming market configured to 5 decimal places,
	// , required field for limit orders, however it is not required for market orders
	Price *num.Uint
	// Size for the order, for example, in a futures market the size equals the number of contracts, cannot be negative
	Size uint64
	// Side for the order, e.g. SIDE_BUY or SIDE_SELL, required field
	Side proto.Side
	// Time in force indicates how long an order will remain active before it is executed or expires, required field
	TimeInForce proto.Order_TimeInForce
	// Timestamp for when the order will expire, in nanoseconds since the epoch,
	ExpiresAt int64
	// Type for the order, required field
	Type proto.Order_Type
	// Reference given for the order, this is typically used to retrieve an order submitted through consensus, currently
	// set internally by the node to return a unique reference identifier for the order submission
	Reference string
	// Used to specify the details for a pegged order
	PeggedOrder *PeggedOrder
}

func (o OrderSubmission) IntoProto() *commandspb.OrderSubmission {
	var pegged *proto.PeggedOrder
	if o.PeggedOrder != nil {
		pegged = o.PeggedOrder.IntoProto()
	}
	return &commandspb.OrderSubmission{
		MarketId: o.MarketId,
		// Need to update protobuf to use string TODO UINT
		Price:       num.UintToUint64(o.Price),
		Size:        o.Size,
		Side:        o.Side,
		TimeInForce: o.TimeInForce,
		ExpiresAt:   o.ExpiresAt,
		Type:        o.Type,
		Reference:   o.Reference,
		PeggedOrder: pegged,
	}
}

func NewOrderSubmissionFromProto(p *commandspb.OrderSubmission) *OrderSubmission {
	return &OrderSubmission{
		MarketId: p.MarketId,
		// Need to update protobuf to use string TODO UINT
		Price:       num.NewUint(p.Price),
		Size:        p.Size,
		Side:        p.Side,
		TimeInForce: p.TimeInForce,
		ExpiresAt:   p.ExpiresAt,
		Type:        p.Type,
		Reference:   p.Reference,
		PeggedOrder: NewPeggedOrderFromProto(p.PeggedOrder),
	}
}

func (o OrderSubmission) String() string {
	return o.IntoProto().String()
}

func (o OrderSubmission) IntoOrder(party string) *Order {
	return &Order{
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
}

type WithdrawSubmission struct {
	// The amount to be withdrawn
	Amount *num.Uint
	// The asset we want to withdraw
	Asset string
	// Foreign chain specifics
	Ext *WithdrawExt
}

func NewWithdrawSubmissionFromProto(p *commandspb.WithdrawSubmission) *WithdrawSubmission {
	return &WithdrawSubmission{
		Amount: num.NewUint(p.Amount),
		Asset:  p.Asset,
		Ext:    p.Ext,
	}
}

func (w WithdrawSubmission) IntoProto() *commandspb.WithdrawSubmission {
	return &commandspb.WithdrawSubmission{
		// Update once the protobuf changes TODO UINT
		Amount: num.UintToUint64(w.Amount),
		Asset:  w.Asset,
		Ext:    w.Ext,
	}
}

func (w WithdrawSubmission) String() string {
	return w.IntoProto().String()
}
