package sqlstore_test

import (
	"testing"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/sqlstore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func addTestParty(t *testing.T, ps *sqlstore.Parties, block entities.Block) entities.Party {
	party := entities.Party{
		ID:       generateID(),
		VegaTime: block.VegaTime,
	}

	err := ps.Add(party)
	require.NoError(t, err)
	return party
}
func TestParty(t *testing.T) {
	defer testStore.DeleteEverything()
	ps := sqlstore.NewParties(testStore)
	bs := sqlstore.NewBlocks(testStore)
	block := addTestBlock(t, bs)

	// Make sure we're starting with an empty set of parties
	parties, err := ps.GetAll()
	assert.NoError(t, err)
	assert.Empty(t, parties)

	// Make a new party
	party := addTestParty(t, ps, block)

	// Add it again, we should get a primary key violation
	err = ps.Add(party)
	assert.Error(t, err)

	// Query and check we've got back a party the same as the one we put in
	fetchedParty, err := ps.GetByID(party.HexId())
	assert.NoError(t, err)
	assert.Equal(t, party, fetchedParty)

	// Get all assets and make sure ours is in there
	parties, err = ps.GetAll()
	assert.NoError(t, err)
	assert.Len(t, parties, 1)
}
