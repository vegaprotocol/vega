package steps

import (
	"context"

	"github.com/cucumber/godog/gherkin"

	"code.vegaprotocol.io/vega/collateral"
	types "code.vegaprotocol.io/vega/proto"
)

func TheFollowingTraders(
	collateralEngine *collateral.Engine,
	markets []types.Market,
	table *gherkin.DataTable,
) error {
	ctx := context.Background()

	for _, row := range TableWrapper(*table).Parse() {
		trader := traderRow{row: row}

		for _, market := range markets {
			asset, err := market.GetAsset()
			if err != nil {
				return err
			}

			_, err = collateralEngine.Deposit(
				ctx,
				trader.partyID(),
				asset,
				trader.generalAccountBalance(),
			)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

type traderRow struct {
	row RowWrapper
}

func (r traderRow) partyID() string {
	return r.row.Str("name")
}

func (r traderRow) generalAccountBalance() uint64 {
	value, err := r.row.U64("amount")
	if err != nil {
		panic(err)
	}
	return value
}
