package adapters

import (
	"context"
	"sort"
	"strings"

	apipb "code.vegaprotocol.io/vega/protos/vega/api/v1"
	nodetypes "code.vegaprotocol.io/vega/wallet/api/node/types"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	"google.golang.org/grpc"
)

type GRPCAdapter struct {
	client     apipb.CoreServiceClient
	connection *grpc.ClientConn

	host string
}

func (c *GRPCAdapter) Host() string {
	return c.host
}

func (c *GRPCAdapter) SpamStatistics(ctx context.Context, party string) (nodetypes.SpamStatistics, error) {
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
		MaxTTL:          r.Statistics.MaxTtl,
	}, nil
}

func (c *GRPCAdapter) Statistics(ctx context.Context) (nodetypes.Statistics, error) {
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

func (c *GRPCAdapter) CheckTransaction(ctx context.Context, req *apipb.CheckTransactionRequest) (*apipb.CheckTransactionResponse, error) {
	return c.client.CheckTransaction(ctx, req)
}

func (c *GRPCAdapter) SubmitTransaction(ctx context.Context, req *apipb.SubmitTransactionRequest) (*apipb.SubmitTransactionResponse, error) {
	return c.client.SubmitTransaction(ctx, req)
}

func (c *GRPCAdapter) LastBlock(ctx context.Context) (nodetypes.LastBlock, error) {
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

func (c *GRPCAdapter) Stop() error {
	return c.connection.Close()
}

func NewGRPCAdapter(host string) (*GRPCAdapter, error) {
	useTLS := strings.HasPrefix(host, "tls://")

	var creds credentials.TransportCredentials
	if useTLS {
		host = host[6:]
		creds = credentials.NewClientTLSFromCert(nil, "")
	} else {
		creds = insecure.NewCredentials()
	}

	connection, err := grpc.Dial(host, grpc.WithTransportCredentials(creds))
	if err != nil {
		return nil, err
	}

	return &GRPCAdapter{
		client:     apipb.NewCoreServiceClient(connection),
		connection: connection,
		host:       host,
	}, nil
}
