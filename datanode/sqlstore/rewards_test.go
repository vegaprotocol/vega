// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.DATANODE file and at https://www.mariadb.com/bsl11.
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
	"fmt"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/sqlstore"
	"code.vegaprotocol.io/vega/libs/num"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func addTestReward(t *testing.T,
	ctx context.Context,
	rs *sqlstore.Rewards,
	party entities.Party,
	asset entities.Asset,
	marketID entities.MarketID,
	epochID int64,
	rewardType string,
	timestamp time.Time,
	block entities.Block,
	seqNum uint64,
	amount num.Decimal,
) entities.Reward {
	t.Helper()
	r := entities.Reward{
		PartyID:        party.ID,
		AssetID:        asset.ID,
		MarketID:       marketID,
		RewardType:     rewardType,
		EpochID:        epochID,
		Amount:         amount,
		PercentOfTotal: 0.2,
		Timestamp:      timestamp.Truncate(time.Microsecond),
		VegaTime:       block.VegaTime,
		SeqNum:         seqNum,
	}
	err := rs.Add(ctx, r)
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
	ctx, rollback := tempTransaction(t)
	defer rollback()
	ps := sqlstore.NewParties(connectionSource)
	as := sqlstore.NewAssets(connectionSource)
	rs := sqlstore.NewRewards(connectionSource)
	bs := sqlstore.NewBlocks(connectionSource)
	block := addTestBlock(t, ctx, bs)

	asset1 := addTestAsset(t, ctx, as, block)
	asset2 := addTestAsset(t, ctx, as, block)

	market1 := entities.MarketID("deadbeef")
	market2 := entities.MarketID("")
	party1 := addTestParty(t, ctx, ps, block)
	party2 := addTestParty(t, ctx, ps, block)

	party2ID := party2.ID.String()
	asset2ID := asset2.ID.String()

	now := time.Now()
	amount := num.DecimalFromInt64(100)
	reward1 := addTestReward(t, ctx, rs, party1, asset1, market1, 1, "RewardMakerPaidFees", now, block, 1, amount)
	reward2 := addTestReward(t, ctx, rs, party1, asset2, market1, 2, "RewardMakerReceivedFees", now, block, 2, amount)
	reward3 := addTestReward(t, ctx, rs, party2, asset1, market2, 3, "GlobalReward", now, block, 3, amount)
	reward4 := addTestReward(t, ctx, rs, party2, asset2, market2, 4, "GlobalReward", now, block, 4, amount)
	reward5 := addTestReward(t, ctx, rs, party2, asset2, market2, 5, "GlobalReward", now, block, 5, amount)

	t.Run("GetAll", func(t *testing.T) {
		expected := []entities.Reward{reward1, reward2, reward3, reward4, reward5}
		actual, err := rs.GetAll(ctx)
		require.NoError(t, err)
		assertRewardsMatch(t, expected, actual)
	})

	t.Run("GetSummary", func(t *testing.T) {
		expected := []entities.RewardSummary{{
			AssetID: asset2.ID,
			PartyID: party2.ID,
			Amount:  decimal.NewFromInt(200),
		}}
		actual, err := rs.GetSummaries(ctx, &party2ID, &asset2ID)
		require.NoError(t, err)
		assert.Equal(t, expected, actual)
	})
}

