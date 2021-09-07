package steps

import (
	"github.com/cucumber/godog"

	"code.vegaprotocol.io/vega/integration/stubs"
)

func PartiesWithdrawFromStakingAccount(
	stakingAccountStub *stubs.StakingAccountStub,
	broker *stubs.BrokerStub,
	table *godog.Table,
) error {
	for _, r := range parseDepositAssetTable(table) {
		row := depositAssetRow{row: r}

		err := stakingAccountStub.DecrementBalance(row.Party(), row.Amount())

		if err := checkExpectedError(row, err); err != nil {
			return err
		}
	}
	return nil
}
