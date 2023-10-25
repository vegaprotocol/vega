package sqlstore_test

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/sqlstore"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/ptr"
	"github.com/georgysavva/scany/pgxscan"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupPartyLockedBalanceTest(t *testing.T) (*sqlstore.Blocks, *sqlstore.PartyLockedBalance) {
	t.Helper()
	bs := sqlstore.NewBlocks(connectionSource)
	plbs := sqlstore.NewPartyLockedBalances(connectionSource)

	return bs, plbs
}

func TestPruneLockedBalance(t *testing.T) {
	_, plbs := setupPartyLockedBalanceTest(t)

	ctx := tempTransaction(t)

	const (
		party1 = "bd90685fffad262d60edafbf073c52769b1cf55c3d467a078cda117c3b05b677"
		asset1 = "493eb5ee83ea22e45dfd29ef495b9292089dcf85ca9979069ede7d486d412d8f"
		asset2 = "2ed862cde875ce32022fd7f1708c991744c266c616abe9d7bf3c8d7b61d7dec4"
	)

	now := time.Now().Truncate(time.Millisecond)

	t.Run("insert multiple lock balance, for a party, asset and until epoch", func(t *testing.T) {
		balances := []entities.PartyLockedBalance{
			{
				PartyID:    entities.PartyID(party1),
				AssetID:    entities.AssetID(asset1),
				AtEpoch:    10,
				UntilEpoch: 15,
				Balance:    num.MustDecimalFromString("100"),
				VegaTime:   now,
			},
			{
				PartyID:    entities.PartyID(party1),
				AssetID:    entities.AssetID(asset1),
				AtEpoch:    10,
				UntilEpoch: 17,
				Balance:    num.MustDecimalFromString("200"),
				VegaTime:   now,
			},
			{
				PartyID:    entities.PartyID(party1),
				AssetID:    entities.AssetID(asset2),
				AtEpoch:    10,
				UntilEpoch: 19,
				Balance:    num.MustDecimalFromString("100"),
				VegaTime:   now,
			},
		}

		for _, v := range balances {
			require.NoError(t, plbs.Add(ctx, v))
		}

		// ensure we can still get them

		entitis, err := plbs.Get(
			ctx, ptr.From(entities.PartyID(party1)), nil)
		require.NoError(t, err)
		require.Len(t, entitis, 3)

		// try prunce, should be no-op
		err = plbs.Prune(ctx, 10)
		assert.NoError(t, err)

		// still same stuff in the DB
		entitis, err = plbs.Get(
			ctx, ptr.From(entities.PartyID(party1)), nil)
		require.NoError(t, err)
		require.Len(t, entitis, 3)
	})

	now = now.Add(24 * time.Hour).Truncate(time.Millisecond)

	t.Run("insert same locked balance with different at epoch, for a party, asset and until epoch, should still keep 3 balances", func(t *testing.T) {
		balances := []entities.PartyLockedBalance{
			{
				PartyID:    entities.PartyID(party1),
				AssetID:    entities.AssetID(asset1),
				AtEpoch:    11,
				UntilEpoch: 15,
				Balance:    num.MustDecimalFromString("100"),
				VegaTime:   now,
			},
			{
				PartyID:    entities.PartyID(party1),
				AssetID:    entities.AssetID(asset1),
				AtEpoch:    11,
				UntilEpoch: 17,
				Balance:    num.MustDecimalFromString("200"),
				VegaTime:   now,
			},
			{
				PartyID:    entities.PartyID(party1),
				AssetID:    entities.AssetID(asset2),
				AtEpoch:    11,
				UntilEpoch: 19,
				Balance:    num.MustDecimalFromString("100"),
				VegaTime:   now,
			},
		}

		for _, v := range balances {
			require.NoError(t, plbs.Add(ctx, v))
		}

		// ensure we can still get them

		entitis, err := plbs.Get(
			ctx, ptr.From(entities.PartyID(party1)), nil)
		require.NoError(t, err)
		require.Len(t, entitis, 3)

		// ensure we have the last version
		for _, v := range entitis {
			require.Equal(t, 11, int(v.AtEpoch))
		}
	})

	t.Run("then try pruning", func(t *testing.T) {
		// assume we are moving a couple of epoch later, we should have only
		// 2 locked balances left

		require.NoError(t, plbs.Prune(ctx, 16))
		entitis, err := plbs.Get(
			ctx, ptr.From(entities.PartyID(party1)), nil)
		require.NoError(t, err)
		require.Len(t, entitis, 2)
	})
}

