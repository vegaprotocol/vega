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

package events

import (
	"context"

	vegapb "code.vegaprotocol.io/vega/protos/vega"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

type OracleSpec struct {
	*Base
	o *vegapb.OracleSpec
}

func NewOracleSpecEvent(ctx context.Context, spec *vegapb.OracleSpec) *OracleSpec {
	return &OracleSpec{
		Base: newBase(ctx, OracleSpecEvent),
		o:    spec,
	}
}

func (o *OracleSpec) OracleSpec() *vegapb.OracleSpec {
	return o.o
}

func (o OracleSpec) Proto() *vegapb.OracleSpec {
	return o.o
}

func (o OracleSpec) StreamMessage() *eventspb.BusEvent {
	spec := o.o

	busEvent := newBusEventFromBase(o.Base)
	busEvent.Event = &eventspb.BusEvent_OracleSpec{
		OracleSpec: spec,
	}

	return busEvent
}

func OracleSpecEventFromStream(ctx context.Context, be *eventspb.BusEvent) *OracleSpec {
	return &OracleSpec{
		Base: newBaseFromBusEvent(ctx, OracleSpecEvent, be),
		o:    be.GetOracleSpec(),
	}
}
