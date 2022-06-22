// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package staking_test

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/data-node/staking"
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
	"github.com/stretchr/testify/assert"
)

type stakingServiceTest struct {
	*staking.Service
	ctx   context.Context
	cfunc context.CancelFunc
}

func getStakingService(t *testing.T) *stakingServiceTest {
	log := logging.NewTestLogger()
	ctx, cfunc := context.WithCancel(context.Background())
	s := staking.NewService(ctx, log)
	return &stakingServiceTest{
		Service: s,
		ctx:     ctx,
		cfunc:   cfunc,
	}
}

func (s *stakingServiceTest) Finish() {
	s.cfunc() // cancel context
}

func TestStakingServicePlugin(t *testing.T) {
	t.Run("Get deposit by party", testGetStakeByParty)
}

func testGetStakeByParty(t *testing.T) {
	svc := getStakingService(t)
	defer svc.Finish()

	evts := []events.Event{
		// party 1 just one deposit
		events.NewStakeLinking(
			svc.ctx,
			types.StakeLinking{
				ID:          "1",
				Party:       "party-1",
				TS:          1,
				Amount:      num.NewUint(100),
				Type:        types.StakeLinkingTypeDeposited,
				Status:      types.StakeLinkingStatusAccepted,
				FinalizedAt: 1,
				TxHash:      "0xdeabeef",
			},
		),
		// party 2 inverted order multi deposits
		// we received first a later deposit
		// balance is 0
		events.NewStakeLinking(
			svc.ctx,
			types.StakeLinking{
				ID:          "4",
				Party:       "party-2",
				TS:          1000,
				Amount:      num.NewUint(1000),
				Type:        types.StakeLinkingTypeDeposited,
				Status:      types.StakeLinkingStatusAccepted,
				FinalizedAt: 1000,
				TxHash:      "0xdeabeef",
			},
		),
		// then a remove
		// balance will still be 0
		events.NewStakeLinking(
			svc.ctx,
			types.StakeLinking{
				ID:          "3",
				Party:       "party-2",
				TS:          500,
				Amount:      num.NewUint(100),
				Type:        types.StakeLinkingTypeRemoved,
				Status:      types.StakeLinkingStatusAccepted,
				FinalizedAt: 500,
				TxHash:      "0xdeabeef",
			},
		),
		// then the initial deposit, balance is unlocked to 1100
		events.NewStakeLinking(
			svc.ctx,
			types.StakeLinking{
				ID:          "2",
				Party:       "party-2",
				TS:          1,
				Amount:      num.NewUint(200),
				Type:        types.StakeLinkingTypeDeposited,
				Status:      types.StakeLinkingStatusAccepted,
				FinalizedAt: 1,
				TxHash:      "0xdeabeef",
			},
		),
		// this is rejected, no impact on balance
		events.NewStakeLinking(
			svc.ctx,
			types.StakeLinking{
				ID:          "6",
				Party:       "party-2",
				TS:          1001,
				Amount:      num.NewUint(200),
				Type:        types.StakeLinkingTypeDeposited,
				Status:      types.StakeLinkingStatusRejected,
				FinalizedAt: 1001,
				TxHash:      "0xdeabeef",
			},
		),
	}

	svc.Push(evts...)
	var (
		hasError = true
		retries  = 50
	)
	for hasError && retries > 0 {
		time.Sleep(50 * time.Millisecond)
		retries -= 1
		bal, links := svc.GetStake("party-1")
		if !bal.EQ(num.NewUint(100)) {
			continue
		}
		assert.Len(t, links, 1)

		bal, links = svc.GetStake("party-2")
		if !bal.EQ(num.NewUint(1100)) {
			continue
		}
		assert.Len(t, links, 4)

		hasError = false
	}

	assert.False(t, hasError)
}
