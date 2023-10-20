package entities

import (
	"time"

	"code.vegaprotocol.io/vega/libs/num"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
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

func PartyVestingBalanceFromProto(
	partyID string,
	atEpoch uint64,
	pvb *eventspb.PartyVestingBalance,
	t time.Time,
) (*PartyVestingBalance, error) {
	balance, err := num.DecimalFromString(pvb.Balance)
	if err != nil {
		return nil, err
	}

	return &PartyVestingBalance{
		PartyID:  PartyID(partyID),
		AssetID:  AssetID(pvb.Asset),
		AtEpoch:  atEpoch,
		Balance:  balance,
		VegaTime: t,
	}, nil
}

func PartyLockedBalanceFromProto(
	partyID string,
	atEpoch uint64,
	pvb *eventspb.PartyLockedBalance,
	t time.Time,
) (*PartyLockedBalance, error) {
	balance, err := num.DecimalFromString(pvb.Balance)
	if err != nil {
		return nil, err
	}

	return &PartyLockedBalance{
		PartyID:    PartyID(partyID),
		AssetID:    AssetID(pvb.Asset),
		AtEpoch:    atEpoch,
		UntilEpoch: pvb.UntilEpoch,
		Balance:    balance,
		VegaTime:   t,
	}, nil
}
