package api

import (
	"context"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/blockchain"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/metrics"
	"code.vegaprotocol.io/vega/monitoring"
	types "code.vegaprotocol.io/vega/proto"
	protoapi "code.vegaprotocol.io/vega/proto/api"

	"github.com/golang/protobuf/proto"
	uuid "github.com/satori/go.uuid"
	"google.golang.org/grpc/codes"
)

// TradeOrderService ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/trade_order_service_mock.go -package mocks code.vegaprotocol.io/vega/api TradeOrderService
type TradeOrderService interface {
	PrepareSubmitOrder(ctx context.Context, submission *types.OrderSubmission) error
	PrepareCancelOrder(ctx context.Context, cancellation *types.OrderCancellation) error
	PrepareAmendOrder(ctx context.Context, amendment *types.OrderAmendment) error
	SubmitTransaction(ctx context.Context, bundle *types.SignedBundle) (bool, error)
}

// AccountService ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/account_service_mock.go -package mocks code.vegaprotocol.io/vega/api  AccountService
type AccountService interface {
	NotifyTraderAccount(ctx context.Context, notify *types.NotifyTraderAccount) (bool, error)
	Withdraw(context.Context, *types.Withdraw) (bool, error)
}

// GovernanceService ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/governance_service_mock.go -package mocks code.vegaprotocol.io/vega/api  GovernanceService
type GovernanceService interface {
	PrepareProposal(ctx context.Context, author, reference string, terms *types.ProposalTerms) (*types.Proposal, error)
	PrepareVote(vote *types.Vote) (*types.Vote, error)
}

type tradingService struct {
	log               *logging.Logger
	tradeOrderService TradeOrderService
	accountService    AccountService
	marketService     MarketService
	governanceService GovernanceService
	statusChecker     *monitoring.Status

	mu sync.Mutex
}

func (s *tradingService) PrepareSubmitOrder(ctx context.Context, req *protoapi.SubmitOrderRequest) (*protoapi.PrepareSubmitOrderResponse, error) {
	startTime := time.Now()
	defer metrics.APIRequestAndTimeGRPC("PrepareSubmitOrder", startTime)
	err := s.tradeOrderService.PrepareSubmitOrder(ctx, req.Submission)
	if err != nil {
		return nil, apiError(codes.Internal, ErrMalformedRequest, err)
	}
	raw, err := proto.Marshal(req.Submission)
	if err != nil {
		return nil, apiError(codes.Internal, ErrSubmitOrder, err)
	}
	if raw, err = txEncode(raw, blockchain.SubmitOrderCommand); err != nil {
		return nil, apiError(codes.Internal, ErrSubmitOrder, err)
	}
	return &protoapi.PrepareSubmitOrderResponse{
		Blob:     raw,
		SubmitID: req.Submission.Reference,
	}, nil
}

func (s *tradingService) PrepareCancelOrder(ctx context.Context, req *protoapi.CancelOrderRequest) (*protoapi.PrepareCancelOrderResponse, error) {
	startTime := time.Now()
	defer metrics.APIRequestAndTimeGRPC("PrepareCancelOrder", startTime)
	err := s.tradeOrderService.PrepareCancelOrder(ctx, req.Cancellation)
	if err != nil {
		return nil, apiError(codes.Internal, ErrCancelOrder, err)
	}
	raw, err := proto.Marshal(req.Cancellation)
	if err != nil {
		return nil, apiError(codes.Internal, ErrCancelOrder, err)
	}
	if raw, err = txEncode(raw, blockchain.CancelOrderCommand); err != nil {
		return nil, apiError(codes.Internal, ErrCancelOrder, err)
	}
	return &protoapi.PrepareCancelOrderResponse{
		Blob: raw,
	}, nil
}

func (s *tradingService) PrepareAmendOrder(ctx context.Context, req *protoapi.AmendOrderRequest) (*protoapi.PrepareAmendOrderResponse, error) {
	startTime := time.Now()
	defer metrics.APIRequestAndTimeGRPC("PrepareAmendOrder", startTime)
	err := s.tradeOrderService.PrepareAmendOrder(ctx, req.Amendment)
	if err != nil {
		return nil, apiError(codes.Internal, ErrAmendOrder, err)
	}
	raw, err := proto.Marshal(req.Amendment)
	if err != nil {
		return nil, apiError(codes.Internal, ErrAmendOrder, err)
	}
	if raw, err = txEncode(raw, blockchain.AmendOrderCommand); err != nil {
		return nil, apiError(codes.Internal, ErrAmendOrder, err)
	}
	return &protoapi.PrepareAmendOrderResponse{
		Blob: raw,
	}, nil
}

