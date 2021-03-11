package steps

import (
	"context"
	"fmt"

	"github.com/cucumber/godog/gherkin"

	"code.vegaprotocol.io/vega/collateral"
	"code.vegaprotocol.io/vega/integration/stubs"
)

func TheTradersDepositAssets(
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
			return err
		}

		_, err = broker.GetTraderGeneralAccount(trader.trader(), trader.asset())
		if err != nil {
			return fmt.Errorf("trader(%v) has no general account for asset(%v)",
				trader.trader(),
				trader.asset(),
			)
		}
	}
	return nil
}

type traderRow struct {
	row RowWrapper
}

func (r traderRow) trader() string {
	return r.row.Str("trader")
}

func (r traderRow) asset() string {
	return r.row.Str("asset")
}

func (r traderRow) generalAccountBalance() uint64 {
	value, err := r.row.U64("amount")
	if err != nil {
		panic(err)
	}
	return value
}
