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
	"fmt"
	"testing"

	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"

	"github.com/stretchr/testify/assert"

	"github.com/georgysavva/scany/pgxscan"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/sqlstore"
	"code.vegaprotocol.io/vega/datanode/sqlstore/helpers"
	"github.com/stretchr/testify/require"

	vgtesting "code.vegaprotocol.io/vega/datanode/libs/testing"
)

func TestPaidLiquidityFeeStats_AddFeeStats(t *testing.T) {
	t.Run("Should add the stats for the epoch if they do not exist", testAddPaidLiquidityFeeStatsEpochIfNotExists)
	t.Run("Should return an error if the epoch already exists for the market/asset", testAddPaidLiquidityFeeStatsEpochExists)
}

type paidLiquidityFeeStatsTestStores struct {
	bs *sqlstore.Blocks
	ms *sqlstore.Markets
	ps *sqlstore.Parties
	as *sqlstore.Assets
	ls *sqlstore.PaidLiquidityFeeStats
}

func setupPaidLiquidityFeeStatsStores(t *testing.T) *paidLiquidityFeeStatsTestStores {
	t.Helper()
	bs := sqlstore.NewBlocks(connectionSource)
	ms := sqlstore.NewMarkets(connectionSource)
	ps := sqlstore.NewParties(connectionSource)
	as := sqlstore.NewAssets(connectionSource)
	ls := sqlstore.NewPaidLiquidityFeeStats(connectionSource)

	return &paidLiquidityFeeStatsTestStores{
		bs: bs,
		ms: ms,
		ps: ps,
		as: as,
		ls: ls,
	}
}

func testAddPaidLiquidityFeeStatsEpochIfNotExists(t *testing.T) {
	stores := setupPaidLiquidityFeeStatsStores(t)
	ctx := tempTransaction(t)
	block := addTestBlock(t, ctx, stores.bs)
	market := helpers.AddTestMarket(t, ctx, stores.ms, block)
	asset := addTestAsset(t, ctx, stores.as, block)

	want := entities.PaidLiquidityFeeStats{
		MarketID:      market.ID,
		AssetID:       asset.ID,
		EpochSeq:      100,
		TotalFeesPaid: "100",
		FeesPaidPerParty: []*eventspb.PartyAmount{
			{Party: "party-1", Amount: "50"},
			{Party: "party-2", Amount: "50"},
			{Party: "party-3", Amount: "50"},
		},
	}

	err := stores.ls.Add(ctx, &want)
	require.NoError(t, err)

	// Check that the stats were added
	var got entities.PaidLiquidityFeeStats
	err = pgxscan.Get(ctx, connectionSource.Connection, &got,
		`SELECT * FROM paid_liquidity_fees WHERE market_id = $1 AND asset_id = $2 AND epoch_seq = $3`,
		market.ID, asset.ID, want.EpochSeq,
	)
	require.NoError(t, err)
	vgtesting.AssertProtoEqual(t, want.ToProto(), got.ToProto())
}

func testAddPaidLiquidityFeeStatsEpochExists(t *testing.T) {
	stores := setupPaidLiquidityFeeStatsStores(t)
	ctx := tempTransaction(t)
	block := addTestBlock(t, ctx, stores.bs)
	market := helpers.AddTestMarket(t, ctx, stores.ms, block)
	asset := addTestAsset(t, ctx, stores.as, block)

	want := entities.PaidLiquidityFeeStats{
		MarketID:      market.ID,
		AssetID:       asset.ID,
		EpochSeq:      100,
		TotalFeesPaid: "100",
		FeesPaidPerParty: []*eventspb.PartyAmount{
			{Party: "party-1", Amount: "50"},
			{Party: "party-2", Amount: "50"},
			{Party: "party-3", Amount: "50"},
		},
	}

	err := stores.ls.Add(ctx, &want)
	require.NoError(t, err)

	// now try to insert again and make sure we get an error
	err = stores.ls.Add(ctx, &want)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate key value violates unique constraint")
}

