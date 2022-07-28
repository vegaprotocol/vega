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

func TheNetworkMovesAheadNBlocks(block *helpers.Block, time *stubs.TimeStub, n string, epochService EpochService) error {
	nr, err := strconv.ParseInt(n, 10, 0)
	if err != nil {
		return err
	}
	now := time.GetTimeNow()
	for i := int64(0); i < nr; i++ {
		epochService.OnBlockEnd(context.Background())
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
	now := ts.GetTimeNow()
	target := now.Add(td)
	for now.Before(target) {
		now = now.Add(block.GetStep(bd))
		ts.SetTime(now)
	}
	return nil
}
