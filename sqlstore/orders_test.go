package sqlstore_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/data-node/sqlstore"
	v2 "code.vegaprotocol.io/protos/data-node/api/v2"
	"code.vegaprotocol.io/vega/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func addTestOrder(t *testing.T, os *sqlstore.Orders, id entities.OrderID, block entities.Block, party entities.Party, market entities.Market, reference string,
	side types.Side, timeInForce types.OrderTimeInForce, orderType types.OrderType, status types.OrderStatus,
	price, size, remaining int64, seqNum uint64, version int32,
) entities.Order {
	order := entities.Order{
		ID:              id,
		MarketID:        market.ID,
		PartyID:         party.ID,
		Side:            side,
		Price:           price,
		Size:            size,
		Remaining:       remaining,
		TimeInForce:     timeInForce,
		Type:            orderType,
		Status:          status,
		Reference:       reference,
		Version:         version,
		PeggedOffset:    0,
		PeggedReference: types.PeggedReferenceMid,
		CreatedAt:       time.Now().Truncate(time.Microsecond),
		UpdatedAt:       time.Now().Add(5 * time.Second).Truncate(time.Microsecond),
		ExpiresAt:       time.Now().Add(10 * time.Second).Truncate(time.Microsecond),
		VegaTime:        block.VegaTime,
		SeqNum:          seqNum,
	}

	err := os.Add(order)
	require.NoError(t, err)
	return order
}

const numTestOrders = 30

