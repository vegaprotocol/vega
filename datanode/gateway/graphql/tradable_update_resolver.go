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
	"strconv"

	types "code.vegaprotocol.io/vega/protos/vega"
)

type tradeUpdateResolver VegaResolverRoot

func (r *tradeUpdateResolver) Size(ctx context.Context, obj *types.Trade) (string, error) {
	return strconv.FormatUint(obj.Size, 10), nil
}

func (r *tradeUpdateResolver) CreatedAt(ctx context.Context, obj *types.Trade) (int64, error) {
	return obj.Timestamp, nil
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
