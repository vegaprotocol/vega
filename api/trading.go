package api

import (
	"context"
	"fmt"
	"sync"

	"code.vegaprotocol.io/vega/auth"
	"code.vegaprotocol.io/vega/blockchain"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/monitoring"
	types "code.vegaprotocol.io/vega/proto"
	protoapi "code.vegaprotocol.io/vega/proto/api"

	"github.com/golang/protobuf/proto"
	uuid "github.com/satori/go.uuid"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
)

// TradeOrderService ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/trade_order_service_mock.go -package mocks code.vegaprotocol.io/vega/api TradeOrderService
type TradeOrderService interface {
	PrepareSubmitOrder(ctx context.Context, submission *types.OrderSubmission) (*types.PendingOrder, error)
	PrepareCancelOrder(ctx context.Context, cancellation *types.OrderCancellation) (*types.PendingOrder, error)
	PrepareAmendOrder(ctx context.Context, amendment *types.OrderAmendment) (*types.PendingOrder, error)
	SubmitTransaction(ctx context.Context, bundle *types.SignedBundle) (bool, error)
	CreateOrder(ctx context.Context, submission *types.OrderSubmission) (*types.PendingOrder, error)
	CancelOrder(ctx context.Context, cancellation *types.OrderCancellation) (*types.PendingOrder, error)
	AmendOrder(ctx context.Context, amendment *types.OrderAmendment) (*types.PendingOrder, error)
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
	PrepareProposal(ctx context.Context, author, reference string, terms *types.Proposal_Terms) (*types.Proposal, error)
}

type tradingService struct {
	log               *logging.Logger
	tradeOrderService TradeOrderService
	accountService    AccountService
	marketService     MarketService
	governanceService GovernanceService
	statusChecker     *monitoring.Status

	authEnabled bool
	parties     []auth.PartyInfo
	mu          sync.Mutex
}

func (s *tradingService) UpdateParties(parties []auth.PartyInfo) {
	s.mu.Lock()
	s.parties = parties
	s.mu.Unlock()
}

func (s *tradingService) validateToken(partyID string, tkn string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, v := range s.parties {
		if v.ID == partyID && v.Token == tkn {
			return nil
		}
	}
	return ErrInvalidCredentials
}

func (s *tradingService) CheckToken(
	ctx context.Context, req *protoapi.CheckTokenRequest,
) (*protoapi.CheckTokenResponse, error) {
	if req == nil {
		return nil, apiError(codes.Internal, ErrMalformedRequest)
	}
	if !s.authEnabled {
		return nil, apiError(codes.InvalidArgument, ErrAuthDisabled)
	}
	if len(req.PartyID) <= 0 {
		return nil, apiError(codes.InvalidArgument, ErrMissingPartyID)
	}
	if len(req.Token) <= 0 {
		return nil, apiError(codes.InvalidArgument, ErrMissingToken)
	}

	err := s.validateToken(req.PartyID, req.Token)
	if err != nil {
		if err == ErrInvalidCredentials {
			return &protoapi.CheckTokenResponse{Ok: false}, nil
		}
		return nil, apiError(codes.Internal, err)
	}

	return &protoapi.CheckTokenResponse{Ok: true}, nil
}

func (s *tradingService) SignIn(
	ctx context.Context, req *protoapi.SignInRequest,
) (*protoapi.SignInResponse, error) {
	if req == nil {
		return nil, apiError(codes.Internal, ErrMalformedRequest)
	}
	if len(req.Id) <= 0 {
		return nil, apiError(codes.PermissionDenied, ErrInvalidCredentials)
	}
	if len(req.Password) <= 0 {
		return nil, apiError(codes.PermissionDenied, ErrInvalidCredentials)
	}

	var tkn string
	saltpass := fmt.Sprintf("vega%v", req.Password)

	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.authEnabled {
		return nil, apiError(codes.PermissionDenied, ErrAuthDisabled)
	}
	for _, v := range s.parties {
		if v.ID == req.Id {
			if err := bcrypt.CompareHashAndPassword([]byte(v.Password), []byte(saltpass)); err != nil {
				s.log.Debug("invalid password",
					logging.String("user-id", v.ID),
					logging.Error(err),
				)
				return nil, apiError(codes.PermissionDenied, ErrInvalidCredentials)
			}
			tkn = v.Token
		}
	}

	if len(tkn) <= 0 {
		return nil, apiError(codes.PermissionDenied, ErrInvalidCredentials)
	}

	return &protoapi.SignInResponse{
		Token: tkn,
	}, nil
}

