// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package steps

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/cucumber/godog"

	"code.vegaprotocol.io/vega/core/integration/helpers"
	"code.vegaprotocol.io/vega/core/integration/stubs"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/ptr"
)

type Only string

const (
	None   Only = ""
	Post   Only = "post"
	Reduce Only = "reduce"
)

var onlyTypes = map[string]Only{
	"":       None,
	"post":   Post,
	"reduce": Reduce,
}

func PartiesPlaceTheFollowingOrdersWithTicks(exec Execution, time *stubs.TimeStub, epochService EpochService, table *godog.Table) error {
	// ensure time is set + idgen is not nil
	now := time.GetTimeNow()
	time.SetTime(now)
	for _, r := range parseSubmitOrderTable(table) {
		row := newSubmitOrderRow(r)

		orderSubmission := types.OrderSubmission{
			MarketID:    row.MarketID(),
			Side:        row.Side(),
			Price:       row.Price(),
			Size:        row.Volume(),
			ExpiresAt:   row.ExpirationDate(now),
			Type:        row.OrderType(),
			TimeInForce: row.TimeInForce(),
			Reference:   row.Reference(),
		}
		only := row.Only()
		switch only {
		case Post:
			orderSubmission.PostOnly = true
		case Reduce:
			orderSubmission.ReduceOnly = true
		}

		resp, err := exec.SubmitOrder(context.Background(), &orderSubmission, row.Party())
		if ceerr := checkExpectedError(row, err, nil); ceerr != nil {
			return ceerr
		}

		if !row.ExpectResultingTrades() || err != nil {
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
		// make it look like we start a new block
		epochService.OnBlockEnd(context.Background())
		// indicate block has ended, so we MTM if needed
		exec.BlockEnd(context.Background())
		// trigger OnTick calls, but without actually progressing time
		time.SetTime(time.GetTimeNow())
	}
	return nil
}

func PartiesPlaceTheFollowingOrdersBlocksApart(exec Execution, time *stubs.TimeStub, block *helpers.Block, epochService EpochService, table *godog.Table, blockCount string) error {
	// ensure time is set + idgen is not nil
	now := time.GetTimeNow()
	time.SetTime(now)
	nr, err := strconv.ParseInt(blockCount, 10, 0)
	if err != nil {
		return err
	}
	for _, r := range parseSubmitOrderTable(table) {
		row := newSubmitOrderRow(r)

		orderSubmission := types.OrderSubmission{
			MarketID:    row.MarketID(),
			Side:        row.Side(),
			Price:       row.Price(),
			Size:        row.Volume(),
			ExpiresAt:   row.ExpirationDate(now),
			Type:        row.OrderType(),
			TimeInForce: row.TimeInForce(),
			Reference:   row.Reference(),
		}
		only := row.Only()
		switch only {
		case Post:
			orderSubmission.PostOnly = true
		case Reduce:
			orderSubmission.ReduceOnly = true
		}

		resp, err := exec.SubmitOrder(context.Background(), &orderSubmission, row.Party())
		if ceerr := checkExpectedError(row, err, nil); ceerr != nil {
			return ceerr
		}

		if !row.ExpectResultingTrades() || err != nil {
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
		now := time.GetTimeNow()
		for i := int64(0); i < nr; i++ {
			epochService.OnBlockEnd(context.Background())
			// end of block
			exec.BlockEnd(context.Background())
			now = now.Add(block.GetDuration())
			// progress time
			time.SetTime(now)
		}
	}
	return nil
}

func PartiesPlaceTheFollowingOrders(
	exec Execution,
	ts *stubs.TimeStub,
	table *godog.Table,
) error {
	now := ts.GetTimeNow()
	for _, r := range parseSubmitOrderTable(table) {
		row := newSubmitOrderRow(r)

		orderSubmission := types.OrderSubmission{
			MarketID:    row.MarketID(),
			Side:        row.Side(),
			Price:       row.Price(),
			Size:        row.Volume(),
			ExpiresAt:   row.ExpirationDate(now),
			Type:        row.OrderType(),
			TimeInForce: row.TimeInForce(),
			Reference:   row.Reference(),
		}
		only := row.Only()
		switch only {
		case Post:
			orderSubmission.PostOnly = true
		case Reduce:
			orderSubmission.ReduceOnly = true
		}

		// check for stop orders
		stopOrderSubmission, err := buildStopOrder(&orderSubmission, row, now)
		if err != nil {
			return err
		}

		var resp *types.OrderConfirmation
		if stopOrderSubmission != nil {
			resp, err = exec.SubmitStopOrder(
				context.Background(),
				stopOrderSubmission,
				row.Party(),
			)
		} else {
			resp, err = exec.SubmitOrder(context.Background(), &orderSubmission, row.Party())
		}
		if ceerr := checkExpectedError(row, err, nil); ceerr != nil {
			return ceerr
		}

		if !row.ExpectResultingTrades() || err != nil {
			continue
		}

		if resp == nil {
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

func PartyAddsTheFollowingOrdersToABatch(party string, exec Execution, time *stubs.TimeStub, table *godog.Table) error {
	// ensure time is set + idgen is not nil
	now := time.GetTimeNow()
	time.SetTime(now)
	for _, r := range parseAddOrderToBatchTable(table) {
		row := newSubmitOrderRow(r)

		orderSubmission := types.OrderSubmission{
			MarketID:    row.MarketID(),
			Side:        row.Side(),
			Price:       row.Price(),
			Size:        row.Volume(),
			ExpiresAt:   row.ExpirationDate(now),
			Type:        row.OrderType(),
			TimeInForce: row.TimeInForce(),
			Reference:   row.Reference(),
		}
		only := row.Only()
		switch only {
		case Post:
			orderSubmission.PostOnly = true
		case Reduce:
			orderSubmission.ReduceOnly = true
		}
		if err := exec.AddSubmitOrderToBatch(&orderSubmission, party); err != nil {
			return err
		}
	}
	return nil
}

func PartySubmitsTheirBatchInstruction(party string, exec Execution) error {
	return exec.ProcessBatch(context.Background(), party)
}

func PartyStartsABatchInstruction(party string, exec Execution) error {
	return exec.StartBatch(party)
}

func buildStopOrder(
	submission *types.OrderSubmission,
	row submitOrderRow,
	now time.Time,
) (*types.StopOrdersSubmission, error) {
	var (
		fbPriced, raPriced     = row.FallsBellowPriceTrigger(), row.RisesAbovePriceTrigger()
		fbTrailing, raTrailing = row.FallsBellowTrailing(), row.RisesAboveTrailing()
	)

	if fbPriced == nil && fbTrailing.IsZero() && raPriced == nil && raTrailing.IsZero() {
		return nil, nil
	}

	if fbPriced != nil && !fbTrailing.IsZero() {
		return nil, errors.New("cannot use bot trailing and priced trigger for falls below")
	}

	if raPriced != nil && !raTrailing.IsZero() {
		return nil, errors.New("cannot use bot trailing and priced trigger for rises above")
	}

	sub := &types.StopOrdersSubmission{}

	switch {
	case fbPriced != nil:
		sub.FallsBelow = &types.StopOrderSetup{
			OrderSubmission: submission,
			Expiry:          &types.StopOrderExpiry{},
			Trigger: types.NewPriceStopOrderTrigger(
				types.StopOrderTriggerDirectionFallsBelow,
				fbPriced.Clone(),
			),
		}
	case !fbTrailing.IsZero():
		sub.FallsBelow = &types.StopOrderSetup{
			OrderSubmission: submission,
			Expiry:          &types.StopOrderExpiry{},
			Trigger: types.NewTrailingStopOrderTrigger(
				types.StopOrderTriggerDirectionFallsBelow,
				fbTrailing,
			),
		}
	}

	var (
		strategy        *types.StopOrderExpiryStrategy
		stopOrderExpiry *time.Time
	)
	if stopOrderExp := row.StopOrderExpirationDate(now); stopOrderExp != 0 {
		strategy = ptr.From(row.ExpiryStrategy())
		stopOrderExpiry = ptr.From(time.Unix(0, stopOrderExp))
	}

	switch {
	case raPriced != nil:
		sub.RisesAbove = &types.StopOrderSetup{
			OrderSubmission: ptr.From(*submission),
			Expiry: &types.StopOrderExpiry{
				ExpiryStrategy: strategy,
				ExpiresAt:      stopOrderExpiry,
			},
			Trigger: types.NewPriceStopOrderTrigger(
				types.StopOrderTriggerDirectionRisesAbove,
				raPriced.Clone(),
			),
		}
	case !raTrailing.IsZero():
		sub.RisesAbove = &types.StopOrderSetup{
			OrderSubmission: ptr.From(*submission),
			Expiry: &types.StopOrderExpiry{
				ExpiryStrategy: strategy,
				ExpiresAt:      stopOrderExpiry,
			},
			Trigger: types.NewTrailingStopOrderTrigger(
				types.StopOrderTriggerDirectionRisesAbove,
				raTrailing,
			),
		}
	}

	// Handle OCO references
	if sub.RisesAbove != nil && sub.FallsBelow != nil {
		sub.FallsBelow.OrderSubmission.Reference += "-1"
		sub.RisesAbove.OrderSubmission.Reference += "-2"
	}

	return sub, nil
}

func parseSubmitOrderTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"party",
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
		"only",
		"fb price trigger",
		"fb trailing",
		"ra price trigger",
		"ra trailing",
		"so expires in",
		"so expiry strategy",
	})
}

func parseAddOrderToBatchTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"market id",
		"side",
		"volume",
		"price",
		"type",
		"tif",
	}, []string{
		"reference",
		"error",
		"expires in",
		"only",
	})
}

