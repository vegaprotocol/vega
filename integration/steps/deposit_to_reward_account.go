package steps

import (
	"context"

	"github.com/cucumber/godog"

	"code.vegaprotocol.io/vega/collateral"
	"code.vegaprotocol.io/vega/types/num"
)

func DepositToRewardAccount(
	collateralEngine *collateral.Engine,
	table *godog.Table,
	netDeposits *num.Uint,
) error {
	for _, r := range parseRewardDepositTable(table) {
		row := rewardDeposit{row: r}

		rewardAccount, _ := collateralEngine.GetGlobalRewardAccount(row.Asset())
		collateralEngine.IncrementBalance(context.Background(), rewardAccount.ID, row.Amount())
		netDeposits.Add(netDeposits, row.Amount())
	}
	return nil
}

func parseRewardDepositTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"asset",
		"amount",
	}, nil)
}

type rewardDeposit struct {
	row RowWrapper
}

func (r rewardDeposit) Asset() string {
	return r.row.MustStr("asset")
}

func (r rewardDeposit) Amount() *num.Uint {
	return r.row.MustUint("amount")
}
