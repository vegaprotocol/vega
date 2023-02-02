package adapters

import (
	"context"
	"sort"

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

func toSpamStatistic(st *apipb.SpamStatistic) *nodetypes.SpamStatistic {
	return &nodetypes.SpamStatistic{
		CountForEpoch: st.CountForEpoch,
		MaxForEpoch:   st.MaxForEpoch,
		BannedUntil:   st.BannedUntil,
	}
}

func (c *InsecureGRPCAdapter) SpamStatistics(ctx context.Context, party string) (nodetypes.SpamStatistics, error) {
	r, err := c.client.GetSpamStatistics(ctx,
		&apipb.GetSpamStatisticsRequest{
			PartyId: party,
		})
	if err != nil {
		return nodetypes.SpamStatistics{}, err
	}

	proposals := map[string]uint64{}
	for _, st := range r.Statistics.Votes.Statistics {
		proposals[st.Proposal] = st.CountForEpoch
	}

	blockStates := []nodetypes.PoWBlockState{}
	for _, b := range r.Statistics.Pow.BlockStates {
		blockStates = append(blockStates, nodetypes.PoWBlockState{
			BlockHeight:          b.BlockHeight,
			BlockHash:            b.BlockHash,
			TransactionsSeen:     b.TransactionsSeen,
			ExpectedDifficulty:   b.ExpectedDifficulty,
			HashFunction:         b.HashFunction,
			Difficulty:           b.Difficulty,
			TxPerBlock:           b.TxPerBlock,
			IncreasingDifficulty: b.IncreasingDifficulty,
		})
	}

	// sort by block-height so latest block is first
	sort.Slice(blockStates, func(i int, j int) bool {
		return blockStates[i].BlockHeight > blockStates[j].BlockHeight
	})

	var lastBlockHeight uint64
	if len(blockStates) > 0 {
		lastBlockHeight = blockStates[0].BlockHeight
	}
	return nodetypes.SpamStatistics{
		Proposals:         toSpamStatistic(r.Statistics.Proposals),
		Delegations:       toSpamStatistic(r.Statistics.Delegations),
		Transfers:         toSpamStatistic(r.Statistics.Transfers),
		NodeAnnouncements: toSpamStatistic(r.Statistics.NodeAnnouncements),
		IssuesSignatures:  toSpamStatistic(r.Statistics.IssueSignatures),
		Votes: &nodetypes.VoteSpamStatistics{
			Proposals:   proposals,
			MaxForEpoch: r.Statistics.Votes.MaxForEpoch,
			BannedUntil: r.Statistics.Votes.BannedUntil,
		},
		PoW: &nodetypes.PoWStatistics{
			PowBlockStates: blockStates,
			BannedUntil:    r.Statistics.Pow.BannedUntil,
			PastBlocks:     r.Statistics.Pow.NumberOfPastBlocks,
		},
		ChainID:         r.ChainId,
		EpochSeq:        r.Statistics.EpochSeq,
		LastBlockHeight: lastBlockHeight,
	}, nil
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
