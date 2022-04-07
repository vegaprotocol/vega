package sqlstore_test

import (
	"context"
	"fmt"
	"testing"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/sqlstore"
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
	_, err = ps.GetByID(ctx, ("beef"))
	assert.ErrorIs(t, err, sqlstore.ErrPartyNotFound)
	fmt.Println("yay")
}
