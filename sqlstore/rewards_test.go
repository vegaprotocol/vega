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

package sqlstore_test

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/sqlstore"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func addTestReward(t *testing.T, rs *sqlstore.Rewards,
	party entities.Party,
	asset entities.Asset,
	marketID entities.MarketID,
	epochID int64,
	rewardType string,
	timestamp time.Time,
	block entities.Block,
) entities.Reward {
	r := entities.Reward{
		PartyID:        party.ID,
		AssetID:        asset.ID,
		MarketID:       marketID,
		RewardType:     rewardType,
		EpochID:        epochID,
		Amount:         decimal.NewFromInt(100),
		PercentOfTotal: 0.2,
		Timestamp:      timestamp.Truncate(time.Microsecond),
		VegaTime:       block.VegaTime,
	}
	err := rs.Add(context.Background(), r)
	require.NoError(t, err)
	return r
}

func rewardLessThan(x, y entities.Reward) bool {
	if x.EpochID != y.EpochID {
		return x.EpochID < y.EpochID
	}
	if x.PartyID.String() != y.PartyID.String() {
		return x.PartyID.String() < y.PartyID.String()
	}
	if x.AssetID.String() != y.AssetID.String() {
		return x.AssetID.String() < y.AssetID.String()
	}
	return x.Amount.LessThan(y.Amount)
}

func assertRewardsMatch(t *testing.T, expected, actual []entities.Reward) {
	t.Helper()
	assert.Empty(t, cmp.Diff(expected, actual, cmpopts.SortSlices(rewardLessThan)))
}

func TestRewards(t *testing.T) {
	defer DeleteEverything()
	ps := sqlstore.NewParties(connectionSource)
	as := sqlstore.NewAssets(connectionSource)
	rs := sqlstore.NewRewards(connectionSource)
	bs := sqlstore.NewBlocks(connectionSource)
	block := addTestBlock(t, bs)

	asset1 := addTestAsset(t, as, block)
	asset2 := addTestAsset(t, as, block)

	market1 := entities.MarketID{ID: "deadbeef"}
	market2 := entities.MarketID{ID: ""}
	party1 := addTestParty(t, ps, block)
	party2 := addTestParty(t, ps, block)

	party1ID := party1.ID.String()
	asset1ID := asset1.ID.String()
	party2ID := party2.ID.String()
	asset2ID := asset2.ID.String()

	now := time.Now()
	reward1 := addTestReward(t, rs, party1, asset1, market1, 1, "RewardTakerPaidFees", now, block)
	reward2 := addTestReward(t, rs, party1, asset2, market1, 2, "RewardMakerReceivedFees", now, block)
	reward3 := addTestReward(t, rs, party2, asset1, market2, 3, "GlobalReward", now, block)
	reward4 := addTestReward(t, rs, party2, asset2, market2, 4, "GlobalReward", now, block)
	reward5 := addTestReward(t, rs, party2, asset2, market2, 5, "GlobalReward", now, block)

	t.Run("GetAll", func(t *testing.T) {
		expected := []entities.Reward{reward1, reward2, reward3, reward4, reward5}
		actual, err := rs.GetAll(context.Background())
		require.NoError(t, err)
		assertRewardsMatch(t, expected, actual)
	})

	t.Run("GetByParty", func(t *testing.T) {
		expected := []entities.Reward{reward1, reward2}
		actual, err := rs.GetByOffset(context.Background(), &party1ID, nil, nil)
		require.NoError(t, err)
		assertRewardsMatch(t, expected, actual)
	})

	t.Run("GetByAsset", func(t *testing.T) {
		expected := []entities.Reward{reward1, reward3}
		actual, err := rs.GetByOffset(context.Background(), nil, &asset1ID, nil)
		require.NoError(t, err)
		assertRewardsMatch(t, expected, actual)
	})

	t.Run("GetByAssetAndParty", func(t *testing.T) {
		expected := []entities.Reward{reward1}
		actual, err := rs.GetByOffset(context.Background(), &party1ID, &asset1ID, nil)
		require.NoError(t, err)
		assertRewardsMatch(t, expected, actual)
	})

	t.Run("GetPagination", func(t *testing.T) {
		expected := []entities.Reward{reward4, reward3, reward2}
		p := entities.OffsetPagination{Skip: 1, Limit: 3, Descending: true}
		actual, err := rs.GetByOffset(context.Background(), nil, nil, &p)
		require.NoError(t, err)
		assert.Equal(t, expected, actual) // Explicitly check the order on this one
	})

	t.Run("GetSummary", func(t *testing.T) {
		expected := []entities.RewardSummary{{
			AssetID: asset2.ID,
			PartyID: party2.ID,
			Amount:  decimal.NewFromInt(200),
		}}
		actual, err := rs.GetSummaries(context.Background(), &party2ID, &asset2ID)
		require.NoError(t, err)
		assert.Equal(t, expected, actual)
	})
}