func TestPaidLiquidityFeeStats_GetFeeStats(t *testing.T) {
	t.Run("Should return the stats for the market and epoch requested", testListPaidLiquidityFeeStatsForMarketAndEpoch)
	// t.Run("Should return the stats for the asset and epoch requested", testListPaidLiquidityFeeStatsForAssetAndEpoch)
	// t.Run("Should return the latest stats for the market requested", testListPaidLiquidityFeeStatsForMarketLatest)
	// t.Run("Should return the latest stats for the asset requested", testListPaidLiquidityFeeStatsForAssetLatest)
	// t.Run("Should return an error if an asset and market is provided", testListPaidLiquidityFeeStatsNoAssetOrMarket)
	// t.Run("Should return the stats for the referrer and epoch requested", testListPaidLiquidityFeeStatsForReferrerAndEpoch)
	// t.Run("Should return the stats for the referee and epoch requested", testListPaidLiquidityFeeStatsForRefereeAndEpoch)
	// t.Run("Should return the latest stats for the referrer", testListPaidLiquidityFeeStatsForReferrerLatest)
	// t.Run("Should return the latest stats for the referee", testListPaidLiquidityFeeStatsForRefereeLatest)
	// t.Run("Should return the latest stats for all asset given a referrer", testListPaidLiquidityFeeStatsReferrer)
	// t.Run("Should return the latest stats for all asset given a referee", testListPaidLiquidityFeeStatsReferee)
}

func setupPaidLiquidityFeeStats(t *testing.T, ctx context.Context, ls *sqlstore.PaidLiquidityFeeStats) []entities.PaidLiquidityFeeStats {
	t.Helper()
	stats := []entities.PaidLiquidityFeeStats{
		{
			MarketID:      entities.MarketID("deadbeef01"),
			AssetID:       entities.AssetID("deadbaad01"),
			EpochSeq:      1,
			TotalFeesPaid: "1000000",
			FeesPaidPerParty: []*eventspb.PartyAmount{
				{
					Party:  "cafed00d01",
					Amount: "500000",
				},
				{
					Party:  "cafed00d02",
					Amount: "500000",
				},
			},
		},
		{
			MarketID:      entities.MarketID("deadbeef01"),
			AssetID:       entities.AssetID("deadbaad01"),
			EpochSeq:      2,
			TotalFeesPaid: "1200000",
			FeesPaidPerParty: []*eventspb.PartyAmount{
				{
					Party:  "cafed00d01",
					Amount: "600000",
				},
				{
					Party:  "cafed00d02",
					Amount: "600000",
				},
			},
		},
		{
			MarketID:      entities.MarketID("deadbeef01"),
			AssetID:       entities.AssetID("deadbaad01"),
			EpochSeq:      3,
			TotalFeesPaid: "1400000",
			FeesPaidPerParty: []*eventspb.PartyAmount{
				{
					Party:  "cafed00d01",
					Amount: "700000",
				},
				{
					Party:  "cafed00d02",
					Amount: "700000",
				},
			},
		},
		{
			MarketID:      entities.MarketID("deadbeef02"),
			AssetID:       entities.AssetID("deadbaad02"),
			EpochSeq:      1,
			TotalFeesPaid: "1200000",
			FeesPaidPerParty: []*eventspb.PartyAmount{
				{
					Party:  "cafed00d01",
					Amount: "700000",
				},
				{
					Party:  "cafed00d02",
					Amount: "500000",
				},
			},
		},
		{
			MarketID:      entities.MarketID("deadbeef02"),
			AssetID:       entities.AssetID("deadbaad02"),
			EpochSeq:      2,
			TotalFeesPaid: "1000000",
			FeesPaidPerParty: []*eventspb.PartyAmount{
				{
					Party:  "cafed00d01",
					Amount: "500000",
				},
				{
					Party:  "cafed00d02",
					Amount: "500000",
				},
			},
		},
		{
			MarketID:      entities.MarketID("deadbeef02"),
			AssetID:       entities.AssetID("deadbaad02"),
			EpochSeq:      3,
			TotalFeesPaid: "5000000",
			FeesPaidPerParty: []*eventspb.PartyAmount{
				{
					Party:  "cafed00d01",
					Amount: "25000",
				},
				{
					Party:  "cafed00d02",
					Amount: "25000",
				},
			},
		},
	}

	for _, stat := range stats {
		err := ls.Add(ctx, &stat)
		require.NoError(t, err)
	}

	return stats
}

