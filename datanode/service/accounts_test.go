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
	"sync"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/service"
	"code.vegaprotocol.io/vega/datanode/service/mocks"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/protos/vega"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestObserveAccountBalances(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	balanceStore := mocks.NewMockBalanceStore(ctrl)
	accountStore := mocks.NewMockAccountStore(ctrl)
	log := logging.NewTestLogger()
	accounts := service.NewAccount(accountStore, balanceStore, log)

	ctx := context.Background()

	partyIDs := map[string]string{
		"party_id":  "parent_party_id",
		"party_id2": "parent_party_id2",
		"party_id3": "parent_party_id3",
	}

	balances := []entities.AccountBalance{}

	for partyID := range partyIDs {
		balances = append(balances, entities.AccountBalance{
			Account: &entities.Account{
				PartyID:  entities.PartyID(partyID),
				AssetID:  "asset_id",
				MarketID: "market_id",
				Type:     vega.AccountType_ACCOUNT_TYPE_GENERAL,
			},
		})
	}

	balances = append(balances, []entities.AccountBalance{
		{
			Account: &entities.Account{
				PartyID:  "party_id",
				AssetID:  "asset_id",
				MarketID: "market_id2",
				Type:     vega.AccountType_ACCOUNT_TYPE_GENERAL,
			},
		},
		{
			Account: &entities.Account{
				PartyID:  "party_id10",
				AssetID:  "asset_id",
				MarketID: "market_id",
				Type:     vega.AccountType_ACCOUNT_TYPE_GENERAL,
			},
		},
		{
			Account: &entities.Account{
				PartyID:  "party_id",
				AssetID:  "asset_id",
				MarketID: "market_id50",
				Type:     vega.AccountType_ACCOUNT_TYPE_GENERAL,
			},
		},
		{
			Account: &entities.Account{
				PartyID:  "party_id",
				AssetID:  "asset_id",
				MarketID: "market_id",
				Type:     vega.AccountType_ACCOUNT_TYPE_GLOBAL_REWARD,
			},
		},
	}...)

	accountsChan, _ := accounts.ObserveAccountBalances(ctx, 20, "market_id", "asset_id",
		vega.AccountType_ACCOUNT_TYPE_GENERAL, partyIDs)

	balanceStore.EXPECT().Flush(ctx).Return(balances, nil).Times(1)

	// first 3 balances should be received
	expectedBalances := balances[:3]

	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()

		receivedBalances := <-accountsChan
		require.Equal(t, len(expectedBalances), len(receivedBalances))

		for i, expected := range expectedBalances {
			require.Equal(t, expected.PartyID, receivedBalances[i].PartyID)
			require.Equal(t, expected.MarketID, receivedBalances[i].MarketID)
			require.Equal(t, expected.AssetID, receivedBalances[i].AssetID)
		}
	}()

	time.Sleep(500 * time.Millisecond)
	// by calling Flush we can mimic sending the balances to the channel and receiving them in Observe method
	require.NoError(t, accounts.Flush(ctx))
	wg.Wait()

	var remainingBalances []entities.AccountBalance
	select {
	case balances := <-accountsChan:
		remainingBalances = balances
	default:
	}

	require.Len(t, remainingBalances, 0)
}
