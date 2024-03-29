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
	vgtest "code.vegaprotocol.io/vega/libs/test"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"

	"github.com/stretchr/testify/assert"
)

func TestApplyReferralCode(t *testing.T) {
	t.Run("Applying referral code succeeds", testApplyReferralCodeSucceeds)
	t.Run("Applying referral code with team ID fails", testApplyReferralCodeWithoutTeamIDFails)
}

func testApplyReferralCodeSucceeds(t *testing.T) {
	err := checkApplyReferralCode(t, &commandspb.ApplyReferralCode{
		Id: vgtest.RandomVegaID(),
	})

	assert.Empty(t, err)
}

func testApplyReferralCodeWithoutTeamIDFails(t *testing.T) {
	err := checkApplyReferralCode(t, &commandspb.ApplyReferralCode{
		Id: "",
	})

	assert.Contains(t, err.Get("apply_referral_code.id"), commands.ErrShouldBeAValidVegaID)
}

func checkApplyReferralCode(t *testing.T, cmd *commandspb.ApplyReferralCode) commands.Errors {
	t.Helper()

	err := commands.CheckApplyReferralCode(cmd)

	var e commands.Errors
	if ok := errors.As(err, &e); !ok {
		return commands.NewErrors()
	}

	return e
}
