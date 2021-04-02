package steps

import (
	"context"
	"fmt"

	"github.com/cucumber/godog/gherkin"

	"code.vegaprotocol.io/vega/collateral"
	"code.vegaprotocol.io/vega/integration/stubs"
)

func TradersDepositAssets(
	collateralEngine *collateral.Engine,
	broker *stubs.BrokerStub,
	table *gherkin.DataTable,
) error {
	ctx := context.Background()

	for _, row := range TableWrapper(*table).Parse() {
		trader := traderRow{row: row}
		_, err := collateralEngine.Deposit(
			ctx,
			trader.trader(),
			trader.asset(),
			trader.generalAccountBalance(),
		)
		if err != nil {
			return errCannotDeposit(trader.trader(), trader.asset(), err)
		}

		_, err = broker.GetTraderGeneralAccount(trader.trader(), trader.asset())
		if err != nil {
			return errNoGeneralAccountForTrader(trader, err)
		}
	}
	return nil
}

func errNoGeneralAccountForTrader(trader traderRow, err error) error {
	return fmt.Errorf("trader(%v) has no general account for asset(%v): %s",
		trader.trader(),
		trader.asset(),
		err.Error(),
	)
}

func errCannotDeposit(partyID, asset string, err error) error {
	return fmt.Errorf("couldn't deposit for party(%s) and asset(%s): %s", partyID, asset, err.Error())
}

type traderRow struct {
	row RowWrapper
}

func (r traderRow) trader() string {
	return r.row.MustStr("trader")
}

func (r traderRow) asset() string {
	return r.row.MustStr("asset")
}

func (r traderRow) generalAccountBalance() uint64 {
	return r.row.MustU64("amount")
}
