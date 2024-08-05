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

package types_test

import (
	"testing"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"

	"github.com/stretchr/testify/require"
)

func TestTradeSerialisation(t *testing.T) {
	trade := &types.Trade{
		ID:        "1",
		MarketID:  "m1",
		Price:     num.NewUint(100),
		Size:      10,
		Buyer:     "z1",
		Seller:    "z2",
		Aggressor: types.SideBuy,
		BuyOrder:  "buy",
		SellOrder: "sell",
		Timestamp: 100,
		Type:      types.TradeTypeDefault,
		BuyerFee: &types.Fee{
			MakerFee:                          num.NewUint(1),
			InfrastructureFee:                 num.NewUint(2),
			LiquidityFee:                      num.NewUint(3),
			MakerFeeVolumeDiscount:            num.NewUint(4),
			InfrastructureFeeVolumeDiscount:   num.NewUint(5),
			LiquidityFeeVolumeDiscount:        num.NewUint(6),
			MakerFeeReferrerDiscount:          num.NewUint(7),
			InfrastructureFeeReferrerDiscount: num.NewUint(8),
			LiquidityFeeReferrerDiscount:      num.NewUint(9),
			TreasuryFee:                       num.NewUint(30),
			BuyBackFee:                        num.NewUint(40),
			HighVolumeMakerFee:                num.NewUint(50),
		},
		SellerFee: &types.Fee{
			MakerFee:                          num.NewUint(11),
			InfrastructureFee:                 num.NewUint(12),
			LiquidityFee:                      num.NewUint(13),
			MakerFeeVolumeDiscount:            num.NewUint(14),
			InfrastructureFeeVolumeDiscount:   num.NewUint(15),
			LiquidityFeeVolumeDiscount:        num.NewUint(16),
			MakerFeeReferrerDiscount:          num.NewUint(17),
			InfrastructureFeeReferrerDiscount: num.NewUint(18),
			LiquidityFeeReferrerDiscount:      num.NewUint(19),
			TreasuryFee:                       num.NewUint(20),
			BuyBackFee:                        num.NewUint(21),
			HighVolumeMakerFee:                num.NewUint(22),
		},
		BuyerAuctionBatch:  1,
		SellerAuctionBatch: 2,
	}
	p := trade.IntoProto()
	tradePrime := types.TradeFromProto(p)
	require.Equal(t, trade, tradePrime)
	pPrime := tradePrime.IntoProto()
	require.Equal(t, p.String(), pPrime.String())
}
