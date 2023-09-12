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
	"testing"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/sqlstore"
	vgcrypto "code.vegaprotocol.io/vega/libs/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func randomEthAddress() entities.EthereumAddress {
	hash256bit := vgcrypto.RandomHash()
	hash160bit := hash256bit[:40]
	return entities.EthereumAddress("0x" + hash160bit)
}

func addTestEthereumKeyRotation(t *testing.T,
	ctx context.Context,
	store *sqlstore.EthereumKeyRotations,
	block entities.Block,
	seqNum uint64,
) entities.EthereumKeyRotation {
	t.Helper()
	kr := entities.EthereumKeyRotation{
		NodeID:      entities.NodeID("beef"),
		OldAddress:  randomEthAddress(),
		NewAddress:  randomEthAddress(),
		VegaTime:    block.VegaTime,
		BlockHeight: 42,
		SeqNum:      seqNum,
		TxHash:      generateTxHash(),
	}
	err := store.Add(ctx, kr)
	require.NoError(t, err)
	return kr
}

func TestEthereumKeyRotations(t *testing.T) {
	ctx := tempTransaction(t)

	blockStore := sqlstore.NewBlocks(connectionSource)
	block := addTestBlock(t, ctx, blockStore)
	nodeStore := sqlstore.NewNode(connectionSource)
	node := addTestNode(t, ctx, nodeStore, block, "beef")

	krStore := sqlstore.NewEthereumKeyRotations(connectionSource)

	var kr entities.EthereumKeyRotation
	t.Run("adding", func(t *testing.T) {
		kr = addTestEthereumKeyRotation(t, ctx, krStore, block, 0)
	})

	t.Run("GetByTxHash", func(t *testing.T) {
		fetched, err := krStore.GetByTxHash(ctx, kr.TxHash)
		require.NoError(t, err)
		require.Len(t, fetched, 1)
		assert.Equal(t, fetched[0], kr)
	})

	t.Run("fetching all", func(t *testing.T) {
		fetched, _, err := krStore.List(ctx, entities.NodeID(""), entities.CursorPagination{})
		require.NoError(t, err)
		require.Len(t, fetched, 1)
		assert.Equal(t, fetched[0], kr)
	})

	t.Run("fetching all by node", func(t *testing.T) {
		fetched, _, err := krStore.List(ctx, node.ID, entities.CursorPagination{})
		require.NoError(t, err)
		require.Len(t, fetched, 1)
		assert.Equal(t, fetched[0], kr)
	})

	t.Run("fetching all by bad node", func(t *testing.T) {
		fetched, _, err := krStore.List(ctx, entities.NodeID("baad"), entities.CursorPagination{})
		require.NoError(t, err)
		require.Len(t, fetched, 0)
	})

	t.Run("adding more", func(t *testing.T) {
		for i := 0; i < 10; i++ {
			addTestEthereumKeyRotation(t, ctx, krStore, block, uint64(i+1))
		}
	})

	t.Run("with pagination", func(t *testing.T) {
		five := int32(5)
		pagination, err := entities.NewCursorPagination(&five, nil, nil, nil, true)
		require.NoError(t, err)

		fetched, pageInfo, err := krStore.List(ctx, entities.NodeID(""), pagination)
		require.NoError(t, err)
		require.Len(t, fetched, 5)
		require.True(t, pageInfo.HasNextPage)

		t.Run("using cursor paging forwards", func(t *testing.T) {
			pagination, err := entities.NewCursorPagination(&five, &pageInfo.StartCursor, nil, nil, true)
			require.NoError(t, err)

			fetchedAgain, pageInfo, err := krStore.List(ctx, entities.NodeID(""), pagination)
			require.NoError(t, err)
			require.Len(t, fetched, 5)
			require.True(t, pageInfo.HasNextPage)
			// Passing a cursor gets the next element
			require.Equal(t, fetched[1:5], fetchedAgain[0:4])
		})

		t.Run("using cursor paging back", func(t *testing.T) {
			pagination, err := entities.NewCursorPagination(nil, nil, &five, &pageInfo.EndCursor, true)
			require.NoError(t, err)

			fetchedAgain, pageInfo, err := krStore.List(ctx, entities.NodeID(""), pagination)
			require.NoError(t, err)
			// The last one won't be included
			require.Len(t, fetchedAgain, 4)
			require.True(t, pageInfo.HasNextPage)
			// But we should get the same 4 first guys
			require.Equal(t, fetched[0:4], fetchedAgain[0:4])
		})
	})
}
