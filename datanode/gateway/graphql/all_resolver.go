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
	resp, err := r.clt2.GetEpoch(ctx, req)
	return resp.Epoch, err
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
	order, err := r.clt2.GetOrder(ctx, orderReq)
	if err != nil {
		return nil, err
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
	res, err := r.clt.AssetByID(ctx, req)
	if err != nil {
		return nil, err
	}
	return res.Asset, nil
}

func (r *allResolver) getNodeByID(ctx context.Context, id string) (*types.Node, error) {
	if len(id) <= 0 {
		return nil, ErrMissingNodeID
	}
	resp, err := r.clt.GetNodeByID(
		ctx, &protoapi.GetNodeByIDRequest{Id: id})
	if err != nil {
		return nil, err
	}

	return resp.Node, nil
}

func (r allResolver) allAssets(ctx context.Context) ([]*types.Asset, error) {
	req := &protoapi.AssetsRequest{}
	res, err := r.clt.Assets(ctx, req)
	if err != nil {
		return nil, err
	}
	return res.Assets, nil
}

func (r *allResolver) getMarketByID(ctx context.Context, id string) (*types.Market, error) {
	req := v2.GetMarketRequest{MarketId: id}
	res, err := r.clt2.GetMarket(ctx, &req)
	if err != nil {
		r.log.Error("tradingData client", logging.Error(err))
		return nil, customErrorFromStatus(err)
	}
	// no error / no market = we did not find it
	if res.Market == nil {
		return nil, nil
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
	res, err := r.clt.Markets(ctx, &protoapi.MarketsRequest{})
	if err != nil {
		r.log.Error("tradingData client", logging.Error(err))
		return nil, customErrorFromStatus(err)
	}
	return res.Markets, nil
}

func (r *allResolver) allRewards(ctx context.Context, partyID, assetID string, skip, first, last *int) ([]*types.Reward, error) {
	req := &protoapi.GetRewardsRequest{
		PartyId:    partyID,
		AssetId:    assetID,
		Pagination: makePagination(skip, first, last),
	}
	resp, err := r.clt.GetRewards(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp.Rewards, nil
}

func (r *allResolver) transferInstructionsConnection(
	ctx context.Context,
	partyID *string,
	direction *TransferInstructionDirection,
	pagination *v2.Pagination,
) (*v2.TransferInstructionConnection, error) {
	// if direction is nil just default to ToOrFrom
	if direction == nil {
		d := TransferInstructionDirectionToOrFrom
		direction = &d
	}

	var transferDirection v2.TransferInstructionDirection
	switch *direction {
	case TransferInstructionDirectionFrom:
		transferDirection = v2.TransferInstructionDirection_TRANSFER_INSTRUCTION_DIRECTION_TRANSFER_FROM
	case TransferInstructionDirectionTo:
		transferDirection = v2.TransferInstructionDirection_TRANSFER_INSTRUCTION_DIRECTION_TRANSFER_TO
	case TransferInstructionDirectionToOrFrom:
		transferDirection = v2.TransferInstructionDirection_TRANSFER_INSTRUCTION_DIRECTION_TRANSFER_TO_OR_FROM
	}

	res, err := r.clt2.ListTransferInstructions(ctx, &v2.ListTransferInstructionsRequest{
		Pubkey:     partyID,
		Direction:  transferDirection,
		Pagination: pagination,
	})
	if err != nil {
		return nil, err
	}

	return res.TransferInstructions, nil
}