func TestEpochRewardSummary(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()
	ps := sqlstore.NewParties(connectionSource)
	as := sqlstore.NewAssets(connectionSource)
	rs := sqlstore.NewRewards(connectionSource)
	bs := sqlstore.NewBlocks(connectionSource)
	block := addTestBlock(t, ctx, bs)

	asset1 := addTestAsset(t, ctx, as, block)
	asset2 := addTestAsset(t, ctx, as, block)

	market1 := entities.MarketID("deadbeef")
	market2 := entities.MarketID("")
	party1 := addTestParty(t, ctx, ps, block)
	party2 := addTestParty(t, ctx, ps, block)
	party3 := addTestParty(t, ctx, ps, block)

	now := time.Now()
	// rewards for epoch1
	addTestReward(t, ctx, rs, party1, asset1, market1, 1, "RewardMakerPaidFees", now, block, 1, num.DecimalFromInt64(100))
	addTestReward(t, ctx, rs, party2, asset1, market1, 1, "RewardMakerPaidFees", now, block, 2, num.DecimalFromInt64(200))
	addTestReward(t, ctx, rs, party3, asset1, market1, 1, "RewardMakerPaidFees", now, block, 3, num.DecimalFromInt64(300))
	addTestReward(t, ctx, rs, party1, asset1, market2, 1, "RewardMakerPaidFees", now, block, 4, num.DecimalFromInt64(110))
	addTestReward(t, ctx, rs, party2, asset1, market2, 1, "RewardMakerPaidFees", now, block, 5, num.DecimalFromInt64(220))
	addTestReward(t, ctx, rs, party3, asset1, market2, 1, "RewardMakerPaidFees", now, block, 6, num.DecimalFromInt64(330))
	addTestReward(t, ctx, rs, party1, asset2, market1, 1, "RewardMakerPaidFees", now, block, 7, num.DecimalFromInt64(400))
	addTestReward(t, ctx, rs, party2, asset2, market1, 1, "RewardMakerPaidFees", now, block, 8, num.DecimalFromInt64(500))
	addTestReward(t, ctx, rs, party3, asset2, market1, 1, "RewardMakerPaidFees", now, block, 9, num.DecimalFromInt64(600))
	addTestReward(t, ctx, rs, party1, asset2, market2, 1, "RewardMakerPaidFees", now, block, 10, num.DecimalFromInt64(410))
	addTestReward(t, ctx, rs, party2, asset2, market2, 1, "RewardMakerPaidFees", now, block, 11, num.DecimalFromInt64(520))
	addTestReward(t, ctx, rs, party3, asset2, market2, 1, "RewardMakerPaidFees", now, block, 12, num.DecimalFromInt64(630))
	addTestReward(t, ctx, rs, party1, asset1, market1, 1, "RewardMakerReceivedFees", now, block, 13, num.DecimalFromInt64(1000))
	addTestReward(t, ctx, rs, party2, asset1, market1, 1, "RewardMakerReceivedFees", now, block, 14, num.DecimalFromInt64(2000))
	addTestReward(t, ctx, rs, party3, asset1, market1, 1, "RewardMakerReceivedFees", now, block, 15, num.DecimalFromInt64(3000))
	addTestReward(t, ctx, rs, party1, asset1, market2, 1, "GlobalReward", now, block, 16, num.DecimalFromInt64(1100))
	addTestReward(t, ctx, rs, party2, asset1, market2, 1, "GlobalReward", now, block, 17, num.DecimalFromInt64(2200))
	addTestReward(t, ctx, rs, party3, asset1, market2, 1, "GlobalReward", now, block, 18, num.DecimalFromInt64(3300))
	addTestReward(t, ctx, rs, party1, asset2, market1, 1, "RewardMakerReceivedFees", now, block, 19, num.DecimalFromInt64(4000))
	addTestReward(t, ctx, rs, party2, asset2, market1, 1, "RewardMakerReceivedFees", now, block, 20, num.DecimalFromInt64(5000))
	addTestReward(t, ctx, rs, party3, asset2, market1, 1, "RewardMakerReceivedFees", now, block, 21, num.DecimalFromInt64(6000))
	addTestReward(t, ctx, rs, party1, asset2, market2, 1, "GlobalReward", now, block, 22, num.DecimalFromInt64(4100))
	addTestReward(t, ctx, rs, party2, asset2, market2, 1, "GlobalReward", now, block, 23, num.DecimalFromInt64(5200))
	addTestReward(t, ctx, rs, party3, asset2, market2, 1, "GlobalReward", now, block, 24, num.DecimalFromInt64(6300))

	// rewards for epoch2
	addTestReward(t, ctx, rs, party1, asset1, market1, 2, "RewardMakerPaidFees", now, block, 25, num.DecimalFromInt64(10000))
	addTestReward(t, ctx, rs, party2, asset1, market1, 2, "RewardMakerPaidFees", now, block, 26, num.DecimalFromInt64(20000))
	addTestReward(t, ctx, rs, party3, asset1, market1, 2, "RewardMakerPaidFees", now, block, 27, num.DecimalFromInt64(30000))
	addTestReward(t, ctx, rs, party1, asset1, market2, 2, "RewardMakerPaidFees", now, block, 28, num.DecimalFromInt64(11000))
	addTestReward(t, ctx, rs, party2, asset1, market2, 2, "RewardMakerPaidFees", now, block, 29, num.DecimalFromInt64(22000))
	addTestReward(t, ctx, rs, party3, asset1, market2, 2, "RewardMakerPaidFees", now, block, 30, num.DecimalFromInt64(33000))
	addTestReward(t, ctx, rs, party1, asset2, market1, 2, "RewardMakerPaidFees", now, block, 31, num.DecimalFromInt64(40000))
	addTestReward(t, ctx, rs, party2, asset2, market1, 2, "RewardMakerPaidFees", now, block, 32, num.DecimalFromInt64(50000))
	addTestReward(t, ctx, rs, party3, asset2, market1, 2, "RewardMakerPaidFees", now, block, 33, num.DecimalFromInt64(60000))
	addTestReward(t, ctx, rs, party1, asset2, market2, 2, "RewardMakerPaidFees", now, block, 34, num.DecimalFromInt64(41000))
	addTestReward(t, ctx, rs, party2, asset2, market2, 2, "RewardMakerPaidFees", now, block, 35, num.DecimalFromInt64(52000))
	addTestReward(t, ctx, rs, party3, asset2, market2, 2, "RewardMakerPaidFees", now, block, 36, num.DecimalFromInt64(63000))
	addTestReward(t, ctx, rs, party1, asset1, market1, 2, "RewardMakerReceivedFees", now, block, 37, num.DecimalFromInt64(100000))
	addTestReward(t, ctx, rs, party2, asset1, market1, 2, "RewardMakerReceivedFees", now, block, 38, num.DecimalFromInt64(200000))
	addTestReward(t, ctx, rs, party3, asset1, market1, 2, "RewardMakerReceivedFees", now, block, 39, num.DecimalFromInt64(300000))
	addTestReward(t, ctx, rs, party1, asset1, market2, 2, "GlobalReward", now, block, 40, num.DecimalFromInt64(110000))
	addTestReward(t, ctx, rs, party2, asset1, market2, 2, "GlobalReward", now, block, 41, num.DecimalFromInt64(220000))
	addTestReward(t, ctx, rs, party3, asset1, market2, 2, "GlobalReward", now, block, 42, num.DecimalFromInt64(330000))
	addTestReward(t, ctx, rs, party1, asset2, market1, 2, "RewardMakerReceivedFees", now, block, 43, num.DecimalFromInt64(400000))
	addTestReward(t, ctx, rs, party2, asset2, market1, 2, "RewardMakerReceivedFees", now, block, 44, num.DecimalFromInt64(500000))
	addTestReward(t, ctx, rs, party3, asset2, market1, 2, "RewardMakerReceivedFees", now, block, 45, num.DecimalFromInt64(600000))
	addTestReward(t, ctx, rs, party1, asset2, market2, 2, "GlobalReward", now, block, 46, num.DecimalFromInt64(410000))
	addTestReward(t, ctx, rs, party2, asset2, market2, 2, "GlobalReward", now, block, 47, num.DecimalFromInt64(520000))
	addTestReward(t, ctx, rs, party3, asset2, market2, 2, "GlobalReward", now, block, 48, num.DecimalFromInt64(630000))

	first := int32(1000)
	pagination, _ := entities.NewCursorPagination(&first, nil, nil, nil, false)
	summaries, _, _ := rs.GetEpochSummaries(ctx, nil, nil, pagination)

	// we expect to get all sumarries because we defined no from/to
	// so 16 summaries
	// epoch1 / asset1 / market1 / RewardMakerPaidFees = 600
	// epoch1 / asset1 / market2 / RewardMakerPaidFees = 660
	// epoch1 / asset2 / market1 / RewardMakerPaidFees = 1500
	// epoch1 / asset2 / market2 / RewardMakerPaidFees = 1560
	// epoch1 / asset1 / market1 / RewardMakerReceivedFees  = 6000
	// epoch1 / asset1 / market2 / GlobalReward  = 6600
	// epoch1 / asset2 / market1 / RewardMakerPaidFees  = 15000
	// epoch1 / asset2 / market2 / GlobalReward  = 15600

	// epoch2 / asset1 / market1 / RewardMakerPaidFees  = 60000
	// epoch2 / asset1 / market2 / RewardMakerPaidFees = 66000
	// epoch2 / asset2 / market1 / RewardMakerPaidFees = 150000
	// epoch2 / asset2 / market2 / RewardMakerPaidFees = 156000
	// epoch2 / asset1 / market1 / RewardMakerReceivedFees  = 600000
	// epoch2 / asset1 / market2 / GlobalReward  = 660000
	// epoch2 / asset2 / market1 / RewardMakerPaidFees  = 1500000
	// epoch2 / asset2 / market2 / GlobalReward  = 1560000

	require.Equal(t, 16, len(summaries))
	verifyRewardsForEpoch(t, summaries, 1, asset1.ID.String(), asset2.ID.String())
	verifyRewardsForEpoch(t, summaries, 2, asset1.ID.String(), asset2.ID.String())

	// now request with from = 1 with no to, expect the same result
	from := uint64(1)
	summaries, _, _ = rs.GetEpochSummaries(ctx, &from, nil, pagination)
	require.Equal(t, 16, len(summaries))
	verifyRewardsForEpoch(t, summaries, 1, asset1.ID.String(), asset2.ID.String())
	verifyRewardsForEpoch(t, summaries, 2, asset1.ID.String(), asset2.ID.String())

	// now request with from = nil and to = 2, expect the same result
	to := uint64(2)
	summaries, _, _ = rs.GetEpochSummaries(ctx, nil, &to, pagination)
	require.Equal(t, 16, len(summaries))
	verifyRewardsForEpoch(t, summaries, 1, asset1.ID.String(), asset2.ID.String())
	verifyRewardsForEpoch(t, summaries, 2, asset1.ID.String(), asset2.ID.String())

	// now request from = 2 to = nil expect only epoch 2
	from = 2
	summaries, _, _ = rs.GetEpochSummaries(ctx, &from, nil, pagination)
	require.Equal(t, 8, len(summaries))
	verifyRewardsForEpoch(t, summaries, 2, asset1.ID.String(), asset2.ID.String())

	// now request to = 1 from = nil expect only epoch 1
	to = 1
	summaries, _, _ = rs.GetEpochSummaries(ctx, nil, &to, pagination)
	require.Equal(t, 8, len(summaries))
	verifyRewardsForEpoch(t, summaries, 1, asset1.ID.String(), asset2.ID.String())

	// now request from = 1 and to = 1
	from = 1
	summaries, _, _ = rs.GetEpochSummaries(ctx, &from, &to, pagination)
	require.Equal(t, 8, len(summaries))
	verifyRewardsForEpoch(t, summaries, 1, asset1.ID.String(), asset2.ID.String())
}

