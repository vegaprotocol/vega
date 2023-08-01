package execution_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	dstypes "code.vegaprotocol.io/vega/core/datasource/common"
	"code.vegaprotocol.io/vega/core/execution"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	vgtest "code.vegaprotocol.io/vega/libs/test"
	paths2 "code.vegaprotocol.io/vega/paths"
	"github.com/stretchr/testify/require"
)

func TestVerifyUpdateMarketState(t *testing.T) {
	now := time.Now()
	exec := getEngine(t, paths2.New(t.TempDir()), now)
	pubKey := &dstypes.SignerPubKey{
		PubKey: &dstypes.PubKey{
			Key: "0xDEADBEEF",
		},
	}
	mkt := newMarket("MarketID", pubKey)
	err := exec.engine.SubmitMarket(context.Background(), mkt, "", time.Now())
	require.NoError(t, err)

	config := &types.MarketStateUpdateConfiguration{
		MarketID:   "wrong",
		UpdateType: types.MarketStateUpdateTypeTerminate,
	}

	require.Equal(t, execution.ErrMarketDoesNotExist, exec.engine.VerifyUpdateMarketState(config))

	config.MarketID = mkt.ID
	require.Equal(t, fmt.Errorf("missing settlement price for governance initiated futures market termination"), exec.engine.VerifyUpdateMarketState(config))
}

func TestTerminateMarketViaGovernance(t *testing.T) {
	ctx := vgtest.VegaContext("chainid", 100)

	now := time.Now()
	exec := getEngine(t, paths2.New(t.TempDir()), now)
	pubKey := &dstypes.SignerPubKey{
		PubKey: &dstypes.PubKey{
			Key: "0xDEADBEEF",
		},
	}
	mkt := newMarket("MarketID", pubKey)
	err := exec.engine.SubmitMarket(context.Background(), mkt, "", time.Now())
	require.NoError(t, err)

	exec.engine.StartOpeningAuction(context.Background(), mkt.ID)

	config := &types.MarketStateUpdateConfiguration{
		MarketID:        mkt.ID,
		UpdateType:      types.MarketStateUpdateTypeTerminate,
		SettlementPrice: num.NewUint(100),
	}
	require.NoError(t, exec.engine.UpdateMarketState(ctx, config))
	state, err := exec.engine.GetMarketState(mkt.ID)
	require.NoError(t, err)
	require.Equal(t, types.MarketStateClosed, state)
}

func TestSuspendMarketViaGovernance(t *testing.T) {
	ctx := vgtest.VegaContext("chainid", 100)
	now := time.Now()
	exec := getEngine(t, paths2.New(t.TempDir()), now)
	pubKey := &dstypes.SignerPubKey{
		PubKey: &dstypes.PubKey{
			Key: "0xDEADBEEF",
		},
	}
	mkt := newMarket("MarketID", pubKey)
	err := exec.engine.SubmitMarket(context.Background(), mkt, "", time.Now())
	require.NoError(t, err)

	exec.engine.StartOpeningAuction(context.Background(), mkt.ID)

	// during opening auction
	state, err := exec.engine.GetMarketData(mkt.ID)
	require.NoError(t, err)
	require.Equal(t, types.MarketStateActive, state.MarketState)
	require.Equal(t, types.MarketTradingModeContinuous, state.MarketTradingMode)

	config := &types.MarketStateUpdateConfiguration{
		MarketID:        mkt.ID,
		UpdateType:      types.MarketStateUpdateTypeSuspend,
		SettlementPrice: num.NewUint(100),
	}
	require.NoError(t, exec.engine.UpdateMarketState(ctx, config))

	// after governance suspension
	state, err = exec.engine.GetMarketData(mkt.ID)
	require.NoError(t, err)
	require.Equal(t, types.MarketStateSuspendedViaGovernance, state.MarketState)
	require.Equal(t, types.MarketTradingModeSuspendedViaGovernance, state.MarketTradingMode)

	exec.engine.OnTick(ctx, exec.timeService.GetTimeNow())

	config.UpdateType = types.MarketStateUpdateTypeResume
	require.NoError(t, exec.engine.UpdateMarketState(ctx, config))

	// after governance suspension ended - enter liquidity auction
	state, err = exec.engine.GetMarketData(mkt.ID)
	require.NoError(t, err)
	require.Equal(t, types.MarketStateSuspended, state.MarketState)
	require.Equal(t, types.MarketTradingModeMonitoringAuction, state.MarketTradingMode)

	// now suspend via governance again
	config.UpdateType = types.MarketStateUpdateTypeSuspend
	require.NoError(t, exec.engine.UpdateMarketState(ctx, config))

	exec.engine.OnTick(ctx, exec.timeService.GetTimeNow())

	// after governance suspension
	state, err = exec.engine.GetMarketData(mkt.ID)
	require.NoError(t, err)
	require.Equal(t, types.MarketStateSuspendedViaGovernance, state.MarketState)
	// because we're in monitoring auction and the state here is taken from the auction state it is reported as monitoring auction
	require.Equal(t, types.MarketTradingModeMonitoringAuction, state.MarketTradingMode)

	// release suspension should go back to liquidity auction
	config.UpdateType = types.MarketStateUpdateTypeResume
	require.NoError(t, exec.engine.UpdateMarketState(ctx, config))

	exec.engine.OnTick(ctx, exec.timeService.GetTimeNow())

	// after governance suspension ended - enter liquidity auction
	state, err = exec.engine.GetMarketData(mkt.ID)
	require.NoError(t, err)
	require.Equal(t, types.MarketStateSuspended, state.MarketState)
	require.Equal(t, types.MarketTradingModeMonitoringAuction, state.MarketTradingMode)
}
