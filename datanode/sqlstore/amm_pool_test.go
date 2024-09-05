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
	"math/rand"
	"sort"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/sqlstore"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/ptr"

	"github.com/georgysavva/scany/pgxscan"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAMMPool_Upsert(t *testing.T) {
	ctx := tempTransaction(t)

	bs := sqlstore.NewBlocks(connectionSource)
	ps := sqlstore.NewAMMPools(connectionSource)
	block := addTestBlock(t, ctx, bs)

	partyID := entities.PartyID(GenerateID())
	marketID := entities.MarketID(GenerateID())
	poolID := entities.AMMPoolID(GenerateID())
	ammPartyID := entities.PartyID(GenerateID())

	t.Run("Upsert statuses", func(t *testing.T) {
		upsertTests := []struct {
			Status entities.AMMStatus
			Reason entities.AMMStatusReason
		}{
			{entities.AMMStatusActive, entities.AMMStatusReasonUnspecified},
			{entities.AMMStatusStopped, entities.AMMStatusReasonUnspecified},
			{entities.AMMStatusCancelled, entities.AMMStatusReasonUnspecified},
			{entities.AMMStatusRejected, entities.AMMStatusReasonCancelledByParty},
			{entities.AMMStatusRejected, entities.AMMStatusReasonCannotRebase},
			{entities.AMMStatusRejected, entities.AMMStatusReasonMarketClosed},
			{entities.AMMStatusRejected, entities.AMMStatusReasonCannotFillCommitment},
			{entities.AMMStatusRejected, entities.AMMStatusReasonCommitmentTooLow},
			{entities.AMMStatusRejected, entities.AMMStatusReasonPartyAlreadyOwnsAPool},
			{entities.AMMStatusRejected, entities.AMMStatusReasonPartyClosedOut},
		}

		upsertTime := block.VegaTime
		for i, test := range upsertTests {
			upsertTime = upsertTime.Add(time.Duration(i) * time.Minute)
			pool := entities.AMMPool{
				PartyID:                        partyID,
				MarketID:                       marketID,
				ID:                             poolID,
				AmmPartyID:                     ammPartyID,
				Commitment:                     num.DecimalFromInt64(100),
				Status:                         test.Status,
				StatusReason:                   test.Reason,
				ParametersBase:                 num.DecimalFromInt64(100),
				ParametersLowerBound:           ptr.From(num.DecimalFromInt64(100)),
				ParametersUpperBound:           ptr.From(num.DecimalFromInt64(100)),
				ParametersLeverageAtLowerBound: ptr.From(num.DecimalFromInt64(100)),
				ParametersLeverageAtUpperBound: ptr.From(num.DecimalFromInt64(100)),
				CreatedAt:                      block.VegaTime,
				LastUpdated:                    upsertTime,
				LowerVirtualLiquidity:          num.DecimalOne(),
				UpperVirtualLiquidity:          num.DecimalOne(),
				LowerTheoreticalPosition:       num.DecimalOne(),
				UpperTheoreticalPosition:       num.DecimalOne(),
			}
			require.NoError(t, ps.Upsert(ctx, pool))
			var upserted entities.AMMPool
			require.NoError(t, pgxscan.Get(
				ctx,
				connectionSource,
				&upserted,
				`SELECT * FROM amms WHERE party_id = $1 AND market_id = $2 AND id = $3 AND amm_party_id = $4`,
				partyID, marketID, poolID, ammPartyID))
			assert.Equal(t, pool, upserted)
		}
	})

	t.Run("Upsert with different commitments and bounds", func(t *testing.T) {
		amounts := []num.Decimal{
			num.DecimalFromInt64(100),
			num.DecimalFromInt64(200),
			num.DecimalFromInt64(300),
		}
		upsertTime := block.VegaTime
		for i, amount := range amounts {
			upsertTime = upsertTime.Add(time.Duration(i) * time.Minute)
			pool := entities.AMMPool{
				PartyID:                        partyID,
				MarketID:                       marketID,
				ID:                             poolID,
				AmmPartyID:                     ammPartyID,
				Commitment:                     amount,
				Status:                         entities.AMMStatusActive,
				StatusReason:                   entities.AMMStatusReasonUnspecified,
				ParametersBase:                 amount,
				ParametersLowerBound:           ptr.From(amount),
				ParametersUpperBound:           ptr.From(amount),
				ParametersLeverageAtLowerBound: ptr.From(amount),
				ParametersLeverageAtUpperBound: ptr.From(amount),
				CreatedAt:                      block.VegaTime,
				LastUpdated:                    upsertTime,
				LowerVirtualLiquidity:          num.DecimalOne(),
				UpperVirtualLiquidity:          num.DecimalOne(),
				LowerTheoreticalPosition:       num.DecimalOne(),
				UpperTheoreticalPosition:       num.DecimalOne(),
			}
			require.NoError(t, ps.Upsert(ctx, pool))
			var upserted entities.AMMPool
			require.NoError(t, pgxscan.Get(
				ctx,
				connectionSource,
				&upserted,
				`SELECT * FROM amms WHERE party_id = $1 AND market_id = $2 AND id = $3 AND amm_party_id = $4`,
				partyID, marketID, poolID, ammPartyID))
			assert.Equal(t, pool, upserted)
		}
	})

	t.Run("Upsert with empty leverages", func(t *testing.T) {
		upsertTime := block.VegaTime

		pool := entities.AMMPool{
			PartyID:                        partyID,
			MarketID:                       marketID,
			ID:                             poolID,
			AmmPartyID:                     ammPartyID,
			Commitment:                     num.DecimalFromInt64(100),
			Status:                         entities.AMMStatusActive,
			StatusReason:                   entities.AMMStatusReasonUnspecified,
			ParametersBase:                 num.DecimalFromInt64(1800),
			ParametersLowerBound:           ptr.From(num.DecimalFromInt64(2000)),
			ParametersUpperBound:           ptr.From(num.DecimalFromInt64(2200)),
			ParametersLeverageAtLowerBound: nil,
			ParametersLeverageAtUpperBound: nil,
			CreatedAt:                      block.VegaTime,
			LastUpdated:                    upsertTime,
			LowerVirtualLiquidity:          num.DecimalOne(),
			UpperVirtualLiquidity:          num.DecimalOne(),
			LowerTheoreticalPosition:       num.DecimalOne(),
			UpperTheoreticalPosition:       num.DecimalOne(),
		}
		require.NoError(t, ps.Upsert(ctx, pool))
		var upserted entities.AMMPool
		require.NoError(t, pgxscan.Get(
			ctx,
			connectionSource,
			&upserted,
			`SELECT * FROM amms WHERE party_id = $1 AND market_id = $2 AND id = $3 AND amm_party_id = $4`,
			partyID, marketID, poolID, ammPartyID))
		assert.Equal(t, pool, upserted)
	})
}

