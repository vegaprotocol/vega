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
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"

	"github.com/stretchr/testify/assert"
)

func TestCheckCancelAMM(t *testing.T) {
	cases := []struct {
		submission commandspb.CancelAMM
		errStr     string
	}{
		{
			submission: commandspb.CancelAMM{},
			errStr:     "cancel_amm.market_id (is required)",
		},
		{
			submission: commandspb.CancelAMM{
				MarketId: "notavalidmarketid",
			},
			errStr: "cancel_amm.market_id (should be a valid Vega ID)",
		},
		{
			submission: commandspb.CancelAMM{
				MarketId: "e9982447fb4128f9968f9981612c5ea85d19b62058ec2636efc812dcbbc745ca",
			},
		},
	}

	for n, c := range cases {
		if len(c.errStr) <= 0 {
			assert.NoError(t, commands.CheckCancelAMM(&c.submission), n)
			continue
		}

		assert.Contains(t, checkCancelAMM(&c.submission).Error(), c.errStr, n)
	}
}

func checkCancelAMM(cmd *commandspb.CancelAMM) commands.Errors {
	err := commands.CheckCancelAMM(cmd)

	var e commands.Errors
	if ok := errors.As(err, &e); !ok {
		return commands.NewErrors()
	}

	return e
}
