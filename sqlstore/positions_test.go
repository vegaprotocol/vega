package sqlstore_test

import (
	"context"
	"testing"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/sqlstore"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func addTestPosition(t *testing.T,
	ps *sqlstore.Positions,
	market entities.Market,
	party entities.Party,
	volume int64,
	block entities.Block,
) entities.Position {
	pos := entities.NewEmptyPosition(market.ID, party.ID)
	pos.OpenVolume = volume
	pos.VegaTime = block.VegaTime
	err := ps.Add(context.Background(), pos)
	require.NoError(t, err)
	return pos
}

func positionLessThan(x, y entities.Position) bool {
	if x.MarketID != y.MarketID {
		return x.MarketID.String() < y.MarketID.String()
	}
	return x.PartyID.String() < y.PartyID.String()
}

func assertPositionsMatch(t *testing.T, expected, actual []entities.Position) {
	t.Helper()
	sortPositions := cmpopts.SortSlices(positionLessThan)
	assert.Empty(t, cmp.Diff(actual, expected, sortPositions))
}

func TestPosition(t *testing.T) {
	defer testStore.DeleteEverything()
	ctx := context.Background()
	ps := sqlstore.NewPositions(testStore)
	qs := sqlstore.NewParties(testStore)
	bs := sqlstore.NewBlocks(testStore)

	block1 := addTestBlock(t, bs)
	block2 := addTestBlock(t, bs)

	market1 := entities.Market{ID: entities.NewMarketID("dead")}
	market2 := entities.Market{ID: entities.NewMarketID("beef")}
	party1 := addTestParty(t, qs, block1)
	party2 := addTestParty(t, qs, block1)

	pos1a := addTestPosition(t, ps, market1, party1, 100, block1)
	pos1b := addTestPosition(t, ps, market1, party1, 200, block1)
	pos1c := addTestPosition(t, ps, market1, party1, 200, block2)
	pos2 := addTestPosition(t, ps, market1, party2, 300, block2)
	pos3 := addTestPosition(t, ps, market2, party1, 400, block2)
	pos4 := addTestPosition(t, ps, market2, party2, 500, block2)

	_, _ = pos1a, pos1b

	t.Run("GetAll", func(t *testing.T) {
		expected := []entities.Position{pos1c, pos2, pos3, pos4}
		actual, err := ps.GetAll(ctx)
		require.NoError(t, err)
		assertPositionsMatch(t, expected, actual)
	})

	t.Run("GetByParty", func(t *testing.T) {
		expected := []entities.Position{pos1c, pos3}
		actual, err := ps.GetByParty(ctx, party1.ID)
		require.NoError(t, err)
		assertPositionsMatch(t, expected, actual)
	})

	t.Run("GetByMarket", func(t *testing.T) {
		expected := []entities.Position{pos1c, pos2}
		actual, err := ps.GetByMarket(ctx, market1.ID)
		require.NoError(t, err)
		assertPositionsMatch(t, expected, actual)
	})

	t.Run("GetByMarketAndParty", func(t *testing.T) {
		expected := pos4
		actual, err := ps.GetByMarketAndParty(ctx, market2.ID, party2.ID)
		require.NoError(t, err)
		assert.Equal(t, expected, actual)
	})

	t.Run("GetBadMarketAndParty", func(t *testing.T) {
		_, err := ps.GetByMarketAndParty(ctx, market2.ID, entities.NewPartyID("ffff"))
		assert.ErrorIs(t, err, sqlstore.ErrPositionNotFound)
	})
}
