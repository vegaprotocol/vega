package steps

import (
	"context"
	"fmt"

	"github.com/cucumber/godog/gherkin"

	"code.vegaprotocol.io/vega/collateral"
	"code.vegaprotocol.io/vega/integration/stubs"
)

func TradersDepositTheFollowingAssets(
	collateralEngine *collateral.Engine,
	broker *stubs.BrokerStub,
	table *gherkin.DataTable,
) error {
	ctx := context.Background()

	for _, r := range parseDepositAssetTable(table) {
		row := depositAssetRow{row: r}
		_, err := collateralEngine.Deposit(
			ctx,
			row.Party(),
			row.Asset(),
			row.Amount(),
		)
		if err := checkExpectedError(row, err); err != nil {
			return err
		}

		_, err = broker.GetTraderGeneralAccount(row.Party(), row.Asset())
		if err != nil {
			return errNoGeneralAccountForTrader(row, err)
		}
	}
	return nil
}

func errNoGeneralAccountForTrader(party depositAssetRow, err error) error {
	return fmt.Errorf("party(%v) has no general account for asset(%v): %s",
		party.Party(),
		party.Asset(),
		err.Error(),
	)
}

func parseDepositAssetTable(table *gherkin.DataTable) []RowWrapper {
	return StrictParseTable(table, []string{
		"trader",
		"asset",
		"amount",
	}, []string{
		"error",
	})
}

type depositAssetRow struct {
	row RowWrapper
}

func (r depositAssetRow) Party() string {
	return r.row.MustStr("trader")
}

func (r depositAssetRow) Asset() string {
	return r.row.MustStr("asset")
}

func (r depositAssetRow) Amount() uint64 {
	return r.row.MustU64("amount")
}

func (r depositAssetRow) Error() string {
	return r.row.Str("error")
}

func (r depositAssetRow) ExpectError() bool {
	return r.row.HasColumn("error")
}

func (r depositAssetRow) Reference() string {
	return fmt.Sprintf("%s-%s-%d", r.Party(), r.Party(), r.Amount())
}