func verifyRewardsForEpoch(t *testing.T, summaries []entities.EpochRewardSummary, epoch int, asset1, asset2 string) {
	t.Helper()
	m := make(map[string]string, len(summaries))
	for _, s := range summaries {
		id := fmt.Sprintf("%d_%s_%s_%s", s.EpochID, s.AssetID, s.MarketID, s.RewardType)
		m[id] = s.Amount.String()
	}
	if epoch == 1 {
		require.Equal(t, "600", m["1_"+asset1+"_deadbeef_RewardMakerPaidFees"])
		require.Equal(t, "660", m["1_"+asset1+"__RewardMakerPaidFees"])
		require.Equal(t, "1500", m["1_"+asset2+"_deadbeef_RewardMakerPaidFees"])
		require.Equal(t, "1560", m["1_"+asset2+"__RewardMakerPaidFees"])
		require.Equal(t, "6000", m["1_"+asset1+"_deadbeef_RewardMakerReceivedFees"])
		require.Equal(t, "6600", m["1_"+asset1+"__GlobalReward"])
		require.Equal(t, "15000", m["1_"+asset2+"_deadbeef_RewardMakerReceivedFees"])
		require.Equal(t, "15600", m["1_"+asset2+"__GlobalReward"])
	} else if epoch == 2 {
		require.Equal(t, "60000", m["2_"+asset1+"_deadbeef_RewardMakerPaidFees"])
		require.Equal(t, "66000", m["2_"+asset1+"__RewardMakerPaidFees"])
		require.Equal(t, "150000", m["2_"+asset2+"_deadbeef_RewardMakerPaidFees"])
		require.Equal(t, "156000", m["2_"+asset2+"__RewardMakerPaidFees"])
		require.Equal(t, "600000", m["2_"+asset1+"_deadbeef_RewardMakerReceivedFees"])
		require.Equal(t, "660000", m["2_"+asset1+"__GlobalReward"])
		require.Equal(t, "1500000", m["2_"+asset2+"_deadbeef_RewardMakerReceivedFees"])
		require.Equal(t, "1560000", m["2_"+asset2+"__GlobalReward"])
	}
}

