// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.DATANODE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package sqlstore_test

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/sqlstore"
	"code.vegaprotocol.io/vega/datanode/sqlstore/helpers"
	"code.vegaprotocol.io/vega/protos/vega"
)

func addTestVote(t *testing.T,
	ctx context.Context,
	vs *sqlstore.Votes,
	party entities.Party,
	proposal entities.Proposal,
	value entities.VoteValue,
	block entities.Block,
	txHash entities.TxHash,
) entities.Vote {
	t.Helper()
	r := entities.Vote{
		PartyID:                     party.ID,
		ProposalID:                  proposal.ID,
		Value:                       value,
		TotalGovernanceTokenBalance: decimal.NewFromInt(100),
		TotalGovernanceTokenWeight:  decimal.NewFromFloat(0.1),
		TotalEquityLikeShareWeight:  decimal.NewFromFloat(0.3),
		VegaTime:                    block.VegaTime,
		InitialTime:                 block.VegaTime,
		TxHash:                      txHash,
	}
	err := vs.Add(ctx, r)
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
	ctx, rollback := tempTransaction(t)
	defer rollback()

	partyStore := sqlstore.NewParties(connectionSource)
	propStore := sqlstore.NewProposals(connectionSource)
	voteStore := sqlstore.NewVotes(connectionSource)
	blockStore := sqlstore.NewBlocks(connectionSource)
	block1 := addTestBlock(t, ctx, blockStore)
	block2 := addTestBlock(t, ctx, blockStore)

	party1 := addTestParty(t, ctx, partyStore, block1)
	party2 := addTestParty(t, ctx, partyStore, block1)
	prop1 := addTestProposal(t, ctx, propStore, helpers.GenerateID(), party1, helpers.GenerateID(), block1, entities.ProposalStateEnacted, entities.ProposalRationale{ProposalRationale: &vega.ProposalRationale{Title: "myurl1.com", Description: "desc"}}, entities.ProposalTerms{ProposalTerms: &vega.ProposalTerms{Change: &vega.ProposalTerms_NewMarket{}}}, entities.ProposalErrorUnspecified)
	prop2 := addTestProposal(t, ctx, propStore, helpers.GenerateID(), party1, helpers.GenerateID(), block1, entities.ProposalStateEnacted, entities.ProposalRationale{ProposalRationale: &vega.ProposalRationale{Title: "myurl2.com", Description: "desc"}}, entities.ProposalTerms{ProposalTerms: &vega.ProposalTerms{Change: &vega.ProposalTerms_NewMarket{}}}, entities.ProposalErrorUnspecified)

	party1ID := party1.ID.String()
	prop1ID := prop1.ID.String()

	txHash := txHashFromString("tx_vote_1")
	txHash2 := txHashFromString("tx_vote_2")

	vote1 := addTestVote(t, ctx, voteStore, party1, prop1, entities.VoteValueYes, block1, txHash)
	vote2 := addTestVote(t, ctx, voteStore, party1, prop2, entities.VoteValueYes, block1, txHash2)
	// Change vote in same block
	vote3 := addTestVote(t, ctx, voteStore, party2, prop1, entities.VoteValueYes, block1, txHashFromString("tx_vote_3"))
	vote3b := addTestVote(t, ctx, voteStore, party2, prop1, entities.VoteValueNo, block1, txHashFromString("tx_vote_4"))
	// Change vote in next block
	vote4 := addTestVote(t, ctx, voteStore, party2, prop2, entities.VoteValueYes, block1, txHashFromString("tx_vote_5"))
	vote4b := addTestVote(t, ctx, voteStore, party2, prop2, entities.VoteValueNo, block2, txHashFromString("tx_vote_6"))

	_ = vote3
	_ = vote4

	t.Run("GetAll", func(t *testing.T) {
		expected := []entities.Vote{vote1, vote2, vote3b, vote4b}
		actual, err := voteStore.Get(ctx, nil, nil, nil)
		require.NoError(t, err)
		assertVotesMatch(t, expected, actual)
	})

	t.Run("GetByTxHash", func(t *testing.T) {
		expected := []entities.Vote{vote1}
		actual, err := voteStore.GetByTxHash(ctx, txHash)
		require.NoError(t, err)
		assertVotesMatch(t, expected, actual)

		expected = []entities.Vote{vote2}
		actual, err = voteStore.GetByTxHash(ctx, txHash2)
		require.NoError(t, err)
		assertVotesMatch(t, expected, actual)
	})

	t.Run("GetByProposal", func(t *testing.T) {
		expected := []entities.Vote{vote1, vote3b}
		actual, err := voteStore.Get(ctx, &prop1ID, nil, nil)
		require.NoError(t, err)
		assertVotesMatch(t, expected, actual)
	})

	t.Run("GetByParty", func(t *testing.T) {
		expected := []entities.Vote{vote1, vote2}
		actual, err := voteStore.Get(ctx, nil, &party1ID, nil)
		require.NoError(t, err)
		assertVotesMatch(t, expected, actual)
	})

	t.Run("GetByValue", func(t *testing.T) {
		expected := []entities.Vote{vote3b, vote4b}
		no := entities.VoteValueNo
		actual, err := voteStore.Get(ctx, nil, nil, &no)
		require.NoError(t, err)
		assertVotesMatch(t, expected, actual)
	})

	t.Run("GetByEverything", func(t *testing.T) {
		expected := []entities.Vote{vote1}
		yes := entities.VoteValueYes
		actual, err := voteStore.Get(ctx, &prop1ID, &party1ID, &yes)
		require.NoError(t, err)
		assertVotesMatch(t, expected, actual)
	})

	t.Run("GetConnectionByEverything", func(t *testing.T) {
		expected := []entities.Vote{vote1}
		actual, _, err := voteStore.GetConnection(ctx, &prop1ID, &party1ID, entities.DefaultCursorPagination(true))
		require.NoError(t, err)
		assertVotesMatch(t, expected, actual)
	})
}

