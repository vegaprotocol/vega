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
	"time"

	"code.vegaprotocol.io/vega/libs/ptr"

	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"

	"github.com/stretchr/testify/assert"

	"github.com/georgysavva/scany/pgxscan"

	"github.com/stretchr/testify/require"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/sqlstore"
	"code.vegaprotocol.io/vega/datanode/sqlstore/helpers"
)

func TestFeesStats_AddFeesStats(t *testing.T) {
	t.Run("Should add the stats for the epoch if they do not exist", testAddFeesStatsEpochNotExists)
	t.Run("Should return an error if the epoch already exists for the market/asset", testAddFeesStatsEpochExists)
}

type FeesStatsTestStores struct {
	bs *sqlstore.Blocks
	ms *sqlstore.Markets
	ps *sqlstore.Parties
	as *sqlstore.Assets
	fs *sqlstore.FeesStats
}

func setupFeesStatsStores(t *testing.T) *FeesStatsTestStores {
	t.Helper()
	bs := sqlstore.NewBlocks(connectionSource)
	ms := sqlstore.NewMarkets(connectionSource)
	ps := sqlstore.NewParties(connectionSource)
	as := sqlstore.NewAssets(connectionSource)
	fs := sqlstore.NewFeesStats(connectionSource)

	return &FeesStatsTestStores{
		bs: bs,
		ms: ms,
		ps: ps,
		as: as,
		fs: fs,
	}
}