func TestOrders(t *testing.T) {
	defer DeleteEverything()
	logger := logging.NewTestLogger()
	ps := sqlstore.NewParties(connectionSource)
	os := sqlstore.NewOrders(connectionSource, logger)
	bs := sqlstore.NewBlocks(connectionSource)
	block := addTestBlock(t, bs)
	block2 := addTestBlock(t, bs)

	// Make sure we're starting with an empty set of orders
	ctx := context.Background()
	emptyOrders, err := os.GetAll(ctx)
	assert.NoError(t, err)
	assert.Empty(t, emptyOrders)

	// Add other stuff order will use
	parties := []entities.Party{
		addTestParty(t, ps, block),
		addTestParty(t, ps, block),
		addTestParty(t, ps, block),
	}

	markets := []entities.Market{
		{ID: entities.NewMarketID("aa")},
		{ID: entities.NewMarketID("bb")},
	}

	// Make some orders
	orders := make([]entities.Order, numTestOrders)
	updatedOrders := make([]entities.Order, numTestOrders)
	numOrdersUpdatedInDifferentBlock := 0
	version := int32(1)
	for i := 0; i < numTestOrders; i++ {
		order := addTestOrder(t, os,
			entities.NewOrderID(generateID()),
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
		)
		orders[i] = order

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
	os.Flush(ctx)

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
	os.Flush(ctx)

	t.Run("GetAll", func(t *testing.T) {
		// Check we inserted new rows only when the update was in a different block
		allOrders, err := os.GetAll(ctx)
		require.NoError(t, err)
		assert.Equal(t, numTestOrders+numOrdersUpdatedInDifferentBlock, len(allOrders))
	})

	t.Run("GetByOrderID", func(t *testing.T) {
		// Ensure we get the most recently updated version
		for i := 0; i < numTestOrders; i++ {
			fetchedOrder, err := os.GetByOrderID(ctx, orders[i].ID.String(), nil)
			require.NoError(t, err)
			assert.Equal(t, fetchedOrder, updatedOrders[i])
		}
	})

	t.Run("GetByOrderID specific version", func(t *testing.T) {
		for i := 0; i < numTestOrders; i++ {
			ver := updatedOrders[i].Version
			fetchedOrder, err := os.GetByOrderID(ctx, updatedOrders[i].ID.String(), &ver)
			require.NoError(t, err)
			assert.Equal(t, fetchedOrder, updatedOrders[i])
		}
	})

	t.Run("GetByMarket", func(t *testing.T) {
		fetchedOrders, err := os.GetByMarket(ctx, markets[0].ID.String(), entities.OffsetPagination{})
		require.NoError(t, err)
		assert.Len(t, fetchedOrders, numTestOrders/2)
		for _, fetchedOrder := range fetchedOrders {
			assert.Equal(t, markets[0].ID, fetchedOrder.MarketID)
		}

		t.Run("OffsetPagination", func(t *testing.T) {
			fetchedOrdersP, err := os.GetByMarket(ctx,
				markets[0].ID.String(),
				entities.OffsetPagination{Skip: 4, Limit: 3, Descending: true})
			require.NoError(t, err)
			assert.Equal(t, reverseOrderSlice(fetchedOrders)[4:7], fetchedOrdersP)
		})
	})

	t.Run("GetByParty", func(t *testing.T) {
		fetchedOrders, err := os.GetByParty(ctx, parties[0].ID.String(), entities.OffsetPagination{})
		require.NoError(t, err)
		assert.Len(t, fetchedOrders, numTestOrders/3)
		for _, fetchedOrder := range fetchedOrders {
			assert.Equal(t, parties[0].ID, fetchedOrder.PartyID)
		}
	})

	t.Run("GetByReference", func(t *testing.T) {
		fetchedOrders, err := os.GetByReference(ctx, "my_reference_1", entities.OffsetPagination{})
		require.NoError(t, err)
		assert.Len(t, fetchedOrders, 1)
		assert.Equal(t, fetchedOrders[0], updatedOrders[1])
	})

	t.Run("GetAllVersionsByOrderID", func(t *testing.T) {
		fetchedOrders, err := os.GetAllVersionsByOrderID(ctx, orders[3].ID.String(), entities.OffsetPagination{})
		require.NoError(t, err)
		require.Len(t, fetchedOrders, 2)
		assert.Equal(t, int32(1), fetchedOrders[0].Version)
		assert.Equal(t, int32(2), fetchedOrders[1].Version)
	})
}

func reverseOrderSlice(input []entities.Order) (output []entities.Order) {
	for i := len(input) - 1; i >= 0; i-- {
		output = append(output, input[i])
	}
	return output
}

func generateTestBlocks(t *testing.T, numBlocks int, bs *sqlstore.Blocks) []entities.Block {
	t.Helper()
	blocks := make([]entities.Block, numBlocks, numBlocks)
	for i := 0; i < numBlocks; i++ {
		blocks[i] = addTestBlock(t, bs)
		time.Sleep(time.Millisecond)
	}
	return blocks
}

func generateParties(t *testing.T, numParties int, block entities.Block, ps *sqlstore.Parties) []entities.Party {
	t.Helper()
	parties := make([]entities.Party, numParties, numParties)
	for i := 0; i < numParties; i++ {
		parties[i] = addTestParty(t, ps, block)
	}
	return parties
}

func addTestMarket(t *testing.T, ms *sqlstore.Markets, block entities.Block) entities.Market {
	market := entities.Market{
		ID:       entities.NewMarketID(generateID()),
		VegaTime: block.VegaTime,
	}

	err := ms.Upsert(context.Background(), &market)
	require.NoError(t, err)
	return market
}

func generateMarkets(t *testing.T, numMarkets int, block entities.Block, ms *sqlstore.Markets) []entities.Market {
	t.Helper()
	markets := make([]entities.Market, numMarkets, numMarkets)
	for i := 0; i < numMarkets; i++ {
		markets[i] = addTestMarket(t, ms, block)
	}
	return markets
}

func generateOrderIDs(t *testing.T, numIDs int) []entities.OrderID {
	t.Helper()
	orderIDs := make([]entities.OrderID, numIDs)
	for i := 0; i < numIDs; i++ {
		orderIDs[i] = entities.NewOrderID(generateID())
		time.Sleep(time.Millisecond)
	}
	return orderIDs
}

func generateTestOrders(t *testing.T, blocks []entities.Block, parties []entities.Party,
	markets []entities.Market, orderIDs []entities.OrderID, os *sqlstore.Orders,
) []entities.Order {
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
		},
	}

	orders := make([]entities.Order, len(testOrders))

	lastBlockTime := blocks[0].VegaTime
	for i, to := range testOrders {
		// It's important for order triggers that orders are inserted in order. The batcher in the
		// order store does not preserve insert order, so manually flush each block.
		if to.block.VegaTime != lastBlockTime {
			os.Flush(context.Background())
			lastBlockTime = to.block.VegaTime
		}
		ref := fmt.Sprintf("reference-%d", i)
		orders[i] = addTestOrder(t, os, to.id, to.block, to.party, to.market, ref, to.side,
			to.timeInForce, to.orderType, to.status, to.price, to.size, to.remaining, uint64(i), int32(1))
	}

	return orders
}

func TestOrders_GetLiveOrders(t *testing.T) {
	defer DeleteEverything()
	logger := logging.NewTestLogger()
	bs := sqlstore.NewBlocks(connectionSource)
	ps := sqlstore.NewParties(connectionSource)
	ms := sqlstore.NewMarkets(connectionSource)
	os := sqlstore.NewOrders(connectionSource, logger)

	t.Logf("test store port: %d", testDBPort)

	// set up the blocks, parties and markets we need to generate the orders
	blocks := generateTestBlocks(t, 3, bs)
	parties := generateParties(t, 5, blocks[0], ps)
	markets := generateMarkets(t, 3, blocks[0], ms)
	orderIDs := generateOrderIDs(t, 8)
	testOrders := generateTestOrders(t, blocks, parties, markets, orderIDs, os)

	// Make sure we flush the batcher and write the orders to the database
	_, err := os.Flush(context.Background())
	require.NoError(t, err)

	want := append(testOrders[:3], testOrders[4:6]...)
	got, err := os.GetLiveOrders(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 5, len(got))
	assert.ElementsMatch(t, want, got)
}

