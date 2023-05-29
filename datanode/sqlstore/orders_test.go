// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.DATANODE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package sqlstore_test

import (
	"context"
	"encoding/hex"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/sqlstore"
	"code.vegaprotocol.io/vega/datanode/sqlstore/helpers"
	"code.vegaprotocol.io/vega/libs/ptr"
	"code.vegaprotocol.io/vega/protos/vega"
)

func addTestOrder(t *testing.T, os *sqlstore.Orders, id entities.OrderID, block entities.Block, party entities.Party, market entities.Market, reference string,
	side types.Side, timeInForce types.OrderTimeInForce, orderType types.OrderType, status types.OrderStatus,
	price, size, remaining int64, seqNum uint64, version int32, lpID []byte, createdAt time.Time, txHash entities.TxHash,
	icebergOpts *vega.IcebergOrder,
) entities.Order {
	t.Helper()
	order := entities.Order{
		ID:              id,
		MarketID:        market.ID,
		PartyID:         party.ID,
		Side:            side,
		Price:           decimal.NewFromInt(price),
		Size:            size,
		Remaining:       remaining,
		TimeInForce:     timeInForce,
		Type:            orderType,
		Status:          status,
		Reference:       reference,
		Version:         version,
		LpID:            lpID,
		PeggedOffset:    decimal.NewFromInt(0),
		PeggedReference: types.PeggedReferenceMid,
		CreatedAt:       createdAt,
		UpdatedAt:       time.Now().Add(5 * time.Second).Truncate(time.Microsecond),
		ExpiresAt:       time.Now().Add(10 * time.Second).Truncate(time.Microsecond),
		VegaTime:        block.VegaTime,
		SeqNum:          seqNum,
		TxHash:          txHash,
	}
	if icebergOpts != nil {
		order.InitialPeakSize = ptr.From(int64(icebergOpts.InitialPeakSize))
		order.MinimumPeakSize = ptr.From(int64(icebergOpts.MinimumPeakSize))
		order.ReservedRemaining = ptr.From(int64(icebergOpts.ReservedRemaining))
	}
	err := os.Add(order)
	require.NoError(t, err)
	return order
}

const numTestOrders = 30

var defaultTxHash = txHashFromString("")

func txHashFromString(s string) entities.TxHash {
	return entities.TxHash(hex.EncodeToString([]byte(s)))
}

func TestOrders(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	ps := sqlstore.NewParties(connectionSource)
	os := sqlstore.NewOrders(connectionSource)
	bs := sqlstore.NewBlocks(connectionSource)
	block := addTestBlock(t, ctx, bs)
	block2 := addTestBlock(t, ctx, bs)

	// Make sure we're starting with an empty set of orders
	emptyOrders, err := os.GetAll(ctx)
	assert.NoError(t, err)
	assert.Empty(t, emptyOrders)

	// Add other stuff order will use
	parties := []entities.Party{
		addTestParty(t, ctx, ps, block),
		addTestParty(t, ctx, ps, block),
		addTestParty(t, ctx, ps, block),
	}

	markets := []entities.Market{
		{ID: entities.MarketID("aa")},
		{ID: entities.MarketID("bb")},
	}

	// Make some orders
	icebergs := []entities.Order{}
	orders := make([]entities.Order, numTestOrders)
	updatedOrders := make([]entities.Order, numTestOrders)
	numOrdersUpdatedInDifferentBlock := 0
	version := int32(1)

	for i := 0; i < numTestOrders; i++ {
		var iceberg *vega.IcebergOrder
		// every 10th is a berg
		if i%10 == 0 {
			iceberg = &vega.IcebergOrder{
				InitialPeakSize:   8,
				MinimumPeakSize:   5,
				ReservedRemaining: 10,
			}
		}

		order := addTestOrder(t, os,
			entities.OrderID(helpers.GenerateID()),
			block,
			parties[i%3],
			markets[i%2],
			fmt.Sprintf("my_reference_%d", i),
			types.SideBuy,
			types.OrderTimeInForceGTC,
			types.OrderTypeLimit,
			types.OrderStatusActive,
			10,
			100,
			60,
			uint64(i),
			version,
			nil,
			block.VegaTime,
			txHashFromString(fmt.Sprintf("tx_hash_%d", i)),
			iceberg,
		)
		orders[i] = order
		if iceberg != nil {
			icebergs = append(icebergs, order)
		}

		// Don't update 1/4 of the orders
		if i%4 == 0 {
			updatedOrders[i] = order
		}

		// Update 1/4 of the orders in the same block
		if i%4 == 1 {
			updatedOrder := order
			updatedOrder.Remaining = 50
			err = os.Add(updatedOrder)
			require.NoError(t, err)
			updatedOrders[i] = updatedOrder
		}
	}

	// Flush everything from the first block
	_, _ = os.Flush(ctx)
	for i := 0; i < numTestOrders; i++ {
		// Update Another 1/4 of the orders in the next block
		if i%4 == 2 {
			updatedOrder := orders[i]
			updatedOrder.Remaining = 25
			updatedOrder.VegaTime = block2.VegaTime
			err = os.Add(updatedOrder)
			require.NoError(t, err)
			numOrdersUpdatedInDifferentBlock++
			updatedOrders[i] = updatedOrder
		}

		// Update Another 1/4 of the orders in the next block with an incremented version
		if i%4 == 3 {
			updatedOrder := orders[i]
			updatedOrder.Remaining = 10
			updatedOrder.VegaTime = block2.VegaTime
			updatedOrder.Version++
			err = os.Add(updatedOrder)
			require.NoError(t, err)
			numOrdersUpdatedInDifferentBlock++
			updatedOrders[i] = updatedOrder
		}
	}

	// Flush everything from the second block
	_, err = os.Flush(ctx)
	require.NoError(t, err)

	t.Run("GetAll", func(t *testing.T) {
		// Check we inserted new rows only when the update was in a different block
		allOrders, err := os.GetAll(ctx)
		require.NoError(t, err)
		assert.Equal(t, numTestOrders+numOrdersUpdatedInDifferentBlock, len(allOrders))
	})

	t.Run("GetByOrderID", func(t *testing.T) {
		// Ensure we get the most recently updated version
		for i := 0; i < numTestOrders; i++ {
			fetchedOrder, err := os.GetOrder(ctx, orders[i].ID.String(), nil)
			require.NoError(t, err)
			assert.Equal(t, fetchedOrder, updatedOrders[i])
		}
	})

	t.Run("GetByOrderID specific version", func(t *testing.T) {
		for i := 0; i < numTestOrders; i++ {
			ver := updatedOrders[i].Version
			fetchedOrder, err := os.GetOrder(ctx, updatedOrders[i].ID.String(), &ver)
			require.NoError(t, err)
			assert.Equal(t, fetchedOrder, updatedOrders[i])
		}
	})

	t.Run("GetByTxHash", func(t *testing.T) {
		fetchedOrders, err := os.GetByTxHash(ctx, txHashFromString("tx_hash_1"))
		require.NoError(t, err)
		assert.Len(t, fetchedOrders, 1)
		assert.Equal(t, fetchedOrders[0], updatedOrders[1])
	})

	t.Run("GetOrderNotFound", func(t *testing.T) {
		notAnOrderID := entities.OrderID(helpers.GenerateID())
		fetchedOrder, err := os.GetOrder(ctx, notAnOrderID.String(), nil)
		require.Error(t, err)
		assert.Equal(t, entities.ErrNotFound, err)
		assert.Equal(t, entities.Order{}, fetchedOrder)
	})

	t.Run("GetByOrderID_Icebergs", func(t *testing.T) {
		for _, berg := range icebergs {
			o, err := os.GetOrder(ctx, berg.ID.String(), nil)
			assert.NoError(t, err)
			assert.Equal(t, berg.InitialPeakSize, o.InitialPeakSize)
			assert.Equal(t, berg.MinimumPeakSize, o.MinimumPeakSize)
			assert.Equal(t, berg.ReservedRemaining, o.ReservedRemaining)
		}
	})
}

func generateTestBlocks(t *testing.T, ctx context.Context, numBlocks int, bs *sqlstore.Blocks) []entities.Block {
	t.Helper()

	source := &testBlockSource{bs, time.Now()}
	blocks := make([]entities.Block, numBlocks)
	for i := 0; i < numBlocks; i++ {
		blocks[i] = source.getNextBlock(t, ctx)
	}
	return blocks
}

func generateParties(t *testing.T, ctx context.Context, numParties int, block entities.Block, ps *sqlstore.Parties) []entities.Party {
	t.Helper()
	parties := make([]entities.Party, numParties)
	for i := 0; i < numParties; i++ {
		parties[i] = addTestParty(t, ctx, ps, block)
	}
	return parties
}

func generateOrderIDs(t *testing.T, numIDs int) []entities.OrderID {
	t.Helper()
	orderIDs := make([]entities.OrderID, numIDs)
	for i := 0; i < numIDs; i++ {
		orderIDs[i] = entities.OrderID(helpers.GenerateID())
	}

	return orderIDs
}