type partyAccounts struct {
	PartyID    entities.PartyID
	AMMPartyID entities.PartyID
}

func setupAMMPoolsTest(ctx context.Context, t *testing.T) (
	*sqlstore.AMMPools, []entities.AMMPool, []partyAccounts, []entities.MarketID, []entities.AMMPoolID,
) {
	t.Helper()
	const (
		partyCount  = 5 // every party will have a derived-party associated for AMM and this derived-party underlies all the AMM pools that are created
		marketCount = 10
		poolCount   = 10
	)

	bs := sqlstore.NewBlocks(connectionSource)
	ps := sqlstore.NewAMMPools(connectionSource)

	block := addTestBlock(t, tempTransaction(t), bs)

	pools := make([]entities.AMMPool, 0, partyCount*marketCount*poolCount)
	parties := make([]partyAccounts, 0, partyCount)
	markets := make([]entities.MarketID, 0, marketCount)
	poolIDs := make([]entities.AMMPoolID, 0, poolCount)

	for i := 0; i < partyCount; i++ {
		partyID := entities.PartyID(GenerateID())
		ammPartyID := entities.PartyID(GenerateID())

		parties = append(parties, partyAccounts{PartyID: partyID, AMMPartyID: ammPartyID})
		for j := 0; j < marketCount; j++ {
			marketID := entities.MarketID(GenerateID())
			markets = append(markets, marketID)
			for k := 0; k < poolCount; k++ {
				poolID := entities.AMMPoolID(GenerateID())
				poolIDs = append(poolIDs, poolID)
				status := entities.AMMStatusActive
				statusReason := entities.AMMStatusReasonUnspecified
				if (i+j+k)%2 == 0 {
					status = entities.AMMStatusStopped
					statusReason = entities.AMMStatusReasonCancelledByParty
				}
				pool := entities.AMMPool{
					PartyID:                        partyID,
					MarketID:                       marketID,
					ID:                             poolID,
					AmmPartyID:                     ammPartyID,
					Commitment:                     num.DecimalFromInt64(100),
					Status:                         status,
					StatusReason:                   statusReason,
					ParametersBase:                 num.DecimalFromInt64(100),
					ParametersLowerBound:           ptr.From(num.DecimalFromInt64(100)),
					ParametersUpperBound:           ptr.From(num.DecimalFromInt64(100)),
					ParametersLeverageAtLowerBound: ptr.From(num.DecimalFromInt64(100)),
					ParametersLeverageAtUpperBound: ptr.From(num.DecimalFromInt64(100)),
					CreatedAt:                      block.VegaTime,
					LastUpdated:                    block.VegaTime,
					LowerVirtualLiquidity:          num.DecimalOne(),
					UpperVirtualLiquidity:          num.DecimalOne(),
					LowerTheoreticalPosition:       num.DecimalOne(),
					UpperTheoreticalPosition:       num.DecimalOne(),
				}
				require.NoError(t, ps.Upsert(ctx, pool))
				pools = append(pools, pool)
			}
		}
	}

	pools = orderPools(pools)

	return ps, pools, parties, markets, poolIDs
}

