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

package gql

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"code.vegaprotocol.io/vega/datanode/gateway"
	"code.vegaprotocol.io/vega/logging"
	protoapi "code.vegaprotocol.io/vega/protos/data-node/api/v1"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	types "code.vegaprotocol.io/vega/protos/vega"
)

type allResolver struct {
	log  *logging.Logger
	clt  TradingDataServiceClient
	clt2 TradingDataServiceClientV2
}

func (r *allResolver) getEpochByID(ctx context.Context, id uint64) (*types.Epoch, error) {
	req := &v2.GetEpochRequest{
		Id: &id,
	}
	header := metadata.MD{}
	resp, err := r.clt2.GetEpoch(ctx, req, grpc.Header(&header))
	if err != nil {
		return nil, err
	}
	if err = gateway.AddMDHeadersToContext(ctx, header); err != nil {
		r.log.Error("failed to add headers to context", logging.Error(err))
	}

	return resp.Epoch, nil
}

func (r *allResolver) getOrderByID(ctx context.Context, id string, version *int) (*types.Order, error) {
	v, err := convertVersion(version)
	if err != nil {
		r.log.Error("tradingCore client", logging.Error(err))
		return nil, customErrorFromStatus(err)
	}
	orderReq := &v2.GetOrderRequest{
		OrderId: id,
		Version: &v,
	}
	header := metadata.MD{}
	order, err := r.clt2.GetOrder(ctx, orderReq, grpc.Header(&header))
	if err != nil {
		return nil, err
	}

	if err = gateway.AddMDHeadersToContext(ctx, header); err != nil {
		r.log.Error("failed to add headers to context", logging.Error(err))
	}

	return order.Order, nil
}

func (r *allResolver) getAssetByID(ctx context.Context, id string) (*types.Asset, error) {
	if len(id) <= 0 {
		return nil, ErrMissingIDOrReference
	}
	req := &protoapi.AssetByIDRequest{
		Id: id,
	}
	header := metadata.MD{}
	res, err := r.clt.AssetByID(ctx, req, grpc.Header(&header))
	if err != nil {
		return nil, err
	}
	if err = gateway.AddMDHeadersToContext(ctx, header); err != nil {
		r.log.Error("failed to add headers to context", logging.Error(err))
	}

	return res.Asset, nil
}

func (r *allResolver) getNodeByID(ctx context.Context, id string) (*types.Node, error) {
	if len(id) <= 0 {
		return nil, ErrMissingNodeID
	}
	header := metadata.MD{}
	resp, err := r.clt.GetNodeByID(ctx, &protoapi.GetNodeByIDRequest{Id: id}, grpc.Header(&header))
	if err != nil {
		return nil, err
	}

	if err = gateway.AddMDHeadersToContext(ctx, header); err != nil {
		r.log.Error("failed to add headers to context", logging.Error(err))
	}

	return resp.Node, nil
}

func (r allResolver) allAssets(ctx context.Context) ([]*types.Asset, error) {
	req := &protoapi.AssetsRequest{}
	header := metadata.MD{}

	res, err := r.clt.Assets(ctx, req, grpc.Header(&header))
	if err != nil {
		return nil, err
	}
	if err = gateway.AddMDHeadersToContext(ctx, header); err != nil {
		r.log.Error("failed to add headers to context", logging.Error(err))
	}

	return res.Assets, nil
}

func (r *allResolver) getMarketByID(ctx context.Context, id string) (*types.Market, error) {
	req := v2.GetMarketRequest{MarketId: id}
	header := metadata.MD{}

	res, err := r.clt2.GetMarket(ctx, &req, grpc.Header(&header))
	if err != nil {
		r.log.Error("tradingData client", logging.Error(err))
		return nil, customErrorFromStatus(err)
	}
	// no error / no market = we did not find it
	if res.Market == nil {
		return nil, nil
	}
	if err = gateway.AddMDHeadersToContext(ctx, header); err != nil {
		r.log.Error("failed to add headers to context", logging.Error(err))
	}

	return res.Market, nil
}

func (r *allResolver) allMarkets(ctx context.Context, id *string) ([]*types.Market, error) {
	if id != nil {
		mkt, err := r.getMarketByID(ctx, *id)
		if err != nil {
			return nil, err
		}
		if mkt == nil {
			return []*types.Market{}, nil
		}
		return []*types.Market{mkt}, nil
	}
	header := metadata.MD{}
	res, err := r.clt.Markets(ctx, &protoapi.MarketsRequest{}, grpc.Header(&header))
	if err != nil {
		r.log.Error("tradingData client", logging.Error(err))
		return nil, customErrorFromStatus(err)
	}
	if err = gateway.AddMDHeadersToContext(ctx, header); err != nil {
		r.log.Error("failed to add headers to context", logging.Error(err))
	}

	return res.Markets, nil
}

func (r *allResolver) allRewards(ctx context.Context, partyID, assetID string, skip, first, last *int) ([]*types.Reward, error) {
	req := &protoapi.GetRewardsRequest{
		PartyId:    partyID,
		AssetId:    assetID,
		Pagination: makePagination(skip, first, last),
	}
	header := metadata.MD{}
	resp, err := r.clt.GetRewards(ctx, req, grpc.Header(&header))
	if err != nil {
		return nil, err
	}

	if err = gateway.AddMDHeadersToContext(ctx, header); err != nil {
		r.log.Error("failed to add headers to context", logging.Error(err))
	}

	return resp.Rewards, nil
}

func (r *allResolver) transfersConnection(
	ctx context.Context,
	partyID *string,
	direction *TransferDirection,
	pagination *v2.Pagination,
) (*v2.TransferConnection, error) {
	// if direction is nil just default to ToOrFrom
	if direction == nil {
		d := TransferDirectionToOrFrom
		direction = &d
	}

	var transferDirection v2.TransferDirection
	switch *direction {
	case TransferDirectionFrom:
		transferDirection = v2.TransferDirection_TRANSFER_DIRECTION_TRANSFER_FROM
	case TransferDirectionTo:
		transferDirection = v2.TransferDirection_TRANSFER_DIRECTION_TRANSFER_TO
	case TransferDirectionToOrFrom:
		transferDirection = v2.TransferDirection_TRANSFER_DIRECTION_TRANSFER_TO_OR_FROM
	}

	header := metadata.MD{}
	res, err := r.clt2.ListTransfers(ctx, &v2.ListTransfersRequest{
		Pubkey:     partyID,
		Direction:  transferDirection,
		Pagination: pagination,
	}, grpc.Header(&header))
	if err != nil {
		return nil, err
	}

	if err = gateway.AddMDHeadersToContext(ctx, header); err != nil {
		r.log.Error("failed to add headers to context", logging.Error(err))
	}

	return res.Transfers, nil
}