func TestPartyLockedBalance_Add(t *testing.T) {
	bs, plbs := setupPartyLockedBalanceTest(t)

	ctx := tempTransaction(t)

	var partyLockedBalances []entities.PartyLockedBalance
	var partyLockedBalancesCurrent []entities.PartyLockedBalance

	err := pgxscan.Select(ctx, connectionSource.Connection, &partyLockedBalances, "SELECT * from party_locked_balances")
	require.NoError(t, err)

	assert.Len(t, partyLockedBalances, 0)

	err = pgxscan.Select(ctx, connectionSource.Connection, &partyLockedBalancesCurrent, "SELECT * from party_locked_balances_current")
	require.NoError(t, err)

	assert.Len(t, partyLockedBalancesCurrent, 0)

	block := addTestBlock(t, ctx, bs)

	t.Run("Add should insert a new record into the partyLockedBalances table", func(t *testing.T) {
		want := entities.PartyLockedBalance{
			PartyID:    "deadbeef01",
			AssetID:    "cafedaad01",
			AtEpoch:    100,
			UntilEpoch: 200,
			Balance:    num.DecimalFromInt64(10000000000),
			VegaTime:   block.VegaTime,
		}

		err := plbs.Add(ctx, want)
		require.NoError(t, err)

		err = pgxscan.Select(ctx, connectionSource.Connection, &partyLockedBalances, "SELECT * from party_locked_balances")
		require.NoError(t, err)

		assert.Len(t, partyLockedBalances, 1)
		assert.Equal(t, want, partyLockedBalances[0])

		t.Run("And a record into the party_locked_balances_current table if it doesn't already exist", func(t *testing.T) {
			err = pgxscan.Select(ctx, connectionSource.Connection, &partyLockedBalancesCurrent, "SELECT * from party_locked_balances_current")
			require.NoError(t, err)

			assert.Len(t, partyLockedBalancesCurrent, 1)
			assert.Equal(t, want, partyLockedBalancesCurrent[0])
		})

		t.Run("And update the record in the party_locked_balances_current table if the party and asset already exists", func(t *testing.T) {
			block = addTestBlock(t, ctx, bs)
			want2 := entities.PartyLockedBalance{
				PartyID:    "deadbeef01",
				AssetID:    "cafedaad01",
				AtEpoch:    150,
				UntilEpoch: 200,
				Balance:    num.DecimalFromInt64(15000000000),
				VegaTime:   block.VegaTime,
			}

			err = plbs.Add(ctx, want2)
			err = pgxscan.Select(ctx, connectionSource.Connection, &partyLockedBalances, "SELECT * from party_locked_balances order by vega_time")
			require.NoError(t, err)

			assert.Len(t, partyLockedBalances, 2)
			assert.Equal(t, want, partyLockedBalances[0])
			assert.Equal(t, want2, partyLockedBalances[1])

			err = pgxscan.Select(ctx, connectionSource.Connection, &partyLockedBalancesCurrent, "SELECT * from party_locked_balances_current")
			require.NoError(t, err)

			assert.Len(t, partyLockedBalancesCurrent, 1)
			assert.Equal(t, want2, partyLockedBalancesCurrent[0])
		})
	})
}