func orderPools(pools []entities.AMMPool) []entities.AMMPool {
	sort.Slice(pools, func(i, j int) bool {
		return pools[i].CreatedAt.After(pools[j].CreatedAt) ||
			(pools[i].CreatedAt == pools[j].CreatedAt && pools[i].PartyID < pools[j].PartyID) ||
			(pools[i].CreatedAt == pools[j].CreatedAt && pools[i].PartyID == pools[j].PartyID && pools[i].AmmPartyID < pools[j].AmmPartyID) ||
			(pools[i].CreatedAt == pools[j].CreatedAt && pools[i].PartyID == pools[j].PartyID && pools[i].AmmPartyID == pools[j].AmmPartyID && pools[i].MarketID < pools[j].MarketID) ||
			(pools[i].CreatedAt == pools[j].CreatedAt && pools[i].PartyID == pools[j].PartyID && pools[i].AmmPartyID == pools[j].AmmPartyID && pools[i].MarketID == pools[j].MarketID && pools[i].ID <= pools[j].ID)
	})

	return pools
}

func TestAMMPools_ListAll(t *testing.T) {
	ctx := tempTransaction(t)

	ps, pools, _, _, _ := setupAMMPoolsTest(ctx, t)

	t.Run("Should return all pools if no pagination is provided", func(t *testing.T) {
		pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, true)
		require.NoError(t, err)
		listedPools, pageInfo, err := ps.ListAll(ctx, false, pagination)
		require.NoError(t, err)
		assert.Equal(t, len(pools), len(listedPools))
		assert.Equal(t, pools, listedPools)
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     false,
			HasPreviousPage: false,
			StartCursor:     pools[0].Cursor().Encode(),
			EndCursor:       pools[len(pools)-1].Cursor().Encode(),
		}, pageInfo)
	})

	t.Run("Should return the first page of pools", func(t *testing.T) {
		pagination, err := entities.NewCursorPagination(ptr.From(int32(5)), nil, nil, nil, true)
		require.NoError(t, err)
		listedPools, pageInfo, err := ps.ListAll(ctx, false, pagination)
		require.NoError(t, err)
		assert.Equal(t, 5, len(listedPools))
		assert.Equal(t, pools[:5], listedPools)
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     true,
			HasPreviousPage: false,
			StartCursor:     pools[0].Cursor().Encode(),
			EndCursor:       pools[4].Cursor().Encode(),
		}, pageInfo)
	})

	t.Run("Should return the last page of pools", func(t *testing.T) {
		pagination, err := entities.NewCursorPagination(nil, nil, ptr.From(int32(5)), nil, true)
		require.NoError(t, err)
		listedPools, pageInfo, err := ps.ListAll(ctx, false, pagination)
		require.NoError(t, err)
		assert.Equal(t, 5, len(listedPools))
		assert.Equal(t, pools[len(pools)-5:], listedPools)
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     false,
			HasPreviousPage: true,
			StartCursor:     pools[len(pools)-5].Cursor().Encode(),
			EndCursor:       pools[len(pools)-1].Cursor().Encode(),
		}, pageInfo)
	})

	t.Run("Should return the requested page when paging forward", func(t *testing.T) {
		pagination, err := entities.NewCursorPagination(ptr.From(int32(5)), ptr.From(pools[20].Cursor().Encode()), nil, nil, true)
		require.NoError(t, err)
		listedPools, pageInfo, err := ps.ListAll(ctx, false, pagination)
		require.NoError(t, err)
		assert.Equal(t, 5, len(listedPools))
		assert.Equal(t, pools[21:26], listedPools)
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     true,
			HasPreviousPage: true,
			StartCursor:     pools[21].Cursor().Encode(),
			EndCursor:       pools[25].Cursor().Encode(),
		}, pageInfo)
	})

	t.Run("Should return the request page when paging backward", func(t *testing.T) {
		pagination, err := entities.NewCursorPagination(nil, nil, ptr.From(int32(5)), ptr.From(pools[20].Cursor().Encode()), true)
		require.NoError(t, err)
		listedPools, pageInfo, err := ps.ListAll(ctx, false, pagination)
		require.NoError(t, err)
		assert.Equal(t, 5, len(listedPools))
		assert.Equal(t, pools[15:20], listedPools)
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     true,
			HasPreviousPage: true,
			StartCursor:     pools[15].Cursor().Encode(),
			EndCursor:       pools[19].Cursor().Encode(),
		}, pageInfo)
	})
}

