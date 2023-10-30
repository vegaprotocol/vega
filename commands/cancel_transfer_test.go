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

func TestNilCancelTransferFails(t *testing.T) {
	err := checkCancelTransfer(nil)

	assert.Contains(t, err.Get("cancel_transfer"), commands.ErrIsRequired)
}

func TestCancelTransfer(t *testing.T) {
	cases := []struct {
		ctransfer commandspb.CancelTransfer
		errString string
	}{
		{
			ctransfer: commandspb.CancelTransfer{
				TransferId: "18f8b607aad9ef2cd57f2d233766b0c576b27a3e0c50c9db713c00e518c0bbdc",
			},
		},
		{
			ctransfer: commandspb.CancelTransfer{},
			errString: "cancel_transfer.transfer_id (is required)",
		},
	}

	for _, c := range cases {
		err := commands.CheckCancelTransfer(&c.ctransfer)
		if len(c.errString) <= 0 {
			assert.NoError(t, err)
			continue
		}
		assert.EqualError(t, err, c.errString)
	}
}

func checkCancelTransfer(cmd *commandspb.CancelTransfer) commands.Errors {
	err := commands.CheckCancelTransfer(cmd)

	var e commands.Errors
	if ok := errors.As(err, &e); !ok {
		return commands.NewErrors()
	}

	return e
}