func generateTestOrders(t *testing.T, ctx context.Context, blocks []entities.Block, parties []entities.Party,
	markets []entities.Market, orderIDs []entities.OrderID, os *sqlstore.Orders,
) []entities.Order {
	t.Helper()
	// define the orders we're going to insert
	testOrders := []struct {
		id          entities.OrderID
		block       entities.Block
		party       entities.Party
		market      entities.Market
		side        types.Side
		price       int64
		size        int64
		remaining   int64
		timeInForce types.OrderTimeInForce
		orderType   types.OrderType
		status      types.OrderStatus
		createdAt   time.Time
	}{
		{
			id:          orderIDs[0],
			block:       blocks[0],
			party:       parties[0],
			market:      markets[0],
			side:        types.SideBuy,
			price:       100,
			size:        1000,
			remaining:   1000,
			timeInForce: types.OrderTimeInForceGTC,
			orderType:   types.OrderTypeLimit,
			status:      types.OrderStatusActive,
			createdAt:   blocks[0].VegaTime,
		},
		{
			id:          orderIDs[1],
			block:       blocks[0],
			party:       parties[1],
			market:      markets[0],
			side:        types.SideBuy,
			price:       101,
			size:        2000,
			remaining:   2000,
			timeInForce: types.OrderTimeInForceGTC,
			orderType:   types.OrderTypeLimit,
			status:      types.OrderStatusActive,
			createdAt:   blocks[0].VegaTime,
		},
		{
			id:          orderIDs[2],
			block:       blocks[0],
			party:       parties[2],
			market:      markets[0],
			side:        types.SideSell,
			price:       105,
			size:        1500,
			remaining:   1500,
			timeInForce: types.OrderTimeInForceGTC,
			orderType:   types.OrderTypeLimit,
			status:      types.OrderStatusActive,
			createdAt:   blocks[0].VegaTime,
		},
		{
			id:          orderIDs[3],
			block:       blocks[0],
			party:       parties[3],
			market:      markets[0],
			side:        types.SideSell,
			price:       105,
			size:        800,
			remaining:   8500,
			timeInForce: types.OrderTimeInForceGTC,
			orderType:   types.OrderTypeLimit,
			status:      types.OrderStatusActive,
			createdAt:   blocks[0].VegaTime,
		},
		{
			id:          orderIDs[4],
			block:       blocks[0],
			party:       parties[0],
			market:      markets[1],
			side:        types.SideBuy,
			price:       1000,
			size:        10000,
			remaining:   10000,
			timeInForce: types.OrderTimeInForceGTC,
			orderType:   types.OrderTypeLimit,
			status:      types.OrderStatusActive,
			createdAt:   blocks[0].VegaTime,
		},
		{
			id:          orderIDs[5],
			block:       blocks[1],
			party:       parties[2],
			market:      markets[1],
			side:        types.SideSell,
			price:       1005,
			size:        15000,
			remaining:   15000,
			timeInForce: types.OrderTimeInForceGTC,
			orderType:   types.OrderTypeLimit,
			status:      types.OrderStatusActive,
			createdAt:   blocks[0].VegaTime,
		},
		{
			id:          orderIDs[6],
			block:       blocks[1],
			party:       parties[3],
			market:      markets[2],
			side:        types.SideSell,
			price:       1005,
			size:        15000,
			remaining:   15000,
			timeInForce: types.OrderTimeInForceFOK,
			orderType:   types.OrderTypeMarket,
			status:      types.OrderStatusActive,
			createdAt:   blocks[0].VegaTime,
		},
		{
			id:          orderIDs[3],
			block:       blocks[2],
			party:       parties[3],
			market:      markets[0],
			side:        types.SideSell,
			price:       1005,
			size:        15000,
			remaining:   15000,
			timeInForce: types.OrderTimeInForceGTC,
			orderType:   types.OrderTypeLimit,
			status:      types.OrderStatusCancelled,
			createdAt:   blocks[0].VegaTime,
		},
	}

	orders := make([]entities.Order, len(testOrders))

	lastBlockTime := blocks[0].VegaTime
	for i, to := range testOrders {
		// It's important for order triggers that orders are inserted in order. The batcher in the
		// order store does not preserve insert order, so manually flush each block.
		if to.block.VegaTime != lastBlockTime {
			_, _ = os.Flush(ctx)
			lastBlockTime = to.block.VegaTime
		}
		ref := fmt.Sprintf("reference-%d", i)
		orders[i] = addTestOrder(t, os, to.id, to.block, to.party, to.market, ref, to.side,
			to.timeInForce, to.orderType, to.status, to.price, to.size, to.remaining, uint64(i), int32(1), nil, to.createdAt, defaultTxHash, nil)
	}

	return orders
}

func TestOrders_GetLiveOrders(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	bs := sqlstore.NewBlocks(connectionSource)
	ps := sqlstore.NewParties(connectionSource)
	ms := sqlstore.NewMarkets(connectionSource)
	os := sqlstore.NewOrders(connectionSource)

	// set up the blocks, parties and markets we need to generate the orders
	blocks := generateTestBlocks(t, ctx, 3, bs)
	parties := generateParties(t, ctx, 5, blocks[0], ps)
	markets := helpers.GenerateMarkets(t, ctx, 3, blocks[0], ms)
	orderIDs := generateOrderIDs(t, 8)
	testOrders := generateTestOrders(t, ctx, blocks, parties, markets, orderIDs, os)

	// Make sure we flush the batcher and write the orders to the database
	_, err := os.Flush(ctx)
	require.NoError(t, err)

	want := append(testOrders[:3], testOrders[4:6]...)
	got, err := os.GetLiveOrders(ctx)
	require.NoError(t, err)
	assert.Equal(t, 5, len(got))
	assert.ElementsMatch(t, want, got)
}

func TestOrders_CursorPagination(t *testing.T) {
	t.Run("Should return all current orders for a given market when no cursor is given - Newest First", testOrdersCursorPaginationByMarketNoCursorNewestFirst)
	t.Run("Should return all current orders for a given party when no cursor is given - Newest First", testOrdersCursorPaginationByPartyNoCursorNewestFirst)
	t.Run("Should return all versions of a given order ID when no cursor is given - Newest First", testOrdersCursorPaginationByOrderIDNoCursorNewestFirst)
	t.Run("Should return all current orders for a given party and market when no cursor is given - Newest First", testOrdersCursorPaginationByMarketAndPartyNoCursorNewestFirst)

	t.Run("Should return the first page of current orders for a given market when a first cursor is given - Newest First", testOrdersCursorPaginationByMarketFirstCursorNewestFirst)
	t.Run("Should return the first page of current orders for a given party when a first cursor is given - Newest First", testOrdersCursorPaginationByPartyFirstCursorNewestFirst)
	t.Run("Should return the first page of order versions of a given order ID when a first cursor is given - Newest First", testOrdersCursorPaginationByOrderIDFirstCursorNewestFirst)
	t.Run("Should return the first page of current orders for a given party and market when a first cursor is given - Newest First", testOrdersCursorPaginationByMarketAndPartyFirstCursorNewestFirst)

	t.Run("Should return the page of current orders for a given market when a first and after cursor is given - Newest First", testOrdersCursorPaginationByMarketFirstAndAfterCursorNewestFirst)
	t.Run("Should return the page of current orders for a given party when a first and after cursor is given - Newest First", testOrdersCursorPaginationByPartyFirstAndAfterCursorNewestFirst)
	t.Run("Should return the page of order versions of a given order ID when a first and after cursor is given - Newest First", testOrdersCursorPaginationByOrderIDFirstAndAfterCursorNewestFirst)
	t.Run("Should return the page of current orders for a given party and market when a first and after cursor is given - Newest First", testOrdersCursorPaginationByMarketAndPartyFirstAndAfterCursorNewestFirst)

	t.Run("Should return all current orders between dates for a given market when no cursor is given", testOrdersCursorPaginationBetweenDatesByMarketNoCursor)
	t.Run("Should return the first page of current orders between dates for a given market when a first cursor is given", testOrdersCursorPaginationBetweenDatesByMarketFirstCursor)
	t.Run("Should return the page of current orders for a given market when a first and after cursor is given", testOrdersCursorPaginationBetweenDatesByMarketFirstAndAfterCursor)
}

func TestOrdersFiltering(t *testing.T) {
	t.Run("Should filter orders", testOrdersFilter)
	t.Run("Should filter orders excluding liquidity orders", testOrdersFilterLiquidityOrders)
}

type orderTestStores struct {
	bs *sqlstore.Blocks
	ps *sqlstore.Parties
	ms *sqlstore.Markets
	os *sqlstore.Orders
}

type orderTestData struct {
	blocks  []entities.Block
	parties []entities.Party
	markets []entities.Market
	orders  []entities.Order
	cursors []*entities.Cursor
}

func setupOrderCursorPaginationTests(t *testing.T) *orderTestStores {
	t.Helper()
	stores := &orderTestStores{
		bs: sqlstore.NewBlocks(connectionSource),
		ps: sqlstore.NewParties(connectionSource),
		ms: sqlstore.NewMarkets(connectionSource),
		os: sqlstore.NewOrders(connectionSource),
	}

	return stores
}

