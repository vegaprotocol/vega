package types

import "code.vegaprotocol.io/vega/proto"

type OracleSpecToFutureBinding struct {
	SettlementPriceProperty string
}

func OracleSpecToFutureBindingFromProto(o *proto.OracleSpecToFutureBinding) *OracleSpecToFutureBinding {
	return &OracleSpecToFutureBinding{
		SettlementPriceProperty: o.SettlementPriceProperty,
	}
}

func (o OracleSpecToFutureBinding) IntoProto() *proto.OracleSpecToFutureBinding {
	return &proto.OracleSpecToFutureBinding{
		SettlementPriceProperty: o.SettlementPriceProperty,
	}
}

func (o OracleSpecToFutureBinding) String() string {
	return o.IntoProto().String()
}
