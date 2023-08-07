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
	"testing"
	"time"

	dstypes "code.vegaprotocol.io/vega/core/datasource/common"
	datav1 "code.vegaprotocol.io/vega/protos/vega/data/v1"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/sqlstore"
	"code.vegaprotocol.io/vega/datanode/sqlstore/helpers"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/protos/vega"
)

func addTestProposal(
	t *testing.T,
	ctx context.Context,
	ps *sqlstore.Proposals,
	id string,
	party entities.Party,
	reference string,
	block entities.Block,
	state entities.ProposalState,
	rationale entities.ProposalRationale,
	terms entities.ProposalTerms,
	reason entities.ProposalError,
) entities.Proposal {
	t.Helper()
	p := entities.Proposal{
		ID:                      entities.ProposalID(id),
		PartyID:                 party.ID,
		Reference:               reference,
		Terms:                   terms,
		State:                   state,
		VegaTime:                block.VegaTime,
		ProposalTime:            block.VegaTime,
		Rationale:               rationale,
		Reason:                  reason,
		RequiredMajority:        num.MustDecimalFromString("0.5"),
		RequiredParticipation:   num.MustDecimalFromString("0.7"),
		RequiredLPMajority:      nil,
		RequiredLPParticipation: nil,
		TxHash:                  generateTxHash(),
	}
	assert.NoError(t, ps.Add(ctx, p))
	return p
}

func proposalLessThan(x, y entities.Proposal) bool {
	return x.ID.String() < y.ID.String()
}

func assertProposalsMatch(t *testing.T, expected, actual []entities.Proposal) {
	t.Helper()
	sortProposals := cmpopts.SortSlices(proposalLessThan)
	ignoreProtoState := cmpopts.IgnoreUnexported(vega.ProposalTerms{}, vega.ProposalRationale{}, vega.NewMarket{}, vega.NewAsset{})
	assert.Empty(t, cmp.Diff(actual, expected, sortProposals, ignoreProtoState))
}

func assertProposalMatch(t *testing.T, expected, actual entities.Proposal) {
	t.Helper()
	ignoreProtoState := cmpopts.IgnoreUnexported(
		vega.ProposalTerms{},
		vega.ProposalRationale{},
		vega.NewMarket{},
		vega.NewAsset{},
		vega.NewMarketConfiguration{},
		vega.SuccessorConfiguration{},
	)
	assert.Empty(t, cmp.Diff(actual, expected, ignoreProtoState))
}

func TestProposals(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()
	partyStore := sqlstore.NewParties(connectionSource)
	propStore := sqlstore.NewProposals(connectionSource)
	blockStore := sqlstore.NewBlocks(connectionSource)
	block1 := addTestBlock(t, ctx, blockStore)

	party1 := addTestParty(t, ctx, partyStore, block1)
	party2 := addTestParty(t, ctx, partyStore, block1)
	rationale1 := entities.ProposalRationale{ProposalRationale: &vega.ProposalRationale{Title: "myurl1.com", Description: "desc"}}
	rationale2 := entities.ProposalRationale{ProposalRationale: &vega.ProposalRationale{Title: "myurl2.com", Description: "desc"}}
	terms1 := entities.ProposalTerms{ProposalTerms: &vega.ProposalTerms{Change: &vega.ProposalTerms_NewMarket{NewMarket: &vega.NewMarket{}}}}
	terms2 := entities.ProposalTerms{ProposalTerms: &vega.ProposalTerms{Change: &vega.ProposalTerms_NewAsset{NewAsset: &vega.NewAsset{}}}}
	id1 := helpers.GenerateID()
	id2 := helpers.GenerateID()

	reference1 := helpers.GenerateID()
	reference2 := helpers.GenerateID()
	prop1 := addTestProposal(t, ctx, propStore, id1, party1, reference1, block1, entities.ProposalStateEnacted, rationale1, terms1, entities.ProposalErrorUnspecified)
	prop2 := addTestProposal(t, ctx, propStore, id2, party2, reference2, block1, entities.ProposalStateEnacted, rationale2, terms2, entities.ProposalErrorUnspecified)

	party1ID := party1.ID.String()
	prop1ID := prop1.ID.String()
	propType := &entities.ProposalTypeNewMarket

	t.Run("GetById", func(t *testing.T) {
		expected := prop1
		actual, err := propStore.GetByID(ctx, prop1ID)
		require.NoError(t, err)
		assertProposalMatch(t, expected, actual)
	})

	t.Run("GetByTxHash", func(t *testing.T) {
		expected := prop1
		actual, err := propStore.GetByTxHash(ctx, expected.TxHash)
		require.NoError(t, err)
		assertProposalMatch(t, expected, actual[0])

		expected = prop2
		actual, err = propStore.GetByTxHash(ctx, expected.TxHash)
		require.NoError(t, err)
		assertProposalMatch(t, expected, actual[0])
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
		actual, _, err := propStore.Get(ctx, &enacted, nil, nil, entities.CursorPagination{})
		require.NoError(t, err)
		assertProposalsMatch(t, expected, actual)
	})

	t.Run("GetByParty", func(t *testing.T) {
		expected := []entities.Proposal{prop1}
		actual, _, err := propStore.Get(ctx, nil, &party1ID, nil, entities.CursorPagination{})
		require.NoError(t, err)
		assertProposalsMatch(t, expected, actual)
	})

	t.Run("GetByType", func(t *testing.T) {
		expected := []entities.Proposal{prop1}
		actual, _, err := propStore.Get(ctx, nil, nil, propType, entities.CursorPagination{})
		require.NoError(t, err)
		assertProposalsMatch(t, expected, actual)
	})

	t.Run("Add with proposal error", func(t *testing.T) {
		propError := entities.ProposalInvalidPerpetualProduct
		expected := addTestProposal(t, ctx, propStore, helpers.GenerateID(), party1, reference1, block1, entities.ProposalStateEnacted, rationale1, terms1, propError)
		actual, err := propStore.GetByID(ctx, string(expected.ID))
		require.NoError(t, err)
		assert.Equal(t, expected.Reason, actual.Reason)
	})
}

