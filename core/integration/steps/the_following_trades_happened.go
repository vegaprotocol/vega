// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package steps

import (
	"fmt"
	"time"

	"github.com/cucumber/godog"

	"code.vegaprotocol.io/vega/core/integration/stubs"
	"code.vegaprotocol.io/vega/protos/vega"
)

func TheFollowingTradesShouldBeExecuted(
	broker *stubs.BrokerStub,
	table *godog.Table,
) error {
	var err error

	for _, row := range parseExecutedTradesTable(table) {
		buyer := row.MustStr("buyer")
		seller := row.MustStr("seller")
		price := row.MustU64("price")
		size := row.MustU64("size")
		aggressorRaw := row.Str("aggressor side")
		aggressor, aerr := Side(aggressorRaw)
		if aggressorRaw != "" && aerr != nil {
			return aerr
		}
		buyerFee, hasBuyeFee := row.U64B("buyer fee")
		sellerFee, hasSellerFee := row.U64B("seller fee")
		infraFee, hasInfraFee := row.U64B("infrastructure fee")
		makerFee, hasMakerFee := row.U64B("maker fee")
		liqFee, hasLiqFee := row.U64B("liquidity fee")

		data := broker.GetTrades()
		var found bool
		for _, v := range data {
			if v.Buyer == buyer &&
				v.Seller == seller &&
				stringToU64(v.Price) == price &&
				v.Size == size &&
				(aggressorRaw == "" || aggressor == v.GetAggressor()) &&
				(!hasBuyeFee || buyerFee == feeToU64(v.BuyerFee)) &&
				(!hasSellerFee || sellerFee == feeToU64(v.SellerFee)) &&
				(!hasInfraFee || infraFee == stringToU64(v.BuyerFee.InfrastructureFee)+stringToU64(v.SellerFee.InfrastructureFee)) &&
				(!hasMakerFee || makerFee == stringToU64(v.BuyerFee.MakerFee)+stringToU64(v.SellerFee.MakerFee)) &&
				(!hasLiqFee || liqFee == stringToU64(v.BuyerFee.LiquidityFee)+stringToU64(v.SellerFee.LiquidityFee)) {
				found = true
			}
		}
		if !found {
			return errMissingTrade(buyer, seller, price, size)
		}
	}
	return err
}

func feeToU64(fee *vega.Fee) uint64 {
	if fee == nil {
		return uint64(0)
	}
	return stringToU64(fee.InfrastructureFee) + stringToU64(fee.LiquidityFee) + stringToU64(fee.MakerFee)
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
		"seller fee",
		"infrastructure fee",
		"liquidity fee",
		"maker fee",
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
