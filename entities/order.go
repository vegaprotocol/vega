package entities

import (
	"encoding/hex"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/types"
)

type Order struct {
	ID              []byte
	MarketID        []byte
	PartyID         []byte
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
}

func MakeOrderID(stringID string) ([]byte, error) {
	id, err := hex.DecodeString(stringID)
	if err != nil {
		return nil, fmt.Errorf("order id is not valid hex string: %v", stringID)
	}
	return id, nil
}

func (o *Order) HexID() string {
	return strings.ToUpper(hex.EncodeToString(o.ID))
}

func (o *Order) ToProto() *vega.Order {
	var peggedOrder *vega.PeggedOrder
	if o.PeggedReference != types.PeggedReferenceUnspecified {
		peggedOrder = &vega.PeggedOrder{
			Reference: o.PeggedReference,
			Offset:    fmt.Sprint(o.PeggedOffset)}
	}

	vo := vega.Order{
		Id:                   o.HexID(),
		MarketId:             Market{ID: o.MarketID}.HexID(),
		PartyId:              Party{ID: o.PartyID}.HexID(),
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

func OrderFromProto(po *vega.Order) (Order, error) {
	id, err := hex.DecodeString(po.Id)
	if err != nil {
		return Order{}, fmt.Errorf("Order ID is not a valid hex string: %v", po.Id)
	}

	marketId, err := hex.DecodeString(po.MarketId)
	if err != nil {
		return Order{}, fmt.Errorf("Market ID is not a valid hex string: %v", po.MarketId)
	}

	partyId, err := hex.DecodeString(po.PartyId)
	if err != nil {
		return Order{}, fmt.Errorf("Party ID is not a valid hex string: %v", po.PartyId)
	}

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
		ID:              id,
		MarketID:        marketId,
		PartyID:         partyId,
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
		CreatedAt:       time.Unix(0, po.CreatedAt),
		UpdatedAt:       time.Unix(0, po.ExpiresAt),
		ExpiresAt:       time.Unix(0, po.ExpiresAt),
	}

	return o, nil
}
