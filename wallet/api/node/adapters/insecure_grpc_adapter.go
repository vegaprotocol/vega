package adapters

import (
	"context"

	apipb "code.vegaprotocol.io/vega/protos/vega/api/v1"
	nodetypes "code.vegaprotocol.io/vega/wallet/api/node/types"
	"google.golang.org/grpc/credentials/insecure"

	"google.golang.org/grpc"
)

type InsecureGRPCAdapter struct {
	client     apipb.CoreServiceClient
	connection *grpc.ClientConn

	host string
}

func (c *InsecureGRPCAdapter) Host() string {
	return c.host
}

func (c *InsecureGRPCAdapter) Statistics(ctx context.Context) (nodetypes.Statistics, error) {
	statistics, err := c.client.Statistics(ctx, &apipb.StatisticsRequest{})
	if err != nil {
		return nodetypes.Statistics{}, err
	}

	return nodetypes.Statistics{
		BlockHeight: statistics.Statistics.BlockHeight,
		BlockHash:   statistics.Statistics.BlockHash,
		ChainID:     statistics.Statistics.ChainId,
		VegaTime:    statistics.Statistics.VegaTime,
	}, nil
}

func (c *InsecureGRPCAdapter) SubmitTransaction(ctx context.Context, req *apipb.SubmitTransactionRequest) (*apipb.SubmitTransactionResponse, error) {
	return c.client.SubmitTransaction(ctx, req)
}

func (c *InsecureGRPCAdapter) LastBlock(ctx context.Context) (nodetypes.LastBlock, error) {
	lastBlock, err := c.client.LastBlockHeight(ctx, &apipb.LastBlockHeightRequest{})
	if err != nil {
		return nodetypes.LastBlock{}, err
	}

	return nodetypes.LastBlock{
		ChainID:                         lastBlock.ChainId,
		BlockHeight:                     lastBlock.Height,
		BlockHash:                       lastBlock.Hash,
		ProofOfWorkHashFunction:         lastBlock.SpamPowHashFunction,
		ProofOfWorkDifficulty:           lastBlock.SpamPowDifficulty,
		ProofOfWorkPastBlocks:           lastBlock.SpamPowNumberOfPastBlocks,
		ProofOfWorkIncreasingDifficulty: lastBlock.SpamPowIncreasingDifficulty,
		ProofOfWorkTxPerBlock:           lastBlock.SpamPowNumberOfTxPerBlock,
	}, nil
}

func (c *InsecureGRPCAdapter) Stop() error {
	return c.connection.Close()
}

func NewInsecureGRPCAdapter(host string) (*InsecureGRPCAdapter, error) {
	connection, err := grpc.Dial(host, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	return &InsecureGRPCAdapter{
		client:     apipb.NewCoreServiceClient(connection),
		connection: connection,
		host:       host,
	}, nil
}