func filterPools(pools []entities.AMMPool, filter func(entities.AMMPool) bool) []entities.AMMPool {
	filtered := make([]entities.AMMPool, 0, len(pools))
	for _, pool := range pools {
		if filter(pool) {
			filtered = append(filtered, pool)
		}
	}
	return filtered
}

func TestAMMPools_ListByMarket(t *testing.T) {
	ctx := tempTransaction(t)

	ps, pools, _, markets, _ := setupAMMPoolsTest(ctx, t)
	src := rand.NewSource(time.Now().UnixNano())
	r := rand.New(src)
	n := len(markets)

	t.Run("Should return all pools if no pagination is provided", func(t *testing.T) {
		// Randomly pick a market
		market := markets[r.Intn(n)]
		want := orderPools(filterPools(pools, func(pool entities.AMMPool) bool {
			return pool.MarketID == market
		}))
		pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, true)
		require.NoError(t, err)
		listedPools, pageInfo, err := ps.ListByMarket(ctx, market, false, pagination)
		require.NoError(t, err)
		assert.Equal(t, len(want), len(listedPools))
		assert.Equal(t, want, listedPools)
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     false,
			HasPreviousPage: false,
			StartCursor:     want[0].Cursor().Encode(),
			EndCursor:       want[len(want)-1].Cursor().Encode(),
		}, pageInfo)
	})

	t.Run("Should return all active pools", func(t *testing.T) {
		// Randomly pick a market
		market := markets[r.Intn(n)]
		want := orderPools(filterPools(pools, func(pool entities.AMMPool) bool {
			if pool.MarketID != market {
				return false
			}
			return pool.Status == entities.AMMStatusActive || pool.Status == entities.AMMStatusReduceOnly
		}))
		pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, true)
		require.NoError(t, err)
		listedPools, pageInfo, err := ps.ListByMarket(ctx, market, true, pagination)
		require.NoError(t, err)
		assert.Equal(t, len(want), len(listedPools))

		assert.Equal(t, want, listedPools)
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     false,
			HasPreviousPage: false,
			StartCursor:     want[0].Cursor().Encode(),
			EndCursor:       want[len(want)-1].Cursor().Encode(),
		}, pageInfo)
	})

	t.Run("Should return the first page of pools", func(t *testing.T) {
		// Randomly pick a market
		market := markets[r.Intn(n)]
		want := orderPools(filterPools(pools, func(pool entities.AMMPool) bool {
			return pool.MarketID == market
		}))
		pagination, err := entities.NewCursorPagination(ptr.From(int32(3)), nil, nil, nil, true)
		require.NoError(t, err)
		listedPools, pageInfo, err := ps.ListByMarket(ctx, market, false, pagination)
		require.NoError(t, err)
		assert.Equal(t, 3, len(listedPools))
		assert.Equal(t, want[:3], listedPools)
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     true,
			HasPreviousPage: false,
			StartCursor:     want[0].Cursor().Encode(),
			EndCursor:       want[2].Cursor().Encode(),
		}, pageInfo)
	})

	t.Run("Should return the last page of pools", func(t *testing.T) {
		// Randomly pick a market
		market := markets[r.Intn(n)]
		want := orderPools(filterPools(pools, func(pool entities.AMMPool) bool {
			return pool.MarketID == market
		}))
		pagination, err := entities.NewCursorPagination(nil, nil, ptr.From(int32(3)), nil, true)
		require.NoError(t, err)
		listedPools, pageInfo, err := ps.ListByMarket(ctx, market, false, pagination)
		require.NoError(t, err)
		assert.Equal(t, 3, len(listedPools))
		assert.Equal(t, want[len(want)-3:], listedPools)
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     false,
			HasPreviousPage: true,
			StartCursor:     want[len(want)-3].Cursor().Encode(),
			EndCursor:       want[len(want)-1].Cursor().Encode(),
		}, pageInfo)
	})

	t.Run("Should return the requested page when paging forward", func(t *testing.T) {
		// Randomly pick a market
		market := markets[r.Intn(n)]
		want := orderPools(filterPools(pools, func(pool entities.AMMPool) bool {
			return pool.MarketID == market
		}))
		pagination, err := entities.NewCursorPagination(ptr.From(int32(3)), ptr.From(want[0].Cursor().Encode()), nil, nil, true)
		require.NoError(t, err)
		listedPools, pageInfo, err := ps.ListByMarket(ctx, market, false, pagination)
		require.NoError(t, err)
		assert.Equal(t, 3, len(listedPools))
		assert.Equal(t, want[1:4], listedPools)
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     true,
			HasPreviousPage: true,
			StartCursor:     want[1].Cursor().Encode(),
			EndCursor:       want[3].Cursor().Encode(),
		}, pageInfo)
	})

	t.Run("Should return the request page when paging backward", func(t *testing.T) {
		// Randomly pick a market
		market := markets[r.Intn(n)]
		want := orderPools(filterPools(pools, func(pool entities.AMMPool) bool {
			return pool.MarketID == market
		}))
		pagination, err := entities.NewCursorPagination(nil, nil, ptr.From(int32(3)), ptr.From(want[4].Cursor().Encode()), true)
		require.NoError(t, err)
		listedPools, pageInfo, err := ps.ListByMarket(ctx, market, false, pagination)
		require.NoError(t, err)
		assert.Equal(t, 3, len(listedPools))
		assert.Equal(t, want[1:4], listedPools)
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     true,
			HasPreviousPage: true,
			StartCursor:     want[1].Cursor().Encode(),
			EndCursor:       want[3].Cursor().Encode(),
		}, pageInfo)
	})
}

