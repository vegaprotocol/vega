package events

import (
	"context"

	eventspb "code.vegaprotocol.io/vega/proto/events/v1"
	oraclespb "code.vegaprotocol.io/vega/proto/oracles/v1"
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
	return &eventspb.BusEvent{
		Id:    o.eventID(),
		Block: o.TraceID(),
		Type:  o.et.ToProto(),
		Event: &eventspb.BusEvent_OracleData{
			OracleData: &spec,
		},
	}
}
