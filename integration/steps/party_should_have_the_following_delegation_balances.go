package steps

import (
	"fmt"

	"code.vegaprotocol.io/vega/types/num"
	"github.com/cucumber/godog"

	"code.vegaprotocol.io/vega/integration/stubs"
)

func PartiesShouldHaveTheFollowingDelegationBalances(
	broker *stubs.BrokerStub,
	table *godog.Table,
	epochSeq string,
) error {
	delegationBalances := broker.GetDelegationBalance(epochSeq)

	validatorToAmount := map[string]map[string]string{}
	for _, v := range delegationBalances {
		partyDelegations, ok := validatorToAmount[v.Party]
		if !ok {
			validatorToAmount[v.Party] = map[string]string{}
			partyDelegations = validatorToAmount[v.Party]
		}
		partyDelegations[v.NodeId] = v.Amount
	}

	for _, r := range parseDelegationBalanceTable(table) {
		row := delegationBalanceRow{row: r}

		actualBalance := num.Zero().String()
		partyDelegations, ok := validatorToAmount[row.Party()]
		if ok {
			actualBalance = partyDelegations[row.NodeID()]
		}

		if row.ExpectedAmount() != actualBalance {
			return errMismatchedBalance(row, actualBalance)
		}
	}
	return nil
}

func errMismatchedBalance(row delegationBalanceRow, selfStake string) error {
	return formatDiff(
		fmt.Sprintf("delegated balances did not match for node(%s)", row.NodeID()),
		map[string]string{
			"delegation balance": row.ExpectedAmount(),
		},
		map[string]string{
			"delegation balance": selfStake,
		},
	)
}

func parseDelegationBalanceTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"party",
		"node id",
		"amount",
	}, nil)
}

type delegationBalanceRow struct {
	row RowWrapper
}

func (r delegationBalanceRow) Party() string {
	return r.row.MustStr("party")
}

func (r delegationBalanceRow) NodeID() string {
	return r.row.MustStr("node id")
}

func (r delegationBalanceRow) ExpectedAmount() string {
	return r.row.MustStr("amount")
}
