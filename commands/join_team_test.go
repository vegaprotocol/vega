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

func TestJoinTeam(t *testing.T) {
	t.Run("Joining team succeeds", testJoiningTeamSucceeds)
	t.Run("Joining team with team ID fails", testJoinTeamWithoutTeamIDFails)
}

func testJoiningTeamSucceeds(t *testing.T) {
	err := checkJoinTeam(t, &commandspb.JoinTeam{
		Id: vgtest.RandomVegaID(),
	})

	assert.Empty(t, err)
}

func testJoinTeamWithoutTeamIDFails(t *testing.T) {
	err := checkJoinTeam(t, &commandspb.JoinTeam{
		Id: "",
	})

	assert.Contains(t, err.Get("join_team.id"), commands.ErrShouldBeAValidVegaID)
}

func checkJoinTeam(t *testing.T, cmd *commandspb.JoinTeam) commands.Errors {
	t.Helper()

	err := commands.CheckJoinTeam(cmd)

	var e commands.Errors
	if ok := errors.As(err, &e); !ok {
		return commands.NewErrors()
	}

	return e
}
