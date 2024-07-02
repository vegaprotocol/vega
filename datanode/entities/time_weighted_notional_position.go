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
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

type TimeWeightedNotionalPosition struct {
	EpochSeq                     uint64
	AssetID                      AssetID
	PartyID                      PartyID
	GameID                       GameID
	TimeWeightedNotionalPosition num.Decimal
	VegaTime                     time.Time
}

func TimeWeightedNotionalPositionFromProto(event *eventspb.TimeWeightedNotionalPositionUpdated, vegaTime time.Time) (*TimeWeightedNotionalPosition, error) {
	twNotionalPosition, err := num.DecimalFromString(event.TimeWeightedNotionalPosition)
	if err != nil {
		return nil, err
	}
	return &TimeWeightedNotionalPosition{
		EpochSeq:                     event.EpochSeq,
		AssetID:                      AssetID(event.Asset),
		PartyID:                      PartyID(event.Party),
		GameID:                       GameID(event.GameId),
		TimeWeightedNotionalPosition: twNotionalPosition,
		VegaTime:                     vegaTime,
	}, nil
}

func (tw *TimeWeightedNotionalPosition) ToProto() *v2.TimeWeightedNotionalPosition {
	return &v2.TimeWeightedNotionalPosition{
		AssetId:                      tw.AssetID.String(),
		PartyId:                      tw.PartyID.String(),
		GameId:                       tw.GameID.String(),
		AtEpoch:                      tw.EpochSeq,
		TimeWeightedNotionalPosition: tw.TimeWeightedNotionalPosition.String(),
		LastUpdated:                  tw.VegaTime.UnixNano(),
	}
}
