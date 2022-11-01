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

	datapb "code.vegaprotocol.io/vega/protos/vega/data/v1"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

type OracleData struct {
	*Base
	o datapb.OracleData
}

func NewOracleDataEvent(ctx context.Context, spec datapb.OracleData) *OracleData {
	cpy := &datapb.Data{}
	if spec.ExternalData != nil {
		if spec.ExternalData.Data != nil {
			cpy = spec.ExternalData.Data.DeepClone()
		}
	}

	return &OracleData{
		Base: newBase(ctx, OracleDataEvent),
		o:    datapb.OracleData{ExternalData: &datapb.ExternalData{Data: cpy}},
	}
}

func (o *OracleData) OracleData() datapb.OracleData {
	data := datapb.OracleData{
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

func (o OracleData) Proto() datapb.OracleData {
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
