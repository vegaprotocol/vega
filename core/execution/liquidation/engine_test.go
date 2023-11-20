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

package liquidation_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	bmocks "code.vegaprotocol.io/vega/core/broker/mocks"
	"code.vegaprotocol.io/vega/core/events"
	cmocks "code.vegaprotocol.io/vega/core/execution/common/mocks"
	"code.vegaprotocol.io/vega/core/execution/liquidation"
	"code.vegaprotocol.io/vega/core/execution/liquidation/mocks"
	"code.vegaprotocol.io/vega/core/types"
	vegacontext "code.vegaprotocol.io/vega/libs/context"
	vgcrypto "code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/num"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

type tstEngine struct {
	*liquidation.Engine
	ctrl   *gomock.Controller
	book   *mocks.MockBook
	ml     *mocks.MockMarketLiquidity
	idgen  *mocks.MockIDGen
	as     *cmocks.MockAuctionState
	broker *bmocks.MockBroker
	tSvc   *cmocks.MockTimeService
	pos    *mocks.MockPositions
}

type marginStub struct {
	party  string
	size   int64
	market string
}

type SliceLenMatcher[T any] int

func TestOrderbookPriceLimits(t *testing.T) {
	t.Run("orderbook has no volume", testOrderbookHasNoVolume)
	t.Run("orderbook has a volume of one (consumed fraction rounding)", testOrderbookFractionRounding)
	t.Run("orderbook has plenty of volume (should not increase order size)", testOrderbookExceedsVolume)
}