func TestProposalCursorPagination(t *testing.T) {
	t.Run("should return all proposals when no paging is provided", testProposalCursorPaginationNoPagination)
	t.Run("should return only the first page of proposals when first is provided", testProposalCursorPaginationWithFirst)
	t.Run("should return only the requested page of proposals when first and after is provided", testProposalCursorPaginationWithFirstAndAfter)
	t.Run("should return only the last page of proposals when last is provided", testProposalCursorPaginationWithLast)
	t.Run("should return only the requested page of proposals when last and before is provided", testProposalCursorPaginationWithLastAndBefore)

	t.Run("should return all proposals when no paging is provided - newest first", testProposalCursorPaginationNoPaginationNewestFirst)
	t.Run("should return only the first page of proposals when first is provided - newest first", testProposalCursorPaginationWithFirstNewestFirst)
	t.Run("should return only the requested page of proposals when first and after is provided - newest first", testProposalCursorPaginationWithFirstAndAfterNewestFirst)
	t.Run("should return only the last page of proposals when last is provided - newest first", testProposalCursorPaginationWithLastNewestFirst)
	t.Run("should return only the requested page of proposals when last and before is provided - newest first", testProposalCursorPaginationWithLastAndBeforeNewestFirst)

	t.Run("should return all proposals for a given party when no paging is provided", testProposalCursorPaginationNoPaginationByParty)
	t.Run("should return only the first page of proposals for a given party when first is provided", testProposalCursorPaginationWithFirstByParty)
	t.Run("should return only the requested page of proposals for a given party when first and after is provided", testProposalCursorPaginationWithFirstAndAfterByParty)
	t.Run("should return only the last page of proposals for a given party when last is provided", testProposalCursorPaginationWithLastByParty)
	t.Run("should return only the requested page of proposals for a given party when last and before is provided", testProposalCursorPaginationWithLastAndBeforeByParty)

	t.Run("should return all proposals for a given party when no paging is provided - newest first", testProposalCursorPaginationNoPaginationByPartyNewestFirst)
	t.Run("should return only the first page of proposals for a given party when first is provided - newest first", testProposalCursorPaginationWithFirstByPartyNewestFirst)
	t.Run("should return only the requested page of proposals for a given party when first and after is provided - newest first", testProposalCursorPaginationWithFirstAndAfterByPartyNewestFirst)
	t.Run("should return only the last page of proposals for a given party when last is provided - newest first", testProposalCursorPaginationWithLastByPartyNewestFirst)
	t.Run("should return only the requested page of proposals for a given party when last and before is provided - newest first", testProposalCursorPaginationWithLastAndBeforeByPartyNewestFirst)

	t.Run("should return only the open proposals if open state is provided in the filter", testProposalCursorPaginationOpenOnly)
	t.Run("should return the specified proposal state if one is provided", testProposalCursorPaginationGivenState)
}

func testProposalCursorPaginationNoPagination(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	ps := sqlstore.NewProposals(connectionSource)
	proposals, _ := createPaginationTestProposals(t, ctx, ps)
	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, false)
	require.NoError(t, err)

	got, pageInfo, err := ps.Get(ctx, nil, nil, nil, pagination)
	require.NoError(t, err)
	// Proposals should be listed in order of their status, then time, then id
	want := []entities.Proposal{
		proposals[0],
		proposals[10],
		proposals[1],
		proposals[11],
		proposals[2],
		proposals[12],
		proposals[8],
		proposals[18],
		proposals[3],
		proposals[13],
		proposals[4],
		proposals[14],
		proposals[5],
		proposals[15],
		proposals[6],
		proposals[16],
		proposals[7],
		proposals[17],
		proposals[9],
		proposals[19],
	}
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     proposals[0].Cursor().Encode(),
		EndCursor:       proposals[19].Cursor().Encode(),
	}, pageInfo)
}

func testProposalCursorPaginationWithFirst(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	ps := sqlstore.NewProposals(connectionSource)
	proposals, _ := createPaginationTestProposals(t, ctx, ps)
	first := int32(3)
	pagination, err := entities.NewCursorPagination(&first, nil, nil, nil, false)
	require.NoError(t, err)

	got, pageInfo, err := ps.Get(ctx, nil, nil, nil, pagination)
	require.NoError(t, err)
	// Proposals should be listed in order of their status, then time, then id
	want := []entities.Proposal{
		proposals[0],
		proposals[10],
		proposals[1],
	}
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: false,
		StartCursor:     proposals[0].Cursor().Encode(),
		EndCursor:       proposals[1].Cursor().Encode(),
	}, pageInfo)
}

func testProposalCursorPaginationWithFirstAndAfter(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	ps := sqlstore.NewProposals(connectionSource)
	proposals, _ := createPaginationTestProposals(t, ctx, ps)
	first := int32(8)
	after := proposals[1].Cursor().Encode()
	pagination, err := entities.NewCursorPagination(&first, &after, nil, nil, false)
	require.NoError(t, err)

	got, pageInfo, err := ps.Get(ctx, nil, nil, nil, pagination)
	require.NoError(t, err)
	// Proposals should be listed in order of their status, then time, then id
	want := []entities.Proposal{
		proposals[11],
		proposals[2],
		proposals[12],
		proposals[8],
		proposals[18],
		proposals[3],
		proposals[13],
		proposals[4],
	}
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     proposals[11].Cursor().Encode(),
		EndCursor:       proposals[4].Cursor().Encode(),
	}, pageInfo)
}

func testProposalCursorPaginationWithLast(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	ps := sqlstore.NewProposals(connectionSource)
	proposals, _ := createPaginationTestProposals(t, ctx, ps)
	last := int32(3)
	pagination, err := entities.NewCursorPagination(nil, nil, &last, nil, false)
	require.NoError(t, err)

	got, pageInfo, err := ps.Get(ctx, nil, nil, nil, pagination)
	require.NoError(t, err)
	// Proposals should be listed in order of their status, then time, then id
	want := []entities.Proposal{
		proposals[17],
		proposals[9],
		proposals[19],
	}
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: true,
		StartCursor:     proposals[17].Cursor().Encode(),
		EndCursor:       proposals[19].Cursor().Encode(),
	}, pageInfo)
}

