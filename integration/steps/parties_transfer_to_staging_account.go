package steps

import (
	"github.com/cucumber/godog"

	"code.vegaprotocol.io/vega/integration/stubs"
)

func PartiesTransferToStakingAccount(
	stakingAccountStub *stubs.StakingAccountStub,
	broker *stubs.BrokerStub,
	table *godog.Table,
	epoch string,
) error {
	for _, r := range parseDepositAssetTable(table) {
		row := depositAssetRow{row: r}
		err := stakingAccountStub.IncrementBalance(row.Party(), row.Amount())
		if err := checkExpectedError(row, err); err != nil {
			return err
		}
	}
	return nil
}
