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
	"encoding/hex"
	"fmt"
	"math/rand"
	"reflect"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/sqlstore"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/ptr"

	"github.com/georgysavva/scany/pgxscan"
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
	txHash entities.TxHash,
	gameID *entities.GameID,
) entities.Reward {
	t.Helper()
	r := entities.Reward{
		PartyID:        party.ID,
		AssetID:        asset.ID,
		MarketID:       marketID,
		RewardType:     rewardType,
		EpochID:        epochID,
		Amount:         amount,
		QuantumAmount:  amount,
		PercentOfTotal: 0.2,
		Timestamp:      timestamp.Truncate(time.Microsecond),
		VegaTime:       block.VegaTime,
		SeqNum:         seqNum,
		TxHash:         txHash,
		GameID:         gameID,
	}
	require.NoError(t, rs.Add(ctx, r))
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
	ctx := tempTransaction(t)

	ps := sqlstore.NewParties(connectionSource)
	as := sqlstore.NewAssets(connectionSource)
	rs := sqlstore.NewRewards(ctx, connectionSource)
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
	reward1 := addTestReward(t, ctx, rs, party1, asset1, market1, 1, "RewardMakerPaidFees", now, block, 1, amount, generateTxHash(), nil)
	reward2 := addTestReward(t, ctx, rs, party1, asset2, market1, 2, "RewardMakerReceivedFees", now, block, 2, amount, generateTxHash(), nil)
	reward3 := addTestReward(t, ctx, rs, party2, asset1, market2, 3, "GlobalReward", now, block, 3, amount, generateTxHash(), nil)
	reward4 := addTestReward(t, ctx, rs, party2, asset2, market2, 4, "GlobalReward", now, block, 4, amount, generateTxHash(), nil)
	reward5 := addTestReward(t, ctx, rs, party2, asset2, market2, 5, "GlobalReward", now, block, 5, amount, generateTxHash(), nil)

	t.Run("GetAll", func(t *testing.T) {
		expected := []entities.Reward{reward1, reward2, reward3, reward4, reward5}
		actual, err := rs.GetAll(ctx)
		require.NoError(t, err)
		assertRewardsMatch(t, expected, actual)
	})

	t.Run("GetByTxHash", func(t *testing.T) {
		expected := []entities.Reward{reward2}
		actual, err := rs.GetByTxHash(ctx, reward2.TxHash)
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
	ctx := tempTransaction(t)

	ps := sqlstore.NewParties(connectionSource)
	as := sqlstore.NewAssets(connectionSource)
	rs := sqlstore.NewRewards(ctx, connectionSource)
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
	addTestReward(t, ctx, rs, party1, asset1, market1, 1, "RewardMakerPaidFees", now, block, 1, num.DecimalFromInt64(100), generateTxHash(), nil)
	addTestReward(t, ctx, rs, party2, asset1, market1, 1, "RewardMakerPaidFees", now, block, 2, num.DecimalFromInt64(200), generateTxHash(), nil)
	addTestReward(t, ctx, rs, party3, asset1, market1, 1, "RewardMakerPaidFees", now, block, 3, num.DecimalFromInt64(300), generateTxHash(), nil)
	addTestReward(t, ctx, rs, party1, asset1, market2, 1, "RewardMakerPaidFees", now, block, 4, num.DecimalFromInt64(110), generateTxHash(), nil)
	addTestReward(t, ctx, rs, party2, asset1, market2, 1, "RewardMakerPaidFees", now, block, 5, num.DecimalFromInt64(220), generateTxHash(), nil)
	addTestReward(t, ctx, rs, party3, asset1, market2, 1, "RewardMakerPaidFees", now, block, 6, num.DecimalFromInt64(330), generateTxHash(), nil)
	addTestReward(t, ctx, rs, party1, asset2, market1, 1, "RewardMakerPaidFees", now, block, 7, num.DecimalFromInt64(400), generateTxHash(), nil)
	addTestReward(t, ctx, rs, party2, asset2, market1, 1, "RewardMakerPaidFees", now, block, 8, num.DecimalFromInt64(500), generateTxHash(), nil)
	addTestReward(t, ctx, rs, party3, asset2, market1, 1, "RewardMakerPaidFees", now, block, 9, num.DecimalFromInt64(600), generateTxHash(), nil)
	addTestReward(t, ctx, rs, party1, asset2, market2, 1, "RewardMakerPaidFees", now, block, 10, num.DecimalFromInt64(410), generateTxHash(), nil)
	addTestReward(t, ctx, rs, party2, asset2, market2, 1, "RewardMakerPaidFees", now, block, 11, num.DecimalFromInt64(520), generateTxHash(), nil)
	addTestReward(t, ctx, rs, party3, asset2, market2, 1, "RewardMakerPaidFees", now, block, 12, num.DecimalFromInt64(630), generateTxHash(), nil)
	addTestReward(t, ctx, rs, party1, asset1, market1, 1, "RewardMakerReceivedFees", now, block, 13, num.DecimalFromInt64(1000), generateTxHash(), nil)
	addTestReward(t, ctx, rs, party2, asset1, market1, 1, "RewardMakerReceivedFees", now, block, 14, num.DecimalFromInt64(2000), generateTxHash(), nil)
	addTestReward(t, ctx, rs, party3, asset1, market1, 1, "RewardMakerReceivedFees", now, block, 15, num.DecimalFromInt64(3000), generateTxHash(), nil)
	addTestReward(t, ctx, rs, party1, asset1, market2, 1, "GlobalReward", now, block, 16, num.DecimalFromInt64(1100), generateTxHash(), nil)
	addTestReward(t, ctx, rs, party2, asset1, market2, 1, "GlobalReward", now, block, 17, num.DecimalFromInt64(2200), generateTxHash(), nil)
	addTestReward(t, ctx, rs, party3, asset1, market2, 1, "GlobalReward", now, block, 18, num.DecimalFromInt64(3300), generateTxHash(), nil)
	addTestReward(t, ctx, rs, party1, asset2, market1, 1, "RewardMakerReceivedFees", now, block, 19, num.DecimalFromInt64(4000), generateTxHash(), nil)
	addTestReward(t, ctx, rs, party2, asset2, market1, 1, "RewardMakerReceivedFees", now, block, 20, num.DecimalFromInt64(5000), generateTxHash(), nil)
	addTestReward(t, ctx, rs, party3, asset2, market1, 1, "RewardMakerReceivedFees", now, block, 21, num.DecimalFromInt64(6000), generateTxHash(), nil)
	addTestReward(t, ctx, rs, party1, asset2, market2, 1, "GlobalReward", now, block, 22, num.DecimalFromInt64(4100), generateTxHash(), nil)
	addTestReward(t, ctx, rs, party2, asset2, market2, 1, "GlobalReward", now, block, 23, num.DecimalFromInt64(5200), generateTxHash(), nil)
	addTestReward(t, ctx, rs, party3, asset2, market2, 1, "GlobalReward", now, block, 24, num.DecimalFromInt64(6300), generateTxHash(), nil)

	// rewards for epoch2
	addTestReward(t, ctx, rs, party1, asset1, market1, 2, "RewardMakerPaidFees", now, block, 25, num.DecimalFromInt64(10000), generateTxHash(), nil)
	addTestReward(t, ctx, rs, party2, asset1, market1, 2, "RewardMakerPaidFees", now, block, 26, num.DecimalFromInt64(20000), generateTxHash(), nil)
	addTestReward(t, ctx, rs, party3, asset1, market1, 2, "RewardMakerPaidFees", now, block, 27, num.DecimalFromInt64(30000), generateTxHash(), nil)
	addTestReward(t, ctx, rs, party1, asset1, market2, 2, "RewardMakerPaidFees", now, block, 28, num.DecimalFromInt64(11000), generateTxHash(), nil)
	addTestReward(t, ctx, rs, party2, asset1, market2, 2, "RewardMakerPaidFees", now, block, 29, num.DecimalFromInt64(22000), generateTxHash(), nil)
	addTestReward(t, ctx, rs, party3, asset1, market2, 2, "RewardMakerPaidFees", now, block, 30, num.DecimalFromInt64(33000), generateTxHash(), nil)
	addTestReward(t, ctx, rs, party1, asset2, market1, 2, "RewardMakerPaidFees", now, block, 31, num.DecimalFromInt64(40000), generateTxHash(), nil)
	addTestReward(t, ctx, rs, party2, asset2, market1, 2, "RewardMakerPaidFees", now, block, 32, num.DecimalFromInt64(50000), generateTxHash(), nil)
	addTestReward(t, ctx, rs, party3, asset2, market1, 2, "RewardMakerPaidFees", now, block, 33, num.DecimalFromInt64(60000), generateTxHash(), nil)
	addTestReward(t, ctx, rs, party1, asset2, market2, 2, "RewardMakerPaidFees", now, block, 34, num.DecimalFromInt64(41000), generateTxHash(), nil)
	addTestReward(t, ctx, rs, party2, asset2, market2, 2, "RewardMakerPaidFees", now, block, 35, num.DecimalFromInt64(52000), generateTxHash(), nil)
	addTestReward(t, ctx, rs, party3, asset2, market2, 2, "RewardMakerPaidFees", now, block, 36, num.DecimalFromInt64(63000), generateTxHash(), nil)
	addTestReward(t, ctx, rs, party1, asset1, market1, 2, "RewardMakerReceivedFees", now, block, 37, num.DecimalFromInt64(100000), generateTxHash(), nil)
	addTestReward(t, ctx, rs, party2, asset1, market1, 2, "RewardMakerReceivedFees", now, block, 38, num.DecimalFromInt64(200000), generateTxHash(), nil)
	addTestReward(t, ctx, rs, party3, asset1, market1, 2, "RewardMakerReceivedFees", now, block, 39, num.DecimalFromInt64(300000), generateTxHash(), nil)
	addTestReward(t, ctx, rs, party1, asset1, market2, 2, "GlobalReward", now, block, 40, num.DecimalFromInt64(110000), generateTxHash(), nil)
	addTestReward(t, ctx, rs, party2, asset1, market2, 2, "GlobalReward", now, block, 41, num.DecimalFromInt64(220000), generateTxHash(), nil)
	addTestReward(t, ctx, rs, party3, asset1, market2, 2, "GlobalReward", now, block, 42, num.DecimalFromInt64(330000), generateTxHash(), nil)
	addTestReward(t, ctx, rs, party1, asset2, market1, 2, "RewardMakerReceivedFees", now, block, 43, num.DecimalFromInt64(400000), generateTxHash(), nil)
	addTestReward(t, ctx, rs, party2, asset2, market1, 2, "RewardMakerReceivedFees", now, block, 44, num.DecimalFromInt64(500000), generateTxHash(), nil)
	addTestReward(t, ctx, rs, party3, asset2, market1, 2, "RewardMakerReceivedFees", now, block, 45, num.DecimalFromInt64(600000), generateTxHash(), nil)
	addTestReward(t, ctx, rs, party1, asset2, market2, 2, "GlobalReward", now, block, 46, num.DecimalFromInt64(410000), generateTxHash(), nil)
	addTestReward(t, ctx, rs, party2, asset2, market2, 2, "GlobalReward", now, block, 47, num.DecimalFromInt64(520000), generateTxHash(), nil)
	addTestReward(t, ctx, rs, party3, asset2, market2, 2, "GlobalReward", now, block, 48, num.DecimalFromInt64(630000), generateTxHash(), nil)

	first := int32(1000)
	pagination, _ := entities.NewCursorPagination(&first, nil, nil, nil, false)
	filter := entities.RewardSummaryFilter{}
	summaries, _, err := rs.GetEpochSummaries(ctx, filter, pagination)
	require.NoError(t, err)

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
	filter = entities.RewardSummaryFilter{FromEpoch: &from}
	summaries, _, _ = rs.GetEpochSummaries(ctx, filter, pagination)
	require.Equal(t, 16, len(summaries))
	verifyRewardsForEpoch(t, summaries, 1, asset1.ID.String(), asset2.ID.String())
	verifyRewardsForEpoch(t, summaries, 2, asset1.ID.String(), asset2.ID.String())

	// now request with from = nil and to = 2, expect the same result
	to := uint64(2)
	filter = entities.RewardSummaryFilter{ToEpoch: &to}
	summaries, _, _ = rs.GetEpochSummaries(ctx, filter, pagination)
	require.Equal(t, 16, len(summaries))
	verifyRewardsForEpoch(t, summaries, 1, asset1.ID.String(), asset2.ID.String())
	verifyRewardsForEpoch(t, summaries, 2, asset1.ID.String(), asset2.ID.String())

	// now request from = 2 to = nil expect only epoch 2
	from = 2
	filter = entities.RewardSummaryFilter{FromEpoch: &from}
	summaries, _, _ = rs.GetEpochSummaries(ctx, filter, pagination)
	require.Equal(t, 8, len(summaries))
	verifyRewardsForEpoch(t, summaries, 2, asset1.ID.String(), asset2.ID.String())

	// now request to = 1 from = nil expect only epoch 1
	to = 1
	filter = entities.RewardSummaryFilter{ToEpoch: &to}
	summaries, _, _ = rs.GetEpochSummaries(ctx, filter, pagination)
	require.Equal(t, 8, len(summaries))
	verifyRewardsForEpoch(t, summaries, 1, asset1.ID.String(), asset2.ID.String())

	// now request from = 1 and to = 1
	from = 1
	filter = entities.RewardSummaryFilter{FromEpoch: &from, ToEpoch: &to}
	summaries, _, _ = rs.GetEpochSummaries(ctx, filter, pagination)
	require.Equal(t, 8, len(summaries))
	verifyRewardsForEpoch(t, summaries, 1, asset1.ID.String(), asset2.ID.String())

	// full filter
	to = 2
	filter = entities.RewardSummaryFilter{
		FromEpoch: &from,
		ToEpoch:   &to,
		AssetIDs:  []entities.AssetID{asset1.ID, asset2.ID},
		MarketIDs: []entities.MarketID{market1, market2},
	}
	summaries, _, err = rs.GetEpochSummaries(ctx, filter, pagination)
	require.NoError(t, err)
	require.Equal(t, 16, len(summaries))
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

func setupRewardsTest(t *testing.T, ctx context.Context) (*sqlstore.Blocks, *sqlstore.Rewards, *sqlstore.Parties, *sqlstore.Assets) {
	t.Helper()
	bs := sqlstore.NewBlocks(connectionSource)
	rs := sqlstore.NewRewards(ctx, connectionSource)
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
	ctx := tempTransaction(t)
	bs, rs, ps, as := setupRewardsTest(t, ctx)

	populateTestRewards(ctx, t, bs, ps, as, rs)
	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, false)
	require.NoError(t, err)
	partyID := "89c701d1ae2819263e45538d0b25022988bc2508a02c654462d22e0afb626a7d"
	assetID := "8aa92225c32adb54e527fcb1aee2930cbadb4df6f068ab2c2d667eb057ef00fa"

	got, pageInfo, err := rs.GetByCursor(ctx, &partyID, &assetID, nil, nil, pagination, nil, nil)
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
	ctx := tempTransaction(t)
	bs, rs, ps, as := setupRewardsTest(t, ctx)

	populateTestRewards(ctx, t, bs, ps, as, rs)
	first := int32(3)
	pagination, err := entities.NewCursorPagination(&first, nil, nil, nil, false)
	require.NoError(t, err)
	partyID := "89c701d1ae2819263e45538d0b25022988bc2508a02c654462d22e0afb626a7d"
	assetID := "8aa92225c32adb54e527fcb1aee2930cbadb4df6f068ab2c2d667eb057ef00fa"

	got, pageInfo, err := rs.GetByCursor(ctx, &partyID, &assetID, nil, nil, pagination, nil, nil)
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
	ctx := tempTransaction(t)
	bs, rs, ps, as := setupRewardsTest(t, ctx)

	populateTestRewards(ctx, t, bs, ps, as, rs)
	last := int32(3)
	pagination, err := entities.NewCursorPagination(nil, nil, &last, nil, false)
	require.NoError(t, err)
	partyID := "89c701d1ae2819263e45538d0b25022988bc2508a02c654462d22e0afb626a7d"
	assetID := "8aa92225c32adb54e527fcb1aee2930cbadb4df6f068ab2c2d667eb057ef00fa"

	got, pageInfo, err := rs.GetByCursor(ctx, &partyID, &assetID, nil, nil, pagination, nil, nil)
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
	ctx := tempTransaction(t)
	bs, rs, ps, as := setupRewardsTest(t, ctx)

	populateTestRewards(ctx, t, bs, ps, as, rs)
	partyID := "89c701d1ae2819263e45538d0b25022988bc2508a02c654462d22e0afb626a7d"
	assetID := "8aa92225c32adb54e527fcb1aee2930cbadb4df6f068ab2c2d667eb057ef00fa"

	first := int32(3)
	after := entities.NewCursor(entities.RewardCursor{PartyID: partyID, AssetID: assetID, EpochID: 643}.String()).Encode()
	pagination, err := entities.NewCursorPagination(&first, &after, nil, nil, false)
	require.NoError(t, err)

	got, pageInfo, err := rs.GetByCursor(ctx, &partyID, &assetID, nil, nil, pagination, nil, nil)
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
	ctx := tempTransaction(t)
	bs, rs, ps, as := setupRewardsTest(t, ctx)

	populateTestRewards(ctx, t, bs, ps, as, rs)
	partyID := "89c701d1ae2819263e45538d0b25022988bc2508a02c654462d22e0afb626a7d"
	assetID := "8aa92225c32adb54e527fcb1aee2930cbadb4df6f068ab2c2d667eb057ef00fa"

	last := int32(3)
	before := entities.NewCursor(entities.RewardCursor{PartyID: partyID, AssetID: assetID, EpochID: 757}.String()).Encode()
	pagination, err := entities.NewCursorPagination(nil, nil, &last, &before, false)
	require.NoError(t, err)
	got, pageInfo, err := rs.GetByCursor(ctx, &partyID, &assetID, nil, nil, pagination, nil, nil)
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
	ctx := tempTransaction(t)
	bs, rs, ps, as := setupRewardsTest(t, ctx)

	populateTestRewards(ctx, t, bs, ps, as, rs)
	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, true)
	require.NoError(t, err)
	partyID := "89c701d1ae2819263e45538d0b25022988bc2508a02c654462d22e0afb626a7d"
	assetID := "8aa92225c32adb54e527fcb1aee2930cbadb4df6f068ab2c2d667eb057ef00fa"

	got, pageInfo, err := rs.GetByCursor(ctx, &partyID, &assetID, nil, nil, pagination, nil, nil)
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
	ctx := tempTransaction(t)
	bs, rs, ps, as := setupRewardsTest(t, ctx)

	populateTestRewards(ctx, t, bs, ps, as, rs)
	first := int32(3)
	pagination, err := entities.NewCursorPagination(&first, nil, nil, nil, true)
	require.NoError(t, err)
	partyID := "89c701d1ae2819263e45538d0b25022988bc2508a02c654462d22e0afb626a7d"
	assetID := "8aa92225c32adb54e527fcb1aee2930cbadb4df6f068ab2c2d667eb057ef00fa"

	got, pageInfo, err := rs.GetByCursor(ctx, &partyID, &assetID, nil, nil, pagination, nil, nil)
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
	ctx := tempTransaction(t)
	bs, rs, ps, as := setupRewardsTest(t, ctx)

	populateTestRewards(ctx, t, bs, ps, as, rs)
	last := int32(3)
	pagination, err := entities.NewCursorPagination(nil, nil, &last, nil, true)
	require.NoError(t, err)
	partyID := "89c701d1ae2819263e45538d0b25022988bc2508a02c654462d22e0afb626a7d"
	assetID := "8aa92225c32adb54e527fcb1aee2930cbadb4df6f068ab2c2d667eb057ef00fa"

	got, pageInfo, err := rs.GetByCursor(ctx, &partyID, &assetID, nil, nil, pagination, nil, nil)
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
	ctx := tempTransaction(t)
	bs, rs, ps, as := setupRewardsTest(t, ctx)

	populateTestRewards(ctx, t, bs, ps, as, rs)
	partyID := "89c701d1ae2819263e45538d0b25022988bc2508a02c654462d22e0afb626a7d"
	assetID := "8aa92225c32adb54e527fcb1aee2930cbadb4df6f068ab2c2d667eb057ef00fa"

	first := int32(3)
	after := entities.NewCursor(entities.RewardCursor{PartyID: partyID, AssetID: assetID, EpochID: 757}.String()).Encode()
	pagination, err := entities.NewCursorPagination(&first, &after, nil, nil, true)
	require.NoError(t, err)

	got, pageInfo, err := rs.GetByCursor(ctx, &partyID, &assetID, nil, nil, pagination, nil, nil)
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
	ctx := tempTransaction(t)
	bs, rs, ps, as := setupRewardsTest(t, ctx)

	populateTestRewards(ctx, t, bs, ps, as, rs)
	partyID := "89c701d1ae2819263e45538d0b25022988bc2508a02c654462d22e0afb626a7d"
	assetID := "8aa92225c32adb54e527fcb1aee2930cbadb4df6f068ab2c2d667eb057ef00fa"

	last := int32(3)
	before := entities.NewCursor(entities.RewardCursor{PartyID: partyID, AssetID: assetID, EpochID: 643}.String()).Encode()
	pagination, err := entities.NewCursorPagination(nil, nil, &last, &before, true)
	require.NoError(t, err)
	got, pageInfo, err := rs.GetByCursor(ctx, &partyID, &assetID, nil, nil, pagination, nil, nil)
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

func Test_FilterRewardsQuery(t *testing.T) {
	type args struct {
		table    string
		inFilter entities.RewardSummaryFilter
	}
	tests := []struct {
		name      string
		args      args
		wantQuery string
		wantArgs  []any
		wantErr   assert.ErrorAssertionFunc
	}{
		{
			name: "filter with all values",
			args: args{
				table: "test",
				inFilter: entities.RewardSummaryFilter{
					AssetIDs:  []entities.AssetID{"8aa92225c32adb54e527fcb1aee2930cbadb4df6f068ab2c2d667eb057ef00fa"},
					MarketIDs: []entities.MarketID{"deadbeef"},
					FromEpoch: ptr.From(uint64(123)),
					ToEpoch:   ptr.From(uint64(124)),
				},
			},
			wantQuery: ` WHERE asset_id = ANY($1) AND market_id = ANY($2) AND epoch_id >= $3 AND epoch_id <= $4`,
			wantArgs: []any{
				"8aa92225c32adb54e527fcb1aee2930cbadb4df6f068ab2c2d667eb057ef00fa",
				"deadbeef",
				ptr.From(uint64(123)),
				ptr.From(uint64(124)),
			},
			wantErr: assert.NoError,
		}, {
			name: "filter with no values",
			args: args{
				table:    "test",
				inFilter: entities.RewardSummaryFilter{},
			},
			wantQuery: "",
			wantArgs:  []any{},
			wantErr:   assert.NoError,
		}, {
			name: "filter with only asset ids",
			args: args{
				table: "test",
				inFilter: entities.RewardSummaryFilter{
					AssetIDs: []entities.AssetID{"8aa92225c32adb54e527fcb1aee2930cbadb4df6f068ab2c2d667eb057ef00fa"},
				},
			},
			wantQuery: ` WHERE asset_id = ANY($1)`,
			wantArgs: []any{
				"8aa92225c32adb54e527fcb1aee2930cbadb4df6f068ab2c2d667eb057ef00fa",
			},
			wantErr: assert.NoError,
		}, {
			name: "filter with only market ids",
			args: args{
				table: "test",
				inFilter: entities.RewardSummaryFilter{
					MarketIDs: []entities.MarketID{"deadbeef"},
				},
			},
			wantQuery: ` WHERE market_id = ANY($1)`,
			wantArgs: []any{
				"deadbeef",
			},
			wantErr: assert.NoError,
		}, {
			name: "filter with only from epoch",
			args: args{
				table: "test",
				inFilter: entities.RewardSummaryFilter{
					FromEpoch: ptr.From(uint64(123)),
				},
			},
			wantQuery: ` WHERE epoch_id >= $1`,
			wantArgs: []any{
				ptr.From(uint64(123)),
			},
			wantErr: assert.NoError,
		}, {
			name: "filter with only to epoch",
			args: args{
				table: "test",
				inFilter: entities.RewardSummaryFilter{
					ToEpoch: ptr.From(uint64(123)),
				},
			},
			wantQuery: ` WHERE epoch_id <= $1`,
			wantArgs: []any{
				ptr.From(uint64(123)),
			},
			wantErr: assert.NoError,
		}, {
			name: "filter with only from and to epoch",
			args: args{
				table: "test",
				inFilter: entities.RewardSummaryFilter{
					FromEpoch: ptr.From(uint64(123)),
					ToEpoch:   ptr.From(uint64(124)),
				},
			},
			wantQuery: ` WHERE epoch_id >= $1 AND epoch_id <= $2`,
			wantArgs: []any{
				ptr.From(uint64(123)),
				ptr.From(uint64(124)),
			},
			wantErr: assert.NoError,
		}, {
			name: "filter with only asset ids and from epoch",
			args: args{
				table: "test",
				inFilter: entities.RewardSummaryFilter{
					AssetIDs:  []entities.AssetID{"8aa92225c32adb54e527fcb1aee2930cbadb4df6f068ab2c2d667eb057ef00fa"},
					FromEpoch: ptr.From(uint64(123)),
				},
			},
			wantQuery: ` WHERE asset_id = ANY($1) AND epoch_id >= $2`,
			wantArgs: []any{
				"8aa92225c32adb54e527fcb1aee2930cbadb4df6f068ab2c2d667eb057ef00fa",
				ptr.From(uint64(123)),
			},
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, args, err := sqlstore.FilterRewardsQuery(tt.args.inFilter)
			if !tt.wantErr(t, err, fmt.Sprintf("filterSQL(%v, %v)", tt.args.table, tt.args.inFilter)) {
				return
			}
			assert.Equalf(t, tt.wantQuery, got, "filterSQL(%v, %v)", tt.args.table, tt.args.inFilter)
			for i, arg := range args {
				if reflect.TypeOf(arg).Kind() == reflect.Slice {
					arg = hex.EncodeToString(arg.([][]uint8)[0])
				}
				assert.Equalf(t, tt.wantArgs[i], arg, "filterSQL(%v, %v)", tt.args.table, tt.args.inFilter)
			}
		})
	}
}

func TestRewardsGameTotals(t *testing.T) {
	ctx := tempTransaction(t)
	// teams
	teams := []entities.Team{
		{
			ID:             "deadd00d01",
			Referrer:       "beefbeef01",
			Name:           "aaaa",
			TeamURL:        nil,
			AvatarURL:      nil,
			Closed:         false,
			CreatedAt:      time.Now(),
			CreatedAtEpoch: 0,
			VegaTime:       time.Now(),
		},
		{
			ID:             "deadd00d02",
			Referrer:       "beefbeef02",
			Name:           "bbbb",
			TeamURL:        nil,
			AvatarURL:      nil,
			Closed:         false,
			CreatedAt:      time.Now(),
			CreatedAtEpoch: 0,
			VegaTime:       time.Now(),
		},
		{
			ID:             "deadd00d03",
			Referrer:       "beefbeef03",
			Name:           "cccc",
			TeamURL:        nil,
			AvatarURL:      nil,
			Closed:         false,
			CreatedAt:      time.Now(),
			CreatedAtEpoch: 0,
			VegaTime:       time.Now(),
		},
	}
	for _, team := range teams {
		_, err := connectionSource.Connection.Exec(ctx,
			`INSERT INTO teams (id, referrer, name, team_url, avatar_url, closed, created_at_epoch, created_at, vega_time)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
			team.ID, team.Referrer, team.Name, team.TeamURL, team.AvatarURL, team.Closed, team.CreatedAtEpoch, team.CreatedAt, team.VegaTime)
		require.NoError(t, err)
	}
	// team members data
	teamMembers := []entities.TeamMember{
		{
			TeamID:        "deadd00d01",
			PartyID:       "deadbeef01",
			JoinedAt:      time.Now(),
			JoinedAtEpoch: 0,
			VegaTime:      time.Now(),
		},
		{
			TeamID:        "deadd00d02",
			PartyID:       "deadbeef02",
			JoinedAt:      time.Now(),
			JoinedAtEpoch: 0,
			VegaTime:      time.Now(),
		},
		{
			TeamID:        "deadd00d03",
			PartyID:       "deadbeef03",
			JoinedAt:      time.Now(),
			JoinedAtEpoch: 0,
			VegaTime:      time.Now(),
		},
	}
	for _, member := range teamMembers {
		_, err := connectionSource.Connection.Exec(ctx,
			`INSERT INTO team_members (team_id, party_id, joined_at_epoch, joined_at, vega_time)
		VALUES ($1, $2, $3, $4, $5)`,
			member.TeamID, member.PartyID, member.JoinedAtEpoch, member.JoinedAt, member.VegaTime)
		require.NoError(t, err)
	}
	// populate the game reward totals with some test data
	existingTotals := []entities.RewardTotals{
		{
			GameID:              "deadbeef01",
			PartyID:             "cafedaad01",
			AssetID:             "deadbaad01",
			MarketID:            "beefcafe01",
			EpochID:             1,
			TeamID:              "deadd00d01",
			TotalRewards:        decimal.NewFromFloat(1000),
			TotalRewardsQuantum: decimal.NewFromFloat(1000),
		},
		{
			GameID:              "deadbeef02",
			PartyID:             "cafedaad02",
			AssetID:             "deadbaad02",
			MarketID:            "beefcafe02",
			EpochID:             1,
			TeamID:              "deadd00d02",
			TotalRewards:        decimal.NewFromFloat(2000),
			TotalRewardsQuantum: decimal.NewFromFloat(2000),
		},
		{
			GameID:              "deadbeef03",
			PartyID:             "cafedaad03",
			AssetID:             "deadbaad03",
			MarketID:            "beefcafe03",
			EpochID:             1,
			TeamID:              "deadd00d03",
			TotalRewards:        decimal.NewFromFloat(3000),
			TotalRewardsQuantum: decimal.NewFromFloat(3000),
		},
	}
	for _, total := range existingTotals {
		_, err := connectionSource.Connection.Exec(ctx,
			`INSERT INTO game_reward_totals (game_id, party_id, asset_id, market_id, epoch_id, team_id, total_rewards, total_rewards_quantum)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
			total.GameID, total.PartyID, total.AssetID, total.MarketID, total.EpochID, total.TeamID, total.TotalRewards, total.TotalRewardsQuantum)
		require.NoError(t, err)
	}

	ts := time.Now()
	ts2 := ts.Add(time.Minute)
	// add rewards
	rewardsToAdd := []entities.Reward{
		{
			PartyID:            "cafedaad01",
			AssetID:            "deadbaad01",
			MarketID:           "beefcafe01",
			EpochID:            2,
			Amount:             decimal.NewFromFloat(1000),
			QuantumAmount:      decimal.NewFromFloat(1000),
			PercentOfTotal:     0,
			RewardType:         "ACCOUNT_TYPE_UNSPECIFIED",
			Timestamp:          ts,
			TxHash:             "",
			VegaTime:           ts,
			SeqNum:             1,
			LockedUntilEpochID: 30,
			GameID:             ptr.From(entities.GameID("deadbeef01")),
		},
		{
			PartyID:            "cafedaad02",
			AssetID:            "deadbaad02",
			MarketID:           "beefcafe02",
			EpochID:            2,
			Amount:             decimal.NewFromFloat(1000),
			QuantumAmount:      decimal.NewFromFloat(1000),
			PercentOfTotal:     0,
			RewardType:         "ACCOUNT_TYPE_UNSPECIFIED",
			Timestamp:          ts,
			TxHash:             "",
			VegaTime:           ts,
			SeqNum:             2,
			LockedUntilEpochID: 30,
			GameID:             ptr.From(entities.GameID("deadbeef02")),
		},
		{
			PartyID:            "cafedaad03",
			AssetID:            "deadbaad03",
			MarketID:           "beefcafe03",
			EpochID:            2,
			Amount:             decimal.NewFromFloat(1000),
			QuantumAmount:      decimal.NewFromFloat(1000),
			PercentOfTotal:     0,
			RewardType:         "ACCOUNT_TYPE_UNSPECIFIED",
			Timestamp:          ts,
			TxHash:             "",
			VegaTime:           ts,
			SeqNum:             3,
			LockedUntilEpochID: 30,
			GameID:             ptr.From(entities.GameID("deadbeef03")),
		},
		{
			PartyID:            "cafedaad01",
			AssetID:            "deadbaad01",
			MarketID:           "beefcafe01",
			EpochID:            3,
			Amount:             decimal.NewFromFloat(1000),
			QuantumAmount:      decimal.NewFromFloat(1000),
			PercentOfTotal:     0,
			RewardType:         "ACCOUNT_TYPE_UNSPECIFIED",
			Timestamp:          ts2,
			TxHash:             "",
			VegaTime:           ts2,
			SeqNum:             1,
			LockedUntilEpochID: 30,
			GameID:             ptr.From(entities.GameID("deadbeef01")),
		},
		{
			PartyID:            "cafedaad02",
			AssetID:            "deadbaad02",
			MarketID:           "beefcafe02",
			EpochID:            3,
			Amount:             decimal.NewFromFloat(1000),
			QuantumAmount:      decimal.NewFromFloat(1000),
			PercentOfTotal:     0,
			RewardType:         "ACCOUNT_TYPE_UNSPECIFIED",
			Timestamp:          ts2,
			TxHash:             "",
			VegaTime:           ts2,
			SeqNum:             2,
			LockedUntilEpochID: 30,
			GameID:             ptr.From(entities.GameID("deadbeef02")),
		},
		{
			PartyID:            "cafedaad03",
			AssetID:            "deadbaad03",
			MarketID:           "beefcafe03",
			EpochID:            3,
			Amount:             decimal.NewFromFloat(1000),
			QuantumAmount:      decimal.NewFromFloat(1000),
			PercentOfTotal:     0,
			RewardType:         "ACCOUNT_TYPE_UNSPECIFIED",
			Timestamp:          ts2,
			TxHash:             "",
			VegaTime:           ts2,
			SeqNum:             3,
			LockedUntilEpochID: 30,
			GameID:             ptr.From(entities.GameID("deadbeef03")),
		},
	}

	rs := sqlstore.NewRewards(ctx, connectionSource)
	for _, r := range rewardsToAdd {
		require.NoError(t, rs.Add(ctx, r))
	}

	// Now make sure the totals are updated and correct
	testCases := []struct {
		game_id  entities.GameID
		party_id entities.PartyID
		epoch_id int64
		want     decimal.Decimal
	}{
		{
			game_id:  "deadbeef01",
			party_id: "cafedaad01",
			epoch_id: 2,
			want:     decimal.NewFromFloat(2000),
		},
		{
			game_id:  "deadbeef01",
			party_id: "cafedaad01",
			epoch_id: 3,
			want:     decimal.NewFromFloat(3000),
		},
		{
			game_id:  "deadbeef02",
			party_id: "cafedaad02",
			epoch_id: 2,
			want:     decimal.NewFromFloat(3000),
		},
		{
			game_id:  "deadbeef02",
			party_id: "cafedaad02",
			epoch_id: 3,
			want:     decimal.NewFromFloat(4000),
		},
	}
	for _, tc := range testCases {
		var totals []entities.RewardTotals
		require.NoError(t, pgxscan.Select(ctx, connectionSource.Connection, &totals,
			`SELECT * FROM game_reward_totals WHERE game_id = $1 AND party_id = $2 AND epoch_id = $3`,
			tc.game_id, tc.party_id, tc.epoch_id))
		assert.Equal(t, 1, len(totals))
		assert.True(t, tc.want.Equal(totals[0].TotalRewards), "totals don't match, got: %s, want: %s", totals[0].TotalRewards, tc.want)
		assert.True(t, tc.want.Equal(totals[0].TotalRewardsQuantum), "totals don't match, got: %s, want: %s", totals[0].TotalRewardsQuantum, tc.want)
	}
}

func filterRewardsByParty(rewards []entities.Reward, partyID entities.PartyID) []entities.Reward {
	filtered := make([]entities.Reward, 0)
	for _, r := range rewards {
		if r.PartyID == partyID {
			filtered = append(filtered, r)
		}
	}

	return filtered
}

func filterRewardsByTeam(rewards []entities.Reward, teamID entities.TeamID) []entities.Reward {
	filtered := make([]entities.Reward, 0)
	for _, r := range rewards {
		if r.TeamID != nil && *r.TeamID == teamID {
			filtered = append(filtered, r)
		}
	}

	return filtered
}

func filterRewardsByGame(rewards []entities.Reward, gameID entities.GameID) []entities.Reward {
	filtered := make([]entities.Reward, 0)
	for _, r := range rewards {
		if *r.GameID == gameID {
			filtered = append(filtered, r)
		}
	}

	return filtered
}

func TestRewardFilterByTeamIDAndGameID(t *testing.T) {
	// going to use the games setup because we need to make sure the rewards have game and associated team data too.
	ctx := tempTransaction(t)
	stores := setupGamesTest(t, ctx)
	startingBlock := addTestBlockForTime(t, ctx, stores.blocks, time.Now())
	_, gameIDs, gameRewards, teams, _ := setupGamesData(ctx, t, stores, startingBlock, 50)
	rewards := make([]entities.Reward, 0)
	src := rand.NewSource(time.Now().UnixNano())
	r := rand.New(src)
	// setup, get the unique individuals and from all the rewards
	individuals := make(map[entities.PartyID]struct{})
	for _, gr := range gameRewards {
		for _, r := range gr {
			rewards = append(rewards, r)
			individuals[r.PartyID] = struct{}{}
		}
	}

	t.Run("Should return rewards for all teams the party has been a member of if no team ID is provided", func(t *testing.T) {
		// create a list of parties so we can pick one at random
		parties := make([]entities.PartyID, 0)
		for p := range individuals {
			parties = append(parties, p)
		}
		// pick a random party
		i := r.Intn(len(parties))
		partyID := parties[i]
		page := entities.DefaultCursorPagination(true)
		// get the rewards for that party
		got, _, err := stores.rewards.GetByCursor(ctx, ptr.From(partyID.String()), nil, nil, nil, page, nil, nil)
		require.NoError(t, err)
		want := filterRewardsByParty(rewards, partyID)
		// we don't care about the ordering as other tests already validate that, we just want to make sure we have all the rewards for the party
		assert.ElementsMatchf(t, want, got, "got: %v, want: %v", got, want)
	})

	t.Run("Should return rewards for the specified team if a team ID is provided", func(t *testing.T) {
		allTeams := make(map[string]struct{})
		for team := range teams {
			allTeams[team] = struct{}{}
		}
		teamIDs := make([]string, 0)
		for team := range allTeams {
			teamIDs = append(teamIDs, team)
		}
		i := r.Intn(len(teamIDs))
		teamID := teamIDs[i]
		i = r.Intn(len(teams[teamID]))
		party := teams[teamID][i]
		page := entities.DefaultCursorPagination(true)
		got, _, err := stores.rewards.GetByCursor(ctx, ptr.From(party.ID.String()), nil, nil, nil, page, ptr.From(teamID), nil)
		require.NoError(t, err)
		want := filterRewardsByParty(filterRewardsByTeam(rewards, entities.TeamID(teamID)), party.ID)
		assert.ElementsMatchf(t, want, got, "got: %v, want: %v", got, want)
	})

	t.Run("Should return rewards for the specified game if a game ID is provided", func(t *testing.T) {
		i := r.Intn(len(gameIDs))
		gameID := gameIDs[i]
		page := entities.DefaultCursorPagination(true)
		got, _, err := stores.rewards.GetByCursor(ctx, nil, nil, nil, nil, page, nil, ptr.From(gameID))
		require.NoError(t, err)
		want := filterRewardsByGame(rewards, entities.GameID(gameID))
		assert.ElementsMatchf(t, want, got, "got: %v, want: %v", got, want)
	})
}
