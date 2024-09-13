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

package service

import (
	"testing"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/libs/num"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"

	"github.com/stretchr/testify/require"
)

func TestPartyFees(t *testing.T) {
	data := &v2.GetPartyDiscountStatsResponse{}
	mkt := entities.Market{
		ID: entities.MarketID("1234"),
		Fees: entities.Fees{
			Factors: &entities.FeeFactors{
				MakerFee:          "0.0002",
				InfrastructureFee: "0.0005",
				LiquidityFee:      "0.00001",
				TreasuryFee:       "0.0003",
				BuyBackFee:        "0.0001",
			},
		},
	}
	rfDiscount := partyFeeFactors{
		maker:     num.DecimalFromFloat(0.01),
		infra:     num.DecimalFromFloat(0.02),
		liquidity: num.DecimalFromFloat(0.03),
	}
	rfReward := partyFeeFactors{
		maker:     num.DecimalFromFloat(0.001),
		infra:     num.DecimalFromFloat(0.002),
		liquidity: num.DecimalFromFloat(0.003),
	}
	vdFactors := partyFeeFactors{
		maker:     num.DecimalFromFloat(0.0001),
		infra:     num.DecimalFromFloat(0.0002),
		liquidity: num.DecimalFromFloat(0.0003),
	}
	rebate := num.DecimalFromFloat(0.005)
	setMarketFees(data, mkt, rfDiscount, rfReward, vdFactors, rebate)
	require.Equal(t, 1, len(data.PartyMarketFees))
	// 0.0002 + 0.0005 + 0.00001 + 0.0003 + 0.0001
	require.Equal(t, "0.00111", data.PartyMarketFees[0].UndiscountedTakerFee)

	// 0.0002 * (1-0.01) * (1 - 0.0001) * (1 - 0.001) +
	// 0.0005 * (1-0.02) * (1 - 0.0002) * (1 - 0.002) +
	// 0.00001 * (1-0.03) * (1 - 0.0003) * (1 - 0.003) +
	// 0.0003 +
	// 0.0001 =
	// 0.0001977822198 + 0.000488922196 + 0.00000966799873 + 0.0003 + 0.0001 = 0.00109637241453
	require.Equal(t, "0.00109637241453", data.PartyMarketFees[0].DiscountedTakerFee)
	require.Equal(t, "0.0002", data.PartyMarketFees[0].BaseMakerRebate)
	// effective rebate is min(0.0003+0.0001=0.0004, 0.005) = 0.0004 + 0.0002 = 0.0006
	require.Equal(t, "0.0006", data.PartyMarketFees[0].UserMakerRebate)
}
