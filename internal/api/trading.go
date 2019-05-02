package api

import (
	"context"
	"sync"

	"code.vegaprotocol.io/vega/internal/auth"
	"code.vegaprotocol.io/vega/internal/monitoring"
	types "code.vegaprotocol.io/vega/proto"
	protoapi "code.vegaprotocol.io/vega/proto/api"
	"github.com/pkg/errors"
)

var (
	ErrAuthDisabled       = errors.New("auth disabled")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrAuthRequired       = errors.New("auth required")
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/trade_order_service_mock.go -package mocks code.vegaprotocol.io/vega/internal/api TradeOrderService
type TradeOrderService interface {
	CreateOrder(ctx context.Context, order *types.OrderSubmission) (*types.PendingOrder, error)
	CancelOrder(ctx context.Context, order *types.OrderCancellation) (*types.PendingOrder, error)
	AmendOrder(ctx context.Context, amendment *types.OrderAmendment) (success bool, err error)
}

type tradingService struct {
	tradeOrderService TradeOrderService
	statusChecker     *monitoring.Status

	parties []auth.PartyInfo
	mu      sync.Mutex
}

func (s tradingService) UpdateParties(parties []auth.PartyInfo) {
	s.mu.Lock()
	s.parties = parties
	s.mu.Unlock()
}

func (s *tradingService) validateToken(tkn string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, v := range s.parties {
		if v.Token == tkn {
			return nil
		}
	}
	return ErrInvalidCredentials
}

func (s *tradingService) SignIn(
	ctx context.Context, req protoapi.SignInRequest,
) (*protoapi.SignInResponse, error) {
	if len(req.Username) <= 0 {
		return errors.New("missing username")
	}
	if len(req.Password) <= 0 {
		return errors.New("missing password")
	}

	var tkn string

	s.mu.Lock()
	if !s.AuthEnabled {
		s.mu.Unlock()
		return ErrAuthDisabled
	}
	for _, v := range s.parties {
		if v.Username == req.Username && v.Password == req.Password {
			tkn = v.Token
		}
	}
	s.mu.Unlock()

	if len(tkn) <= 0 {
		return ErrInvalidCredentials
	}

	return &protoapi.SignInResponse{
		Token: tkn,
	}, nil
}

// CreateOrder is used to request sending an order into the VEGA platform, via consensus.
func (s *tradingService) SubmitOrder(
	ctx context.Context, order *types.OrderSubmission,
) (*types.PendingOrder, error) {
	if s.statusChecker.ChainStatus() != types.ChainStatus_CONNECTED {
		return nil, ErrChainNotConnected
	}
	pendingOrder, err := s.tradeOrderService.CreateOrder(ctx, order)
	return pendingOrder, err
}

// CancelOrder is used to request cancelling an order into the VEGA platform, via consensus.
func (s *tradingService) CancelOrder(
	ctx context.Context, order *types.OrderCancellation,
) (*types.PendingOrder, error) {
	if s.statusChecker.ChainStatus() != types.ChainStatus_CONNECTED {
		return nil, ErrChainNotConnected
	}
	pendingOrder, err := s.tradeOrderService.CancelOrder(ctx, order)
	return pendingOrder, err
}

// AmendOrder is used to request editing an order onto the VEGA platform, via consensus.
func (s *tradingService) AmendOrder(
	ctx context.Context, amendment *types.OrderAmendment,
) (*protoapi.OrderResponse, error) {
	success, err := s.tradeOrderService.AmendOrder(ctx, amendment)
	return &protoapi.OrderResponse{Success: success}, err
}
