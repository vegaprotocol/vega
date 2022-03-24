package entities

import (
	"time"

	"code.vegaprotocol.io/protos/vega"
)

type NetworkParameter struct {
	Key      string
	Value    string
	VegaTime time.Time
}

func (np *NetworkParameter) ToProto() *vega.NetworkParameter {
	pnp := vega.NetworkParameter{
		Key:   np.Key,
		Value: np.Value,
	}
	return &pnp
}

func NetworkParameterFromProto(pnp *vega.NetworkParameter) (NetworkParameter, error) {
	np := NetworkParameter{
		Key:   pnp.Key,
		Value: pnp.Value,
	}
	return np, nil
}
