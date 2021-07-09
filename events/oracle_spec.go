package events

import (
	"context"

	eventspb "code.vegaprotocol.io/vega/proto/events/v1"
	oraclespb "code.vegaprotocol.io/vega/proto/oracles/v1"
)

type OracleSpec struct {
	*Base
	o oraclespb.OracleSpec
}

func NewOracleSpecEvent(ctx context.Context, spec oraclespb.OracleSpec) *OracleSpec {
	cpy := spec.DeepClone()
	return &OracleSpec{
		Base: newBase(ctx, OracleSpecEvent),
		o:    *cpy,
	}
}

func (o *OracleSpec) OracleSpec() oraclespb.OracleSpec {
	return o.o
}

func (o OracleSpec) Proto() oraclespb.OracleSpec {
	return o.o
}

func (o OracleSpec) StreamMessage() *eventspb.BusEvent {
	spec := o.o
	return &eventspb.BusEvent{
		Id:    o.eventID(),
		Block: o.TraceID(),
		Type:  o.et.ToProto(),
		Event: &eventspb.BusEvent_OracleSpec{
			OracleSpec: &spec,
		},
	}
}
