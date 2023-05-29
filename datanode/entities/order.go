// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.DATANODE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package entities

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/ptr"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	"code.vegaprotocol.io/vega/protos/vega"
	"github.com/shopspring/decimal"
)

type _Order struct{}

type OrderID = ID[_Order]

type Order struct {
	ID              OrderID
	MarketID        MarketID
	PartyID         PartyID
	Side            Side
	Price           decimal.Decimal
	Size            int64
	Remaining       int64
	TimeInForce     OrderTimeInForce
	Type            OrderType
	Status          OrderStatus
	Reference       string
	Reason          OrderError
	Version         int32
	PeggedOffset    decimal.Decimal
	BatchID         int32
	PeggedReference PeggedReference
	LpID            []byte
	CreatedAt       time.Time
	UpdatedAt       time.Time
	ExpiresAt       time.Time
	TxHash          TxHash
	VegaTime        time.Time
	SeqNum          uint64
	PostOnly        bool
	ReduceOnly      bool

	// Iceberg fields
	ReservedRemaining *int64
	InitialPeakSize   *int64
	MinimumPeakSize   *int64
}

func (o Order) ToProto() *vega.Order {
	var peggedOrder *vega.PeggedOrder
	if o.PeggedReference != types.PeggedReferenceUnspecified {
		peggedOrder = &vega.PeggedOrder{
			Reference: o.PeggedReference,
			Offset:    o.PeggedOffset.String(),
		}
	}

	var reason *OrderError
	if o.Reason != OrderErrorUnspecified {
		reason = ptr.From(o.Reason)
	}

	var icebergOrder *vega.IcebergOrder
	if o.InitialPeakSize != nil {
		icebergOrder = &vega.IcebergOrder{
			ReservedRemaining: uint64(*o.ReservedRemaining),
			InitialPeakSize:   uint64(*o.InitialPeakSize),
			MinimumPeakSize:   uint64(*o.MinimumPeakSize),
		}
	}

	vo := vega.Order{
		Id:                   o.ID.String(),
		MarketId:             o.MarketID.String(),
		PartyId:              o.PartyID.String(),
		Side:                 o.Side,
		Price:                o.Price.String(),
		Size:                 uint64(o.Size),
		Remaining:            uint64(o.Remaining),
		TimeInForce:          o.TimeInForce,
		Type:                 o.Type,
		CreatedAt:            o.CreatedAt.UnixNano(),
		Status:               o.Status,
		ExpiresAt:            o.ExpiresAt.UnixNano(),
		Reference:            o.Reference,
		Reason:               reason,
		UpdatedAt:            o.UpdatedAt.UnixNano(),
		Version:              uint64(o.Version),
		BatchId:              uint64(o.BatchID),
		PeggedOrder:          peggedOrder,
		LiquidityProvisionId: hex.EncodeToString(o.LpID),
		PostOnly:             o.PostOnly,
		ReduceOnly:           o.ReduceOnly,
		IcebergOrder:         icebergOrder,
	}
	return &vo
}

func (o Order) ToProtoEdge(_ ...any) (*v2.OrderEdge, error) {
	return &v2.OrderEdge{
		Node:   o.ToProto(),
		Cursor: o.Cursor().Encode(),
	}, nil
}