func setupRewardsTest(t *testing.T) (*sqlstore.Blocks, *sqlstore.Rewards, *sqlstore.Parties, *sqlstore.Assets, sqlstore.Config) {
	t.Helper()
	bs := sqlstore.NewBlocks(connectionSource)
	rs := sqlstore.NewRewards(connectionSource)
	ps := sqlstore.NewParties(connectionSource)
	as := sqlstore.NewAssets(connectionSource)
	DeleteEverything()

	config := sqlstore.NewDefaultConfig()
	config.ConnectionConfig.Port = testDBPort

	return bs, rs, ps, as, config
}

func populateTestRewards(ctx context.Context, t *testing.T, bs *sqlstore.Blocks, ps *sqlstore.Parties, as *sqlstore.Assets, rs *sqlstore.Rewards) {
	partyID := entities.NewPartyID("89C701D1AE2819263E45538D0B25022988BC2508A02C654462D22E0AFB626A7D")
	assetID := entities.NewAssetID("8AA92225C32ADB54E527FCB1AEE2930CBADB4DF6F068AB2C2D667EB057EF00FA")

	rewards := []entities.Reward{
		{
			PartyID:        partyID,
			AssetID:        assetID,
			EpochID:        637,
			Amount:         decimal.NewFromFloat(1),
			PercentOfTotal: 100,
			RewardType:     "ACCOUNT_TYPE_UNSPECIFIED",
			Timestamp:      time.Date(2022, 3, 24, 15, 27, 28, 357155000, time.UTC),
			VegaTime:       time.Date(2022, 3, 24, 15, 27, 28, 357155000, time.UTC),
		},
		{
			PartyID:        partyID,
			AssetID:        assetID,
			EpochID:        642,
			Amount:         decimal.NewFromFloat(0),
			PercentOfTotal: 0,
			RewardType:     "ACCOUNT_TYPE_UNSPECIFIED",
			Timestamp:      time.Date(2022, 3, 24, 15, 28, 1, 508305000, time.UTC),
			VegaTime:       time.Date(2022, 3, 24, 15, 28, 1, 508305000, time.UTC),
		},
		{
			PartyID:        partyID,
			AssetID:        assetID,
			EpochID:        643,
			Amount:         decimal.NewFromFloat(1),
			PercentOfTotal: 100,
			RewardType:     "ACCOUNT_TYPE_UNSPECIFIED",
			Timestamp:      time.Date(2022, 3, 24, 15, 28, 8, 168980000, time.UTC),
			VegaTime:       time.Date(2022, 3, 24, 15, 28, 8, 168980000, time.UTC),
		},
		{
			PartyID:        partyID,
			AssetID:        assetID,
			EpochID:        737,
			Amount:         decimal.NewFromFloat(1),
			PercentOfTotal: 100,
			RewardType:     "ACCOUNT_TYPE_UNSPECIFIED",
			Timestamp:      time.Date(2022, 3, 24, 15, 38, 22, 855711000, time.UTC),
			VegaTime:       time.Date(2022, 3, 24, 15, 38, 22, 855711000, time.UTC),
		},
		{
			PartyID:        partyID,
			AssetID:        assetID,
			EpochID:        741,
			Amount:         decimal.NewFromFloat(5),
			PercentOfTotal: 62.5,
			RewardType:     "ACCOUNT_TYPE_UNSPECIFIED",
			Timestamp:      time.Date(2022, 3, 24, 15, 38, 49, 338318000, time.UTC),
			VegaTime:       time.Date(2022, 3, 24, 15, 38, 49, 338318000, time.UTC),
		},
		{
			PartyID:        partyID,
			AssetID:        assetID,
			EpochID:        744,
			Amount:         decimal.NewFromFloat(1),
			PercentOfTotal: 33.33333333333333,
			RewardType:     "ACCOUNT_TYPE_UNSPECIFIED",
			Timestamp:      time.Date(2022, 3, 24, 15, 39, 9, 595917000, time.UTC),
			VegaTime:       time.Date(2022, 3, 24, 15, 39, 9, 595917000, time.UTC),
		},
		{
			PartyID:        partyID,
			AssetID:        assetID,
			EpochID:        747,
			Amount:         decimal.NewFromFloat(6),
			PercentOfTotal: 60,
			RewardType:     "ACCOUNT_TYPE_UNSPECIFIED",
			Timestamp:      time.Date(2022, 3, 24, 15, 39, 29, 400906000, time.UTC),
			VegaTime:       time.Date(2022, 3, 24, 15, 39, 29, 400906000, time.UTC),
		},
		{
			PartyID:        partyID,
			AssetID:        assetID,
			EpochID:        757,
			Amount:         decimal.NewFromFloat(6),
			PercentOfTotal: 60,
			RewardType:     "ACCOUNT_TYPE_UNSPECIFIED",
			Timestamp:      time.Date(2022, 3, 24, 15, 40, 34, 750010000, time.UTC),
			VegaTime:       time.Date(2022, 3, 24, 15, 40, 34, 750010000, time.UTC),
		},
		{
			PartyID:        partyID,
			AssetID:        assetID,
			EpochID:        1025,
			Amount:         decimal.NewFromFloat(1),
			PercentOfTotal: 50,
			RewardType:     "ACCOUNT_TYPE_UNSPECIFIED",
			Timestamp:      time.Date(2022, 3, 24, 16, 9, 52, 556102000, time.UTC),
			VegaTime:       time.Date(2022, 3, 24, 16, 9, 52, 556102000, time.UTC),
		},
		{
			PartyID:        partyID,
			AssetID:        assetID,
			EpochID:        1027,
			Amount:         decimal.NewFromFloat(1),
			PercentOfTotal: 100,
			RewardType:     "ACCOUNT_TYPE_UNSPECIFIED",
			Timestamp:      time.Date(2022, 3, 24, 16, 10, 5, 602243000, time.UTC),
			VegaTime:       time.Date(2022, 3, 24, 16, 10, 5, 602243000, time.UTC),
		},
	}

	b := addTestBlock(t, bs)
	err := ps.Add(ctx, entities.Party{ID: partyID, VegaTime: &b.VegaTime})
	require.NoError(t, err)

	err = as.Add(ctx, entities.Asset{ID: assetID, VegaTime: b.VegaTime})
	require.NoError(t, err)

	for _, reward := range rewards {
		addTestBlockForTime(t, bs, reward.VegaTime)
		err := rs.Add(ctx, reward)
		require.NoError(t, err)
	}
}

