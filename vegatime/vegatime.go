package vegatime

import (
	"time"
	"vega/core"
)

type VegaTime interface {
	BlockToTime(blockNumber uint64) time.Time
	TimeToBlock(time time.Time) uint64
}

type vegaTimeConverter struct {
	vega *core.Vega
}

func NewVegaTimeConverter(vega *core.Vega) VegaTime {
	return &vegaTimeConverter{vega}
}

func (vt *vegaTimeConverter) BlockToTime(blockNumber uint64) time.Time {
	timeDuration := time.Duration(blockNumber) * time.Second
	return vt.vega.GetGenesisTime().Add(timeDuration)
}

func (vt *vegaTimeConverter) TimeToBlock(time time.Time) uint64 {
	delta := time.Sub(vt.vega.GetGenesisTime())
	if delta < 0 {
		return 0
	}
	return uint64(delta.Seconds())
}