func generateTestOrdersForCursorPagination(t *testing.T, ctx context.Context, stores *orderTestStores) orderTestData {
	t.Helper()
	blocks := generateTestBlocks(t, ctx, 12, stores.bs)
	parties := generateParties(t, ctx, 2, blocks[0], stores.ps)
	markets := helpers.GenerateMarkets(t, ctx, 2, blocks[0], stores.ms)
	orderIDs := generateOrderIDs(t, 20)

	// Order with multiple versions orderIDs[1]

	testOrders := []struct {
		id          entities.OrderID
		block       entities.Block
		party       entities.Party
		market      entities.Market
		reference   string
		side        types.Side
		price       int64
		size        int64
		remaining   int64
		version     int32
		timeInForce types.OrderTimeInForce
		orderType   types.OrderType
		status      types.OrderStatus
		cursor      *entities.Cursor
		lpID        []byte
		createdAt   time.Time
	}{
		{
			// testOrders[0]
			id:          orderIDs[0],
			block:       blocks[0],
			party:       parties[0],
			market:      markets[0],
			side:        types.SideBuy,
			price:       100,
			size:        100,
			remaining:   100,
			version:     1,
			timeInForce: types.OrderTimeInForceGTT,
			orderType:   types.OrderTypeLimit,
			status:      types.OrderStatusActive,
			createdAt:   blocks[0].VegaTime,
		},
		{
			// testOrders[1]
			id:          orderIDs[1],
			block:       blocks[0],
			party:       parties[1],
			market:      markets[0],
			side:        types.SideBuy,
			price:       100,
			size:        100,
			remaining:   100,
			version:     1,
			timeInForce: types.OrderTimeInForceGTC,
			orderType:   types.OrderTypeLimit,
			status:      types.OrderStatusActive,
			createdAt:   blocks[0].VegaTime,
		},
		{
			// testOrders[2]
			id:          orderIDs[1],
			block:       blocks[1],
			party:       parties[1],
			market:      markets[0],
			side:        types.SideBuy,
			price:       100,
			size:        100,
			remaining:   100,
			version:     2,
			timeInForce: types.OrderTimeInForceGTC,
			orderType:   types.OrderTypeLimit,
			status:      types.OrderStatusActive,
			createdAt:   blocks[0].VegaTime,
		},
		{
			// testOrders[3]
			id:          orderIDs[2],
			block:       blocks[2],
			party:       parties[1],
			market:      markets[0],
			side:        types.SideBuy,
			reference:   "DEADBEEF",
			price:       100,
			size:        100,
			remaining:   100,
			version:     1,
			timeInForce: types.OrderTimeInForceGTC,
			orderType:   types.OrderTypeLimit,
			status:      types.OrderStatusActive,
			createdAt:   blocks[2].VegaTime,
		},
		{
			// testOrders[4]
			id:          orderIDs[3],
			block:       blocks[2],
			party:       parties[0],
			market:      markets[1],
			side:        types.SideBuy,
			price:       100,
			size:        100,
			remaining:   100,
			version:     1,
			timeInForce: types.OrderTimeInForceGTC,
			orderType:   types.OrderTypeLimit,
			status:      types.OrderStatusActive,
			createdAt:   blocks[3].VegaTime,
		},
		{
			// testOrders[5]
			id:          orderIDs[4],
			block:       blocks[3],
			party:       parties[1],
			market:      markets[1],
			side:        types.SideBuy,
			price:       100,
			size:        100,
			remaining:   100,
			version:     1,
			timeInForce: types.OrderTimeInForceGTC,
			orderType:   types.OrderTypeMarket,
			status:      types.OrderStatusActive,
			createdAt:   blocks[3].VegaTime,
		},
		{
			// testOrders[6]
			id:          orderIDs[1],
			block:       blocks[4],
			party:       parties[1],
			market:      markets[0],
			side:        types.SideBuy,
			price:       100,
			size:        100,
			remaining:   100,
			version:     3,
			timeInForce: types.OrderTimeInForceGTC,
			orderType:   types.OrderTypeLimit,
			status:      types.OrderStatusActive,
			createdAt:   blocks[0].VegaTime,
		},
		{
			// testOrders[7]
			id:          orderIDs[5],
			block:       blocks[5],
			party:       parties[1],
			market:      markets[0],
			side:        types.SideBuy,
			price:       100,
			size:        100,
			remaining:   100,
			version:     1,
			timeInForce: types.OrderTimeInForceGTC,
			orderType:   types.OrderTypeMarket,
			status:      types.OrderStatusActive,
			createdAt:   blocks[5].VegaTime,
		},
		{
			// testOrders[8]
			id:          orderIDs[1],
			block:       blocks[5],
			party:       parties[1],
			market:      markets[0],
			side:        types.SideBuy,
			price:       100,
			size:        100,
			remaining:   100,
			version:     4,
			timeInForce: types.OrderTimeInForceGTC,
			orderType:   types.OrderTypeLimit,
			status:      types.OrderStatusActive,
			createdAt:   blocks[0].VegaTime,
		},
		{
			// testOrders[9]
			id:          orderIDs[6],
			block:       blocks[6],
			party:       parties[1],
			market:      markets[1],
			side:        types.SideBuy,
			price:       100,
			size:        100,
			remaining:   100,
			version:     1,
			timeInForce: types.OrderTimeInForceGTT,
			orderType:   types.OrderTypeLimit,
			status:      types.OrderStatusActive,
			createdAt:   blocks[6].VegaTime,
		},
		{
			// testOrders[10]
			id:          orderIDs[7],
			block:       blocks[7],
			party:       parties[0],
			market:      markets[1],
			side:        types.SideBuy,
			price:       100,
			size:        100,
			remaining:   100,
			version:     1,
			timeInForce: types.OrderTimeInForceGTC,
			orderType:   types.OrderTypeLimit,
			status:      types.OrderStatusActive,
			createdAt:   blocks[7].VegaTime,
		},
		{
			// testOrders[11]
			id:          orderIDs[1],
			block:       blocks[8],
			party:       parties[1],
			market:      markets[0],
			side:        types.SideBuy,
			price:       100,
			size:        100,
			remaining:   100,
			version:     5,
			timeInForce: types.OrderTimeInForceGTC,
			orderType:   types.OrderTypeLimit,
			status:      types.OrderStatusActive,
			createdAt:   blocks[0].VegaTime,
		},
		{
			// testOrders[12]
			id:          orderIDs[8],
			block:       blocks[8],
			party:       parties[1],
			market:      markets[1],
			side:        types.SideBuy,
			price:       100,
			size:        100,
			remaining:   100,
			version:     1,
			timeInForce: types.OrderTimeInForceGTC,
			orderType:   types.OrderTypeLimit,
			status:      types.OrderStatusActive,
			createdAt:   blocks[8].VegaTime,
		},
		{
			// testOrders[13] -- current OrderIDs[1]
			id:          orderIDs[1],
			block:       blocks[9],
			party:       parties[1],
			market:      markets[0],
			side:        types.SideBuy,
			price:       100,
			size:        100,
			remaining:   100,
			version:     6,
			timeInForce: types.OrderTimeInForceGTC,
			orderType:   types.OrderTypeLimit,
			status:      types.OrderStatusFilled,
			createdAt:   blocks[0].VegaTime,
		},
		{
			// testOrders[14]
			id:          orderIDs[9],
			block:       blocks[9],
			party:       parties[0],
			market:      markets[0],
			side:        types.SideBuy,
			price:       100,
			size:        100,
			remaining:   100,
			version:     1,
			timeInForce: types.OrderTimeInForceFOK,
			orderType:   types.OrderTypeLimit,
			status:      types.OrderStatusFilled,
			createdAt:   blocks[9].VegaTime,
		},
		{
			// testOrders[15]
			id:          orderIDs[10],
			block:       blocks[10],
			party:       parties[1],
			market:      markets[0],
			reference:   "DEADBEEF",
			side:        types.SideBuy,
			price:       100,
			size:        100,
			remaining:   100,
			version:     1,
			timeInForce: types.OrderTimeInForceGTC,
			orderType:   types.OrderTypeLimit,
			status:      types.OrderStatusActive,
			lpID:        []byte("TEST_LP1"),
			createdAt:   blocks[10].VegaTime,
		},
		{
			// testOrders[16]
			id:          orderIDs[11],
			block:       blocks[11],
			party:       parties[1],
			market:      markets[0],
			side:        types.SideBuy,
			price:       100,
			size:        100,
			remaining:   100,
			version:     1,
			timeInForce: types.OrderTimeInForceGTC,
			orderType:   types.OrderTypeLimit,
			status:      types.OrderStatusActive,
			lpID:        []byte("TEST_LP2"),
			createdAt:   blocks[11].VegaTime,
		},
	}

	orders := make([]entities.Order, len(testOrders))
	cursors := make([]*entities.Cursor, len(testOrders))

	lastBlockTime := testOrders[0].block.VegaTime
	for i, order := range testOrders {
		// It's important for order triggers that orders are inserted in order. The batcher in the
		// order store does not preserve insert order, so manually flush each block.
		if order.block.VegaTime != lastBlockTime {
			_, _ = stores.os.Flush(ctx)
			lastBlockTime = order.block.VegaTime
		}

		seqNum := uint64(i)
		orderCursor := entities.OrderCursor{
			CreatedAt: order.createdAt,
			ID:        order.id,
			VegaTime:  order.block.VegaTime,
		}
		cursors[i] = entities.NewCursor(orderCursor.String())
		orders[i] = addTestOrder(t, stores.os, order.id, order.block, order.party, order.market, order.reference, order.side, order.timeInForce,
			order.orderType, order.status, order.price, order.size, order.remaining, seqNum, order.version, order.lpID, order.createdAt, defaultTxHash, nil)
	}

	// Make sure we flush the batcher and write the orders to the database
	_, err := stores.os.Flush(ctx)
	require.NoError(t, err, "Could not insert test order data to the test database")

	return orderTestData{
		blocks:  blocks,
		parties: parties,
		markets: markets,
		orders:  orders,
		cursors: cursors,
	}
}

func sortOrders(orders []entities.Order) {
	f := func(i, j int) bool {
		if orders[i].CreatedAt == orders[j].CreatedAt {
			return orders[i].ID < orders[j].ID
		}

		return orders[i].CreatedAt.After(orders[j].CreatedAt)
	}

	sort.Slice(orders, f)
}

func testOrdersCursorPaginationByMarketNoCursorNewestFirst(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	stores := setupOrderCursorPaginationTests(t)
	testData := generateTestOrdersForCursorPagination(t, ctx, stores)

	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, true)
	require.NoError(t, err)

	filter := entities.OrderFilter{
		MarketIDs: []string{testData.markets[0].ID.String()},
	}
	got, pageInfo, err := stores.os.ListOrders(ctx, pagination, filter)
	require.NoError(t, err)
	assert.Len(t, got, 7)
	want := append([]entities.Order{},
		testData.orders[16],
		testData.orders[15],
		testData.orders[14],
		testData.orders[13],
		testData.orders[7],
		testData.orders[3],
		testData.orders[0],
	)
	sortOrders(want)

	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[len(want)-1].Cursor().Encode(),
	}, pageInfo)
}

func testOrdersCursorPaginationByPartyNoCursorNewestFirst(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	stores := setupOrderCursorPaginationTests(t)
	testData := generateTestOrdersForCursorPagination(t, ctx, stores)

	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, true)
	require.NoError(t, err)

	filter := entities.OrderFilter{
		PartyIDs: []string{testData.parties[1].ID.String()},
	}
	got, pageInfo, err := stores.os.ListOrders(ctx, pagination, filter)
	require.NoError(t, err)
	assert.Len(t, got, 8)
	want := append([]entities.Order{},
		testData.orders[16],
		testData.orders[15],
		testData.orders[13],
		testData.orders[12],
		testData.orders[9],
		testData.orders[7],
		testData.orders[5],
		testData.orders[3],
	)
	sortOrders(want)

	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[len(want)-1].Cursor().Encode(),
	}, pageInfo)
}

