package types_test

import (
	"testing"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/proto"
	cometbftdb "github.com/cometbft/cometbft-db"
	"github.com/cosmos/iavl"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSnapshot(t *testing.T) {
	t.Run("A tree can be exported serialised and then imported", testTreeExportImport)
}

func testTreeExportImport(t *testing.T) {
	// get a avl tree with some payloads in it
	db := cometbftdb.NewMemDB()
	tree := getPopulatedTree(t, db)

	// export the tree into snapshot data
	snap, err := types.SnapshotFromTree(tree.ImmutableTree)
	require.NoError(t, err)
	hash, _ := tree.Hash()
	require.Equal(t, snap.Hash, hash)
	require.Equal(t, snap.Meta.Version, tree.Version())
	require.Equal(t, len(snap.Nodes), int(tree.Size()))
	// We expect more nodehashes than nodes since nodes only contain the leaf nodes
	// with payloads whereas nodehashes contain the payload-less subtree-roots
	require.Greater(t, len(snap.Meta.NodeHashes), len(snap.Nodes))

	// Note IRL it would be now that snapshot is serialised and sent
	// via TM to the node restoring from a snapshot

	// Make a new tree waiting to import the snapshot
	importedTree, err := iavl.NewMutableTree(db, 0, false)
	require.NoError(t, err)

	_, err = importedTree.Load()
	require.NoError(t, err)

	// import the snapshot data into a new avl tree
	_, err = types.SnapshotFromTree(importedTree.ImmutableTree)
	require.NoError(t, err)

	// The new tree should be identical to the previous
	treeHash, err := tree.Hash()
	require.NoError(t, err)

	importedTreeHash, err := importedTree.Hash()
	require.NoError(t, err)

	assert.Equal(t, treeHash, importedTreeHash)
	assert.Equal(t, tree.Size(), importedTree.Size())
	assert.Equal(t, tree.Height(), importedTree.Height())
	assert.Equal(t, tree.Version(), importedTree.Version())
}

// returns an avl tree populated with some payloads.
func getPopulatedTree(t *testing.T, db *cometbftdb.MemDB) *iavl.MutableTree {
	t.Helper()
	testPayloads := []types.Payload{
		{
			Data: &types.PayloadAppState{
				AppState: &types.AppState{
					Height: 64,
				},
			},
		},
		{
			Data: &types.PayloadGovernanceActive{
				GovernanceActive: &types.GovernanceActive{},
			},
		},
		{
			Data: &types.PayloadGovernanceEnacted{
				GovernanceEnacted: &types.GovernanceEnacted{},
			},
		},
		{
			Data: &types.PayloadDelegationActive{
				DelegationActive: &types.DelegationActive{},
			},
		},
		{
			Data: &types.PayloadEpoch{
				EpochState: &types.EpochState{
					Seq:                  7,
					ReadyToStartNewEpoch: true,
				},
			},
		},
	}

	tree, err := iavl.NewMutableTree(db, 0, false)
	require.NoError(t, err)
	_, err = tree.Load()
	require.NoError(t, err)

	for _, p := range testPayloads {
		v, _ := proto.Marshal(p.IntoProto())
		_, err = tree.Set([]byte(p.TreeKey()), v)
		require.NoError(t, err)
	}

	_, _, err = tree.SaveVersion()
	require.NoError(t, err)

	return tree
}
