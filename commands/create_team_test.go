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

func TestCreateReferralSet(t *testing.T) {
	t.Run("Creating team succeeds", testCreatingTeamSucceeds)
}

func testCreatingTeamSucceeds(t *testing.T) {
	tcs := []struct {
		name string
		cmd  *commandspb.CreateReferralSet
	}{
		{
			name: "with empty values",
			cmd:  &commandspb.CreateReferralSet{},
		}, {
			name: "with just enabled rewards",
			cmd: &commandspb.CreateReferralSet{
				Name:      nil,
				TeamUrl:   nil,
				AvatarUrl: nil,
			},
		}, {
			name: "with just name",
			cmd: &commandspb.CreateReferralSet{
				Name:      ptr.From(vgrand.RandomStr(5)),
				TeamUrl:   nil,
				AvatarUrl: nil,
			},
		}, {
			name: "with just team URL",
			cmd: &commandspb.CreateReferralSet{
				Name:      nil,
				TeamUrl:   ptr.From(vgrand.RandomStr(5)),
				AvatarUrl: nil,
			},
		}, {
			name: "with just avatar URL",
			cmd: &commandspb.CreateReferralSet{
				Name:      nil,
				TeamUrl:   nil,
				AvatarUrl: ptr.From(vgrand.RandomStr(5)),
			},
		}, {
			name: "with all at once",
			cmd: &commandspb.CreateReferralSet{
				Name:      ptr.From(vgrand.RandomStr(5)),
				TeamUrl:   ptr.From(vgrand.RandomStr(5)),
				AvatarUrl: ptr.From(vgrand.RandomStr(5)),
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			require.Empty(tt, checkCreateReferralSet(tt, tc.cmd))
		})
	}
}

func checkCreateReferralSet(t *testing.T, cmd *commandspb.CreateReferralSet) commands.Errors {
	t.Helper()

	err := commands.CheckCreateReferralSet(cmd)

	var e commands.Errors
	if ok := errors.As(err, &e); !ok {
		return commands.NewErrors()
	}

	return e
}