func (s *tradingService) SubmitTransaction(ctx context.Context, req *protoapi.SubmitTransactionRequest) (*protoapi.SubmitTransactionResponse, error) {
	startTime := time.Now()
	defer metrics.APIRequestAndTimeGRPC("SubmitTransaction", startTime)
	if req == nil || req.Tx == nil {
		return nil, apiError(codes.InvalidArgument, ErrMalformedRequest)
	}
	if ok, err := s.tradeOrderService.SubmitTransaction(ctx, req.Tx); err != nil || !ok {
		s.log.Error("unable to submit transaction", logging.Error(err))
		return nil, apiError(codes.Internal, err)
	}
	return &protoapi.SubmitTransactionResponse{
		Success: true,
	}, nil
}

func (s *tradingService) NotifyTraderAccount(
	ctx context.Context, req *protoapi.NotifyTraderAccountRequest,
) (*protoapi.NotifyTraderAccountResponse, error) {
	startTime := time.Now()
	defer metrics.APIRequestAndTimeGRPC("NotifyTraderAccount", startTime)
	if req == nil || req.Notif == nil {
		return nil, apiError(codes.InvalidArgument, ErrMalformedRequest)
	}
	if len(req.Notif.TraderID) <= 0 {
		return nil, apiError(codes.InvalidArgument, ErrMissingTraderID)
	}

	submitted, err := s.accountService.NotifyTraderAccount(ctx, req.Notif)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	return &protoapi.NotifyTraderAccountResponse{
		Submitted: submitted,
	}, nil
}

func (s *tradingService) Withdraw(
	ctx context.Context, req *protoapi.WithdrawRequest,
) (*protoapi.WithdrawResponse, error) {
	startTime := time.Now()
	defer metrics.APIRequestAndTimeGRPC("Withdraw", startTime)
	if len(req.Withdraw.PartyID) <= 0 {
		return nil, apiError(codes.InvalidArgument, ErrMissingTraderID)
	}
	if len(req.Withdraw.Asset) <= 0 {
		return nil, apiError(codes.InvalidArgument, ErrMissingAsset)
	}
	if req.Withdraw.Amount == 0 {
		return nil, apiError(codes.InvalidArgument, ErrInvalidWithdrawAmount)
	}

	ok, err := s.accountService.Withdraw(ctx, req.Withdraw)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	return &protoapi.WithdrawResponse{
		Success: ok,
	}, nil
}

func (s *tradingService) PrepareProposal(
	ctx context.Context, req *protoapi.PrepareProposalRequest,
) (*protoapi.PrepareProposalResponse, error) {
	startTime := time.Now()
	defer metrics.APIRequestAndTimeGRPC("PrepareProposal", startTime)

	if err := req.Validate(); err != nil {
		return nil, apiError(codes.InvalidArgument, ErrMalformedRequest, err)
	}
	proposal, err := s.governanceService.PrepareProposal(ctx, req.PartyID, req.Reference, req.Proposal)
	if err != nil {
		return nil, apiError(codes.Internal, ErrPrepareProposal, err)
	}
	raw, err := proto.Marshal(proposal) // marshal whole proposal
	if err != nil {
		return nil, apiError(codes.Internal, ErrPrepareProposal, err)
	}
	if raw, err = txEncode(raw, blockchain.ProposeCommand); err != nil {
		return nil, apiError(codes.Internal, ErrPrepareProposal, err)
	}
	return &protoapi.PrepareProposalResponse{
		Blob:            raw,
		PendingProposal: proposal,
	}, nil
}

func (s *tradingService) PrepareVote(ctx context.Context, req *protoapi.PrepareVoteRequest) (*protoapi.PrepareVoteResponse, error) {
	startTime := time.Now()
	defer metrics.APIRequestAndTimeGRPC("PrepareVote", startTime)

	if err := req.Validate(); err != nil {
		return nil, apiError(codes.InvalidArgument, ErrMalformedRequest, err)
	}

	if req.Vote.Value == types.Vote_VALUE_UNSPECIFIED {
		return nil, apiError(codes.InvalidArgument, ErrMalformedRequest)
	}

	vote, err := s.governanceService.PrepareVote(req.Vote)
	if err != nil {
		return nil, apiError(codes.Internal, ErrPrepareVote, err)
	}
	raw, err := proto.Marshal(vote)
	if err != nil {
		return nil, apiError(codes.Internal, ErrPrepareVote, err)
	}
	if raw, err = txEncode(raw, blockchain.VoteCommand); err != nil {
		return nil, apiError(codes.Internal, ErrPrepareVote, err)
	}
	return &protoapi.PrepareVoteResponse{
		Blob: raw,
		Vote: vote,
	}, nil
}

func txEncode(input []byte, cmd blockchain.Command) (proto []byte, err error) {
	prefix := uuid.NewV4().String()
	prefixBytes := []byte(prefix)
	commandInput := append([]byte{byte(cmd)}, input...)
	return append(prefixBytes, commandInput...), nil
}
