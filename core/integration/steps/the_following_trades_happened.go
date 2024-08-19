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

package steps

import (
	"fmt"
	"time"

	"code.vegaprotocol.io/vega/core/integration/stubs"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/protos/vega"

	"github.com/cucumber/godog"
)

func TheFollowingTradesShouldBeExecuted(
	exec Execution,
	broker *stubs.BrokerStub,
	table *godog.Table,
) error {
	var err error

	for _, row := range parseExecutedTradesTable(table) {
		buyer := row.MustStr("buyer")
		buyerIn := buyer
		seller := row.MustStr("seller")
		sellerIn := seller
		price := row.MustU64("price")
		size := row.MustU64("size")
		aggressorRaw := row.Str("aggressor side")
		aggressor, aerr := Side(aggressorRaw)
		if aggressorRaw != "" && aerr != nil {
			return aerr
		}
		// remap buyer/seller to AMM IDs
		if row.HasColumn("is amm") && row.MustBool("is amm") {
			if id, ok := exec.GetAMMSubAccountID(buyer); ok {
				buyer = id
			}
			if id, ok := exec.GetAMMSubAccountID(seller); ok {
				seller = id
			}
		}

		buyerFee, hasBuyerFee := row.U64B("buyer fee")
		buyerInfraFee, hasBuyerInfraFee := row.U64B("buyer infrastructure fee")
		buyerMakerFee, hasBuyerMakerFee := row.U64B("buyer maker fee")
		buyerLiqFee, hasBuyerLiqFee := row.U64B("buyer liquidity fee")
		buyerInfraFeeVolumeDiscount, hasBuyerInfraFeeVolumeDiscount := row.DecimalB("buyer infrastructure fee volume discount")
		buyerMakerFeeVolumeDiscount, hasBuyerMakerFeeVolumeDiscount := row.DecimalB("buyer maker fee volume discount")
		buyerLiqFeeVolumeDiscount, hasBuyerLiqFeeVolumeDiscount := row.DecimalB("buyer liquidity fee volume discount")
		buyerInfraFeeReferrerDiscount, hasBuyerInfraFeeReferrerDiscount := row.DecimalB("buyer infrastructure fee referrer discount")
		buyerMakerFeeReferrerDiscount, hasBuyerMakerFeeReferrerDiscount := row.DecimalB("buyer maker fee referrer discount")
		buyerLiqFeeReferrerDiscount, hasBuyerLiqFeeReferrerDiscount := row.DecimalB("buyer liquidity fee referrer discount")
		buyerHighVolumeMakerFee, hasBuyerHighVolumeMakerFee := row.DecimalB("buyer high volume maker fee")

		sellerFee, hasSellerFee := row.U64B("seller fee")
		sellerInfraFee, hasSellerInfraFee := row.U64B("seller infrastructure fee")
		sellerMakerFee, hasSellerMakerFee := row.U64B("seller maker fee")
		sellerLiqFee, hasSellerLiqFee := row.U64B("seller liquidity fee")
		sellerInfraFeeVolumeDiscount, hasSellerInfraFeeVolumeDiscount := row.DecimalB("seller infrastructure fee volume discount")
		sellerMakerFeeVolumeDiscount, hasSellerMakerFeeVolumeDiscount := row.DecimalB("seller maker fee volume discount")
		sellerLiqFeeVolumeDiscount, hasSellerLiqFeeVolumeDiscount := row.DecimalB("seller liquidity fee volume discount")
		sellerInfraFeeReferrerDiscount, hasSellerInfraFeeReferrerDiscount := row.DecimalB("seller infrastructure fee referrer discount")
		sellerMakerFeeReferrerDiscount, hasSellerMakerFeeReferrerDiscount := row.DecimalB("seller maker fee referrer discount")
		sellerLiqFeeReferrerDiscount, hasSellerLiqFeeReferrerDiscount := row.DecimalB("seller liquidity fee referrer discount")
		sellerHighVolumeMakerFee, hasSellerHighVolumeMakerFee := row.DecimalB("seller high volume maker fee")

		data := broker.GetTrades()
		var found bool
		for _, v := range data {
			if v.Buyer == buyer &&
				v.Seller == seller &&
				stringToU64(v.Price) == price &&
				v.Size == size &&
				(aggressorRaw == "" || aggressor == v.GetAggressor()) &&
				(!hasBuyerFee || buyerFee == feeToU64(v.BuyerFee)) &&
				(!hasBuyerInfraFee || buyerInfraFee == stringToU64(v.BuyerFee.InfrastructureFee)) &&
				(!hasBuyerMakerFee || buyerMakerFee == stringToU64(v.BuyerFee.MakerFee)) &&
				(!hasBuyerLiqFee || buyerLiqFee == stringToU64(v.BuyerFee.LiquidityFee)) &&
				(!hasBuyerInfraFeeVolumeDiscount || buyerInfraFeeVolumeDiscount.Equal(num.MustDecimalFromString(v.BuyerFee.InfrastructureFeeVolumeDiscount))) &&
				(!hasBuyerMakerFeeVolumeDiscount || buyerMakerFeeVolumeDiscount.Equal(num.MustDecimalFromString(v.BuyerFee.MakerFeeVolumeDiscount))) &&
				(!hasBuyerLiqFeeVolumeDiscount || buyerLiqFeeVolumeDiscount.Equal(num.MustDecimalFromString(v.BuyerFee.LiquidityFeeVolumeDiscount))) &&
				(!hasBuyerInfraFeeReferrerDiscount || buyerInfraFeeReferrerDiscount.Equal(num.MustDecimalFromString(v.BuyerFee.InfrastructureFeeReferrerDiscount))) &&
				(!hasBuyerMakerFeeReferrerDiscount || buyerMakerFeeReferrerDiscount.Equal(num.MustDecimalFromString(v.BuyerFee.MakerFeeReferrerDiscount))) &&
				(!hasBuyerLiqFeeReferrerDiscount || buyerLiqFeeReferrerDiscount.Equal(num.MustDecimalFromString(v.BuyerFee.LiquidityFeeReferrerDiscount))) &&
				(!hasBuyerHighVolumeMakerFee || buyerHighVolumeMakerFee.Equal(num.MustDecimalFromString(v.BuyerFee.HighVolumeMakerFee))) &&
				(!hasSellerFee || sellerFee == feeToU64(v.SellerFee)) &&
				(!hasSellerInfraFee || sellerInfraFee == stringToU64(v.SellerFee.InfrastructureFee)) &&
				(!hasSellerMakerFee || sellerMakerFee == stringToU64(v.SellerFee.MakerFee)) &&
				(!hasSellerLiqFee || sellerLiqFee == stringToU64(v.SellerFee.LiquidityFee)) &&
				(!hasSellerInfraFeeVolumeDiscount || sellerInfraFeeVolumeDiscount.Equal(num.MustDecimalFromString(v.SellerFee.InfrastructureFeeVolumeDiscount))) &&
				(!hasSellerMakerFeeVolumeDiscount || sellerMakerFeeVolumeDiscount.Equal(num.MustDecimalFromString(v.SellerFee.MakerFeeVolumeDiscount))) &&
				(!hasSellerLiqFeeVolumeDiscount || sellerLiqFeeVolumeDiscount.Equal(num.MustDecimalFromString(v.SellerFee.LiquidityFeeVolumeDiscount))) &&
				(!hasSellerInfraFeeReferrerDiscount || sellerInfraFeeReferrerDiscount.Equal(num.MustDecimalFromString(v.SellerFee.InfrastructureFeeReferrerDiscount))) &&
				(!hasSellerMakerFeeReferrerDiscount || sellerMakerFeeReferrerDiscount.Equal(num.MustDecimalFromString(v.SellerFee.MakerFeeReferrerDiscount))) &&
				(!hasSellerLiqFeeReferrerDiscount || sellerLiqFeeReferrerDiscount.Equal(num.MustDecimalFromString(v.SellerFee.LiquidityFeeReferrerDiscount))) &&
				(!hasSellerHighVolumeMakerFee || sellerHighVolumeMakerFee.Equal(num.MustDecimalFromString(v.SellerFee.HighVolumeMakerFee))) {
				found = true
			}
		}
		if !found {
			return errMissingTrade(buyerIn, sellerIn, price, size)
		}
	}
	return err
}

