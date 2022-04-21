package sqlstore_test

import (
	"context"
	"testing"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/sqlstore"
	"code.vegaprotocol.io/protos/vega"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func addTestProposal(t *testing.T, ps *sqlstore.Proposals, party entities.Party, block entities.Block) entities.Proposal {
	terms := entities.ProposalTerms{ProposalTerms: &vega.ProposalTerms{}}
	p := entities.Proposal{
		ID:           entities.NewProposalID(generateID()),
		PartyID:      party.ID,
		Reference:    generateID(),
		Terms:        terms,
		State:        entities.ProposalStateEnacted,
		VegaTime:     block.VegaTime,
		ProposalTime: block.VegaTime,
	}
	ps.Add(context.Background(), p)
	return p
}

func proposalLessThan(x, y entities.Proposal) bool {
	return x.ID.String() < y.ID.String()
}

func assertProposalsMatch(t *testing.T, expected, actual []entities.Proposal) {
	t.Helper()
	sortProposals := cmpopts.SortSlices(proposalLessThan)
	ignoreProtoState := cmpopts.IgnoreUnexported(vega.ProposalTerms{})
	assert.Empty(t, cmp.Diff(actual, expected, sortProposals, ignoreProtoState))
}

func assertProposalMatch(t *testing.T, expected, actual entities.Proposal) {
	t.Helper()
	ignoreProtoState := cmpopts.IgnoreUnexported(vega.ProposalTerms{})
	assert.Empty(t, cmp.Diff(actual, expected, ignoreProtoState))
}

func TestProposals(t *testing.T) {
	defer DeleteEverything()
	ctx := context.Background()
	partyStore := sqlstore.NewParties(connectionSource)
	propStore := sqlstore.NewProposals(connectionSource)
	blockStore := sqlstore.NewBlocks(connectionSource)
	block1 := addTestBlock(t, blockStore)

	party1 := addTestParty(t, partyStore, block1)
	party2 := addTestParty(t, partyStore, block1)
	prop1 := addTestProposal(t, propStore, party1, block1)
	prop2 := addTestProposal(t, propStore, party2, block1)

	party1ID := party1.ID.String()
	prop1ID := prop1.ID.String()

	t.Run("GetById", func(t *testing.T) {
		expected := prop1
		actual, err := propStore.GetByID(ctx, prop1ID)
		require.NoError(t, err)
		assertProposalMatch(t, expected, actual)
	})

	t.Run("GetByReference", func(t *testing.T) {
		expected := prop2
		actual, err := propStore.GetByReference(ctx, prop2.Reference)
		require.NoError(t, err)
		assertProposalMatch(t, expected, actual)
	})

	t.Run("GetInState", func(t *testing.T) {
		enacted := entities.ProposalStateEnacted
		expected := []entities.Proposal{prop1, prop2}
		actual, err := propStore.Get(ctx, &enacted, nil, nil)
		require.NoError(t, err)
		assertProposalsMatch(t, expected, actual)
	})

	t.Run("GetByParty", func(t *testing.T) {
		expected := []entities.Proposal{prop1}
		actual, err := propStore.Get(ctx, nil, &party1ID, nil)
		require.NoError(t, err)
		assertProposalsMatch(t, expected, actual)
	})
}