func TestAMMPools_ListByParty(t *testing.T) {
	ctx := tempTransaction(t)

	ps, pools, parties, _, _ := setupAMMPoolsTest(ctx, t)
	src := rand.NewSource(time.Now().UnixNano())
	r := rand.New(src)
	n := len(parties)

	t.Run("Should return all pools if no pagination is provided", func(t *testing.T) {
		// Randomly pick a party
		party := parties[r.Intn(n)]
		want := filterPools(pools, func(pool entities.AMMPool) bool {
			return pool.PartyID == party.PartyID
		})
		pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, true)
		require.NoError(t, err)
		listedPools, pageInfo, err := ps.ListByParty(ctx, party.PartyID, false, pagination)
		require.NoError(t, err)
		assert.Equal(t, len(want), len(listedPools))
		assert.Equal(t, want, listedPools)
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     false,
			HasPreviousPage: false,
			StartCursor:     want[0].Cursor().Encode(),
			EndCursor:       want[len(want)-1].Cursor().Encode(),
		}, pageInfo)
	})

	t.Run("Should return the first page of pools", func(t *testing.T) {
		// Randomly pick a party
		party := parties[r.Intn(n)]
		want := filterPools(pools, func(pool entities.AMMPool) bool {
			return pool.PartyID == party.PartyID
		})
		pagination, err := entities.NewCursorPagination(ptr.From(int32(5)), nil, nil, nil, true)
		require.NoError(t, err)
		listedPools, pageInfo, err := ps.ListByParty(ctx, party.PartyID, false, pagination)
		require.NoError(t, err)
		assert.Equal(t, 5, len(listedPools))
		assert.Equal(t, want[:5], listedPools)
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     true,
			HasPreviousPage: false,
			StartCursor:     want[0].Cursor().Encode(),
			EndCursor:       want[4].Cursor().Encode(),
		}, pageInfo)
	})

	t.Run("Should return the last page of pools", func(t *testing.T) {
		// Randomly pick a party
		party := parties[r.Intn(n)]
		want := filterPools(pools, func(pool entities.AMMPool) bool {
			return pool.PartyID == party.PartyID
		})
		pagination, err := entities.NewCursorPagination(nil, nil, ptr.From(int32(5)), nil, true)
		require.NoError(t, err)
		listedPools, pageInfo, err := ps.ListByParty(ctx, party.PartyID, false, pagination)
		require.NoError(t, err)
		assert.Equal(t, 5, len(listedPools))
		assert.Equal(t, want[len(want)-5:], listedPools)
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     false,
			HasPreviousPage: true,
			StartCursor:     want[len(want)-5].Cursor().Encode(),
			EndCursor:       want[len(want)-1].Cursor().Encode(),
		}, pageInfo)
	})

	t.Run("Should return the requested page when paging forward", func(t *testing.T) {
		// Randomly pick a party
		party := parties[r.Intn(n)]
		want := filterPools(pools, func(pool entities.AMMPool) bool {
			return pool.PartyID == party.PartyID
		})
		pagination, err := entities.NewCursorPagination(ptr.From(int32(5)), ptr.From(want[10].Cursor().Encode()), nil, nil, true)
		require.NoError(t, err)
		listedPools, pageInfo, err := ps.ListByParty(ctx, party.PartyID, false, pagination)
		require.NoError(t, err)
		assert.Equal(t, 5, len(listedPools))
		assert.Equal(t, want[11:16], listedPools)
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     true,
			HasPreviousPage: true,
			StartCursor:     want[11].Cursor().Encode(),
			EndCursor:       want[15].Cursor().Encode(),
		}, pageInfo)
	})

	t.Run("Should return the request page when paging backward", func(t *testing.T) {
		// Randomly pick a party
		party := parties[r.Intn(n)]
		want := filterPools(pools, func(pool entities.AMMPool) bool {
			return pool.PartyID == party.PartyID
		})
		pagination, err := entities.NewCursorPagination(nil, nil, ptr.From(int32(5)), ptr.From(want[10].Cursor().Encode()), true)
		require.NoError(t, err)
		listedPools, pageInfo, err := ps.ListByParty(ctx, party.PartyID, false, pagination)
		require.NoError(t, err)
		assert.Equal(t, 5, len(listedPools))
		assert.Equal(t, want[5:10], listedPools)
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     true,
			HasPreviousPage: true,
			StartCursor:     want[5].Cursor().Encode(),
			EndCursor:       want[9].Cursor().Encode(),
		}, pageInfo)
	})
}

