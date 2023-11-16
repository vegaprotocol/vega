// Copyright (C) 2023  Gobalsky Labs Limited
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
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"

	"github.com/stretchr/testify/assert"
)

func TestCheckSubmitAMM(t *testing.T) {
	cases := []struct {
		submission commandspb.SubmitAMM
		errStr     string
	}{
		{
			submission: commandspb.SubmitAMM{},
			errStr:     "submit_amm.market_id (is required)",
		},
		{
			submission: commandspb.SubmitAMM{
				MarketId: "notavalidmarketid",
			},
			errStr: "submit_amm.market_id (should be a valid Vega ID)",
		},
		{
			submission: commandspb.SubmitAMM{
				SlippageTolerance: "",
			},
			errStr: "submit_amm.slippage_tolerance (is required)",
		},
		{
			submission: commandspb.SubmitAMM{
				SlippageTolerance: "abc",
			},
			errStr: "submit_amm.slippage_tolerance (is not a valid number)",
		},
		{
			submission: commandspb.SubmitAMM{
				SlippageTolerance: "-0.5",
			},
			errStr: "submit_amm.slippage_tolerance (must be between 0 (excluded) and 1 (included))",
		},
		{
			submission: commandspb.SubmitAMM{
				SlippageTolerance: "0",
			},
			errStr: "submit_amm.slippage_tolerance (must be between 0 (excluded) and 1 (included))",
		},
		{
			submission: commandspb.SubmitAMM{
				SlippageTolerance: "2",
			},
			errStr: "submit_amm.slippage_tolerance (must be between 0 (excluded) and 1 (included))",
		},
		{
			submission: commandspb.SubmitAMM{
				CommitmentAmount: "",
			},
			errStr: "submit_amm.commitment_amount (is required)",
		},
		{
			submission: commandspb.SubmitAMM{
				CommitmentAmount: "abc",
			},
			errStr: "submit_amm.commitment_amount (is not a valid number)",
		},
		{
			submission: commandspb.SubmitAMM{
				CommitmentAmount: "-10",
			},
			errStr: "submit_amm.commitment_amount (must be positive)",
		},
		{
			submission: commandspb.SubmitAMM{
				CommitmentAmount: "0",
			},
			errStr: "submit_amm.commitment_amount (must be positive)",
		},
		{
			submission: commandspb.SubmitAMM{
				ConcentratedLiquidityParameters: nil,
			},
			errStr: "submit_amm.concentrated_liquidity_parameters (is required)",
		},
		{
			submission: commandspb.SubmitAMM{
				ConcentratedLiquidityParameters: &commandspb.SubmitAMM_ConcentratedLiquidityParameters{
					Base: "",
				},
			},
			errStr: "submit_amm.concentrated_liquidity_parameters.base (is required)",
		},
		{
			submission: commandspb.SubmitAMM{
				ConcentratedLiquidityParameters: &commandspb.SubmitAMM_ConcentratedLiquidityParameters{
					Base: "abc",
				},
			},
			errStr: "submit_amm.concentrated_liquidity_parameters.base (is not a valid number)",
		},
		{
			submission: commandspb.SubmitAMM{
				ConcentratedLiquidityParameters: &commandspb.SubmitAMM_ConcentratedLiquidityParameters{
					Base: "-10",
				},
			},
			errStr: "submit_amm.concentrated_liquidity_parameters.base (must be positive)",
		},
		{
			submission: commandspb.SubmitAMM{
				ConcentratedLiquidityParameters: &commandspb.SubmitAMM_ConcentratedLiquidityParameters{
					Base: "0",
				},
			},
			errStr: "submit_amm.concentrated_liquidity_parameters.base (must be positive)",
		},
		{
			submission: commandspb.SubmitAMM{
				ConcentratedLiquidityParameters: &commandspb.SubmitAMM_ConcentratedLiquidityParameters{
					LowerBound: "",
				},
			},
			errStr: "submit_amm.concentrated_liquidity_parameters.lower_bound (is required)",
		},
		{
			submission: commandspb.SubmitAMM{
				ConcentratedLiquidityParameters: &commandspb.SubmitAMM_ConcentratedLiquidityParameters{
					LowerBound: "abc",
				},
			},
			errStr: "submit_amm.concentrated_liquidity_parameters.lower_bound (is not a valid number)",
		},
		{
			submission: commandspb.SubmitAMM{
				ConcentratedLiquidityParameters: &commandspb.SubmitAMM_ConcentratedLiquidityParameters{
					LowerBound: "-10",
				},
			},
			errStr: "submit_amm.concentrated_liquidity_parameters.lower_bound (must be positive)",
		},
		{
			submission: commandspb.SubmitAMM{
				ConcentratedLiquidityParameters: &commandspb.SubmitAMM_ConcentratedLiquidityParameters{
					LowerBound: "0",
				},
			},
			errStr: "submit_amm.concentrated_liquidity_parameters.lower_bound (must be positive)",
		},
		{
			submission: commandspb.SubmitAMM{
				ConcentratedLiquidityParameters: &commandspb.SubmitAMM_ConcentratedLiquidityParameters{
					UpperBound: "",
				},
			},
			errStr: "submit_amm.concentrated_liquidity_parameters.upper_bound (is required)",
		},
		{
			submission: commandspb.SubmitAMM{
				ConcentratedLiquidityParameters: &commandspb.SubmitAMM_ConcentratedLiquidityParameters{
					UpperBound: "abc",
				},
			},
			errStr: "submit_amm.concentrated_liquidity_parameters.upper_bound (is not a valid number)",
		},
		{
			submission: commandspb.SubmitAMM{
				ConcentratedLiquidityParameters: &commandspb.SubmitAMM_ConcentratedLiquidityParameters{
					UpperBound: "-10",
				},
			},
			errStr: "submit_amm.concentrated_liquidity_parameters.upper_bound (must be positive)",
		},
		{
			submission: commandspb.SubmitAMM{
				ConcentratedLiquidityParameters: &commandspb.SubmitAMM_ConcentratedLiquidityParameters{
					UpperBound: "0",
				},
			},
			errStr: "submit_amm.concentrated_liquidity_parameters.upper_bound (must be positive)",
		},
		{
			submission: commandspb.SubmitAMM{
				ConcentratedLiquidityParameters: &commandspb.SubmitAMM_ConcentratedLiquidityParameters{
					MarginRatioAtBounds: "",
				},
			},
			errStr: "submit_amm.concentrated_liquidity_parameters.margin_ratio_at_bounds (is required)",
		},
		{
			submission: commandspb.SubmitAMM{
				ConcentratedLiquidityParameters: &commandspb.SubmitAMM_ConcentratedLiquidityParameters{
					MarginRatioAtBounds: "abc",
				},
			},
			errStr: "submit_amm.concentrated_liquidity_parameters.margin_ratio_at_bounds (is not a valid number)",
		},
		{
			submission: commandspb.SubmitAMM{
				ConcentratedLiquidityParameters: &commandspb.SubmitAMM_ConcentratedLiquidityParameters{
					MarginRatioAtBounds: "-10",
				},
			},
			errStr: "submit_amm.concentrated_liquidity_parameters.margin_ratio_at_bounds (must be positive)",
		},
		{
			submission: commandspb.SubmitAMM{
				ConcentratedLiquidityParameters: &commandspb.SubmitAMM_ConcentratedLiquidityParameters{
					Base:       "1000",
					UpperBound: "900",
				},
			},
			errStr: "submit_amm.concentrated_liquidity_parameters.base (should be a smaller value than upper_bound)",
		},
		{
			submission: commandspb.SubmitAMM{
				ConcentratedLiquidityParameters: &commandspb.SubmitAMM_ConcentratedLiquidityParameters{
					Base:       "1000",
					LowerBound: "1100",
				},
			},
			errStr: "submit_amm.concentrated_liquidity_parameters.base (should be a bigger value than lower_bound)",
		},
		{
			submission: commandspb.SubmitAMM{
				MarketId:          "e9982447fb4128f9968f9981612c5ea85d19b62058ec2636efc812dcbbc745ca",
				SlippageTolerance: "0.09",
				CommitmentAmount:  "10000",
				ConcentratedLiquidityParameters: &commandspb.SubmitAMM_ConcentratedLiquidityParameters{
					Base:                "20000",
					UpperBound:          "30000",
					LowerBound:          "10000",
					MarginRatioAtBounds: "0.1",
				},
			},
		},
	}

	for n, c := range cases {
		if len(c.errStr) <= 0 {
			assert.NoError(t, commands.CheckSubmitAMM(&c.submission), n)
			continue
		}

		assert.Contains(t, checkSubmitAMM(&c.submission).Error(), c.errStr, n)
	}
}

func checkSubmitAMM(cmd *commandspb.SubmitAMM) commands.Errors {
	err := commands.CheckSubmitAMM(cmd)

	var e commands.Errors
	if ok := errors.As(err, &e); !ok {
		return commands.NewErrors()
	}

	return e
}
