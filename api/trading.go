package api

import (
	"context"
	"encoding/hex"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/evtforward"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/metrics"
	"code.vegaprotocol.io/vega/monitoring"
	types "code.vegaprotocol.io/vega/proto"
	protoapi "code.vegaprotocol.io/vega/proto/api"
	"code.vegaprotocol.io/vega/txn"
	"code.vegaprotocol.io/vega/wallet/crypto"

	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
)

var (
	ErrInvalidSignature = errors.New("invalid signature")
)

// TradeOrderService ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/trade_order_service_mock.go -package mocks code.vegaprotocol.io/vega/api TradeOrderService
type TradeOrderService interface {
	PrepareSubmitOrder(ctx context.Context, submission *types.OrderSubmission) error
	PrepareCancelOrder(ctx context.Context, cancellation *types.OrderCancellation) error
	PrepareAmendOrder(ctx context.Context, amendment *types.OrderAmendment) error
}

// LiquidityService ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/liquidity_service_mock.go -package mocks code.vegaprotocol.io/vega/api LiquidityService
type LiquidityService interface {
	PrepareLiquidityProvisionSubmission(context.Context, *types.LiquidityProvisionSubmission) error
	Get(market, party string) ([]types.LiquidityProvision, error)
}

// AccountService ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/account_service_mock.go -package mocks code.vegaprotocol.io/vega/api  AccountService
type AccountService interface {
	PrepareWithdraw(context.Context, *types.WithdrawSubmission) error
}

// GovernanceService ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/governance_service_mock.go -package mocks code.vegaprotocol.io/vega/api  GovernanceService
type GovernanceService interface {
	PrepareProposal(ctx context.Context, author, reference string, terms *types.ProposalTerms) (*types.Proposal, error)
	PrepareVote(vote *types.Vote) (*types.Vote, error)
}

// EvtForwarder
//go:generate go run github.com/golang/mock/mockgen -destination mocks/evt_forwarder_mock.go -package mocks code.vegaprotocol.io/vega/api  EvtForwarder
type EvtForwarder interface {
	Forward(e *types.ChainEvent, pk string) error
}

// Blockchain ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/blockchain_mock.go -package mocks code.vegaprotocol.io/vega/api  Blockchain
type Blockchain interface {
	SubmitTransaction(ctx context.Context, bundle *types.SignedBundle) (bool, error)
}

type tradingService struct {
	log *logging.Logger

	blockchain        Blockchain
	tradeOrderService TradeOrderService
	liquidityService  LiquidityService
	accountService    AccountService
	marketService     MarketService
	governanceService GovernanceService
	evtForwarder      EvtForwarder

	statusChecker *monitoring.Status
	mu            sync.Mutex
}

