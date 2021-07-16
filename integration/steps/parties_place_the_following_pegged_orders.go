package steps

import (
	"context"

	"code.vegaprotocol.io/vega/execution"
	"code.vegaprotocol.io/vega/types"

	"github.com/cucumber/godog/gherkin"
)

func PartiesPlaceTheFollowingPeggedOrders(exec *execution.Engine, table *gherkin.DataTable) error {
	for _, r := range parseSubmitPeggedOrderTable(table) {
		row := submitPeggedOrderRow{row: r}

		orderSubmission := &types.OrderSubmission{
			Type:        types.OrderTypeLimit,
			TimeInForce: types.OrderTimeInForceGTC,
			Side:        row.Side(),
			MarketId:    row.MarketID(),
			Size:        row.Volume(),
			Reference:   row.Reference(),
			PeggedOrder: &types.PeggedOrder{
				Reference: row.PeggedReference(),
				Offset:    row.Offset(),
			},
		}
		_, err := exec.SubmitOrder(context.Background(), orderSubmission, row.Party())
		if err := checkExpectedError(row, err); err != nil {
			return err
		}
	}
	return nil
}

func parseSubmitPeggedOrderTable(table *gherkin.DataTable) []RowWrapper {
	return StrictParseTable(table, []string{
		"party",
		"market id",
		"side",
		"volume",
		"pegged reference",
		"offset",
	}, []string{
		"error",
		"reference",
	})
}

type submitPeggedOrderRow struct {
	row RowWrapper
}

func (r submitPeggedOrderRow) Party() string {
	return r.row.MustStr("party")
}

func (r submitPeggedOrderRow) MarketID() string {
	return r.row.MustStr("market id")
}

func (r submitPeggedOrderRow) Side() types.Side {
	return r.row.MustSide("side")
}

func (r submitPeggedOrderRow) PeggedReference() types.PeggedReference {
	return r.row.MustPeggedReference("pegged reference")
}

func (r submitPeggedOrderRow) Volume() uint64 {
	return r.row.MustU64("volume")
}

func (r submitPeggedOrderRow) Offset() int64 {
	return r.row.MustI64("offset")
}

func (r submitPeggedOrderRow) Error() string {
	return r.row.Str("error")
}

func (r submitPeggedOrderRow) ExpectError() bool {
	return r.row.HasColumn("error")
}

func (r submitPeggedOrderRow) Reference() string {
	return r.row.Str("reference")
}
