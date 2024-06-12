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

package sqlstore_test

import (
	"context"
	"testing"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/sqlstore"
	"code.vegaprotocol.io/vega/libs/num"

	"github.com/georgysavva/scany/pgxscan"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupPartyVestingBalanceTest(t *testing.T) (*sqlstore.Blocks, *sqlstore.PartyVestingBalance) {
	t.Helper()
	bs := sqlstore.NewBlocks(connectionSource)
	plbs := sqlstore.NewPartyVestingBalances(connectionSource)

	return bs, plbs
}

func TestPartyVestingBalance_Add(t *testing.T) {
	bs, plbs := setupPartyVestingBalanceTest(t)

	ctx := tempTransaction(t)

	var partyVestingBalances []entities.PartyVestingBalance
	var partyVestingBalancesCurrent []entities.PartyVestingBalance

	err := pgxscan.Select(ctx, connectionSource, &partyVestingBalances, "SELECT * from party_vesting_balances")
	require.NoError(t, err)

	assert.Len(t, partyVestingBalances, 0)

	err = pgxscan.Select(ctx, connectionSource, &partyVestingBalancesCurrent, "SELECT * from party_vesting_balances_current")
	require.NoError(t, err)

	assert.Len(t, partyVestingBalancesCurrent, 0)

	block := addTestBlock(t, ctx, bs)

	t.Run("Add should insert a new record into the party_vesting_balances table", func(t *testing.T) {
		want := entities.PartyVestingBalance{
			PartyID:  "deadbeef01",
			AssetID:  "cafedaad01",
			AtEpoch:  200,
			Balance:  num.DecimalFromInt64(10000000000),
			VegaTime: block.VegaTime,
		}

		err := plbs.Add(ctx, want)
		require.NoError(t, err)

		err = pgxscan.Select(ctx, connectionSource, &partyVestingBalances, "SELECT * from party_vesting_balances")
		require.NoError(t, err)

		assert.Len(t, partyVestingBalances, 1)
		assert.Equal(t, want, partyVestingBalances[0])

		t.Run("And a record into the party_vesting_balances_current table if it doesn't already exist", func(t *testing.T) {
			err = pgxscan.Select(ctx, connectionSource, &partyVestingBalancesCurrent, "SELECT * from party_vesting_balances_current")
			require.NoError(t, err)

			assert.Len(t, partyVestingBalancesCurrent, 1)
			assert.Equal(t, want, partyVestingBalancesCurrent[0])
		})

		t.Run("And update the record in the party_vesting_balances_current table if the party and asset already exists", func(t *testing.T) {
			block = addTestBlock(t, ctx, bs)
			want2 := entities.PartyVestingBalance{
				PartyID:  "deadbeef01",
				AssetID:  "cafedaad01",
				AtEpoch:  250,
				Balance:  num.DecimalFromInt64(15000000000),
				VegaTime: block.VegaTime,
			}

			err = plbs.Add(ctx, want2)
			err = pgxscan.Select(ctx, connectionSource, &partyVestingBalances, "SELECT * from party_vesting_balances order by vega_time")
			require.NoError(t, err)

			assert.Len(t, partyVestingBalances, 2)
			assert.Equal(t, want, partyVestingBalances[0])
			assert.Equal(t, want2, partyVestingBalances[1])

			err = pgxscan.Select(ctx, connectionSource, &partyVestingBalancesCurrent, "SELECT * from party_vesting_balances_current")
			require.NoError(t, err)

			assert.Len(t, partyVestingBalancesCurrent, 1)
			assert.Equal(t, want2, partyVestingBalancesCurrent[0])
		})
	})
}

func setupHistoricPartyVestingBalances(t *testing.T, ctx context.Context, bs *sqlstore.Blocks, plbs *sqlstore.PartyVestingBalance) []entities.PartyVestingBalance {
	t.Helper()

	parties := []string{
		"deadbeef01",
		"deadbeef02",
		"deadbeef03",
	}

	assets := []string{
		"cafedaad01",
		"cafedaad02",
	}

	currentBalances := make([]entities.PartyVestingBalance, 0)

	for i := 0; i < 3; i++ { // versions
		block := addTestBlock(t, ctx, bs)
		for _, party := range parties {
			for _, asset := range assets {
				balance := entities.PartyVestingBalance{
					PartyID:  entities.PartyID(party),
					AssetID:  entities.AssetID(asset),
					AtEpoch:  100 + uint64(i),
					Balance:  num.DecimalFromInt64(10000000000 + int64(i*10000000)),
					VegaTime: block.VegaTime,
				}
				err := plbs.Add(ctx, balance)
				require.NoError(t, err)
				if i == 2 {
					currentBalances = append(currentBalances, balance)
				}
			}
		}
	}
	return currentBalances
}

func TestPartyVestingBalance_Get(t *testing.T) {
	t.Run("Get should return all current record if party and asset is not provided", testPartyVestingBalanceGetAll)
	t.Run("Get should return all current record for a party if it is provided", testPartyVestingBalanceGetByParty)
	t.Run("Get should return all current records for an asset if it is provided", testPartyVestingBalancesGetByAsset)
	t.Run("Get should return all current records for a party and asset", testPartyVestingBalancesGetByPartyAndAsset)
}

func testPartyVestingBalanceGetAll(t *testing.T) {
	bs, plvs := setupPartyVestingBalanceTest(t)

	ctx := tempTransaction(t)

	currentBalances := setupHistoricPartyVestingBalances(t, ctx, bs, plvs)

	balances, err := plvs.Get(ctx, nil, nil)
	require.NoError(t, err)

	assert.Len(t, balances, len(currentBalances))
	assert.Equal(t, currentBalances, balances)
}

func testPartyVestingBalanceGetByParty(t *testing.T) {
	bs, plvs := setupPartyVestingBalanceTest(t)

	ctx := tempTransaction(t)

	currentBalances := setupHistoricPartyVestingBalances(t, ctx, bs, plvs)
	partyID := entities.PartyID("deadbeef01")

	want := make([]entities.PartyVestingBalance, 0)

	for _, balance := range currentBalances {
		if balance.PartyID == partyID {
			want = append(want, balance)
		}
	}

	balances, err := plvs.Get(ctx, &partyID, nil)
	require.NoError(t, err)

	assert.Len(t, balances, len(want))
	assert.Equal(t, want, balances)
}

func testPartyVestingBalancesGetByAsset(t *testing.T) {
	bs, plvs := setupPartyVestingBalanceTest(t)

	ctx := tempTransaction(t)

	currentBalances := setupHistoricPartyVestingBalances(t, ctx, bs, plvs)
	assetID := entities.AssetID("cafedaad01")

	want := make([]entities.PartyVestingBalance, 0)

	for _, balance := range currentBalances {
		if balance.AssetID == assetID {
			want = append(want, balance)
		}
	}

	balances, err := plvs.Get(ctx, nil, &assetID)
	require.NoError(t, err)

	assert.Len(t, balances, len(want))
	assert.Equal(t, want, balances)
}

func testPartyVestingBalancesGetByPartyAndAsset(t *testing.T) {
	bs, plvs := setupPartyVestingBalanceTest(t)

	ctx := tempTransaction(t)

	currentBalances := setupHistoricPartyVestingBalances(t, ctx, bs, plvs)
	partyID := entities.PartyID("deadbeef01")
	assetID := entities.AssetID("cafedaad01")

	want := make([]entities.PartyVestingBalance, 0)

	for _, balance := range currentBalances {
		if balance.PartyID == partyID && balance.AssetID == assetID {
			want = append(want, balance)
		}
	}

	balances, err := plvs.Get(ctx, &partyID, &assetID)
	require.NoError(t, err)

	assert.Len(t, balances, len(want))
	assert.Equal(t, want, balances)
}
