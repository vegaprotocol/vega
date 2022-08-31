package gql

import (
	"context"
	"strconv"

	"code.vegaprotocol.io/vega/datanode/vegatime"
	types "code.vegaprotocol.io/vega/protos/vega"
)

type tradeUpdateResolver VegaResolverRoot

func (r *tradeUpdateResolver) Size(ctx context.Context, obj *types.Trade) (string, error) {
	return strconv.FormatUint(obj.Size, 10), nil
}

func (r *tradeUpdateResolver) CreatedAt(ctx context.Context, obj *types.Trade) (string, error) {
	return vegatime.Format(vegatime.UnixNano(obj.Timestamp)), nil
}

func (r *tradeUpdateResolver) BuyerID(ctx context.Context, obj *types.Trade) (string, error) {
	return obj.Buyer, nil
}

func (r *tradeUpdateResolver) SellerID(ctx context.Context, obj *types.Trade) (string, error) {
	return obj.Seller, nil
}

func (r *tradeUpdateResolver) SellerAuctionBatch(ctx context.Context, obj *types.Trade) (*int, error) {
	i := int(obj.SellerAuctionBatch)
	return &i, nil
}

func (r *tradeUpdateResolver) BuyerAuctionBatch(ctx context.Context, obj *types.Trade) (*int, error) {
	i := int(obj.BuyerAuctionBatch)
	return &i, nil
}

func (r *tradeUpdateResolver) BuyerFee(ctx context.Context, obj *types.Trade) (*TradeFee, error) {
	fee := TradeFee{
		MakerFee:          "0",
		InfrastructureFee: "0",
		LiquidityFee:      "0",
	}
	if obj.BuyerFee != nil {
		fee.MakerFee = obj.BuyerFee.MakerFee
		fee.InfrastructureFee = obj.BuyerFee.InfrastructureFee
		fee.LiquidityFee = obj.BuyerFee.LiquidityFee
	}
	return &fee, nil
}

func (r *tradeUpdateResolver) SellerFee(ctx context.Context, obj *types.Trade) (*TradeFee, error) {
	fee := TradeFee{
		MakerFee:          "0",
		InfrastructureFee: "0",
		LiquidityFee:      "0",
	}
	if obj.SellerFee != nil {
		fee.MakerFee = obj.SellerFee.MakerFee
		fee.InfrastructureFee = obj.SellerFee.InfrastructureFee
		fee.LiquidityFee = obj.SellerFee.LiquidityFee
	}
	return &fee, nil
}
