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

func addTestVote(t *testing.T, vs *sqlstore.Votes,
	party entities.Party,
	proposal entities.Proposal,
	value entities.VoteValue,
	block entities.Block,
) entities.Vote {
	r := entities.Vote{
		PartyID:                     party.ID,
		ProposalID:                  proposal.ID,
		Value:                       value,
		TotalGovernanceTokenBalance: decimal.NewFromInt(100),
		TotalGovernanceTokenWeight:  decimal.NewFromFloat(0.1),
		TotalEquityLikeShareWeight:  decimal.NewFromFloat(0.3),
		VegaTime:                    block.VegaTime,
	}
	err := vs.Add(context.Background(), r)
	require.NoError(t, err)
	return r
}

func voteLessThan(x, y entities.Vote) bool {
	if x.PartyID.String() != y.PartyID.String() {
		return x.PartyID.String() < y.PartyID.String()
	}
	return x.ProposalID.String() < y.ProposalID.String()
}

func assertVotesMatch(t *testing.T, expected, actual []entities.Vote) {
	t.Helper()
	assert.Empty(t, cmp.Diff(actual, expected, cmpopts.SortSlices(voteLessThan)))
}

func TestVotes(t *testing.T) {
	defer testStore.DeleteEverything()
	partyStore := sqlstore.NewParties(testStore)
	propStore := sqlstore.NewProposals(testStore)
	voteStore := sqlstore.NewVotes(testStore)
	blockStore := sqlstore.NewBlocks(testStore)
	block1 := addTestBlock(t, blockStore)
	block2 := addTestBlock(t, blockStore)

	party1 := addTestParty(t, partyStore, block1)
	party2 := addTestParty(t, partyStore, block1)
	prop1 := addTestProposal(t, propStore, party1, block1)
	prop2 := addTestProposal(t, propStore, party1, block1)

	party1ID := party1.ID.String()
	prop1ID := prop1.ID.String()

	vote1 := addTestVote(t, voteStore, party1, prop1, entities.VoteValueYes, block1)
	vote2 := addTestVote(t, voteStore, party1, prop2, entities.VoteValueYes, block1)
	// Change vote in same block
	vote3 := addTestVote(t, voteStore, party2, prop1, entities.VoteValueYes, block1)
	vote3b := addTestVote(t, voteStore, party2, prop1, entities.VoteValueNo, block1)
	// Change vote in next block
	vote4 := addTestVote(t, voteStore, party2, prop2, entities.VoteValueYes, block1)
	vote4b := addTestVote(t, voteStore, party2, prop2, entities.VoteValueNo, block2)

	_ = vote3
	_ = vote4

	t.Run("GetAll", func(t *testing.T) {
		expected := []entities.Vote{vote1, vote2, vote3b, vote4b}
		actual, err := voteStore.Get(context.Background(), nil, nil, nil)
		require.NoError(t, err)
		assertVotesMatch(t, expected, actual)
	})

	t.Run("GetByProposal", func(t *testing.T) {
		expected := []entities.Vote{vote1, vote3b}
		actual, err := voteStore.Get(context.Background(), &prop1ID, nil, nil)
		require.NoError(t, err)
		assertVotesMatch(t, expected, actual)
	})

	t.Run("GetByParty", func(t *testing.T) {
		expected := []entities.Vote{vote1, vote2}
		actual, err := voteStore.Get(context.Background(), nil, &party1ID, nil)
		require.NoError(t, err)
		assertVotesMatch(t, expected, actual)
	})

	t.Run("GetByValue", func(t *testing.T) {
		expected := []entities.Vote{vote3b, vote4b}
		no := entities.VoteValueNo
		actual, err := voteStore.Get(context.Background(), nil, nil, &no)
		require.NoError(t, err)
		assertVotesMatch(t, expected, actual)
	})

	t.Run("GetByEverything", func(t *testing.T) {
		expected := []entities.Vote{vote1}
		yes := entities.VoteValueYes
		actual, err := voteStore.Get(context.Background(), &prop1ID, &party1ID, &yes)
		require.NoError(t, err)
		assertVotesMatch(t, expected, actual)
	})
}
