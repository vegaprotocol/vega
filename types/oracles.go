package types

import proto "code.vegaprotocol.io/data-node/proto/vega"

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

func (o OracleSpecToFutureBinding) DeepClone() *OracleSpecToFutureBinding {
	return &OracleSpecToFutureBinding{
		SettlementPriceProperty: o.SettlementPriceProperty,
	}
}