func (s *tradingService) PrepareSubmitOrder(ctx context.Context, req *protoapi.SubmitOrderRequest) (*protoapi.PrepareSubmitOrderResponse, error) {
	pending, err := s.tradeOrderService.PrepareSubmitOrder(ctx, req.Submission)
	if err != nil {
		return nil, apiError(codes.Internal, ErrMalformedRequest)
	}
	raw, err := proto.Marshal(req.Submission)
	if err != nil {
		return nil, apiError(codes.Internal, ErrSubmitOrder)
	}
	if raw, err = txEncode(raw, blockchain.SubmitOrderCommand); err != nil {
		return nil, apiError(codes.Internal, ErrSubmitOrder)
	}
	return &protoapi.PrepareSubmitOrderResponse{
		Blob:         raw,
		PendingOrder: pending,
	}, nil
}

func (s *tradingService) PrepareCancelOrder(ctx context.Context, req *protoapi.CancelOrderRequest) (*protoapi.PrepareCancelOrderResponse, error) {
	pending, err := s.tradeOrderService.PrepareCancelOrder(ctx, req.Cancellation)
	if err != nil {
		return nil, apiError(codes.Internal, ErrCancelOrder)
	}
	raw, err := proto.Marshal(req.Cancellation)
	if err != nil {
		return nil, apiError(codes.Internal, ErrCancelOrder)
	}
	if raw, err = txEncode(raw, blockchain.CancelOrderCommand); err != nil {
		return nil, apiError(codes.Internal, ErrCancelOrder)
	}
	return &protoapi.PrepareCancelOrderResponse{
		Blob:         raw,
		PendingOrder: pending,
	}, nil
}

func (s *tradingService) PrepareAmendOrder(ctx context.Context, req *protoapi.AmendOrderRequest) (*protoapi.PrepareAmendOrderResponse, error) {
	pending, err := s.tradeOrderService.PrepareAmendOrder(ctx, req.Amendment)
	if err != nil {
		return nil, apiError(codes.Internal, ErrAmendOrder)
	}
	raw, err := proto.Marshal(req.Amendment)
	if err != nil {
		return nil, apiError(codes.Internal, ErrAmendOrder)
	}
	if raw, err = txEncode(raw, blockchain.AmendOrderCommand); err != nil {
		return nil, apiError(codes.Internal, ErrAmendOrder)
	}
	return &protoapi.PrepareAmendOrderResponse{
		Blob:         raw,
		PendingOrder: pending,
	}, nil
}

func (s *tradingService) SubmitTransaction(ctx context.Context, req *protoapi.SubmitTransactionRequest) (*protoapi.SubmitTransactionResponse, error) {
	if ok, err := s.tradeOrderService.SubmitTransaction(ctx, req.Tx); err != nil || !ok {
		s.log.Error("unable to submit transaction", logging.Error(err))
		return nil, apiError(codes.Internal, err)
	}
	return &protoapi.SubmitTransactionResponse{
		Success: true,
	}, nil
}

