// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
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
	"fmt"
	"testing"

	"code.vegaprotocol.io/data-node/datanode/entities"
	"code.vegaprotocol.io/data-node/datanode/service"
	"code.vegaprotocol.io/data-node/datanode/service/mocks"
	"code.vegaprotocol.io/data-node/logging"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

var (
	market1ID     = entities.NewMarketID("aa")
	market2ID     = entities.NewMarketID("bb")
	party1ID      = entities.NewPartyID("cc")
	party2ID      = entities.NewPartyID("dd")
	testPosition1 = entities.Position{MarketID: market1ID, PartyID: party1ID, OpenVolume: 1}
	testPosition2 = entities.Position{MarketID: market2ID, PartyID: party2ID, OpenVolume: 2}
	testPosition3 = entities.Position{MarketID: market1ID, PartyID: party1ID, OpenVolume: 10}
)

func TestPositionAdd(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Make a new store and add a position to it
	store := mocks.NewMockPositionStore(ctrl)
	store.EXPECT().Add(ctx, gomock.Any()).Return(nil).Times(3)

	svc := service.NewPosition(store, logging.NewTestLogger())
	svc.Add(ctx, testPosition1)
	svc.Add(ctx, testPosition2)
	svc.Add(ctx, testPosition3)

	// We don't expect a call to the store's GetByMarketAndParty() method as it should be cached
	// testPosition3 has the same market/party as testPosition1 so should replace it
	fetched, err := svc.GetByMarketAndParty(ctx, market1ID, party1ID)
	assert.NoError(t, err)
	assert.Equal(t, testPosition3, fetched)

	fetched, err = svc.GetByMarketAndParty(ctx, market2ID, party2ID)
	assert.NoError(t, err)
	assert.Equal(t, testPosition2, fetched)
}

func TestCache(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Simulate store with one position in it
	store := mocks.NewMockPositionStore(ctrl)
	store.EXPECT().GetByMarketAndParty(ctx, market1ID, party1ID).Return(testPosition1, nil)

	svc := service.NewPosition(store, logging.NewTestLogger())

	// First time should call through to the store
	fetched, err := svc.GetByMarketAndParty(ctx, market1ID, party1ID)
	assert.NoError(t, err)
	assert.Equal(t, testPosition1, fetched)

	// Second time should use cache (we only EXPECT one call above)
	fetched, err = svc.GetByMarketAndParty(ctx, market1ID, party1ID)
	assert.NoError(t, err)
	assert.Equal(t, testPosition1, fetched)
}

func TestCacheError(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	notFoundErr := fmt.Errorf("nothing here i'm afraid")
	defer ctrl.Finish()

	// Simulate store with no positions in it
	store := mocks.NewMockPositionStore(ctrl)
	store.EXPECT().GetByMarketAndParty(ctx, market1ID, party1ID).Return(
		entities.Position{},
		notFoundErr,
	)

	svc := service.NewPosition(store, logging.NewTestLogger())

	// First time should call through to the store and return error
	fetched, err := svc.GetByMarketAndParty(ctx, market1ID, party1ID)
	assert.ErrorIs(t, notFoundErr, err)
	assert.Equal(t, entities.Position{}, fetched)

	// Second time should use cache but still get same error
	fetched, err = svc.GetByMarketAndParty(ctx, market1ID, party1ID)
	assert.ErrorIs(t, notFoundErr, err)
	assert.Equal(t, entities.Position{}, fetched)

}