type submitOrderRow struct {
	row RowWrapper
}

func newSubmitOrderRow(r RowWrapper) submitOrderRow {
	row := submitOrderRow{
		row: r,
	}

	if row.ExpectError() && row.ExpectResultingTrades() && row.ResultingTrades() > 0 {
		panic("you can't expect trades and an error at the same time")
	}

	return row
}

func (r submitOrderRow) Party() string {
	return r.row.MustStr("party")
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

func (r submitOrderRow) OrderType() types.OrderType {
	return r.row.MustOrderType("type")
}

func (r submitOrderRow) TimeInForce() types.OrderTimeInForce {
	return r.row.MustTIF("tif")
}

func (r submitOrderRow) ExpirationDate(now time.Time) int64 {
	if r.OrderType() == types.OrderTypeMarket {
		return 0
	}

	if r.TimeInForce() == types.OrderTimeInForceGTT {
		return now.Add(r.row.MustDurationSec("expires in")).Local().UnixNano()
	}
	// non GTT orders don't need an expiry time
	return 0
}

func (r submitOrderRow) ExpectResultingTrades() bool {
	return r.row.HasColumn("resulting trades")
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

func (r submitOrderRow) Only() Only {
	if !r.row.HasColumn("only") {
		return None
	}
	v := r.row.MustStr("only")
	t, ok := onlyTypes[v]
	if !ok {
		panic(fmt.Errorf("unsupported type %v", v))
	}
	return t
}

func (r submitOrderRow) FallsBellowPriceTrigger() *num.Uint {
	if !r.row.HasColumn("fb price trigger") {
		return nil
	}
	return r.row.MustUint("fb price trigger")
}

func (r submitOrderRow) RisesAbovePriceTrigger() *num.Uint {
	if !r.row.HasColumn("ra price trigger") {
		return nil
	}
	return r.row.MustUint("ra price trigger")
}

func (r submitOrderRow) FallsBellowTrailing() num.Decimal {
	if !r.row.HasColumn("fb trailing") {
		return num.DecimalZero()
	}
	return r.row.MustDecimal("fb trailing")
}

func (r submitOrderRow) RisesAboveTrailing() num.Decimal {
	if !r.row.HasColumn("ra trailing") {
		return num.DecimalZero()
	}
	return r.row.MustDecimal("ra trailing")
}

func (r submitOrderRow) StopOrderExpirationDate(now time.Time) int64 {
	if !r.row.HasColumn("so expires in") {
		return 0
	}
	return now.Add(r.row.MustDurationSec("so expires in")).Local().UnixNano()
}

func (r submitOrderRow) ExpiryStrategy() types.StopOrderExpiryStrategy {
	if !r.row.HasColumn("so expiry strategy") {
		return types.StopOrderExpiryStrategyCancels
	}
	return r.row.MustExpiryStrategy("so expiry strategy")
}
