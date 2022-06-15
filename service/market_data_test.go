package service_test

import (
	"context"
	"testing"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/data-node/service"
	"code.vegaprotocol.io/data-node/service/mocks"
	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/assert"
)

var (
	testData1      = entities.MarketData{Market: entities.NewMarketID("aa"), SeqNum: 1}
	testData2      = entities.MarketData{Market: entities.NewMarketID("bb"), SeqNum: 2}
	testData3      = entities.MarketData{Market: entities.NewMarketID("aa"), SeqNum: 3}
	testData4      = entities.MarketData{Market: entities.NewMarketID("cc"), SeqNum: 4}
	sortMarketData = cmpopts.SortSlices(func(a, b entities.MarketData) bool { return a.SeqNum < b.SeqNum })
)

func TestInitialise(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Set up mock store to have some initial data in it when we initialise
	store := mocks.NewMockMarketDataStore(ctrl)
	store.EXPECT().GetMarketsData(gomock.Any()).Return([]entities.MarketData{
		testData1,
		testData2,
	}, nil)

	// Initialise and check that we get that data out of the cache (e.g. no other calls to store)
	svc := service.NewMarketData(store, logging.NewTestLogger())
	svc.Initialise(ctx)

	allData, err := svc.GetMarketsData(ctx)
	assert.NoError(t, err)
	assert.Empty(t, cmp.Diff(allData, []entities.MarketData{testData1, testData2}, sortMarketData))
}

func TestAdd(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Set up mock store to have some initial data in it when we initialise
	store := mocks.NewMockMarketDataStore(ctrl)
	store.EXPECT().GetMarketsData(gomock.Any()).Return([]entities.MarketData{
		testData1,
		testData2,
	}, nil)

	// Expect a couple of calls to Add, we don't need to do anything with them as service should cache
	store.EXPECT().Add(gomock.Any()).Return(nil).Times(2)

	// Make service, initialise (mock store has 2 records in it), and add two more bits of data.
	svc := service.NewMarketData(store, logging.NewTestLogger())
	svc.Initialise(ctx)
	svc.Add(&testData3)
	svc.Add(&testData4)

	// testData3 has the same market as testData1 so check we replaced it. Expect no calls to
	// the store as this should be in the service cache.
	allData, err := svc.GetMarketsData(ctx)
	assert.NoError(t, err)
	assert.Empty(t, cmp.Diff(allData, []entities.MarketData{testData2, testData3, testData4}, sortMarketData))

	// Then try getting for just one market. Should be cached.
	oneData, err := svc.GetMarketDataByID(ctx, "aa")
	assert.NoError(t, err)
	assert.Equal(t, testData3, oneData)
}
