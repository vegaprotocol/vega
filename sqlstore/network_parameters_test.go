package sqlstore_test

import (
	"context"
	"testing"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/sqlstore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func addNetParam(t *testing.T, ns *sqlstore.NetworkParameters, key, value string, block entities.Block) entities.NetworkParameter {
	p := entities.NetworkParameter{
		Key:      key,
		Value:    value,
		VegaTime: block.VegaTime,
	}
	ns.Add(context.Background(), p)
	return p
}

func TestNetParams(t *testing.T) {
	defer testStore.DeleteEverything()
	ctx := context.Background()
	netParamStore := sqlstore.NewNetworkParameters(testStore)
	blockStore := sqlstore.NewBlocks(testStore)
	block1 := addTestBlock(t, blockStore)
	block2 := addTestBlock(t, blockStore)

	param1a := addNetParam(t, netParamStore, "foo", "bar", block1)
	param1b := addNetParam(t, netParamStore, "foo", "baz", block1)
	param2a := addNetParam(t, netParamStore, "cake", "apples", block1)
	param2b := addNetParam(t, netParamStore, "cake", "bananna", block2)

	_ = param1a
	_ = param2a

	t.Run("GetAll", func(t *testing.T) {
		expected := []entities.NetworkParameter{param2b, param1b}
		actual, err := netParamStore.GetAll(ctx)
		require.NoError(t, err)
		assert.Equal(t, expected, actual)
	})
}
