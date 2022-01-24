//go:build !qa
// +build !qa

package statevar

import vegapb "code.vegaprotocol.io/protos/vega"

func AddNoise(kvb []*vegapb.KeyValueBundle) []*vegapb.KeyValueBundle {
	return kvb
}
