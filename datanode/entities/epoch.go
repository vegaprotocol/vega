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

	"code.vegaprotocol.io/vega/protos/vega"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

type Epoch struct {
	ID         int64
	StartTime  time.Time
	ExpireTime time.Time
	EndTime    *time.Time
	TxHash     TxHash
	VegaTime   time.Time
	FirstBlock *int64
	LastBlock  *int64
}

func (e *Epoch) ToProto() *vega.Epoch {
	protoEpoch := vega.Epoch{
		Seq: uint64(e.ID),
		Timestamps: &vega.EpochTimestamps{
			StartTime:  e.StartTime.UnixNano(),
			ExpiryTime: e.ExpireTime.UnixNano(),
		},
	}
	if e.EndTime != nil {
		protoEpoch.Timestamps.EndTime = e.EndTime.UnixNano()
	}
	if e.FirstBlock != nil {
		protoEpoch.Timestamps.FirstBlock = uint64(*e.FirstBlock)
	}
	if e.LastBlock != nil {
		protoEpoch.Timestamps.LastBlock = uint64(*e.LastBlock)
	}
	return &protoEpoch
}

func EpochFromProto(ee eventspb.EpochEvent, txHash TxHash) Epoch {
	var endTime *time.Time
	if ee.Action == vega.EpochAction_EPOCH_ACTION_END {
		t := NanosToPostgresTimestamp(ee.EndTime)
		endTime = &t
	}
	epoch := Epoch{
		ID:         int64(ee.Seq),
		StartTime:  NanosToPostgresTimestamp(ee.StartTime),
		ExpireTime: NanosToPostgresTimestamp(ee.ExpireTime),
		EndTime:    endTime,
		TxHash:     txHash,
	}
	return epoch
}
