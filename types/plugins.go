//lint:file-ignore ST1003 Ignore underscores in names, this is straight copied from the proto package to ease introducing the domain types

package types

import (
	"code.vegaprotocol.io/vega/types/num"
)

//type Position = proto.Position

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
