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
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	types "code.vegaprotocol.io/vega/protos/vega"
)

type allResolver struct {
	log  *logging.Logger
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
		return nil, err
	}
	orderReq := &v2.GetOrderRequest{
		OrderId: id,
		Version: v,
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
	req := &v2.GetAssetRequest{
		AssetId: id,
	}
	res, err := r.clt2.GetAsset(ctx, req)
	if err != nil {
		return nil, err
	}
	return res.Asset, nil
}

func (r *allResolver) getNodeByID(ctx context.Context, id string) (*types.Node, error) {
	if len(id) <= 0 {
		return nil, ErrMissingNodeID
	}
	resp, err := r.clt2.GetNode(
		ctx, &v2.GetNodeRequest{Id: id})
	if err != nil {
		return nil, err
	}

	return resp.Node, nil
}

func (r *allResolver) getMarketByID(ctx context.Context, id string) (*types.Market, error) {
	req := v2.GetMarketRequest{MarketId: id}
	res, err := r.clt2.GetMarket(ctx, &req)
	if err != nil {
		r.log.Error("tradingData client", logging.Error(err))
		return nil, err
	}
	// no error / no market = we did not find it
	if res.Market == nil {
		return nil, nil
	}
	return res.Market, nil
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

	res, err := r.clt2.ListTransfers(ctx, &v2.ListTransfersRequest{
		Pubkey:     partyID,
		Direction:  transferDirection,
		Pagination: pagination,
	})
	if err != nil {
		return nil, err
	}

	return res.Transfers, nil
}