func TestOrders_CursorPagination(t *testing.T) {
	t.Run("Should return all current orders for a given market when no cursor is given", testOrdersCursorPaginationByMarketNoCursor)
	t.Run("Should return all current orders for a given party when no cursor is given", testOrdersCursorPaginationByPartyNoCursor)
	t.Run("Should return all versions of a given order ID when no cursor is given", testOrdersCursorPaginationByOrderIDNoCursor)
	t.Run("Should return all current orders for a given party and market when no cursor is given", testOrdersCursorPaginationByMarketAndPartyNoCursor)

	t.Run("Should return the first page of current orders for a given market when a first cursor is given", testOrdersCursorPaginationByMarketFirstCursor)
	t.Run("Should return the first page of current orders for a given party when a first cursor is given", testOrdersCursorPaginationByPartyFirstCursor)
	t.Run("Should return the first page of order versions of a given order ID when a first cursor is given", testOrdersCursorPaginationByOrderIDFirstCursor)
	t.Run("Should return the first page of current orders for a given party and market when a first cursor is given", testOrdersCursorPaginationByMarketAndPartyFirstCursor)

	t.Run("Should return the last page of current orders for a given market when a last cursor is given", testOrdersCursorPaginationByMarketLastCursor)
	t.Run("Should return the last page of current orders for a given party when a last cursor is given", testOrdersCursorPaginationByPartyLastCursor)
	t.Run("Should return the last page of order versions of a given order ID when a last cursor is given", testOrdersCursorPaginationByOrderIDLastCursor)
	t.Run("Should return the last page of current orders for a given party and market when a last cursor is given", testOrdersCursorPaginationByMarketAndPartyLastCursor)

	t.Run("Should return the page of current orders for a given market when a first and after cursor is given", testOrdersCursorPaginationByMarketFirstAndAfterCursor)
	t.Run("Should return the page of current orders for a given party when a first and after cursor is given", testOrdersCursorPaginationByPartyFirstAndAfterCursor)
	t.Run("Should return the page of order versions of a given order ID when a first and after cursor is given", testOrdersCursorPaginationByOrderIDFirstAndAfterCursor)
	t.Run("Should return the page of current orders for a given party and market when a first and after cursor is given", testOrdersCursorPaginationByMarketAndPartyFirstAndAfterCursor)

	t.Run("Should return the page of current orders for a given market when a last and before cursor is given", testOrdersCursorPaginationByMarketLastAndBeforeCursor)
	t.Run("Should return the page of current orders for a given party when a last and before cursor is given", testOrdersCursorPaginationByPartyLastAndBeforeCursor)
	t.Run("Should return the page of order versions of a given order ID when a last and before cursor is given", testOrdersCursorPaginationByOrderIDLastAndBeforeCursor)
	t.Run("Should return the page of current orders for a given party and market when a last and before cursor is given", testOrdersCursorPaginationByMarketAndPartyLastAndBeforeCursor)
}

type orderTestStores struct {
	bs     *sqlstore.Blocks
	ps     *sqlstore.Parties
	ms     *sqlstore.Markets
	os     *sqlstore.Orders
	config sqlstore.Config
}

type orderTestData struct {
	blocks  []entities.Block
	parties []entities.Party
	markets []entities.Market
	orders  []entities.Order
	cursors []*entities.Cursor
}

func setupOrderCursorPaginationTests(t *testing.T) (*orderTestStores, func(t *testing.T)) {
	t.Helper()
	DeleteEverything()
	logger := logging.NewTestLogger()
	stores := &orderTestStores{
		bs:     sqlstore.NewBlocks(connectionSource),
		ps:     sqlstore.NewParties(connectionSource),
		ms:     sqlstore.NewMarkets(connectionSource),
		os:     sqlstore.NewOrders(connectionSource, logger),
		config: sqlstore.NewDefaultConfig(),
	}

	stores.config.ConnectionConfig.Port = testDBPort
	return stores, func(t *testing.T) {
		t.Helper()
		DeleteEverything()
	}
}

