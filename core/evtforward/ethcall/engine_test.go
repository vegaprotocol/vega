package ethcall_test

import (
	"context"
	"math/big"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/config/encoding"
	"code.vegaprotocol.io/vega/core/evtforward/ethcall"
	"code.vegaprotocol.io/vega/core/evtforward/ethcall/mocks"
	"code.vegaprotocol.io/vega/logging"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var TEST_CONFIG = ethcall.Config{
	Level: encoding.LogLevel{Level: logging.DebugLevel},
}

func TestEngine(t *testing.T) {
	ctx := context.Background()
	tc, err := NewToyChain()
	require.NoError(t, err)

	ctrl := gomock.NewController(t)
	forwarder := mocks.NewMockForwarder(ctrl)

	log := logging.NewTestLogger()
	e, err := ethcall.NewEngine(log, TEST_CONFIG, tc.client, forwarder)
	require.NoError(t, err)

	call, err := ethcall.NewCall("get_uint256", []any{big.NewInt(66)}, tc.contractAddr.Hex(), tc.abiBytes)
	require.NoError(t, err)
	currentEthTime := tc.client.Blockchain().CurrentBlock().Time
	trigger := ethcall.TimeTrigger{Initial: currentEthTime, Every: 20}

	spec := ethcall.Spec{
		Call:    call,
		Trigger: trigger,
	}

	specID, err := e.AddSpec(spec)
	require.NoError(t, err)

	// Make sure engine has a previous block to compare to
	e.OnTick(ctx, time.Now())

	// Every commit advances chain time 10 seconds.
	// This one shouldn't trigger our call because we're set to fire every 20 seconds
	tc.client.Commit()
	e.OnTick(ctx, time.Now())

	// But this one should
	forwarder.EXPECT().ForwardFromSelf(gomock.Any()).Return().Do(func(ce *commandspb.ChainEvent) {
		cc := ce.GetContractCall()
		require.NotNil(t, cc)
		res, err := spec.UnpackResult(cc.Result)
		require.NoError(t, err)

		assert.Equal(t, cc.BlockHeight, uint64(3))
		assert.Equal(t, cc.BlockTime, uint64(30))
		assert.Equal(t, cc.SpecId, specID)
		require.Equal(t, res, []any{big.NewInt(66)})
	})
	tc.client.Commit()
	e.OnTick(ctx, time.Now())
}
