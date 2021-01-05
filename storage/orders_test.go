package storage_test

import (
	"context"
	"strings"
	"testing"

	"code.vegaprotocol.io/vega/execution"

	"code.vegaprotocol.io/vega/logging"
	types "code.vegaprotocol.io/vega/proto/gen/golang"
	"code.vegaprotocol.io/vega/storage"

	"github.com/stretchr/testify/assert"
)

const testMarket = "market"
const testParty = "party"
const testPartyA = "partyA"
const testPartyB = "partyB"

func TestStorage_NewOrders(t *testing.T) {
	config, err := storage.NewTestConfig()
	if err != nil {
		t.Fatalf("unable to setup badger dirs: %v", err)
	}

	storage.FlushStores(logging.NewTestLogger(), config)

	orderStore, err := storage.NewOrders(logging.NewTestLogger(), config, func() {})
	assert.NotNil(t, orderStore)
	assert.Nil(t, err)

	config.OrdersDirPath = ""

	orderStore, err = storage.NewOrders(logging.NewTestLogger(), config, func() {})
	assert.Nil(t, orderStore)
	assert.NotNil(t, err)

	nsf := strings.Contains(err.Error(), "no such file or directory")
	assert.True(t, nsf)
}

func TestStorage_PostAndGetNewOrder(t *testing.T) {
	config, err := storage.NewTestConfig()
	if err != nil {
		t.Fatalf("unable to setup badger dirs: %v", err)
	}

	storage.FlushStores(logging.NewTestLogger(), config)
	orderStore, err := storage.NewOrders(logging.NewTestLogger(), config, func() {})
	assert.NoError(t, err)
	defer orderStore.Close()

	var order = types.Order{
		Id:       "45305210ff7a9bb9450b1833cc10368a",
		MarketID: "testMarket",
		PartyID:  "testParty",
	}

	err = orderStore.SaveBatch([]types.Order{order})
	assert.NoError(t, err)

	o, err := orderStore.GetByMarketAndID(context.Background(), "testMarket", order.Id)
	assert.Nil(t, err)
	assert.Equal(t, order.Id, o.Id)
}

func TestStorage_PostAndGetByReference(t *testing.T) {
	config, err := storage.NewTestConfig()
	if err != nil {
		t.Fatalf("unable to setup badger dirs: %v", err)
	}

	storage.FlushStores(logging.NewTestLogger(), config)
	orderStore, err := storage.NewOrders(logging.NewTestLogger(), config, func() {})
	assert.NoError(t, err)
	defer orderStore.Close()

	var order = types.Order{
		Reference: "83cfdf76-8eac-4c7e-8f6a-2aa51e89364f",
		Id:        "45305210ff7a9bb9450b1833cc10368a",
		MarketID:  "testMarket",
		PartyID:   "testParty",
	}

	err = orderStore.SaveBatch([]types.Order{order})
	assert.NoError(t, err)

	o, err := orderStore.GetByReference(context.Background(), order.Reference)
	assert.Nil(t, err)
	assert.Equal(t, order.Id, o.Id)
}