func setupPaginationTestVotes(t *testing.T, ctx context.Context) (*sqlstore.Votes, entities.Party, []entities.Vote) {
	t.Helper()
	votes := make([]entities.Vote, 0, 10)

	partyStore := sqlstore.NewParties(connectionSource)
	propStore := sqlstore.NewProposals(connectionSource)
	voteStore := sqlstore.NewVotes(connectionSource)
	blockStore := sqlstore.NewBlocks(connectionSource)

	blockTime := time.Now()
	block := addTestBlockForTime(t, ctx, blockStore, blockTime)
	party := addTestParty(t, ctx, partyStore, block)

	rand.Seed(time.Now().UnixNano())

	for i := 0; i < 10; i++ {
		blockTime = blockTime.Add(time.Second)
		block = addTestBlockForTime(t, ctx, blockStore, blockTime)
		prop := addTestProposal(t,
			ctx,
			propStore,
			helpers.GenerateID(),
			party,
			helpers.GenerateID(),
			block,
			entities.ProposalStateEnacted,
			entities.ProposalRationale{ProposalRationale: &vega.ProposalRationale{Title: fmt.Sprintf("myurl%02d.com", i+1), Description: "desc"}},
			entities.ProposalTerms{ProposalTerms: &vega.ProposalTerms{Change: &vega.ProposalTerms_NewMarket{}}},
			entities.ProposalErrorUnspecified,
		)

		voteValue := entities.VoteValueYes
		if rand.Intn(100)%2 == 0 {
			voteValue = entities.VoteValueNo
		}

		vote := addTestVote(t, ctx, voteStore, party, prop, voteValue, block, txHashFromString("tx_hash_1"))
		votes = append(votes, vote)
	}

	return voteStore, party, votes
}

func TestVotesCursorPagination(t *testing.T) {
	t.Run("Should return all votes if no pagination is provided", testVotesCursorPaginationNoPagination)
	t.Run("Should return first page of votes if first is provided no after cursor", testVotesCursorPaginationFirstNoAfter)
	t.Run("Should return requested page of votes if first is provided with after cursor", testVotesCursorPaginationFirstWithAfter)
	t.Run("Should return last page of votes if last is provided no before cursor", testVotesCursorPaginationLastNoBefore)
	t.Run("Should return requested page of votes if last is provided with before cursor", testVotesCursorPaginationLastWithBefore)

	t.Run("Should return all votes if no pagination is provided - newest first", testVotesCursorPaginationNoPaginationNewestFirst)
	t.Run("Should return first page of votes if first is provided no after cursor - newest first", testVotesCursorPaginationFirstNoAfterNewestFirst)
	t.Run("Should return requested page of votes if first is provided with after cursor - newest first", testVotesCursorPaginationFirstWithAfterNewestFirst)
	t.Run("Should return last page of votes if last is provided no before cursor - newest first", testVotesCursorPaginationLastNoBeforeNewestFirst)
	t.Run("Should return requested page of votes if last is provided with before cursor - newest first", testVotesCursorPaginationLastWithBeforeNewestFirst)
}

func testVotesCursorPaginationNoPagination(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	vs, party, votes := setupPaginationTestVotes(t, ctx)
	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, false)
	require.NoError(t, err)
	got, pageInfo, err := vs.GetByPartyConnection(ctx, party.ID.String(), pagination)
	require.NoError(t, err)
	require.Equal(t, votes, got)
	require.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     votes[0].Cursor().Encode(),
		EndCursor:       votes[len(votes)-1].Cursor().Encode(),
	}, pageInfo)
}

func testVotesCursorPaginationFirstNoAfter(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	vs, party, votes := setupPaginationTestVotes(t, ctx)
	first := int32(3)
	pagination, err := entities.NewCursorPagination(&first, nil, nil, nil, false)
	require.NoError(t, err)
	got, pageInfo, err := vs.GetByPartyConnection(ctx, party.ID.String(), pagination)
	require.NoError(t, err)
	require.Equal(t, votes[:3], got)
	require.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: false,
		StartCursor:     votes[0].Cursor().Encode(),
		EndCursor:       votes[2].Cursor().Encode(),
	}, pageInfo)
}