func TestAMMPools_ListByPartyMarketStatus(t *testing.T) {
	ctx := tempTransaction(t)

	ps, pools, parties, _, _ := setupAMMPoolsTest(ctx, t)
	src := rand.NewSource(time.Now().UnixNano())
	r := rand.New(src)
	n := len(parties)

	t.Run("Should list active only", func(t *testing.T) {
		// Randomly pick a party
		party := parties[r.Intn(n)]
		want := filterPools(pools, func(pool entities.AMMPool) bool {
			if pool.PartyID != party.PartyID {
				return false
			}
			return pool.Status == entities.AMMStatusActive || pool.Status == entities.AMMStatusReduceOnly
		})
		pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, true)
		require.NoError(t, err)
		listedPools, pageInfo, err := ps.ListByPartyMarketStatus(ctx, &party.PartyID, nil, nil, true, pagination)
		require.NoError(t, err)
		assert.Equal(t, len(want), len(listedPools))
		assert.Equal(t, want, listedPools)
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     false,
			HasPreviousPage: false,
			StartCursor:     want[0].Cursor().Encode(),
			EndCursor:       want[len(want)-1].Cursor().Encode(),
		}, pageInfo)
	})
}

func TestAMMPools_ListByPool(t *testing.T) {
	ctx := tempTransaction(t)

	ps, pools, _, _, poolIDs := setupAMMPoolsTest(ctx, t)
	src := rand.NewSource(time.Now().UnixNano())
	r := rand.New(src)
	n := len(poolIDs)

	t.Run("Should return the pool if the pool ID exists", func(t *testing.T) {
		pa := poolIDs[r.Intn(n)]
		want := filterPools(pools, func(pool entities.AMMPool) bool {
			return pool.ID == pa
		})
		pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, true)
		require.NoError(t, err)
		listedPools, pageInfo, err := ps.ListByPool(ctx, pa, false, pagination)
		require.NoError(t, err)
		assert.Equal(t, len(want), len(listedPools))
		assert.Equal(t, want, listedPools)
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     false,
			HasPreviousPage: false,
			StartCursor:     want[0].Cursor().Encode(),
			EndCursor:       want[len(want)-1].Cursor().Encode(),
		}, pageInfo)
	})
}