func testProposalCursorPaginationWithLastAndBefore(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	ps := sqlstore.NewProposals(connectionSource)
	proposals, _ := createPaginationTestProposals(t, ctx, ps)
	last := int32(8)
	before := proposals[5].Cursor().Encode()
	pagination, err := entities.NewCursorPagination(nil, nil, &last, &before, false)
	require.NoError(t, err)

	got, pageInfo, err := ps.Get(ctx, nil, nil, nil, pagination)
	require.NoError(t, err)
	// Proposals should be listed in order of their status, then time, then id
	want := []entities.Proposal{
		proposals[2],
		proposals[12],
		proposals[8],
		proposals[18],
		proposals[3],
		proposals[13],
		proposals[4],
		proposals[14],
	}
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     proposals[2].Cursor().Encode(),
		EndCursor:       proposals[14].Cursor().Encode(),
	}, pageInfo)
}

func testProposalCursorPaginationNoPaginationNewestFirst(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	ps := sqlstore.NewProposals(connectionSource)
	proposals, _ := createPaginationTestProposals(t, ctx, ps)
	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, true)
	require.NoError(t, err)

	got, pageInfo, err := ps.Get(ctx, nil, nil, nil, pagination)
	require.NoError(t, err)
	// Proposals should be listed in order of their status, then time, then id
	want := []entities.Proposal{
		proposals[18],
		proposals[8],
		proposals[12],
		proposals[2],
		proposals[11],
		proposals[1],
		proposals[10],
		proposals[0],
		proposals[19],
		proposals[9],
		proposals[17],
		proposals[7],
		proposals[16],
		proposals[6],
		proposals[15],
		proposals[5],
		proposals[14],
		proposals[4],
		proposals[13],
		proposals[3],
	}
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     proposals[18].Cursor().Encode(),
		EndCursor:       proposals[3].Cursor().Encode(),
	}, pageInfo)
}

func testProposalCursorPaginationWithFirstNewestFirst(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	ps := sqlstore.NewProposals(connectionSource)
	proposals, _ := createPaginationTestProposals(t, ctx, ps)
	first := int32(3)
	pagination, err := entities.NewCursorPagination(&first, nil, nil, nil, true)
	require.NoError(t, err)

	got, pageInfo, err := ps.Get(ctx, nil, nil, nil, pagination)
	require.NoError(t, err)
	// Proposals should be listed in order of their status, then time, then id
	want := []entities.Proposal{
		proposals[18],
		proposals[8],
		proposals[12],
	}
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: false,
		StartCursor:     proposals[18].Cursor().Encode(),
		EndCursor:       proposals[12].Cursor().Encode(),
	}, pageInfo)
}

func testProposalCursorPaginationWithFirstAndAfterNewestFirst(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	ps := sqlstore.NewProposals(connectionSource)
	proposals, _ := createPaginationTestProposals(t, ctx, ps)
	first := int32(8)
	after := proposals[12].Cursor().Encode()
	pagination, err := entities.NewCursorPagination(&first, &after, nil, nil, true)
	require.NoError(t, err)

	got, pageInfo, err := ps.Get(ctx, nil, nil, nil, pagination)
	require.NoError(t, err)
	// Proposals should be listed in order of their status, then time, then id
	want := []entities.Proposal{
		proposals[2],
		proposals[11],
		proposals[1],
		proposals[10],
		proposals[0],
		proposals[19],
		proposals[9],
		proposals[17],
	}
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     proposals[2].Cursor().Encode(),
		EndCursor:       proposals[17].Cursor().Encode(),
	}, pageInfo)
}

func testProposalCursorPaginationWithLastNewestFirst(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	ps := sqlstore.NewProposals(connectionSource)
	proposals, _ := createPaginationTestProposals(t, ctx, ps)
	last := int32(3)
	pagination, err := entities.NewCursorPagination(nil, nil, &last, nil, true)
	require.NoError(t, err)

	got, pageInfo, err := ps.Get(ctx, nil, nil, nil, pagination)
	require.NoError(t, err)
	// Proposals should be listed in order of their status, then time, then id
	want := []entities.Proposal{
		proposals[4],
		proposals[13],
		proposals[3],
	}
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: true,
		StartCursor:     proposals[4].Cursor().Encode(),
		EndCursor:       proposals[3].Cursor().Encode(),
	}, pageInfo)
}

func testProposalCursorPaginationWithLastAndBeforeNewestFirst(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	ps := sqlstore.NewProposals(connectionSource)
	proposals, _ := createPaginationTestProposals(t, ctx, ps)
	last := int32(8)
	before := proposals[16].Cursor().Encode()
	pagination, err := entities.NewCursorPagination(nil, nil, &last, &before, true)
	require.NoError(t, err)

	got, pageInfo, err := ps.Get(ctx, nil, nil, nil, pagination)
	require.NoError(t, err)
	// Proposals should be listed in order of their status, then time, then id
	want := []entities.Proposal{
		proposals[11],
		proposals[1],
		proposals[10],
		proposals[0],
		proposals[19],
		proposals[9],
		proposals[17],
		proposals[7],
	}
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     proposals[11].Cursor().Encode(),
		EndCursor:       proposals[7].Cursor().Encode(),
	}, pageInfo)
}

func testProposalCursorPaginationNoPaginationByParty(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	ps := sqlstore.NewProposals(connectionSource)
	proposals, parties := createPaginationTestProposals(t, ctx, ps)
	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, false)
	require.NoError(t, err)

	partyID := parties[0].ID.String()
	got, pageInfo, err := ps.Get(ctx, nil, &partyID, nil, pagination)
	require.NoError(t, err)
	// Proposals should be listed in order of their status, then time, then id
	want := []entities.Proposal{
		proposals[0],
		proposals[1],
		proposals[2],
		proposals[8],
		proposals[3],
		proposals[4],
		proposals[5],
		proposals[6],
		proposals[7],
		proposals[9],
	}
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     proposals[0].Cursor().Encode(),
		EndCursor:       proposals[9].Cursor().Encode(),
	}, pageInfo)
}