func setupRewardsTest(t *testing.T) (*sqlstore.Blocks, *sqlstore.Rewards, *sqlstore.Parties, *sqlstore.Assets) {
	t.Helper()
	bs := sqlstore.NewBlocks(connectionSource)
	rs := sqlstore.NewRewards(connectionSource)
	ps := sqlstore.NewParties(connectionSource)
	as := sqlstore.NewAssets(connectionSource)

	return bs, rs, ps, as
}

func populateTestRewards(ctx context.Context, t *testing.T, bs *sqlstore.Blocks, ps *sqlstore.Parties, as *sqlstore.Assets, rs *sqlstore.Rewards) {
	t.Helper()
	partyID := entities.PartyID("89C701D1AE2819263E45538D0B25022988BC2508A02C654462D22E0AFB626A7D")
	assetID := entities.AssetID("8AA92225C32ADB54E527FCB1AEE2930CBADB4DF6F068AB2C2D667EB057EF00FA")

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
			SeqNum:         1,
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
			SeqNum:         1,
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
			SeqNum:         1,
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
			SeqNum:         1,
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
			SeqNum:         1,
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
			SeqNum:         1,
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
			SeqNum:         1,
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
			SeqNum:         1,
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
			SeqNum:         1,
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
			SeqNum:         1,
		},
	}

	b := addTestBlock(t, ctx, bs)
	err := ps.Add(ctx, entities.Party{ID: partyID, VegaTime: &b.VegaTime})
	require.NoError(t, err)

	err = as.Add(ctx, entities.Asset{ID: assetID, VegaTime: b.VegaTime})
	require.NoError(t, err)

	for _, reward := range rewards {
		addTestBlockForTime(t, ctx, bs, reward.VegaTime)
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
	bs, rs, ps, as := setupRewardsTest(t)
	ctx, rollback := tempTransaction(t)
	defer rollback()
	populateTestRewards(ctx, t, bs, ps, as, rs)
	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, false)
	require.NoError(t, err)
	partyID := "89c701d1ae2819263e45538d0b25022988bc2508a02c654462d22e0afb626a7d"
	assetID := "8aa92225c32adb54e527fcb1aee2930cbadb4df6f068ab2c2d667eb057ef00fa"

	got, pageInfo, err := rs.GetByCursor(ctx, &partyID, &assetID, nil, nil, pagination)
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
	bs, rs, ps, as := setupRewardsTest(t)
	ctx, rollback := tempTransaction(t)
	defer rollback()
	populateTestRewards(ctx, t, bs, ps, as, rs)
	first := int32(3)
	pagination, err := entities.NewCursorPagination(&first, nil, nil, nil, false)
	require.NoError(t, err)
	partyID := "89c701d1ae2819263e45538d0b25022988bc2508a02c654462d22e0afb626a7d"
	assetID := "8aa92225c32adb54e527fcb1aee2930cbadb4df6f068ab2c2d667eb057ef00fa"

	got, pageInfo, err := rs.GetByCursor(ctx, &partyID, &assetID, nil, nil, pagination)
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
	bs, rs, ps, as := setupRewardsTest(t)
	ctx, rollback := tempTransaction(t)
	defer rollback()
	populateTestRewards(ctx, t, bs, ps, as, rs)
	last := int32(3)
	pagination, err := entities.NewCursorPagination(nil, nil, &last, nil, false)
	require.NoError(t, err)
	partyID := "89c701d1ae2819263e45538d0b25022988bc2508a02c654462d22e0afb626a7d"
	assetID := "8aa92225c32adb54e527fcb1aee2930cbadb4df6f068ab2c2d667eb057ef00fa"

	got, pageInfo, err := rs.GetByCursor(ctx, &partyID, &assetID, nil, nil, pagination)
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
	bs, rs, ps, as := setupRewardsTest(t)
	ctx, rollback := tempTransaction(t)
	defer rollback()
	populateTestRewards(ctx, t, bs, ps, as, rs)
	partyID := "89c701d1ae2819263e45538d0b25022988bc2508a02c654462d22e0afb626a7d"
	assetID := "8aa92225c32adb54e527fcb1aee2930cbadb4df6f068ab2c2d667eb057ef00fa"

	first := int32(3)
	after := entities.NewCursor(entities.RewardCursor{PartyID: partyID, AssetID: assetID, EpochID: 643}.String()).Encode()
	pagination, err := entities.NewCursorPagination(&first, &after, nil, nil, false)
	require.NoError(t, err)

	got, pageInfo, err := rs.GetByCursor(ctx, &partyID, &assetID, nil, nil, pagination)
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
	bs, rs, ps, as := setupRewardsTest(t)
	ctx, rollback := tempTransaction(t)
	defer rollback()
	populateTestRewards(ctx, t, bs, ps, as, rs)
	partyID := "89c701d1ae2819263e45538d0b25022988bc2508a02c654462d22e0afb626a7d"
	assetID := "8aa92225c32adb54e527fcb1aee2930cbadb4df6f068ab2c2d667eb057ef00fa"

	last := int32(3)
	before := entities.NewCursor(entities.RewardCursor{PartyID: partyID, AssetID: assetID, EpochID: 757}.String()).Encode()
	pagination, err := entities.NewCursorPagination(nil, nil, &last, &before, false)
	require.NoError(t, err)
	got, pageInfo, err := rs.GetByCursor(ctx, &partyID, &assetID, nil, nil, pagination)
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
	bs, rs, ps, as := setupRewardsTest(t)
	ctx, rollback := tempTransaction(t)
	defer rollback()
	populateTestRewards(ctx, t, bs, ps, as, rs)
	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, true)
	require.NoError(t, err)
	partyID := "89c701d1ae2819263e45538d0b25022988bc2508a02c654462d22e0afb626a7d"
	assetID := "8aa92225c32adb54e527fcb1aee2930cbadb4df6f068ab2c2d667eb057ef00fa"

	got, pageInfo, err := rs.GetByCursor(ctx, &partyID, &assetID, nil, nil, pagination)
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
	bs, rs, ps, as := setupRewardsTest(t)
	ctx, rollback := tempTransaction(t)
	defer rollback()
	populateTestRewards(ctx, t, bs, ps, as, rs)
	first := int32(3)
	pagination, err := entities.NewCursorPagination(&first, nil, nil, nil, true)
	require.NoError(t, err)
	partyID := "89c701d1ae2819263e45538d0b25022988bc2508a02c654462d22e0afb626a7d"
	assetID := "8aa92225c32adb54e527fcb1aee2930cbadb4df6f068ab2c2d667eb057ef00fa"

	got, pageInfo, err := rs.GetByCursor(ctx, &partyID, &assetID, nil, nil, pagination)
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
	bs, rs, ps, as := setupRewardsTest(t)
	ctx, rollback := tempTransaction(t)
	defer rollback()
	populateTestRewards(ctx, t, bs, ps, as, rs)
	last := int32(3)
	pagination, err := entities.NewCursorPagination(nil, nil, &last, nil, true)
	require.NoError(t, err)
	partyID := "89c701d1ae2819263e45538d0b25022988bc2508a02c654462d22e0afb626a7d"
	assetID := "8aa92225c32adb54e527fcb1aee2930cbadb4df6f068ab2c2d667eb057ef00fa"

	got, pageInfo, err := rs.GetByCursor(ctx, &partyID, &assetID, nil, nil, pagination)
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
	bs, rs, ps, as := setupRewardsTest(t)
	ctx, rollback := tempTransaction(t)
	defer rollback()
	populateTestRewards(ctx, t, bs, ps, as, rs)
	partyID := "89c701d1ae2819263e45538d0b25022988bc2508a02c654462d22e0afb626a7d"
	assetID := "8aa92225c32adb54e527fcb1aee2930cbadb4df6f068ab2c2d667eb057ef00fa"

	first := int32(3)
	after := entities.NewCursor(entities.RewardCursor{PartyID: partyID, AssetID: assetID, EpochID: 757}.String()).Encode()
	pagination, err := entities.NewCursorPagination(&first, &after, nil, nil, true)
	require.NoError(t, err)

	got, pageInfo, err := rs.GetByCursor(ctx, &partyID, &assetID, nil, nil, pagination)
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
	bs, rs, ps, as := setupRewardsTest(t)
	ctx, rollback := tempTransaction(t)
	defer rollback()
	populateTestRewards(ctx, t, bs, ps, as, rs)
	partyID := "89c701d1ae2819263e45538d0b25022988bc2508a02c654462d22e0afb626a7d"
	assetID := "8aa92225c32adb54e527fcb1aee2930cbadb4df6f068ab2c2d667eb057ef00fa"

	last := int32(3)
	before := entities.NewCursor(entities.RewardCursor{PartyID: partyID, AssetID: assetID, EpochID: 643}.String()).Encode()
	pagination, err := entities.NewCursorPagination(nil, nil, &last, &before, true)
	require.NoError(t, err)
	got, pageInfo, err := rs.GetByCursor(ctx, &partyID, &assetID, nil, nil, pagination)
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
