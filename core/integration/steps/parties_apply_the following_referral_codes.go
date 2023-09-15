package steps

import (
	"context"

	"code.vegaprotocol.io/vega/core/referral"
	"code.vegaprotocol.io/vega/core/types"
	"github.com/cucumber/godog"
)

func PartiesApplyTheFollowingReferralCode(referralEngine *referral.Engine, table *godog.Table) error {
	ctx := context.Background()

	for _, r := range parseApplyReferralCodeTable(table) {
		row := newApplyReferralCodeRow(r)
		err := referralEngine.ApplyReferralCode(ctx, row.Party(), row.Code())
		if err := checkExpectedError(row, err, nil); err != nil {
			return err
		}
	}
	return nil
}

func parseApplyReferralCodeTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"party",
		"code",
	}, []string{
		"error",
		"reference",
	})
}

type applyReferralCodeRow struct {
	row RowWrapper
}

func newApplyReferralCodeRow(r RowWrapper) applyReferralCodeRow {
	row := applyReferralCodeRow{
		row: r,
	}
	return row
}

func (r applyReferralCodeRow) Party() types.PartyID {
	return types.PartyID(r.row.MustStr("party"))
}

func (r applyReferralCodeRow) Code() types.ReferralSetID {
	return types.ReferralSetID(r.row.MustStr("code"))
}

func (r applyReferralCodeRow) Error() string {
	return r.row.Str("error")
}

func (r applyReferralCodeRow) ExpectError() bool {
	return r.row.HasColumn("error")
}

func (r applyReferralCodeRow) Reference() string {
	return r.row.MustStr("reference")
}