func testProposalCursorPaginationWithFirstByParty(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	ps := sqlstore.NewProposals(connectionSource)
	proposals, parties := createPaginationTestProposals(t, ctx, ps)
	first := int32(3)
	pagination, err := entities.NewCursorPagination(&first, nil, nil, nil, false)
	require.NoError(t, err)

	partyID := parties[0].ID.String()
	got, pageInfo, err := ps.Get(ctx, nil, &partyID, nil, pagination)
	require.NoError(t, err)
	// Proposals should be listed in order of their status, then time, then id
	want := []entities.Proposal{
		proposals[0],
		proposals[1],
		proposals[2],
	}
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: false,
		StartCursor:     proposals[0].Cursor().Encode(),
		EndCursor:       proposals[2].Cursor().Encode(),
	}, pageInfo)
}

func testProposalCursorPaginationWithFirstAndAfterByParty(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	ps := sqlstore.NewProposals(connectionSource)
	proposals, parties := createPaginationTestProposals(t, ctx, ps)
	first := int32(3)
	after := proposals[2].Cursor().Encode()
	pagination, err := entities.NewCursorPagination(&first, &after, nil, nil, false)
	require.NoError(t, err)

	partyID := parties[0].ID.String()
	got, pageInfo, err := ps.Get(ctx, nil, &partyID, nil, pagination)
	require.NoError(t, err)
	// Proposals should be listed in order of their status, then time, then id
	want := []entities.Proposal{
		proposals[8],
		proposals[3],
		proposals[4],
	}
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     proposals[8].Cursor().Encode(),
		EndCursor:       proposals[4].Cursor().Encode(),
	}, pageInfo)
}

func testProposalCursorPaginationWithLastByParty(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	ps := sqlstore.NewProposals(connectionSource)
	proposals, parties := createPaginationTestProposals(t, ctx, ps)
	last := int32(3)
	pagination, err := entities.NewCursorPagination(nil, nil, &last, nil, false)
	require.NoError(t, err)

	partyID := parties[0].ID.String()
	got, pageInfo, err := ps.Get(ctx, nil, &partyID, nil, pagination)
	require.NoError(t, err)
	// Proposals should be listed in order of their status, then time, then id
	want := []entities.Proposal{
		proposals[6],
		proposals[7],
		proposals[9],
	}
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: true,
		StartCursor:     proposals[6].Cursor().Encode(),
		EndCursor:       proposals[9].Cursor().Encode(),
	}, pageInfo)
}

func testProposalCursorPaginationWithLastAndBeforeByParty(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	ps := sqlstore.NewProposals(connectionSource)
	proposals, parties := createPaginationTestProposals(t, ctx, ps)
	last := int32(5)
	before := proposals[6].Cursor().Encode()
	pagination, err := entities.NewCursorPagination(nil, nil, &last, &before, false)
	require.NoError(t, err)

	partyID := parties[0].ID.String()
	got, pageInfo, err := ps.Get(ctx, nil, &partyID, nil, pagination)
	require.NoError(t, err)
	// Proposals should be listed in order of their status, then time, then id
	want := []entities.Proposal{
		proposals[2],
		proposals[8],
		proposals[3],
		proposals[4],
		proposals[5],
	}
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     proposals[2].Cursor().Encode(),
		EndCursor:       proposals[5].Cursor().Encode(),
	}, pageInfo)
}

func testProposalCursorPaginationNoPaginationByPartyNewestFirst(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	ps := sqlstore.NewProposals(connectionSource)
	proposals, parties := createPaginationTestProposals(t, ctx, ps)
	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, true)
	require.NoError(t, err)

	partyID := parties[0].ID.String()

	got, pageInfo, err := ps.Get(ctx, nil, &partyID, nil, pagination)
	require.NoError(t, err)
	// Proposals should be listed in order of their status, then time, then id
	want := []entities.Proposal{
		proposals[8],
		proposals[2],
		proposals[1],
		proposals[0],
		proposals[9],
		proposals[7],
		proposals[6],
		proposals[5],
		proposals[4],
		proposals[3],
	}
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     proposals[8].Cursor().Encode(),
		EndCursor:       proposals[3].Cursor().Encode(),
	}, pageInfo)
}

func testProposalCursorPaginationWithFirstByPartyNewestFirst(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	ps := sqlstore.NewProposals(connectionSource)
	proposals, parties := createPaginationTestProposals(t, ctx, ps)
	first := int32(3)
	pagination, err := entities.NewCursorPagination(&first, nil, nil, nil, true)
	require.NoError(t, err)

	partyID := parties[0].ID.String()

	got, pageInfo, err := ps.Get(ctx, nil, &partyID, nil, pagination)
	require.NoError(t, err)
	// Proposals should be listed in order of their status, then time, then id
	want := []entities.Proposal{
		proposals[8],
		proposals[2],
		proposals[1],
	}
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: false,
		StartCursor:     proposals[8].Cursor().Encode(),
		EndCursor:       proposals[1].Cursor().Encode(),
	}, pageInfo)
}

func testProposalCursorPaginationWithFirstAndAfterByPartyNewestFirst(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	ps := sqlstore.NewProposals(connectionSource)
	proposals, parties := createPaginationTestProposals(t, ctx, ps)
	first := int32(3)
	after := proposals[1].Cursor().Encode()
	pagination, err := entities.NewCursorPagination(&first, &after, nil, nil, true)
	require.NoError(t, err)

	partyID := parties[0].ID.String()

	got, pageInfo, err := ps.Get(ctx, nil, &partyID, nil, pagination)
	require.NoError(t, err)
	// Proposals should be listed in order of their status, then time, then id
	want := []entities.Proposal{
		proposals[0],
		proposals[9],
		proposals[7],
	}
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     proposals[0].Cursor().Encode(),
		EndCursor:       proposals[7].Cursor().Encode(),
	}, pageInfo)
}

