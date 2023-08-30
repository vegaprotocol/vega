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
	t.Run("Updating referral set succeeds", testUpdatingTeamSucceeds)
	t.Run("Updating referral set with team ID fails", testUpdateReferralSetWithoutTeamIDFails)
	t.Run("Updating referral set fails", testUpdateReferralSetFails)
}

func testUpdatingTeamSucceeds(t *testing.T) {
	tcs := []struct {
		name string
		cmd  *commandspb.UpdateReferralSet
	}{
		{
			name: "with empty values",
			cmd: &commandspb.UpdateReferralSet{
				Id: vgtest.RandomVegaID(),
			},
		}, {
			name: "with just enabled rewards",
			cmd: &commandspb.UpdateReferralSet{
				Id:     vgtest.RandomVegaID(),
				IsTeam: false,
			},
		}, {
			name: "with just name",
			cmd: &commandspb.UpdateReferralSet{
				Id:     vgtest.RandomVegaID(),
				IsTeam: true,
				Team: &commandspb.UpdateReferralSet_Team{
					Name:      ptr.From(vgrand.RandomStr(5)),
					TeamUrl:   nil,
					AvatarUrl: nil,
					Closed:    nil,
				},
			},
		}, {
			name: "with just team URL",
			cmd: &commandspb.UpdateReferralSet{
				Id:     vgtest.RandomVegaID(),
				IsTeam: true,
				Team: &commandspb.UpdateReferralSet_Team{
					Name:      nil,
					TeamUrl:   ptr.From(vgrand.RandomStr(5)),
					AvatarUrl: nil,
					Closed:    nil,
				},
			},
		}, {
			name: "with just avatar URL",
			cmd: &commandspb.UpdateReferralSet{
				Id:     vgtest.RandomVegaID(),
				IsTeam: true,
				Team: &commandspb.UpdateReferralSet_Team{
					Name:      nil,
					TeamUrl:   nil,
					AvatarUrl: ptr.From(vgrand.RandomStr(5)),
					Closed:    nil,
				},
			},
		}, {
			name: "with all at once",
			cmd: &commandspb.UpdateReferralSet{
				Id:     vgtest.RandomVegaID(),
				IsTeam: true,
				Team: &commandspb.UpdateReferralSet_Team{
					Name:      ptr.From(vgrand.RandomStr(5)),
					TeamUrl:   ptr.From(vgrand.RandomStr(5)),
					AvatarUrl: ptr.From(vgrand.RandomStr(5)),
					Closed:    ptr.From(true),
				},
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
		Id: "",
	})

	assert.Contains(t, err.Get("update_referral_set.id"), commands.ErrShouldBeAValidVegaID)
}

func testUpdateReferralSetFails(t *testing.T) {
	err := checkUpdateReferralSet(t, &commandspb.UpdateReferralSet{
		Id:     "someid",
		IsTeam: true,
		Team:   nil,
	})

	assert.Contains(t, err.Get("update_referral_set.team"), commands.ErrIsRequired)

	err = checkUpdateReferralSet(t, &commandspb.UpdateReferralSet{
		Id:     "someid",
		IsTeam: true,
		Team: &commandspb.UpdateReferralSet_Team{
			Name: ptr.From(""),
		},
	})

	assert.Contains(t, err.Get("update_referral_set.team.name"), commands.ErrIsRequired)

	err = checkUpdateReferralSet(t, &commandspb.UpdateReferralSet{
		IsTeam: true,
		Team: &commandspb.UpdateReferralSet_Team{
			Name: ptr.From(vgrand.RandomStr(101)),
		},
	})

	assert.Contains(t, err.Get("update_referral_set.team.name"), commands.ErrMustBeLessThan100Chars)

	err = checkUpdateReferralSet(t, &commandspb.UpdateReferralSet{
		IsTeam: true,
		Team: &commandspb.UpdateReferralSet_Team{
			AvatarUrl: ptr.From(vgrand.RandomStr(201)),
		},
	})

	assert.Contains(t, err.Get("update_referral_set.team.avatar_url"), commands.ErrMustBeLessThan200Chars)

	err = checkUpdateReferralSet(t, &commandspb.UpdateReferralSet{
		IsTeam: true,
		Team: &commandspb.UpdateReferralSet_Team{
			TeamUrl: ptr.From(vgrand.RandomStr(201)),
		},
	})

	assert.Contains(t, err.Get("update_referral_set.team.team_url"), commands.ErrMustBeLessThan200Chars)
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