func generateTestOrdersForCursorPagination(t *testing.T, stores *orderTestStores) orderTestData {
	t.Helper()

	blocks := generateTestBlocks(t, 10, stores.bs)
	parties := generateParties(t, 2, blocks[0], stores.ps)
	markets := generateMarkets(t, 2, blocks[0], stores.ms)
	orderIDs := generateOrderIDs(t, 20)

	// Order with multiple versions orderIDs[1]

	testOrders := []struct {
		id          entities.OrderID
		block       entities.Block
		party       entities.Party
		market      entities.Market
		side        types.Side
		price       int64
		size        int64
		remaining   int64
		version     int32
		timeInForce types.OrderTimeInForce
		orderType   types.OrderType
		status      types.OrderStatus
		cursor      *entities.Cursor
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
			timeInForce: types.OrderTimeInForceGTC,
			orderType:   types.OrderTypeLimit,
			status:      types.OrderStatusActive,
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
		},
		{
			// testOrders[3]
			id:          orderIDs[2],
			block:       blocks[2],
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
			orderType:   types.OrderTypeLimit,
			status:      types.OrderStatusActive,
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
			orderType:   types.OrderTypeLimit,
			status:      types.OrderStatusActive,
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
			timeInForce: types.OrderTimeInForceGTC,
			orderType:   types.OrderTypeLimit,
			status:      types.OrderStatusActive,
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
			status:      types.OrderStatusActive,
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
			timeInForce: types.OrderTimeInForceGTC,
			orderType:   types.OrderTypeLimit,
			status:      types.OrderStatusActive,
		},
	}

	orders := make([]entities.Order, len(testOrders))
	cursors := make([]*entities.Cursor, len(testOrders))

	lastBlockTime := testOrders[0].block.VegaTime
	for i, order := range testOrders {
		// It's important for order triggers that orders are inserted in order. The batcher in the
		// order store does not preserve insert order, so manually flush each block.
		if order.block.VegaTime != lastBlockTime {
			stores.os.Flush(context.Background())
			lastBlockTime = order.block.VegaTime
		}

		seqNum := uint64(i)
		orderCursor := entities.OrderCursor{
			VegaTime: order.block.VegaTime,
			SeqNum:   seqNum,
		}
		cursors[i] = entities.NewCursor(orderCursor.String())
		orders[i] = addTestOrder(t, stores.os, order.id, order.block, order.party, order.market, "", order.side, order.timeInForce,
			order.orderType, order.status, order.price, order.size, order.remaining, seqNum, order.version)
	}

	// Make sure we flush the batcher and write the orders to the database
	_, err := stores.os.Flush(context.Background())
	require.NoError(t, err, "Could not insert test order data to the test database")

	return orderTestData{
		blocks:  blocks,
		parties: parties,
		markets: markets,
		orders:  orders,
		cursors: cursors,
	}
}

func testOrdersCursorPaginationByMarketNoCursor(t *testing.T) {
	stores, teardown := setupOrderCursorPaginationTests(t)
	defer teardown(t)
	testData := generateTestOrdersForCursorPagination(t, stores)

	t.Logf("Test DB Port: %d", testDBPort)

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	pagination, err := entities.PaginationFromProto(&v2.Pagination{})
	require.NoError(t, err)

	marketID := testData.markets[0].ID
	orders, pageInfo, err := stores.os.GetByMarketPaged(ctx, marketID.String(), pagination)
	require.NoError(t, err)
	assert.Len(t, orders, 5)
	want := append([]entities.Order{}, testData.orders[0], testData.orders[3], testData.orders[7], testData.orders[13], testData.orders[14])
	assert.Equal(t, want, orders)
	assert.False(t, pageInfo.HasNextPage)
	assert.False(t, pageInfo.HasPreviousPage)
	assert.Equal(t, pageInfo.StartCursor, testData.cursors[0].Encode())
	assert.Equal(t, pageInfo.EndCursor, testData.cursors[14].Encode())
}

func testOrdersCursorPaginationByPartyNoCursor(t *testing.T) {
	stores, teardown := setupOrderCursorPaginationTests(t)
	defer teardown(t)
	testData := generateTestOrdersForCursorPagination(t, stores)

	t.Logf("Test DB Port: %d", testDBPort)

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	pagination, err := entities.PaginationFromProto(&v2.Pagination{})
	require.NoError(t, err)

	partyID := testData.parties[1].ID
	orders, pageInfo, err := stores.os.GetByPartyPaged(ctx, partyID.String(), pagination)
	require.NoError(t, err)
	assert.Len(t, orders, 6)
	want := append([]entities.Order{}, testData.orders[3], testData.orders[5], testData.orders[7], testData.orders[9], testData.orders[12], testData.orders[13])
	assert.Equal(t, want, orders)
	assert.False(t, pageInfo.HasNextPage)
	assert.False(t, pageInfo.HasPreviousPage)
	assert.Equal(t, pageInfo.StartCursor, testData.cursors[3].Encode())
	assert.Equal(t, pageInfo.EndCursor, testData.cursors[13].Encode())
}

func testOrdersCursorPaginationByOrderIDNoCursor(t *testing.T) {
	stores, teardown := setupOrderCursorPaginationTests(t)
	defer teardown(t)
	testData := generateTestOrdersForCursorPagination(t, stores)

	t.Logf("Test DB Port: %d", testDBPort)

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	pagination, err := entities.PaginationFromProto(&v2.Pagination{})
	require.NoError(t, err)

	orderID := testData.orders[1].ID
	orders, pageInfo, err := stores.os.GetOrderVersionsByIDPaged(ctx, orderID.String(), pagination)
	require.NoError(t, err)
	assert.Len(t, orders, 6)
	want := append([]entities.Order{}, testData.orders[1], testData.orders[2], testData.orders[6], testData.orders[8], testData.orders[11], testData.orders[13])
	assert.Equal(t, want, orders)
	assert.False(t, pageInfo.HasNextPage)
	assert.False(t, pageInfo.HasPreviousPage)
	assert.Equal(t, pageInfo.StartCursor, testData.cursors[1].Encode())
	assert.Equal(t, pageInfo.EndCursor, testData.cursors[13].Encode())
}

