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

package execution_test

import (
	"context"
	"encoding/hex"
	"fmt"
	"testing"
	"time"

	dstypes "code.vegaprotocol.io/vega/core/datasource/common"
	"code.vegaprotocol.io/vega/core/execution"
	"code.vegaprotocol.io/vega/core/types"
	vgcontext "code.vegaprotocol.io/vega/libs/context"
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
	require.Equal(t, types.MarketStateActive, state.MarketState)
	require.Equal(t, types.MarketTradingModeContinuous, state.MarketTradingMode)

	// now suspend via governance again
	config.UpdateType = types.MarketStateUpdateTypeSuspend
	require.NoError(t, exec.engine.UpdateMarketState(ctx, config))

	exec.engine.OnTick(ctx, exec.timeService.GetTimeNow())

	// after governance suspension
	state, err = exec.engine.GetMarketData(mkt.ID)
	require.NoError(t, err)
	require.Equal(t, types.MarketStateSuspendedViaGovernance, state.MarketState)
	// because we're in monitoring auction and the state here is taken from the auction state it is reported as monitoring auction
	require.Equal(t, types.MarketTradingModeSuspendedViaGovernance, state.MarketTradingMode)

	// release suspension should go back to liquidity auction
	config.UpdateType = types.MarketStateUpdateTypeResume
	require.NoError(t, exec.engine.UpdateMarketState(ctx, config))

	exec.engine.OnTick(ctx, exec.timeService.GetTimeNow())

	// after governance suspension ended - enter liquidity auction
	state, err = exec.engine.GetMarketData(mkt.ID)
	require.NoError(t, err)
	require.Equal(t, types.MarketStateActive, state.MarketState)
	require.Equal(t, types.MarketTradingModeContinuous, state.MarketTradingMode)
}

func TestSubmitOrderWhenSuspended(t *testing.T) {
	ctx := vgtest.VegaContext("chainid", 100)
	now := time.Now()
	exec := getEngineWithParties(t, now, num.NewUint(1000000000), "lp", "p1", "p2", "p3", "p4")
	pubKey := &dstypes.SignerPubKey{
		PubKey: &dstypes.PubKey{
			Key: "0xDEADBEEF",
		},
	}
	mkt := newMarket("MarketID", pubKey)
	err := exec.engine.SubmitMarket(context.Background(), mkt, "", now)
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

	// check we can submit an order
	os1 := &types.OrderSubmission{
		MarketID:    mkt.ID,
		Price:       num.NewUint(99),
		Size:        1,
		Side:        types.SideBuy,
		TimeInForce: types.OrderTimeInForceGTC,
		Type:        types.OrderTypeLimit,
		Reference:   "o1",
	}
	idgen := &stubIDGen{}
	vgctx := vgcontext.WithTraceID(context.Background(), hex.EncodeToString([]byte("0deadbeef")))
	_, err = exec.engine.SubmitOrder(vgctx, os1, "p1", idgen, "o1p1")
	require.NoError(t, err)
}
