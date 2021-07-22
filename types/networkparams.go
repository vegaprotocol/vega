package types

import proto "code.vegaprotocol.io/data-node/proto/vega"

type NetworkParameter struct {
	Key, Value string
}

func NetworkParameterFromProto(p *proto.NetworkParameter) *NetworkParameter {
	return &NetworkParameter{
		Key:   p.Key,
		Value: p.Value,
	}
}

func (n NetworkParameter) IntoProto() *proto.NetworkParameter {
	return &proto.NetworkParameter{
		Key:   n.Key,
		Value: n.Value,
	}
}

func (n NetworkParameter) String() string {
	return n.IntoProto().String()
}

func (n NetworkParameter) DeepClone() *NetworkParameter {
	return &NetworkParameter{
		Key:   n.Key,
		Value: n.Value,
	}
}