func TestRewardsPagination(t *testing.T) {
	t.Run("should return all the rewards when no paging is provided", testRewardsCursorPaginationNoPagination)
	t.Run("should return the first page when the first limit is provided with no after cursor", testRewardsCursorPaginationFirstPage)
	t.Run("should return the last page when the last limit is provided with no before cursor", testRewardsCursorPaginationLastPage)
	t.Run("should return the page specified by the first limit and after cursor", testRewardsCursorPaginationFirstPageAfter)
	t.Run("should return the page specified by the last limit and before cursor", testRewardsCursorPaginationLastPageBefore)

	t.Run("should return all the rewards when no paging is provided", testRewardsCursorPaginationNoPaginationNewestFirst)
	t.Run("should return the first page when the first limit is provided with no after cursor", testRewardsCursorPaginationFirstPageNewestFirst)
	t.Run("should return the last page when the last limit is provided with no before cursor", testRewardsCursorPaginationLastPageNewestFirst)
	t.Run("should return the page specified by the first limit and after cursor", testRewardsCursorPaginationFirstPageAfterNewestFirst)
	t.Run("should return the page specified by the last limit and before cursor", testRewardsCursorPaginationLastPageBeforeNewestFirst)
}

func testRewardsCursorPaginationNoPagination(t *testing.T) {
	bs, rs, ps, as, _ := setupRewardsTest(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	populateTestRewards(ctx, t, bs, ps, as, rs)
	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, false)
	require.NoError(t, err)
	partyID := "89c701d1ae2819263e45538d0b25022988bc2508a02c654462d22e0afb626a7d"
	assetID := "8aa92225c32adb54e527fcb1aee2930cbadb4df6f068ab2c2d667eb057ef00fa"

	got, pageInfo, err := rs.GetByCursor(ctx, &partyID, &assetID, pagination)
	assert.NoError(t, err)
	assert.Equal(t, 10, len(got))
	assert.Equal(t, int64(637), got[0].EpochID)
	assert.Equal(t, int64(1027), got[len(got)-1].EpochID)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     entities.NewCursor(entities.RewardCursor{PartyID: partyID, AssetID: assetID, EpochID: 637}.String()).Encode(),
		EndCursor:       entities.NewCursor(entities.RewardCursor{PartyID: partyID, AssetID: assetID, EpochID: 1027}.String()).Encode(),
	}, pageInfo)
}

