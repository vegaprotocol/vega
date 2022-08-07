// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.DATANODE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package api

import (
	"context"
	"time"

	"code.vegaprotocol.io/vega/datanode/metrics"
	"code.vegaprotocol.io/vega/logging"
	protoapi "code.vegaprotocol.io/vega/protos/vega/api/v1"
	"github.com/pkg/errors"
)

const defaultRequestTimeout = time.Second * 5

// CoreServiceClient ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/core_service_client_mock.go -package mocks code.vegaprotocol.io/vega/datanode/api CoreServiceClient
type CoreServiceClient interface {
	protoapi.CoreServiceClient
}

// core service acts as a proxy to the trading service in core node.
type coreProxyService struct {
	protoapi.UnimplementedCoreServiceServer
	log  *logging.Logger
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
