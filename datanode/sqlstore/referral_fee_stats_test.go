package sqlstore_test

import (
	"context"
	"testing"
	"time"

	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"

	"github.com/stretchr/testify/assert"

	"github.com/georgysavva/scany/pgxscan"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/sqlstore"
	"code.vegaprotocol.io/vega/datanode/sqlstore/helpers"
	"github.com/stretchr/testify/require"
)

func TestReferralFeeStats_AddFeeStats(t *testing.T) {
	t.Run("Should add the stats for the epoch if they do not exist", testAddFeeStatsEpochNotExists)
	t.Run("Should return an error if the epoch already exists for the market/asset", testAddFeeStatsEpochExists)
}

type referralFeeStatsTestStores struct {
	bs *sqlstore.Blocks
	ms *sqlstore.Markets
	ps *sqlstore.Parties
	as *sqlstore.Assets
	fs *sqlstore.ReferralFeeStats
}

func setupReferralFeeStatsStores(t *testing.T) *referralFeeStatsTestStores {
	t.Helper()
	bs := sqlstore.NewBlocks(connectionSource)
	ms := sqlstore.NewMarkets(connectionSource)
	ps := sqlstore.NewParties(connectionSource)
	as := sqlstore.NewAssets(connectionSource)
	fs := sqlstore.NewReferralFeeStats(connectionSource)

	return &referralFeeStatsTestStores{
		bs: bs,
		ms: ms,
		ps: ps,
		as: as,
		fs: fs,
	}
}