func testListPaidLiquidityFeeStatsForMarketAndEpoch(t *testing.T) {
	stores := setupPaidLiquidityFeeStatsStores(t)
	ctx := tempTransaction(t)
	stats := setupPaidLiquidityFeeStats(t, ctx, stores.ls)

	pagination := entities.DefaultCursorPagination(true)

	// get the stats for the first market and epoch
	want := stats[0:1]
	got, _, err := stores.ls.List(ctx, &want[0].MarketID, nil, &want[0].EpochSeq, nil, pagination)
	require.NoError(t, err)

	for _, g := range got {
		fmt.Printf("===== got: %+v \n", g)
	}

	assert.Len(t, len(want), len(got))
	vgtesting.AssertProtoEqual(t, want[0].ToProto(), got[0].ToProto())

	// get the stats for the second market and epoch
	want = stats[3:4]
	got, _, err = stores.ls.List(ctx, &want[0].MarketID, nil, &want[0].EpochSeq, nil, pagination)
	require.NoError(t, err)

	vgtesting.AssertProtoEqual(t, want[0].ToProto(), got[0].ToProto())
}

// func testListPaidLiquidityFeeStatsForAssetAndEpoch(t *testing.T) {
// 	stores := setupReferralFeeStatsStores(t)
// 	ctx := tempTransaction(t)
// 	stats := setupPaidLiquidityFeeStats(t, ctx, stores.fs)

// 	// get the stats for the first market and epoch
// 	want := stats[0]
// 	got, err := stores.fs.GetFeeStats(ctx, nil, &want.AssetID, &want.EpochSeq, nil, nil)
// 	require.NoError(t, err)
// 	assert.Equal(t, want, *got)

// 	// get the stats for the second market and epoch
// 	want = stats[3]
// 	got, err = stores.fs.GetFeeStats(ctx, nil, &want.AssetID, &want.EpochSeq, nil, nil)
// 	require.NoError(t, err)
// 	assert.Equal(t, want, *got)
// }

// func testListPaidLiquidityFeeStatsForMarketLatest(t *testing.T) {
// 	stores := setupReferralFeeStatsStores(t)
// 	ctx := tempTransaction(t)
// 	stats := setupPaidLiquidityFeeStats(t, ctx, stores.fs)

// 	// get the stats for the first market and epoch
// 	want := stats[2]
// 	got, err := stores.fs.GetFeeStats(ctx, &want.MarketID, nil, nil, nil, nil)
// 	require.NoError(t, err)
// 	assert.Equal(t, want, *got)

// 	// get the stats for the second market and epoch
// 	want = stats[5]
// 	got, err = stores.fs.GetFeeStats(ctx, &want.MarketID, nil, nil, nil, nil)
// 	require.NoError(t, err)
// 	assert.Equal(t, want, *got)
// }

// func testListPaidLiquidityFeeStatsForAssetLatest(t *testing.T) {
// 	stores := setupReferralFeeStatsStores(t)
// 	ctx := tempTransaction(t)
// 	stats := setupPaidLiquidityFeeStats(t, ctx, stores.fs)

// 	// get the stats for the first market and epoch
// 	want := stats[2]
// 	got, err := stores.fs.GetFeeStats(ctx, nil, &want.AssetID, nil, nil, nil)
// 	require.NoError(t, err)
// 	assert.Equal(t, want, *got)

// 	// get the stats for the second market and epoch
// 	want = stats[5]
// 	got, err = stores.fs.GetFeeStats(ctx, nil, &want.AssetID, nil, nil, nil)
// 	require.NoError(t, err)
// 	assert.Equal(t, want, *got)
// }

// func testListPaidLiquidityFeeStatsNoAssetOrMarket(t *testing.T) {
// 	stores := setupReferralFeeStatsStores(t)
// 	ctx := tempTransaction(t)

