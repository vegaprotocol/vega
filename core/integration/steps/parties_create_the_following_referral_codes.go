package steps

import (
	"context"

	"code.vegaprotocol.io/vega/core/referral"
	"code.vegaprotocol.io/vega/core/teams"
	"code.vegaprotocol.io/vega/core/types"
	"github.com/cucumber/godog"

	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
)

func PartiesCreateTheFollowingReferralCode(referralEngine *referral.Engine, teamsEngine *teams.Engine, table *godog.Table) error {
	ctx := context.Background()

	for _, r := range parseCreateReferralCodeTable(table) {
		row := newCreateReferralCodeRow(r)
		err := referralEngine.CreateReferralSet(ctx, row.Party(), row.Code())
		if err := checkExpectedError(row, err, nil); err != nil {
			return err
		}

		if r.HasColumn("is_team") && row.IsTeam() {
			team := &commandspb.CreateReferralSet_Team{
				Name: row.Team(),
			}

			err = teamsEngine.CreateTeam(ctx, row.Party(), types.TeamID(row.Team()), team)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func parseCreateReferralCodeTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"party",
		"code",
	}, []string{
		"is_team",
		"team",
		"error",
		"reference",
	})
}

type createReferralCodeRow struct {
	row RowWrapper
}

func newCreateReferralCodeRow(r RowWrapper) createReferralCodeRow {
	row := createReferralCodeRow{
		row: r,
	}
	return row
}

func (r createReferralCodeRow) Party() types.PartyID {
	return types.PartyID(r.row.MustStr("party"))
}

func (r createReferralCodeRow) Code() types.ReferralSetID {
	return types.ReferralSetID(r.row.MustStr("code"))
}

func (r createReferralCodeRow) Error() string {
	return r.row.Str("error")
}

func (r createReferralCodeRow) ExpectError() bool {
	return r.row.HasColumn("error")
}

func (r createReferralCodeRow) Reference() string {
	return r.row.MustStr("reference")
}

func (r createReferralCodeRow) IsTeam() bool {
	return r.row.Bool("is_team")
}

func (r createReferralCodeRow) Team() string {
	return r.row.Str("team")
}
