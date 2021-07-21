package api

import (
	"context"
	"time"

	"code.vegaprotocol.io/data-node/logging"
	protoapiv1 "code.vegaprotocol.io/data-node/proto/api/v1"
	protocommandsv1 "code.vegaprotocol.io/data-node/proto/commands/v1"
	coreprotoapi "code.vegaprotocol.io/vega/proto/api"
	coreprotocommandsv1 "code.vegaprotocol.io/vega/proto/commands/v1"
)

const defaultRequestTimeout = time.Second * 5

// TradingServiceClient ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/trading_service_client_mock.go -package mocks code.vegaprotocol.io/data-node/api TradingServiceClient
type TradingServiceClient = coreprotoapi.TradingServiceClient

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

	coreReq := internalToCoreSubmitTransactionRequest(req)
	coreResp, err := s.tradingServiceClient.SubmitTransactionV2(ctx, coreReq)
	if err != nil {
		return nil, err
	}

	return coreToInternalSubmitTransactionResponse(coreResp), nil
}

func coreToInternalSubmitTransactionResponse(res *coreprotoapi.SubmitTransactionV2Response) *protoapiv1.SubmitTransactionResponse {
	return (*protoapiv1.SubmitTransactionResponse)(res)
}

func internalToCoreSubmitTransactionRequest(req *protoapiv1.SubmitTransactionRequest) *coreprotoapi.SubmitTransactionV2Request {
	requestType, ok := internalToCoreTransactionRequestType[req.Type]
	if !ok {
		requestType = coreprotoapi.SubmitTransactionV2Request_TYPE_UNSPECIFIED
	}

	return &coreprotoapi.SubmitTransactionV2Request{
		Tx:   internalToCoreTransacation(req.Tx),
		Type: requestType,
	}
}

func internalToCoreTransacation(tx *protocommandsv1.Transaction) *coreprotocommandsv1.Transaction {
	coreTx := &coreprotocommandsv1.Transaction{
		InputData: tx.InputData,
		Signature: (*coreprotocommandsv1.Signature)(tx.Signature),
		Version:   tx.Version,
	}

	switch from := tx.From.(type) {
	case *protocommandsv1.Transaction_Address:
		coreTx.From = &coreprotocommandsv1.Transaction_Address{Address: from.Address}
	case *protocommandsv1.Transaction_PubKey:
		coreTx.From = &coreprotocommandsv1.Transaction_PubKey{PubKey: from.PubKey}
	}

	return coreTx
}

var internalToCoreTransactionRequestType = map[protoapiv1.SubmitTransactionRequest_Type]coreprotoapi.SubmitTransactionV2Request_Type{
	protoapiv1.SubmitTransactionRequest_TYPE_UNSPECIFIED: coreprotoapi.SubmitTransactionV2Request_TYPE_UNSPECIFIED,
	protoapiv1.SubmitTransactionRequest_TYPE_ASYNC:       coreprotoapi.SubmitTransactionV2Request_TYPE_ASYNC,
	protoapiv1.SubmitTransactionRequest_TYPE_SYNC:        coreprotoapi.SubmitTransactionV2Request_TYPE_SYNC,
	protoapiv1.SubmitTransactionRequest_TYPE_COMMIT:      coreprotoapi.SubmitTransactionV2Request_TYPE_COMMIT,
}
