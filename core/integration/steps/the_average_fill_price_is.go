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

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"

	"github.com/cucumber/godog"
)

func TheAverageFillPriceIs(exec Execution, table *godog.Table) error {
	rows := parseFillRowTable(table)
	for _, row := range rows {
		fRow := fillRow{row: row}
		market := fRow.market()
		volume := fRow.volume()
		side := fRow.side()
		expectedFillPrice := fRow.fillPrice()

		actualfillPrice, err := exec.GetFillPriceForMarket(market, volume, side)
		if err != nil {
			return err
		}

		if expectedFillPrice != nil {
			if actualfillPrice.NEQ(expectedFillPrice) {
				return errWrongFillPrice(market, volume, side, expectedFillPrice, actualfillPrice)
			}
		}

		expectedMarkPrice := fRow.markPrice()
		if expectedMarkPrice != nil {
			md, err := exec.GetMarketData(market)
			if err != nil {
				return errMarketDataNotFound(market, err)
			}
			if md.MarkPrice.NEQ(expectedMarkPrice) {
				return errWrongMarkPrice(market, expectedMarkPrice, md)
			}
		}

		expectedFactor, b := fRow.equivalentLinearSlippageFactor()

		if b {
			refPrice := fRow.refPrice()
			if refPrice == nil {
				return fmt.Errorf("'ref price' must be specified if 'equivalent linear slippage factor' is provided")
			}
			dRefPrice := refPrice.ToDecimal()
			actualFactor := num.MaxD(num.DecimalZero(), actualfillPrice.ToDecimal().Sub(dRefPrice).Div(dRefPrice))

			if !actualFactor.Equal(expectedFactor) {
				return errWrongFactor(market, volume, side, expectedFactor, actualFactor)
			}
		}
	}
	return nil
}

func parseFillRowTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"market",
		"volume",
		"side",
	}, []string{
		"fill price",
		"ref price",
		"equivalent linear slippage factor",
		"mark price",
	})
}

type fillRow struct {
	row RowWrapper
}

func (r fillRow) market() string {
	return r.row.MustStr("market")
}

func (r fillRow) volume() uint64 {
	return r.row.MustU64("volume")
}

func (r fillRow) side() types.Side {
	return r.row.MustSide("side")
}

func (r fillRow) fillPrice() *num.Uint {
	return r.row.MaybeUint("fill price")
}

func (r fillRow) refPrice() *num.Uint {
	return r.row.MaybeUint("ref price")
}

func (r fillRow) markPrice() *num.Uint {
	return r.row.MaybeUint("mark price")
}

func (r fillRow) equivalentLinearSlippageFactor() (num.Decimal, bool) {
	return r.row.DecimalB("equivalent linear slippage factor")
}

func errWrongFillPrice(market string, volume uint64, side types.Side, expected, actual *num.Uint) error {
	return fmt.Errorf("wrong fill price for market(%v), volume(%v), side(%v): expected(%v) got(%v)",
		market, volume, side, expected, actual,
	)
}

func errWrongFactor(market string, volume uint64, side types.Side, expected, actual num.Decimal) error {
	return fmt.Errorf("wrong effective linear slippage factor for market(%v), volume(%v), side(%v): expected(%v) got(%v)",
		market, volume, side, expected, actual,
	)
}
