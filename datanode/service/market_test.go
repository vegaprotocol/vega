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

package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/assert"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/service"
	"code.vegaprotocol.io/vega/datanode/service/mocks"
)

var (
	testMarket1 = entities.Market{ID: entities.MarketID("aa"), VegaTime: time.Unix(0, 1)}
	testMarket2 = entities.Market{ID: entities.MarketID("bb"), VegaTime: time.Unix(0, 2)}
	testMarket3 = entities.Market{ID: entities.MarketID("aa"), VegaTime: time.Unix(0, 3)}
	testMarket4 = entities.Market{ID: entities.MarketID("cc"), VegaTime: time.Unix(0, 4)}
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
	svc := service.NewMarkets(store)
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
	svc := service.NewMarkets(store)
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