func TestNetworkReducesOverTime(t *testing.T) {
	// basic setup can be shared across these tests
	mID := "intervalMkt"
	ctx := vegacontext.WithTraceID(context.Background(), vgcrypto.RandomHash())
	config := &types.LiquidationStrategy{
		DisposalTimeStep:    5 * time.Second,           // decrease volume every 5 seconds
		DisposalFraction:    num.DecimalFromFloat(0.1), // remove 10% each step
		FullDisposalSize:    10,                        //  a volume of 10 or less can be removed in one go
		MaxFractionConsumed: num.DecimalFromFloat(0.2), // never use more than 20% of the available volume
	}
	eng := getTestEngine(t, mID, config.DeepClone())
	defer eng.Finish()
	// setup: create a party with volume of 10 long as the distressed party
	closed := []events.Margin{
		createMarginEvent("party1", mID, 10),
		createMarginEvent("party2", mID, 10),
		createMarginEvent("party3", mID, 10),
		createMarginEvent("party4", mID, 10),
		createMarginEvent("party5", mID, 10),
	}
	totalSize := uint64(50)
	now := time.Now()
	eng.tSvc.EXPECT().GetTimeNow().Times(2).Return(now)
	idCount := len(closed) * 3
	eng.idgen.EXPECT().NextID().Times(idCount).Return("nextID")
	// 2 orders per closed position
	eng.broker.EXPECT().SendBatch(SliceLenMatcher[events.Event](2 * len(closed))).Times(1)
	// 1 trade per closed position
	eng.broker.EXPECT().SendBatch(SliceLenMatcher[events.Event](1 * len(closed))).Times(1)
	eng.pos.EXPECT().RegisterOrder(gomock.Any(), gomock.Any()).Times(2 * len(closed))
	eng.pos.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(len(closed))
	pos, parties, trades := eng.ClearDistressedParties(ctx, eng.idgen, closed, num.UintZero(), num.UintZero())
	require.Equal(t, len(closed), len(trades))
	require.Equal(t, len(closed), len(pos))
	require.Equal(t, len(closed), len(parties))
	require.Equal(t, closed[0].Party(), parties[0])

	t.Run("call to ontick within the time step does nothing", func(t *testing.T) {
		now = now.Add(2 * time.Second)
		eng.as.EXPECT().InAuction().Times(1).Return(false)
		order, err := eng.OnTick(ctx, now)
		require.Nil(t, order)
		require.NoError(t, err)
	})

	t.Run("after the time step passes, the first batch is disposed of", func(t *testing.T) {
		now = now.Add(3 * time.Second)
		minP, maxP := num.UintZero(), num.UintOne()
		eng.as.EXPECT().InAuction().Times(1).Return(false)
		eng.ml.EXPECT().ValidOrdersPriceRange().Times(1).Return(minP, maxP, nil)
		// return a large volume so the full step is disposed
		eng.book.EXPECT().GetVolumeAtPrice(gomock.Any(), gomock.Any()).Times(1).Return(uint64(1000))
		order, err := eng.OnTick(ctx, now)
		require.NoError(t, err)
		require.NotNil(t, order)
		require.Equal(t, uint64(5), order.Size)
	})

	t.Run("ensure the next time step is set", func(t *testing.T) {
		now = now.Add(2 * time.Second)
		eng.as.EXPECT().InAuction().Times(1).Return(false)
		order, err := eng.OnTick(ctx, now)
		require.Nil(t, order)
		require.NoError(t, err)
	})

	// ready to dispose again from here on
	t.Run("while in auction, the position is not reduced", func(t *testing.T) {
		// pass another step
		now = now.Add(3 * time.Second)
		eng.as.EXPECT().InAuction().Times(1).Return(true)
		order, err := eng.OnTick(ctx, now)
		require.Nil(t, order)
		require.NoError(t, err)
	})

	t.Run("when not in auction, if there is no price range, there is no trade", func(t *testing.T) {
		eng.as.EXPECT().InAuction().Times(1).Return(false)
		mlErr := errors.New("some error")
		eng.ml.EXPECT().ValidOrdersPriceRange().Times(1).Return(nil, nil, mlErr)
		order, err := eng.OnTick(ctx, now)
		require.Nil(t, order)
		require.Error(t, err)
		require.Equal(t, mlErr, err)
	})

	t.Run("No longer in auction and we have a price range finally generates the order", func(t *testing.T) {
		minP, maxP := num.UintZero(), num.UintOne()
		eng.as.EXPECT().InAuction().Times(1).Return(false)
		eng.ml.EXPECT().ValidOrdersPriceRange().Times(1).Return(minP, maxP, nil)
		// return a large volume so the full step is disposed
		eng.book.EXPECT().GetVolumeAtPrice(gomock.Any(), gomock.Any()).Times(1).Return(uint64(1000))
		order, err := eng.OnTick(ctx, now)
		require.NoError(t, err)
		require.NotNil(t, order)
		require.Equal(t, uint64(5), order.Size)
	})

	t.Run("increasing the position of the network does not change the time step", func(t *testing.T) {
		now = now.Add(time.Second)
		closed := []events.Margin{
			createMarginEvent("party", mID, 1),
		}
		eng.tSvc.EXPECT().GetTimeNow().Times(1).Return(now)
		idCount := len(closed) * 3
		eng.idgen.EXPECT().NextID().Times(idCount).Return("nextID")
		// 2 orders per closed position
		eng.broker.EXPECT().SendBatch(SliceLenMatcher[events.Event](2 * len(closed))).Times(1)
		// 1 trade per closed position
		eng.broker.EXPECT().SendBatch(SliceLenMatcher[events.Event](1 * len(closed))).Times(1)
		eng.pos.EXPECT().RegisterOrder(gomock.Any(), gomock.Any()).Times(len(closed) * 2)
		eng.pos.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(len(closed))
		pos, parties, trades := eng.ClearDistressedParties(ctx, eng.idgen, closed, num.UintZero(), num.UintZero())
		require.Equal(t, len(closed), len(trades))
		require.Equal(t, len(closed), len(pos))
		require.Equal(t, len(closed), len(parties))
		require.Equal(t, closed[0].Party(), parties[0])
		totalSize++
		// now increase time by 4 seconds should dispose 5.1 -> 5
		now = now.Add(4 * time.Second)
		minP, maxP := num.UintZero(), num.UintOne()
		eng.as.EXPECT().InAuction().Times(1).Return(false)
		eng.ml.EXPECT().ValidOrdersPriceRange().Times(1).Return(minP, maxP, nil)
		// return a large volume so the full step is disposed
		eng.book.EXPECT().GetVolumeAtPrice(gomock.Any(), gomock.Any()).Times(1).Return(uint64(1000))
		order, err := eng.OnTick(ctx, now)
		require.NoError(t, err)
		require.NotNil(t, order)
		require.Equal(t, totalSize/10, order.Size)
	})

	t.Run("Updating the config changes the time left until the next step", func(t *testing.T) {
		now = now.Add(time.Second)
		eng.as.EXPECT().InAuction().Times(1).Return(false)
		order, err := eng.OnTick(ctx, now)
		require.Nil(t, order)
		require.NoError(t, err)
		// 4s to go, but...
		config.DisposalTimeStep = 3 * time.Second
		eng.Update(config.DeepClone())
		now = now.Add(2 * time.Second)
		// only 3 seconds later and we dispose of the next batch
		minP, maxP := num.UintZero(), num.UintOne()
		eng.as.EXPECT().InAuction().Times(1).Return(false)
		eng.ml.EXPECT().ValidOrdersPriceRange().Times(1).Return(minP, maxP, nil)
		// return a large volume so the full step is disposed
		eng.book.EXPECT().GetVolumeAtPrice(gomock.Any(), gomock.Any()).Times(1).Return(uint64(1000))
		order, err = eng.OnTick(ctx, now)
		require.NoError(t, err)
		require.NotNil(t, order)
		require.Equal(t, totalSize/10, order.Size)
	})

	t.Run("Once the remaining volume of the network is LTE full disposal position, the network creates an order for its full position", func(t *testing.T) {
		// use trades to reduce its position
		size := uint64(eng.GetNetworkPosition().Size()) - config.FullDisposalSize
		eng.UpdateNetworkPosition([]*types.Trade{
			{
				ID:       "someTrade",
				MarketID: mID,
				Size:     size,
			},
		})
		require.True(t, uint64(eng.GetNetworkPosition().Size()) <= config.FullDisposalSize)
		now = now.Add(3 * time.Second)
		// only 3 seconds later and we dispose of the next batch
		minP, maxP := num.UintZero(), num.UintOne()
		eng.as.EXPECT().InAuction().Times(1).Return(false)
		eng.ml.EXPECT().ValidOrdersPriceRange().Times(1).Return(minP, maxP, nil)
		// return a large volume so the full step is disposed
		eng.book.EXPECT().GetVolumeAtPrice(gomock.Any(), gomock.Any()).Times(1).Return(uint64(1000))
		order, err := eng.OnTick(ctx, now)
		require.NoError(t, err)
		require.NotNil(t, order)
		require.Equal(t, config.FullDisposalSize, order.Size)
	})
}

