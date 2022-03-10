package staking_test

import (
	"context"
	"testing"

	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/staking"
	"code.vegaprotocol.io/vega/staking/mocks"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