// CreateOrder is used to request sending an order into the VEGA platform, via consensus.
func (s *tradingService) SubmitOrder(
	ctx context.Context, req *protoapi.SubmitOrderRequest,
) (*types.PendingOrder, error) {
	if req == nil {
		return nil, apiError(codes.Internal, ErrMalformedRequest)
	}
	if s.statusChecker.ChainStatus() != types.ChainStatus_CONNECTED {
		return nil, apiError(codes.Internal, ErrChainNotConnected)
	}
	if req.Submission == nil {
		return nil, apiError(codes.InvalidArgument, ErrMissingOrder)
	}

	// check auth if required
	if s.authEnabled {
		if len(req.Token) <= 0 {
			return nil, apiError(codes.PermissionDenied, ErrMissingToken)
		}
		if err := s.validateToken(req.Submission.PartyID, req.Token); err != nil {
			return nil, apiError(codes.PermissionDenied, ErrInvalidToken, err)
		}
	}

	// Validate market early
	_, err := s.marketService.GetByID(ctx, req.Submission.MarketID)
	if err != nil {
		s.log.Error("Invalid Market ID during SubmitOrder",
			logging.String("marketID", req.Submission.MarketID),
		)
		return nil, apiError(codes.Internal, ErrInvalidMarketID)
	}

	po, err := s.tradeOrderService.CreateOrder(ctx, req.Submission)
	if err != nil {
		return nil, apiError(codes.Internal, ErrSubmitOrder, err)
	}
	return po, nil
}

// CancelOrder is used to request cancelling an order into the VEGA platform, via consensus.
func (s *tradingService) CancelOrder(
	ctx context.Context, req *protoapi.CancelOrderRequest,
) (*types.PendingOrder, error) {
	if req == nil {
		return nil, apiError(codes.Internal, ErrMalformedRequest)
	}
	if s.statusChecker.ChainStatus() != types.ChainStatus_CONNECTED {
		return nil, apiError(codes.Internal, ErrChainNotConnected)
	}
	if req.Cancellation == nil {
		return nil, apiError(codes.InvalidArgument, ErrMissingOrder)
	}

	// check auth if required
	if s.authEnabled {
		if len(req.Token) <= 0 {
			return nil, apiError(codes.PermissionDenied, ErrMissingToken)
		}
		if err := s.validateToken(req.Cancellation.PartyID, req.Token); err != nil {
			return nil, apiError(codes.PermissionDenied, ErrInvalidToken, err)
		}
	}

	po, err := s.tradeOrderService.CancelOrder(ctx, req.Cancellation)
	if err != nil {
		return nil, apiError(codes.Internal, ErrCancelOrder, err)
	}
	return po, nil
}

// AmendOrder is used to request editing an order onto the VEGA platform, via consensus.
func (s *tradingService) AmendOrder(
	ctx context.Context, req *protoapi.AmendOrderRequest,
) (*types.PendingOrder, error) {
	if req == nil {
		return nil, apiError(codes.Internal, ErrMalformedRequest)
	}
	if req.Amendment == nil {
		return nil, apiError(codes.InvalidArgument, ErrMissingOrder)
	}

	// check auth if required
	if s.authEnabled {
		if len(req.Token) <= 0 {
			return nil, apiError(codes.PermissionDenied, ErrMissingToken)
		}
		if err := s.validateToken(req.Amendment.PartyID, req.Token); err != nil {
			return nil, apiError(codes.PermissionDenied, ErrInvalidToken, err)
		}
	}

	po, err := s.tradeOrderService.AmendOrder(ctx, req.Amendment)
	if err != nil {
		return nil, apiError(codes.Internal, ErrAmendOrder, err)
	}
	return po, nil
}

func (s *tradingService) NotifyTraderAccount(
	ctx context.Context, req *protoapi.NotifyTraderAccountRequest,
) (*protoapi.NotifyTraderAccountResponse, error) {
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
	proposal, err := s.governanceService.PrepareProposal(ctx,
		req.PartyID, req.Reference, req.Proposal)
	if err != nil {
		return nil, apiError(codes.Internal, ErrMalformedRequest)
	}
	raw, err := proto.Marshal(proposal) // marshal whole proposal
	if err != nil {
		return nil, apiError(codes.Internal, ErrPrepareProposal)
	}
	if raw, err = txEncode(raw, blockchain.NewProposalCommand); err != nil {
		return nil, apiError(codes.Internal, ErrPrepareProposal)
	}
	return &protoapi.PrepareProposalResponse{
		Blob:            raw,
		PendingProposal: proposal,
	}, nil
}

func txEncode(input []byte, cmd blockchain.Command) (proto []byte, err error) {
	prefix := uuid.NewV4().String()
	prefixBytes := []byte(prefix)
	commandInput := append([]byte{byte(cmd)}, input...)
	return append(prefixBytes, commandInput...), nil
}
