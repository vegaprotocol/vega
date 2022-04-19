package sqlstore_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/sqlstore"
	"code.vegaprotocol.io/vega/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func addTestOrder(t *testing.T, os *sqlstore.Orders, id entities.OrderID, block entities.Block, party entities.Party, market entities.Market, reference string,
	side types.Side, timeInForce types.OrderTimeInForce, orderType types.OrderType, status types.OrderStatus,
	price, size, remaining int64, seqNum uint64) entities.Order {
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
		Version:         1,
		PeggedOffset:    0,
		PeggedReference: types.PeggedReferenceMid,
		CreatedAt:       time.Now().Truncate(time.Microsecond),
		UpdatedAt:       time.Now().Add(5 * time.Second).Truncate(time.Microsecond),
		ExpiresAt:       time.Now().Add(10 * time.Second).Truncate(time.Microsecond),
		VegaTime:        block.VegaTime,
		SeqNum:          seqNum,
	}

	err := os.Add(context.Background(), order)
	require.NoError(t, err)
	return order
}

const numTestOrders = 30

func TestOrders(t *testing.T) {
	defer testStore.DeleteEverything()
	ps := sqlstore.NewParties(testStore)
	os := sqlstore.NewOrders(testStore)
	bs := sqlstore.NewBlocks(testStore)
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
	orders := []entities.Order{}
	updatedOrders := []entities.Order{}
	numOrdersUpdatedInDifferentBlock := 0
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
		)
		orders = append(orders, order)

		// Don't update 1/4 of the orders
		updatedOrder := order

		// Update 1/4 of the orders in the same block
		if i%4 == 1 {
			updatedOrder = order
			updatedOrder.Remaining = 50
			err = os.Add(context.Background(), updatedOrder)
			require.NoError(t, err)
		}

		// Update Another 1/4 of the orders in the next block
		if i%4 == 2 {
			updatedOrder = order
			updatedOrder.Remaining = 25
			updatedOrder.VegaTime = block2.VegaTime
			err = os.Add(context.Background(), updatedOrder)
			require.NoError(t, err)
			numOrdersUpdatedInDifferentBlock++
		}

		// Update Another 1/4 of the orders in the next block with an incremented version
		if i%4 == 3 {
			updatedOrder = order
			updatedOrder.Remaining = 10
			updatedOrder.VegaTime = block2.VegaTime
			updatedOrder.Version++
			err = os.Add(context.Background(), updatedOrder)
			require.NoError(t, err)
			numOrdersUpdatedInDifferentBlock++
		}

		updatedOrders = append(updatedOrders, updatedOrder)
	}

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
		fetchedOrders, err := os.GetByMarket(ctx, markets[0].ID.String(), entities.Pagination{})
		require.NoError(t, err)
		assert.Len(t, fetchedOrders, numTestOrders/2)
		for _, fetchedOrder := range fetchedOrders {
			assert.Equal(t, markets[0].ID, fetchedOrder.MarketID)
		}

		t.Run("Pagination", func(t *testing.T) {
			fetchedOrdersP, err := os.GetByMarket(ctx,
				markets[0].ID.String(),
				entities.Pagination{Skip: 4, Limit: 3, Descending: true})
			require.NoError(t, err)
			assert.Equal(t, reverseOrderSlice(fetchedOrders)[4:7], fetchedOrdersP)
		})
	})

	t.Run("GetByParty", func(t *testing.T) {
		fetchedOrders, err := os.GetByParty(ctx, parties[0].ID.String(), entities.Pagination{})
		require.NoError(t, err)
		assert.Len(t, fetchedOrders, numTestOrders/3)
		for _, fetchedOrder := range fetchedOrders {
			assert.Equal(t, parties[0].ID, fetchedOrder.PartyID)
		}
	})

	t.Run("GetByReference", func(t *testing.T) {
		fetchedOrders, err := os.GetByReference(ctx, "my_reference_1", entities.Pagination{})
		require.NoError(t, err)
		assert.Len(t, fetchedOrders, 1)
		assert.Equal(t, fetchedOrders[0], updatedOrders[1])
	})

	t.Run("GetAllVersionsByOrderID", func(t *testing.T) {
		fetchedOrders, err := os.GetAllVersionsByOrderID(ctx, orders[3].ID.String(), entities.Pagination{})
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

	err := ms.Upsert(&market)
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

	for i, to := range testOrders {
		ref := fmt.Sprintf("reference-%d", i)
		orders[i] = addTestOrder(t, os, to.id, to.block, to.party, to.market, ref, to.side,
			to.timeInForce, to.orderType, to.status, to.price, to.size, to.remaining, uint64(i))
	}

	return orders
}

func TestOrders_GetLiveOrders(t *testing.T) {
	defer testStore.DeleteEverything()

	bs := sqlstore.NewBlocks(testStore)
	ps := sqlstore.NewParties(testStore)
	ms := sqlstore.NewMarkets(testStore)
	os := sqlstore.NewOrders(testStore)

	t.Logf("test store port: %d", testDBPort)

	// set up the blocks, parties and markets we need to generate the orders
	blocks := generateTestBlocks(t, 3, bs)
	parties := generateParties(t, 5, blocks[0], ps)
	markets := generateMarkets(t, 3, blocks[0], ms)
	orderIDs := generateOrderIDs(t, 8)
	testOrders := generateTestOrders(t, blocks, parties, markets, orderIDs, os)

	// Make sure we flush the batcher and write the orders to the database
	err := os.Flush(context.Background())
	require.NoError(t, err)

	want := append(testOrders[:3], testOrders[4:6]...)
	got, err := os.GetLiveOrders(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 5, len(got))
	assert.ElementsMatch(t, want, got)
}
