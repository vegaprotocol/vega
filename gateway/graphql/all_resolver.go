package gql

import (
	"context"

	"code.vegaprotocol.io/data-node/logging"
	protoapi "code.vegaprotocol.io/protos/data-node/api/v1"
	types "code.vegaprotocol.io/protos/vega"
)

type allResolver struct {
	log *logging.Logger
	clt TradingDataServiceClient
}

func (r *allResolver) getEpochByID(ctx context.Context, id uint64) (*types.Epoch, error) {
	req := &protoapi.GetEpochRequest{
		Id: id,
	}
	resp, err := r.clt.GetEpoch(ctx, req)
	return resp.Epoch, err
}

func (r *allResolver) getOrderByID(ctx context.Context, id string, version *int) (*types.Order, error) {
	v, err := convertVersion(version)
	if err != nil {
		r.log.Error("tradingCore client", logging.Error(err))
		return nil, customErrorFromStatus(err)
	}
	orderReq := &protoapi.OrderByIDRequest{
		OrderId: id,
		Version: v,
	}
	order, err := r.clt.OrderByID(ctx, orderReq)
	return order.Order, err
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
	req := protoapi.MarketByIDRequest{MarketId: id}
	res, err := r.clt.MarketByID(ctx, &req)
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
