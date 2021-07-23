package api

import (
	"context"
	"time"

	"code.vegaprotocol.io/data-node/logging"
	protoapiv1 "code.vegaprotocol.io/data-node/proto/api/v1"
	vegaprotoapi "code.vegaprotocol.io/data-node/proto/vega/api"
)

const defaultRequestTimeout = time.Second * 5

// TradingServiceClient ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/trading_service_client_mock.go -package mocks code.vegaprotocol.io/data-node/api TradingServiceClient
type TradingServiceClient interface {
	vegaprotoapi.TradingServiceClient
}

// trading service acts as a proxy to the trading service in core node
type tradingProxyService struct {
	log  *logging.Logger
	conf Config

	tradingServiceClient TradingServiceClient
}

// no need for a mutext - we only access the config through a value receiver
func (s *tradingProxyService) updateConfig(conf Config) {
	s.conf = conf
}

func (s *tradingProxyService) SubmitTransaction(ctx context.Context, req *protoapiv1.SubmitTransactionRequest) (*protoapiv1.SubmitTransactionResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultRequestTimeout)
	defer cancel()

	vegaReq := internalToCoreSubmitTransactionRequest(req)
	vegaResp, err := s.tradingServiceClient.SubmitTransactionV2(ctx, vegaReq)
	if err != nil {
		return nil, err
	}

	return &protoapiv1.SubmitTransactionResponse{
		Success: vegaResp.Success,
	}, nil
}

func internalToCoreSubmitTransactionRequest(req *protoapiv1.SubmitTransactionRequest) *vegaprotoapi.SubmitTransactionV2Request {
	requestType, ok := internalToCoreTransactionRequestType[req.Type]
	if !ok {
		requestType = vegaprotoapi.SubmitTransactionV2Request_TYPE_UNSPECIFIED
	}

	return &vegaprotoapi.SubmitTransactionV2Request{
		Tx:   req.Tx,
		Type: requestType,
	}
}

var internalToCoreTransactionRequestType = map[protoapiv1.SubmitTransactionRequest_Type]vegaprotoapi.SubmitTransactionV2Request_Type{
	protoapiv1.SubmitTransactionRequest_TYPE_UNSPECIFIED: vegaprotoapi.SubmitTransactionV2Request_TYPE_UNSPECIFIED,
	protoapiv1.SubmitTransactionRequest_TYPE_ASYNC:       vegaprotoapi.SubmitTransactionV2Request_TYPE_ASYNC,
	protoapiv1.SubmitTransactionRequest_TYPE_SYNC:        vegaprotoapi.SubmitTransactionV2Request_TYPE_SYNC,
	protoapiv1.SubmitTransactionRequest_TYPE_COMMIT:      vegaprotoapi.SubmitTransactionV2Request_TYPE_COMMIT,
}
