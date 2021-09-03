package steps

import (
	"context"
	"errors"

	"github.com/cucumber/godog"

	"code.vegaprotocol.io/vega/collateral"
	"code.vegaprotocol.io/vega/integration/stubs"
)

func PartiesWithdrawFromStakingAccount(
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
			return errors.New("incorrect amount")
		}

		err = collateralEngine.DecrementBalance(context.Background(), generalAccount.ID, row.Amount())

		if err := checkExpectedError(row, err); err != nil {
			return err
		}
	}
	return nil
}