func testOrdersCursorPaginationByMarketAndPartyNoCursor(t *testing.T) {
	stores, teardown := setupOrderCursorPaginationTests(t)
	defer teardown(t)
	testData := generateTestOrdersForCursorPagination(t, stores)

	t.Logf("Test DB Port: %d", testDBPort)

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	pagination, err := entities.PaginationFromProto(&v2.Pagination{})
	require.NoError(t, err)

	partyID := testData.parties[1].ID
	marketID := testData.markets[1].ID
	orders, pageInfo, err := stores.os.GetByPartyAndMarketPaged(ctx, partyID.String(), marketID.String(), pagination)
	require.NoError(t, err)
	assert.Len(t, orders, 3)
	want := append([]entities.Order{}, testData.orders[5], testData.orders[9], testData.orders[12])
	assert.Equal(t, want, orders)
	assert.False(t, pageInfo.HasNextPage)
	assert.False(t, pageInfo.HasPreviousPage)
	assert.Equal(t, pageInfo.StartCursor, testData.cursors[5].Encode())
	assert.Equal(t, pageInfo.EndCursor, testData.cursors[12].Encode())
}

func testOrdersCursorPaginationByMarketFirstCursor(t *testing.T) {
	stores, teardown := setupOrderCursorPaginationTests(t)
	defer teardown(t)
	testData := generateTestOrdersForCursorPagination(t, stores)

	t.Logf("Test DB Port: %d", testDBPort)

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	first := int32(3)
	pagination, err := entities.PaginationFromProto(&v2.Pagination{
		First: &first,
	})
	require.NoError(t, err)

	marketID := testData.markets[0].ID
	orders, pageInfo, err := stores.os.GetByMarketPaged(ctx, marketID.String(), pagination)
	require.NoError(t, err)
	assert.Len(t, orders, 3)
	want := append([]entities.Order{}, testData.orders[0], testData.orders[3], testData.orders[7])
	assert.Equal(t, want, orders)
	assert.True(t, pageInfo.HasNextPage)
	assert.False(t, pageInfo.HasPreviousPage)
	assert.Equal(t, pageInfo.StartCursor, testData.cursors[0].Encode())
	assert.Equal(t, pageInfo.EndCursor, testData.cursors[7].Encode())
}

func testOrdersCursorPaginationByPartyFirstCursor(t *testing.T) {
	stores, teardown := setupOrderCursorPaginationTests(t)
	defer teardown(t)
	testData := generateTestOrdersForCursorPagination(t, stores)

	t.Logf("Test DB Port: %d", testDBPort)

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	first := int32(3)
	pagination, err := entities.PaginationFromProto(&v2.Pagination{
		First: &first,
	})
	require.NoError(t, err)

	partyID := testData.parties[1].ID
	orders, pageInfo, err := stores.os.GetByPartyPaged(ctx, partyID.String(), pagination)
	require.NoError(t, err)
	assert.Len(t, orders, 3)
	want := append([]entities.Order{}, testData.orders[3], testData.orders[5], testData.orders[7])
	assert.Equal(t, want, orders)
	assert.True(t, pageInfo.HasNextPage)
	assert.False(t, pageInfo.HasPreviousPage)
	assert.Equal(t, pageInfo.StartCursor, testData.cursors[3].Encode())
	assert.Equal(t, pageInfo.EndCursor, testData.cursors[7].Encode())
}

func testOrdersCursorPaginationByOrderIDFirstCursor(t *testing.T) {
	stores, teardown := setupOrderCursorPaginationTests(t)
	defer teardown(t)
	testData := generateTestOrdersForCursorPagination(t, stores)

	t.Logf("Test DB Port: %d", testDBPort)

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	first := int32(3)
	pagination, err := entities.PaginationFromProto(&v2.Pagination{
		First: &first,
	})
	require.NoError(t, err)

	orderID := testData.orders[1].ID
	orders, pageInfo, err := stores.os.GetOrderVersionsByIDPaged(ctx, orderID.String(), pagination)
	require.NoError(t, err)
	assert.Len(t, orders, 3)
	want := append([]entities.Order{}, testData.orders[1], testData.orders[2], testData.orders[6])
	assert.Equal(t, want, orders)
	assert.True(t, pageInfo.HasNextPage)
	assert.False(t, pageInfo.HasPreviousPage)
	assert.Equal(t, pageInfo.StartCursor, testData.cursors[1].Encode())
	assert.Equal(t, pageInfo.EndCursor, testData.cursors[6].Encode())
}

