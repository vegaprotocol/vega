package sqlstore_test

import (
	"context"
	"testing"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/sqlstore"
	"github.com/stretchr/testify/require"
)

func addTestNode(t *testing.T, ps *sqlstore.Node, block entities.Block) entities.Node {
	node := entities.Node{
		ID:              entities.NewNodeID(generateID()),
		PubKey:          entities.VegaPublicKey(generateID()),
		EthereumAddress: entities.EthereumAddress(generateEthereumAddress()),
		TmPubKey:        entities.TendermintPublicKey(generateTendermintPublicKey()),
		VegaTime:        block.VegaTime,
	}

	err := ps.UpsertNode(context.Background(), &node)
	require.NoError(t, err)
	return node
}
