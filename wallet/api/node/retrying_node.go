package node

import (
	"context"
	"fmt"
	"time"

	apipb "code.vegaprotocol.io/vega/protos/vega/api/v1"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	"code.vegaprotocol.io/vega/wallet/api/node/adapters"
	nodetypes "code.vegaprotocol.io/vega/wallet/api/node/types"
	"github.com/cenkalti/backoff/v4"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/node_mock.go -package mocks code.vegaprotocol.io/vega/wallet/api/node GRPCAdapter

type GRPCAdapter interface {
	Host() string
	Statistics(ctx context.Context) (nodetypes.Statistics, error)
	SpamStatistics(ctx context.Context, pubKey string) (nodetypes.SpamStatistics, error)
	SubmitTransaction(ctx context.Context, in *apipb.SubmitTransactionRequest) (*apipb.SubmitTransactionResponse, error)
	LastBlock(ctx context.Context) (nodetypes.LastBlock, error)
	Stop() error
}

type RetryingNode struct {
	log *zap.Logger

	grpcAdapter GRPCAdapter

	retries uint64
}

func (n *RetryingNode) Host() string {
	return n.grpcAdapter.Host()
}

func (n *RetryingNode) Statistics(ctx context.Context) (nodetypes.Statistics, error) {
	n.log.Debug("querying the node statistics through the graphQL API", zap.String("host", n.grpcAdapter.Host()))
	requestTime := time.Now()
	resp, err := n.grpcAdapter.Statistics(ctx)
	if err != nil {
		n.log.Error("could not get the statistics",
			zap.String("host", n.grpcAdapter.Host()),
			zap.Error(err),
		)
		return nodetypes.Statistics{}, err
	}
	n.log.Debug("response from Statistics",
		zap.String("host", n.grpcAdapter.Host()),
		zap.Uint64("block-height", resp.BlockHeight),
		zap.String("block-hash", resp.BlockHash),
		zap.String("chain-id", resp.ChainID),
		zap.String("vega-time", resp.VegaTime),
		zap.Time("request-time", requestTime),
	)
	return resp, nil
}

func (n *RetryingNode) SpamStatistics(ctx context.Context, pubKey string) (nodetypes.SpamStatistics, error) {
	n.log.Debug("querying the node statistics through the graphQL API", zap.String("host", n.grpcAdapter.Host()))
	requestTime := time.Now()
	resp, err := n.grpcAdapter.SpamStatistics(ctx, pubKey)
	if err != nil {
		n.log.Error("could not get the statistics",
			zap.String("host", n.grpcAdapter.Host()),
			zap.Error(err),
		)
		return nodetypes.SpamStatistics{}, err
	}

	n.log.Debug("response from SpamStatistics",
		zap.String("host", n.grpcAdapter.Host()),
		zap.String("chain-id", resp.ChainID),
		zap.Uint64("epoch", resp.EpochSeq),
		zap.Uint64("block-height", resp.LastBlockHeight),
		zap.Uint64("prosposals-count-for-epoch", resp.Proposals.CountForEpoch),
		zap.Uint64("transfers-count-for-epoch", resp.Transfers.CountForEpoch),
		zap.Uint64("delegations-count-for-epoch", resp.Delegations.CountForEpoch),
		zap.Uint64("issue-signatures-count-for-epoch", resp.IssuesSignatures.CountForEpoch),
		zap.Uint64("node-announcements-count-for-epoch", resp.NodeAnnouncements.CountForEpoch),
		zap.Time("request-time", requestTime),
	)
	return resp, nil
}

// LastBlock returns information about the last block acknowledged by the node.
func (n *RetryingNode) LastBlock(ctx context.Context) (nodetypes.LastBlock, error) {
	n.log.Debug("getting the last block from the gRPC API", zap.String("host", n.grpcAdapter.Host()))
	var resp nodetypes.LastBlock
	if err := n.retry(func() error {
		requestTime := time.Now()
		r, err := n.grpcAdapter.LastBlock(ctx)
		if err != nil {
			return err
		}
		resp = r
		n.log.Debug("response from LastBlockHeight",
			zap.String("host", n.grpcAdapter.Host()),
			zap.Uint64("block-height", r.BlockHeight),
			zap.String("block-hash", r.BlockHash),
			zap.Uint32("pow-difficulty", r.ProofOfWorkDifficulty),
			zap.String("pow-hash-function", r.ProofOfWorkHashFunction),
			zap.Time("request-time", requestTime),
		)
		return nil
	}); err != nil {
		n.log.Error("could not the get last block",
			zap.String("host", n.grpcAdapter.Host()),
			zap.Error(err),
		)
		return nodetypes.LastBlock{}, err
	}

	return resp, nil
}

func (n *RetryingNode) SendTransaction(ctx context.Context, tx *commandspb.Transaction, ty apipb.SubmitTransactionRequest_Type) (string, error) {
	n.log.Debug("sending the transaction through the gRPC API", zap.String("host", n.grpcAdapter.Host()))
	var resp *apipb.SubmitTransactionResponse
	if err := n.retry(func() error {
		req := apipb.SubmitTransactionRequest{
			Tx:   tx,
			Type: ty,
		}
		requestTime := time.Now()
		r, err := n.grpcAdapter.SubmitTransaction(ctx, &req)
		if err != nil {
			return n.handleSubmissionError(err)
		}
		n.log.Debug("response from SubmitTransaction",
			zap.String("host", n.grpcAdapter.Host()),
			zap.Bool("success", r.Success),
			zap.String("hash", r.TxHash),
			zap.Time("request-time", requestTime),
		)
		resp = r
		return nil
	}); err != nil {
		return "", err
	}

	if !resp.Success {
		return "", nodetypes.TransactionError{
			ABCICode: resp.Code,
			Message:  resp.Data,
		}
	}

	return resp.TxHash, nil
}

func (n *RetryingNode) Stop() error {
	n.log.Debug("closing the gRPC API client", zap.String("host", n.grpcAdapter.Host()))
	if err := n.grpcAdapter.Stop(); err != nil {
		n.log.Warn("could not stop the gRPC API client-",
			zap.String("host", n.grpcAdapter.Host()),
			zap.Error(err),
		)
		return fmt.Errorf("could not close properly stop the gRPC API client: %w", err)
	}
	n.log.Info("the gRPC API client successfully closed", zap.String("host", n.grpcAdapter.Host()))
	return nil
}

func (n *RetryingNode) handleSubmissionError(err error) error {
	statusErr := intoStatusError(err)

	if statusErr == nil {
		n.log.Error("could not submit the transaction",
			zap.String("host", n.grpcAdapter.Host()),
			zap.Error(err),
		)
		return err
	}

	if statusErr.Code == codes.InvalidArgument {
		n.log.Error(
			"the transaction has been rejected because of an invalid argument or state, skipping retry...",
			zap.String("host", n.grpcAdapter.Host()),
			zap.Error(statusErr),
		)
		// Returning a permanent error kills the retry loop.
		return backoff.Permanent(statusErr)
	}

	n.log.Error("could not submit the transaction",
		zap.String("host", n.grpcAdapter.Host()),
		zap.Error(statusErr),
	)
	return statusErr
}

func (n *RetryingNode) retry(o backoff.Operation) error {
	return backoff.Retry(o, backoff.WithMaxRetries(backoff.NewExponentialBackOff(), n.retries))
}

func NewRetryingNode(log *zap.Logger, host string, retries uint64) (*RetryingNode, error) {
	grpcAdapter, err := adapters.NewInsecureGRPCAdapter(host)
	if err != nil {
		log.Error("could not initialise an insecure gRPC adapter",
			zap.String("host", host),
			zap.Error(err),
		)
		return nil, err
	}

	return BuildRetryingNode(log, grpcAdapter, retries), nil
}

func BuildRetryingNode(log *zap.Logger, grpcAdapter GRPCAdapter, retries uint64) *RetryingNode {
	return &RetryingNode{
		log:         log,
		grpcAdapter: grpcAdapter,
		retries:     retries,
	}
}