func testOrdersCursorPaginationByMarketAndPartyFirstCursor(t *testing.T) {
	stores, teardown := setupOrderCursorPaginationTests(t)
	defer teardown(t)
	testData := generateTestOrdersForCursorPagination(t, stores)

	t.Logf("Test DB Port: %d", testDBPort)

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	first := int32(2)
	pagination, err := entities.PaginationFromProto(&v2.Pagination{
		First: &first,
	})
	require.NoError(t, err)

	partyID := testData.parties[1].ID
	marketID := testData.markets[1].ID
	orders, pageInfo, err := stores.os.GetByPartyAndMarketPaged(ctx, partyID.String(), marketID.String(), pagination)
	require.NoError(t, err)
	assert.Len(t, orders, 2)
	want := append([]entities.Order{}, testData.orders[5], testData.orders[9])
	assert.Equal(t, want, orders)
	assert.True(t, pageInfo.HasNextPage)
	assert.False(t, pageInfo.HasPreviousPage)
	assert.Equal(t, pageInfo.StartCursor, testData.cursors[5].Encode())
	assert.Equal(t, pageInfo.EndCursor, testData.cursors[9].Encode())
}

func testOrdersCursorPaginationByMarketLastCursor(t *testing.T) {
	stores, teardown := setupOrderCursorPaginationTests(t)
	defer teardown(t)
	testData := generateTestOrdersForCursorPagination(t, stores)

	t.Logf("Test DB Port: %d", testDBPort)

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	last := int32(3)
	pagination, err := entities.PaginationFromProto(&v2.Pagination{
		Last: &last,
	})
	require.NoError(t, err)

	marketID := testData.markets[0].ID
	orders, pageInfo, err := stores.os.GetByMarketPaged(ctx, marketID.String(), pagination)
	require.NoError(t, err)
	assert.Len(t, orders, 3)
	want := append([]entities.Order{}, testData.orders[7], testData.orders[13], testData.orders[14])
	assert.Equal(t, want, orders)
	assert.False(t, pageInfo.HasNextPage)
	assert.True(t, pageInfo.HasPreviousPage)
	assert.Equal(t, pageInfo.StartCursor, testData.cursors[7].Encode())
	assert.Equal(t, pageInfo.EndCursor, testData.cursors[14].Encode())
}

func testOrdersCursorPaginationByPartyLastCursor(t *testing.T) {
	stores, teardown := setupOrderCursorPaginationTests(t)
	defer teardown(t)
	testData := generateTestOrdersForCursorPagination(t, stores)

	t.Logf("Test DB Port: %d", testDBPort)

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	last := int32(3)
	pagination, err := entities.PaginationFromProto(&v2.Pagination{
		Last: &last,
	})
	require.NoError(t, err)

	partyID := testData.parties[1].ID
	orders, pageInfo, err := stores.os.GetByPartyPaged(ctx, partyID.String(), pagination)
	require.NoError(t, err)
	assert.Len(t, orders, 3)
	want := append([]entities.Order{}, testData.orders[9], testData.orders[12], testData.orders[13])
	assert.Equal(t, want, orders)
	assert.False(t, pageInfo.HasNextPage)
	assert.True(t, pageInfo.HasPreviousPage)
	assert.Equal(t, pageInfo.StartCursor, testData.cursors[9].Encode())
	assert.Equal(t, pageInfo.EndCursor, testData.cursors[13].Encode())
}

func testOrdersCursorPaginationByOrderIDLastCursor(t *testing.T) {
	stores, teardown := setupOrderCursorPaginationTests(t)
	defer teardown(t)
	testData := generateTestOrdersForCursorPagination(t, stores)

	t.Logf("Test DB Port: %d", testDBPort)

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	last := int32(3)
	pagination, err := entities.PaginationFromProto(&v2.Pagination{
		Last: &last,
	})
	require.NoError(t, err)

	orderID := testData.orders[1].ID
	orders, pageInfo, err := stores.os.GetOrderVersionsByIDPaged(ctx, orderID.String(), pagination)
	require.NoError(t, err)
	assert.Len(t, orders, 3)
	want := append([]entities.Order{}, testData.orders[8], testData.orders[11], testData.orders[13])
	assert.Equal(t, want, orders)
	assert.False(t, pageInfo.HasNextPage)
	assert.True(t, pageInfo.HasPreviousPage)
	assert.Equal(t, pageInfo.StartCursor, testData.cursors[8].Encode())
	assert.Equal(t, pageInfo.EndCursor, testData.cursors[13].Encode())
}

func testOrdersCursorPaginationByMarketAndPartyLastCursor(t *testing.T) {
	stores, teardown := setupOrderCursorPaginationTests(t)
	defer teardown(t)
	testData := generateTestOrdersForCursorPagination(t, stores)

	t.Logf("Test DB Port: %d", testDBPort)

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	last := int32(2)
	pagination, err := entities.PaginationFromProto(&v2.Pagination{
		Last: &last,
	})
	require.NoError(t, err)

	partyID := testData.parties[1].ID
	marketID := testData.markets[1].ID
	orders, pageInfo, err := stores.os.GetByPartyAndMarketPaged(ctx, partyID.String(), marketID.String(), pagination)
	require.NoError(t, err)
	assert.Len(t, orders, 2)
	want := append([]entities.Order{}, testData.orders[9], testData.orders[12])
	assert.Equal(t, want, orders)
	assert.False(t, pageInfo.HasNextPage)
	assert.True(t, pageInfo.HasPreviousPage)
	assert.Equal(t, pageInfo.StartCursor, testData.cursors[9].Encode())
	assert.Equal(t, pageInfo.EndCursor, testData.cursors[12].Encode())
}