func testAddFeeStatsEpochNotExists(t *testing.T) {
	stores := setupReferralFeeStatsStores(t)
	ctx := tempTransaction(t)
	block := addTestBlock(t, ctx, stores.bs)
	market := helpers.AddTestMarket(t, ctx, stores.ms, block)
	asset := addTestAsset(t, ctx, stores.as, block)

	want := entities.ReferralFeeStats{
		MarketID:                 market.ID,
		AssetID:                  asset.ID,
		EpochSeq:                 100,
		TotalRewardsPaid:         nil,
		ReferrerRewardsGenerated: nil,
		RefereesDiscountApplied:  nil,
		VolumeDiscountApplied:    nil,
		VegaTime:                 block.VegaTime,
	}

	err := stores.fs.AddFeeStats(ctx, &want)
	require.NoError(t, err)

	// Check that the stats were added
	var got entities.ReferralFeeStats
	err = pgxscan.Get(ctx, connectionSource.Connection, &got,
		`SELECT * FROM referral_fee_stats WHERE market_id = $1 AND asset_id = $2 AND epoch_seq = $3`,
		market.ID, asset.ID, want.EpochSeq,
	)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func testAddFeeStatsEpochExists(t *testing.T) {
	stores := setupReferralFeeStatsStores(t)
	ctx := tempTransaction(t)
	block := addTestBlock(t, ctx, stores.bs)
	market := helpers.AddTestMarket(t, ctx, stores.ms, block)
	asset := addTestAsset(t, ctx, stores.as, block)

	want := entities.ReferralFeeStats{
		MarketID:                 market.ID,
		AssetID:                  asset.ID,
		EpochSeq:                 100,
		TotalRewardsPaid:         nil,
		ReferrerRewardsGenerated: nil,
		RefereesDiscountApplied:  nil,
		VolumeDiscountApplied:    nil,
		VegaTime:                 block.VegaTime,
	}

	err := stores.fs.AddFeeStats(ctx, &want)
	require.NoError(t, err)

	// now try to insert again and make sure we get an error
	err = stores.fs.AddFeeStats(ctx, &want)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate key value violates unique constraint")
}

func TestReferralFeeStats_GetFeeStats(t *testing.T) {
	t.Run("Should return the stats for the market and epoch requested", testGetFeeStatsForMarketAndEpoch)
	t.Run("Should return the stats for the asset and epoch requested", testGetFeeStatsForAssetAndEpoch)
	t.Run("Should return the latest stats for the market requested", testGetFeeStatsForMarketLatest)
	t.Run("Should return the latest stats for the asset requested", testGetFeeStatsForAssetLatest)
	t.Run("Should return an error if an asset or market is not provided", testGetFeeStatsNoAssetOrMarket)
}

func setupFeeStats(t *testing.T, ctx context.Context, fs *sqlstore.ReferralFeeStats) []entities.ReferralFeeStats {
	t.Helper()
	vegaTime := time.Now().Add(-time.Minute).Round(time.Microsecond) // round to microsecond because Postgres doesn't store nanoseconds at the current time
	stats := []entities.ReferralFeeStats{
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
			VolumeDiscountApplied: []*eventspb.PartyAmount{},
			VegaTime:              vegaTime.Add(5 * time.Second),
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
		err := fs.AddFeeStats(ctx, &stat)
		require.NoError(t, err)
	}

	return stats
}

func testGetFeeStatsForMarketAndEpoch(t *testing.T) {
	stores := setupReferralFeeStatsStores(t)
	ctx := tempTransaction(t)
	stats := setupFeeStats(t, ctx, stores.fs)

	// get the stats for the first market and epoch
	want := stats[0]
	got, err := stores.fs.GetFeeStats(ctx, &want.MarketID, nil, &want.EpochSeq)
	require.NoError(t, err)
	assert.Equal(t, want, *got)

	// get the stats for the second market and epoch
	want = stats[3]
	got, err = stores.fs.GetFeeStats(ctx, &want.MarketID, nil, &want.EpochSeq)
	require.NoError(t, err)
	assert.Equal(t, want, *got)
}

func testGetFeeStatsForAssetAndEpoch(t *testing.T) {
	stores := setupReferralFeeStatsStores(t)
	ctx := tempTransaction(t)
	stats := setupFeeStats(t, ctx, stores.fs)

	// get the stats for the first market and epoch
	want := stats[0]
	got, err := stores.fs.GetFeeStats(ctx, nil, &want.AssetID, &want.EpochSeq)
	require.NoError(t, err)
	assert.Equal(t, want, *got)

	// get the stats for the second market and epoch
	want = stats[3]
	got, err = stores.fs.GetFeeStats(ctx, nil, &want.AssetID, &want.EpochSeq)
	require.NoError(t, err)
	assert.Equal(t, want, *got)
}

func testGetFeeStatsForMarketLatest(t *testing.T) {
	stores := setupReferralFeeStatsStores(t)
	ctx := tempTransaction(t)
	stats := setupFeeStats(t, ctx, stores.fs)

	// get the stats for the first market and epoch
	want := stats[2]
	got, err := stores.fs.GetFeeStats(ctx, &want.MarketID, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, want, *got)

	// get the stats for the second market and epoch
	want = stats[5]
	got, err = stores.fs.GetFeeStats(ctx, &want.MarketID, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, want, *got)
}

func testGetFeeStatsForAssetLatest(t *testing.T) {
	stores := setupReferralFeeStatsStores(t)
	ctx := tempTransaction(t)
	stats := setupFeeStats(t, ctx, stores.fs)

	// get the stats for the first market and epoch
	want := stats[2]
	got, err := stores.fs.GetFeeStats(ctx, nil, &want.AssetID, nil)
	require.NoError(t, err)
	assert.Equal(t, want, *got)

	// get the stats for the second market and epoch
	want = stats[5]
	got, err = stores.fs.GetFeeStats(ctx, nil, &want.AssetID, nil)
	require.NoError(t, err)
	assert.Equal(t, want, *got)
}

func testGetFeeStatsNoAssetOrMarket(t *testing.T) {
	stores := setupReferralFeeStatsStores(t)
	ctx := tempTransaction(t)

	_, err := stores.fs.GetFeeStats(ctx, nil, nil, nil)
	require.Error(t, err)
}