package sqlstore_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/sqlstore"
	v2 "code.vegaprotocol.io/protos/data-node/api/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func addTestParty(t *testing.T, ps *sqlstore.Parties, block entities.Block) entities.Party {
	party := entities.Party{
		ID:       entities.NewPartyID(generateID()),
		VegaTime: &block.VegaTime,
	}

	err := ps.Add(context.Background(), party)
	require.NoError(t, err)
	return party
}

func TestParty(t *testing.T) {
	defer DeleteEverything()
	ctx := context.Background()
	ps := sqlstore.NewParties(connectionSource)
	ps.Initialise()
	bs := sqlstore.NewBlocks(connectionSource)
	block := addTestBlock(t, bs)

	// Make sure we're starting with an empty set of parties (except network party)
	parties, err := ps.GetAll(ctx)
	require.NoError(t, err)
	assert.Len(t, parties, 1)
	assert.Equal(t, "network", parties[0].ID.String())

	// Make a new party
	party := addTestParty(t, ps, block)

	// Add it again, we shouldn't get a primary key violation (we just ignore)
	err = ps.Add(ctx, party)
	require.NoError(t, err)

	// Query and check we've got back a party the same as the one we put in
	fetchedParty, err := ps.GetByID(ctx, party.ID.String())
	require.NoError(t, err)
	assert.Equal(t, party, fetchedParty)

	// Get all assets and make sure ours is in there (along with built in network party)
	parties, err = ps.GetAll(ctx)
	require.NoError(t, err)
	assert.Len(t, parties, 2)

	// Check we get the right error if we ask for a non-existent party
	_, err = ps.GetByID(ctx, "beef")
	assert.ErrorIs(t, err, sqlstore.ErrPartyNotFound)
	fmt.Println("yay")
}

func setupPartyTest(t *testing.T) (*sqlstore.Blocks, *sqlstore.Parties, sqlstore.Config, func(t *testing.T)) {
	t.Helper()
	bs := sqlstore.NewBlocks(connectionSource)
	pt := sqlstore.NewParties(connectionSource)

	DeleteEverything()

	config := sqlstore.NewDefaultConfig()
	config.ConnectionConfig.Port = testDBPort

	return bs, pt, config, func(t *testing.T) {
		DeleteEverything()
	}
}

func populateTestParties(ctx context.Context, t *testing.T, bs *sqlstore.Blocks, ps *sqlstore.Parties, blockTimes map[string]time.Time) {
	t.Helper()
	parties := []entities.Party{
		{
			ID: entities.NewPartyID("02a16077"),
		},
		{
			ID: entities.NewPartyID("44eea1bc"),
		},
		{
			ID: entities.NewPartyID("65be62cd"),
		},
		{
			ID: entities.NewPartyID("7a797e0e"),
		},
		{
			ID: entities.NewPartyID("7bb2356e"),
		},
		{
			ID: entities.NewPartyID("b7c84b8e"),
		},
		{
			ID: entities.NewPartyID("c612300d"),
		},
		{
			ID: entities.NewPartyID("c8744329"),
		},
		{
			ID: entities.NewPartyID("da8d1803"),
		},
		{
			ID: entities.NewPartyID("fb1528a5"),
		},
	}

	for _, party := range parties {
		block := addTestBlock(t, bs)
		party.VegaTime = &block.VegaTime
		blockTimes[party.ID.String()] = block.VegaTime
		err := ps.Add(ctx, party)
		require.NoError(t, err)
		time.Sleep(time.Microsecond * 100)
	}
}

func TestPartyPagination(t *testing.T) {
	t.Run("CursorPagination should return the party if Party ID is provided", testPartyPaginationReturnsTheSpecifiedParty)
	t.Run("CursorPagination should return all parties if no party ID and no cursor is provided", testPartyPaginationReturnAllParties)
	t.Run("CursorPagination should return the first page when first limit is provided with no after cursor", testPartyPaginationReturnsFirstPage)
	t.Run("CursorPagination should return the last page when last limit is provided with no before cursor", testPartyPaginationReturnsLastPage)
	t.Run("CursorPagination should return the page specified by the first limit and after cursor", testPartyPaginationReturnsPageTraversingForward)
	t.Run("CursorPagination should return the page specified by the last limit and before cursor", testPartyPaginationReturnsPageTraversingBackward)
}

