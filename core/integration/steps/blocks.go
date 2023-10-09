// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package steps

import (
	"context"
	"strconv"
	"time"

	"code.vegaprotocol.io/vega/core/integration/helpers"
	"code.vegaprotocol.io/vega/core/integration/stubs"
)

type EpochService interface {
	OnBlockEnd(ctx context.Context)
}

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
	v, err := strconv.ParseFloat(variance, 64)
	if err != nil {
		return err
	}
	block.Variance = v
	return nil
}

func TheNetworkMovesAheadNBlocks(exec Execution, block *helpers.Block, time *stubs.TimeStub, n string, epochService EpochService) error {
	nr, err := strconv.ParseInt(n, 10, 0)
	if err != nil {
		return err
	}
	now := time.GetTimeNow()
	for i := int64(0); i < nr; i++ {
		epochService.OnBlockEnd(context.Background())
		exec.BlockEnd(context.Background())
		now = now.Add(block.GetDuration())
		// progress time
		time.SetTime(now)
	}
	return nil
}

func TheNetworkMovesAheadNEpochs(broker *stubs.BrokerStub, block *helpers.Block, exec Execution, epochService EpochService, ts *stubs.TimeStub, epochs string) error {
	nr, err := strconv.ParseInt(epochs, 10, 0)
	if err != nil {
		return err
	}
	for i := int64(0); i < nr; i++ {
		if err := TheNetworkMovesAheadToTheNextEpoch(broker, block, exec, epochService, ts); err != nil {
			return err
		}
	}
	return nil
}

func TheNetworkMovesAheadToTheNextEpoch(broker *stubs.BrokerStub, block *helpers.Block, exec Execution, epochService EpochService, ts *stubs.TimeStub) error {
	ee := broker.GetCurrentEpoch()
	last := ee.Epoch().GetSeq()
	current := last
	now := ts.GetTimeNow()
	for current == last {
		epochService.OnBlockEnd(context.Background())
		exec.BlockEnd(context.Background())
		now = now.Add(block.GetDuration())
		ts.SetTime(now)
		ee = broker.GetCurrentEpoch()
		current = ee.Epoch().GetSeq()
	}
	return nil
}

func TheNetworkMovesAheadDurationWithBlocks(exec Execution, block *helpers.Block, ts *stubs.TimeStub, delta, dur string) error {
	td, err := time.ParseDuration(delta)
	if err != nil {
		return err
	}
	bd, err := time.ParseDuration(dur)
	if err != nil {
		return err
	}
	now := ts.GetTimeNow()
	target := now.Add(td)
	for now.Before(target) {
		exec.BlockEnd(context.Background())
		now = now.Add(block.GetStep(bd))
		ts.SetTime(now)
	}
	return nil
}
