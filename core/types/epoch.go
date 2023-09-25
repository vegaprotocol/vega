// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

//lint:file-ignore ST1003 Ignore underscores in names, this is straigh copied from the proto package to ease introducing the domain types

package types

import (
	"fmt"
	"time"

	proto "code.vegaprotocol.io/vega/protos/vega"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

type Epoch struct {
	// Unique identifier that increases by one each epoch
	Seq uint64
	// What time did this epoch start
	StartTime time.Time
	// What time should this epoch end
	ExpireTime time.Time
	// What time did it actually end
	EndTime time.Time
	// What action took place
	Action proto.EpochAction
}

func (e Epoch) String() string {
	str := fmt.Sprintf(
		"seq(%d) action(%s) startTime(%s) expireTime(%s)",
		e.Seq,
		e.Action.String(),
		e.StartTime,
		e.ExpireTime,
	)

	// End time is only defined when the epoch event is an end "action".
	if e.Action == proto.EpochAction_EPOCH_ACTION_END {
		str = fmt.Sprintf("%s endTime(%s)", str, e.EndTime)
	}

	return str
}

func NewEpochFromProto(p *eventspb.EpochEvent) *Epoch {
	e := &Epoch{
		Seq:        p.Seq,
		StartTime:  time.Unix(0, p.StartTime),
		ExpireTime: time.Unix(0, p.ExpireTime),
		EndTime:    time.Unix(0, p.EndTime),
		Action:     p.Action,
	}
	return e
}

func (e Epoch) IntoProto() *eventspb.EpochEvent {
	eProto := &eventspb.EpochEvent{
		Seq:        e.Seq,
		StartTime:  e.StartTime.UnixNano(),
		ExpireTime: e.ExpireTime.UnixNano(),
		Action:     e.Action,
	}

	// End time is only defined when the epoch event is an end "action".
	if e.Action == proto.EpochAction_EPOCH_ACTION_END {
		eProto.EndTime = e.EndTime.UnixNano()
	}

	return eProto
}
