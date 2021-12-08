package steps

import (
	"fmt"

	"code.vegaprotocol.io/vega/types/num"
	"github.com/cucumber/godog"

	"code.vegaprotocol.io/vega/integration/stubs"
)

func PartiesShouldReceiveTheFollowingReward(
	broker *stubs.BrokerStub,
	table *godog.Table,
	epochSeq string,
) error {
	rewards := broker.GetRewards(epochSeq)

	for _, r := range parseRewardsTable(table) {
		row := rewardRow{row: r}

		actualReward := num.Zero().String()
		if reward, ok := rewards[stubs.AssetParty{Asset: row.Asset(), Party: row.Party()}]; ok {
			actualReward = reward.Amount.String()
		}

		if row.Amount() != actualReward {
			return errMismatchedReward(row, actualReward)
		}
	}
	return nil
}

func errMismatchedReward(row rewardRow, actualReward string) error {
	return formatDiff(
		fmt.Sprintf("reward amount did not match for party(%s)", row.Party()),
		map[string]string{
			"reward amount": row.Amount(),
		},
		map[string]string{
			"reward amount": actualReward,
		},
	)
}

func parseRewardsTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"party",
		"asset",
		"amount",
	}, nil)
}

type rewardRow struct {
	row RowWrapper
}

func (r rewardRow) Asset() string {
	return r.row.MustStr("asset")
}

func (r rewardRow) Party() string {
	return r.row.MustStr("party")
}

func (r rewardRow) Amount() string {
	return r.row.MustStr("amount")
}
