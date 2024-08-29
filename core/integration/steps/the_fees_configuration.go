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

	"code.vegaprotocol.io/vega/core/integration/steps/market"
	"code.vegaprotocol.io/vega/libs/ptr"
	types "code.vegaprotocol.io/vega/protos/vega"

	"github.com/cucumber/godog"
)

func TheFeesConfiguration(config *market.Config, name string, table *godog.Table) error {
	row := feesConfigRow{row: parseFeesConfigTable(table)}

	liquidityFeeSettings := &types.LiquidityFeeSettings{
		Method: types.LiquidityFeeSettings_METHOD_MARGINAL_COST,
	}

	method, err := row.liquidityFeeMethod()
	if err != nil {
		return err
	}

	if method != types.LiquidityFeeSettings_METHOD_UNSPECIFIED {
		liquidityFeeSettings = &types.LiquidityFeeSettings{
			Method:      method,
			FeeConstant: row.liquidityFeeConstant(),
		}
	}

	return config.FeesConfig.Add(name, &types.Fees{
		Factors: &types.FeeFactors{
			InfrastructureFee: row.infrastructureFee(),
			MakerFee:          row.makerFee(),
			BuyBackFee:        row.buyBackFee(),
			TreasuryFee:       row.treasuryFee(),
		},
		LiquidityFeeSettings: liquidityFeeSettings,
	})
}

func parseFeesConfigTable(table *godog.Table) RowWrapper {
	return StrictParseFirstRow(table,
		[]string{
			"maker fee",
			"infrastructure fee",
		},
		[]string{
			"liquidity fee method",
			"liquidity fee constant",
			"buy back fee",
			"treasury fee",
		},
	)
}

type feesConfigRow struct {
	row RowWrapper
}

func (r feesConfigRow) makerFee() string {
	return r.row.MustStr("maker fee")
}

func (r feesConfigRow) infrastructureFee() string {
	return r.row.MustStr("infrastructure fee")
}

func (r feesConfigRow) buyBackFee() string {
	if r.row.HasColumn("buy back fee") {
		return r.row.MustStr("buy back fee")
	}
	return "0"
}

func (r feesConfigRow) treasuryFee() string {
	if r.row.HasColumn("treasury fee") {
		return r.row.MustStr("treasury fee")
	}
	return "0"
}

func (r feesConfigRow) liquidityFeeMethod() (types.LiquidityFeeSettings_Method, error) {
	if !r.row.HasColumn("liquidity fee method") {
		return types.LiquidityFeeSettings_METHOD_UNSPECIFIED, nil
	}
	return LiquidityFeeMethodType(r.row.Str("liquidity fee method"))
}

func (r feesConfigRow) liquidityFeeConstant() *string {
	if !r.row.HasColumn("liquidity fee constant") {
		return nil
	}
	return ptr.From(r.row.Str("liquidity fee constant"))
}

func LiquidityFeeMethodType(rawValue string) (types.LiquidityFeeSettings_Method, error) {
	ty, ok := types.LiquidityFeeSettings_Method_value[rawValue]
	if !ok {
		return types.LiquidityFeeSettings_Method(ty), fmt.Errorf("invalid liquidity fee method: %v", rawValue)
	}
	return types.LiquidityFeeSettings_Method(ty), nil
}
