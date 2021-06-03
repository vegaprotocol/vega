package steps

import (
	"strconv"
	"time"

	"code.vegaprotocol.io/vega/integration/helpers"
	"code.vegaprotocol.io/vega/integration/stubs"
)

func TheAverageBlockDurationIs(block *helpers.Block, dur string) error {
	avg, err := strconv.ParseInt(dur, 10, 0)
	if err != nil {
		return err
	}
	block.Duration = avg
	return nil
}

func TheAverageBlockDurationWithVariance(block *helpers.Block, dur, variance string) error {
	if err := TheAverageBlockDurationIs(block, dur); err != nil {
		return err
	}
	v, err := strconv.ParseFloat(variance, 10)
	if err != nil {
		return err
	}
	block.Variance = v
	return nil
}

func TheNetworkMovesAheadNBlocks(block *helpers.Block, time *stubs.TimeStub, n string) error {
	nr, err := strconv.ParseInt(n, 10, 0)
	if err != nil {
		return err
	}
	now, err := time.GetTimeNow()
	if err != nil {
		return err
	}
	for i := int64(0); i < nr; i++ {
		now = now.Add(block.GetDuration())
		// progress time
		time.SetTime(now)
	}
	return nil
}

func TheNetworkMovesAheadDurationWithBlocks(block *helpers.Block, ts *stubs.TimeStub, delta, dur string) error {
	td, err := time.ParseDuration(delta)
	if err != nil {
		return err
	}
	bd, err := time.ParseDuration(dur)
	if err != nil {
		return err
	}
	now, err := ts.GetTimeNow()
	if err != nil {
		return err
	}
	target := now.Add(td)
	for now.Before(target) {
		now = now.Add(block.GetStep(bd))
		ts.SetTime(now)
	}
	return nil
}
