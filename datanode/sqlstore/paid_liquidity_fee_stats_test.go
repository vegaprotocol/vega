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

	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"

	"github.com/stretchr/testify/assert"

	"github.com/georgysavva/scany/pgxscan"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/sqlstore"
	"code.vegaprotocol.io/vega/datanode/sqlstore/helpers"
	"github.com/stretchr/testify/require"

	vgtesting "code.vegaprotocol.io/vega/datanode/libs/testing"
)

func TestPaidLiquidityFeesStats_Add(t *testing.T) {
	t.Run("Should add the stats for the epoch if they do not exist", testAddPaidLiquidityFeesStatsEpochIfNotExists)
	t.Run("Should return an error if the epoch already exists for the market/asset", testAddPaidLiquidityFeesStatsEpochExists)
}

type paidLiquidityFeesStatsTestStores struct {
	bs *sqlstore.Blocks
	ms *sqlstore.Markets
	ps *sqlstore.Parties
	as *sqlstore.Assets
	ls *sqlstore.PaidLiquidityFeesStats
}

func setupPaidLiquidityFeesStatsStores(t *testing.T) *paidLiquidityFeesStatsTestStores {
	t.Helper()
	bs := sqlstore.NewBlocks(connectionSource)
	ms := sqlstore.NewMarkets(connectionSource)
	ps := sqlstore.NewParties(connectionSource)
	as := sqlstore.NewAssets(connectionSource)
	ls := sqlstore.NewPaidLiquidityFeesStats(connectionSource)

	return &paidLiquidityFeesStatsTestStores{
		bs: bs,
		ms: ms,
		ps: ps,
		as: as,
		ls: ls,
	}
}

func testAddPaidLiquidityFeesStatsEpochIfNotExists(t *testing.T) {
	stores := setupPaidLiquidityFeesStatsStores(t)
	ctx := tempTransaction(t)
	block := addTestBlock(t, ctx, stores.bs)
	market := helpers.AddTestMarket(t, ctx, stores.ms, block)
	asset := addTestAsset(t, ctx, stores.as, block)

	want := entities.PaidLiquidityFeesStats{
		MarketID:      market.ID,
		AssetID:       asset.ID,
		EpochSeq:      100,
		TotalFeesPaid: "100",
		FeesPerParty: []*eventspb.PartyAmount{
			{Party: "party-1", Amount: "50"},
			{Party: "party-2", Amount: "50"},
			{Party: "party-3", Amount: "50"},
		},
	}

	err := stores.ls.Add(ctx, &want)
	require.NoError(t, err)

	// Check that the stats were added
	var got entities.PaidLiquidityFeesStats
	err = pgxscan.Get(ctx, connectionSource.Connection, &got,
		`SELECT market_id, asset_id, epoch_seq, total_fees_paid, fees_paid_per_party as fees_per_party
		FROM paid_liquidity_fees WHERE market_id = $1 AND asset_id = $2 AND epoch_seq = $3`,
		market.ID, asset.ID, want.EpochSeq,
	)
	require.NoError(t, err)
	vgtesting.AssertProtoEqual(t, want.ToProto(), got.ToProto())
}

