package sqlstore_test

import (
	"context"
	"testing"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/sqlstore"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func addTestNode(t *testing.T, ps *sqlstore.Node, block entities.Block) entities.Node {
	t.Helper()
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

func addRankingScore(t *testing.T, ps *sqlstore.Node, block entities.Block, node entities.Node, epoch uint64) entities.RankingScore {
	t.Helper()
	r := entities.RankingScore{
		StakeScore:       decimal.NewFromFloat(0.5),
		PerformanceScore: decimal.NewFromFloat(0.25),
		PreviousStatus:   entities.ValidatorNodeStatusErsatz,
		Status:           entities.ValidatorNodeStatusTendermint,
		EpochSeq:         epoch,
		VegaTime:         block.VegaTime,
	}

	aux := entities.RankingScoreAux{
		NodeId:   node.ID,
		EpochSeq: epoch,
	}

	err := ps.UpsertRanking(context.Background(), &r, &aux)
	require.NoError(t, err)
	return r
}

func TestGetNodes(t *testing.T) {
	DeleteEverything()
	defer DeleteEverything()
	ctx := context.Background()
	bs := sqlstore.NewBlocks(connectionSource)
	ns := sqlstore.NewNode(connectionSource)
	block := addTestBlock(t, bs)

	node1 := addTestNode(t, ns, block)
	addRankingScore(t, ns, block, node1, 3)

	// get all nodes
	found, err := ns.GetNodes(ctx, 3)
	require.NoError(t, err)
	require.Len(t, found, 1)

	// get single node in epoch where it had a ranking
	node, err := ns.GetNodeByID(ctx, node1.ID.String(), 3)
	require.NoError(t, err)
	require.NotNil(t, node)
	require.NotNil(t, node.RankingScore)

	// get single node in epoch where it didn't have a ranking
	node, err = ns.GetNodeByID(ctx, node1.ID.String(), 2)
	require.NoError(t, err)
	require.NotNil(t, node)
	require.Nil(t, node.RankingScore)

	node, err = ns.GetNodeByID(ctx, "DEADBEEF", 3)
	require.Error(t, err)
}
