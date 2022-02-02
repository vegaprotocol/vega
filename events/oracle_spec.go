package events

import (
	"context"

	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	oraclespb "code.vegaprotocol.io/protos/vega/oracles/v1"
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

	busEvent := newBusEventFromBase(o.Base)
	busEvent.Event = &eventspb.BusEvent_OracleSpec{
		OracleSpec: &spec,
	}

	return busEvent
}

func OracleSpecEventFromStream(ctx context.Context, be *eventspb.BusEvent) *OracleSpec {
	return &OracleSpec{
		Base: newBaseFromBusEvent(ctx, OracleSpecEvent, be),
		o:    *be.GetOracleSpec(),
	}
}
