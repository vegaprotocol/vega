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

package commands_test

import (
	"errors"
	"testing"

	"code.vegaprotocol.io/vega/commands"
	"code.vegaprotocol.io/vega/libs/ptr"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"

	"github.com/stretchr/testify/assert"
)

func TestCheckAmendAMM(t *testing.T) {
	cases := []struct {
		submission commandspb.AmendAMM
		errStr     string
	}{
		{
			submission: commandspb.AmendAMM{},
			errStr:     "amend_amm.market_id (is required)",
		},
		{
			submission: commandspb.AmendAMM{
				MarketId: "notavalidmarketid",
			},
			errStr: "amend_amm.market_id (should be a valid Vega ID)",
		},
		{
			submission: commandspb.AmendAMM{
				SlippageTolerance: "",
			},
			errStr: "amend_amm.slippage_tolerance (is required)",
		},
		{
			submission: commandspb.AmendAMM{
				SlippageTolerance: "abc",
			},
			errStr: "amend_amm.slippage_tolerance (is not a valid number)",
		},
		{
			submission: commandspb.AmendAMM{
				SlippageTolerance: "-0.5",
			},
			errStr: "amend_amm.slippage_tolerance (must be between 0 (excluded) and 1 (included))",
		},
		{
			submission: commandspb.AmendAMM{
				SlippageTolerance: "0",
			},
			errStr: "amend_amm.slippage_tolerance (must be between 0 (excluded) and 1 (included))",
		},
		{
			submission: commandspb.AmendAMM{
				SlippageTolerance: "2",
			},
			errStr: "amend_amm.slippage_tolerance (must be between 0 (excluded) and 1 (included))",
		},
		{
			submission: commandspb.AmendAMM{
				CommitmentAmount: ptr.From(""),
			},
			errStr: "amend_amm.commitment_amount (is not a valid number)",
		},
		{
			submission: commandspb.AmendAMM{
				CommitmentAmount: ptr.From("abc"),
			},
			errStr: "amend_amm.commitment_amount (is not a valid number)",
		},
		{
			submission: commandspb.AmendAMM{
				CommitmentAmount: ptr.From("-10"),
			},
			errStr: "amend_amm.commitment_amount (must be positive)",
		},
		{
			submission: commandspb.AmendAMM{
				CommitmentAmount: ptr.From("0"),
			},
			errStr: "amend_amm.commitment_amount (must be positive)",
		},
		{
			submission: commandspb.AmendAMM{
				ConcentratedLiquidityParameters: &commandspb.AmendAMM_ConcentratedLiquidityParameters{
					Base: ptr.From(""),
				},
			},
			errStr: "amend_amm.concentrated_liquidity_parameters.base (is not a valid number)",
		},
		{
			submission: commandspb.AmendAMM{
				ConcentratedLiquidityParameters: &commandspb.AmendAMM_ConcentratedLiquidityParameters{
					Base: ptr.From("abc"),
				},
			},
			errStr: "amend_amm.concentrated_liquidity_parameters.base (is not a valid number)",
		},
		{
			submission: commandspb.AmendAMM{
				ConcentratedLiquidityParameters: &commandspb.AmendAMM_ConcentratedLiquidityParameters{
					Base: ptr.From("-10"),
				},
			},
			errStr: "amend_amm.concentrated_liquidity_parameters.base (must be positive)",
		},
		{
			submission: commandspb.AmendAMM{
				ConcentratedLiquidityParameters: &commandspb.AmendAMM_ConcentratedLiquidityParameters{
					Base: ptr.From("0"),
				},
			},
			errStr: "amend_amm.concentrated_liquidity_parameters.base (must be positive)",
		},
		{
			submission: commandspb.AmendAMM{
				ConcentratedLiquidityParameters: &commandspb.AmendAMM_ConcentratedLiquidityParameters{
					LowerBound: ptr.From(""),
				},
			},
			errStr: "amend_amm.concentrated_liquidity_parameters.lower_bound (is not a valid number)",
		},
		{
			submission: commandspb.AmendAMM{
				ConcentratedLiquidityParameters: &commandspb.AmendAMM_ConcentratedLiquidityParameters{
					LowerBound: ptr.From("abc"),
				},
			},
			errStr: "amend_amm.concentrated_liquidity_parameters.lower_bound (is not a valid number)",
		},
		{
			submission: commandspb.AmendAMM{
				ConcentratedLiquidityParameters: &commandspb.AmendAMM_ConcentratedLiquidityParameters{
					LowerBound: ptr.From("-10"),
				},
			},
			errStr: "amend_amm.concentrated_liquidity_parameters.lower_bound (must be positive)",
		},
		{
			submission: commandspb.AmendAMM{
				ConcentratedLiquidityParameters: &commandspb.AmendAMM_ConcentratedLiquidityParameters{
					LowerBound: ptr.From("0"),
				},
			},
			errStr: "amend_amm.concentrated_liquidity_parameters.lower_bound (must be positive)",
		},
		{
			submission: commandspb.AmendAMM{
				ConcentratedLiquidityParameters: &commandspb.AmendAMM_ConcentratedLiquidityParameters{
					UpperBound: ptr.From(""),
				},
			},
			errStr: "amend_amm.concentrated_liquidity_parameters.upper_bound (is not a valid number)",
		},
		{
			submission: commandspb.AmendAMM{
				ConcentratedLiquidityParameters: &commandspb.AmendAMM_ConcentratedLiquidityParameters{
					UpperBound: ptr.From("abc"),
				},
			},
			errStr: "amend_amm.concentrated_liquidity_parameters.upper_bound (is not a valid number)",
		},
		{
			submission: commandspb.AmendAMM{
				ConcentratedLiquidityParameters: &commandspb.AmendAMM_ConcentratedLiquidityParameters{
					UpperBound: ptr.From("-10"),
				},
			},
			errStr: "amend_amm.concentrated_liquidity_parameters.upper_bound (must be positive)",
		},
		{
			submission: commandspb.AmendAMM{
				ConcentratedLiquidityParameters: &commandspb.AmendAMM_ConcentratedLiquidityParameters{
					UpperBound: ptr.From("0"),
				},
			},
			errStr: "amend_amm.concentrated_liquidity_parameters.upper_bound (must be positive)",
		},
		{
			submission: commandspb.AmendAMM{
				ConcentratedLiquidityParameters: &commandspb.AmendAMM_ConcentratedLiquidityParameters{
					MarginRatioAtUpperBound: ptr.From(""),
				},
			},
			errStr: "amend_amm.concentrated_liquidity_parameters.margin_ratio_at_upper_bound (is not a valid number)",
		},
		{
			submission: commandspb.AmendAMM{
				ConcentratedLiquidityParameters: &commandspb.AmendAMM_ConcentratedLiquidityParameters{
					MarginRatioAtUpperBound: ptr.From("abc"),
				},
			},
			errStr: "amend_amm.concentrated_liquidity_parameters.margin_ratio_at_upper_bound (is not a valid number)",
		},
		{
			submission: commandspb.AmendAMM{
				ConcentratedLiquidityParameters: &commandspb.AmendAMM_ConcentratedLiquidityParameters{
					MarginRatioAtUpperBound: ptr.From("-10"),
				},
			},
			errStr: "amend_amm.concentrated_liquidity_parameters.margin_ratio_at_upper_bound (must be positive)",
		},
		{
			submission: commandspb.AmendAMM{
				ConcentratedLiquidityParameters: &commandspb.AmendAMM_ConcentratedLiquidityParameters{
					MarginRatioAtLowerBound: ptr.From(""),
				},
			},
			errStr: "amend_amm.concentrated_liquidity_parameters.margin_ratio_at_lower_bound (is not a valid number)",
		},
		{
			submission: commandspb.AmendAMM{
				ConcentratedLiquidityParameters: &commandspb.AmendAMM_ConcentratedLiquidityParameters{
					MarginRatioAtLowerBound: ptr.From("abc"),
				},
			},
			errStr: "amend_amm.concentrated_liquidity_parameters.margin_ratio_at_lower_bound (is not a valid number)",
		},
		{
			submission: commandspb.AmendAMM{
				ConcentratedLiquidityParameters: &commandspb.AmendAMM_ConcentratedLiquidityParameters{
					MarginRatioAtLowerBound: ptr.From("-10"),
				},
			},
			errStr: "amend_amm.concentrated_liquidity_parameters.margin_ratio_at_lower_bound (must be positive)",
		},
		{
			submission: commandspb.AmendAMM{
				MarketId:          "e9982447fb4128f9968f9981612c5ea85d19b62058ec2636efc812dcbbc745ca",
				SlippageTolerance: "0.09",
			},
			errStr: "* (no updates provided)",
		},
		{
			submission: commandspb.AmendAMM{
				MarketId:          "e9982447fb4128f9968f9981612c5ea85d19b62058ec2636efc812dcbbc745ca",
				SlippageTolerance: "0.09",
				CommitmentAmount:  ptr.From("10000"),
				ConcentratedLiquidityParameters: &commandspb.AmendAMM_ConcentratedLiquidityParameters{
					Base:       ptr.From("20000"),
					UpperBound: ptr.From("30000"),
					LowerBound: ptr.From("10000"),
				},
			},
		},
	}

	for n, c := range cases {
		if len(c.errStr) <= 0 {
			assert.NoError(t, commands.CheckAmendAMM(&c.submission), n)
			continue
		}

		assert.Contains(t, checkAmendAMM(&c.submission).Error(), c.errStr, n)
	}
}

func checkAmendAMM(cmd *commandspb.AmendAMM) commands.Errors {
	err := commands.CheckAmendAMM(cmd)

	var e commands.Errors
	if ok := errors.As(err, &e); !ok {
		return commands.NewErrors()
	}

	return e
}
