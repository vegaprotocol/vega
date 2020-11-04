package gql

import (
	"context"

	"code.vegaprotocol.io/vega/logging"
	types "code.vegaprotocol.io/vega/proto"
	protoapi "code.vegaprotocol.io/vega/proto/api"
	"github.com/golang/protobuf/ptypes/empty"
)

type allResolver struct {
	log *logging.Logger
	clt TradingDataClient
}

func (r *allResolver) getOrderByID(ctx context.Context, id string, version *int) (*types.Order, error) {
	v, err := convertVersion(version)
	if err != nil {
		r.log.Error("tradingCore client", logging.Error(err))
		return nil, customErrorFromStatus(err)
	}
	orderReq := &protoapi.OrderByIDRequest{
		OrderID: id,
		Version: v,
	}
	order, err := r.clt.OrderByID(ctx, orderReq)
	return order, err
}

func (r *allResolver) getAssetByID(ctx context.Context, id string) (*Asset, error) {
	if len(id) <= 0 {
		return nil, ErrMissingIDOrReference
	}
	req := &protoapi.AssetByIDRequest{
		ID: id,
	}
	res, err := r.clt.AssetByID(ctx, req)
	if err != nil {
		return nil, err
	}
	return AssetFromProto(res.Asset)
}

func (r allResolver) allAssets(ctx context.Context) ([]*Asset, error) {
	req := &protoapi.AssetsRequest{}
	res, err := r.clt.Assets(ctx, req)
	if err != nil {
		return nil, err
	}
	out := make([]*Asset, 0, len(res.Assets))
	for _, v := range res.Assets {
		a, err := AssetFromProto(v)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}

	return out, nil
}

func (r *allResolver) getMarketByID(ctx context.Context, id string) (*types.Market, error) {
	req := protoapi.MarketByIDRequest{MarketID: id}
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
	res, err := r.clt.Markets(ctx, &empty.Empty{})
	if err != nil {
		r.log.Error("tradingData client", logging.Error(err))
		return nil, customErrorFromStatus(err)
	}
	return res.Markets, nil

}
