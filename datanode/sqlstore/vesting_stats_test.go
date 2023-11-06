package sqlstore_test

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/sqlstore"
	"code.vegaprotocol.io/vega/libs/num"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupVestingStatsTest(t *testing.T) (*sqlstore.Blocks, *sqlstore.VestingStats) {
	t.Helper()
	bs := sqlstore.NewBlocks(connectionSource)
	plbs := sqlstore.NewVestingStats(connectionSource)

	return bs, plbs
}

func TestVestingStats(t *testing.T) {
	_, vs := setupVestingStatsTest(t)

	const (
		party1 = "70432aa1dc6bc20a9b404d30f23e6a8def11a1692609dcef0ad8dc558d9df7db"
		party2 = "a696300fec90755c90e2489af68fe2dfede5744184711ea3acde0ca55ae19585"
	)

	t.Run("return error if do not exists", func(t *testing.T) {
		_, err := vs.GetByPartyID(context.Background(), party1)
		require.EqualError(t, err, "no resource corresponding to this id")
		_, err = vs.GetByPartyID(context.Background(), party2)
		require.EqualError(t, err, "no resource corresponding to this id")
	})

	now := time.Now().Truncate(time.Millisecond)

	t.Run("can insert successfully", func(t *testing.T) {
		w := entities.VestingStatsUpdated{
			AtEpoch:  1,
			VegaTime: now,
			PartyVestingStats: []*entities.PartyVestingStats{
				{
					PartyID:               entities.PartyID(party1),
					RewardBonusMultiplier: num.MustDecimalFromString("0.5"),
					QuantumBalance:        num.MustDecimalFromString("10001"),
					VegaTime:              now,
					AtEpoch:               1,
				},
				{
					PartyID:               entities.PartyID(party2),
					RewardBonusMultiplier: num.MustDecimalFromString("1.5"),
					QuantumBalance:        num.MustDecimalFromString("20001"),
					VegaTime:              now,
					AtEpoch:               1,
				},
			},
		}

		assert.NoError(t, vs.Add(context.Background(), &w))

		pvs1, err := vs.GetByPartyID(context.Background(), party1)
		require.NoError(t, err)
		require.Equal(t, *w.PartyVestingStats[0], pvs1)
		pvs2, err := vs.GetByPartyID(context.Background(), party2)
		require.NoError(t, err)
		require.Equal(t, *w.PartyVestingStats[1], pvs2)
	})

	now = now.Add(24 * time.Hour).Truncate(time.Millisecond)

	t.Run("can replace exisisting values", func(t *testing.T) {
		w := entities.VestingStatsUpdated{
			AtEpoch:  2,
			VegaTime: now,
			PartyVestingStats: []*entities.PartyVestingStats{
				{
					PartyID:               entities.PartyID(party1),
					RewardBonusMultiplier: num.MustDecimalFromString("1"),
					QuantumBalance:        num.MustDecimalFromString("12001"),
					VegaTime:              now,
					AtEpoch:               2,
				},
				{
					PartyID:               entities.PartyID(party2),
					RewardBonusMultiplier: num.MustDecimalFromString("2"),
					QuantumBalance:        num.MustDecimalFromString("30001"),
					VegaTime:              now,
					AtEpoch:               2,
				},
			},
		}

		assert.NoError(t, vs.Add(context.Background(), &w))

		pvs1, err := vs.GetByPartyID(context.Background(), party1)
		require.NoError(t, err)
		require.Equal(t, *w.PartyVestingStats[0], pvs1)
		pvs2, err := vs.GetByPartyID(context.Background(), party2)
		require.NoError(t, err)
		require.Equal(t, *w.PartyVestingStats[1], pvs2)
	})
}
