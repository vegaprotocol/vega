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

func addTestOrder(t *testing.T, os *sqlstore.Orders, block entities.Block, party entities.Party, market entities.Market, reference string) entities.Order {
	order := entities.Order{
		ID:              entities.NewOrderID(generateID()),
		MarketID:        market.ID,
		PartyID:         party.ID,
		Side:            types.SideBuy,
		Price:           10,
		Size:            100,
		Remaining:       60,
		TimeInForce:     types.OrderTimeInForceGTC,
		Type:            types.OrderTypeLimit,
		Status:          types.OrderStatusActive,
		Reference:       reference,
		Version:         1,
		PeggedOffset:    0,
		PeggedReference: types.PeggedReferenceMid,
		CreatedAt:       time.Now().Truncate(time.Microsecond),
		UpdatedAt:       time.Now().Add(5 * time.Second).Truncate(time.Microsecond),
		ExpiresAt:       time.Now().Add(10 * time.Second).Truncate(time.Microsecond),
		VegaTime:        block.VegaTime,
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
		order := addTestOrder(t, os, block, parties[i%3], markets[i%2], fmt.Sprintf("my_reference_%d", i))
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
