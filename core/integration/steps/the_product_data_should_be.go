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
	"code.vegaprotocol.io/vega/protos/vega"

	"github.com/cucumber/godog"
)

func TheProductDataShouldBe(engine Execution, mID string, data *godog.Table) error {
	actual, err := engine.GetMarketData(mID)
	if err != nil {
		return err
	}

	for _, row := range parseProductDataTable(data) {
		expect := ProductDataWrapper{
			row: row,
		}
		err := checkProductData(*actual.ProductData, expect)
		if err != nil {
			return err
		}
	}
	return nil
}

func checkProductData(pd types.ProductData, row ProductDataWrapper) error {
	perpData := pd.Data.IntoProto().GetPerpetualData()

	expectedInternalTwap := row.InternalTWAP()
	actualInternalTwap := perpData.InternalTwap
	if expectedInternalTwap != actualInternalTwap {
		return fmt.Errorf("expected '%s' for InternalTWAP, instead got '%s'", expectedInternalTwap, actualInternalTwap)
	}

	expectedExternalTwap := row.ExternalTWAP()
	actualExternalTwap := perpData.ExternalTwap
	if expectedExternalTwap != actualExternalTwap {
		return fmt.Errorf("expected '%s' for InternalTWAP, instead got '%s'", expectedExternalTwap, actualExternalTwap)
	}

	expectedFundingPayment, b := row.FundingPayment()
	actualFundingPayment := perpData.FundingPayment
	if b && expectedFundingPayment != actualFundingPayment {
		return fmt.Errorf("expected '%s' for funding payment, instead got '%s'", expectedFundingPayment, actualFundingPayment)
	}

	expectedFundingRate, b := row.FundingRate()
	actualFundingRate := perpData.FundingRate
	if b && expectedFundingRate != actualFundingRate {
		return fmt.Errorf("expected '%s' for funding rate, instead got '%s'", expectedFundingRate, actualFundingRate)
	}

	expectedInternalCompositePrice, b := row.InternalCompositePrice()
	actualInternalCompositePrice := perpData.InternalCompositePrice
	if b && expectedInternalCompositePrice != actualInternalCompositePrice {
		return fmt.Errorf("expected '%s' for funding rate, instead got '%s'", expectedFundingRate, actualFundingRate)
	}

	expectedInternalCompositePriceType, b := row.PriceType()
	actualInternalCompositePriceType := perpData.InternalCompositePriceType
	if b && expectedInternalCompositePriceType != actualInternalCompositePriceType {
		return fmt.Errorf("expected '%s' for funding rate, instead got '%s'", expectedFundingRate, actualFundingRate)
	}

	return nil
}

func parseProductDataTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"internal twap",
		"external twap",
	}, []string{
		"funding payment",
		"funding rate",
		"internal composite price",
		"price type",
	})
}

type ProductDataWrapper struct {
	row RowWrapper
}

func (f ProductDataWrapper) InternalTWAP() string {
	return f.row.MustStr("internal twap")
}

func (f ProductDataWrapper) ExternalTWAP() string {
	return f.row.MustStr("external twap")
}

func (f ProductDataWrapper) FundingPayment() (string, bool) {
	return f.row.StrB("funding payment")
}

func (f ProductDataWrapper) FundingRate() (string, bool) {
	return f.row.StrB("funding rate")
}

func (f ProductDataWrapper) InternalCompositePrice() (string, bool) {
	return f.row.StrB("internal composite price")
}

func (f ProductDataWrapper) PriceType() (vega.CompositePriceType, bool) {
	if !f.row.HasColumn("price type") {
		return types.CompositePriceTypeByLastTrade, false
	}
	return f.row.MarkPriceType(), true
}
