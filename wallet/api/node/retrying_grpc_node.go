package node

import (
	"context"
	"time"

	apipb "code.vegaprotocol.io/vega/protos/vega/api/v1"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	"github.com/cenkalti/backoff/v4"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
)

type RetryingGRPCNode struct {
	log *zap.Logger

	client CoreClient

	retries uint64
}

func (n *RetryingGRPCNode) Host() string {
	return n.client.Host()
}

// HealthCheck returns an error if the node is not healthy.
// Contrary to the other calls, it doesn't retry if the call ends up in error
// to avoid useless delay in during node selection.
func (n *RetryingGRPCNode) HealthCheck(ctx context.Context) error {
	n.log.Debug("verifying the node health through the core client", zap.String("host", n.client.Host()))
	req := apipb.GetVegaTimeRequest{}
	resp, err := n.client.GetVegaTime(ctx, &req)
	if err != nil {
		n.log.Error("could not get the vega time",
			zap.String("host", n.client.Host()),
			zap.Error(err),
		)
		return err
	}
	n.log.Debug("response from GetVegaTime",
		zap.String("host", n.client.Host()),
		zap.Int64("timestamp", resp.Timestamp),
	)
	return nil
}

// LastBlock returns information about the last block acknowledged by the node.
func (n *RetryingGRPCNode) LastBlock(ctx context.Context) (*apipb.LastBlockHeightResponse, error) {
	n.log.Debug("getting the last block from the core client", zap.String("host", n.client.Host()))
	var resp *apipb.LastBlockHeightResponse
	if err := n.retry(func() error {
		request := apipb.LastBlockHeightRequest{}
		r, err := n.client.LastBlockHeight(ctx, &request)
		if err != nil {
			return err
		}
		resp = r
		n.log.Debug("response from LastBlockHeight",
			zap.String("host", n.client.Host()),
			zap.Uint64("block-height", r.Height),
			zap.String("block-hash", r.Hash),
			zap.Time("request-time", time.Now()),
			zap.Uint32("pow-difficulty", r.SpamPowDifficulty),
			zap.String("pow-hash-function", r.SpamPowHashFunction),
		)
		return nil
	}); err != nil {
		n.log.Error("could not the get last block",
			zap.String("host", n.client.Host()),
			zap.Error(err),
		)
		return nil, err
	}

	return resp, nil
}

func (n *RetryingGRPCNode) CheckTransaction(ctx context.Context, tx *commandspb.Transaction) (*apipb.CheckTransactionResponse, error) {
	n.log.Debug("checking the transaction through the core client", zap.String("host", n.client.Host()))
	req := apipb.CheckTransactionRequest{
		Tx: tx,
	}
	var resp *apipb.CheckTransactionResponse
	if err := n.retry(func() error {
		r, err := n.client.CheckTransaction(ctx, &req)
		if err != nil {
			return err
		}
		n.log.Debug("response from CheckTransaction",
			zap.Bool("success", r.Success),
		)
		resp = r
		return nil
	}); err != nil {
		n.log.Error("could not check the transaction",
			zap.String("host", n.client.Host()),
			zap.Error(err),
		)
		return nil, err
	}

	return resp, nil
}

func (n *RetryingGRPCNode) SendTransaction(ctx context.Context, tx *commandspb.Transaction, ty apipb.SubmitTransactionRequest_Type) (string, error) {
	n.log.Debug("sending the transaction to the core", zap.String("host", n.client.Host()))
	var resp *apipb.SubmitTransactionResponse
	if err := n.retry(func() error {
		req := apipb.SubmitTransactionRequest{
			Tx:   tx,
			Type: ty,
		}
		r, err := n.client.SubmitTransaction(ctx, &req)
		if err != nil {
			return n.handleSubmissionError(err)
		}
		n.log.Debug("response from SubmitTransaction",
			zap.String("host", n.client.Host()),
			zap.Bool("success", r.Success),
			zap.String("hash", r.TxHash),
		)
		resp = r
		return nil
	}); err != nil {
		return "", err
	}

	return resp.TxHash, nil
}

func (n *RetryingGRPCNode) Stop() error {
	n.log.Debug("closing the core client", zap.String("host", n.client.Host()))
	if err := n.client.Stop(); err != nil {
		n.log.Warn("could not stop the core client",
			zap.String("host", n.client.Host()),
			zap.Error(err),
		)
		return err
	}
	n.log.Info("the core client successfully closed", zap.String("host", n.client.Host()))
	return nil
}

func (n *RetryingGRPCNode) handleSubmissionError(err error) error {
	statusErr := intoStatusError(err)

	if statusErr == nil {
		n.log.Error("could not submit the transaction",
			zap.String("host", n.client.Host()),
			zap.Error(err),
		)
		return err
	}

	if statusErr.Code == codes.InvalidArgument {
		n.log.Error(
			"the transaction has been rejected because of an invalid argument or state, skipping retry...",
			zap.String("host", n.client.Host()),
			zap.Error(statusErr),
		)
		// Returning a permanent error kills the retry loop.
		return backoff.Permanent(statusErr)
	}

	n.log.Error("could not submit the transaction",
		zap.String("host", n.client.Host()),
		zap.Error(statusErr),
	)
	return statusErr
}

func (n *RetryingGRPCNode) retry(o backoff.Operation) error {
	return backoff.Retry(o, backoff.WithMaxRetries(backoff.NewExponentialBackOff(), n.retries))
}

func NewRetryingGRPCNode(log *zap.Logger, host string, retries uint64) (*RetryingGRPCNode, error) {
	client, err := NewInsecureGRPCClient(host)
	if err != nil {
		log.Error("could not initialise an insecure gRPC client",
			zap.String("host", host),
			zap.Error(err),
		)
		return nil, err
	}

	return BuildRetryingGRPCNode(log, client, retries), nil
}

func BuildRetryingGRPCNode(log *zap.Logger, client CoreClient, retries uint64) *RetryingGRPCNode {
	return &RetryingGRPCNode{
		log:     log,
		client:  client,
		retries: retries,
	}
}
