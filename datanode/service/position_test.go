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

package service_test

import (
	"context"
	"fmt"
	"testing"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/service"
	"code.vegaprotocol.io/vega/datanode/service/mocks"
	"code.vegaprotocol.io/vega/logging"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

var (
	market1ID     = entities.MarketID("aa")
	market2ID     = entities.MarketID("bb")
	party1ID      = entities.PartyID("cc")
	party2ID      = entities.PartyID("dd")
	testPosition1 = entities.Position{MarketID: market1ID, PartyID: party1ID, OpenVolume: 1}
	testPosition2 = entities.Position{MarketID: market2ID, PartyID: party2ID, OpenVolume: 2}
	testPosition3 = entities.Position{MarketID: market1ID, PartyID: party1ID, OpenVolume: 10}
)

func TestPositionAdd(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)

	// Make a new store and add a position to it
	store := mocks.NewMockPositionStore(ctrl)
	store.EXPECT().Add(ctx, gomock.Any()).Return(nil).Times(3)

	svc := service.NewPosition(store, logging.NewTestLogger())
	svc.Add(ctx, testPosition1)
	svc.Add(ctx, testPosition2)
	svc.Add(ctx, testPosition3)

	// We don't expect a call to the store's GetByMarketAndParty() method as it should be cached
	// testPosition3 has the same market/party as testPosition1 so should replace it
	fetched, err := svc.GetByMarketAndParty(ctx, market1ID.String(), party1ID.String())
	assert.NoError(t, err)
	assert.Equal(t, testPosition3, fetched)

	fetched, err = svc.GetByMarketAndParty(ctx, market2ID.String(), party2ID.String())
	assert.NoError(t, err)
	assert.Equal(t, testPosition2, fetched)
}

func TestCache(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)

	// Simulate store with one position in it
	store := mocks.NewMockPositionStore(ctrl)
	store.EXPECT().GetByMarketAndParty(ctx, market1ID.String(), party1ID.String()).Return(testPosition1, nil)

	svc := service.NewPosition(store, logging.NewTestLogger())

	// First time should call through to the store
	fetched, err := svc.GetByMarketAndParty(ctx, market1ID.String(), party1ID.String())
	assert.NoError(t, err)
	assert.Equal(t, testPosition1, fetched)

	// Second time should use cache (we only EXPECT one call above)
	fetched, err = svc.GetByMarketAndParty(ctx, market1ID.String(), party1ID.String())
	assert.NoError(t, err)
	assert.Equal(t, testPosition1, fetched)
}

func TestCacheError(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	notFoundErr := fmt.Errorf("nothing here i'm afraid")

	// Simulate store with no positions in it
	store := mocks.NewMockPositionStore(ctrl)
	store.EXPECT().GetByMarketAndParty(ctx, market1ID.String(), party1ID.String()).Times(2).Return(
		entities.Position{},
		notFoundErr,
	)

	svc := service.NewPosition(store, logging.NewTestLogger())

	// First time should call through to the store and return error
	fetched, err := svc.GetByMarketAndParty(ctx, market1ID.String(), party1ID.String())
	assert.ErrorIs(t, notFoundErr, err)
	assert.Equal(t, entities.Position{}, fetched)

	// Second time should use cache but still get same error
	fetched, err = svc.GetByMarketAndParty(ctx, market1ID.String(), party1ID.String())
	assert.ErrorIs(t, notFoundErr, err)
	assert.Equal(t, entities.Position{}, fetched)
}