func testOrderbookHasNoVolume(t *testing.T) {
	mID := "market"
	ctx := vegacontext.WithTraceID(context.Background(), vgcrypto.RandomHash())
	eng := getTestEngine(t, mID, nil)
	defer eng.Finish()
	// setup: create a party with volume of 10 long as the distressed party
	closed := []events.Margin{
		createMarginEvent("party", mID, 10),
	}
	now := time.Now()
	eng.tSvc.EXPECT().GetTimeNow().Times(2).Return(now)
	idCount := len(closed) * 3
	eng.idgen.EXPECT().NextID().Times(idCount).Return("nextID")
	// 2 orders per closed position
	eng.broker.EXPECT().SendBatch(SliceLenMatcher[events.Event](2 * len(closed))).Times(1)
	// 1 trade per closed position
	eng.broker.EXPECT().SendBatch(SliceLenMatcher[events.Event](1 * len(closed))).Times(1)
	eng.pos.EXPECT().RegisterOrder(gomock.Any(), gomock.Any()).Times(len(closed) * 2)
	eng.pos.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(len(closed))
	pos, parties, trades := eng.ClearDistressedParties(ctx, eng.idgen, closed, num.UintZero(), num.UintZero())
	require.Equal(t, len(closed), len(trades))
	require.Equal(t, len(closed), len(pos))
	require.Equal(t, len(closed), len(parties))
	require.Equal(t, closed[0].Party(), parties[0])
	// now when we close out, the book returns a volume of 0 is available
	minP, maxP := num.UintZero(), num.UintOne()
	eng.as.EXPECT().InAuction().Times(1).Return(false)
	eng.ml.EXPECT().ValidOrdersPriceRange().Times(1).Return(minP, maxP, nil)
	eng.book.EXPECT().GetVolumeAtPrice(minP, types.SideBuy).Times(1).Return(uint64(0))
	order, err := eng.OnTick(ctx, now)
	require.NoError(t, err)
	require.Nil(t, order)
}