func (s *tradingService) PrepareSubmitOrder(ctx context.Context, req *protoapi.SubmitOrderRequest) (*protoapi.PrepareSubmitOrderResponse, error) {
	startTime := time.Now()
	defer metrics.APIRequestAndTimeGRPC("PrepareSubmitOrder", startTime)
	err := s.tradeOrderService.PrepareSubmitOrder(ctx, req.Submission)
	if err != nil {
		return nil, apiError(codes.InvalidArgument, ErrMalformedRequest, err)
	}
	raw, err := proto.Marshal(req.Submission)
	if err != nil {
		return nil, apiError(codes.Internal, ErrSubmitOrder, err)
	}
	if raw, err = txn.Encode(raw, txn.SubmitOrderCommand); err != nil {
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
	if raw, err = txn.Encode(raw, txn.CancelOrderCommand); err != nil {
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
	if raw, err = txn.Encode(raw, txn.AmendOrderCommand); err != nil {
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

	if ok, err := s.blockchain.SubmitTransaction(ctx, req.Tx); err != nil || !ok {
		s.log.Error("unable to submit transaction", logging.Error(err))
		return nil, apiError(codes.Internal, err)
	}

	return &protoapi.SubmitTransactionResponse{
		Success: true,
	}, nil
}

func (s *tradingService) PrepareWithdraw(
	ctx context.Context, req *protoapi.PrepareWithdrawRequest,
) (*protoapi.PrepareWithdrawResponse, error) {
	startTime := time.Now()
	defer metrics.APIRequestAndTimeGRPC("Withdraw", startTime)
	err := s.accountService.PrepareWithdraw(ctx, req.Withdraw)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}
	raw, err := proto.Marshal(req.Withdraw)
	if err != nil {
		return nil, apiError(codes.Internal, ErrPrepareWithdraw, err)
	}
	if raw, err = txn.Encode(raw, txn.WithdrawCommand); err != nil {
		return nil, apiError(codes.Internal, ErrPrepareWithdraw, err)
	}
	return &protoapi.PrepareWithdrawResponse{
		Blob: raw,
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

	if raw, err = txn.Encode(raw, txn.ProposeCommand); err != nil {
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
	if raw, err = txn.Encode(raw, txn.VoteCommand); err != nil {
		return nil, apiError(codes.Internal, ErrPrepareVote, err)
	}
	return &protoapi.PrepareVoteResponse{
		Blob: raw,
		Vote: vote,
	}, nil
}

func (s *tradingService) PrepareLiquidityProvision(ctx context.Context, req *protoapi.PrepareLiquidityProvisionRequest) (*protoapi.PrepareLiquidityProvisionResponse, error) {
	startTime := time.Now()
	defer metrics.APIRequestAndTimeGRPC("PrepareLiquidity", startTime)

	if err := req.Validate(); err != nil {
		return nil, apiError(codes.InvalidArgument, ErrMalformedRequest, err)
	}

	if err := s.liquidityService.PrepareLiquidityProvisionSubmission(ctx, req.Submission); err != nil {
		return nil, apiError(codes.Internal, ErrPrepareVote, err)
	}

	raw, err := proto.Marshal(req.Submission)
	if err != nil {
		return nil, apiError(codes.Internal, ErrPrepareVote, err)
	}

	if raw, err = txn.Encode(raw, txn.LiquidityProvisionCommand); err != nil {
		return nil, apiError(codes.Internal, ErrPrepareVote, err)
	}

	return &protoapi.PrepareLiquidityProvisionResponse{
		Blob: raw,
	}, nil
}

func (s *tradingService) PropagateChainEvent(ctx context.Context, req *protoapi.PropagateChainEventRequest) (*protoapi.PropagateChainEventResponse, error) {
	if req.Evt == nil {
		return nil, apiError(codes.InvalidArgument, ErrMalformedRequest)
	}

	msg, err := req.Evt.PrepareToSign()
	if err != nil {
		return nil, apiError(codes.InvalidArgument, err)
	}

	// verify the signature then
	err = verifySignature(s.log, msg, req.Signature, req.PubKey)
	if err != nil {
		// we try the other signature format
		msg, err = proto.Marshal(req.Evt)
		if err != nil {
			return nil, apiError(codes.InvalidArgument, ErrMalformedRequest)
		}
		if err = verifySignature(s.log, msg, req.Signature, req.PubKey); err != nil {
			s.log.Debug("invalid tx signature", logging.String("pubkey", req.PubKey))
			return nil, apiError(codes.InvalidArgument, ErrMalformedRequest)
		}
	}

	var ok = true
	err = s.evtForwarder.Forward(req.Evt, req.PubKey)
	if err != nil {
		s.log.Error("unable to forward chain event",
			logging.String("pubkey", req.PubKey),
			logging.Error(err))
		if err == evtforward.ErrEvtAlreadyExist {
			return nil, apiError(codes.AlreadyExists, err)
		} else if err == evtforward.ErrPubKeyNotWhitelisted {
			return nil, apiError(codes.PermissionDenied, err)
		} else {
			return nil, apiError(codes.Internal, err)
		}
	}

	return &protoapi.PropagateChainEventResponse{
		Success: ok,
	}, nil
}

func verifySignature(
	log *logging.Logger,
	message []byte,
	sig []byte,
	pubKey string,
) error {
	validator, err := crypto.NewSignatureAlgorithm(crypto.Ed25519)
	if err != nil {
		if log != nil {
			log.Error("unable to instanciate new algorithm", logging.Error(err))
		}
		return err
	}

	pubKeyBytes, err := hex.DecodeString(pubKey)
	if err != nil {
		if log != nil {
			log.Error("unable to decode hexencoded ubkey", logging.Error(err))
		}
		return err
	}
	ok, err := validator.Verify(pubKeyBytes, message, sig)
	if err != nil {
		if log != nil {
			log.Error("unable to verify bundle", logging.Error(err))
		}
		return err
	}
	if !ok {
		return ErrInvalidSignature
	}
	return nil
}
