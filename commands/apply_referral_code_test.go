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