func testOrderbookFractionRounding(t *testing.T) {
	mID := "smallMkt"
	ctx := vegacontext.WithTraceID(context.Background(), vgcrypto.RandomHash())
	config := types.LiquidationStrategy{
		DisposalTimeStep:    0,
		DisposalFraction:    num.DecimalOne(),
		FullDisposalSize:    1000000, // plenty
		MaxFractionConsumed: num.DecimalFromFloat(0.5),
	}
	eng := getTestEngine(t, mID, &config)
	defer eng.Finish()
	closed := []events.Margin{
		createMarginEvent("party", mID, 10),
	}
	var netVol int64
	for _, c := range closed {
		netVol += c.Size()
	}
	now := time.Now()
	eng.tSvc.EXPECT().GetTimeNow().Times(2).Return(now)
	idCount := len(closed) * 3
	eng.idgen.EXPECT().NextID().Times(idCount).Return("nextID")
	// 2 orders per closed position
	eng.broker.EXPECT().SendBatch(SliceLenMatcher[events.Event](2 * len(closed))).Times(1)
	// 1 trade per closed position
	eng.broker.EXPECT().SendBatch(SliceLenMatcher[events.Event](1 * len(closed))).Times(1)
	eng.pos.EXPECT().RegisterOrder(gomock.Any(), gomock.Any()).Times(len(closed) * 2)
	eng.pos.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(len(closed))
	pos, parties, trades := eng.ClearDistressedParties(ctx, eng.idgen, closed, num.UintZero(), num.UintZero())
	require.Equal(t, len(closed), len(trades))
	require.Equal(t, len(closed), len(pos))
	require.Equal(t, len(closed), len(parties))
	require.Equal(t, closed[0].Party(), parties[0])
	// now the available volume on the book is 1, with the fraction that gets rounded to 0.5
	// which should be rounded UP to 1.
	minP, maxP := num.UintZero(), num.UintOne()
	eng.as.EXPECT().InAuction().Times(1).Return(false)
	eng.ml.EXPECT().ValidOrdersPriceRange().Times(1).Return(minP, maxP, nil)
	eng.book.EXPECT().GetVolumeAtPrice(minP, types.SideBuy).Times(1).Return(uint64(1))
	order, err := eng.OnTick(ctx, now)
	require.NoError(t, err)
	require.Equal(t, uint64(1), order.Size)
}

func testOrderbookExceedsVolume(t *testing.T) {
	mID := "market"
	ctx := vegacontext.WithTraceID(context.Background(), vgcrypto.RandomHash())
	config := types.LiquidationStrategy{
		DisposalTimeStep:    0,
		DisposalFraction:    num.DecimalOne(),
		FullDisposalSize:    1000000, // plenty
		MaxFractionConsumed: num.DecimalFromFloat(0.5),
	}
	eng := getTestEngine(t, mID, &config)
	defer eng.Finish()
	closed := []events.Margin{
		createMarginEvent("party", mID, 10),
	}
	var netVol int64
	for _, c := range closed {
		netVol += c.Size()
	}
	now := time.Now()
	eng.tSvc.EXPECT().GetTimeNow().Times(2).Return(now)
	idCount := len(closed) * 3
	eng.idgen.EXPECT().NextID().Times(idCount).Return("nextID")
	// 2 orders per closed position
	eng.broker.EXPECT().SendBatch(SliceLenMatcher[events.Event](2 * len(closed))).Times(1)
	// 1 trade per closed position
	eng.broker.EXPECT().SendBatch(SliceLenMatcher[events.Event](1 * len(closed))).Times(1)
	eng.pos.EXPECT().RegisterOrder(gomock.Any(), gomock.Any()).Times(len(closed) * 2)
	eng.pos.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(len(closed))
	pos, parties, trades := eng.ClearDistressedParties(ctx, eng.idgen, closed, num.UintZero(), num.UintZero())
	require.Equal(t, len(closed), len(trades))
	require.Equal(t, len(closed), len(pos))
	require.Equal(t, len(closed), len(parties))
	require.Equal(t, closed[0].Party(), parties[0])
	minP, maxP := num.UintZero(), num.UintOne()
	eng.as.EXPECT().InAuction().Times(1).Return(false)
	eng.ml.EXPECT().ValidOrdersPriceRange().Times(1).Return(minP, maxP, nil)
	// orderbook has 100x the available volume, with a factor of 0.5, that's still 50x
	eng.book.EXPECT().GetVolumeAtPrice(minP, types.SideBuy).Times(1).Return(uint64(netVol * 10))
	order, err := eng.OnTick(ctx, now)
	require.NoError(t, err)
	require.Equal(t, uint64(netVol), order.Size)
}

