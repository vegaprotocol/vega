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
	"github.com/stretchr/testify/require"
)

func TestApplyReferralCode(t *testing.T) {
	t.Run("Joining team succeeds", testJoiningTeamSucceeds)
	t.Run("Joining team with team ID fails", testApplyReferralCodeWithoutTeamIDFails)
}

func testJoiningTeamSucceeds(t *testing.T) {
	err := checkApplyReferralCode(t, &commandspb.ApplyReferralCode{
		Id: vgtest.RandomVegaID(),
	})

	assert.Empty(t, err)
}

func testApplyReferralCodeWithoutTeamIDFails(t *testing.T) {
	err := checkApplyReferralCode(t, &commandspb.ApplyReferralCode{
		Id: "",
	})

	assert.Contains(t, err.Get("join_team.team_id"), commands.ErrShouldBeAValidVegaID)
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

func TestJoinTeamCommand(t *testing.T) {
	t.Run("Joining team succeeds", testJoinTeamCommandSucceeds)
	t.Run("Joining team with invalid team ID fails", testJoinTeamCommandFails)
}

func testJoinTeamCommandSucceeds(t *testing.T) {
	err := commands.CheckJoinTeamReferralCode(&commandspb.JoinTeam{
		Id: vgtest.RandomVegaID(),
	})

	assert.Nil(t, err)
}

func testJoinTeamCommandFails(t *testing.T) {
	err := commands.CheckJoinTeamReferralCode(&commandspb.JoinTeam{
		Id: "",
	})

	var e commands.Errors
	require.True(t, errors.As(err, &e))
	assert.Contains(t, e.Get("join_team.team_id"), commands.ErrShouldBeAValidVegaID)
}