func testAddPaidLiquidityFeesStatsEpochExists(t *testing.T) {
	stores := setupPaidLiquidityFeesStatsStores(t)
	ctx := tempTransaction(t)
	block := addTestBlock(t, ctx, stores.bs)
	market := helpers.AddTestMarket(t, ctx, stores.ms, block)
	asset := addTestAsset(t, ctx, stores.as, block)

	want := entities.PaidLiquidityFeesStats{
		MarketID:      market.ID,
		AssetID:       asset.ID,
		EpochSeq:      100,
		TotalFeesPaid: "100",
		FeesPerParty: []*eventspb.PartyAmount{
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

func TestPaidLiquidityFeesStats_List(t *testing.T) {
	t.Run("Should return the stats for the market and epoch requested", testListPaidLiquidityFeesStatsForMarketAndEpoch)
	t.Run("Should return the stats for the asset and epoch requested", testListPaidLiquidityFeesStatsForAssetAndEpoch)
	t.Run("Should return the latest stats for the market requested", testListPaidLiquidityFeesStatsForMarketLatest)
	t.Run("Should return the latest stats for the asset requested", testListPaidLiquidityFeesStatsForAssetLatest)
	t.Run("Should return the stats for the party and epoch requested", testListPaidLiquidityFeesStatsForPartyAndEpoch)
	t.Run("Should return the latest stats for the party", testListPaidLiquidityFeesStatsForPartyLatest)
}

func setupPaidLiquidityFeesStats(t *testing.T, ctx context.Context, ls *sqlstore.PaidLiquidityFeesStats) []entities.PaidLiquidityFeesStats {
	t.Helper()
	stats := []entities.PaidLiquidityFeesStats{
		{
			MarketID:      entities.MarketID("deadbeef01"),
			AssetID:       entities.AssetID("deadbaad01"),
			EpochSeq:      1,
			TotalFeesPaid: "1000000",
			FeesPerParty: []*eventspb.PartyAmount{
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
			FeesPerParty: []*eventspb.PartyAmount{
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
			FeesPerParty: []*eventspb.PartyAmount{
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
			FeesPerParty: []*eventspb.PartyAmount{
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
			FeesPerParty: []*eventspb.PartyAmount{
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
			FeesPerParty: []*eventspb.PartyAmount{
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

func testListPaidLiquidityFeesStatsForMarketAndEpoch(t *testing.T) {
	stores := setupPaidLiquidityFeesStatsStores(t)
	ctx := tempTransaction(t)
	stats := setupPaidLiquidityFeesStats(t, ctx, stores.ls)

	pagination := entities.DefaultCursorPagination(true)

	// get the stats for the first market and epoch
	want := stats[0:1]
	got, _, err := stores.ls.List(ctx, &want[0].MarketID, nil, &want[0].EpochSeq, nil, pagination)
	require.NoError(t, err)

	assert.Len(t, got, len(want))
	vgtesting.AssertProtoEqual(t, want[0].ToProto(), got[0].ToProto())

	// get the stats for the second market and epoch
	want = stats[3:4]
	got, _, err = stores.ls.List(ctx, &want[0].MarketID, nil, &want[0].EpochSeq, nil, pagination)
	require.NoError(t, err)

	assert.Len(t, got, len(want))
	vgtesting.AssertProtoEqual(t, want[0].ToProto(), got[0].ToProto())
}

func testListPaidLiquidityFeesStatsForAssetAndEpoch(t *testing.T) {
	stores := setupPaidLiquidityFeesStatsStores(t)
	ctx := tempTransaction(t)
	stats := setupPaidLiquidityFeesStats(t, ctx, stores.ls)

	pagination := entities.DefaultCursorPagination(true)

	// get the stats for the first market and epoch
	want := stats[1:2]
	got, _, err := stores.ls.List(ctx, nil, &want[0].AssetID, &want[0].EpochSeq, nil, pagination)
	require.NoError(t, err)

	assert.Len(t, got, len(want))
	vgtesting.AssertProtoEqual(t, want[0].ToProto(), got[0].ToProto())

	// get the stats for the second market and epoch
	want = stats[4:5]
	got, _, err = stores.ls.List(ctx, nil, &want[0].AssetID, &want[0].EpochSeq, nil, pagination)
	require.NoError(t, err)

	assert.Len(t, got, len(want))
	vgtesting.AssertProtoEqual(t, want[0].ToProto(), got[0].ToProto())
}

func testListPaidLiquidityFeesStatsForMarketLatest(t *testing.T) {
	stores := setupPaidLiquidityFeesStatsStores(t)
	ctx := tempTransaction(t)
	stats := setupPaidLiquidityFeesStats(t, ctx, stores.ls)

	pagination := entities.DefaultCursorPagination(true)

	// get the stats for the first market and epoch
	want := stats[2:3]
	got, _, err := stores.ls.List(ctx, &want[0].MarketID, nil, nil, nil, pagination)
	require.NoError(t, err)

	assert.Len(t, got, len(want))
	vgtesting.AssertProtoEqual(t, want[0].ToProto(), got[0].ToProto())

	// get the stats for the second market and epoch
	want = stats[5:6]
	got, _, err = stores.ls.List(ctx, &want[0].MarketID, nil, nil, nil, pagination)
	require.NoError(t, err)

	assert.Len(t, got, len(want))
	vgtesting.AssertProtoEqual(t, want[0].ToProto(), got[0].ToProto())
}

func testListPaidLiquidityFeesStatsForAssetLatest(t *testing.T) {
	stores := setupPaidLiquidityFeesStatsStores(t)
	ctx := tempTransaction(t)
	stats := setupPaidLiquidityFeesStats(t, ctx, stores.ls)

	pagination := entities.DefaultCursorPagination(true)

	// get the stats for the first market and epoch
	want := stats[2:3]
	got, _, err := stores.ls.List(ctx, nil, &want[0].AssetID, nil, nil, pagination)
	require.NoError(t, err)

	assert.Len(t, got, len(want))
	vgtesting.AssertProtoEqual(t, want[0].ToProto(), got[0].ToProto())

	// get the stats for the second market and epoch
	want = stats[5:6]
	got, _, err = stores.ls.List(ctx, nil, &want[0].AssetID, nil, nil, pagination)
	require.NoError(t, err)

	assert.Len(t, got, len(want))
	vgtesting.AssertProtoEqual(t, want[0].ToProto(), got[0].ToProto())
}

func testListPaidLiquidityFeesStatsForPartyAndEpoch(t *testing.T) {
	stores := setupPaidLiquidityFeesStatsStores(t)
	ctx := tempTransaction(t)
	stats := setupPaidLiquidityFeesStats(t, ctx, stores.ls)

	pagination := entities.DefaultCursorPagination(true)

	// get the stats for the first market and epoch
	want := append(stats[3:4], stats[0:1]...)
	want[0].FeesPerParty = want[0].FeesPerParty[:1]
	want[1].FeesPerParty = want[1].FeesPerParty[:1]

	got, _, err := stores.ls.List(ctx, nil, nil, &want[0].EpochSeq, []string{want[0].FeesPerParty[0].Party}, pagination)
	require.NoError(t, err)

	assert.Len(t, got, len(want))
	vgtesting.AssertProtoEqual(t, want[0].ToProto(), got[0].ToProto())
	vgtesting.AssertProtoEqual(t, want[1].ToProto(), got[1].ToProto())
}

func testListPaidLiquidityFeesStatsForPartyLatest(t *testing.T) {
	stores := setupPaidLiquidityFeesStatsStores(t)
	ctx := tempTransaction(t)
	stats := setupPaidLiquidityFeesStats(t, ctx, stores.ls)

	pagination := entities.DefaultCursorPagination(true)

	// get the stats for the first market and epoch
	want := append(stats[5:6], stats[2:3]...)
	want[0].FeesPerParty = want[0].FeesPerParty[:1]
	want[1].FeesPerParty = want[1].FeesPerParty[:1]

	got, _, err := stores.ls.List(ctx, nil, nil, nil, []string{want[0].FeesPerParty[0].Party}, pagination)
	require.NoError(t, err)

	assert.Len(t, got, len(want))
	vgtesting.AssertProtoEqual(t, want[0].ToProto(), got[0].ToProto())
	vgtesting.AssertProtoEqual(t, want[1].ToProto(), got[1].ToProto())
}