func TestAMMPools_ListBySubAccount(t *testing.T) {
	ctx := tempTransaction(t)

	ps, pools, parties, _, _ := setupAMMPoolsTest(ctx, t)
	src := rand.NewSource(time.Now().UnixNano())
	r := rand.New(src)
	n := len(parties)

	t.Run("Should return all pools if no pagination is provided", func(t *testing.T) {
		// Randomly pick a sub account
		party := parties[r.Intn(n)]
		want := filterPools(pools, func(pool entities.AMMPool) bool {
			return pool.AmmPartyID == party.AMMPartyID
		})
		pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, true)
		require.NoError(t, err)
		listedPools, pageInfo, err := ps.ListBySubAccount(ctx, party.AMMPartyID, false, pagination)
		require.NoError(t, err)
		assert.Equal(t, len(want), len(listedPools))
		assert.Equal(t, want, listedPools)
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     false,
			HasPreviousPage: false,
			StartCursor:     want[0].Cursor().Encode(),
			EndCursor:       want[len(want)-1].Cursor().Encode(),
		}, pageInfo)
	})

	t.Run("Should return the first page of pools", func(t *testing.T) {
		// Randomly pick a sub account
		party := parties[r.Intn(n)]
		want := filterPools(pools, func(pool entities.AMMPool) bool {
			return pool.AmmPartyID == party.AMMPartyID
		})
		pagination, err := entities.NewCursorPagination(ptr.From(int32(5)), nil, nil, nil, true)
		require.NoError(t, err)
		listedPools, pageInfo, err := ps.ListBySubAccount(ctx, party.AMMPartyID, false, pagination)
		require.NoError(t, err)
		assert.Equal(t, 5, len(listedPools))
		assert.Equal(t, want[:5], listedPools)
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     true,
			HasPreviousPage: false,
			StartCursor:     want[0].Cursor().Encode(),
			EndCursor:       want[4].Cursor().Encode(),
		}, pageInfo)
	})

	t.Run("Should return the last page of pools", func(t *testing.T) {
		// Randomly pick a sub account
		party := parties[r.Intn(n)]
		want := filterPools(pools, func(pool entities.AMMPool) bool {
			return pool.AmmPartyID == party.AMMPartyID
		})
		pagination, err := entities.NewCursorPagination(nil, nil, ptr.From(int32(5)), nil, true)
		require.NoError(t, err)
		listedPools, pageInfo, err := ps.ListBySubAccount(ctx, party.AMMPartyID, false, pagination)
		require.NoError(t, err)
		assert.Equal(t, 5, len(listedPools))
		assert.Equal(t, want[len(want)-5:], listedPools)
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     false,
			HasPreviousPage: true,
			StartCursor:     want[len(want)-5].Cursor().Encode(),
			EndCursor:       want[len(want)-1].Cursor().Encode(),
		}, pageInfo)
	})

	t.Run("Should return the requested page when paging forward", func(t *testing.T) {
		// Randomly pick a sub account
		party := parties[r.Intn(n)]
		want := filterPools(pools, func(pool entities.AMMPool) bool {
			return pool.AmmPartyID == party.AMMPartyID
		})
		pagination, err := entities.NewCursorPagination(ptr.From(int32(5)), ptr.From(want[10].Cursor().Encode()), nil, nil, true)
		require.NoError(t, err)
		listedPools, pageInfo, err := ps.ListBySubAccount(ctx, party.AMMPartyID, false, pagination)
		require.NoError(t, err)
		assert.Equal(t, 5, len(listedPools))
		assert.Equal(t, want[11:16], listedPools)
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     true,
			HasPreviousPage: true,
			StartCursor:     want[11].Cursor().Encode(),
			EndCursor:       want[15].Cursor().Encode(),
		}, pageInfo)
	})

	t.Run("Should return the request page when paging backward", func(t *testing.T) {
		// Randomly pick a sub account
		party := parties[r.Intn(n)]
		want := filterPools(pools, func(pool entities.AMMPool) bool {
			return pool.AmmPartyID == party.AMMPartyID
		})
		pagination, err := entities.NewCursorPagination(nil, nil, ptr.From(int32(5)), ptr.From(want[10].Cursor().Encode()), true)
		require.NoError(t, err)
		listedPools, pageInfo, err := ps.ListBySubAccount(ctx, party.AMMPartyID, false, pagination)
		require.NoError(t, err)
		assert.Equal(t, 5, len(listedPools))
		assert.Equal(t, want[5:10], listedPools)
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     true,
			HasPreviousPage: true,
			StartCursor:     want[5].Cursor().Encode(),
			EndCursor:       want[9].Cursor().Encode(),
		}, pageInfo)
	})
}

