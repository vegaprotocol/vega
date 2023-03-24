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
	proto "code.vegaprotocol.io/vega/protos/vega"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
)

type PriceLevel struct {
	Price          *num.Uint
	NumberOfOrders uint64
	Volume         uint64
}

func (p PriceLevel) IntoProto() *proto.PriceLevel {
	return &proto.PriceLevel{
		Price:          num.UintToString(p.Price),
		NumberOfOrders: p.NumberOfOrders,
		Volume:         p.Volume,
	}
}

type PriceLevels []*PriceLevel

func (p PriceLevels) IntoProto() []*proto.PriceLevel {
	out := make([]*proto.PriceLevel, 0, len(p))
	for _, v := range p {
		out = append(out, v.IntoProto())
	}
	return out
}

type OrderAmendment struct {
	Price           *num.Uint
	ExpiresAt       *int64 // timestamp
	PeggedOffset    *num.Uint
	OrderID         string
	MarketID        string
	SizeDelta       int64
	TimeInForce     OrderTimeInForce
	PeggedReference PeggedReference
}

func NewOrderAmendmentFromProto(p *commandspb.OrderAmendment) (*OrderAmendment, error) {
	var (
		price, peggedOffset *num.Uint
		exp                 *int64
	)
	if p.Price != nil && len(*p.Price) > 0 {
		var overflowed bool
		price, overflowed = num.UintFromString(*p.Price, 10)
		if overflowed {
			return nil, errors.New("invalid amount")
		}
	}

	if p.ExpiresAt != nil {
		exp = toPtr(*p.ExpiresAt)
	}
	if p.PeggedOffset != "" {
		var overflowed bool
		peggedOffset, overflowed = num.UintFromString(p.PeggedOffset, 10)
		if overflowed {
			return nil, errors.New("invalid offset")
		}
	}
	return &OrderAmendment{
		OrderID:         p.OrderId,
		MarketID:        p.MarketId,
		Price:           price,
		SizeDelta:       p.SizeDelta,
		ExpiresAt:       exp,
		TimeInForce:     p.TimeInForce,
		PeggedOffset:    peggedOffset,
		PeggedReference: p.PeggedReference,
	}, nil
}

func (o OrderAmendment) IntoProto() *commandspb.OrderAmendment {
	r := &commandspb.OrderAmendment{
		OrderId:         o.OrderID,
		MarketId:        o.MarketID,
		SizeDelta:       o.SizeDelta,
		TimeInForce:     o.TimeInForce,
		PeggedReference: o.PeggedReference,
	}
	if o.Price != nil {
		r.Price = toPtr(num.UintToString(o.Price))
	}
	if o.ExpiresAt != nil {
		r.ExpiresAt = toPtr(*o.ExpiresAt)
	}
	if o.PeggedOffset != nil {
		r.PeggedOffset = o.PeggedOffset.String()
	}
	return r
}

// Validate santiy-checks the order amendment as-is, the market will further validate the amendment
// based on the order it's actually trying to amend.
func (o OrderAmendment) Validate() error {
	// check TIME_IN_FORCE and expiry
	if o.TimeInForce == OrderTimeInForceGTT && o.ExpiresAt == nil {
		return OrderErrorCannotAmendToGTTWithoutExpiryAt
	}

	if o.TimeInForce == OrderTimeInForceGTC && o.ExpiresAt != nil {
		// this is cool, but we need to ensure and expiry is not set
		return OrderErrorCannotHaveGTCAndExpiryAt
	}

	if o.TimeInForce == OrderTimeInForceFOK || o.TimeInForce == OrderTimeInForceIOC {
		// IOC and FOK are not acceptable for amend order
		return OrderErrorCannotAmendToFOKOrIOC
	}

	return nil
}

func (o OrderAmendment) String() string {
	return fmt.Sprintf(
		"orderID(%s) marketID(%s) sizeDelta(%v) timeInForce(%s) peggedReference(%s) price(%s) expiresAt(%v) peggedOffset(%s)",
		o.OrderID,
		o.MarketID,
		o.SizeDelta,
		o.TimeInForce.String(),
		o.PeggedReference.String(),
		uintPointerToString(o.Price),
		int64PointerToString(o.ExpiresAt),
		uintPointerToString(o.PeggedOffset),
	)
}

func (o OrderAmendment) GetOrderID() string {
	return o.OrderID
}

func (o OrderAmendment) GetMarketID() string {
	return o.MarketID
}