func feeToU64(fee *vega.Fee) uint64 {
	if fee == nil {
		return uint64(0)
	}
	return stringToU64(fee.InfrastructureFee) + stringToU64(fee.LiquidityFee) + stringToU64(fee.MakerFee) + stringToU64(fee.BuyBackFee) + stringToU64(fee.TreasuryFee)
}

func parseExecutedTradesTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"buyer",
		"seller",
		"price",
		"size",
	}, []string{
		"aggressor side",

		"buyer fee",
		"buyer infrastructure fee",
		"buyer liquidity fee",
		"buyer maker fee",
		"buyer infrastructure fee volume discount",
		"buyer liquidity fee volume discount",
		"buyer maker fee volume discount",
		"buyer infrastructure fee referrer discount",
		"buyer liquidity fee referrer discount",
		"buyer maker fee referrer discount",
		"buyer high volume maker fee",

		"seller fee",
		"seller infrastructure fee",
		"seller liquidity fee",
		"seller maker fee",
		"seller infrastructure fee volume discount",
		"seller liquidity fee volume discount",
		"seller maker fee volume discount",
		"seller infrastructure fee referrer discount",
		"seller liquidity fee referrer discount",
		"seller maker fee referrer discount",
		"seller high volume maker fee",
		"is amm",
	})
}

// TheAuctionTradedVolumeAndPriceShouldBe pass in time at which the trades should happen in case there are previous trades in the broker stub.
func TheAuctionTradedVolumeAndPriceShouldBe(broker *stubs.BrokerStub, volume, price string, now time.Time) error {
	v, err := U64(volume)
	if err != nil {
		return err
	}
	p, err := U64(price)
	if err != nil {
		return err
	}
	// get all trades from stub
	trades := broker.GetTrades()
	sawV := uint64(0)
	for _, t := range trades {
		// no trades after the given time
		if t.Timestamp > now.UnixNano() {
			continue
		}
		if stringToU64(t.Price) != p {
			return fmt.Errorf(
				"expected trades to happen at price %d, instead saw a trade of size %d at price %s (%#v)",
				p, t.Size, t.Price, t,
			)
		}
		sawV += t.Size
	}
	if sawV != v {
		return fmt.Errorf(
			"expected a total traded volume of %d, instead saw a traded volume of %d len(%d): (%#v)",
			v, sawV, len(trades), trades,
		)
	}
	return nil
}

func errMissingTrade(buyer string, seller string, price uint64, volume uint64) error {
	return fmt.Errorf(
		"expecting trade was missing: buyer(%v), seller(%v), price(%v), volume(%v)",
		buyer, seller, price, volume,
	)
}
