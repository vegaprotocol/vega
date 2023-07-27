package commands_test

import (
	"errors"
	"testing"

	"code.vegaprotocol.io/vega/commands"
	"code.vegaprotocol.io/vega/libs/ptr"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateTeam(t *testing.T) {
	t.Run("Updating team succeeds", testUpdatingTeamSucceeds)
	t.Run("Updating team with team ID fails", testUpdateTeamWithoutTeamIDFails)
}

func testUpdatingTeamSucceeds(t *testing.T) {
	tcs := []struct {
		name string
		cmd  *commandspb.UpdateTeam
	}{
		{
			name: "with empty values",
			cmd: &commandspb.UpdateTeam{
				TeamId: RandomStr(5),
			},
		}, {
			name: "with just enabled rewards",
			cmd: &commandspb.UpdateTeam{
				TeamId:        RandomStr(5),
				EnableRewards: true,
				Name:          nil,
				TeamUrl:       nil,
				AvatarUrl:     nil,
			},
		}, {
			name: "with just name",
			cmd: &commandspb.UpdateTeam{
				TeamId:        RandomStr(5),
				EnableRewards: false,
				Name:          ptr.From(vgrand.RandomStr(5)),
				TeamUrl:       nil,
				AvatarUrl:     nil,
			},
		}, {
			name: "with just team URL",
			cmd: &commandspb.UpdateTeam{
				TeamId:        RandomStr(5),
				EnableRewards: false,
				Name:          nil,
				TeamUrl:       ptr.From(vgrand.RandomStr(5)),
				AvatarUrl:     nil,
			},
		}, {
			name: "with just avatar URL",
			cmd: &commandspb.UpdateTeam{
				TeamId:        RandomStr(5),
				EnableRewards: false,
				Name:          nil,
				TeamUrl:       nil,
				AvatarUrl:     ptr.From(vgrand.RandomStr(5)),
			},
		}, {
			name: "with all at once",
			cmd: &commandspb.UpdateTeam{
				TeamId:        RandomStr(5),
				EnableRewards: false,
				Name:          ptr.From(vgrand.RandomStr(5)),
				TeamUrl:       ptr.From(vgrand.RandomStr(5)),
				AvatarUrl:     ptr.From(vgrand.RandomStr(5)),
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			require.Empty(tt, checkUpdateTeam(tt, tc.cmd))
		})
	}
}

func testUpdateTeamWithoutTeamIDFails(t *testing.T) {
	err := checkUpdateTeam(t, &commandspb.UpdateTeam{
		TeamId: "",
	})

	assert.Contains(t, err.Get("update_team.team_id"), commands.ErrShouldBeAValidVegaID)
}

func checkUpdateTeam(t *testing.T, cmd *commandspb.UpdateTeam) commands.Errors {
	t.Helper()

	err := commands.CheckUpdateTeam(cmd)

	var e commands.Errors
	ok := errors.As(err, &e)
	if !ok {
		return commands.NewErrors()
	}

	return e
}
