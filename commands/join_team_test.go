package commands_test

import (
	"errors"
	"testing"

	"code.vegaprotocol.io/vega/commands"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	"github.com/stretchr/testify/assert"
)

func TestJoinTeam(t *testing.T) {
	t.Run("Joining team succeeds", testJoiningTeamSucceeds)
	t.Run("Joining team with team ID fails", testJoinTeamWithoutTeamIDFails)
}

func testJoiningTeamSucceeds(t *testing.T) {
	err := checkJoinTeam(t, &commandspb.JoinTeam{
		TeamId: RandomStr(5),
	})

	assert.Empty(t, err)
}

func testJoinTeamWithoutTeamIDFails(t *testing.T) {
	err := checkJoinTeam(t, &commandspb.JoinTeam{
		TeamId: "",
	})

	assert.Contains(t, err.Get("join_team.team_id"), commands.ErrShouldBeAValidVegaID)
}

func checkJoinTeam(t *testing.T, cmd *commandspb.JoinTeam) commands.Errors {
	t.Helper()

	err := commands.CheckJoinTeam(cmd)

	var e commands.Errors
	ok := errors.As(err, &e)
	if !ok {
		return commands.NewErrors()
	}

	return e
}