func TestLegacySupport(t *testing.T) {
	// simple test to make sure that passing nil for the config does not cause issues.
	mID := "market"
	ctx := vegacontext.WithTraceID(context.Background(), vgcrypto.RandomHash())
	eng := getTestEngine(t, mID, nil)
	defer eng.Finish()
	require.False(t, eng.Stopped())
	// let's check if we get back an order, create the margin events
	closed := []events.Margin{
		createMarginEvent("party", mID, 10),
	}
	var netVol int64
	for _, c := range closed {
		netVol += c.Size()
	}
	now := time.Now()
	eng.tSvc.EXPECT().GetTimeNow().Times(2).Return(now)
	idCount := len(closed) * 3
	eng.idgen.EXPECT().NextID().Times(idCount).Return("nextID")
	// 2 orders per closed position
	eng.broker.EXPECT().SendBatch(SliceLenMatcher[events.Event](2 * len(closed))).Times(1)
	// 1 trade per closed position
	eng.broker.EXPECT().SendBatch(SliceLenMatcher[events.Event](1 * len(closed))).Times(1)
	eng.pos.EXPECT().RegisterOrder(gomock.Any(), gomock.Any()).Times(len(closed) * 2)
	eng.pos.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(len(closed))
	pos, parties, trades := eng.ClearDistressedParties(ctx, eng.idgen, closed, num.UintZero(), num.UintZero())
	require.Equal(t, len(closed), len(trades))
	require.Equal(t, len(closed), len(pos))
	require.Equal(t, len(closed), len(parties))
	require.Equal(t, closed[0].Party(), parties[0])
	// now that the network has a position, do the same thing, we should see the time service gets called only once
	closed = []events.Margin{
		createMarginEvent("another party", mID, 5),
	}
	for _, c := range closed {
		netVol += c.Size()
	}
	eng.tSvc.EXPECT().GetTimeNow().Times(1).Return(now)
	idCount = len(closed) * 3
	eng.idgen.EXPECT().NextID().Times(idCount).Return("nextID")
	// 2 orders per closed position
	eng.broker.EXPECT().SendBatch(SliceLenMatcher[events.Event](2 * len(closed))).Times(1)
	// 1 trade per closed position
	eng.broker.EXPECT().SendBatch(SliceLenMatcher[events.Event](1 * len(closed))).Times(1)
	eng.pos.EXPECT().RegisterOrder(gomock.Any(), gomock.Any()).Times(len(closed) * 2)
	eng.pos.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(len(closed))
	pos, parties, trades = eng.ClearDistressedParties(ctx, eng.idgen, closed, num.UintZero(), num.UintZero())
	require.Equal(t, len(closed), len(trades))
	require.Equal(t, len(closed), len(pos))
	require.Equal(t, len(closed), len(parties))
	require.Equal(t, closed[0].Party(), parties[0])
	// now we should see an order for size 15 returned
	minP, maxP := num.UintZero(), num.UintOne()
	eng.as.EXPECT().InAuction().Times(1).Return(false)
	eng.ml.EXPECT().ValidOrdersPriceRange().Times(1).Return(minP, maxP, nil)
	eng.book.EXPECT().GetVolumeAtPrice(minP, types.SideBuy).Times(1).Return(uint64(netVol))
	order, err := eng.OnTick(ctx, now)
	require.NoError(t, err)
	require.Equal(t, uint64(netVol), order.Size)
	// now reduce the network size through distressed short position
	closed = []events.Margin{
		createMarginEvent("another party", mID, -netVol),
	}
	for _, c := range closed {
		netVol += c.Size()
	}
	require.Equal(t, int64(0), netVol)
	// just check the margin position event we return, too
	eng.tSvc.EXPECT().GetTimeNow().Times(1).Return(now)
	idCount = len(closed) * 3
	eng.idgen.EXPECT().NextID().Times(idCount).Return("nextID")
	// 2 orders per closed position
	eng.broker.EXPECT().SendBatch(SliceLenMatcher[events.Event](2 * len(closed))).Times(1)
	// 1 trade per closed position
	eng.broker.EXPECT().SendBatch(SliceLenMatcher[events.Event](1 * len(closed))).Times(1)
	eng.pos.EXPECT().RegisterOrder(gomock.Any(), gomock.Any()).Times(len(closed) * 2)
	eng.pos.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(len(closed))
	pos, parties, trades = eng.ClearDistressedParties(ctx, eng.idgen, closed, num.UintZero(), num.UintZero())
	require.Equal(t, len(closed), len(trades))
	require.Equal(t, len(closed), len(pos))
	require.Equal(t, len(closed), len(parties))
	require.Equal(t, closed[0].Party(), parties[0])
	require.Equal(t, netVol, eng.GetNetworkPosition().Size())
	// now we should see no error, and no order returned
	order, err = eng.OnTick(ctx, now)
	require.NoError(t, err)
	require.Nil(t, order)
	// now just make sure stopping for snapshots works as expected
	eng.StopSnapshots()
	require.True(t, eng.Stopped())
}