func testAddFeesStatsEpochNotExists(t *testing.T) {
	stores := setupFeesStatsStores(t)
	ctx := tempTransaction(t)
	block := addTestBlock(t, ctx, stores.bs)
	market := helpers.AddTestMarket(t, ctx, stores.ms, block)
	asset := addTestAsset(t, ctx, stores.as, block)

	want := entities.FeesStats{
		MarketID:                 market.ID,
		AssetID:                  asset.ID,
		EpochSeq:                 100,
		TotalRewardsPaid:         nil,
		ReferrerRewardsGenerated: nil,
		RefereesDiscountApplied:  nil,
		VolumeDiscountApplied:    nil,
		VegaTime:                 block.VegaTime,
	}

	err := stores.fs.AddFeesStats(ctx, &want)
	require.NoError(t, err)

	// Check that the stats were added
	var got entities.FeesStats
	err = pgxscan.Get(ctx, connectionSource.Connection, &got,
		`SELECT * FROM fees_stats WHERE market_id = $1 AND asset_id = $2 AND epoch_seq = $3`,
		market.ID, asset.ID, want.EpochSeq,
	)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func testAddFeesStatsEpochExists(t *testing.T) {
	stores := setupFeesStatsStores(t)
	ctx := tempTransaction(t)
	block := addTestBlock(t, ctx, stores.bs)
	market := helpers.AddTestMarket(t, ctx, stores.ms, block)
	asset := addTestAsset(t, ctx, stores.as, block)

	want := entities.FeesStats{
		MarketID:                 market.ID,
		AssetID:                  asset.ID,
		EpochSeq:                 100,
		TotalRewardsPaid:         nil,
		ReferrerRewardsGenerated: nil,
		RefereesDiscountApplied:  nil,
		VolumeDiscountApplied:    nil,
		VegaTime:                 block.VegaTime,
	}

	err := stores.fs.AddFeesStats(ctx, &want)
	require.NoError(t, err)

	// now try to insert again and make sure we get an error
	err = stores.fs.AddFeesStats(ctx, &want)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate key value violates unique constraint")
}

func TestFeesStats_GetFeesStats(t *testing.T) {
	t.Run("Should return the stats for the market and epoch requested", testGetFeesStatsForMarketAndEpoch)
	t.Run("Should return the stats for the asset and epoch requested", testGetFeesStatsForAssetAndEpoch)
	t.Run("Should return the latest stats for the market requested", testGetFeesStatsForMarketLatest)
	t.Run("Should return the latest stats for the asset requested", testGetFeesStatsForAssetLatest)
	t.Run("Should return an error if an asset and market is provided", testGetFeesStatsNoAssetOrMarket)
	t.Run("Should return the stats for the party and epoch requested", testGetFeesStatsForPartyAndEpoch)
	t.Run("Should return the latest stats for the party", testGetFeesStatsForPartyLatest)
	t.Run("Should return the latest stats for all asset given a party", testGetFeesStatsParty)
}

func setupFeesStats(t *testing.T, ctx context.Context, fs *sqlstore.FeesStats) []entities.FeesStats {
	t.Helper()
	vegaTime := time.Now().Add(-time.Minute).Round(time.Microsecond) // round to microsecond because Postgres doesn't store nanoseconds at the current time
	stats := []entities.FeesStats{
		{
			MarketID: entities.MarketID("deadbeef01"),
			AssetID:  entities.AssetID("deadbaad01"),
			EpochSeq: 1,
			TotalRewardsPaid: []*eventspb.PartyAmount{
				{
					Party:  "cafedaad01",
					Amount: "1000000",
				},
			},
			ReferrerRewardsGenerated: []*eventspb.ReferrerRewardsGenerated{
				{
					Referrer: "cafedaad01",
					GeneratedReward: []*eventspb.PartyAmount{
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
			},
			RefereesDiscountApplied: []*eventspb.PartyAmount{
				{
					Party:  "cafed00d01",
					Amount: "100000",
				},
				{
					Party:  "cafed00d02",
					Amount: "100000",
				},
			},
			VegaTime: vegaTime.Add(5 * time.Second),
		},
		{
			MarketID: entities.MarketID("deadbeef01"),
			AssetID:  entities.AssetID("deadbaad01"),
			EpochSeq: 2,
			TotalRewardsPaid: []*eventspb.PartyAmount{
				{
					Party:  "cafedaad01",
					Amount: "1100000",
				},
			},
			ReferrerRewardsGenerated: []*eventspb.ReferrerRewardsGenerated{
				{
					Referrer: "cafedaad01",
					GeneratedReward: []*eventspb.PartyAmount{
						{
							Party:  "cafed00d01",
							Amount: "550000",
						},
						{
							Party:  "cafed00d02",
							Amount: "550000",
						},
					},
				},
			},
			RefereesDiscountApplied: []*eventspb.PartyAmount{
				{
					Party:  "cafed00d01",
					Amount: "110000",
				},
				{
					Party:  "cafed00d02",
					Amount: "110000",
				},
			},
			VolumeDiscountApplied: []*eventspb.PartyAmount{},
			VegaTime:              vegaTime.Add(10 * time.Second),
		},
		{
			MarketID: entities.MarketID("deadbeef01"),
			AssetID:  entities.AssetID("deadbaad01"),
			EpochSeq: 3,
			TotalRewardsPaid: []*eventspb.PartyAmount{
				{
					Party:  "cafedaad01",
					Amount: "1200000",
				},
			},
			ReferrerRewardsGenerated: []*eventspb.ReferrerRewardsGenerated{
				{
					Referrer: "cafedaad01",
					GeneratedReward: []*eventspb.PartyAmount{
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
			},
			RefereesDiscountApplied: []*eventspb.PartyAmount{
				{
					Party:  "cafed00d01",
					Amount: "120000",
				},
				{
					Party:  "cafed00d02",
					Amount: "120000",
				},
			},
			VolumeDiscountApplied: []*eventspb.PartyAmount{},
			VegaTime:              vegaTime.Add(15 * time.Second),
		},
		{
			MarketID: entities.MarketID("deadbeef02"),
			AssetID:  entities.AssetID("deadbaad02"),
			EpochSeq: 1,
			TotalRewardsPaid: []*eventspb.PartyAmount{
				{
					Party:  "cafedaad02",
					Amount: "2000000",
				},
			},
			ReferrerRewardsGenerated: []*eventspb.ReferrerRewardsGenerated{
				{
					Referrer: "cafedaad02",
					GeneratedReward: []*eventspb.PartyAmount{
						{
							Party:  "cafed00d03",
							Amount: "2000000",
						},
						{
							Party:  "cafed00d04",
							Amount: "2000000",
						},
					},
				},
			},
			RefereesDiscountApplied: []*eventspb.PartyAmount{
				{
					Party:  "cafed00d03",
					Amount: "200000",
				},
				{
					Party:  "cafed00d04",
					Amount: "200000",
				},
			},
			VolumeDiscountApplied: []*eventspb.PartyAmount{},
			VegaTime:              vegaTime.Add(5 * time.Second),
		},
		{
			MarketID: entities.MarketID("deadbeef02"),
			AssetID:  entities.AssetID("deadbaad02"),
			EpochSeq: 2,
			TotalRewardsPaid: []*eventspb.PartyAmount{
				{
					Party:  "cafedaad02",
					Amount: "2100000",
				},
			},
			ReferrerRewardsGenerated: []*eventspb.ReferrerRewardsGenerated{
				{
					Referrer: "cafedaad01",
					GeneratedReward: []*eventspb.PartyAmount{
						{
							Party:  "cafed00d03",
							Amount: "1050000",
						},
						{
							Party:  "cafed00d04",
							Amount: "1050000",
						},
					},
				},
			},
			RefereesDiscountApplied: []*eventspb.PartyAmount{
				{
					Party:  "cafed00d03",
					Amount: "210000",
				},
				{
					Party:  "cafed00d04",
					Amount: "210000",
				},
			},
			VolumeDiscountApplied: []*eventspb.PartyAmount{},
			VegaTime:              vegaTime.Add(10 * time.Second),
		},
		{
			MarketID: entities.MarketID("deadbeef02"),
			AssetID:  entities.AssetID("deadbaad02"),
			EpochSeq: 3,
			TotalRewardsPaid: []*eventspb.PartyAmount{
				{
					Party:  "cafedaad02",
					Amount: "2200000",
				},
			},
			ReferrerRewardsGenerated: []*eventspb.ReferrerRewardsGenerated{
				{
					Referrer: "cafedaad01",
					GeneratedReward: []*eventspb.PartyAmount{
						{
							Party:  "cafed00d03",
							Amount: "1100000",
						},
						{
							Party:  "cafed00d04",
							Amount: "1100000",
						},
					},
				},
			},
			RefereesDiscountApplied: []*eventspb.PartyAmount{
				{
					Party:  "cafed00d03",
					Amount: "220000",
				},
				{
					Party:  "cafed00d04",
					Amount: "220000",
				},
			},
			VolumeDiscountApplied: []*eventspb.PartyAmount{},
			VegaTime:              vegaTime.Add(15 * time.Second),
		},
	}

	for _, stat := range stats {
		err := fs.AddFeesStats(ctx, &stat)
		require.NoError(t, err)
	}

	return stats
}

func testGetFeesStatsForMarketAndEpoch(t *testing.T) {
	stores := setupFeesStatsStores(t)
	ctx := tempTransaction(t)
	stats := setupFeesStats(t, ctx, stores.fs)

	// get the stats for the first market and epoch
	want := stats[0]
	got, err := stores.fs.GetFeesStats(ctx, &want.MarketID, nil, &want.EpochSeq, nil)
	require.NoError(t, err)
	assert.Equal(t, want, *got)

	// get the stats for the second market and epoch
	want = stats[3]
	got, err = stores.fs.GetFeesStats(ctx, &want.MarketID, nil, &want.EpochSeq, nil)
	require.NoError(t, err)
	assert.Equal(t, want, *got)
}

func testGetFeesStatsForAssetAndEpoch(t *testing.T) {
	stores := setupFeesStatsStores(t)
	ctx := tempTransaction(t)
	stats := setupFeesStats(t, ctx, stores.fs)

	// get the stats for the first market and epoch
	want := stats[0]
	got, err := stores.fs.GetFeesStats(ctx, nil, &want.AssetID, &want.EpochSeq, nil)
	require.NoError(t, err)
	assert.Equal(t, want, *got)

	// get the stats for the second market and epoch
	want = stats[3]
	got, err = stores.fs.GetFeesStats(ctx, nil, &want.AssetID, &want.EpochSeq, nil)
	require.NoError(t, err)
	assert.Equal(t, want, *got)
}

func testGetFeesStatsForMarketLatest(t *testing.T) {
	stores := setupFeesStatsStores(t)
	ctx := tempTransaction(t)
	stats := setupFeesStats(t, ctx, stores.fs)

	// get the stats for the first market and epoch
	want := stats[2]
	got, err := stores.fs.GetFeesStats(ctx, &want.MarketID, nil, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, want, *got)

	// get the stats for the second market and epoch
	want = stats[5]
	got, err = stores.fs.GetFeesStats(ctx, &want.MarketID, nil, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, want, *got)
}

func testGetFeesStatsForAssetLatest(t *testing.T) {
	stores := setupFeesStatsStores(t)
	ctx := tempTransaction(t)
	stats := setupFeesStats(t, ctx, stores.fs)

	// get the stats for the first market and epoch
	want := stats[2]
	got, err := stores.fs.GetFeesStats(ctx, nil, &want.AssetID, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, want, *got)

	// get the stats for the second market and epoch
	want = stats[5]
	got, err = stores.fs.GetFeesStats(ctx, nil, &want.AssetID, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, want, *got)
}

func testGetFeesStatsNoAssetOrMarket(t *testing.T) {
	stores := setupFeesStatsStores(t)
	ctx := tempTransaction(t)

	_, err := stores.fs.GetFeesStats(ctx, ptr.From(entities.MarketID("deadbeef01")), ptr.From(entities.AssetID("deadbeef02")), nil, nil)
	require.Error(t, err)
}

func testGetFeesStatsForPartyAndEpoch(t *testing.T) {
	stores := setupFeesStatsStores(t)
	ctx := tempTransaction(t)
	stats := setupFeesStats(t, ctx, stores.fs)

	// get the stats for the first market and epoch
	expected := stats[1]
	want := entities.FeesStats{
		MarketID: entities.MarketID("deadbeef01"),
		AssetID:  entities.AssetID("deadbaad01"),
		EpochSeq: 2,
		TotalRewardsPaid: []*eventspb.PartyAmount{
			{
				Party:  "cafedaad01",
				Amount: "1100000",
			},
		},
		ReferrerRewardsGenerated: []*eventspb.ReferrerRewardsGenerated{
			{
				Referrer: "cafedaad01",
				GeneratedReward: []*eventspb.PartyAmount{
					{
						Party:  "cafed00d01",
						Amount: "550000",
					},
					{
						Party:  "cafed00d02",
						Amount: "550000",
					},
				},
			},
		},
		RefereesDiscountApplied: []*eventspb.PartyAmount{},
		VolumeDiscountApplied:   []*eventspb.PartyAmount{},
		TotalMakerFeesReceived:  []*eventspb.PartyAmount{},
		MakerFeesGenerated:      []*eventspb.MakerFeesGenerated{},
		VegaTime:                expected.VegaTime,
	}

	got, err := stores.fs.GetFeesStats(ctx, nil, &want.AssetID, ptr.From(want.EpochSeq), &want.ReferrerRewardsGenerated[0].Referrer)
	require.NoError(t, err)
	assert.Equal(t, want, *got)
}

func testGetFeesStatsForPartyLatest(t *testing.T) {
	stores := setupFeesStatsStores(t)
	ctx := tempTransaction(t)
	stats := setupFeesStats(t, ctx, stores.fs)

	// get the stats for the first market and epoch
	expected := stats[2]
	want := entities.FeesStats{
		MarketID: entities.MarketID("deadbeef01"),
		AssetID:  entities.AssetID("deadbaad01"),
		EpochSeq: 3,
		TotalRewardsPaid: []*eventspb.PartyAmount{
			{
				Party:  "cafedaad01",
				Amount: "1200000",
			},
		},
		ReferrerRewardsGenerated: []*eventspb.ReferrerRewardsGenerated{
			{
				Referrer: "cafedaad01",
				GeneratedReward: []*eventspb.PartyAmount{
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
		},
		RefereesDiscountApplied: []*eventspb.PartyAmount{},
		VolumeDiscountApplied:   []*eventspb.PartyAmount{},
		TotalMakerFeesReceived:  []*eventspb.PartyAmount{},
		MakerFeesGenerated:      []*eventspb.MakerFeesGenerated{},
		VegaTime:                expected.VegaTime,
	}
	got, err := stores.fs.GetFeesStats(ctx, nil, &want.AssetID, nil, &want.ReferrerRewardsGenerated[0].Referrer)
	require.NoError(t, err)
	assert.Equal(t, want, *got)
}

func testGetFeesStatsParty(t *testing.T) {
	stores := setupFeesStatsStores(t)
	ctx := tempTransaction(t)
	stats := setupFeesStats(t, ctx, stores.fs)

	// get the stats for the first market and epoch
	expected := stats[2]
	want := entities.FeesStats{
		MarketID: entities.MarketID("deadbeef01"),
		AssetID:  entities.AssetID("deadbaad01"),
		EpochSeq: 3,
		TotalRewardsPaid: []*eventspb.PartyAmount{
			{
				Party:  "cafedaad01",
				Amount: "1200000",
			},
		},
		ReferrerRewardsGenerated: []*eventspb.ReferrerRewardsGenerated{
			{
				Referrer: "cafedaad01",
				GeneratedReward: []*eventspb.PartyAmount{
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
		},
		RefereesDiscountApplied: []*eventspb.PartyAmount{},
		VolumeDiscountApplied:   []*eventspb.PartyAmount{},
		TotalMakerFeesReceived:  []*eventspb.PartyAmount{},
		MakerFeesGenerated:      []*eventspb.MakerFeesGenerated{},
		VegaTime:                expected.VegaTime,
	}
	got, err := stores.fs.GetFeesStats(ctx, nil, &want.AssetID, nil, &want.ReferrerRewardsGenerated[0].Referrer)
	require.NoError(t, err)
	assert.Equal(t, want, *got)
}
