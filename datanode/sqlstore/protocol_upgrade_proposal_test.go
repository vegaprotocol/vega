// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package sqlstore_test

import (
	"context"
	"testing"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/sqlstore"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func addTestPUP(t *testing.T,
	ctx context.Context,
	status entities.ProtocolUpgradeProposalStatus,
	height uint64,
	tag string,
	approvers []string,
	store *sqlstore.ProtocolUpgradeProposals,
	block entities.Block,
) entities.ProtocolUpgradeProposal {
	t.Helper()
	pup := entities.ProtocolUpgradeProposal{
		UpgradeBlockHeight: height,
		VegaReleaseTag:     tag,
		Approvers:          approvers,
		Status:             status,
		VegaTime:           block.VegaTime,
		TxHash:             generateTxHash(),
	}
	err := store.Add(ctx, pup)
	require.NoError(t, err)
	if pup.Approvers == nil {
		pup.Approvers = []string{}
	}
	return pup
}

func TestProtocolUpgradeProposal(t *testing.T) {
	ctx := tempTransaction(t)

	pupPending := entities.ProtocolUpgradeProposalStatus(eventspb.ProtocolUpgradeProposalStatus_PROTOCOL_UPGRADE_PROPOSAL_STATUS_PENDING)
	pupApproved := entities.ProtocolUpgradeProposalStatus(eventspb.ProtocolUpgradeProposalStatus_PROTOCOL_UPGRADE_PROPOSAL_STATUS_APPROVED)
	pupRejected := entities.ProtocolUpgradeProposalStatus(eventspb.ProtocolUpgradeProposalStatus_PROTOCOL_UPGRADE_PROPOSAL_STATUS_REJECTED)

	blockStore := sqlstore.NewBlocks(connectionSource)
	block1 := addTestBlock(t, ctx, blockStore)
	block2 := addTestBlock(t, ctx, blockStore)
	block3 := addTestBlock(t, ctx, blockStore)
	store := sqlstore.NewProtocolUpgradeProposals(connectionSource)

	var pup1a, pup1b, pup2a, pup2b, pup3, pup4 entities.ProtocolUpgradeProposal

	t.Run("adding", func(t *testing.T) {
		pup1a = addTestPUP(t, ctx, pupPending, 1, "1.1", []string{"phil"}, store, block1)
		pup1b = addTestPUP(t, ctx, pupApproved, 1, "1.1", []string{"phil", "dave"}, store, block1) // Updated in same block
		pup2a = addTestPUP(t, ctx, pupPending, 2, "2.2", []string{"dave", "jim"}, store, block1)
		pup2b = addTestPUP(t, ctx, pupPending, 2, "2.2", []string{"jim"}, store, block2)           // Updated in next block
		pup3 = addTestPUP(t, ctx, pupApproved, 3, "3.3", []string{"roger", "fred"}, store, block2) // Updated in next block
		pup4 = addTestPUP(t, ctx, pupRejected, 4, "3.4", nil, store, block3)                       // Updated in next block
	})

	t.Run("list all", func(t *testing.T) {
		fetched, _, err := store.List(ctx, nil, nil, entities.CursorPagination{})
		require.NoError(t, err)

		expected := []entities.ProtocolUpgradeProposal{pup1b, pup2b, pup3, pup4}
		assert.Equal(t, expected, fetched)
	})

	t.Run("GetByTxHash", func(t *testing.T) {
		fetched, err := store.GetByTxHash(ctx, pup1b.TxHash)
		require.NoError(t, err)
		expected := []entities.ProtocolUpgradeProposal{pup1b}
		assert.Equal(t, expected, fetched)

		fetched, err = store.GetByTxHash(ctx, pup2b.TxHash)
		require.NoError(t, err)
		expected = []entities.ProtocolUpgradeProposal{pup2b}
		assert.Equal(t, expected, fetched)
	})

	t.Run("list all paged", func(t *testing.T) {
		cursor := pup1b.Cursor().Encode()
		var one int32 = 1
		p, err := entities.NewCursorPagination(&one, &cursor, nil, nil, false)
		require.NoError(t, err)

		fetched, pageInfo, err := store.List(ctx, nil, nil, p)
		require.NoError(t, err)

		expected := []entities.ProtocolUpgradeProposal{pup2b}
		assert.Equal(t, expected, fetched)
		assert.True(t, pageInfo.ToProto().HasNextPage)
	})

	t.Run("list approved", func(t *testing.T) {
		fetched, _, err := store.List(ctx, &pupApproved, nil, entities.CursorPagination{})
		require.NoError(t, err)

		expected := []entities.ProtocolUpgradeProposal{pup1b, pup3}
		assert.Equal(t, expected, fetched)
	})

	t.Run("list approved by", func(t *testing.T) {
		dave := "dave"
		fetched, _, err := store.List(ctx, nil, &dave, entities.CursorPagination{})
		require.NoError(t, err)

		expected := []entities.ProtocolUpgradeProposal{pup1b}
		assert.Equal(t, expected, fetched)
	})

	_, _ = pup1a, pup2a
}

func TestProtocolUpdateProposalStatusEnum(t *testing.T) {
	var protocolUpgradeProposalStatus eventspb.ProtocolUpgradeProposalStatus
	states := getEnums(t, protocolUpgradeProposalStatus)
	assert.Len(t, states, 4)
	for s, state := range states {
		t.Run(state, func(t *testing.T) {
			ctx := tempTransaction(t)
			height := uint64(1)
			tag := "1.1"
			approvers := []string{"phil"}
			blockStore := sqlstore.NewBlocks(connectionSource)
			block := addTestBlock(t, ctx, blockStore)
			store := sqlstore.NewProtocolUpgradeProposals(connectionSource)

			pup := entities.ProtocolUpgradeProposal{
				UpgradeBlockHeight: height,
				VegaReleaseTag:     tag,
				Approvers:          approvers,
				Status:             entities.ProtocolUpgradeProposalStatus(s),
				VegaTime:           block.VegaTime,
				TxHash:             generateTxHash(),
			}
			require.NoError(t, store.Add(ctx, pup))
			got, err := store.GetByTxHash(ctx, pup.TxHash)
			require.NoError(t, err)
			assert.Len(t, got, 1)
			assert.Equal(t, pup.Status, got[0].Status)
		})
	}
}