func TestStorage_GetOrdersForMarket(t *testing.T) {
	config, err := storage.NewTestConfig()
	if err != nil {
		t.Fatalf("unable to setup badger dirs: %v", err)
	}

	storage.FlushStores(logging.NewTestLogger(), config)

	var tests = []struct {
		inMarkets      []string
		inOrders       []*types.Order
		inLimit        uint64
		inMarket       string
		outOrdersCount int
	}{
		{
			inMarkets: []string{"testMarket1", "marketZ"},
			inOrders: []*types.Order{
				{
					Status:   types.Order_STATUS_ACTIVE,
					Id:       "d41d8cd98f00b204e9800998ecf8427e",
					MarketID: "testMarket1",
					PartyID:  testParty,
				},
				{
					Status:   types.Order_STATUS_ACTIVE,
					Id:       "ad2dc275947362c45893bbeb30fc3098",
					MarketID: "marketZ",
					PartyID:  testParty,
				},
				{
					Status:   types.Order_STATUS_ACTIVE,
					Id:       "4e8e41367997cfe705d62ea80592cbcc",
					MarketID: "testMarket1",
					PartyID:  testParty,
				},
			},
			inLimit:        5000,
			inMarket:       "testMarket1",
			outOrdersCount: 2,
		},
		{
			inMarkets: []string{testMarket, "marketABC"},
			inOrders: []*types.Order{
				{
					Status:   types.Order_STATUS_ACTIVE,
					Id:       "d41d8cd98f00b204e9800998ecf8427e",
					MarketID: testMarket,
					PartyID:  testParty,
				},
				{
					Status:   types.Order_STATUS_ACTIVE,
					Id:       "ad2dc275947362c45893bbeb30fc3098",
					MarketID: "marketABC",
					PartyID:  testParty,
				},
				{
					Status:   types.Order_STATUS_ACTIVE,
					Id:       "4e8e41367997cfe705d62ea80592cbcc",
					MarketID: testMarket,
					PartyID:  testParty,
				},
			},
			inLimit:        5000,
			inMarket:       "marketABC",
			outOrdersCount: 1,
		},
		{
			inMarkets: []string{"marketXYZ"},
			inOrders: []*types.Order{
				{
					Status:   types.Order_STATUS_ACTIVE,
					Id:       "d41d8cd98f00b204e9800998ecf8427e",
					MarketID: "marketXYZ",
					PartyID:  testParty,
				},
				{
					Status:   types.Order_STATUS_ACTIVE,
					Id:       "ad2dc275947362c45893bbeb30fc3098",
					MarketID: "marketXYZ",
					PartyID:  testParty,
				},
				{
					Status:   types.Order_STATUS_ACTIVE,
					Id:       "4e8e41367997cfe705d62ea80592cbcc",
					MarketID: "marketXYZ",
					PartyID:  testParty,
				},
			},
			inLimit:        2,
			inMarket:       "marketXYZ",
			outOrdersCount: 2,
		},
	}

	for _, tt := range tests {
		orderStore, err := storage.NewOrders(logging.NewTestLogger(), config, func() {})
		assert.Nil(t, err)

		vOrders := make([]types.Order, len(tt.inOrders))
		for _, order := range tt.inOrders {
			o := *order
			vOrders = append(vOrders, o)
			assert.Nil(t, err)
		}

		err = orderStore.SaveBatch(vOrders)
		assert.NoError(t, err)

		orders, err := orderStore.GetByMarket(context.Background(), tt.inMarket, 0, tt.inLimit, false)
		assert.Nil(t, err)
		assert.Equal(t, tt.outOrdersCount, len(orders))
		orderStore.Close()
	}
}

func TestStorage_GetOrdersForParty(t *testing.T) {
	config, err := storage.NewTestConfig()
	if err != nil {
		t.Fatalf("unable to setup badger dirs: %v", err)
	}

	storage.FlushStores(logging.NewTestLogger(), config)

	orderStore, err := storage.NewOrders(logging.NewTestLogger(), config, func() {})
	assert.Nil(t, err)
	defer orderStore.Close()

	passiveOrder := types.Order{
		Status:    types.Order_STATUS_ACTIVE,
		Id:        "d41d8cd98f00b204e9800998ecf9999e",
		MarketID:  testMarket,
		PartyID:   testPartyA,
		Remaining: 0,
	}

	aggressiveOrder := types.Order{
		Status:    types.Order_STATUS_ACTIVE,
		Id:        "d41d8cd98f00b204e9800998ecf8427e",
		MarketID:  testMarket,
		PartyID:   testPartyB,
		Remaining: 100,
	}

	err = orderStore.SaveBatch([]types.Order{passiveOrder})
	assert.NoError(t, err)
	err = orderStore.SaveBatch([]types.Order{aggressiveOrder})
	assert.NoError(t, err)

	ordersAtPartyA, err := orderStore.GetByParty(context.Background(), testPartyA, 0, 0, false)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(ordersAtPartyA))

	ordersAtPartyB, err := orderStore.GetByParty(context.Background(), testPartyB, 0, 0, false)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(ordersAtPartyB))

	orderAtPartyA, err := orderStore.GetByPartyAndID(context.Background(), testPartyA, passiveOrder.Id)
	assert.Nil(t, err)
	assert.Equal(t, passiveOrder.Id, orderAtPartyA.Id)

	orderAtPartyB, err := orderStore.GetByPartyAndID(context.Background(), testPartyB, aggressiveOrder.Id)
	assert.Nil(t, err)
	assert.Equal(t, aggressiveOrder.Id, orderAtPartyB.Id)

	// update order, parties should also be updated as its a pointer
	updatedAggressiveOrder := types.Order{
		Id:        "d41d8cd98f00b204e9800998ecf8427e",
		MarketID:  testMarket,
		PartyID:   testPartyB,
		Remaining: 0,
	}

	err = orderStore.SaveBatch([]types.Order{updatedAggressiveOrder})
	assert.NoError(t, err)
	orderAtPartyB, err = orderStore.GetByPartyAndID(context.Background(), testPartyB, aggressiveOrder.Id)
	assert.Nil(t, err)
	assert.Equal(t, updatedAggressiveOrder.Id, orderAtPartyB.Id)
}

