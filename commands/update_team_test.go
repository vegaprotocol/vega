package commands_test

import (
	"errors"
	"testing"

	"code.vegaprotocol.io/vega/commands"
	"code.vegaprotocol.io/vega/libs/ptr"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	vgtest "code.vegaprotocol.io/vega/libs/test"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateReferralSet(t *testing.T) {
	t.Run("Updating team succeeds", testUpdatingTeamSucceeds)
	t.Run("Updating team with team ID fails", testUpdateReferralSetWithoutTeamIDFails)
}

func testUpdatingTeamSucceeds(t *testing.T) {
	tcs := []struct {
		name string
		cmd  *commandspb.UpdateReferralSet
	}{
		{
			name: "with empty values",
			cmd: &commandspb.UpdateReferralSet{
				TeamId: vgtest.RandomVegaID(),
			},
		}, {
			name: "with just enabled rewards",
			cmd: &commandspb.UpdateReferralSet{
				TeamId:    vgtest.RandomVegaID(),
				Name:      nil,
				TeamUrl:   nil,
				AvatarUrl: nil,
			},
		}, {
			name: "with just name",
			cmd: &commandspb.UpdateReferralSet{
				TeamId:    vgtest.RandomVegaID(),
				Name:      ptr.From(vgrand.RandomStr(5)),
				TeamUrl:   nil,
				AvatarUrl: nil,
			},
		}, {
			name: "with just team URL",
			cmd: &commandspb.UpdateReferralSet{
				TeamId:    vgtest.RandomVegaID(),
				Name:      nil,
				TeamUrl:   ptr.From(vgrand.RandomStr(5)),
				AvatarUrl: nil,
			},
		}, {
			name: "with just avatar URL",
			cmd: &commandspb.UpdateReferralSet{
				TeamId:    vgtest.RandomVegaID(),
				Name:      nil,
				TeamUrl:   nil,
				AvatarUrl: ptr.From(vgrand.RandomStr(5)),
			},
		}, {
			name: "with all at once",
			cmd: &commandspb.UpdateReferralSet{
				TeamId:    vgtest.RandomVegaID(),
				Name:      ptr.From(vgrand.RandomStr(5)),
				TeamUrl:   ptr.From(vgrand.RandomStr(5)),
				AvatarUrl: ptr.From(vgrand.RandomStr(5)),
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			require.Empty(tt, checkUpdateReferralSet(tt, tc.cmd))
		})
	}
}

func testUpdateReferralSetWithoutTeamIDFails(t *testing.T) {
	err := checkUpdateReferralSet(t, &commandspb.UpdateReferralSet{
		TeamId: "",
	})

	assert.Contains(t, err.Get("update_team.team_id"), commands.ErrShouldBeAValidVegaID)
}

func checkUpdateReferralSet(t *testing.T, cmd *commandspb.UpdateReferralSet) commands.Errors {
	t.Helper()

	err := commands.CheckUpdateReferralSet(cmd)

	var e commands.Errors
	if ok := errors.As(err, &e); !ok {
		return commands.NewErrors()
	}

	return e
}