func testOrdersCursorPaginationByOrderIDNoCursorNewestFirst(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	stores := setupOrderCursorPaginationTests(t)
	testData := generateTestOrdersForCursorPagination(t, ctx, stores)

	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, true)
	require.NoError(t, err)

	orderID := testData.orders[1].ID
	got, pageInfo, err := stores.os.ListOrderVersions(ctx, orderID.String(), pagination)
	require.NoError(t, err)
	assert.Len(t, got, 6)
	want := append([]entities.Order{},
		testData.orders[13],
		testData.orders[11],
		testData.orders[8],
		testData.orders[6],
		testData.orders[2],
		testData.orders[1],
	)
	sortOrders(want)
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[len(want)-1].Cursor().Encode(),
	}, pageInfo)
}

func testOrdersCursorPaginationByMarketAndPartyNoCursorNewestFirst(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	stores := setupOrderCursorPaginationTests(t)
	testData := generateTestOrdersForCursorPagination(t, ctx, stores)

	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, true)
	require.NoError(t, err)

	filter := entities.OrderFilter{
		PartyIDs:  []string{testData.parties[1].ID.String()},
		MarketIDs: []string{testData.markets[1].ID.String()},
	}
	got, pageInfo, err := stores.os.ListOrders(ctx, pagination, filter)
	require.NoError(t, err)
	assert.Len(t, got, 3)
	want := append([]entities.Order{},
		testData.orders[12],
		testData.orders[9],
		testData.orders[5],
	)
	sortOrders(want)
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[len(want)-1].Cursor().Encode(),
	}, pageInfo)
}

// -- First Cursor Tests --

func testOrdersCursorPaginationByMarketFirstCursorNewestFirst(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	stores := setupOrderCursorPaginationTests(t)
	testData := generateTestOrdersForCursorPagination(t, ctx, stores)

	first := int32(5)
	pagination, err := entities.NewCursorPagination(&first, nil, nil, nil, true)
	require.NoError(t, err)

	filter := entities.OrderFilter{
		MarketIDs: []string{testData.markets[0].ID.String()},
	}
	got, pageInfo, err := stores.os.ListOrders(ctx, pagination, filter)
	require.NoError(t, err)
	assert.Len(t, got, 5)
	want := append([]entities.Order{},
		testData.orders[16],
		testData.orders[15],
		testData.orders[14],
		testData.orders[7],
		testData.orders[3],
	)
	sortOrders(want)
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: false,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[len(want)-1].Cursor().Encode(),
	}, pageInfo)
}

func testOrdersCursorPaginationByPartyFirstCursorNewestFirst(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	stores := setupOrderCursorPaginationTests(t)
	testData := generateTestOrdersForCursorPagination(t, ctx, stores)

	first := int32(5)
	pagination, err := entities.NewCursorPagination(&first, nil, nil, nil, true)
	require.NoError(t, err)

	filter := entities.OrderFilter{
		PartyIDs: []string{testData.parties[1].ID.String()},
	}
	got, pageInfo, err := stores.os.ListOrders(ctx, pagination, filter)
	require.NoError(t, err)
	assert.Len(t, got, 5)
	want := append([]entities.Order{},
		testData.orders[16],
		testData.orders[15],
		testData.orders[12],
		testData.orders[9],
		testData.orders[7],
	)
	sortOrders(want)

	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: false,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[len(want)-1].Cursor().Encode(),
	}, pageInfo)
}

func testOrdersCursorPaginationByOrderIDFirstCursorNewestFirst(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	stores := setupOrderCursorPaginationTests(t)
	testData := generateTestOrdersForCursorPagination(t, ctx, stores)

	first := int32(3)
	pagination, err := entities.NewCursorPagination(&first, nil, nil, nil, true)
	require.NoError(t, err)

	orderID := testData.orders[1].ID
	got, pageInfo, err := stores.os.ListOrderVersions(ctx, orderID.String(), pagination)
	require.NoError(t, err)
	assert.Len(t, got, 3)
	want := append([]entities.Order{},
		testData.orders[13],
		testData.orders[11],
		testData.orders[8],
	)
	sortOrders(want)
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: false,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[len(want)-1].Cursor().Encode(),
	}, pageInfo)
}

func testOrdersCursorPaginationByMarketAndPartyFirstCursorNewestFirst(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	stores := setupOrderCursorPaginationTests(t)
	testData := generateTestOrdersForCursorPagination(t, ctx, stores)

	first := int32(2)
	pagination, err := entities.NewCursorPagination(&first, nil, nil, nil, true)
	require.NoError(t, err)

	filter := entities.OrderFilter{
		PartyIDs:  []string{testData.parties[1].ID.String()},
		MarketIDs: []string{testData.markets[1].ID.String()},
	}
	got, pageInfo, err := stores.os.ListOrders(ctx, pagination, filter)
	require.NoError(t, err)
	assert.Len(t, got, 2)
	want := append([]entities.Order{},
		testData.orders[12],
		testData.orders[9],
	)
	sortOrders(want)
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: false,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[len(want)-1].Cursor().Encode(),
	}, pageInfo)
}

// -- First and After tests --

func testOrdersCursorPaginationByMarketFirstAndAfterCursorNewestFirst(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	stores := setupOrderCursorPaginationTests(t)
	testData := generateTestOrdersForCursorPagination(t, ctx, stores)

	first := int32(2)
	after := testData.cursors[14].Encode()
	pagination, err := entities.NewCursorPagination(&first, &after, nil, nil, true)
	require.NoError(t, err)

	filter := entities.OrderFilter{
		MarketIDs: []string{testData.markets[0].ID.String()},
	}
	got, pageInfo, err := stores.os.ListOrders(ctx, pagination, filter)
	require.NoError(t, err)
	assert.Len(t, got, 2)
	want := append([]entities.Order{},
		testData.orders[7],
		testData.orders[3],
	)
	sortOrders(want)
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[len(want)-1].Cursor().Encode(),
	}, pageInfo)
}

func testOrdersCursorPaginationByPartyFirstAndAfterCursorNewestFirst(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	stores := setupOrderCursorPaginationTests(t)
	testData := generateTestOrdersForCursorPagination(t, ctx, stores)

	first := int32(3)
	after := testData.cursors[12].Encode()
	pagination, err := entities.NewCursorPagination(&first, &after, nil, nil, true)
	require.NoError(t, err)

	partyID := testData.parties[1].ID.String()
	got, pageInfo, err := stores.os.ListOrders(ctx, pagination, entities.OrderFilter{
		PartyIDs: []string{partyID},
	})
	require.NoError(t, err)
	assert.Len(t, got, 3)
	want := append([]entities.Order{},
		testData.orders[9],
		testData.orders[7],
		testData.orders[5],
	)
	sortOrders(want)
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[len(want)-1].Cursor().Encode(),
	}, pageInfo)
}

func testOrdersCursorPaginationByOrderIDFirstAndAfterCursorNewestFirst(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	stores := setupOrderCursorPaginationTests(t)
	testData := generateTestOrdersForCursorPagination(t, ctx, stores)

	first := int32(3)
	after := testData.cursors[11].Encode()
	pagination, err := entities.NewCursorPagination(&first, &after, nil, nil, true)
	require.NoError(t, err)

	orderID := testData.orders[1].ID
	got, pageInfo, err := stores.os.ListOrderVersions(ctx, orderID.String(), pagination)
	require.NoError(t, err)
	assert.Len(t, got, 3)
	want := append([]entities.Order{},
		testData.orders[8],
		testData.orders[6],
		testData.orders[2],
	)
	// sortOrders(want)
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[len(want)-1].Cursor().Encode(),
	}, pageInfo)
}

func testOrdersCursorPaginationByMarketAndPartyFirstAndAfterCursorNewestFirst(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	stores := setupOrderCursorPaginationTests(t)
	testData := generateTestOrdersForCursorPagination(t, ctx, stores)

	first := int32(1)
	after := testData.cursors[12].Encode()
	pagination, err := entities.NewCursorPagination(&first, &after, nil, nil, true)
	require.NoError(t, err)

	got, pageInfo, err := stores.os.ListOrders(ctx, pagination, entities.OrderFilter{
		PartyIDs:  []string{testData.parties[1].ID.String()},
		MarketIDs: []string{testData.markets[1].ID.String()},
	})
	require.NoError(t, err)
	assert.Len(t, got, 1)
	want := append([]entities.Order{},
		testData.orders[9],
	)
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     testData.cursors[9].Encode(),
		EndCursor:       testData.cursors[9].Encode(),
	}, pageInfo)
}

// -- Last and Before tests --

func testOrdersCursorPaginationBetweenDatesByMarketNoCursor(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	stores := setupOrderCursorPaginationTests(t)
	testData := generateTestOrdersForCursorPagination(t, ctx, stores)

	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, true)
	require.NoError(t, err)

	filter := entities.OrderFilter{
		MarketIDs: []string{testData.markets[0].ID.String()},
		DateRange: &entities.DateRange{
			Start: &testData.orders[3].VegaTime,
			End:   &testData.orders[14].VegaTime,
		},
	}
	got, pageInfo, err := stores.os.ListOrders(ctx, pagination, filter)

	require.NoError(t, err)
	assert.Len(t, got, 2)
	want := append([]entities.Order{}, testData.orders[7], testData.orders[3]) // order[13] and order[14] have the same block time
	sortOrders(want)
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[len(want)-1].Cursor().Encode(),
	}, pageInfo)
}

func testOrdersCursorPaginationBetweenDatesByMarketFirstCursor(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	stores := setupOrderCursorPaginationTests(t)
	testData := generateTestOrdersForCursorPagination(t, ctx, stores)

	first := int32(3)
	pagination, err := entities.NewCursorPagination(&first, nil, nil, nil, true)
	require.NoError(t, err)

	filter := entities.OrderFilter{
		MarketIDs: []string{testData.markets[0].ID.String()},
		DateRange: &entities.DateRange{
			Start: &testData.orders[3].VegaTime,
			End:   &testData.orders[14].VegaTime,
		},
	}
	got, pageInfo, err := stores.os.ListOrders(ctx, pagination, filter)
	require.NoError(t, err)
	assert.Len(t, got, 2)
	want := append([]entities.Order{}, testData.orders[3], testData.orders[7])
	sortOrders(want)
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[len(want)-1].Cursor().Encode(),
	}, pageInfo)
}

