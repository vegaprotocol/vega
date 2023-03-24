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
	// What time did this epoch start
	StartTime time.Time
	// What time should this epoch end
	ExpireTime time.Time
	// What time did it actually end
	EndTime time.Time
	// Unique identifier that increases by one each epoch
	Seq uint64
	// What action took place
	Action proto.EpochAction
}

func (e Epoch) String() string {
	return fmt.Sprintf(
		"seq(%d) startTime(%s) expireTime(%s) endTime(%s) action(%s)",
		e.Seq,
		e.StartTime,
		e.ExpireTime,
		e.EndTime,
		e.Action.String(),
	)
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
	return &eventspb.EpochEvent{
		Seq:        e.Seq,
		StartTime:  e.StartTime.UnixNano(),
		ExpireTime: e.ExpireTime.UnixNano(),
		EndTime:    e.EndTime.UnixNano(),
		Action:     e.Action,
	}
}