func TestStorage_GetOrderByReference(t *testing.T) {
	config, err := storage.NewTestConfig()
	if err != nil {
		t.Fatalf("unable to setup badger dirs: %v", err)
	}

	log := logging.NewTestLogger()

	storage.FlushStores(log, config)
	newOrderStore, err := storage.NewOrders(log, config, func() {})
	assert.Nil(t, err)
	defer newOrderStore.Close()

	order := &types.Order{
		Id:          "d41d8cd98f00b204e9800998ecf8427b",
		MarketID:    testMarket,
		PartyID:     testPartyA,
		Side:        types.Side_SIDE_BUY,
		Price:       100,
		Size:        1000,
		Remaining:   0,
		TimeInForce: types.Order_TIF_GTC,
		CreatedAt:   0,
		Status:      types.Order_STATUS_ACTIVE,
		Reference:   "123123-34334343-1231231",
	}

	err = newOrderStore.SaveBatch([]types.Order{*order})
	assert.NoError(t, err)

	fetchedOrder, err := newOrderStore.GetByParty(context.Background(), testPartyA, 0, 1, true)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(fetchedOrder))
	assert.Equal(t, order.Id, fetchedOrder[0].Id)
}

func TestStorage_GetOrderByID(t *testing.T) {
	config, err := storage.NewTestConfig()
	if err != nil {
		t.Fatalf("unable to setup badger dirs: %v", err)
	}

	log := logging.NewTestLogger()

	storage.FlushStores(log, config)
	newOrderStore, err := storage.NewOrders(log, config, func() {})
	assert.Nil(t, err)
	defer newOrderStore.Close()

	id := "ALA-MA-KOTA"
	order := &types.Order{
		Id:          id,
		MarketID:    testMarket,
		PartyID:     testPartyA,
		Side:        types.Side_SIDE_BUY,
		Price:       100,
		Size:        1000,
		TimeInForce: types.Order_TIF_GTC,
		Status:      types.Order_STATUS_ACTIVE,
	}

	err = newOrderStore.SaveBatch([]types.Order{*order})
	assert.NoError(t, err)

	t.Run("basic happy path test", func(t *testing.T) {
		fetchedOrder, err := newOrderStore.GetByOrderID(context.Background(), id, nil)
		assert.NoError(t, err)
		assert.NotNil(t, fetchedOrder)
		assert.EqualValues(t, id, fetchedOrder.Id)
	})

	t.Run("negative test - empty id", func(t *testing.T) {
		fetchedOrder, err := newOrderStore.GetByOrderID(context.Background(), "", nil)
		assert.Nil(t, fetchedOrder)
		assert.Error(t, err)
		assert.EqualError(t, err, storage.ErrOrderDoesNotExistForID.Error())
	})
	t.Run("negative test - non-existing id", func(t *testing.T) {
		fetchedOrder, err := newOrderStore.GetByOrderID(context.Background(), id+id, nil)
		assert.Nil(t, fetchedOrder)
		assert.Error(t, err)
		assert.EqualError(t, err, storage.ErrOrderDoesNotExistForID.Error())
	})
}