func setupHistoricPartyLockedBalances(t *testing.T, ctx context.Context, bs *sqlstore.Blocks, plbs *sqlstore.PartyLockedBalance) []entities.PartyLockedBalance {
	t.Helper()

	parties := []string{
		"deadbeef01",
		"deadbeef02",
		"deadbeef03",
	}

	assets := []string{
		"cafedaad01",
		"cafedaad02",
	}

	currentBalances := make([]entities.PartyLockedBalance, 0)

	for i := 0; i < 3; i++ { // versions
		block := addTestBlock(t, ctx, bs)
		for _, party := range parties {
			for _, asset := range assets {
				balance := entities.PartyLockedBalance{
					PartyID:    entities.PartyID(party),
					AssetID:    entities.AssetID(asset),
					AtEpoch:    100 + uint64(i),
					UntilEpoch: 200,
					Balance:    num.DecimalFromInt64(10000000000 + int64(i*10000000)),
					VegaTime:   block.VegaTime,
				}
				err := plbs.Add(ctx, balance)
				require.NoError(t, err)
				if i == 2 {
					currentBalances = append(currentBalances, balance)
				}
			}
		}
	}
	return currentBalances
}

func TestPartyLockedBalance_Get(t *testing.T) {
	t.Run("Get should return all current record if party and asset is not provided", testPartyLockedBalanceGetAll)
	t.Run("Get should return all current record for a party if it is provided", testPartyLockedBalanceGetByParty)
	t.Run("Get should return all current records for an asset if it is provided", testPartyLockedBalancesGetByAsset)
	t.Run("Get should return all current records for a party and asset", testPartyLockedBalancesGetByPartyAndAsset)
}

func testPartyLockedBalanceGetAll(t *testing.T) {
	bs, plbs := setupPartyLockedBalanceTest(t)

	ctx := tempTransaction(t)

	currentBalances := setupHistoricPartyLockedBalances(t, ctx, bs, plbs)

	balances, err := plbs.Get(ctx, nil, nil)
	require.NoError(t, err)

	assert.Len(t, balances, len(currentBalances))
	assert.Equal(t, currentBalances, balances)
}

func testPartyLockedBalanceGetByParty(t *testing.T) {
	bs, plbs := setupPartyLockedBalanceTest(t)

	ctx := tempTransaction(t)

	currentBalances := setupHistoricPartyLockedBalances(t, ctx, bs, plbs)
	partyID := entities.PartyID("deadbeef01")

	want := make([]entities.PartyLockedBalance, 0)

	for _, balance := range currentBalances {
		if balance.PartyID == partyID {
			want = append(want, balance)
		}
	}

	balances, err := plbs.Get(ctx, &partyID, nil)
	require.NoError(t, err)

	assert.Len(t, balances, len(want))
	assert.Equal(t, want, balances)
}

func testPartyLockedBalancesGetByAsset(t *testing.T) {
	bs, plbs := setupPartyLockedBalanceTest(t)

	ctx := tempTransaction(t)

	currentBalances := setupHistoricPartyLockedBalances(t, ctx, bs, plbs)
	assetID := entities.AssetID("cafedaad01")

	want := make([]entities.PartyLockedBalance, 0)

	for _, balance := range currentBalances {
		if balance.AssetID == assetID {
			want = append(want, balance)
		}
	}

	balances, err := plbs.Get(ctx, nil, &assetID)
	require.NoError(t, err)

	assert.Len(t, balances, len(want))
	assert.Equal(t, want, balances)
}

func testPartyLockedBalancesGetByPartyAndAsset(t *testing.T) {
	bs, plbs := setupPartyLockedBalanceTest(t)

	ctx := tempTransaction(t)

	currentBalances := setupHistoricPartyLockedBalances(t, ctx, bs, plbs)
	partyID := entities.PartyID("deadbeef01")
	assetID := entities.AssetID("cafedaad01")

	want := make([]entities.PartyLockedBalance, 0)

	for _, balance := range currentBalances {
		if balance.PartyID == partyID && balance.AssetID == assetID {
			want = append(want, balance)
		}
	}

	balances, err := plbs.Get(ctx, &partyID, &assetID)
	require.NoError(t, err)

	assert.Len(t, balances, len(want))
	assert.Equal(t, want, balances)
}
