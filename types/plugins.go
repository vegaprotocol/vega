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

package types

import (
	proto "code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/types/num"
)

type Position struct {
	// Market identifier
	MarketId string
	// Party identifier
	PartyId string
	// Open volume for the position, value is signed +ve for long and -ve for short
	OpenVolume int64
	// Realised profit and loss for the position, value is signed +ve for long and -ve for short
	RealisedPnl num.Decimal
	// Unrealised profit and loss for the position, value is signed +ve for long and -ve for short
	UnrealisedPnl num.Decimal
	// Average entry price for the position, the price is an integer, for example `123456` is a correctly
	// formatted price of `1.23456` assuming market configured to 5 decimal places
	AverageEntryPrice *num.Uint
	// Timestamp for the latest time the position was updated
	UpdatedAt int64
}

func (p *Position) IntoProto() *proto.Position {
	return &proto.Position{
		MarketId:          p.MarketId,
		PartyId:           p.PartyId,
		OpenVolume:        p.OpenVolume,
		RealisedPnl:       p.RealisedPnl.BigInt().String(),
		UnrealisedPnl:     p.UnrealisedPnl.BigInt().String(),
		AverageEntryPrice: num.UintToString(p.AverageEntryPrice),
		UpdatedAt:         p.UpdatedAt,
	}
}

type Positions []*Position

func (p Positions) IntoProto() []*proto.Position {
	out := make([]*proto.Position, 0, len(p))
	for _, v := range p {
		out = append(out, v.IntoProto())
	}
	return out
}
