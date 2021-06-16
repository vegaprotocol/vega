//lint:file-ignore ST1003 Ignore underscores in names, this is straigh copied from the proto package to ease introducing the domain types

package types

import (
	"fmt"

	"code.vegaprotocol.io/vega/proto"
	commandspb "code.vegaprotocol.io/vega/proto/commands/v1"
	"code.vegaprotocol.io/vega/types/num"
)

//type OrderSubmission = commandspb.OrderSubmission
type OrderCancellation = commandspb.OrderCancellation
type OrderAmendment = commandspb.OrderAmendment
type WithdrawSubmission = commandspb.WithdrawSubmission
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

func (o OrderSubmission) ToProto() *commandspb.OrderSubmission {
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
		//		PeggedOrder: o.PeggedOrder.ToProto(),
	}
	return p
}

func (o *OrderSubmission) FromProto(p *commandspb.OrderSubmission) {
	o.MarketId = p.MarketId
	// Need to update protobuf to use string TODO UINT
	o.Price = num.NewUint(p.Price)
	o.Size = p.Size
	o.Side = p.Side
	o.TimeInForce = p.TimeInForce
	o.ExpiresAt = p.ExpiresAt
	o.Type = p.Type
	o.Reference = p.Reference
	//	o.PeggedOrder.FromProto(p.PeggedOrder)
}

func (o OrderSubmission) String() string {
	return fmt.Sprint(o.MarketId, o.Price, o.Size, o.Side, o.TimeInForce, o.ExpiresAt, o.Type, o.Reference, o.PeggedOrder)
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
