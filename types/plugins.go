//lint:file-ignore ST1003 Ignore underscores in names, this is straight copied from the proto package to ease introducing the domain types

package types

import (
	proto "code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/data-node/types/num"
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
		RealisedPnl:       p.RealisedPnl.BigInt().Int64(),
		UnrealisedPnl:     p.UnrealisedPnl.BigInt().Int64(),
		AverageEntryPrice: num.UintToUint64(p.AverageEntryPrice),
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
