package types

import "code.vegaprotocol.io/vega/proto"

type NetworkParameter struct {
	Key, Value string
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