// 	_, err := stores.fs.GetFeeStats(ctx, ptr.From(entities.MarketID("deadbeef01")), ptr.From(entities.AssetID("deadbeef02")),
// 		nil, nil, nil)
// 	require.Error(t, err)
// }

// func testListPaidLiquidityFeeStatsForReferrerAndEpoch(t *testing.T) {
// 	stores := setupReferralFeeStatsStores(t)
// 	ctx := tempTransaction(t)
// 	stats := setupPaidLiquidityFeeStats(t, ctx, stores.fs)

// 	// get the stats for the first market and epoch
// 	want := stats[1]
// 	got, err := stores.fs.GetFeeStats(ctx, nil, &want.AssetID, ptr.From(want.EpochSeq), &want.TotalRewardsPaid[0].Party, nil)
// 	require.NoError(t, err)
// 	assert.Equal(t, want, *got)
// }

// func testListPaidLiquidityFeeStatsForRefereeAndEpoch(t *testing.T) {
// 	stores := setupReferralFeeStatsStores(t)
// 	ctx := tempTransaction(t)
// 	stats := setupPaidLiquidityFeeStats(t, ctx, stores.fs)

// 	// get the stats for the first market and epoch
// 	want := stats[1]
// 	got, err := stores.fs.GetFeeStats(ctx, nil, &want.AssetID, ptr.From(want.EpochSeq), nil,
// 		&want.ReferrerRewardsGenerated[0].GeneratedReward[0].Party)
// 	require.NoError(t, err)
// 	assert.Equal(t, want, *got)
// }

// func testListPaidLiquidityFeeStatsForReferrerLatest(t *testing.T) {
// 	stores := setupReferralFeeStatsStores(t)
// 	ctx := tempTransaction(t)
// 	stats := setupPaidLiquidityFeeStats(t, ctx, stores.fs)

// 	// get the stats for the first market and epoch
// 	want := stats[2]
// 	got, err := stores.fs.GetFeeStats(ctx, nil, &want.AssetID, nil, &want.TotalRewardsPaid[0].Party, nil)
// 	require.NoError(t, err)
// 	assert.Equal(t, want, *got)
// }

// func testListPaidLiquidityFeeStatsForRefereeLatest(t *testing.T) {
// 	stores := setupReferralFeeStatsStores(t)
// 	ctx := tempTransaction(t)
// 	stats := setupPaidLiquidityFeeStats(t, ctx, stores.fs)

// 	// get the stats for the first market and epoch
// 	want := stats[2]
// 	got, err := stores.fs.GetFeeStats(ctx, nil, &want.AssetID, nil, nil,
// 		&want.ReferrerRewardsGenerated[0].GeneratedReward[0].Party)
// 	require.NoError(t, err)
// 	assert.Equal(t, want, *got)
// }

// func testListPaidLiquidityFeeStatsReferee(t *testing.T) {
// 	stores := setupReferralFeeStatsStores(t)
// 	ctx := tempTransaction(t)
// 	stats := setupPaidLiquidityFeeStats(t, ctx, stores.fs)

// 	// get the stats for the first market and epoch
// 	want := stats[2]
// 	got, err := stores.fs.GetFeeStats(ctx, nil, &want.AssetID, nil, nil,
// 		&want.ReferrerRewardsGenerated[0].GeneratedReward[0].Party)
// 	require.NoError(t, err)
// 	assert.Equal(t, want, *got)
// }

// func testListPaidLiquidityFeeStatsReferrer(t *testing.T) {
// 	stores := setupReferralFeeStatsStores(t)
// 	ctx := tempTransaction(t)
// 	stats := setupPaidLiquidityFeeStats(t, ctx, stores.fs)

// 	// get the stats for the first market and epoch
// 	want := stats[2]
// 	got, err := stores.fs.GetFeeStats(ctx, nil, &want.AssetID, nil, &want.TotalRewardsPaid[0].Party, nil)
// 	require.NoError(t, err)
// 	assert.Equal(t, want, *got)
// }
