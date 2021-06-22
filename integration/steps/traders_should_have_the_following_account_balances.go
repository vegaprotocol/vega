package steps

import (
	"fmt"

	"github.com/cucumber/godog/gherkin"

	"code.vegaprotocol.io/vega/integration/stubs"
	types "code.vegaprotocol.io/vega/proto"
)

func TradersShouldHaveTheFollowingAccountBalances(
	broker *stubs.BrokerStub,
	table *gherkin.DataTable,
) error {
	for _, r := range parseAccountBalancesTable(table) {
		row := accountBalancesRow{row: r}
		var hasError bool

		generalAccount, err := broker.GetTraderGeneralAccount(row.Party(), row.Asset())
		if err != nil {
			return errCannotGetTraderGeneralAccount(row.Party(), row.Asset(), err)
		}
		if generalAccount.GetBalance() != row.GeneralAccountBalance() {
			hasError = true
		}

		marginAccount, err := broker.GetTraderMarginAccount(row.Party(), row.MarketID())
		if err != nil {
			return errCannotGetTraderMarginAccount(row.Party(), row.Asset(), err)
		}
		// check bond
		var bondAcc types.Account
		if row.ExpectBondAccountBalance() {
			bondAcc, err = broker.GetTraderBondAccount(row.Party(), row.Asset())
			if err == nil && bondAcc.Balance != row.BondAccountBalance() {
				hasError = true
			}
		}
		if marginAccount.GetBalance() != row.MarginAccountBalance() {
			hasError = true
		}

		if hasError {
			return errMismatchedAccountBalances(row, marginAccount, generalAccount, bondAcc)
		}

	}
	return nil
}

func errCannotGetTraderGeneralAccount(trader, asset string, err error) error {
	return fmt.Errorf("couldn't get general account for trader(%s) and asset(%s): %s",
		trader, asset, err.Error(),
	)
}

func errCannotGetTraderMarginAccount(trader, asset string, err error) error {
	return fmt.Errorf("couldn't get margin account for trader(%s) and asset(%s): %s",
		trader, asset, err.Error(),
	)
}

func errMismatchedAccountBalances(row accountBalancesRow, marginAccount, generalAccount, bondAcc types.Account) error {
	// if bond account was given
	if bondAcc.Type == types.AccountType_ACCOUNT_TYPE_BOND {
		return formatDiff(
			fmt.Sprintf("account balances did not match for party(%s)", row.Party()),
			map[string]string{
				"margin account balance":  u64ToS(row.MarginAccountBalance()),
				"general account balance": u64ToS(row.GeneralAccountBalance()),
				"bond account balance":    u64ToS(row.BondAccountBalance()),
			},
			map[string]string{
				"margin account balance":  u64ToS(marginAccount.GetBalance()),
				"general account balance": u64ToS(generalAccount.GetBalance()),
				"bond account balance":    u64ToS(bondAcc.Balance),
			},
		)
	}
	return formatDiff(
		fmt.Sprintf("account balances did not match for party(%s)", row.Party()),
		map[string]string{
			"margin account balance":  u64ToS(row.MarginAccountBalance()),
			"general account balance": u64ToS(row.GeneralAccountBalance()),
		},
		map[string]string{
			"margin account balance":  u64ToS(marginAccount.GetBalance()),
			"general account balance": u64ToS(generalAccount.GetBalance()),
		},
	)
}

func parseAccountBalancesTable(table *gherkin.DataTable) []RowWrapper {
	return TableWrapper(*table).StrictParse([]string{
		"trader",
		"asset",
		"market id",
		"margin",
		"general",
	}, []string{
		"bond",
	})
}

type accountBalancesRow struct {
	row RowWrapper
}

func (r accountBalancesRow) Party() string {
	return r.row.MustStr("trader")
}

func (r accountBalancesRow) Asset() string {
	return r.row.MustStr("asset")
}

func (r accountBalancesRow) MarketID() string {
	return r.row.MustStr("market id")
}

func (r accountBalancesRow) MarginAccountBalance() uint64 {
	return r.row.MustU64("margin")
}

func (r accountBalancesRow) GeneralAccountBalance() uint64 {
	return r.row.MustU64("general")
}

func (r accountBalancesRow) ExpectBondAccountBalance() bool {
	return r.row.HasColumn("bond")
}

func (r accountBalancesRow) BondAccountBalance() uint64 {
	return r.row.U64("bond")
}
