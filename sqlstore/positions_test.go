package sqlstore_test

import (
	"context"
	"fmt"
	"testing"
	"time"

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
	defer DeleteEverything()
	ctx := context.Background()
	ps := sqlstore.NewPositions(connectionSource)
	qs := sqlstore.NewParties(connectionSource)
	bs := sqlstore.NewBlocks(connectionSource)

	block1 := addTestBlockForTime(t, bs, time.Now().Add((-26*time.Hour)-(2*time.Second)))
	block2 := addTestBlockForTime(t, bs, time.Now().Add((-26*time.Hour)-(1*time.Second)))
	block3 := addTestBlockForTime(t, bs, time.Now().Add(-26*time.Hour))

	market1 := entities.Market{ID: entities.NewMarketID("dead")}
	market2 := entities.Market{ID: entities.NewMarketID("beef")}
	party1 := addTestParty(t, qs, block1)
	party2 := addTestParty(t, qs, block1)

	pos1a := addTestPosition(t, ps, market1, party1, 100, block1)
	pos1b := addTestPosition(t, ps, market1, party1, 200, block1)

	pos2 := addTestPosition(t, ps, market1, party2, 300, block2)
	pos3 := addTestPosition(t, ps, market2, party1, 400, block2)

	ps.Flush(ctx)
	_, _ = pos1a, pos1b

	// Conflate the data and add some new positions so all tests run against a mix of conflated and non-conflated data
	now := time.Now()
	_, err := connectionSource.Connection.Exec(context.Background(), fmt.Sprintf("CALL refresh_continuous_aggregate('conflated_positions', '%s', '%s');",
		now.Add(-48*time.Hour).Format("2006-01-02"),
		time.Now().Format("2006-01-02")))

	assert.NoError(t, err)

	// The refresh of the continuous aggregate completes asynchronously so the following loop is necessary to ensure the data has been materialized
	// before the test continues
	for {
		var counter int
		connectionSource.Connection.QueryRow(context.Background(), "SELECT count(*) FROM conflated_positions").Scan(&counter)
		if counter == 3 {
			break
		}
	}

	_, err = connectionSource.Connection.Exec(context.Background(), "delete from positions")
	assert.NoError(t, err)

	// Add some new positions to the non-conflated data
	pos1c := addTestPosition(t, ps, market1, party1, 200, block3)
	pos4 := addTestPosition(t, ps, market2, party2, 500, block3)
	ps.Flush(ctx)

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
		assert.True(t, expected.Equal(actual))
	})

	t.Run("GetBadMarketAndParty", func(t *testing.T) {
		_, err := ps.GetByMarketAndParty(ctx, market2.ID, entities.NewPartyID("ffff"))
		assert.ErrorIs(t, err, sqlstore.ErrPositionNotFound)
	})

}
