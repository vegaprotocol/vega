//go:build !qa
// +build !qa

package statevar

import vegapb "code.vegaprotocol.io/protos/vega"

func (sv *StateVariable) AddNoise(kvb []*vegapb.KeyValueBundle) []*vegapb.KeyValueBundle {
	return kvb
}