func testProposalCursorPaginationWithLastByPartyNewestFirst(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	ps := sqlstore.NewProposals(connectionSource)
	proposals, parties := createPaginationTestProposals(t, ctx, ps)
	last := int32(3)
	pagination, err := entities.NewCursorPagination(nil, nil, &last, nil, true)
	require.NoError(t, err)

	partyID := parties[0].ID.String()

	got, pageInfo, err := ps.Get(ctx, nil, &partyID, nil, pagination)
	require.NoError(t, err)
	// Proposals should be listed in order of their status, then time, then id
	want := []entities.Proposal{
		proposals[5],
		proposals[4],
		proposals[3],
	}
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: true,
		StartCursor:     proposals[5].Cursor().Encode(),
		EndCursor:       proposals[3].Cursor().Encode(),
	}, pageInfo)
}

func testProposalCursorPaginationWithLastAndBeforeByPartyNewestFirst(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	ps := sqlstore.NewProposals(connectionSource)
	proposals, parties := createPaginationTestProposals(t, ctx, ps)
	last := int32(5)
	before := proposals[5].Cursor().Encode()
	pagination, err := entities.NewCursorPagination(nil, nil, &last, &before, true)
	require.NoError(t, err)

	partyID := parties[0].ID.String()

	got, pageInfo, err := ps.Get(ctx, nil, &partyID, nil, pagination)
	require.NoError(t, err)
	// Proposals should be listed in order of their status, then time, then id
	want := []entities.Proposal{
		proposals[1],
		proposals[0],
		proposals[9],
		proposals[7],
		proposals[6],
	}
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     proposals[1].Cursor().Encode(),
		EndCursor:       proposals[6].Cursor().Encode(),
	}, pageInfo)
}

func testProposalCursorPaginationOpenOnly(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	ps := sqlstore.NewProposals(connectionSource)
	proposals, _ := createPaginationTestProposals(t, ctx, ps)
	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, false)
	require.NoError(t, err)

	state := entities.ProposalStateOpen
	got, pageInfo, err := ps.Get(ctx, &state, nil, nil, pagination)
	require.NoError(t, err)
	// Proposals should be listed in order of their status, then time, then id
	want := []entities.Proposal{
		proposals[0],
		proposals[10],
		proposals[1],
		proposals[11],
		proposals[2],
		proposals[12],
		proposals[8],
		proposals[18],
	}
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     proposals[0].Cursor().Encode(),
		EndCursor:       proposals[18].Cursor().Encode(),
	}, pageInfo)
}

func testProposalCursorPaginationGivenState(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	ps := sqlstore.NewProposals(connectionSource)
	proposals, _ := createPaginationTestProposals(t, ctx, ps)
	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, false)
	require.NoError(t, err)

	t.Run("State is Enacted", func(t *testing.T) {
		state := entities.ProposalStateEnacted
		got, pageInfo, err := ps.Get(ctx, &state, nil, nil, pagination)
		require.NoError(t, err)
		// Proposals should be listed in order of their status, then time, then id
		want := []entities.Proposal{
			proposals[3],
			proposals[13],
			proposals[6],
			proposals[16],
			proposals[9],
			proposals[19],
		}
		assert.Equal(t, want, got)
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     false,
			HasPreviousPage: false,
			StartCursor:     proposals[3].Cursor().Encode(),
			EndCursor:       proposals[19].Cursor().Encode(),
		}, pageInfo)
	})

	t.Run("State is Passed", func(t *testing.T) {
		state := entities.ProposalStatePassed
		got, pageInfo, err := ps.Get(ctx, &state, nil, nil, pagination)
		require.NoError(t, err)
		// Proposals should be listed in order of their status, then time, then id
		want := []entities.Proposal{
			proposals[4],
			proposals[14],
			proposals[5],
			proposals[15],
		}
		assert.Equal(t, want, got)
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     false,
			HasPreviousPage: false,
			StartCursor:     proposals[4].Cursor().Encode(),
			EndCursor:       proposals[15].Cursor().Encode(),
		}, pageInfo)
	})
}

func createPaginationTestProposals(t *testing.T, ctx context.Context, pps *sqlstore.Proposals) ([]entities.Proposal, []entities.Party) {
	t.Helper()
	ps := sqlstore.NewParties(connectionSource)
	bs := sqlstore.NewBlocks(connectionSource)

	proposals := make([]entities.Proposal, 20)

	blockTime := time.Date(2022, 7, 15, 8, 0, 0, 0, time.Local)
	block := addTestBlockForTime(t, ctx, bs, blockTime)

	parties := []entities.Party{
		addTestParty(t, ctx, ps, block),
		addTestParty(t, ctx, ps, block),
	}

	states := []entities.ProposalState{
		entities.ProposalStateOpen,
		entities.ProposalStateOpen,
		entities.ProposalStateOpen,
		entities.ProposalStateEnacted,
		entities.ProposalStatePassed,
		entities.ProposalStatePassed,
		entities.ProposalStateEnacted,
		entities.ProposalStateDeclined,
		entities.ProposalStateOpen,
		entities.ProposalStateEnacted,
	}
	i := 0
	for i < 10 {
		blockTime = blockTime.Add(time.Minute)
		block = addTestBlockForTime(t, ctx, bs, blockTime)
		block2 := addTestBlockForTime(t, ctx, bs, blockTime.Add(time.Second*30))

		id1 := fmt.Sprintf("deadbeef%02d", i)
		id2 := fmt.Sprintf("deadbeef%02d", i+10)

		ref1 := fmt.Sprintf("cafed00d%02d", i)
		ref2 := fmt.Sprintf("cafed00d%02d", i+10)
		rationale1 := entities.ProposalRationale{ProposalRationale: &vega.ProposalRationale{Title: fmt.Sprintf("https://rationale1-%02d.com", i), Description: "desc"}}
		rationale2 := entities.ProposalRationale{ProposalRationale: &vega.ProposalRationale{Title: fmt.Sprintf("https://rationale1-%02d.com", i+10), Description: "desc"}}
		terms1 := entities.ProposalTerms{ProposalTerms: &vega.ProposalTerms{Change: &vega.ProposalTerms_NewMarket{NewMarket: &vega.NewMarket{}}}}
		terms2 := entities.ProposalTerms{ProposalTerms: &vega.ProposalTerms{Change: &vega.ProposalTerms_NewAsset{NewAsset: &vega.NewAsset{}}}}

		proposals[i] = addTestProposal(t, ctx, pps, id1, parties[0], ref1, block, states[i], rationale1, terms1, entities.ProposalErrorUnspecified)
		proposals[i+10] = addTestProposal(t, ctx, pps, id2, parties[1], ref2, block2, states[i], rationale2, terms2, entities.ProposalErrorUnspecified)
		i++
	}

	return proposals, parties
}