func testOrdersCursorPaginationBetweenDatesByMarketFirstAndAfterCursor(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	stores := setupOrderCursorPaginationTests(t)
	testData := generateTestOrdersForCursorPagination(t, ctx, stores)

	first := int32(3)

	after := testData.cursors[14].Encode()
	pagination, err := entities.NewCursorPagination(&first, &after, nil, nil, true)
	require.NoError(t, err)

	filter := entities.OrderFilter{
		MarketIDs: []string{testData.markets[0].ID.String()},
		DateRange: &entities.DateRange{
			Start: &testData.blocks[3].VegaTime,
			End:   &testData.orders[16].VegaTime,
		},
	}
	got, pageInfo, err := stores.os.ListOrders(ctx, pagination, filter)
	require.NoError(t, err)
	assert.Len(t, got, 2)
	want := append([]entities.Order{}, testData.orders[13], testData.orders[7])
	sortOrders(want)
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: true,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[len(want)-1].Cursor().Encode(),
	}, pageInfo)
}

func testOrdersFilter(t *testing.T) {
	t.Run("Should filter orders by market and states", testOrdersFilterByMarketAndStates)
	t.Run("Should filter orders by party and states", testOrdersFilterByPartyAndStates)
	t.Run("Should filter orders by reference and states", testOrdersFilterByReferenceAndStates)
	t.Run("Should filter orders by market and types", testOrdersFilterByMarketAndTypes)
	t.Run("Should filter orders by party and types", testOrdersFilterByPartyAndTypes)
	t.Run("Should filter orders by reference and types", testOrdersFilterByReferenceAndTypes)
	t.Run("Should filter orders by market and time in force", testOrdersFilterByMarketAndTimeInForce)
	t.Run("Should filter orders by party and time in force", testOrdersFilterByPartyAndTimeInForce)
	t.Run("Should filter orders by reference and time in force", testOrdersFilterByReferenceAndTimeInForce)
	t.Run("Should filter by market, states and type", testOrdersFilterByMarketStatesAndTypes)
	t.Run("Should filter by party, states and type", testOrdersFilterByPartyStatesAndTypes)
	t.Run("Should filter by reference, states and type", testOrdersFilterByReferenceStatesAndTypes)
	t.Run("Should filter by market, states and time in force", testOrdersFilterByMarketStatesAndTimeInForce)
	t.Run("Should filter by party, states and time in force", testOrdersFilterByPartyStatesAndTimeInForce)
	t.Run("Should filter by reference, states and time in force", testOrdersFilterByReferenceStatesAndTimeInForce)
	t.Run("Should filter by market states, types and time in force", testOrdersFilterByMarketStatesTypesAndTimeInForce)
	t.Run("Should filter by party states, types and time in force", testOrdersFilterByPartyStatesTypesAndTimeInForce)
	t.Run("Should filter by reference states, types and time in force", testOrdersFilterByReferenceStatesTypesAndTimeInForce)
}

func testOrdersFilterByMarketAndStates(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	stores := setupOrderCursorPaginationTests(t)
	testData := generateTestOrdersForCursorPagination(t, ctx, stores)

	filter := entities.OrderFilter{
		MarketIDs:        []string{testData.markets[0].ID.String()},
		Statuses:         []vega.Order_Status{vega.Order_STATUS_ACTIVE, vega.Order_STATUS_PARTIALLY_FILLED},
		Types:            nil,
		TimeInForces:     nil,
		ExcludeLiquidity: false,
	}

	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, true)
	require.NoError(t, err)

	got, pageInfo, err := stores.os.ListOrders(ctx, pagination, filter)
	require.NoError(t, err)

	want := append(
		[]entities.Order{},
		testData.orders[0],
		testData.orders[3],
		testData.orders[7],
		testData.orders[15],
		testData.orders[16],
	)
	sortOrders(want)
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[len(want)-1].Cursor().Encode(),
	}, pageInfo)
}

func testOrdersFilterByPartyAndStates(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	stores := setupOrderCursorPaginationTests(t)
	testData := generateTestOrdersForCursorPagination(t, ctx, stores)

	filter := entities.OrderFilter{
		PartyIDs:         []string{testData.parties[1].ID.String()},
		Statuses:         []vega.Order_Status{vega.Order_STATUS_ACTIVE, vega.Order_STATUS_PARTIALLY_FILLED},
		Types:            nil,
		TimeInForces:     nil,
		ExcludeLiquidity: false,
	}

	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, true)
	require.NoError(t, err)

	got, pageInfo, err := stores.os.ListOrders(ctx, pagination, filter)
	require.NoError(t, err)

	want := append(
		[]entities.Order{},
		testData.orders[3],
		testData.orders[5],
		testData.orders[7],
		testData.orders[9],
		testData.orders[12],
		testData.orders[15],
		testData.orders[16],
	)
	sortOrders(want)
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[len(want)-1].Cursor().Encode(),
	}, pageInfo)
}

func testOrdersFilterByReferenceAndStates(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	stores := setupOrderCursorPaginationTests(t)
	testData := generateTestOrdersForCursorPagination(t, ctx, stores)

	filter := entities.OrderFilter{
		Statuses:         []vega.Order_Status{vega.Order_STATUS_ACTIVE, vega.Order_STATUS_PARTIALLY_FILLED},
		Types:            nil,
		TimeInForces:     nil,
		ExcludeLiquidity: false,
		Reference:        ptr.From("DEADBEEF"),
	}

	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, true)
	require.NoError(t, err)

	got, pageInfo, err := stores.os.ListOrders(ctx, pagination, filter)
	require.NoError(t, err)

	want := append(
		[]entities.Order{},
		testData.orders[3],
		testData.orders[15],
	)
	sortOrders(want)
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[len(want)-1].Cursor().Encode(),
	}, pageInfo)
}

func testOrdersFilterByMarketAndTypes(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	stores := setupOrderCursorPaginationTests(t)
	testData := generateTestOrdersForCursorPagination(t, ctx, stores)

	filter := entities.OrderFilter{
		MarketIDs:        []string{testData.markets[0].ID.String()},
		Statuses:         nil,
		Types:            []vega.Order_Type{vega.Order_TYPE_LIMIT},
		TimeInForces:     nil,
		ExcludeLiquidity: false,
	}

	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, true)
	require.NoError(t, err)

	got, pageInfo, err := stores.os.ListOrders(ctx, pagination, filter)
	require.NoError(t, err)

	want := append([]entities.Order{},
		testData.orders[0],
		testData.orders[3],
		testData.orders[13],
		testData.orders[14],
		testData.orders[15],
		testData.orders[16],
	)
	sortOrders(want)

	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[len(want)-1].Cursor().Encode(),
	}, pageInfo)
}

func testOrdersFilterByPartyAndTypes(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	stores := setupOrderCursorPaginationTests(t)
	testData := generateTestOrdersForCursorPagination(t, ctx, stores)

	filter := entities.OrderFilter{
		PartyIDs:         []string{testData.parties[1].ID.String()},
		Statuses:         nil,
		Types:            []vega.Order_Type{vega.Order_TYPE_LIMIT},
		TimeInForces:     nil,
		ExcludeLiquidity: false,
	}

	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, true)
	require.NoError(t, err)

	got, pageInfo, err := stores.os.ListOrders(ctx, pagination, filter)
	require.NoError(t, err)

	want := append(
		[]entities.Order{},
		testData.orders[3],
		testData.orders[9],
		testData.orders[12],
		testData.orders[13],
		testData.orders[15],
		testData.orders[16],
	)
	sortOrders(want)
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[len(want)-1].Cursor().Encode(),
	}, pageInfo)
}

func testOrdersFilterByReferenceAndTypes(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	stores := setupOrderCursorPaginationTests(t)
	testData := generateTestOrdersForCursorPagination(t, ctx, stores)

	filter := entities.OrderFilter{
		Statuses:         nil,
		Types:            []vega.Order_Type{vega.Order_TYPE_LIMIT},
		TimeInForces:     nil,
		ExcludeLiquidity: false,
		Reference:        ptr.From("DEADBEEF"),
	}

	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, true)
	require.NoError(t, err)

	got, pageInfo, err := stores.os.ListOrders(ctx, pagination, filter)
	require.NoError(t, err)

	want := append(
		[]entities.Order{},
		testData.orders[3],
		testData.orders[15],
	)
	sortOrders(want)
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[len(want)-1].Cursor().Encode(),
	}, pageInfo)
}

func testOrdersFilterByMarketAndTimeInForce(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	stores := setupOrderCursorPaginationTests(t)
	testData := generateTestOrdersForCursorPagination(t, ctx, stores)

	filter := entities.OrderFilter{
		MarketIDs:        []string{testData.markets[0].ID.String()},
		Statuses:         nil,
		Types:            nil,
		TimeInForces:     []vega.Order_TimeInForce{vega.Order_TIME_IN_FORCE_GTC},
		ExcludeLiquidity: false,
	}

	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, true)
	require.NoError(t, err)

	got, pageInfo, err := stores.os.ListOrders(ctx, pagination, filter)
	require.NoError(t, err)

	want := append(
		[]entities.Order{},
		testData.orders[3],
		testData.orders[7],
		testData.orders[13],
		testData.orders[15],
		testData.orders[16],
	)
	sortOrders(want)
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[len(want)-1].Cursor().Encode(),
	}, pageInfo)
}

