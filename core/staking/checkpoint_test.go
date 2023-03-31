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

package staking_test

import (
	"context"
	_ "embed"
	"testing"

	"code.vegaprotocol.io/vega/core/staking"
	"code.vegaprotocol.io/vega/core/staking/mocks"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/proto"
	"code.vegaprotocol.io/vega/logging"
	checkpoint "code.vegaprotocol.io/vega/protos/vega/checkpoint/v1"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed testcp/20220627071230-316971-11a4d958cb7e0448f0cea0b7c617a1e4535e90c0d0f18fd86e961c97147757d7.cp
var cpFile []byte

type checkpointTest struct {
	*staking.Checkpoint
	sv  *stakeVerifierTest
	acc *accountingTest

	ctrl           *gomock.Controller
	ethEventSource *mocks.MockEthereumEventSource
}

func getCheckpointTest(t *testing.T) *checkpointTest {
	t.Helper()
	sv := getStakeVerifierTest(t)
	acc := getAccountingTest(t)

	ctrl := gomock.NewController(t)
	ethEventSource := mocks.NewMockEthereumEventSource(ctrl)

	return &checkpointTest{
		Checkpoint: staking.NewCheckpoint(
			logging.NewTestLogger(),
			acc.Accounting,
			sv.StakeVerifier,
			ethEventSource,
		),
		sv:             sv,
		acc:            acc,
		ctrl:           ctrl,
		ethEventSource: ethEventSource,
	}
}

func (c *checkpointTest) Finish() {
	c.ctrl.Finish()
	c.sv.ctrl.Finish()
}

// TestCheckpointLoadNoDuplicates is testing that with the recent changes
// balance on mainnet are reconciled accurately. Due to a bug some eth events
// go duplicated and balances became incorrect. The Load call without duplication
// for the events would have panicked, this should not.
func TestCheckpointLoadNoDuplicates(t *testing.T) {
	cptest := getCheckpointTest(t)
	defer cptest.Finish()

	cp := &checkpoint.Checkpoint{}
	if err := proto.Unmarshal(cpFile, cp); err != nil {
		t.Fatal(err)
	}

	cptest.acc.broker.EXPECT().Send(gomock.Any()).AnyTimes()
	cptest.acc.broker.EXPECT().SendBatch(gomock.Any()).Times(1)
	cptest.ethEventSource.EXPECT().UpdateStakingStartingBlock(gomock.Any()).Do(
		func(block uint64) {
			// ensure we restart at the right block
			// which is the last pending event we've seen
			assert.Equal(t, int(block), 15026715)
		},
	)
	require.NotPanics(t, func() { cptest.Load(context.Background(), cp.Staking) })

	// now we ensure the balance which were incorrect are now OK
	balance, err := cptest.acc.GetAvailableBalance(
		"657c2a8a5867c43c831e24820b7544e2fdcc1cf610cfe0ece940fe78137400fd")
	assert.NoError(t, err)
	assert.Equal(t, balance, num.NewUint(0))
}

func TestCheckpoint(t *testing.T) {
	cptest := getCheckpointTest(t)
	defer cptest.Finish()

	cptest.setupAccounting(t)
	cptest.setupStakeVerifier(t)

	cp, err := cptest.Checkpoint.Checkpoint()
	assert.NoError(t, err)
	assert.True(t, len(cp) > 0)

	cptest2 := getCheckpointTest(t)

	cptest2.acc.broker.EXPECT().Send(gomock.Any()).Times(1)
	cptest2.ethEventSource.EXPECT().UpdateStakingStartingBlock(gomock.Any()).Do(
		func(block uint64) {
			// ensure we restart at the right block
			// which is the last pending event we've seen
			assert.Equal(t, int(block), 42)
		},
	)

	cptest2.acc.broker.EXPECT().SendBatch(gomock.Any()).Times(1)
	assert.NoError(t, cptest2.Load(context.Background(), cp))

	bal, err := cptest2.acc.GetAvailableBalance(testParty)
	assert.NoError(t, err)
	assert.Equal(t, bal, num.NewUint(10))
}

func (c *checkpointTest) setupAccounting(t *testing.T) {
	t.Helper()
	ctx := context.Background()
	c.acc.broker.EXPECT().Send(gomock.Any()).Times(1)

	evt := &types.StakeLinking{
		ID:              "someid1",
		Type:            types.StakeLinkingTypeDeposited,
		TS:              100,
		Party:           testParty,
		Amount:          num.NewUint(10),
		Status:          types.StakeLinkingStatusAccepted,
		FinalizedAt:     100,
		TxHash:          "0x123456",
		BlockHeight:     1000,
		BlockTime:       10,
		LogIndex:        100,
		EthereumAddress: "0x123456",
	}
	c.acc.AddEvent(ctx, evt)
}

func (c *checkpointTest) setupStakeVerifier(t *testing.T) {
	t.Helper()
	c.sv.tsvc.EXPECT().GetTimeNow().Times(1)
	c.sv.broker.EXPECT().Send(gomock.Any()).Times(1)
	c.sv.witness.EXPECT().StartCheck(gomock.Any(), gomock.Any(), gomock.Any()).Times(1)

	event := &types.StakeDeposited{
		BlockNumber:     42,
		LogIndex:        1789,
		TxID:            "somehash",
		ID:              "someid",
		VegaPubKey:      "somepubkey",
		EthereumAddress: "0xnothex",
		Amount:          num.NewUint(1000),
		BlockTime:       100000,
	}

	err := c.sv.ProcessStakeDeposited(context.Background(), event)
	require.Nil(t, err)
}