func TestProposeSuccessorMarket(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	partyStore := sqlstore.NewParties(connectionSource)
	propStore := sqlstore.NewProposals(connectionSource)
	blockStore := sqlstore.NewBlocks(connectionSource)
	block1 := addTestBlock(t, ctx, blockStore)

	party1 := addTestParty(t, ctx, partyStore, block1)
	rationale1 := entities.ProposalRationale{ProposalRationale: &vega.ProposalRationale{Title: "myurl1.com", Description: "desc"}}
	rationale2 := entities.ProposalRationale{ProposalRationale: &vega.ProposalRationale{Title: "myurl2.com", Description: "desc"}}
	terms1 := entities.ProposalTerms{ProposalTerms: &vega.ProposalTerms{Change: &vega.ProposalTerms_NewMarket{NewMarket: &vega.NewMarket{
		Changes: &vega.NewMarketConfiguration{
			Instrument:                    nil,
			DecimalPlaces:                 0,
			Metadata:                      nil,
			PriceMonitoringParameters:     nil,
			LiquidityMonitoringParameters: nil,
			RiskParameters:                nil,
			PositionDecimalPlaces:         0,
			LpPriceRange:                  "",
			LinearSlippageFactor:          "",
			QuadraticSlippageFactor:       "",
			Successor:                     nil,
		},
	}}}}
	terms2 := entities.ProposalTerms{ProposalTerms: &vega.ProposalTerms{Change: &vega.ProposalTerms_NewMarket{NewMarket: &vega.NewMarket{
		Changes: &vega.NewMarketConfiguration{
			Instrument:                    nil,
			DecimalPlaces:                 0,
			Metadata:                      nil,
			PriceMonitoringParameters:     nil,
			LiquidityMonitoringParameters: nil,
			RiskParameters:                nil,
			PositionDecimalPlaces:         0,
			LpPriceRange:                  "",
			LinearSlippageFactor:          "",
			QuadraticSlippageFactor:       "",
			Successor: &vega.SuccessorConfiguration{
				ParentMarketId:        "deadbeef",
				InsurancePoolFraction: "0.5",
			},
		},
	}}}}
	id1 := helpers.GenerateID()
	id2 := helpers.GenerateID()

	reference1 := helpers.GenerateID()
	reference2 := helpers.GenerateID()
	prop1 := addTestProposal(t, ctx, propStore, id1, party1, reference1, block1, entities.ProposalStateEnacted, rationale1, terms1, entities.ProposalErrorUnspecified)
	prop2 := addTestProposal(t, ctx, propStore, id2, party1, reference2, block1, entities.ProposalStateRejected, rationale2, terms2, entities.ProposalErrorInvalidSuccessorMarket)

	t.Run("GetByID", func(t *testing.T) {
		want := prop1
		got, err := propStore.GetByID(ctx, prop1.ID.String())
		require.NoError(t, err)
		assertProposalMatch(t, want, got)

		want = prop2
		got, err = propStore.GetByID(ctx, prop2.ID.String())
		require.NoError(t, err)
		assertProposalMatch(t, want, got)
	})
}

func getNewProposal(partyID string) *vega.Proposal {
	return &vega.Proposal{
		Id:        helpers.GenerateID(),
		Reference: helpers.GenerateID(),
		PartyId:   partyID,
		State:     vega.Proposal_STATE_OPEN,
		Timestamp: time.Now().UnixNano(),
		Rationale: &vega.ProposalRationale{Title: "myurl1.com", Description: "desc"},
		Terms: &vega.ProposalTerms{
			Change: &vega.ProposalTerms_NewMarket{
				NewMarket: &vega.NewMarket{
					Changes: &vega.NewMarketConfiguration{
						Instrument: &vega.InstrumentConfiguration{},
					},
				},
			},
		},
	}
}

func getNewSpotMarketProposal(partyID string) *vega.Proposal {
	proposal := getNewProposal(partyID)

	proposal.Terms.Change = &vega.ProposalTerms_NewSpotMarket{
		NewSpotMarket: &vega.NewSpotMarket{
			Changes: &vega.NewSpotMarketConfiguration{
				Instrument: &vega.InstrumentConfiguration{
					Product: &vega.InstrumentConfiguration_Spot{
						Spot: &vega.SpotProduct{
							BaseAsset:  "USD",
							QuoteAsset: "ETH",
							Name:       "ETH/USD",
						},
					},
				},
			},
		},
	}
	return proposal
}

func getSpotMarketUpdateProposal(partyID string) *vega.Proposal {
	proposal := getNewProposal(partyID)
	proposal.Terms.Change = &vega.ProposalTerms_UpdateSpotMarket{
		UpdateSpotMarket: &vega.UpdateSpotMarket{
			MarketId: "USD/ETH",
			Changes: &vega.UpdateSpotMarketConfiguration{
				Metadata: []string{"ETH", "USD"},
				PriceMonitoringParameters: &vega.PriceMonitoringParameters{
					Triggers: []*vega.PriceMonitoringTrigger{
						{
							Horizon:          1,
							Probability:      "0.5",
							AuctionExtension: 0,
						},
					},
				},
				TargetStakeParameters: &vega.TargetStakeParameters{
					TimeWindow:    1,
					ScalingFactor: 1,
				},
				RiskParameters: &vega.UpdateSpotMarketConfiguration_Simple{
					Simple: &vega.SimpleModelParams{
						FactorLong:           1,
						FactorShort:          1,
						MaxMoveUp:            1,
						MinMoveDown:          1,
						ProbabilityOfTrading: 1,
					},
				},
				SlaParams: &vega.LiquiditySLAParameters{
					PriceRange:                      "",
					CommitmentMinTimeFraction:       "0.5",
					ProvidersFeeCalculationTimeStep: 1,
					PerformanceHysteresisEpochs:     2,
					SlaCompetitionFactor:            "0.75",
				},
			},
		},
	}
	return proposal
}

