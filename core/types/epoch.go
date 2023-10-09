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