func testRewardsCursorPaginationFirstPage(t *testing.T) {
	bs, rs, ps, as, _ := setupRewardsTest(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	populateTestRewards(ctx, t, bs, ps, as, rs)
	first := int32(3)
	pagination, err := entities.NewCursorPagination(&first, nil, nil, nil, false)
	require.NoError(t, err)
	partyID := "89c701d1ae2819263e45538d0b25022988bc2508a02c654462d22e0afb626a7d"
	assetID := "8aa92225c32adb54e527fcb1aee2930cbadb4df6f068ab2c2d667eb057ef00fa"

	got, pageInfo, err := rs.GetByCursor(ctx, &partyID, &assetID, pagination)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(got))
	assert.Equal(t, int64(637), got[0].EpochID)
	assert.Equal(t, int64(643), got[len(got)-1].EpochID)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: false,
		StartCursor:     entities.NewCursor(entities.RewardCursor{PartyID: partyID, AssetID: assetID, EpochID: 637}.String()).Encode(),
		EndCursor:       entities.NewCursor(entities.RewardCursor{PartyID: partyID, AssetID: assetID, EpochID: 643}.String()).Encode(),
	}, pageInfo)
}

func testRewardsCursorPaginationLastPage(t *testing.T) {
	bs, rs, ps, as, _ := setupRewardsTest(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	populateTestRewards(ctx, t, bs, ps, as, rs)
	last := int32(3)
	pagination, err := entities.NewCursorPagination(nil, nil, &last, nil, false)
	require.NoError(t, err)
	partyID := "89c701d1ae2819263e45538d0b25022988bc2508a02c654462d22e0afb626a7d"
	assetID := "8aa92225c32adb54e527fcb1aee2930cbadb4df6f068ab2c2d667eb057ef00fa"

	got, pageInfo, err := rs.GetByCursor(ctx, &partyID, &assetID, pagination)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(got))
	assert.Equal(t, int64(757), got[0].EpochID)
	assert.Equal(t, int64(1027), got[len(got)-1].EpochID)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: true,
		StartCursor:     entities.NewCursor(entities.RewardCursor{PartyID: partyID, AssetID: assetID, EpochID: 757}.String()).Encode(),
		EndCursor:       entities.NewCursor(entities.RewardCursor{PartyID: partyID, AssetID: assetID, EpochID: 1027}.String()).Encode(),
	}, pageInfo)
}

