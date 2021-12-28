package steps

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/vega/collateral"
	"code.vegaprotocol.io/vega/types/num"
	"github.com/cucumber/godog"
)

func PartiesWithdrawTheFollowingAssets(
	collateral *collateral.Engine,
	netDeposits *num.Uint,
	table *godog.Table,
) error {
	for _, r := range parseWithdrawAssetTable(table) {
		row := withdrawAssetRow{row: r}
		amount := row.Amount()
		_, err := collateral.Withdraw(context.Background(), row.Party(), row.Asset(), amount)
		if err := checkExpectedError(row, err); err != nil {
			return err
		}
		netDeposits = netDeposits.Sub(netDeposits, amount)
	}
	return nil
}

func parseWithdrawAssetTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"party",
		"asset",
		"amount",
	}, []string{
		"error",
	})
}

type withdrawAssetRow struct {
	row RowWrapper
}

func (r withdrawAssetRow) Party() string {
	return r.row.MustStr("party")
}

func (r withdrawAssetRow) Asset() string {
	return r.row.MustStr("asset")
}

func (r withdrawAssetRow) Amount() *num.Uint {
	return r.row.MustUint("amount")
}

func (r withdrawAssetRow) Reference() string {
	return fmt.Sprintf("%s-%s-%d", r.Party(), r.Party(), r.Amount())
}

func (r withdrawAssetRow) Error() string {
	return r.row.Str("error")
}

func (r withdrawAssetRow) ExpectError() bool {
	return r.row.HasColumn("error")
}