func testOrdersCursorPaginationByMarketFirstAndAfterCursor(t *testing.T) {
	stores, teardown := setupOrderCursorPaginationTests(t)
	defer teardown(t)
	testData := generateTestOrdersForCursorPagination(t, stores)

	t.Logf("Test DB Port: %d", testDBPort)

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	first := int32(3)
	after := testData.cursors[0].Encode()
	pagination, err := entities.PaginationFromProto(&v2.Pagination{
		First: &first,
		After: &after,
	})
	require.NoError(t, err)

	marketID := testData.markets[0].ID
	orders, pageInfo, err := stores.os.GetByMarketPaged(ctx, marketID.String(), pagination)
	require.NoError(t, err)
	assert.Len(t, orders, 3)
	want := append([]entities.Order{}, testData.orders[3], testData.orders[7], testData.orders[13])
	assert.Equal(t, want, orders)
	assert.True(t, pageInfo.HasNextPage)
	assert.True(t, pageInfo.HasPreviousPage)
	assert.Equal(t, pageInfo.StartCursor, testData.cursors[3].Encode())
	assert.Equal(t, pageInfo.EndCursor, testData.cursors[13].Encode())
}

func testOrdersCursorPaginationByPartyFirstAndAfterCursor(t *testing.T) {
	stores, teardown := setupOrderCursorPaginationTests(t)
	defer teardown(t)
	testData := generateTestOrdersForCursorPagination(t, stores)

	t.Logf("Test DB Port: %d", testDBPort)

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	first := int32(3)
	after := testData.cursors[5].Encode()
	pagination, err := entities.PaginationFromProto(&v2.Pagination{
		First: &first,
		After: &after,
	})
	require.NoError(t, err)

	partyID := testData.parties[1].ID
	orders, pageInfo, err := stores.os.GetByPartyPaged(ctx, partyID.String(), pagination)
	require.NoError(t, err)
	assert.Len(t, orders, 3)
	want := append([]entities.Order{}, testData.orders[7], testData.orders[9], testData.orders[12])
	assert.Equal(t, want, orders)
	assert.True(t, pageInfo.HasNextPage)
	assert.True(t, pageInfo.HasPreviousPage)
	assert.Equal(t, pageInfo.StartCursor, testData.cursors[7].Encode())
	assert.Equal(t, pageInfo.EndCursor, testData.cursors[12].Encode())
}

func testOrdersCursorPaginationByOrderIDFirstAndAfterCursor(t *testing.T) {
	stores, teardown := setupOrderCursorPaginationTests(t)
	defer teardown(t)
	testData := generateTestOrdersForCursorPagination(t, stores)

	t.Logf("Test DB Port: %d", testDBPort)

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	first := int32(3)
	after := testData.cursors[2].Encode()
	pagination, err := entities.PaginationFromProto(&v2.Pagination{
		First: &first,
		After: &after,
	})
	require.NoError(t, err)

	orderID := testData.orders[1].ID
	orders, pageInfo, err := stores.os.GetOrderVersionsByIDPaged(ctx, orderID.String(), pagination)
	require.NoError(t, err)
	assert.Len(t, orders, 3)
	want := append([]entities.Order{}, testData.orders[6], testData.orders[8], testData.orders[11])
	assert.Equal(t, want, orders)
	assert.True(t, pageInfo.HasNextPage)
	assert.True(t, pageInfo.HasPreviousPage)
	assert.Equal(t, pageInfo.StartCursor, testData.cursors[6].Encode())
	assert.Equal(t, pageInfo.EndCursor, testData.cursors[11].Encode())
}

func testOrdersCursorPaginationByMarketAndPartyFirstAndAfterCursor(t *testing.T) {
	stores, teardown := setupOrderCursorPaginationTests(t)
	defer teardown(t)
	testData := generateTestOrdersForCursorPagination(t, stores)

	t.Logf("Test DB Port: %d", testDBPort)

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	first := int32(1)
	after := testData.cursors[5].Encode()
	pagination, err := entities.PaginationFromProto(&v2.Pagination{
		First: &first,
		After: &after,
	})
	require.NoError(t, err)

	partyID := testData.parties[1].ID
	marketID := testData.markets[1].ID
	orders, pageInfo, err := stores.os.GetByPartyAndMarketPaged(ctx, partyID.String(), marketID.String(), pagination)
	require.NoError(t, err)
	assert.Len(t, orders, 1)
	want := append([]entities.Order{}, testData.orders[9])
	assert.Equal(t, want, orders)
	assert.True(t, pageInfo.HasNextPage)
	assert.True(t, pageInfo.HasPreviousPage)
	assert.Equal(t, pageInfo.StartCursor, testData.cursors[9].Encode())
	assert.Equal(t, pageInfo.EndCursor, testData.cursors[9].Encode())
}

