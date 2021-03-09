package events

import (
	"context"

	types "code.vegaprotocol.io/vega/proto"
	oraclespb "code.vegaprotocol.io/vega/proto/oracles/v1"
)

type OracleSpec struct {
	*Base
	o oraclespb.OracleSpec
}

func NewOracleSpecEvent(ctx context.Context, spec oraclespb.OracleSpec) *OracleSpec {
	return &OracleSpec{
		Base: newBase(ctx, OracleSpecEvent),
		o:    spec,
	}
}

func (o *OracleSpec) OracleSpec() oraclespb.OracleSpec {
	return o.o
}

func (o OracleSpec) Proto() oraclespb.OracleSpec {
	return o.o
}

func (o OracleSpec) StreamMessage() *types.BusEvent {
	spec := o.o
	return &types.BusEvent{
		Id:    o.eventID(),
		Block: o.TraceID(),
		Type:  o.et.ToProto(),
		Event: &types.BusEvent_OracleSpec{
			OracleSpec: &spec,
		},
	}
}
