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
	table *godog.Table,
) error {
	for _, r := range parseWithdrawAssetTable(table) {
		row := withdrawAssetRow{row: r}

		_, err := collateral.LockFundsForWithdraw(context.Background(), row.Party(), row.Asset(), row.Amount())
		if err != nil {
			return errCannotLockFundsForWithdrawal(row, err)
		}

		_, err = collateral.Withdraw(context.Background(), row.Party(), row.Asset(), row.Amount())
		if err := checkExpectedError(row, err); err != nil {
			return err
		}
	}
	return nil
}

func errCannotLockFundsForWithdrawal(row withdrawAssetRow, err error) error {
	return fmt.Errorf("couldn't lock funds for withdrawal of amount(%d) for party(%s), asset(%s): %s",
		row.Amount(), row.Party(), row.Asset(), err.Error(),
	)
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