func testOrdersFilterByPartyAndTimeInForce(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	stores := setupOrderCursorPaginationTests(t)
	testData := generateTestOrdersForCursorPagination(t, ctx, stores)

	filter := entities.OrderFilter{
		PartyIDs:         []string{testData.parties[1].ID.String()},
		Statuses:         nil,
		Types:            nil,
		TimeInForces:     []vega.Order_TimeInForce{vega.Order_TIME_IN_FORCE_GTC},
		ExcludeLiquidity: false,
	}

	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, true)
	require.NoError(t, err)

	got, pageInfo, err := stores.os.ListOrders(ctx, pagination, filter)
	require.NoError(t, err)

	want := append(
		[]entities.Order{},
		testData.orders[3],
		testData.orders[5],
		testData.orders[7],
		testData.orders[12],
		testData.orders[13],
		testData.orders[15],
		testData.orders[16],
	)
	sortOrders(want)
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[len(want)-1].Cursor().Encode(),
	}, pageInfo)
}

func testOrdersFilterByReferenceAndTimeInForce(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	stores := setupOrderCursorPaginationTests(t)
	testData := generateTestOrdersForCursorPagination(t, ctx, stores)

	filter := entities.OrderFilter{
		Statuses:         nil,
		Types:            nil,
		TimeInForces:     []vega.Order_TimeInForce{vega.Order_TIME_IN_FORCE_GTC},
		ExcludeLiquidity: false,
		Reference:        ptr.From("DEADBEEF"),
	}

	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, true)
	require.NoError(t, err)

	got, pageInfo, err := stores.os.ListOrders(ctx, pagination, filter)
	require.NoError(t, err)

	want := append(
		[]entities.Order{},
		testData.orders[3],
		testData.orders[15],
	)
	sortOrders(want)
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[len(want)-1].Cursor().Encode(),
	}, pageInfo)
}

func testOrdersFilterByMarketStatesAndTypes(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	stores := setupOrderCursorPaginationTests(t)
	testData := generateTestOrdersForCursorPagination(t, ctx, stores)

	filter := entities.OrderFilter{
		MarketIDs:        []string{testData.markets[0].ID.String()},
		Statuses:         []vega.Order_Status{vega.Order_STATUS_ACTIVE, vega.Order_STATUS_PARTIALLY_FILLED},
		Types:            []vega.Order_Type{vega.Order_TYPE_LIMIT},
		TimeInForces:     nil,
		ExcludeLiquidity: false,
	}

	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, true)
	require.NoError(t, err)

	got, pageInfo, err := stores.os.ListOrders(ctx, pagination, filter)
	require.NoError(t, err)

	want := append(
		[]entities.Order{},
		testData.orders[0],
		testData.orders[3],
		testData.orders[15],
		testData.orders[16],
	)
	sortOrders(want)
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[len(want)-1].Cursor().Encode(),
	}, pageInfo)
}

func testOrdersFilterByPartyStatesAndTypes(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	stores := setupOrderCursorPaginationTests(t)
	testData := generateTestOrdersForCursorPagination(t, ctx, stores)

	filter := entities.OrderFilter{
		PartyIDs:         []string{testData.parties[1].ID.String()},
		Statuses:         []vega.Order_Status{vega.Order_STATUS_ACTIVE, vega.Order_STATUS_PARTIALLY_FILLED},
		Types:            []vega.Order_Type{vega.Order_TYPE_LIMIT},
		TimeInForces:     nil,
		ExcludeLiquidity: false,
	}

	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, true)
	require.NoError(t, err)

	got, pageInfo, err := stores.os.ListOrders(ctx, pagination, filter)
	require.NoError(t, err)

	want := append(
		[]entities.Order{},
		testData.orders[3],
		testData.orders[9],
		testData.orders[12],
		testData.orders[15],
		testData.orders[16],
	)
	sortOrders(want)
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[len(want)-1].Cursor().Encode(),
	}, pageInfo)
}

func testOrdersFilterByReferenceStatesAndTypes(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	stores := setupOrderCursorPaginationTests(t)
	testData := generateTestOrdersForCursorPagination(t, ctx, stores)

	filter := entities.OrderFilter{
		Statuses:         []vega.Order_Status{vega.Order_STATUS_ACTIVE, vega.Order_STATUS_PARTIALLY_FILLED},
		Types:            []vega.Order_Type{vega.Order_TYPE_LIMIT},
		TimeInForces:     nil,
		ExcludeLiquidity: false,
		Reference:        ptr.From("DEADBEEF"),
	}

	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, true)
	require.NoError(t, err)

	got, pageInfo, err := stores.os.ListOrders(ctx, pagination, filter)
	require.NoError(t, err)

	want := append([]entities.Order{},
		testData.orders[3],
		testData.orders[15],
	)
	sortOrders(want)
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[len(want)-1].Cursor().Encode(),
	}, pageInfo)
}

func testOrdersFilterByMarketStatesAndTimeInForce(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	stores := setupOrderCursorPaginationTests(t)
	testData := generateTestOrdersForCursorPagination(t, ctx, stores)

	filter := entities.OrderFilter{
		MarketIDs:        []string{testData.markets[0].ID.String()},
		Statuses:         []vega.Order_Status{vega.Order_STATUS_ACTIVE, vega.Order_STATUS_PARTIALLY_FILLED},
		Types:            nil,
		TimeInForces:     []vega.Order_TimeInForce{vega.Order_TIME_IN_FORCE_GTC},
		ExcludeLiquidity: false,
	}

	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, true)
	require.NoError(t, err)

	got, pageInfo, err := stores.os.ListOrders(ctx, pagination, filter)
	require.NoError(t, err)

	want := append(
		[]entities.Order{},
		testData.orders[3],
		testData.orders[7],
		testData.orders[15],
		testData.orders[16],
	)
	sortOrders(want)
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[len(want)-1].Cursor().Encode(),
	}, pageInfo)
}

func testOrdersFilterByPartyStatesAndTimeInForce(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	stores := setupOrderCursorPaginationTests(t)
	testData := generateTestOrdersForCursorPagination(t, ctx, stores)

	filter := entities.OrderFilter{
		PartyIDs:         []string{testData.parties[1].ID.String()},
		Statuses:         []vega.Order_Status{vega.Order_STATUS_ACTIVE, vega.Order_STATUS_PARTIALLY_FILLED},
		Types:            nil,
		TimeInForces:     []vega.Order_TimeInForce{vega.Order_TIME_IN_FORCE_GTC},
		ExcludeLiquidity: false,
	}

	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, true)
	require.NoError(t, err)

	got, pageInfo, err := stores.os.ListOrders(ctx, pagination, filter)
	require.NoError(t, err)

	want := append(
		[]entities.Order{},
		testData.orders[3],
		testData.orders[5],
		testData.orders[7],
		testData.orders[12],
		testData.orders[15],
		testData.orders[16],
	)
	sortOrders(want)
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[len(want)-1].Cursor().Encode(),
	}, pageInfo)
}

func testOrdersFilterByReferenceStatesAndTimeInForce(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	stores := setupOrderCursorPaginationTests(t)
	testData := generateTestOrdersForCursorPagination(t, ctx, stores)

	filter := entities.OrderFilter{
		Statuses:         []vega.Order_Status{vega.Order_STATUS_ACTIVE, vega.Order_STATUS_PARTIALLY_FILLED},
		Types:            nil,
		TimeInForces:     []vega.Order_TimeInForce{vega.Order_TIME_IN_FORCE_GTC},
		ExcludeLiquidity: false,
		Reference:        ptr.From("DEADBEEF"),
	}

	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, true)
	require.NoError(t, err)

	got, pageInfo, err := stores.os.ListOrders(ctx, pagination, filter)
	require.NoError(t, err)

	want := append([]entities.Order{},
		testData.orders[3],
		testData.orders[15],
	)
	sortOrders(want)
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[len(want)-1].Cursor().Encode(),
	}, pageInfo)
}

func testOrdersFilterByMarketStatesTypesAndTimeInForce(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	stores := setupOrderCursorPaginationTests(t)
	testData := generateTestOrdersForCursorPagination(t, ctx, stores)

	filter := entities.OrderFilter{
		MarketIDs:        []string{testData.markets[0].ID.String()},
		Statuses:         []vega.Order_Status{vega.Order_STATUS_ACTIVE, vega.Order_STATUS_PARTIALLY_FILLED},
		Types:            []vega.Order_Type{vega.Order_TYPE_LIMIT},
		TimeInForces:     []vega.Order_TimeInForce{vega.Order_TIME_IN_FORCE_GTC},
		ExcludeLiquidity: false,
	}

	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, true)
	require.NoError(t, err)

	got, pageInfo, err := stores.os.ListOrders(ctx, pagination, filter)
	require.NoError(t, err)

	want := append(
		[]entities.Order{},
		testData.orders[3],
		testData.orders[15],
		testData.orders[16],
	)
	sortOrders(want)
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[len(want)-1].Cursor().Encode(),
	}, pageInfo)
}

func testOrdersFilterByPartyStatesTypesAndTimeInForce(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	stores := setupOrderCursorPaginationTests(t)
	testData := generateTestOrdersForCursorPagination(t, ctx, stores)

	filter := entities.OrderFilter{
		PartyIDs:         []string{testData.parties[1].ID.String()},
		Statuses:         []vega.Order_Status{vega.Order_STATUS_ACTIVE, vega.Order_STATUS_PARTIALLY_FILLED},
		Types:            []vega.Order_Type{vega.Order_TYPE_LIMIT},
		TimeInForces:     []vega.Order_TimeInForce{vega.Order_TIME_IN_FORCE_GTC},
		ExcludeLiquidity: false,
	}

	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, true)
	require.NoError(t, err)

	got, pageInfo, err := stores.os.ListOrders(ctx, pagination, filter)
	require.NoError(t, err)

	want := append(
		[]entities.Order{},
		testData.orders[3],
		testData.orders[12],
		testData.orders[15],
		testData.orders[16],
	)
	sortOrders(want)
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[len(want)-1].Cursor().Encode(),
	}, pageInfo)
}

