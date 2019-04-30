package api

import (
	"context"

	"code.vegaprotocol.io/vega/internal/monitoring"
	types "code.vegaprotocol.io/vega/proto"
	protoapi "code.vegaprotocol.io/vega/proto/api"
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/trade_order_service_mock.go -package mocks code.vegaprotocol.io/vega/internal/api/endpoints/grpc TradeOrderService
type TradeOrderService interface {
	CreateOrder(ctx context.Context, order *types.OrderSubmission) (*types.PendingOrder, error)
	CancelOrder(ctx context.Context, order *types.OrderCancellation) (*types.PendingOrder, error)
	AmendOrder(ctx context.Context, amendment *types.OrderAmendment) (success bool, err error)
}

type tradingService struct {
	tradeOrderService TradeOrderService
	statusChecker     *monitoring.Status
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
