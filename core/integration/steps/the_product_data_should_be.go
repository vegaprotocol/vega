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
	expectedInternalTwap := row.InternalTWAP()
	actualInternalTwap := pd.Data.IntoProto().GetPerpetualData().InternalTwap

	if expectedInternalTwap != actualInternalTwap {
		return fmt.Errorf("expected '%s' for InternalTWAP, instead got '%s'", expectedInternalTwap, actualInternalTwap)
	}

	expectedExternalTwap := row.ExternalTWAP()
	actualExternalTwap := pd.Data.IntoProto().GetPerpetualData().ExternalTwap
	if expectedExternalTwap != actualExternalTwap {
		return fmt.Errorf("expected '%s' for InternalTWAP, instead got '%s'", expectedExternalTwap, actualExternalTwap)
	}

	expectedFundingPayment, b := row.FundingPayment()
	actualFundingPayment := pd.Data.IntoProto().GetPerpetualData().FundingPayment
	if b && expectedFundingPayment != actualFundingPayment {
		return fmt.Errorf("expected '%s' for InternalTWAP, instead got '%s'", expectedFundingPayment, actualFundingPayment)
	}

	expectedFundingRate, b := row.FundingRate()
	actualFundingRate := pd.Data.IntoProto().GetPerpetualData().FundingRate
	if b && expectedFundingRate != actualFundingRate {
		return fmt.Errorf("expected '%s' for InternalTWAP, instead got '%s'", expectedFundingRate, actualFundingRate)
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
