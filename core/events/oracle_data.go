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
	datapb "code.vegaprotocol.io/vega/protos/vega/data/v1"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

type OracleData struct {
	*Base
	o vegapb.OracleData
}

func NewOracleDataEvent(ctx context.Context, spec vegapb.OracleData) *OracleData {
	cpy := &datapb.Data{}
	if spec.ExternalData != nil {
		if spec.ExternalData.Data != nil {
			cpy = spec.ExternalData.Data.DeepClone()
		}
	}

	return &OracleData{
		Base: newBase(ctx, OracleDataEvent),
		o:    vegapb.OracleData{ExternalData: &datapb.ExternalData{Data: cpy}},
	}
}

func (o *OracleData) OracleData() vegapb.OracleData {
	data := vegapb.OracleData{
		ExternalData: &datapb.ExternalData{
			Data: &datapb.Data{},
		},
	}
	if o.o.ExternalData != nil {
		if o.o.ExternalData.Data != nil {
			data.ExternalData.Data = o.o.ExternalData.Data.DeepClone()
		}
	}
	return data
}

func (o OracleData) Proto() vegapb.OracleData {
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