func OrderFromProto(po *vega.Order, seqNum uint64, txHash TxHash) (Order, error) {
	price, err := decimal.NewFromString(po.Price)
	if err != nil {
		return Order{}, fmt.Errorf("price is not a valid integer: %v", po.Price)
	}

	if po.Size > math.MaxInt64 {
		return Order{}, fmt.Errorf("size is larger than a 64-bit integer: %v", po.Size)
	}
	size := int64(po.Size)

	if po.Remaining > math.MaxInt64 {
		return Order{}, fmt.Errorf("remaining is larger than a 64-bit integer: %v", po.Remaining)
	}
	remaining := int64(po.Remaining)

	if po.Version >= math.MaxInt32 {
		return Order{}, fmt.Errorf("version is larger than a 32-bit integer: %v", po.Version)
	}
	version := int32(po.Version)

	if po.BatchId >= math.MaxInt32 {
		return Order{}, fmt.Errorf("batch ID is larger than a 32-bit integer: %v", po.Version)
	}
	batchID := int32(po.BatchId)

	lpID, err := hex.DecodeString(po.LiquidityProvisionId)
	if err != nil {
		return Order{}, fmt.Errorf("liquidity Provision ID is not a valid hex string: %v", po.LiquidityProvisionId)
	}

	peggedOffset := decimal.Zero
	var peggedReference types.PeggedReference
	if po.PeggedOrder != nil {
		peggedOffset, err = decimal.NewFromString(po.PeggedOrder.Offset)
		if err != nil {
			return Order{}, fmt.Errorf("pegged Offset not a valid decimal: %v", po.Price)
		}
		peggedReference = po.PeggedOrder.Reference
	}

	reason := OrderErrorUnspecified
	if po.Reason != nil {
		reason = *po.Reason
	}

	var initialPeakSize, minimumPeakSize, reservedRemaining *int64
	if po.IcebergOrder != nil {
		if po.IcebergOrder.ReservedRemaining > math.MaxInt64 {
			return Order{}, fmt.Errorf("iceberg reserved remaining is larger than a 64-bit integer: %v", po.Remaining)
		}
		reservedRemaining = ptr.From(int64(po.IcebergOrder.ReservedRemaining))

		if po.IcebergOrder.InitialPeakSize > math.MaxInt64 {
			return Order{}, fmt.Errorf("iceberg initial peak size is larger than a 64-bit integer: %v", po.Remaining)
		}
		initialPeakSize = ptr.From(int64(po.IcebergOrder.InitialPeakSize))

		if po.IcebergOrder.MinimumPeakSize > math.MaxInt64 {
			return Order{}, fmt.Errorf("iceberg minimum peak size is larger than a 64-bit integer: %v", po.Remaining)
		}
		minimumPeakSize = ptr.From(int64(po.IcebergOrder.MinimumPeakSize))
	}

	o := Order{
		ID:                OrderID(po.Id),
		MarketID:          MarketID(po.MarketId),
		PartyID:           PartyID(po.PartyId),
		Side:              po.Side,
		Price:             price,
		Size:              size,
		Remaining:         remaining,
		TimeInForce:       po.TimeInForce,
		Type:              po.Type,
		Status:            po.Status,
		Reference:         po.Reference,
		Reason:            reason,
		Version:           version,
		PeggedOffset:      peggedOffset,
		BatchID:           batchID,
		PeggedReference:   peggedReference,
		LpID:              lpID,
		CreatedAt:         NanosToPostgresTimestamp(po.CreatedAt),
		UpdatedAt:         NanosToPostgresTimestamp(po.UpdatedAt),
		ExpiresAt:         NanosToPostgresTimestamp(po.ExpiresAt),
		SeqNum:            seqNum,
		TxHash:            txHash,
		PostOnly:          po.PostOnly,
		ReduceOnly:        po.ReduceOnly,
		ReservedRemaining: reservedRemaining,
		InitialPeakSize:   initialPeakSize,
		MinimumPeakSize:   minimumPeakSize,
	}

	return o, nil
}

type OrderKey struct {
	ID       OrderID
	Version  int32
	VegaTime time.Time
}

func (o Order) Key() OrderKey {
	return OrderKey{o.ID, o.Version, o.VegaTime}
}

func (o Order) ToRow() []interface{} {
	return []interface{}{
		o.ID, o.MarketID, o.PartyID, o.Side, o.Price,
		o.Size, o.Remaining, o.TimeInForce, o.Type, o.Status,
		o.Reference, o.Reason, o.Version, o.PeggedOffset, o.BatchID,
		o.PeggedReference, o.LpID, o.CreatedAt, o.UpdatedAt, o.ExpiresAt,
		o.TxHash, o.VegaTime, o.SeqNum, o.PostOnly, o.ReduceOnly, o.ReservedRemaining,
		o.InitialPeakSize, o.MinimumPeakSize,
	}
}

var OrderColumns = []string{
	"id", "market_id", "party_id", "side", "price",
	"size", "remaining", "time_in_force", "type", "status",
	"reference", "reason", "version", "pegged_offset", "batch_id",
	"pegged_reference", "lp_id", "created_at", "updated_at", "expires_at",
	"tx_hash", "vega_time", "seq_num", "post_only", "reduce_only", "reserved_remaining",
	"initial_peak_size", "minimum_peak_size",
}

type OrderCursor struct {
	CreatedAt time.Time `json:"createdAt"`
	ID        OrderID   `json:"id"`
	VegaTime  time.Time `json:"vegaTime"`
}

func (oc *OrderCursor) Parse(cursorString string) error {
	if cursorString == "" {
		return nil
	}
	return json.Unmarshal([]byte(cursorString), oc)
}

func (oc OrderCursor) String() string {
	bs, err := json.Marshal(oc)
	if err != nil {
		// This should never happen.
		panic(fmt.Errorf("could not marshal order cursor: %w", err))
	}
	return string(bs)
}

func (o Order) Cursor() *Cursor {
	cursor := OrderCursor{
		CreatedAt: o.CreatedAt,
		ID:        o.ID,
		VegaTime:  o.VegaTime,
	}

	return NewCursor(cursor.String())
}
