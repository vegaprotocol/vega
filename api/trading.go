package api

import (
	"context"
	"time"

	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/data-node/metrics"
	protoapi "code.vegaprotocol.io/protos/vega/api"
	"github.com/pkg/errors"
)

const defaultRequestTimeout = time.Second * 5

// TradingServiceClient ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/trading_service_client_mock.go -package mocks code.vegaprotocol.io/data-node/api TradingServiceClient
type TradingServiceClient interface {
	protoapi.TradingServiceClient
}

// trading service acts as a proxy to the trading service in core node
type tradingProxyService struct {
	log  *logging.Logger
	conf Config

	tradingServiceClient TradingServiceClient
	eventObserver        *eventObserver
}

func (t *tradingProxyService) SubmitTransaction(ctx context.Context, req *protoapi.SubmitTransactionRequest) (*protoapi.SubmitTransactionResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultRequestTimeout)
	defer cancel()

	return t.tradingServiceClient.SubmitTransaction(ctx, req)
}

func (t *tradingProxyService) LastBlockHeight(ctx context.Context, req *protoapi.LastBlockHeightRequest) (*protoapi.LastBlockHeightResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultRequestTimeout)
	defer cancel()

	return t.tradingServiceClient.LastBlockHeight(ctx, req)
}

func (t *tradingProxyService) GetVegaTime(ctx context.Context, req *protoapi.GetVegaTimeRequest) (*protoapi.GetVegaTimeResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultRequestTimeout)
	defer cancel()

	return t.tradingServiceClient.GetVegaTime(ctx, req)
}

func (t *tradingProxyService) Statistics(ctx context.Context, req *protoapi.StatisticsRequest) (*protoapi.StatisticsResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultRequestTimeout)
	defer cancel()

	return t.tradingServiceClient.Statistics(ctx, req)
}

func (t *tradingProxyService) ObserveEventBus(
	stream protoapi.TradingService_ObserveEventBusServer) error {
	defer metrics.StartAPIRequestAndTimeGRPC("ObserveEventBus")()
	return t.eventObserver.ObserveEventBus(stream)
}

func (t *tradingProxyService) PropagateChainEvent(ctx context.Context, req *protoapi.PropagateChainEventRequest) (*protoapi.PropagateChainEventResponse, error) {
	return nil, errors.New("unimplemented")
}