func testPartyPaginationReturnsTheSpecifiedParty(t *testing.T) {
	bs, pt, _, cleanup := setupPartyTest(t)
	defer cleanup(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	blockTimes := make(map[string]time.Time)
	populateTestParties(ctx, t, bs, pt, blockTimes)
	cursor, err := entities.CursorPaginationFromProto(&v2.Pagination{
		First:  nil,
		After:  nil,
		Last:   nil,
		Before: nil,
	})

	require.NoError(t, err)
	parties, pageInfo, err := pt.GetAllPaged(ctx, "c612300d", cursor)
	require.NoError(t, err)
	assert.Len(t, parties, 1)
	assert.Equal(t, "c612300d", parties[0].ID.String())

	wantStartCursor := entities.NewCursor(blockTimes["c612300d"].UTC().Format(time.RFC3339Nano)).Encode()
	wantEndCursor := entities.NewCursor(blockTimes["c612300d"].UTC().Format(time.RFC3339Nano)).Encode()
	assert.False(t, pageInfo.HasNextPage)
	assert.False(t, pageInfo.HasPreviousPage)
	assert.Equal(t, wantStartCursor, pageInfo.EndCursor)
	assert.Equal(t, wantEndCursor, pageInfo.StartCursor)
}

func testPartyPaginationReturnAllParties(t *testing.T) {
	bs, pt, _, cleanup := setupPartyTest(t)
	defer cleanup(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	blockTimes := make(map[string]time.Time)
	populateTestParties(ctx, t, bs, pt, blockTimes)
	cursor, err := entities.CursorPaginationFromProto(&v2.Pagination{
		First:  nil,
		After:  nil,
		Last:   nil,
		Before: nil,
	})

	require.NoError(t, err)
	parties, pageInfo, err := pt.GetAllPaged(ctx, "", cursor)
	require.NoError(t, err)
	assert.Len(t, parties, 10)
	assert.Equal(t, "02a16077", parties[0].ID.String())
	assert.Equal(t, "fb1528a5", parties[9].ID.String())

	wantStartCursor := entities.NewCursor(blockTimes["02a16077"].UTC().Format(time.RFC3339Nano)).Encode()
	wantEndCursor := entities.NewCursor(blockTimes["fb1528a5"].UTC().Format(time.RFC3339Nano)).Encode()
	assert.False(t, pageInfo.HasNextPage)
	assert.False(t, pageInfo.HasPreviousPage)
	assert.Equal(t, wantStartCursor, pageInfo.StartCursor)
	assert.Equal(t, wantEndCursor, pageInfo.EndCursor)
}

func testPartyPaginationReturnsFirstPage(t *testing.T) {
	bs, pt, _, cleanup := setupPartyTest(t)
	defer cleanup(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	blockTimes := make(map[string]time.Time)
	populateTestParties(ctx, t, bs, pt, blockTimes)
	first := int32(3)
	cursor, err := entities.CursorPaginationFromProto(&v2.Pagination{
		First:  &first,
		After:  nil,
		Last:   nil,
		Before: nil,
	})

	require.NoError(t, err)

	parties, pageInfo, err := pt.GetAllPaged(ctx, "", cursor)
	require.NoError(t, err)
	assert.Len(t, parties, 3)
	assert.Equal(t, "02a16077", parties[0].ID.String())
	assert.Equal(t, "65be62cd", parties[2].ID.String())

	wantStartCursor := entities.NewCursor(blockTimes["02a16077"].UTC().Format(time.RFC3339Nano)).Encode()
	wantEndCursor := entities.NewCursor(blockTimes["65be62cd"].UTC().Format(time.RFC3339Nano)).Encode()
	assert.True(t, pageInfo.HasNextPage)
	assert.False(t, pageInfo.HasPreviousPage)
	assert.Equal(t, wantStartCursor, pageInfo.StartCursor)
	assert.Equal(t, wantEndCursor, pageInfo.EndCursor)
}

func testPartyPaginationReturnsLastPage(t *testing.T) {
	bs, pt, _, cleanup := setupPartyTest(t)
	defer cleanup(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	blockTimes := make(map[string]time.Time)
	populateTestParties(ctx, t, bs, pt, blockTimes)
	last := int32(3)
	cursor, err := entities.CursorPaginationFromProto(&v2.Pagination{
		First:  nil,
		After:  nil,
		Last:   &last,
		Before: nil,
	})

	require.NoError(t, err)

	parties, pageInfo, err := pt.GetAllPaged(ctx, "", cursor)
	require.NoError(t, err)
	assert.Len(t, parties, 3)
	assert.Equal(t, "c8744329", parties[0].ID.String())
	assert.Equal(t, "fb1528a5", parties[2].ID.String())

	wantStartCursor := entities.NewCursor(blockTimes["c8744329"].UTC().Format(time.RFC3339Nano)).Encode()
	wantEndCursor := entities.NewCursor(blockTimes["fb1528a5"].UTC().Format(time.RFC3339Nano)).Encode()
	assert.False(t, pageInfo.HasNextPage)
	assert.True(t, pageInfo.HasPreviousPage)
	assert.Equal(t, wantStartCursor, pageInfo.StartCursor)
	assert.Equal(t, wantEndCursor, pageInfo.EndCursor)
}

func testPartyPaginationReturnsPageTraversingForward(t *testing.T) {
	bs, pt, _, cleanup := setupPartyTest(t)
	defer cleanup(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	blockTimes := make(map[string]time.Time)
	populateTestParties(ctx, t, bs, pt, blockTimes)
	first := int32(3)
	after := entities.NewCursor(blockTimes["65be62cd"].Format(time.RFC3339Nano)).Encode()
	cursor, err := entities.CursorPaginationFromProto(&v2.Pagination{
		First:  &first,
		After:  &after,
		Last:   nil,
		Before: nil,
	})

	require.NoError(t, err)

	parties, pageInfo, err := pt.GetAllPaged(ctx, "", cursor)
	require.NoError(t, err)
	assert.Len(t, parties, 3)
	assert.Equal(t, "7a797e0e", parties[0].ID.String())
	assert.Equal(t, "b7c84b8e", parties[2].ID.String())

	wantStartCursor := entities.NewCursor(blockTimes["7a797e0e"].UTC().Format(time.RFC3339Nano)).Encode()
	wantEndCursor := entities.NewCursor(blockTimes["b7c84b8e"].UTC().Format(time.RFC3339Nano)).Encode()
	assert.True(t, pageInfo.HasNextPage)
	assert.True(t, pageInfo.HasPreviousPage)
	assert.Equal(t, wantStartCursor, pageInfo.StartCursor)
	assert.Equal(t, wantEndCursor, pageInfo.EndCursor)
}

func testPartyPaginationReturnsPageTraversingBackward(t *testing.T) {
	bs, pt, _, cleanup := setupPartyTest(t)
	defer cleanup(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	blockTimes := make(map[string]time.Time)
	populateTestParties(ctx, t, bs, pt, blockTimes)
	last := int32(3)
	before := entities.NewCursor(blockTimes["c8744329"].Format(time.RFC3339Nano)).Encode()
	cursor, err := entities.CursorPaginationFromProto(&v2.Pagination{
		First:  nil,
		After:  nil,
		Last:   &last,
		Before: &before,
	})

	require.NoError(t, err)

	parties, pageInfo, err := pt.GetAllPaged(ctx, "", cursor)
	require.NoError(t, err)
	assert.Len(t, parties, 3)
	assert.Equal(t, "7bb2356e", parties[0].ID.String())
	assert.Equal(t, "c612300d", parties[2].ID.String())

	wantStartCursor := entities.NewCursor(blockTimes["7bb2356e"].UTC().Format(time.RFC3339Nano)).Encode()
	wantEndCursor := entities.NewCursor(blockTimes["c612300d"].UTC().Format(time.RFC3339Nano)).Encode()
	assert.True(t, pageInfo.HasNextPage)
	assert.True(t, pageInfo.HasPreviousPage)
	assert.Equal(t, wantStartCursor, pageInfo.StartCursor)
	assert.Equal(t, wantEndCursor, pageInfo.EndCursor)
}
