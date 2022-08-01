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

package events

import (
	"context"

	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
	oraclespb "code.vegaprotocol.io/vega/protos/vega/oracles/v1"
)

type OracleData struct {
	*Base
	o oraclespb.OracleData
}

func NewOracleDataEvent(ctx context.Context, spec oraclespb.OracleData) *OracleData {
	cpy := spec.DeepClone()
	return &OracleData{
		Base: newBase(ctx, OracleDataEvent),
		o:    *cpy,
	}
}

func (o *OracleData) OracleData() oraclespb.OracleData {
	return o.o
}

func (o OracleData) Proto() oraclespb.OracleData {
	return o.o
}

func (o OracleData) StreamMessage() *eventspb.BusEvent {
	spec := o.o

	busEvent := newBusEventFromBase(o.Base)
	busEvent.Event = &eventspb.BusEvent_OracleData{
		OracleData: &spec,
	}

	return busEvent
}

func OracleDataEventFromStream(ctx context.Context, be *eventspb.BusEvent) *OracleData {
	return &OracleData{
		Base: newBaseFromBusEvent(ctx, OracleDataEvent, be),
		o:    *be.GetOracleData(),
	}
}
