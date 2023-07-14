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

//lint:file-ignore ST1003 Ignore underscores in names, this is straigh copied from the proto package to ease introducing the domain types

package types

import (
	"errors"
	"fmt"

	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/stringer"
	proto "code.vegaprotocol.io/vega/protos/vega"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
)

type (
	OracleDataSubmission = commandspb.OracleDataSubmission
	AnnounceNode         = commandspb.AnnounceNode
	NodeVote             = commandspb.NodeVote
	ChainEvent           = commandspb.ChainEvent
)

type OrderCancellation struct {
	OrderID  string
	MarketID string
}

func OrderCancellationFromProto(p *commandspb.OrderCancellation) *OrderCancellation {
	return &OrderCancellation{
		OrderID:  p.OrderId,
		MarketID: p.MarketId,
	}
}

func (o OrderCancellation) IntoProto() *commandspb.OrderCancellation {
	return &commandspb.OrderCancellation{
		OrderId:  o.OrderID,
		MarketId: o.MarketID,
	}
}

func (o OrderCancellation) String() string {
	return fmt.Sprintf(
		"marketID(%s) orderID(%s)",
		o.MarketID,
		o.OrderID,
	)
}

type OrderSubmission struct {
	// Market identifier for the order, required field
	MarketID string
	// Price for the order, the price is an integer, for example `123456` is a correctly
	// formatted price of `1.23456` assuming market configured to 5 decimal places,
	// required field for limit orders, however it is not required for market orders
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
	PeggedOrder  *PeggedOrder
	PostOnly     bool
	ReduceOnly   bool
	IcebergOrder *IcebergOrder
}

func (o OrderSubmission) IntoProto() *commandspb.OrderSubmission {
	var pegged *proto.PeggedOrder
	if o.PeggedOrder != nil {
		pegged = o.PeggedOrder.IntoProto()
	}

	var iceberg *commandspb.IcebergOpts
	if o.IcebergOrder != nil {
		iceberg = &commandspb.IcebergOpts{
			PeakSize:           o.IcebergOrder.PeakSize,
			MinimumVisibleSize: o.IcebergOrder.MinimumVisibleSize,
		}
	}

	return &commandspb.OrderSubmission{
		MarketId: o.MarketID,
		// Need to update protobuf to use string TODO UINT
		Price:       num.UintToString(o.Price),
		Size:        o.Size,
		Side:        o.Side,
		TimeInForce: o.TimeInForce,
		ExpiresAt:   o.ExpiresAt,
		Type:        o.Type,
		Reference:   o.Reference,
		PeggedOrder: pegged,
		PostOnly:    o.PostOnly,
		ReduceOnly:  o.ReduceOnly,
		IcebergOpts: iceberg,
	}
}

func NewOrderSubmissionFromProto(p *commandspb.OrderSubmission) (*OrderSubmission, error) {
	price := num.UintZero()
	if len(p.Price) > 0 {
		var overflowed bool
		price, overflowed = num.UintFromString(p.Price, 10)
		if overflowed {
			return nil, errors.New("invalid price")
		}
	}

	peggedOrder, err := NewPeggedOrderFromProto(p.PeggedOrder)
	if err != nil {
		return nil, err
	}

	var iceberg *IcebergOrder
	if p.IcebergOpts != nil {
		iceberg = &IcebergOrder{
			PeakSize:           p.IcebergOpts.PeakSize,
			MinimumVisibleSize: p.IcebergOpts.MinimumVisibleSize,
		}
	}

	return &OrderSubmission{
		MarketID:     p.MarketId,
		Price:        price,
		Size:         p.Size,
		Side:         p.Side,
		TimeInForce:  p.TimeInForce,
		ExpiresAt:    p.ExpiresAt,
		Type:         p.Type,
		Reference:    p.Reference,
		PeggedOrder:  peggedOrder,
		PostOnly:     p.PostOnly,
		ReduceOnly:   p.ReduceOnly,
		IcebergOrder: iceberg,
	}, nil
}

func (o OrderSubmission) String() string {
	return fmt.Sprintf(
		"marketID(%s) price(%s) size(%v) side(%s) timeInForce(%s) expiresAt(%v) type(%s) reference(%s) peggedOrder(%s) postOnly(%v) reduceOnly(%v)",
		o.MarketID,
		stringer.UintPointerToString(o.Price),
		o.Size,
		o.Side.String(),
		o.TimeInForce.String(),
		o.ExpiresAt,
		o.Type.String(),
		o.Reference,
		stringer.ReflectPointerToString(o.PeggedOrder),
		o.PostOnly,
		o.ReduceOnly,
	)
}

func (o OrderSubmission) IntoOrder(party string) *Order {
	var iceberg *IcebergOrder
	if o.IcebergOrder != nil {
		iceberg = &IcebergOrder{
			PeakSize:           o.IcebergOrder.PeakSize,
			MinimumVisibleSize: o.IcebergOrder.MinimumVisibleSize,
		}
	}

	return &Order{
		MarketID:     o.MarketID,
		Party:        party,
		Side:         o.Side,
		Price:        o.Price,
		Size:         o.Size,
		Remaining:    o.Size,
		TimeInForce:  o.TimeInForce,
		Type:         o.Type,
		Status:       proto.Order_STATUS_ACTIVE,
		ExpiresAt:    o.ExpiresAt,
		Reference:    o.Reference,
		PeggedOrder:  o.PeggedOrder,
		PostOnly:     o.PostOnly,
		ReduceOnly:   o.ReduceOnly,
		IcebergOrder: iceberg,
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

func NewWithdrawSubmissionFromProto(p *commandspb.WithdrawSubmission) (*WithdrawSubmission, error) {
	amount := num.UintZero()
	if len(p.Amount) > 0 {
		var overflowed bool
		amount, overflowed = num.UintFromString(p.Amount, 10)
		if overflowed {
			return nil, errors.New("invalid amount")
		}
	}

	return &WithdrawSubmission{
		Amount: amount,
		Asset:  p.Asset,
		Ext:    WithdrawExtFromProto(p.Ext),
	}, nil
}

func (w WithdrawSubmission) IntoProto() *commandspb.WithdrawSubmission {
	return &commandspb.WithdrawSubmission{
		// Update once the protobuf changes TODO UINT
		Amount: num.UintToString(w.Amount),
		Asset:  w.Asset,
		Ext:    w.Ext.IntoProto(),
	}
}

func (w WithdrawSubmission) String() string {
	return fmt.Sprintf(
		"asset(%s) amount(%s) ext(%s)",
		w.Asset,
		stringer.UintPointerToString(w.Amount),
		stringer.ReflectPointerToString(w.Ext),
	)
}
