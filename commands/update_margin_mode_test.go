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
	"fmt"
	"testing"

	"code.vegaprotocol.io/vega/commands"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"

	"github.com/stretchr/testify/require"
)

func TestUpdateMarginMode(t *testing.T) {
	positiveMarginFactor := "123"
	banana := "banana"
	invalidDecimalMarginFactor := "1.2.3"
	negativeMarginFactor := "-0.2"
	zeroMarginFactor := "0"
	largeMarginFactor101 := "1.01"
	validMarginFactor1 := "1"
	validMarginFactorHalf := "0.5"

	txs := []struct {
		name    string
		cmd     *commandspb.UpdateMarginMode
		errName string
		err     error
	}{
		{
			"cmd is nil",
			nil,
			"update_margin_mode",
			commands.ErrIsRequired,
		},
		{
			"unspecified mode",
			&commandspb.UpdateMarginMode{
				Mode: commandspb.UpdateMarginMode_MODE_UNSPECIFIED,
			},
			"update_margin_mode.margin_mode",
			commands.ErrIsNotValid,
		},
		{
			"isolated margin mode without margin factor",
			&commandspb.UpdateMarginMode{
				Mode: commandspb.UpdateMarginMode_MODE_ISOLATED_MARGIN,
			},
			"update_margin_mode.margin_factor",
			fmt.Errorf("margin factor must be defined when margin mode is isolated margin"),
		},
		{
			"cross margin mode with margin factor",
			&commandspb.UpdateMarginMode{
				Mode:         commandspb.UpdateMarginMode_MODE_CROSS_MARGIN,
				MarginFactor: &positiveMarginFactor,
			},
			"update_margin_mode.margin_factor",
			fmt.Errorf("margin factor must not be defined when margin mode is cross margin"),
		},
		{
			"cross margin mode with invalid number as margin factor 1",
			&commandspb.UpdateMarginMode{
				Mode:         commandspb.UpdateMarginMode_MODE_ISOLATED_MARGIN,
				MarginFactor: &banana,
			},
			"update_margin_mode.margin_factor",
			commands.ErrIsNotValidNumber,
		},
		{
			"cross margin mode with invalid number as margin factor 2",
			&commandspb.UpdateMarginMode{
				Mode:         commandspb.UpdateMarginMode_MODE_ISOLATED_MARGIN,
				MarginFactor: &invalidDecimalMarginFactor,
			},
			"update_margin_mode.margin_factor",
			commands.ErrIsNotValidNumber,
		},
		{
			"cross margin mode with negative number as margin factor",
			&commandspb.UpdateMarginMode{
				Mode:         commandspb.UpdateMarginMode_MODE_ISOLATED_MARGIN,
				MarginFactor: &negativeMarginFactor,
			},
			"update_margin_mode.margin_factor",
			commands.ErrMustBePositive,
		},
		{
			"cross margin mode with 0 as margin factor",
			&commandspb.UpdateMarginMode{
				Mode:         commandspb.UpdateMarginMode_MODE_ISOLATED_MARGIN,
				MarginFactor: &zeroMarginFactor,
			},
			"update_margin_mode.margin_factor",
			commands.ErrMustBePositive,
		},
		{
			"cross margin mode with >1 as margin factor",
			&commandspb.UpdateMarginMode{
				Mode:         commandspb.UpdateMarginMode_MODE_ISOLATED_MARGIN,
				MarginFactor: &largeMarginFactor101,
				MarketId:     "123",
			},
			"update_margin_mode.margin_factor",
			nil,
		},
		{
			"cross margin mode with missing market id",
			&commandspb.UpdateMarginMode{
				Mode:         commandspb.UpdateMarginMode_MODE_ISOLATED_MARGIN,
				MarginFactor: &validMarginFactor1,
			},
			"update_margin_mode.market_id",
			commands.ErrIsRequired,
		},
		{
			"invalid vault ID",
			&commandspb.UpdateMarginMode{
				Mode:         commandspb.UpdateMarginMode_MODE_ISOLATED_MARGIN,
				MarginFactor: &validMarginFactor1,
				MarketId:     "123",
				VaultId:      &banana,
			},
			"update_margin_mode.vault_id",
			commands.ErrInvalidVaultID,
		},
		{
			"valid cross margin update",
			&commandspb.UpdateMarginMode{
				Mode:     commandspb.UpdateMarginMode_MODE_CROSS_MARGIN,
				MarketId: "123",
			},
			"",
			nil,
		},
		{
			"valid isolated margin update",
			&commandspb.UpdateMarginMode{
				Mode:         commandspb.UpdateMarginMode_MODE_ISOLATED_MARGIN,
				MarketId:     "123",
				MarginFactor: &validMarginFactorHalf,
			},
			"",
			nil,
		},
	}

	for _, v := range txs {
		err := checkUpdateMarginMode(t, v.cmd)
		if v.err == nil {
			require.Empty(t, len(err), v.name)
		} else {
			require.Contains(t, err.Get(v.errName), v.err, v.name)
		}
	}
}

func checkUpdateMarginMode(t *testing.T, cmd *commandspb.UpdateMarginMode) commands.Errors {
	t.Helper()
	err := commands.CheckUpdateMarginMode(cmd)
	var e commands.Errors
	if ok := errors.As(err, &e); !ok {
		return commands.NewErrors()
	}
	return e
}