func TestAMMPools_ListByStatus(t *testing.T) {
	ctx := tempTransaction(t)

	ps, pools, _, _, _ := setupAMMPoolsTest(ctx, t)

	t.Run("Should return all pools if no pagination is provided", func(t *testing.T) {
		want := filterPools(pools, func(pool entities.AMMPool) bool {
			return pool.Status == entities.AMMStatusActive
		})
		pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, true)
		require.NoError(t, err)
		listedPools, pageInfo, err := ps.ListByStatus(ctx, entities.AMMStatusActive, pagination)
		require.NoError(t, err)
		assert.Equal(t, len(want), len(listedPools))
		assert.Equal(t, want, listedPools)
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     false,
			HasPreviousPage: false,
			StartCursor:     want[0].Cursor().Encode(),
			EndCursor:       want[len(want)-1].Cursor().Encode(),
		}, pageInfo)
	})

	t.Run("Should return the first page of pools", func(t *testing.T) {
		want := filterPools(pools, func(pool entities.AMMPool) bool {
			return pool.Status == entities.AMMStatusActive
		})
		pagination, err := entities.NewCursorPagination(ptr.From(int32(5)), nil, nil, nil, true)
		require.NoError(t, err)
		listedPools, pageInfo, err := ps.ListByStatus(ctx, entities.AMMStatusActive, pagination)
		require.NoError(t, err)
		assert.Equal(t, 5, len(listedPools))
		assert.Equal(t, want[:5], listedPools)
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     true,
			HasPreviousPage: false,
			StartCursor:     want[0].Cursor().Encode(),
			EndCursor:       want[4].Cursor().Encode(),
		}, pageInfo)
	})

	t.Run("Should return the last page of pools", func(t *testing.T) {
		want := filterPools(pools, func(pool entities.AMMPool) bool {
			return pool.Status == entities.AMMStatusActive
		})
		pagination, err := entities.NewCursorPagination(nil, nil, ptr.From(int32(5)), nil, true)
		require.NoError(t, err)
		listedPools, pageInfo, err := ps.ListByStatus(ctx, entities.AMMStatusActive, pagination)
		require.NoError(t, err)
		assert.Equal(t, 5, len(listedPools))
		assert.Equal(t, want[len(want)-5:], listedPools)
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     false,
			HasPreviousPage: true,
			StartCursor:     want[len(want)-5].Cursor().Encode(),
			EndCursor:       want[len(want)-1].Cursor().Encode(),
		}, pageInfo)
	})

	t.Run("Should return the requested page when paging forward", func(t *testing.T) {
		want := filterPools(pools, func(pool entities.AMMPool) bool {
			return pool.Status == entities.AMMStatusActive
		})
		pagination, err := entities.NewCursorPagination(ptr.From(int32(5)), ptr.From(want[10].Cursor().Encode()), nil, nil, true)
		require.NoError(t, err)
		listedPools, pageInfo, err := ps.ListByStatus(ctx, entities.AMMStatusActive, pagination)
		require.NoError(t, err)
		assert.Equal(t, 5, len(listedPools))
		assert.Equal(t, want[11:16], listedPools)
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     true,
			HasPreviousPage: true,
			StartCursor:     want[11].Cursor().Encode(),
			EndCursor:       want[15].Cursor().Encode(),
		}, pageInfo)
	})

	t.Run("Should return the request page when paging backward", func(t *testing.T) {
		want := filterPools(pools, func(pool entities.AMMPool) bool {
			return pool.Status == entities.AMMStatusActive
		})
		pagination, err := entities.NewCursorPagination(nil, nil, ptr.From(int32(5)), ptr.From(want[10].Cursor().Encode()), true)
		require.NoError(t, err)
		listedPools, pageInfo, err := ps.ListByStatus(ctx, entities.AMMStatusActive, pagination)
		require.NoError(t, err)
		assert.Equal(t, 5, len(listedPools))
		assert.Equal(t, want[5:10], listedPools)
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     true,
			HasPreviousPage: true,
			StartCursor:     want[5].Cursor().Encode(),
			EndCursor:       want[9].Cursor().Encode(),
		}, pageInfo)
	})
}

func TestAMMPools_ListActive(t *testing.T) {
	ctx := tempTransaction(t)

	ps, in, _, _, _ := setupAMMPoolsTest(ctx, t)
	var nActive int
	for _, p := range in {
		if p.Status == entities.AMMStatusActive || p.Status == entities.AMMStatusReduceOnly {
			nActive++
		}
	}
	require.NotEqual(t, 0, nActive)

	out, err := ps.ListActive(ctx)
	require.NoError(t, err)
	assert.Equal(t, nActive, len(out))
}
