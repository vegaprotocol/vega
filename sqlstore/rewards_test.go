package sqlstore_test

import (
	"context"
	"testing"

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
	epochID int64,
	block entities.Block,
) entities.Reward {
	r := entities.Reward{
		PartyID:        party.ID,
		AssetID:        asset.ID,
		EpochID:        epochID,
		Amount:         decimal.NewFromInt(100),
		PercentOfTotal: 0.2,
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
	defer testStore.DeleteEverything()
	ps := sqlstore.NewParties(testStore)
	as := sqlstore.NewAssets(testStore)
	rs := sqlstore.NewRewards(testStore)
	bs := sqlstore.NewBlocks(testStore)
	block := addTestBlock(t, bs)

	asset1 := addTestAsset(t, as, block)
	asset2 := addTestAsset(t, as, block)
	party1 := addTestParty(t, ps, block)
	party2 := addTestParty(t, ps, block)

	party1ID := party1.ID.String()
	asset1ID := asset1.ID.String()
	party2ID := party2.ID.String()
	asset2ID := asset2.ID.String()

	reward1 := addTestReward(t, rs, party1, asset1, 1, block)
	reward2 := addTestReward(t, rs, party1, asset2, 2, block)
	reward3 := addTestReward(t, rs, party2, asset1, 3, block)
	reward4 := addTestReward(t, rs, party2, asset2, 4, block)
	reward5 := addTestReward(t, rs, party2, asset2, 5, block)

	t.Run("GetAll", func(t *testing.T) {
		expected := []entities.Reward{reward1, reward2, reward3, reward4, reward5}
		actual, err := rs.GetAll(context.Background())
		require.NoError(t, err)
		assertRewardsMatch(t, expected, actual)
	})

	t.Run("GetByParty", func(t *testing.T) {
		expected := []entities.Reward{reward1, reward2}
		actual, err := rs.Get(context.Background(), &party1ID, nil, nil)
		require.NoError(t, err)
		assertRewardsMatch(t, expected, actual)
	})

	t.Run("GetByAsset", func(t *testing.T) {
		expected := []entities.Reward{reward1, reward3}
		actual, err := rs.Get(context.Background(), nil, &asset1ID, nil)
		require.NoError(t, err)
		assertRewardsMatch(t, expected, actual)
	})

	t.Run("GetByAssetAndParty", func(t *testing.T) {
		expected := []entities.Reward{reward1}
		actual, err := rs.Get(context.Background(), &party1ID, &asset1ID, nil)
		require.NoError(t, err)
		assertRewardsMatch(t, expected, actual)
	})

	t.Run("GetPagination", func(t *testing.T) {
		expected := []entities.Reward{reward4, reward3, reward2}
		p := entities.Pagination{Skip: 1, Limit: 3, Descending: true}
		actual, err := rs.Get(context.Background(), nil, nil, &p)
		require.NoError(t, err)
		assert.Equal(t, expected, actual) // Explicitly check the order on this one
	})

	t.Run("GetSummary", func(t *testing.T) {
		expected := []entities.RewardSummary{{
			AssetID: asset2.ID,
			PartyID: party2.ID,
			Amount:  decimal.NewFromInt(200)}}
		actual, err := rs.GetSummaries(context.Background(), &party2ID, &asset2ID)
		require.NoError(t, err)
		assert.Equal(t, expected, actual)
	})
}
