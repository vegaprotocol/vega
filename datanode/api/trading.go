// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package api

import (
	"context"
	"time"

	"code.vegaprotocol.io/vega/datanode/metrics"
	protoapi "code.vegaprotocol.io/vega/protos/vega/api/v1"

	"github.com/pkg/errors"
)

const defaultRequestTimeout = time.Second * 5

// CoreServiceClient ...
//
//go:generate go run github.com/golang/mock/mockgen -destination mocks/core_service_client_mock.go -package mocks code.vegaprotocol.io/vega/datanode/api CoreServiceClient
type CoreServiceClient interface {
	protoapi.CoreServiceClient
}

// core service acts as a proxy to the trading service in core node.
type coreProxyService struct {
	protoapi.UnimplementedCoreServiceServer
	conf Config

	coreServiceClient CoreServiceClient
	eventObserver     *eventObserver
}

func (t *coreProxyService) SubmitTransaction(ctx context.Context, req *protoapi.SubmitTransactionRequest) (*protoapi.SubmitTransactionResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultRequestTimeout)
	defer cancel()

	return t.coreServiceClient.SubmitTransaction(ctx, req)
}

func (t *coreProxyService) SubmitRawTransaction(ctx context.Context, req *protoapi.SubmitRawTransactionRequest) (*protoapi.SubmitRawTransactionResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultRequestTimeout)
	defer cancel()

	return t.coreServiceClient.SubmitRawTransaction(ctx, req)
}

func (t *coreProxyService) CheckTransaction(ctx context.Context, req *protoapi.CheckTransactionRequest) (*protoapi.CheckTransactionResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultRequestTimeout)
	defer cancel()

	return t.coreServiceClient.CheckTransaction(ctx, req)
}

func (t *coreProxyService) CheckRawTransaction(ctx context.Context, req *protoapi.CheckRawTransactionRequest) (*protoapi.CheckRawTransactionResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultRequestTimeout)
	defer cancel()

	return t.coreServiceClient.CheckRawTransaction(ctx, req)
}

func (t *coreProxyService) LastBlockHeight(ctx context.Context, req *protoapi.LastBlockHeightRequest) (*protoapi.LastBlockHeightResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultRequestTimeout)
	defer cancel()

	return t.coreServiceClient.LastBlockHeight(ctx, req)
}

func (t *coreProxyService) GetVegaTime(ctx context.Context, req *protoapi.GetVegaTimeRequest) (*protoapi.GetVegaTimeResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultRequestTimeout)
	defer cancel()

	return t.coreServiceClient.GetVegaTime(ctx, req)
}

func (t *coreProxyService) Statistics(ctx context.Context, req *protoapi.StatisticsRequest) (*protoapi.StatisticsResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultRequestTimeout)
	defer cancel()

	return t.coreServiceClient.Statistics(ctx, req)
}

func (t *coreProxyService) ObserveEventBus(
	stream protoapi.CoreService_ObserveEventBusServer,
) error {
	defer metrics.StartActiveSubscriptionCountGRPC("EventBus")()
	return t.eventObserver.ObserveEventBus(stream)
}

func (t *coreProxyService) PropagateChainEvent(ctx context.Context, req *protoapi.PropagateChainEventRequest) (*protoapi.PropagateChainEventResponse, error) {
	return nil, errors.New("unimplemented")
}

func (t *coreProxyService) GetSpamStatistics(ctx context.Context, in *protoapi.GetSpamStatisticsRequest) (*protoapi.GetSpamStatisticsResponse, error) {
	defer metrics.StartActiveSubscriptionCountGRPC("GetSpamStatistics")()
	return t.coreServiceClient.GetSpamStatistics(ctx, in)
}