func TestStorage_GetOrderByIDVersioning(t *testing.T) {
	config, err := storage.NewTestConfig()
	if err != nil {
		t.Fatalf("unable to setup badger dirs: %v", err)
	}

	log := logging.NewTestLogger()
	storage.FlushStores(log, config)
	newOrderStore, err := storage.NewOrders(log, config, func() {})
	assert.Nil(t, err)
	defer newOrderStore.Close()

	id := "KOTEK-KLOPOTEK"
	var version uint64 = execution.InitialOrderVersion

	orderV1 := &types.Order{
		Id:          id,
		MarketID:    testMarket,
		PartyID:     testPartyA,
		Side:        types.Side_SIDE_BUY,
		Price:       1,
		Size:        1,
		TimeInForce: types.Order_TIF_GTC,
		Status:      types.Order_STATUS_ACTIVE,
		Version:     version,
	}
	orderV2 := &types.Order{}
	*orderV2 = *orderV1
	version++
	orderV2.Version = version

	orderV3 := &types.Order{}
	*orderV3 = *orderV2
	version++
	orderV3.Version = version

	differentOrder := &types.Order{
		Id:          "d41d8cd98f00b204e9800998ecf8427c",
		MarketID:    testMarket,
		PartyID:     testPartyA,
		Side:        types.Side_SIDE_SELL,
		Price:       222,
		Size:        222,
		TimeInForce: types.Order_TIF_GTC,
		Status:      types.Order_STATUS_ACTIVE,
		Version:     execution.InitialOrderVersion,
	}
	anotherOrder := &types.Order{
		Id:          "000000000000000000000000000000",
		MarketID:    testMarket,
		PartyID:     testPartyA,
		Side:        types.Side_SIDE_SELL,
		Price:       222,
		Size:        222,
		TimeInForce: types.Order_TIF_GTC,
		Status:      types.Order_STATUS_ACTIVE,
		Version:     execution.InitialOrderVersion,
	}

	err = newOrderStore.SaveBatch([]types.Order{*orderV1, *orderV2, *differentOrder, *anotherOrder, *orderV3})
	assert.NoError(t, err)

	t.Run("test if can load distinc orders regardless of versioning", func(t *testing.T) {
		distinctOrders, err := newOrderStore.GetByParty(context.Background(), testPartyA, 0, 100, false)
		assert.NoError(t, err)
		assert.NotNil(t, distinctOrders)
		assert.Equal(t, 3, len(distinctOrders), "must be only 3 distinct orders")
		assert.NotEqual(t, distinctOrders[0].Id, distinctOrders[1].Id, distinctOrders[2].Id)
	})

	t.Run("test all order versions", func(t *testing.T) {
		allVersions, err := newOrderStore.GetAllVersionsByOrderID(context.Background(), id, 0, 100, false)
		assert.NoError(t, err)
		assert.NotNil(t, allVersions)
		assert.Equal(t, 3, len(allVersions))
		assert.NotEqual(t, allVersions[0].Version, allVersions[2].Version)
		assert.EqualValues(t, allVersions[0].Version+1, allVersions[1].Version)
		assert.EqualValues(t, execution.InitialOrderVersion, allVersions[0].Version)
	})

	t.Run("test if default order version is latest", func(t *testing.T) {
		fetchedOrder, err := newOrderStore.GetByOrderID(context.Background(), id, nil)
		assert.NoError(t, err)
		assert.NotNil(t, fetchedOrder)
		assert.Equal(t, id, fetchedOrder.Id)
		assert.EqualValues(t, version, fetchedOrder.Version)
	})

	t.Run("test if searching for invalid order version fails", func(t *testing.T) {
		invalidVersion := version * 100
		fetchedOrder, err := newOrderStore.GetByOrderID(context.Background(), id, &invalidVersion)
		assert.Error(t, err)
		assert.EqualError(t, err, storage.ErrOrderDoesNotExistForID.Error())
		assert.Nil(t, fetchedOrder)
	})

	t.Run("test if able to load middle order version", func(t *testing.T) {
		validVersion := version - 1
		fetchedOrder, err := newOrderStore.GetByOrderID(context.Background(), id, &validVersion)
		assert.NoError(t, err)
		assert.NotNil(t, fetchedOrder)
		assert.Equal(t, id, fetchedOrder.Id)
		assert.EqualValues(t, version-1, fetchedOrder.Version)
	})

	t.Run("test if able to load first order version", func(t *testing.T) {
		var initialVersion uint64 = execution.InitialOrderVersion
		fetchedOrder, err := newOrderStore.GetByOrderID(context.Background(), id, &initialVersion)
		assert.NoError(t, err)
		assert.NotNil(t, fetchedOrder)
		assert.Equal(t, id, fetchedOrder.Id)
		assert.EqualValues(t, execution.InitialOrderVersion, fetchedOrder.Version)
	})

	t.Run("test massive number of versions", func(t *testing.T) {

		orders := make([]types.Order, 0, 10000)
		for i := 0; i < 10000; i++ {
			orderV := &types.Order{}
			*orderV = *orderV1
			version++
			orderV.Version = version
			orders = append(orders, *orderV)
		}
		err = newOrderStore.SaveBatch(orders)
		assert.NoError(t, err)

		fetchedOrder, err := newOrderStore.GetByOrderID(context.Background(), id, nil)
		assert.NoError(t, err)
		assert.NotNil(t, fetchedOrder)
		assert.Equal(t, id, fetchedOrder.Id)
		assert.EqualValues(t, version, fetchedOrder.Version)

		var firstVersion uint64 = execution.InitialOrderVersion
		fetchedOrder, err = newOrderStore.GetByOrderID(context.Background(), id, &firstVersion)
		assert.NoError(t, err)
		assert.NotNil(t, fetchedOrder)
		assert.Equal(t, id, fetchedOrder.Id)
		assert.EqualValues(t, execution.InitialOrderVersion, fetchedOrder.Version)

		allVersions, err := newOrderStore.GetAllVersionsByOrderID(context.Background(), id, 0, 0, true)
		assert.NoError(t, err)
		assert.NotNil(t, allVersions)
		assert.Equal(t, len(orders)+3, len(allVersions))
	})
}