func testOrdersFilterByReferenceStatesTypesAndTimeInForce(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	stores := setupOrderCursorPaginationTests(t)
	testData := generateTestOrdersForCursorPagination(t, ctx, stores)

	filter := entities.OrderFilter{
		Statuses:         []vega.Order_Status{vega.Order_STATUS_ACTIVE, vega.Order_STATUS_PARTIALLY_FILLED},
		Types:            []vega.Order_Type{vega.Order_TYPE_LIMIT},
		TimeInForces:     []vega.Order_TimeInForce{vega.Order_TIME_IN_FORCE_GTC},
		ExcludeLiquidity: false,
		Reference:        ptr.From("DEADBEEF"),
	}

	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, true)
	require.NoError(t, err)

	got, pageInfo, err := stores.os.ListOrders(ctx, pagination, filter)
	require.NoError(t, err)

	want := append([]entities.Order{},
		testData.orders[3],
		testData.orders[15],
	)
	sortOrders(want)
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[len(want)-1].Cursor().Encode(),
	}, pageInfo)
}

func testOrdersFilterLiquidityOrders(t *testing.T) {
	t.Run("Should filter orders by market and states", testOrdersFilterExcludeLiquidityByMarketAndStates)
	t.Run("Should filter orders by party and states", testOrdersFilterExcludeLiquidityByPartyAndStates)
	t.Run("Should filter orders by reference and states", testOrdersFilterExcludeLiquidityByReferenceAndStates)
	t.Run("Should filter orders by market and types", testOrdersFilterExcludeLiquidityByMarketAndTypes)
	t.Run("Should filter orders by party and types", testOrdersFilterExcludeLiquidityByPartyAndTypes)
	t.Run("Should filter orders by reference and types", testOrdersFilterExcludeLiquidityByReferenceAndTypes)
	t.Run("Should filter orders by market and time in force", testOrdersFilterExcludeLiquidityByMarketAndTimeInForce)
	t.Run("Should filter orders by party and time in force", testOrdersFilterExcludeLiquidityByPartyAndTimeInForce)
	t.Run("Should filter orders by reference and time in force", testOrdersFilterExcludeLiquidityByReferenceAndTimeInForce)
	t.Run("Should filter by market, states and type", testOrdersFilterExcludeLiquidityByMarketStatesAndTypes)
	t.Run("Should filter by party, states and type", testOrdersFilterExcludeLiquidityByPartyStatesAndTypes)
	t.Run("Should filter by reference, states and type", testOrdersFilterExcludeLiquidityByReferenceStatesAndTypes)
	t.Run("Should filter by market, states and time in force", testOrdersFilterExcludeLiquidityByMarketStatesAndTimeInForce)
	t.Run("Should filter by party, states and time in force", testOrdersFilterExcludeLiquidityByPartyStatesAndTimeInForce)
	t.Run("Should filter by reference, states and time in force", testOrdersFilterExcludeLiquidityByReferenceStatesAndTimeInForce)
	t.Run("Should filter by market states, types and time in force", testOrdersFilterExcludeLiquidityByMarketStatesTypesAndTimeInForce)
	t.Run("Should filter by party states, types and time in force", testOrdersFilterExcludeLiquidityByPartyStatesTypesAndTimeInForce)
	t.Run("Should filter by reference states, types and time in force", testOrdersFilterExcludeLiquidityByReferenceStatesTypesAndTimeInForce)
}

func testOrdersFilterExcludeLiquidityByMarketAndStates(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	stores := setupOrderCursorPaginationTests(t)
	testData := generateTestOrdersForCursorPagination(t, ctx, stores)

	filter := entities.OrderFilter{
		MarketIDs:        []string{testData.markets[0].ID.String()},
		Statuses:         []vega.Order_Status{vega.Order_STATUS_ACTIVE, vega.Order_STATUS_PARTIALLY_FILLED},
		Types:            nil,
		TimeInForces:     nil,
		ExcludeLiquidity: true,
	}

	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, true)
	require.NoError(t, err)

	got, pageInfo, err := stores.os.ListOrders(ctx, pagination, filter)
	require.NoError(t, err)

	want := append(
		[]entities.Order{},
		testData.orders[0],
		testData.orders[3],
		testData.orders[7],
	)
	sortOrders(want)
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[len(want)-1].Cursor().Encode(),
	}, pageInfo)
}

func testOrdersFilterExcludeLiquidityByPartyAndStates(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	stores := setupOrderCursorPaginationTests(t)
	testData := generateTestOrdersForCursorPagination(t, ctx, stores)

	filter := entities.OrderFilter{
		PartyIDs:         []string{testData.parties[1].ID.String()},
		Statuses:         []vega.Order_Status{vega.Order_STATUS_ACTIVE, vega.Order_STATUS_PARTIALLY_FILLED},
		Types:            nil,
		TimeInForces:     nil,
		ExcludeLiquidity: true,
	}

	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, true)
	require.NoError(t, err)

	got, pageInfo, err := stores.os.ListOrders(ctx, pagination, filter)
	require.NoError(t, err)

	want := append(
		[]entities.Order{},
		testData.orders[3],
		testData.orders[5],
		testData.orders[7],
		testData.orders[9],
		testData.orders[12],
	)
	sortOrders(want)
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[len(want)-1].Cursor().Encode(),
	}, pageInfo)
}

func testOrdersFilterExcludeLiquidityByReferenceAndStates(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	stores := setupOrderCursorPaginationTests(t)
	testData := generateTestOrdersForCursorPagination(t, ctx, stores)

	filter := entities.OrderFilter{
		Statuses:         []vega.Order_Status{vega.Order_STATUS_ACTIVE, vega.Order_STATUS_PARTIALLY_FILLED},
		Types:            nil,
		TimeInForces:     nil,
		ExcludeLiquidity: true,
		Reference:        ptr.From("DEADBEEF"),
	}

	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, true)
	require.NoError(t, err)

	got, pageInfo, err := stores.os.ListOrders(ctx, pagination, filter)
	require.NoError(t, err)

	want := append([]entities.Order{}, testData.orders[3])
	sortOrders(want)
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[len(want)-1].Cursor().Encode(),
	}, pageInfo)
}

func testOrdersFilterExcludeLiquidityByMarketAndTypes(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	stores := setupOrderCursorPaginationTests(t)
	testData := generateTestOrdersForCursorPagination(t, ctx, stores)

	filter := entities.OrderFilter{
		MarketIDs:        []string{testData.markets[0].ID.String()},
		Statuses:         nil,
		Types:            []vega.Order_Type{vega.Order_TYPE_LIMIT},
		TimeInForces:     nil,
		ExcludeLiquidity: true,
	}

	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, true)
	require.NoError(t, err)

	got, pageInfo, err := stores.os.ListOrders(ctx, pagination, filter)
	require.NoError(t, err)

	want := append([]entities.Order{},
		testData.orders[0],
		testData.orders[3],
		testData.orders[13],
		testData.orders[14],
	)
	sortOrders(want)
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[len(want)-1].Cursor().Encode(),
	}, pageInfo)
}

func testOrdersFilterExcludeLiquidityByPartyAndTypes(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	stores := setupOrderCursorPaginationTests(t)
	testData := generateTestOrdersForCursorPagination(t, ctx, stores)

	filter := entities.OrderFilter{
		PartyIDs:         []string{testData.parties[1].ID.String()},
		Statuses:         nil,
		Types:            []vega.Order_Type{vega.Order_TYPE_LIMIT},
		TimeInForces:     nil,
		ExcludeLiquidity: true,
	}

	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, true)
	require.NoError(t, err)

	got, pageInfo, err := stores.os.ListOrders(ctx, pagination, filter)
	require.NoError(t, err)

	want := append(
		[]entities.Order{},
		testData.orders[3],
		testData.orders[9],
		testData.orders[12],
		testData.orders[13],
	)
	sortOrders(want)
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[len(want)-1].Cursor().Encode(),
	}, pageInfo)
}

func testOrdersFilterExcludeLiquidityByReferenceAndTypes(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	stores := setupOrderCursorPaginationTests(t)
	testData := generateTestOrdersForCursorPagination(t, ctx, stores)

	filter := entities.OrderFilter{
		Statuses:         nil,
		Types:            []vega.Order_Type{vega.Order_TYPE_LIMIT},
		TimeInForces:     nil,
		ExcludeLiquidity: true,
		Reference:        ptr.From("DEADBEEF"),
	}

	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, true)
	require.NoError(t, err)

	got, pageInfo, err := stores.os.ListOrders(ctx, pagination, filter)
	require.NoError(t, err)

	want := append(
		[]entities.Order{},
		testData.orders[3],
	)
	sortOrders(want)
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[len(want)-1].Cursor().Encode(),
	}, pageInfo)
}

func testOrdersFilterExcludeLiquidityByMarketAndTimeInForce(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	stores := setupOrderCursorPaginationTests(t)
	testData := generateTestOrdersForCursorPagination(t, ctx, stores)

	filter := entities.OrderFilter{
		MarketIDs:        []string{testData.markets[0].ID.String()},
		Statuses:         nil,
		Types:            nil,
		TimeInForces:     []vega.Order_TimeInForce{vega.Order_TIME_IN_FORCE_GTC},
		ExcludeLiquidity: true,
	}

	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, true)
	require.NoError(t, err)

	got, pageInfo, err := stores.os.ListOrders(ctx, pagination, filter)
	require.NoError(t, err)

	want := append(
		[]entities.Order{},
		testData.orders[3],
		testData.orders[7],
		testData.orders[13],
	)
	sortOrders(want)
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[len(want)-1].Cursor().Encode(),
	}, pageInfo)
}

func testOrdersFilterExcludeLiquidityByPartyAndTimeInForce(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	stores := setupOrderCursorPaginationTests(t)
	testData := generateTestOrdersForCursorPagination(t, ctx, stores)

	filter := entities.OrderFilter{
		PartyIDs:         []string{testData.parties[1].ID.String()},
		Statuses:         nil,
		Types:            nil,
		TimeInForces:     []vega.Order_TimeInForce{vega.Order_TIME_IN_FORCE_GTC},
		ExcludeLiquidity: true,
	}

	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, true)
	require.NoError(t, err)

	got, pageInfo, err := stores.os.ListOrders(ctx, pagination, filter)
	require.NoError(t, err)

	want := append(
		[]entities.Order{},
		testData.orders[3],
		testData.orders[5],
		testData.orders[7],
		testData.orders[12],
		testData.orders[13],
	)
	sortOrders(want)
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[len(want)-1].Cursor().Encode(),
	}, pageInfo)
}

