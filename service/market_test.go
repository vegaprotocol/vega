package service_test

import (
	"context"
	"testing"
	"time"

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
	testMarket1 = entities.Market{ID: entities.NewMarketID("aa"), VegaTime: time.Unix(0, 1)}
	testMarket2 = entities.Market{ID: entities.NewMarketID("bb"), VegaTime: time.Unix(0, 2)}
	testMarket3 = entities.Market{ID: entities.NewMarketID("aa"), VegaTime: time.Unix(0, 3)}
	testMarket4 = entities.Market{ID: entities.NewMarketID("cc"), VegaTime: time.Unix(0, 4)}
	sortMarkets = cmpopts.SortSlices(func(a, b entities.Market) bool { return a.VegaTime.Before(b.VegaTime) })
)

func TestMarketInitialise(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Set up mock store to have some initial data in it when we initialise
	store := mocks.NewMockMarketStore(ctrl)
	store.EXPECT().GetAll(gomock.Any(), gomock.Any()).Return([]entities.Market{
		testMarket1,
		testMarket2,
	}, nil)

	// Initialise and check that we get that data out of the cache (e.g. no other calls to store)
	svc := service.NewMarkets(store, logging.NewTestLogger())
	svc.Initialise(ctx)

	allData, err := svc.GetAll(ctx, entities.OffsetPagination{})
	assert.NoError(t, err)
	assert.Empty(t, cmp.Diff(allData, []entities.Market{testMarket1, testMarket2}, sortMarkets))
}

func TestMarketUpsert(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Set up mock store to have some initial data in it when we initialise
	store := mocks.NewMockMarketStore(ctrl)
	store.EXPECT().GetAll(gomock.Any(), gomock.Any()).Return([]entities.Market{
		testMarket1,
		testMarket2,
	}, nil)

	// Expect a couple of calls to Add, we don't need to do anything with them as service should cache
	store.EXPECT().Upsert(ctx, &testMarket3).Return(nil)
	store.EXPECT().Upsert(ctx, &testMarket4).Return(nil)

	// Make service; initialise (mock store has 2 records in it); and add two more bits of data.
	svc := service.NewMarkets(store, logging.NewTestLogger())
	svc.Initialise(ctx)
	svc.Upsert(ctx, &testMarket3)
	svc.Upsert(ctx, &testMarket4)

	// testMarket3 has the same id as testMarket1 so check we replaced it. Expect no calls to
	// the store as this should be in the service cache.
	allData, err := svc.GetAll(ctx, entities.OffsetPagination{})
	assert.NoError(t, err)
	assert.Empty(t, cmp.Diff(allData, []entities.Market{testMarket2, testMarket3, testMarket4}, sortMarkets))

	// Then try getting for just one market. Should be cached.
	oneData, err := svc.GetByID(ctx, "aa")
	assert.NoError(t, err)
	assert.Equal(t, testMarket3, oneData)
}