func getNewPerpetualMarketProposal(partyID string) *vega.Proposal {
	pk := dstypes.CreateSignerFromString("0xDEADBEEF", dstypes.SignerTypePubKey)
	proposal := getNewProposal(partyID)

	proposal.Terms.Change = &vega.ProposalTerms_NewMarket{
		NewMarket: &vega.NewMarket{
			Changes: &vega.NewMarketConfiguration{
				Instrument: &vega.InstrumentConfiguration{
					Product: &vega.InstrumentConfiguration_Perpetual{
						Perpetual: &vega.PerpetualProduct{
							SettlementAsset:     "Ethereum/Ether",
							QuoteName:           "ETH-230929",
							MarginFundingFactor: "0.5",
							InterestRate:        "0.0125",
							ClampLowerBound:     "0.2",
							ClampUpperBound:     "0.8",
							DataSourceSpecForSettlementSchedule: &vega.DataSourceDefinition{
								SourceType: &vega.DataSourceDefinition_External{
									External: &vega.DataSourceDefinitionExternal{
										SourceType: &vega.DataSourceDefinitionExternal_Oracle{
											Oracle: &vega.DataSourceSpecConfiguration{
												Signers: []*datav1.Signer{pk.IntoProto()},
												Filters: []*datav1.Filter{
													{
														Key: &datav1.PropertyKey{
															Name: "prices.ETH.value",
															Type: datav1.PropertyKey_TYPE_INTEGER,
														},
														Conditions: []*datav1.Condition{},
													},
												},
											},
										},
									},
								},
							},
							DataSourceSpecForSettlementData: &vega.DataSourceDefinition{
								SourceType: &vega.DataSourceDefinition_Internal{
									Internal: &vega.DataSourceDefinitionInternal{
										SourceType: &vega.DataSourceDefinitionInternal_Time{
											Time: &vega.DataSourceSpecConfigurationTime{
												Conditions: []*datav1.Condition{
													{
														Operator: datav1.Condition_OPERATOR_GREATER_THAN_OR_EQUAL,
														Value:    "2023-09-29T00:00:00.000000000Z",
													},
												},
											},
										},
									},
								},
							},
							DataSourceSpecBinding: &vega.DataSourceSpecToPerpetualBinding{
								SettlementDataProperty:     "prices.ETH.value",
								SettlementScheduleProperty: "2023-09-29T00:00:00.000000000Z",
							},
						},
					},
				},
			},
		},
	}
	return proposal
}

func getPerpetualMarketUpdateProposal(partyID string) *vega.Proposal {
	pk := dstypes.CreateSignerFromString("0xDEADBEEF", dstypes.SignerTypePubKey)
	proposal := getNewProposal(partyID)

	proposal.Terms.Change = &vega.ProposalTerms_UpdateMarket{
		UpdateMarket: &vega.UpdateMarket{
			Changes: &vega.UpdateMarketConfiguration{
				Instrument: &vega.UpdateInstrumentConfiguration{
					Product: &vega.UpdateInstrumentConfiguration_Perpetual{
						Perpetual: &vega.UpdatePerpetualProduct{
							QuoteName:           "ETH-230929",
							MarginFundingFactor: "0.6",
							InterestRate:        "0.015",
							ClampLowerBound:     "0.1",
							ClampUpperBound:     "0.9",
							DataSourceSpecForSettlementSchedule: &vega.DataSourceDefinition{
								SourceType: &vega.DataSourceDefinition_External{
									External: &vega.DataSourceDefinitionExternal{
										SourceType: &vega.DataSourceDefinitionExternal_Oracle{
											Oracle: &vega.DataSourceSpecConfiguration{
												Signers: []*datav1.Signer{pk.IntoProto()},
												Filters: []*datav1.Filter{
													{
														Key: &datav1.PropertyKey{
															Name: "prices.ETH.value",
															Type: datav1.PropertyKey_TYPE_INTEGER,
														},
														Conditions: []*datav1.Condition{},
													},
												},
											},
										},
									},
								},
							},
							DataSourceSpecForSettlementData: &vega.DataSourceDefinition{
								SourceType: &vega.DataSourceDefinition_Internal{
									Internal: &vega.DataSourceDefinitionInternal{
										SourceType: &vega.DataSourceDefinitionInternal_Time{
											Time: &vega.DataSourceSpecConfigurationTime{
												Conditions: []*datav1.Condition{
													{
														Operator: datav1.Condition_OPERATOR_GREATER_THAN_OR_EQUAL,
														Value:    "2023-09-29T00:00:00.000000000Z",
													},
												},
											},
										},
									},
								},
							},
							DataSourceSpecBinding: &vega.DataSourceSpecToPerpetualBinding{
								SettlementDataProperty:     "prices.ETH.value",
								SettlementScheduleProperty: "2023-09-29T00:00:00.000000000Z",
							},
						},
					},
				},
			},
		},
	}

	return proposal
}

func setupProposalTest(t *testing.T) (*sqlstore.Blocks, *sqlstore.Parties, *sqlstore.Proposals) {
	t.Helper()
	partyStore := sqlstore.NewParties(connectionSource)
	propStore := sqlstore.NewProposals(connectionSource)
	blockStore := sqlstore.NewBlocks(connectionSource)

	return blockStore, partyStore, propStore
}

func TestSpotMarketProposal(t *testing.T) {
	t.Run("Should save and retrieve new spot market proposals to the store", testShouldSaveNewSpotMarketProposalsToStore)
	t.Run("Should save amd retrieve update spot market proposals to the store", testShouldSaveUpdateSpotMarketProposalsToStore)
}