func testOrdersCursorPaginationByMarketLastAndBeforeCursor(t *testing.T) {
	stores, teardown := setupOrderCursorPaginationTests(t)
	defer teardown(t)
	testData := generateTestOrdersForCursorPagination(t, stores)

	t.Logf("Test DB Port: %d", testDBPort)

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	last := int32(3)
	before := testData.cursors[14].Encode()
	pagination, err := entities.PaginationFromProto(&v2.Pagination{
		Last:   &last,
		Before: &before,
	})
	require.NoError(t, err)

	marketID := testData.markets[0].ID
	orders, pageInfo, err := stores.os.GetByMarketPaged(ctx, marketID.String(), pagination)
	require.NoError(t, err)
	assert.Len(t, orders, 3)
	want := append([]entities.Order{}, testData.orders[3], testData.orders[7], testData.orders[13])
	assert.Equal(t, want, orders)
	assert.True(t, pageInfo.HasNextPage)
	assert.True(t, pageInfo.HasPreviousPage)
	assert.Equal(t, pageInfo.StartCursor, testData.cursors[3].Encode())
	assert.Equal(t, pageInfo.EndCursor, testData.cursors[13].Encode())
}

func testOrdersCursorPaginationByPartyLastAndBeforeCursor(t *testing.T) {
	stores, teardown := setupOrderCursorPaginationTests(t)
	defer teardown(t)
	testData := generateTestOrdersForCursorPagination(t, stores)

	t.Logf("Test DB Port: %d", testDBPort)

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	last := int32(3)
	before := testData.cursors[12].Encode()
	pagination, err := entities.PaginationFromProto(&v2.Pagination{
		Last:   &last,
		Before: &before,
	})
	require.NoError(t, err)

	partyID := testData.parties[1].ID
	orders, pageInfo, err := stores.os.GetByPartyPaged(ctx, partyID.String(), pagination)
	require.NoError(t, err)
	assert.Len(t, orders, 3)
	want := append([]entities.Order{}, testData.orders[5], testData.orders[7], testData.orders[9])
	assert.Equal(t, want, orders)
	assert.True(t, pageInfo.HasNextPage)
	assert.True(t, pageInfo.HasPreviousPage)
	assert.Equal(t, pageInfo.StartCursor, testData.cursors[5].Encode())
	assert.Equal(t, pageInfo.EndCursor, testData.cursors[9].Encode())
}

func testOrdersCursorPaginationByOrderIDLastAndBeforeCursor(t *testing.T) {
	stores, teardown := setupOrderCursorPaginationTests(t)
	defer teardown(t)
	testData := generateTestOrdersForCursorPagination(t, stores)

	t.Logf("Test DB Port: %d", testDBPort)

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	last := int32(3)
	before := testData.cursors[11].Encode()
	pagination, err := entities.PaginationFromProto(&v2.Pagination{
		Last:   &last,
		Before: &before,
	})
	require.NoError(t, err)

	orderID := testData.orders[1].ID
	orders, pageInfo, err := stores.os.GetOrderVersionsByIDPaged(ctx, orderID.String(), pagination)
	require.NoError(t, err)
	assert.Len(t, orders, 3)
	want := append([]entities.Order{}, testData.orders[2], testData.orders[6], testData.orders[8])
	assert.Equal(t, want, orders)
	assert.True(t, pageInfo.HasNextPage)
	assert.True(t, pageInfo.HasPreviousPage)
	assert.Equal(t, pageInfo.StartCursor, testData.cursors[2].Encode())
	assert.Equal(t, pageInfo.EndCursor, testData.cursors[8].Encode())
}

func testOrdersCursorPaginationByMarketAndPartyLastAndBeforeCursor(t *testing.T) {
	stores, teardown := setupOrderCursorPaginationTests(t)
	defer teardown(t)
	testData := generateTestOrdersForCursorPagination(t, stores)

	t.Logf("Test DB Port: %d", testDBPort)

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	last := int32(1)
	before := testData.cursors[12].Encode()
	pagination, err := entities.PaginationFromProto(&v2.Pagination{
		Last:   &last,
		Before: &before,
	})
	require.NoError(t, err)

	partyID := testData.parties[1].ID
	marketID := testData.markets[1].ID
	orders, pageInfo, err := stores.os.GetByPartyAndMarketPaged(ctx, partyID.String(), marketID.String(), pagination)
	require.NoError(t, err)
	assert.Len(t, orders, 1)
	want := append([]entities.Order{}, testData.orders[9])
	assert.Equal(t, want, orders)
	assert.True(t, pageInfo.HasNextPage)
	assert.True(t, pageInfo.HasPreviousPage)
	assert.Equal(t, pageInfo.StartCursor, testData.cursors[9].Encode())
	assert.Equal(t, pageInfo.EndCursor, testData.cursors[9].Encode())
}
