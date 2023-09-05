// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

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
