package vegatime

import (
	"fmt"
	"testing"
	"time"
	"vega/core"
	"github.com/stretchr/testify/assert"
)

func TestVegaTimeConverter_BlockToTime(t *testing.T) {
	vega := &core.Vega{
		Config: &core.Config{},
		State: core.NewState(),
	}

	genesisTime := time.Now()
	vega.SetGenesisTime(genesisTime)
	vega.State.Height = 10

	fmt.Printf("%+v", vega.GetGenesisTime())

	vtc := NewVegaTimeConverter(vega)

	// check genesis is block zero
	blockNumber := vtc.TimeToBlock(genesisTime)
	assert.Equal(t, uint64(0), blockNumber)

	// check 5 seconds later from genesis is block number 5
	fiveSecondsFromGenesis := genesisTime.Add(time.Duration(5) * time.Second)
	blockNumber = vtc.TimeToBlock(fiveSecondsFromGenesis)
	assert.Equal(t, uint64(5), blockNumber)

	// check block number 5 is genesis time + 5
	testTime := vtc.BlockToTime(uint64(5))
	assert.Equal(t, fiveSecondsFromGenesis, testTime)

	// check block number 0 is genesis time
	testTime = vtc.BlockToTime(uint64(0))
	assert.Equal(t, genesisTime, testTime)
}