func testRewardsCursorPaginationFirstPageAfter(t *testing.T) {
	bs, rs, ps, as, _ := setupRewardsTest(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	populateTestRewards(ctx, t, bs, ps, as, rs)
	partyID := "89c701d1ae2819263e45538d0b25022988bc2508a02c654462d22e0afb626a7d"
	assetID := "8aa92225c32adb54e527fcb1aee2930cbadb4df6f068ab2c2d667eb057ef00fa"

	first := int32(3)
	after := entities.NewCursor(entities.RewardCursor{PartyID: partyID, AssetID: assetID, EpochID: 643}.String()).Encode()
	pagination, err := entities.NewCursorPagination(&first, &after, nil, nil, false)
	require.NoError(t, err)

	got, pageInfo, err := rs.GetByCursor(ctx, &partyID, &assetID, pagination)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(got))
	assert.Equal(t, int64(737), got[0].EpochID)
	assert.Equal(t, int64(744), got[len(got)-1].EpochID)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     entities.NewCursor(entities.RewardCursor{PartyID: partyID, AssetID: assetID, EpochID: 737}.String()).Encode(),
		EndCursor:       entities.NewCursor(entities.RewardCursor{PartyID: partyID, AssetID: assetID, EpochID: 744}.String()).Encode(),
	}, pageInfo)
}

func testRewardsCursorPaginationLastPageBefore(t *testing.T) {
	bs, rs, ps, as, _ := setupRewardsTest(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	populateTestRewards(ctx, t, bs, ps, as, rs)
	partyID := "89c701d1ae2819263e45538d0b25022988bc2508a02c654462d22e0afb626a7d"
	assetID := "8aa92225c32adb54e527fcb1aee2930cbadb4df6f068ab2c2d667eb057ef00fa"

	last := int32(3)
	before := entities.NewCursor(entities.RewardCursor{PartyID: partyID, AssetID: assetID, EpochID: 757}.String()).Encode()
	pagination, err := entities.NewCursorPagination(nil, nil, &last, &before, false)
	require.NoError(t, err)
	got, pageInfo, err := rs.GetByCursor(ctx, &partyID, &assetID, pagination)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(got))
	assert.Equal(t, int64(741), got[0].EpochID)
	assert.Equal(t, int64(747), got[len(got)-1].EpochID)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     entities.NewCursor(entities.RewardCursor{PartyID: partyID, AssetID: assetID, EpochID: 741}.String()).Encode(),
		EndCursor:       entities.NewCursor(entities.RewardCursor{PartyID: partyID, AssetID: assetID, EpochID: 747}.String()).Encode(),
	}, pageInfo)
}

func testRewardsCursorPaginationNoPaginationNewestFirst(t *testing.T) {
	bs, rs, ps, as, _ := setupRewardsTest(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	populateTestRewards(ctx, t, bs, ps, as, rs)
	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, true)
	require.NoError(t, err)
	partyID := "89c701d1ae2819263e45538d0b25022988bc2508a02c654462d22e0afb626a7d"
	assetID := "8aa92225c32adb54e527fcb1aee2930cbadb4df6f068ab2c2d667eb057ef00fa"

	got, pageInfo, err := rs.GetByCursor(ctx, &partyID, &assetID, pagination)
	assert.NoError(t, err)
	assert.Equal(t, 10, len(got))
	assert.Equal(t, int64(1027), got[0].EpochID)
	assert.Equal(t, int64(637), got[len(got)-1].EpochID)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     entities.NewCursor(entities.RewardCursor{PartyID: partyID, AssetID: assetID, EpochID: 1027}.String()).Encode(),
		EndCursor:       entities.NewCursor(entities.RewardCursor{PartyID: partyID, AssetID: assetID, EpochID: 637}.String()).Encode(),
	}, pageInfo)
}

func testRewardsCursorPaginationFirstPageNewestFirst(t *testing.T) {
	bs, rs, ps, as, _ := setupRewardsTest(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	populateTestRewards(ctx, t, bs, ps, as, rs)
	first := int32(3)
	pagination, err := entities.NewCursorPagination(&first, nil, nil, nil, true)
	require.NoError(t, err)
	partyID := "89c701d1ae2819263e45538d0b25022988bc2508a02c654462d22e0afb626a7d"
	assetID := "8aa92225c32adb54e527fcb1aee2930cbadb4df6f068ab2c2d667eb057ef00fa"

	got, pageInfo, err := rs.GetByCursor(ctx, &partyID, &assetID, pagination)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(got))
	assert.Equal(t, int64(1027), got[0].EpochID)
	assert.Equal(t, int64(757), got[len(got)-1].EpochID)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: false,
		StartCursor:     entities.NewCursor(entities.RewardCursor{PartyID: partyID, AssetID: assetID, EpochID: 1027}.String()).Encode(),
		EndCursor:       entities.NewCursor(entities.RewardCursor{PartyID: partyID, AssetID: assetID, EpochID: 757}.String()).Encode(),
	}, pageInfo)
}

