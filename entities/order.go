// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
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
	"strconv"
	"time"

	v2 "code.vegaprotocol.io/protos/data-node/api/v2"
	"code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/types"
)

type OrderID struct{ ID }

func NewOrderID(id string) OrderID {
	return OrderID{ID: ID(id)}
}

type Order struct {
	ID              OrderID
	MarketID        MarketID
	PartyID         PartyID
	Side            Side
	Price           int64
	Size            int64
	Remaining       int64
	TimeInForce     OrderTimeInForce
	Type            OrderType
	Status          OrderStatus
	Reference       string
	Reason          OrderError
	Version         int32
	PeggedOffset    int32
	BatchID         int32
	PeggedReference PeggedReference
	LpID            []byte
	CreatedAt       time.Time
	UpdatedAt       time.Time
	ExpiresAt       time.Time
	VegaTime        time.Time
	SeqNum          uint64
}

func (o Order) ToProto() *vega.Order {
	var peggedOrder *vega.PeggedOrder
	if o.PeggedReference != types.PeggedReferenceUnspecified {
		peggedOrder = &vega.PeggedOrder{
			Reference: o.PeggedReference,
			Offset:    fmt.Sprint(o.PeggedOffset),
		}
	}

	vo := vega.Order{
		Id:                   o.ID.String(),
		MarketId:             o.MarketID.String(),
		PartyId:              o.PartyID.String(),
		Side:                 o.Side,
		Price:                strconv.FormatInt(o.Price, 10),
		Size:                 uint64(o.Size),
		Remaining:            uint64(o.Remaining),
		TimeInForce:          o.TimeInForce,
		Type:                 o.Type,
		CreatedAt:            o.CreatedAt.UnixNano(),
		Status:               o.Status,
		ExpiresAt:            o.ExpiresAt.UnixNano(),
		Reference:            o.Reference,
		Reason:               o.Reason,
		UpdatedAt:            o.UpdatedAt.UnixNano(),
		Version:              uint64(o.Version),
		BatchId:              uint64(o.BatchID),
		PeggedOrder:          peggedOrder,
		LiquidityProvisionId: hex.EncodeToString(o.LpID),
	}
	return &vo
}

func (o Order) ToProtoEdge(_ ...any) (*v2.OrderEdge, error) {
	return &v2.OrderEdge{
		Node:   o.ToProto(),
		Cursor: o.Cursor().Encode(),
	}, nil
}

func OrderFromProto(po *vega.Order, seqNum uint64) (Order, error) {
	price, err := strconv.ParseInt(po.Price, 10, 64)
	if err != nil {
		return Order{}, fmt.Errorf("Price is not a valid integer: %v", po.Price)
	}

	if po.Size > math.MaxInt64 {
		return Order{}, fmt.Errorf("Size is to large for int64: %v", po.Size)
	}
	size := int64(po.Size)

	if po.Size > math.MaxInt64 {
		return Order{}, fmt.Errorf("Remaining is to large for int64: %v", po.Remaining)
	}
	remaining := int64(po.Remaining)

	if po.Version >= math.MaxInt32 {
		return Order{}, fmt.Errorf("Version is too large for int32: %v", po.Version)
	}
	version := int32(po.Version)

	if po.BatchId >= math.MaxInt32 {
		return Order{}, fmt.Errorf("Batch ID is too large for int32: %v", po.Version)
	}
	batchID := int32(po.BatchId)

	lpID, err := hex.DecodeString(po.LiquidityProvisionId)
	if err != nil {
		return Order{}, fmt.Errorf("Liquidity Provision ID is not a valid hex string: %v", po.LiquidityProvisionId)
	}

	var peggedOffset int32
	var peggedReference types.PeggedReference
	if po.PeggedOrder != nil {
		peggedOffset64, err := strconv.ParseInt(po.PeggedOrder.Offset, 10, 32)
		if err != nil {
			return Order{}, fmt.Errorf("Pegged Offset not valid int32: %v", po.Price)
		}
		peggedOffset = int32(peggedOffset64)
		peggedReference = po.PeggedOrder.Reference
	}

	o := Order{
		ID:              NewOrderID(po.Id),
		MarketID:        NewMarketID(po.MarketId),
		PartyID:         NewPartyID(po.PartyId),
		Side:            po.Side,
		Price:           price,
		Size:            size,
		Remaining:       remaining,
		TimeInForce:     po.TimeInForce,
		Type:            po.Type,
		Status:          po.Status,
		Reference:       po.Reference,
		Reason:          po.Reason,
		Version:         version,
		PeggedOffset:    peggedOffset,
		BatchID:         batchID,
		PeggedReference: peggedReference,
		LpID:            lpID,
		CreatedAt:       NanosToPostgresTimestamp(po.CreatedAt),
		UpdatedAt:       NanosToPostgresTimestamp(po.UpdatedAt),
		ExpiresAt:       NanosToPostgresTimestamp(po.ExpiresAt),
		SeqNum:          seqNum,
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
		o.VegaTime, o.SeqNum,
	}
}

var OrderColumns = []string{
	"id", "market_id", "party_id", "side", "price",
	"size", "remaining", "time_in_force", "type", "status",
	"reference", "reason", "version", "pegged_offset", "batch_id",
	"pegged_reference", "lp_id", "created_at", "updated_at", "expires_at",
	"vega_time", "seq_num",
}

type OrderCursor struct {
	VegaTime time.Time `json:"vegaTime"`
	SeqNum   uint64    `json:"seqNum"`
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
		VegaTime: o.VegaTime,
		SeqNum:   o.SeqNum,
	}

	return NewCursor(cursor.String())
}
