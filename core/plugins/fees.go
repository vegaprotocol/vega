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

package plugins

import (
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/protos/vega"
)

type feeAmounts struct {
	side  types.Side
	maker *num.Uint
	taker *num.Uint
	other *num.Uint
}

func newFeeAmounts(side types.Side) *feeAmounts {
	return &feeAmounts{
		side:  side,
		maker: num.UintZero(),
		taker: num.UintZero(),
		other: num.UintZero(),
	}
}

func getFeeAmounts(trade *vega.Trade) (*feeAmounts, *feeAmounts) {
	buyer, seller := newFeeAmounts(types.SideBuy), newFeeAmounts(types.SideSell)
	buyer.setAmounts(trade)
	seller.setAmounts(trade)
	// auction end trades don't really have an aggressor, maker and taker fees are split.
	if trade.Aggressor == types.SideSell {
		buyer.maker.AddSum(seller.taker)
	} else if trade.Aggressor == types.SideBuy {
		seller.maker.AddSum(buyer.taker)
	} else {
		buyer.maker.AddSum(seller.taker)
		seller.maker.AddSum(buyer.taker)
	}
	return buyer, seller
}

func (f *feeAmounts) setAmounts(trade *vega.Trade) {
	fee := trade.BuyerFee
	if f.side == types.SideSell {
		fee = trade.SellerFee
	}
	if fee == nil {
		return
	}
	maker, infra, lFee, tFee, bbFee, hvFee := num.UintZero(), num.UintZero(), num.UintZero(), num.UintZero(), num.UintZero(), num.UintZero()
	if len(fee.MakerFee) > 0 {
		maker, _ = num.UintFromString(fee.MakerFee, 10)
	}
	if len(fee.InfrastructureFee) > 0 {
		infra, _ = num.UintFromString(fee.InfrastructureFee, 10)
	}
	if len(fee.LiquidityFee) > 0 {
		lFee, _ = num.UintFromString(fee.LiquidityFee, 10)
	}
	if len(fee.TreasuryFee) > 0 {
		tFee, _ = num.UintFromString(fee.TreasuryFee, 10)
	}
	if len(fee.BuyBackFee) > 0 {
		bbFee, _ = num.UintFromString(fee.BuyBackFee, 10)
	}
	if len(fee.HighVolumeMakerFee) > 0 {
		hvFee, _ = num.UintFromString(fee.HighVolumeMakerFee, 10)
	}
	f.other.AddSum(infra, lFee, tFee, bbFee, hvFee)
	f.taker.AddSum(maker)
}
