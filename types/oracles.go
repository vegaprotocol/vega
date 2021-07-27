package types

import proto "code.vegaprotocol.io/protos/vega"

type OracleSpecToFutureBinding struct {
	SettlementPriceProperty    string
	TradingTerminationProperty string
}

func OracleSpecToFutureBindingFromProto(o *proto.OracleSpecToFutureBinding) *OracleSpecToFutureBinding {
	return &OracleSpecToFutureBinding{
		SettlementPriceProperty:    o.SettlementPriceProperty,
		TradingTerminationProperty: o.TradingTerminationProperty,
	}
}

func (o OracleSpecToFutureBinding) IntoProto() *proto.OracleSpecToFutureBinding {
	return &proto.OracleSpecToFutureBinding{
		SettlementPriceProperty:    o.SettlementPriceProperty,
		TradingTerminationProperty: o.TradingTerminationProperty,
	}
}

func (o OracleSpecToFutureBinding) String() string {
	return o.IntoProto().String()
}

func (o OracleSpecToFutureBinding) DeepClone() *OracleSpecToFutureBinding {
	return &OracleSpecToFutureBinding{
		SettlementPriceProperty:    o.SettlementPriceProperty,
		TradingTerminationProperty: o.TradingTerminationProperty,
	}
}