func createMarginEvent(party, market string, size int64) events.Margin {
	return &marginStub{
		party:  party,
		market: market,
		size:   size,
	}
}

func (m *marginStub) Party() string {
	return m.party
}

func (m *marginStub) Size() int64 {
	return m.size
}

func (m *marginStub) Buy() int64 {
	return 0
}

func (m *marginStub) Sell() int64 {
	return 0
}

func (m *marginStub) Price() *num.Uint {
	return nil
}

func (m *marginStub) BuySumProduct() *num.Uint {
	return nil
}

func (m *marginStub) SellSumProduct() *num.Uint {
	return nil
}

func (m *marginStub) VWBuy() *num.Uint {
	return nil
}

func (m *marginStub) VWSell() *num.Uint {
	return nil
}

func (m *marginStub) AverageEntryPrice() *num.Uint {
	return nil
}

func (m *marginStub) Asset() string {
	return ""
}

func (m *marginStub) MarginBalance() *num.Uint {
	return nil
}

func (m *marginStub) GeneralBalance() *num.Uint {
	return nil
}

func (m *marginStub) BondBalance() *num.Uint {
	return nil
}

func (m *marginStub) MarketID() string {
	return m.market
}

func (m *marginStub) MarginShortFall() *num.Uint {
	return nil
}

func getTestEngine(t *testing.T, marketID string, config *types.LiquidationStrategy) *tstEngine {
	t.Helper()
	ctrl := gomock.NewController(t)
	book := mocks.NewMockBook(ctrl)
	ml := mocks.NewMockMarketLiquidity(ctrl)
	idgen := mocks.NewMockIDGen(ctrl)
	as := cmocks.NewMockAuctionState(ctrl)
	broker := bmocks.NewMockBroker(ctrl)
	tSvc := cmocks.NewMockTimeService(ctrl)
	pe := mocks.NewMockPositions(ctrl)
	engine := liquidation.New(config, marketID, broker, book, as, tSvc, ml, pe)
	return &tstEngine{
		Engine: engine,
		ctrl:   ctrl,
		book:   book,
		ml:     ml,
		idgen:  idgen,
		as:     as,
		broker: broker,
		tSvc:   tSvc,
		pos:    pe,
	}
}

func (t *tstEngine) Finish() {
	t.ctrl.Finish()
}

func (l SliceLenMatcher[T]) Matches(v any) bool {
	sv, ok := v.([]T)
	if !ok {
		return false
	}
	return len(sv) == int(l)
}

func (l SliceLenMatcher[T]) String() string {
	return fmt.Sprintf("matches slice of length %d", int(l))
}