func testVotesCursorPaginationFirstWithAfter(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	vs, party, votes := setupPaginationTestVotes(t, ctx)
	first := int32(3)
	after := votes[2].Cursor().Encode()
	pagination, err := entities.NewCursorPagination(&first, &after, nil, nil, false)
	require.NoError(t, err)
	got, pageInfo, err := vs.GetByPartyConnection(ctx, party.ID.String(), pagination)
	require.NoError(t, err)
	require.Equal(t, votes[3:6], got)
	require.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     votes[3].Cursor().Encode(),
		EndCursor:       votes[5].Cursor().Encode(),
	}, pageInfo)
}

func testVotesCursorPaginationLastNoBefore(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	vs, party, votes := setupPaginationTestVotes(t, ctx)
	last := int32(3)
	pagination, err := entities.NewCursorPagination(nil, nil, &last, nil, false)
	require.NoError(t, err)
	got, pageInfo, err := vs.GetByPartyConnection(ctx, party.ID.String(), pagination)
	require.NoError(t, err)
	require.Equal(t, votes[7:], got)
	require.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: true,
		StartCursor:     votes[7].Cursor().Encode(),
		EndCursor:       votes[9].Cursor().Encode(),
	}, pageInfo)
}

func testVotesCursorPaginationLastWithBefore(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	vs, party, votes := setupPaginationTestVotes(t, ctx)
	last := int32(3)
	before := votes[7].Cursor().Encode()
	pagination, err := entities.NewCursorPagination(nil, nil, &last, &before, false)
	require.NoError(t, err)
	got, pageInfo, err := vs.GetByPartyConnection(ctx, party.ID.String(), pagination)
	require.NoError(t, err)
	require.Equal(t, votes[4:7], got)
	require.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     votes[4].Cursor().Encode(),
		EndCursor:       votes[6].Cursor().Encode(),
	}, pageInfo)
}

func testVotesCursorPaginationNoPaginationNewestFirst(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	vs, party, votes := setupPaginationTestVotes(t, ctx)
	votes = entities.ReverseSlice(votes)
	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, true)
	require.NoError(t, err)
	got, pageInfo, err := vs.GetByPartyConnection(ctx, party.ID.String(), pagination)
	require.NoError(t, err)
	require.Equal(t, votes, got)
	require.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     votes[0].Cursor().Encode(),
		EndCursor:       votes[len(votes)-1].Cursor().Encode(),
	}, pageInfo)
}

func testVotesCursorPaginationFirstNoAfterNewestFirst(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	vs, party, votes := setupPaginationTestVotes(t, ctx)
	votes = entities.ReverseSlice(votes)
	first := int32(3)
	pagination, err := entities.NewCursorPagination(&first, nil, nil, nil, true)
	require.NoError(t, err)
	got, pageInfo, err := vs.GetByPartyConnection(ctx, party.ID.String(), pagination)
	require.NoError(t, err)
	require.Equal(t, votes[:3], got)
	require.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: false,
		StartCursor:     votes[0].Cursor().Encode(),
		EndCursor:       votes[2].Cursor().Encode(),
	}, pageInfo)
}

func testVotesCursorPaginationFirstWithAfterNewestFirst(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	vs, party, votes := setupPaginationTestVotes(t, ctx)
	votes = entities.ReverseSlice(votes)
	first := int32(3)
	after := votes[2].Cursor().Encode()
	pagination, err := entities.NewCursorPagination(&first, &after, nil, nil, true)
	require.NoError(t, err)
	got, pageInfo, err := vs.GetByPartyConnection(ctx, party.ID.String(), pagination)
	require.NoError(t, err)
	require.Equal(t, votes[3:6], got)
	require.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     votes[3].Cursor().Encode(),
		EndCursor:       votes[5].Cursor().Encode(),
	}, pageInfo)
}

func testVotesCursorPaginationLastNoBeforeNewestFirst(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	vs, party, votes := setupPaginationTestVotes(t, ctx)
	votes = entities.ReverseSlice(votes)
	last := int32(3)
	pagination, err := entities.NewCursorPagination(nil, nil, &last, nil, true)
	require.NoError(t, err)
	got, pageInfo, err := vs.GetByPartyConnection(ctx, party.ID.String(), pagination)
	require.NoError(t, err)
	require.Equal(t, votes[7:], got)
	require.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: true,
		StartCursor:     votes[7].Cursor().Encode(),
		EndCursor:       votes[9].Cursor().Encode(),
	}, pageInfo)
}

func testVotesCursorPaginationLastWithBeforeNewestFirst(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	vs, party, votes := setupPaginationTestVotes(t, ctx)
	votes = entities.ReverseSlice(votes)
	last := int32(3)
	before := votes[7].Cursor().Encode()
	pagination, err := entities.NewCursorPagination(nil, nil, &last, &before, true)
	require.NoError(t, err)
	got, pageInfo, err := vs.GetByPartyConnection(ctx, party.ID.String(), pagination)
	require.NoError(t, err)
	require.Equal(t, votes[4:7], got)
	require.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     votes[4].Cursor().Encode(),
		EndCursor:       votes[6].Cursor().Encode(),
	}, pageInfo)
}
