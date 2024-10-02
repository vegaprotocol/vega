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

package types

import (
	"code.vegaprotocol.io/vega/libs/num"
	proto "code.vegaprotocol.io/vega/protos/vega"
)

type Position struct {
	// Market identifier
	MarketID string
	// Party identifier
	PartyID string
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
	UpdatedAt                 int64
	TakerFeesPaid             *num.Uint
	TakerFeesPaidSince        *num.Uint
	MakerFeesReceived         *num.Uint
	MakerFeesReceivedSince    *num.Uint
	FeesPaid                  *num.Uint
	FeesPaidSince             *num.Uint
	FundingPaymentAmount      *num.Int
	FundingPaymentAmountSince *num.Int
}

func NewPosition(marketID, partyID string) Position {
	return Position{
		MarketID:                  marketID,
		PartyID:                   partyID,
		AverageEntryPrice:         num.UintZero(),
		TakerFeesPaid:             num.UintZero(),
		MakerFeesReceived:         num.UintZero(),
		FeesPaid:                  num.UintZero(),
		TakerFeesPaidSince:        num.UintZero(),
		MakerFeesReceivedSince:    num.UintZero(),
		FeesPaidSince:             num.UintZero(),
		FundingPaymentAmount:      num.IntZero(),
		FundingPaymentAmountSince: num.IntZero(),
	}
}

func (p *Position) IntoProto() *proto.Position {
	if p.FeesPaid == nil {
		p.FeesPaid = num.UintZero()
	}
	if p.FeesPaidSince == nil {
		p.FeesPaidSince = num.UintZero()
	}
	if p.TakerFeesPaid == nil {
		p.TakerFeesPaid = num.UintZero()
	}
	if p.TakerFeesPaidSince == nil {
		p.TakerFeesPaidSince = num.UintZero()
	}
	if p.MakerFeesReceived == nil {
		p.MakerFeesReceived = num.UintZero()
	}
	if p.MakerFeesReceivedSince == nil {
		p.MakerFeesReceivedSince = num.UintZero()
	}
	if p.FundingPaymentAmount == nil {
		p.FundingPaymentAmount = num.IntZero()
	}
	if p.FundingPaymentAmountSince == nil {
		p.FundingPaymentAmountSince = num.IntZero()
	}
	return &proto.Position{
		MarketId:                  p.MarketID,
		PartyId:                   p.PartyID,
		OpenVolume:                p.OpenVolume,
		RealisedPnl:               p.RealisedPnl.BigInt().String(),
		UnrealisedPnl:             p.UnrealisedPnl.BigInt().String(),
		AverageEntryPrice:         num.UintToString(p.AverageEntryPrice),
		UpdatedAt:                 p.UpdatedAt,
		TakerFeesPaid:             p.TakerFeesPaid.String(),
		MakerFeesReceived:         p.MakerFeesReceived.String(),
		FeesPaid:                  p.FeesPaid.String(),
		TakerFeesPaidSince:        p.TakerFeesPaidSince.String(),
		MakerFeesReceivedSince:    p.MakerFeesReceivedSince.String(),
		FundingPaymentAmount:      p.FundingPaymentAmount.String(),
		FundingPaymentAmountSince: p.FundingPaymentAmountSince.String(),
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

func (p *Position) ResetSince() {
	p.TakerFeesPaidSince = num.UintZero()
	p.MakerFeesReceivedSince = num.UintZero()
	p.FeesPaidSince = num.UintZero()
	p.FundingPaymentAmountSince = num.IntZero()
}
