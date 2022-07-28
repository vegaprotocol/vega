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

package entities

import (
	"time"

	"code.vegaprotocol.io/protos/vega"
	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
)

type Epoch struct {
	ID         int64
	StartTime  time.Time
	ExpireTime time.Time
	EndTime    *time.Time
	VegaTime   time.Time
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
	return &protoEpoch
}

func EpochFromProto(ee eventspb.EpochEvent) Epoch {
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
	}
	return epoch
}