func testRewardsCursorPaginationLastPageNewestFirst(t *testing.T) {
	bs, rs, ps, as, _ := setupRewardsTest(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	populateTestRewards(ctx, t, bs, ps, as, rs)
	last := int32(3)
	pagination, err := entities.NewCursorPagination(nil, nil, &last, nil, true)
	require.NoError(t, err)
	partyID := "89c701d1ae2819263e45538d0b25022988bc2508a02c654462d22e0afb626a7d"
	assetID := "8aa92225c32adb54e527fcb1aee2930cbadb4df6f068ab2c2d667eb057ef00fa"

	got, pageInfo, err := rs.GetByCursor(ctx, &partyID, &assetID, pagination)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(got))
	assert.Equal(t, int64(643), got[0].EpochID)
	assert.Equal(t, int64(637), got[len(got)-1].EpochID)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: true,
		StartCursor:     entities.NewCursor(entities.RewardCursor{PartyID: partyID, AssetID: assetID, EpochID: 643}.String()).Encode(),
		EndCursor:       entities.NewCursor(entities.RewardCursor{PartyID: partyID, AssetID: assetID, EpochID: 637}.String()).Encode(),
	}, pageInfo)
}

func testRewardsCursorPaginationFirstPageAfterNewestFirst(t *testing.T) {
	bs, rs, ps, as, _ := setupRewardsTest(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	populateTestRewards(ctx, t, bs, ps, as, rs)
	partyID := "89c701d1ae2819263e45538d0b25022988bc2508a02c654462d22e0afb626a7d"
	assetID := "8aa92225c32adb54e527fcb1aee2930cbadb4df6f068ab2c2d667eb057ef00fa"

	first := int32(3)
	after := entities.NewCursor(entities.RewardCursor{PartyID: partyID, AssetID: assetID, EpochID: 757}.String()).Encode()
	pagination, err := entities.NewCursorPagination(&first, &after, nil, nil, true)
	require.NoError(t, err)

	got, pageInfo, err := rs.GetByCursor(ctx, &partyID, &assetID, pagination)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(got))
	assert.Equal(t, int64(747), got[0].EpochID)
	assert.Equal(t, int64(741), got[len(got)-1].EpochID)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     entities.NewCursor(entities.RewardCursor{PartyID: partyID, AssetID: assetID, EpochID: 747}.String()).Encode(),
		EndCursor:       entities.NewCursor(entities.RewardCursor{PartyID: partyID, AssetID: assetID, EpochID: 741}.String()).Encode(),
	}, pageInfo)
}

func testRewardsCursorPaginationLastPageBeforeNewestFirst(t *testing.T) {
	bs, rs, ps, as, _ := setupRewardsTest(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	populateTestRewards(ctx, t, bs, ps, as, rs)
	partyID := "89c701d1ae2819263e45538d0b25022988bc2508a02c654462d22e0afb626a7d"
	assetID := "8aa92225c32adb54e527fcb1aee2930cbadb4df6f068ab2c2d667eb057ef00fa"

	last := int32(3)
	before := entities.NewCursor(entities.RewardCursor{PartyID: partyID, AssetID: assetID, EpochID: 643}.String()).Encode()
	pagination, err := entities.NewCursorPagination(nil, nil, &last, &before, true)
	require.NoError(t, err)
	got, pageInfo, err := rs.GetByCursor(ctx, &partyID, &assetID, pagination)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(got))
	assert.Equal(t, int64(744), got[0].EpochID)
	assert.Equal(t, int64(737), got[len(got)-1].EpochID)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     entities.NewCursor(entities.RewardCursor{PartyID: partyID, AssetID: assetID, EpochID: 744}.String()).Encode(),
		EndCursor:       entities.NewCursor(entities.RewardCursor{PartyID: partyID, AssetID: assetID, EpochID: 737}.String()).Encode(),
	}, pageInfo)
}
