package steps

import (
	"context"
	"fmt"
	"time"

	"github.com/cucumber/godog/gherkin"

	"code.vegaprotocol.io/vega/execution"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
)

func TradersPlaceTheFollowingOrders(
	exec *execution.Engine,
	table *gherkin.DataTable,
) error {
	for _, r := range parseSubmitOrderTable(table) {
		row := newSubmitOrderRow(r)

		orderSubmission := types.OrderSubmission{
			MarketId:    row.MarketID(),
			Side:        row.Side(),
			Price:       row.Price(),
			Size:        row.Volume(),
			ExpiresAt:   row.ExpirationDate(),
			Type:        row.OrderType(),
			TimeInForce: row.TimeInForce(),
			Reference:   row.Reference(),
		}

		resp, err := exec.SubmitOrder(context.Background(), &orderSubmission, row.Party())
		if err := checkExpectedError(row, err); err != nil {
			return err
		}

		if !row.ExpectResultingTrades() {
			continue
		}

		actualTradeCount := int64(len(resp.Trades))
		if actualTradeCount != row.ResultingTrades() {
			return formatDiff(fmt.Sprintf("the resulting trades didn't match the expectation for order \"%v\"", row.Reference()),
				map[string]string{
					"total": i64ToS(row.ResultingTrades()),
				},
				map[string]string{
					"total": i64ToS(actualTradeCount),
				},
			)
		}
	}
	return nil
}

func parseSubmitOrderTable(table *gherkin.DataTable) []RowWrapper {
	return StrictParseTable(table, []string{
		"trader",
		"market id",
		"side",
		"volume",
		"price",
		"type",
		"tif",
	}, []string{
		"reference",
		"error",
		"resulting trades",
		"expires in",
	})
}

type submitOrderRow struct {
	row RowWrapper
}

func newSubmitOrderRow(r RowWrapper) submitOrderRow {
	row := submitOrderRow{
		row: r,
	}

	if row.ExpectResultingTrades() && row.ExpectError() {
		panic("you can't expect trades and an error at the same time")
	}

	return row
}

func (r submitOrderRow) Party() string {
	return r.row.MustStr("trader")
}

func (r submitOrderRow) MarketID() string {
	return r.row.MustStr("market id")
}

func (r submitOrderRow) Side() types.Side {
	return r.row.MustSide("side")
}

func (r submitOrderRow) Volume() uint64 {
	return r.row.MustU64("volume")
}

func (r submitOrderRow) Price() *num.Uint {
	return r.row.MustUint("price")
}

func (r submitOrderRow) OrderType() types.Order_Type {
	return r.row.MustOrderType("type")
}

func (r submitOrderRow) TimeInForce() types.Order_TimeInForce {
	return r.row.MustTIF("tif")
}

func (r submitOrderRow) ExpirationDate() int64 {
	if r.OrderType() == types.Order_TYPE_MARKET {
		return 0
	}

	now := time.Now()
	if r.TimeInForce() == types.Order_TIME_IN_FORCE_GTT {
		return now.Add(r.row.MustDurationSec("expires in")).Local().UnixNano()
	} else {
		return now.Add(24 * time.Hour).UnixNano()
	}
}

func (r submitOrderRow) ExpectResultingTrades() bool {
	if !r.row.HasColumn("resulting trades") {
		return false
	}
	return r.ResultingTrades() > 0
}

func (r submitOrderRow) ResultingTrades() int64 {
	return r.row.I64("resulting trades")
}

func (r submitOrderRow) Reference() string {
	return r.row.Str("reference")
}

func (r submitOrderRow) Error() string {
	return r.row.Str("error")
}

func (r submitOrderRow) ExpectError() bool {
	return r.row.HasColumn("error")
}