func testShouldSaveNewSpotMarketProposalsToStore(t *testing.T) {
	bs, pts, ps := setupProposalTest(t)

	ctx, rollback := tempTransaction(t)
	defer rollback()

	block1 := addTestBlock(t, ctx, bs)
	party1 := addTestParty(t, ctx, pts, block1)

	proposalProto := getNewSpotMarketProposal(party1.ID.String())
	proposal, err := entities.ProposalFromProto(proposalProto, generateTxHash())
	require.NoError(t, err)
	t.Run("Add should save the spot market proposal to the database", func(t *testing.T) {
		err = ps.Add(ctx, proposal)
		require.NoError(t, err)
	})

	var savedProp []entities.Proposal
	t.Run("Get should return the saved spot market proposal", func(t *testing.T) {
		savedProp, _, err = ps.Get(ctx, nil, nil, nil, entities.CursorPagination{})
		require.NoError(t, err)
		require.Len(t, savedProp, 1)
	})

	t.Run("Proposal terms should be for a new spot market", func(t *testing.T) {
		savedToProto := savedProp[0].ToProto()
		assert.Nil(t, savedToProto.Terms.GetUpdateSpotMarket())
		assert.NotNil(t, savedToProto.Terms.GetNewSpotMarket())
	})
}

func testShouldSaveUpdateSpotMarketProposalsToStore(t *testing.T) {
	bs, pts, ps := setupProposalTest(t)

	ctx, rollback := tempTransaction(t)
	defer rollback()

	block1 := addTestBlock(t, ctx, bs)
	party1 := addTestParty(t, ctx, pts, block1)

	proposalProto := getSpotMarketUpdateProposal(party1.ID.String())
	proposal, err := entities.ProposalFromProto(proposalProto, generateTxHash())
	require.NoError(t, err)
	t.Run("Add should save the spot market proposal to the database", func(t *testing.T) {
		err = ps.Add(ctx, proposal)
		require.NoError(t, err)
	})

	var savedProp []entities.Proposal
	t.Run("Get should return the saved spot market proposal", func(t *testing.T) {
		savedProp, _, err = ps.Get(ctx, nil, nil, nil, entities.CursorPagination{})
		require.NoError(t, err)
		require.Len(t, savedProp, 1)
	})

	t.Run("Proposal terms should be for a new spot market", func(t *testing.T) {
		savedToProto := savedProp[0].ToProto()
		assert.Nil(t, savedToProto.Terms.GetUpdateMarket())
		assert.NotNil(t, savedToProto.Terms.GetUpdateSpotMarket())
	})
}

func TestPerpetualMarketProposal(t *testing.T) {
	t.Run("Should save and retrieve new perpetual market proposals to the store", testShouldSaveNewPerpetualMarketProposalsToStore)
	t.Run("Should save and retrieve update perpetual market proposals to the store", testShouldUpdateSavePerpetualMarketProposalsToStore)
}

func testShouldSaveNewPerpetualMarketProposalsToStore(t *testing.T) {
	bs, pts, ps := setupProposalTest(t)

	ctx, rollback := tempTransaction(t)
	defer rollback()

	block1 := addTestBlock(t, ctx, bs)
	party1 := addTestParty(t, ctx, pts, block1)

	proposalProto := getNewPerpetualMarketProposal(party1.ID.String())
	proposal, err := entities.ProposalFromProto(proposalProto, generateTxHash())
	require.NoError(t, err)
	t.Run("Add should create a new perps market proposal  in the database", func(t *testing.T) {
		err = ps.Add(ctx, proposal)
		require.NoError(t, err)
	})

	var savedProp []entities.Proposal
	t.Run("Get should return the perps market proposal from the database", func(t *testing.T) {
		savedProp, _, err = ps.Get(ctx, nil, nil, nil, entities.CursorPagination{})
		require.NoError(t, err)
		require.Len(t, savedProp, 1)
	})

	t.Run("The saved proposal's instrument should be a perp and not a future or spot", func(t *testing.T) {
		savedToProto := savedProp[0].ToProto()
		assert.Nil(t, savedToProto.Terms.GetNewSpotMarket())
		assert.NotNil(t, savedToProto.Terms.GetNewMarket())
		future := savedToProto.Terms.GetNewMarket().GetChanges().GetInstrument().GetFuture()
		assert.Nil(t, future)
		perps := savedToProto.Terms.GetNewMarket().GetChanges().GetInstrument().GetPerpetual()
		assert.NotNil(t, perps)
	})
}

func testShouldUpdateSavePerpetualMarketProposalsToStore(t *testing.T) {
	bs, pts, ps := setupProposalTest(t)

	ctx, rollback := tempTransaction(t)
	defer rollback()

	block1 := addTestBlock(t, ctx, bs)
	party1 := addTestParty(t, ctx, pts, block1)

	proposalProto := getPerpetualMarketUpdateProposal(party1.ID.String())
	proposal, err := entities.ProposalFromProto(proposalProto, generateTxHash())
	require.NoError(t, err)
	t.Run("Add should create a update perps market proposal  in the database", func(t *testing.T) {
		err = ps.Add(ctx, proposal)
		require.NoError(t, err)
	})

	var savedProp []entities.Proposal
	t.Run("Get should return the perps market proposal from the database", func(t *testing.T) {
		savedProp, _, err = ps.Get(ctx, nil, nil, nil, entities.CursorPagination{})
		require.NoError(t, err)
		require.Len(t, savedProp, 1)
	})

	t.Run("The saved proposal's instrument should be a perp and not a future or spot", func(t *testing.T) {
		savedToProto := savedProp[0].ToProto()
		assert.Nil(t, savedToProto.Terms.GetNewSpotMarket())
		assert.NotNil(t, savedToProto.Terms.GetUpdateMarket())
		future := savedToProto.Terms.GetUpdateMarket().GetChanges().GetInstrument().GetFuture()
		assert.Nil(t, future)
		perps := savedToProto.Terms.GetUpdateMarket().GetChanges().GetInstrument().GetPerpetual()
		assert.NotNil(t, perps)
	})
}
