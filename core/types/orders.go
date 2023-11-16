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
	OrderID         string
	MarketID        string
	Price           *num.Uint
	SizeDelta       int64
	Size            *uint64
	ExpiresAt       *int64 // timestamp
	TimeInForce     OrderTimeInForce
	PeggedOffset    *num.Uint
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
		Size:            p.Size,
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
		Size:            o.Size,
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
		"orderID(%s) marketID(%s) sizeDelta(%v) size(%v) timeInForce(%s) peggedReference(%s) price(%s) expiresAt(%v) peggedOffset(%s)",
		o.OrderID,
		o.MarketID,
		o.SizeDelta,
		stringer.PtrToString(o.Size),
		o.TimeInForce.String(),
		o.PeggedReference.String(),
		stringer.PtrToString(o.Price),
		stringer.PtrToString(o.ExpiresAt),
		stringer.PtrToString(o.PeggedOffset),
	)
}

func (o OrderAmendment) GetOrderID() string {
	return o.OrderID
}

func (o OrderAmendment) GetMarketID() string {
	return o.MarketID
}
