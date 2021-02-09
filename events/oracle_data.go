package events

import (
	"context"

	types "code.vegaprotocol.io/vega/proto"
	oraclespb "code.vegaprotocol.io/vega/proto/oracles/v1"
)

type OracleData struct {
	*Base
	o oraclespb.OracleData
}

func NewOracleDataEvent(ctx context.Context, spec oraclespb.OracleData) *OracleData {
	return &OracleData{
		Base: newBase(ctx, OracleDataEvent),
		o:    spec,
	}
}

func (o *OracleData) OracleData() oraclespb.OracleData {
	return o.o
}

func (o OracleData) Proto() oraclespb.OracleData {
	return o.o
}

func (o OracleData) StreamMessage() *types.BusEvent {
	spec := o.o
	return &types.BusEvent{
		Id:    o.eventID(),
		Block: o.TraceID(),
		Type:  o.et.ToProto(),
		Event: &types.BusEvent_OracleData{
			OracleData: &spec,
		},
	}
}
