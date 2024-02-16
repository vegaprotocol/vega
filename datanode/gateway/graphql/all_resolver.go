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

package gql

import (
	"context"

	"code.vegaprotocol.io/vega/libs/ptr"
	"code.vegaprotocol.io/vega/logging"
	apipb "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

type allResolver struct {
	log  *logging.Logger
	clt2 TradingDataServiceClientV2
}

func (r *allResolver) getEpochByID(ctx context.Context, id uint64) (*vegapb.Epoch, error) {
	req := &apipb.GetEpochRequest{
		Id: &id,
	}
	resp, err := r.clt2.GetEpoch(ctx, req)
	return resp.Epoch, err
}

func (r *allResolver) getOrderByID(ctx context.Context, id string, version *int) (*vegapb.Order, error) {
	v, err := convertVersion(version)
	if err != nil {
		r.log.Error("tradingCore client", logging.Error(err))
		return nil, err
	}
	orderReq := &apipb.GetOrderRequest{
		OrderId: id,
		Version: v,
	}
	order, err := r.clt2.GetOrder(ctx, orderReq)
	if err != nil {
		return nil, err
	}

	return order.Order, nil
}

func (r *allResolver) getAssetByID(ctx context.Context, id string) (*vegapb.Asset, error) {
	if len(id) <= 0 {
		return nil, ErrMissingIDOrReference
	}
	req := &apipb.GetAssetRequest{
		AssetId: id,
	}
	res, err := r.clt2.GetAsset(ctx, req)
	if err != nil {
		return nil, err
	}
	return res.Asset, nil
}

func (r *allResolver) getNodeByID(ctx context.Context, id string) (*vegapb.Node, error) {
	if len(id) <= 0 {
		return nil, ErrMissingNodeID
	}
	resp, err := r.clt2.GetNode(
		ctx, &apipb.GetNodeRequest{Id: id})
	if err != nil {
		return nil, err
	}

	return resp.Node, nil
}

func (r *allResolver) getMarketByID(ctx context.Context, id string) (*vegapb.Market, error) {
	req := apipb.GetMarketRequest{MarketId: id}
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

func (r *allResolver) transfersConnection(ctx context.Context, partyID *string, direction *TransferDirection, pagination *apipb.Pagination, isReward *bool, fromEpoch *int, toEpoch *int, status *eventspb.Transfer_Status, scope *apipb.ListTransfersRequest_Scope, gameID *string) (*apipb.TransferConnection, error) {
	// if direction is nil just default to ToOrFrom, except when isReward is not nil and true, and partyID is not nil, in which case the API requires the direction to be FROM
	if direction == nil && (isReward != nil && *isReward && partyID != nil) {
		d := TransferDirectionFrom
		direction = &d
	} else if direction == nil {
		d := TransferDirectionToOrFrom
		direction = &d
	}

	var transferDirection apipb.TransferDirection
	switch *direction {
	case TransferDirectionFrom:
		transferDirection = apipb.TransferDirection_TRANSFER_DIRECTION_TRANSFER_FROM
	case TransferDirectionTo:
		transferDirection = apipb.TransferDirection_TRANSFER_DIRECTION_TRANSFER_TO
	case TransferDirectionToOrFrom:
		transferDirection = apipb.TransferDirection_TRANSFER_DIRECTION_TRANSFER_TO_OR_FROM
	}

	var fromEpochU, toEpochU *uint64
	if fromEpoch != nil {
		fromEpochU = ptr.From(uint64(*fromEpoch))
	}
	if toEpoch != nil {
		toEpochU = ptr.From(uint64(*toEpoch))
	}

	res, err := r.clt2.ListTransfers(ctx, &apipb.ListTransfersRequest{
		Pubkey:     partyID,
		Direction:  transferDirection,
		Pagination: pagination,
		IsReward:   isReward,
		FromEpoch:  fromEpochU,
		ToEpoch:    toEpochU,
		Status:     status,
		Scope:      scope,
		GameId:     gameID,
	})
	if err != nil {
		return nil, err
	}

	return res.Transfers, nil
}
