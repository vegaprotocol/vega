package steps

import (
	"errors"

	"github.com/cucumber/godog"

	"code.vegaprotocol.io/vega/collateral"
	"code.vegaprotocol.io/vega/integration/stubs"
)

func PartiesTransferToStakingAccount(
	collateralEngine *collateral.Engine,
	broker *stubs.BrokerStub,
	table *godog.Table,
) error {
	for _, r := range parseDepositAssetTable(table) {
		row := depositAssetRow{row: r}
		generalAccount, err := collateralEngine.GetPartyGeneralAccount(row.Party(), row.Asset())
		if err != nil {
			return errNoGeneralAccountForParty(row, err)
		}
		if generalAccount.Balance.LT(row.Amount()) {
			err = errors.New("insufficient balance")
		}

		if err := checkExpectedError(row, err); err != nil {
			return err
		}
	}
	return nil
}
