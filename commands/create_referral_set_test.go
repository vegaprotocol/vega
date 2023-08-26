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

func TestCreateReferralSet(t *testing.T) {
	t.Run("Creating referral set succeeds", testCreatingTeamSucceeds)
	t.Run("Creating referral set fails", testCreateReferralSetFails)
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
				IsTeam: false,
			},
		}, {
			name: "with just name",
			cmd: &commandspb.CreateReferralSet{
				IsTeam: true,
				Team: &commandspb.CreateReferralSet_Team{
					Name:      vgrand.RandomStr(5),
					TeamUrl:   nil,
					AvatarUrl: nil,
				},
			},
		}, {
			name: "with just team URL",
			cmd: &commandspb.CreateReferralSet{
				IsTeam: true,
				Team: &commandspb.CreateReferralSet_Team{
					Name:      "some team",
					TeamUrl:   ptr.From(vgrand.RandomStr(5)),
					AvatarUrl: nil,
				},
			},
		}, {
			name: "with just avatar URL",
			cmd: &commandspb.CreateReferralSet{
				IsTeam: true,
				Team: &commandspb.CreateReferralSet_Team{
					Name:      "some team",
					TeamUrl:   nil,
					AvatarUrl: ptr.From(vgrand.RandomStr(5)),
				},
			},
		}, {
			name: "with all at once",
			cmd: &commandspb.CreateReferralSet{
				IsTeam: true,
				Team: &commandspb.CreateReferralSet_Team{
					Name:      vgrand.RandomStr(5),
					TeamUrl:   ptr.From(vgrand.RandomStr(5)),
					AvatarUrl: ptr.From(vgrand.RandomStr(5)),
					Closed:    true,
				},
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			require.Empty(tt, checkCreateReferralSet(tt, tc.cmd), tc.name)
		})
	}
}

func testCreateReferralSetFails(t *testing.T) {
	err := checkCreateReferralSet(t, &commandspb.CreateReferralSet{
		IsTeam: true,
		Team:   nil,
	})

	assert.Contains(t, err.Get("create_referral_set.team"), commands.ErrIsRequired)

	err = checkCreateReferralSet(t, &commandspb.CreateReferralSet{
		IsTeam: true,
		Team:   &commandspb.CreateReferralSet_Team{},
	})

	assert.Contains(t, err.Get("create_referral_set.team.name"), commands.ErrIsRequired)

	err = checkCreateReferralSet(t, &commandspb.CreateReferralSet{
		IsTeam: true,
		Team: &commandspb.CreateReferralSet_Team{
			Name: vgrand.RandomStr(101),
		},
	})

	assert.Contains(t, err.Get("create_referral_set.team.name"), commands.ErrMustBeLessThan100Chars)

	err = checkCreateReferralSet(t, &commandspb.CreateReferralSet{
		IsTeam: true,
		Team: &commandspb.CreateReferralSet_Team{
			AvatarUrl: ptr.From(vgrand.RandomStr(201)),
		},
	})

	assert.Contains(t, err.Get("create_referral_set.team.avatar_url"), commands.ErrMustBeLessThan200Chars)

	err = checkCreateReferralSet(t, &commandspb.CreateReferralSet{
		IsTeam: true,
		Team: &commandspb.CreateReferralSet_Team{
			TeamUrl: ptr.From(vgrand.RandomStr(201)),
		},
	})

	assert.Contains(t, err.Get("create_referral_set.team.team_url"), commands.ErrMustBeLessThan200Chars)
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
