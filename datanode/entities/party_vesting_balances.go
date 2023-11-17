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
