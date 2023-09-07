package sqlstore_test

import (
	"testing"
	"time"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/sqlstore"
	"code.vegaprotocol.io/vega/libs/ptr"
	"github.com/stretchr/testify/assert"
)

type partyActivityStreakStores struct {
	streaksStore *sqlstore.PartyActivityStreaks
}

func newPartyActivityStreakStores(t *testing.T) *partyActivityStreakStores {
	t.Helper()
	streaks := sqlstore.NewPartyActivityStreaks(connectionSource)

	return &partyActivityStreakStores{
		streaksStore: streaks,
	}
}

func TestPartyActivityStreak(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	stores := newPartyActivityStreakStores(t)

	now := time.Now()
	activityStreaks := []entities.PartyActivityStreak{
		{
			PartyID:                              entities.PartyID("09d82547b823da327af14727d02936db75c33cffe8e09341a9fc729fe53865e0"),
			ActiveFor:                            1,
			InactiveFor:                          0,
			IsActive:                             true,
			RewardDistributionActivityMultiplier: "1",
			RewardVestingActivityMultiplier:      "1",
			Epoch:                                1,
			TradedVolume:                         "1000",
			OpenVolume:                           "500",
			VegaTime:                             now,
			TxHash:                               "09d82547b823da327af14727d02936db75c33cffe8e09341a9fc729fe53865e0",
		},
		{
			PartyID:                              entities.PartyID("46d66ea0a00609615e04aaf6b41e5e9f552650535ed85059444d68bb6456852a"),
			ActiveFor:                            1,
			InactiveFor:                          0,
			IsActive:                             true,
			RewardDistributionActivityMultiplier: "1.5",
			RewardVestingActivityMultiplier:      "1.5",
			Epoch:                                1,
			TradedVolume:                         "10000",
			OpenVolume:                           "5000",
			VegaTime:                             now,
			TxHash:                               "09d82547b823da327af14727d02936db75c33cffe8e09341a9fc729fe53865e1",
		},
		{
			PartyID:                              entities.PartyID("46d66ea0a00609615e04aaf6b41e5e9f552650535ed85059444d68bb6456852a"),
			ActiveFor:                            2,
			InactiveFor:                          0,
			IsActive:                             true,
			RewardDistributionActivityMultiplier: "1",
			RewardVestingActivityMultiplier:      "1",
			Epoch:                                2,
			TradedVolume:                         "1000",
			OpenVolume:                           "500",
			VegaTime:                             now.Add(1 * time.Second),
			TxHash:                               "09d82547b823da327af14727d02936db75c33cffe8e09341a9fc729fe53865e2",
		},
		{
			PartyID:                              entities.PartyID("09d82547b823da327af14727d02936db75c33cffe8e09341a9fc729fe53865e0"),
			ActiveFor:                            2,
			InactiveFor:                          0,
			IsActive:                             true,
			RewardDistributionActivityMultiplier: "1",
			RewardVestingActivityMultiplier:      "1",
			Epoch:                                2,
			TradedVolume:                         "1000",
			OpenVolume:                           "500",
			VegaTime:                             now.Add(1 * time.Second),
			TxHash:                               "09d82547b823da327af14727d02936db75c33cffe8e09341a9fc729fe53865e3",
		},
	}

	for _, as := range activityStreaks {
		assert.NoError(t, stores.streaksStore.Add(ctx, &as))
	}

	// now try to get them, without an epoch
	as, err := stores.streaksStore.Get(ctx, "46d66ea0a00609615e04aaf6b41e5e9f552650535ed85059444d68bb6456852a", nil)
	assert.NoError(t, err)
	// should be the last one for this party
	assert.Equal(t, as.Epoch, uint64(2))
	assert.Equal(t, as.PartyID.String(), "46d66ea0a00609615e04aaf6b41e5e9f552650535ed85059444d68bb6456852a")

	// now try to get them with an epoch
	as, err = stores.streaksStore.Get(ctx, "46d66ea0a00609615e04aaf6b41e5e9f552650535ed85059444d68bb6456852a", ptr.From(uint64(1)))
	assert.NoError(t, err)
	// should be the last one for this party
	assert.Equal(t, as.Epoch, uint64(1))
	assert.Equal(t, as.PartyID.String(), "46d66ea0a00609615e04aaf6b41e5e9f552650535ed85059444d68bb6456852a")

	// now try to get them, without the wrong epoch
	as, err = stores.streaksStore.Get(ctx, "46d66ea0a00609615e04aaf6b41e5e9f552650535ed85059444d68bb6456852a", ptr.From(uint64(4)))
	assert.EqualError(t, err, entities.ErrNotFound.Error())
	assert.Nil(t, as)
}
