package commands_test

import (
	"errors"
	"testing"

	"code.vegaprotocol.io/vega/commands"
	"code.vegaprotocol.io/vega/libs/ptr"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	"github.com/stretchr/testify/require"
)

func TestCreateTeam(t *testing.T) {
	t.Run("Creating team succeeds", testCreatingTeamSucceeds)
}

func testCreatingTeamSucceeds(t *testing.T) {
	tcs := []struct {
		name string
		cmd  *commandspb.CreateTeam
	}{
		{
			name: "with empty values",
			cmd:  &commandspb.CreateTeam{},
		}, {
			name: "with just enabled rewards",
			cmd: &commandspb.CreateTeam{
				Name:      nil,
				TeamUrl:   nil,
				AvatarUrl: nil,
			},
		}, {
			name: "with just name",
			cmd: &commandspb.CreateTeam{
				Name:      ptr.From(vgrand.RandomStr(5)),
				TeamUrl:   nil,
				AvatarUrl: nil,
			},
		}, {
			name: "with just team URL",
			cmd: &commandspb.CreateTeam{
				Name:      nil,
				TeamUrl:   ptr.From(vgrand.RandomStr(5)),
				AvatarUrl: nil,
			},
		}, {
			name: "with just avatar URL",
			cmd: &commandspb.CreateTeam{
				Name:      nil,
				TeamUrl:   nil,
				AvatarUrl: ptr.From(vgrand.RandomStr(5)),
			},
		}, {
			name: "with all at once",
			cmd: &commandspb.CreateTeam{
				Name:      ptr.From(vgrand.RandomStr(5)),
				TeamUrl:   ptr.From(vgrand.RandomStr(5)),
				AvatarUrl: ptr.From(vgrand.RandomStr(5)),
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			require.Empty(tt, checkCreateTeam(tt, tc.cmd))
		})
	}
}

func checkCreateTeam(t *testing.T, cmd *commandspb.CreateTeam) commands.Errors {
	t.Helper()

	err := commands.CheckCreateTeam(cmd)

	var e commands.Errors
	if ok := errors.As(err, &e); !ok {
		return commands.NewErrors()
	}

	return e
}