func testOrdersFilterExcludeLiquidityByReferenceAndTimeInForce(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	stores := setupOrderCursorPaginationTests(t)
	testData := generateTestOrdersForCursorPagination(t, ctx, stores)

	filter := entities.OrderFilter{
		Statuses:         nil,
		Types:            nil,
		TimeInForces:     []vega.Order_TimeInForce{vega.Order_TIME_IN_FORCE_GTC},
		ExcludeLiquidity: true,
		Reference:        ptr.From("DEADBEEF"),
	}

	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, true)
	require.NoError(t, err)

	got, pageInfo, err := stores.os.ListOrders(ctx, pagination, filter)
	require.NoError(t, err)

	want := append(
		[]entities.Order{},
		testData.orders[3],
	)
	sortOrders(want)
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[len(want)-1].Cursor().Encode(),
	}, pageInfo)
}

func testOrdersFilterExcludeLiquidityByMarketStatesAndTypes(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	stores := setupOrderCursorPaginationTests(t)
	testData := generateTestOrdersForCursorPagination(t, ctx, stores)

	filter := entities.OrderFilter{
		MarketIDs:        []string{testData.markets[0].ID.String()},
		Statuses:         []vega.Order_Status{vega.Order_STATUS_ACTIVE, vega.Order_STATUS_PARTIALLY_FILLED},
		Types:            []vega.Order_Type{vega.Order_TYPE_LIMIT},
		TimeInForces:     nil,
		ExcludeLiquidity: true,
	}

	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, true)
	require.NoError(t, err)

	got, pageInfo, err := stores.os.ListOrders(ctx, pagination, filter)
	require.NoError(t, err)

	want := append(
		[]entities.Order{},
		testData.orders[0],
		testData.orders[3],
	)
	sortOrders(want)
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[len(want)-1].Cursor().Encode(),
	}, pageInfo)
}

func testOrdersFilterExcludeLiquidityByPartyStatesAndTypes(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	stores := setupOrderCursorPaginationTests(t)
	testData := generateTestOrdersForCursorPagination(t, ctx, stores)

	filter := entities.OrderFilter{
		PartyIDs:         []string{testData.parties[1].ID.String()},
		Statuses:         []vega.Order_Status{vega.Order_STATUS_ACTIVE, vega.Order_STATUS_PARTIALLY_FILLED},
		Types:            []vega.Order_Type{vega.Order_TYPE_LIMIT},
		TimeInForces:     nil,
		ExcludeLiquidity: true,
	}

	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, true)
	require.NoError(t, err)

	got, pageInfo, err := stores.os.ListOrders(ctx, pagination, filter)
	require.NoError(t, err)

	want := append(
		[]entities.Order{},
		testData.orders[3],
		testData.orders[9],
		testData.orders[12],
	)
	sortOrders(want)
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[len(want)-1].Cursor().Encode(),
	}, pageInfo)
}

func testOrdersFilterExcludeLiquidityByReferenceStatesAndTypes(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	stores := setupOrderCursorPaginationTests(t)
	testData := generateTestOrdersForCursorPagination(t, ctx, stores)

	filter := entities.OrderFilter{
		Statuses:         []vega.Order_Status{vega.Order_STATUS_ACTIVE, vega.Order_STATUS_PARTIALLY_FILLED},
		Types:            []vega.Order_Type{vega.Order_TYPE_LIMIT},
		TimeInForces:     nil,
		ExcludeLiquidity: true,
		Reference:        ptr.From("DEADBEEF"),
	}

	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, true)
	require.NoError(t, err)

	got, pageInfo, err := stores.os.ListOrders(ctx, pagination, filter)
	require.NoError(t, err)

	want := append([]entities.Order{},
		testData.orders[3],
	)
	sortOrders(want)
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[len(want)-1].Cursor().Encode(),
	}, pageInfo)
}

func testOrdersFilterExcludeLiquidityByMarketStatesAndTimeInForce(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	stores := setupOrderCursorPaginationTests(t)
	testData := generateTestOrdersForCursorPagination(t, ctx, stores)

	filter := entities.OrderFilter{
		MarketIDs:        []string{testData.markets[0].ID.String()},
		Statuses:         []vega.Order_Status{vega.Order_STATUS_ACTIVE, vega.Order_STATUS_PARTIALLY_FILLED},
		Types:            nil,
		TimeInForces:     []vega.Order_TimeInForce{vega.Order_TIME_IN_FORCE_GTC},
		ExcludeLiquidity: true,
	}

	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, true)
	require.NoError(t, err)

	got, pageInfo, err := stores.os.ListOrders(ctx, pagination, filter)
	require.NoError(t, err)

	want := append(
		[]entities.Order{},
		testData.orders[3],
		testData.orders[7],
	)
	sortOrders(want)
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[len(want)-1].Cursor().Encode(),
	}, pageInfo)
}

func testOrdersFilterExcludeLiquidityByPartyStatesAndTimeInForce(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	stores := setupOrderCursorPaginationTests(t)
	testData := generateTestOrdersForCursorPagination(t, ctx, stores)

	filter := entities.OrderFilter{
		PartyIDs:         []string{testData.parties[1].ID.String()},
		Statuses:         []vega.Order_Status{vega.Order_STATUS_ACTIVE, vega.Order_STATUS_PARTIALLY_FILLED},
		Types:            nil,
		TimeInForces:     []vega.Order_TimeInForce{vega.Order_TIME_IN_FORCE_GTC},
		ExcludeLiquidity: true,
	}

	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, true)
	require.NoError(t, err)

	got, pageInfo, err := stores.os.ListOrders(ctx, pagination, filter)
	require.NoError(t, err)

	want := append(
		[]entities.Order{},
		testData.orders[3],
		testData.orders[5],
		testData.orders[7],
		testData.orders[12],
	)
	sortOrders(want)
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[len(want)-1].Cursor().Encode(),
	}, pageInfo)
}

func testOrdersFilterExcludeLiquidityByReferenceStatesAndTimeInForce(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	stores := setupOrderCursorPaginationTests(t)
	testData := generateTestOrdersForCursorPagination(t, ctx, stores)

	filter := entities.OrderFilter{
		Statuses:         []vega.Order_Status{vega.Order_STATUS_ACTIVE, vega.Order_STATUS_PARTIALLY_FILLED},
		Types:            nil,
		TimeInForces:     []vega.Order_TimeInForce{vega.Order_TIME_IN_FORCE_GTC},
		ExcludeLiquidity: true,
		Reference:        ptr.From("DEADBEEF"),
	}

	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, true)
	require.NoError(t, err)

	got, pageInfo, err := stores.os.ListOrders(ctx, pagination, filter)
	require.NoError(t, err)

	want := append([]entities.Order{}, testData.orders[3])
	sortOrders(want)
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[len(want)-1].Cursor().Encode(),
	}, pageInfo)
}

func testOrdersFilterExcludeLiquidityByMarketStatesTypesAndTimeInForce(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	stores := setupOrderCursorPaginationTests(t)
	testData := generateTestOrdersForCursorPagination(t, ctx, stores)

	filter := entities.OrderFilter{
		MarketIDs:        []string{testData.markets[0].ID.String()},
		Statuses:         []vega.Order_Status{vega.Order_STATUS_ACTIVE, vega.Order_STATUS_PARTIALLY_FILLED},
		Types:            []vega.Order_Type{vega.Order_TYPE_LIMIT},
		TimeInForces:     []vega.Order_TimeInForce{vega.Order_TIME_IN_FORCE_GTC},
		ExcludeLiquidity: true,
	}

	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, true)
	require.NoError(t, err)

	got, pageInfo, err := stores.os.ListOrders(ctx, pagination, filter)
	require.NoError(t, err)

	want := append(
		[]entities.Order{},
		testData.orders[3],
	)
	sortOrders(want)
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[len(want)-1].Cursor().Encode(),
	}, pageInfo)
}

func testOrdersFilterExcludeLiquidityByPartyStatesTypesAndTimeInForce(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	stores := setupOrderCursorPaginationTests(t)
	testData := generateTestOrdersForCursorPagination(t, ctx, stores)

	filter := entities.OrderFilter{
		PartyIDs:         []string{testData.parties[1].ID.String()},
		Statuses:         []vega.Order_Status{vega.Order_STATUS_ACTIVE, vega.Order_STATUS_PARTIALLY_FILLED},
		Types:            []vega.Order_Type{vega.Order_TYPE_LIMIT},
		TimeInForces:     []vega.Order_TimeInForce{vega.Order_TIME_IN_FORCE_GTC},
		ExcludeLiquidity: true,
	}

	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, true)
	require.NoError(t, err)

	got, pageInfo, err := stores.os.ListOrders(ctx, pagination, filter)
	require.NoError(t, err)

	want := append(
		[]entities.Order{},
		testData.orders[3],
		testData.orders[12],
	)
	sortOrders(want)
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[len(want)-1].Cursor().Encode(),
	}, pageInfo)
}

func testOrdersFilterExcludeLiquidityByReferenceStatesTypesAndTimeInForce(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	stores := setupOrderCursorPaginationTests(t)
	testData := generateTestOrdersForCursorPagination(t, ctx, stores)

	filter := entities.OrderFilter{
		Statuses:         []vega.Order_Status{vega.Order_STATUS_ACTIVE, vega.Order_STATUS_PARTIALLY_FILLED},
		Types:            []vega.Order_Type{vega.Order_TYPE_LIMIT},
		TimeInForces:     []vega.Order_TimeInForce{vega.Order_TIME_IN_FORCE_GTC},
		ExcludeLiquidity: true,
		Reference:        ptr.From("DEADBEEF"),
	}

	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, true)
	require.NoError(t, err)

	got, pageInfo, err := stores.os.ListOrders(ctx, pagination, filter)
	require.NoError(t, err)

	want := append([]entities.Order{}, testData.orders[3])
	sortOrders(want)
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[len(want)-1].Cursor().Encode(),
	}, pageInfo)
}
