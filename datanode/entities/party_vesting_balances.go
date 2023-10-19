package entities

import (
	"time"

	"code.vegaprotocol.io/vega/libs/num"
)

type (
	PartyLockedBalance struct {
		PartyID    PartyID
		AssetID    AssetID
		AtEpoch    uint64
		UntilEpoch uint64
		Balance    num.Decimal
		VegaTime   time.Time
	}

	PartyVestingBalance struct {
		PartyID  PartyID
		AssetID  AssetID
		AtEpoch  uint64
		Balance  num.Decimal
		VegaTime time.Time
	}
)
