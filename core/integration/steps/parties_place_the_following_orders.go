// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package steps

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"code.vegaprotocol.io/vega/core/integration/helpers"
	"code.vegaprotocol.io/vega/core/integration/stubs"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/ptr"

	"github.com/cucumber/godog"
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

var refToOrderId = map[string]string{}

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

		// check for pegged orders
		if row.PeggedReference() != types.PeggedReferenceUnspecified {
			orderSubmission.PeggedOrder = &types.PeggedOrder{Reference: row.PeggedReference(), Offset: row.PeggedOffset()}
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

		if resp != nil {
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

		// check for pegged orders
		if row.PeggedReference() != types.PeggedReferenceUnspecified {
			orderSubmission.PeggedOrder = &types.PeggedOrder{Reference: row.PeggedReference(), Offset: row.PeggedOffset()}
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

		// If we have a reference, add a reference -> orderID lookup
		if len(resp.Order.Reference) > 0 {
			refToOrderId[resp.Order.Reference] = resp.Order.ID
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
		fbPriced, raPriced     = row.FallsBelowPriceTrigger(), row.RisesAbovePriceTrigger()
		fbTrailing, raTrailing = row.FallsBelowTrailing(), row.RisesAboveTrailing()
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

	if row.row.HasColumn("ra size override setting") {
		value := row.RisesAboveSizeOverrideSetting()
		if value == types.StopOrderSizeOverrideSettingOrder {
			sub.RisesAbove.SizeOverrideSetting = value
			if value == types.StopOrderSizeOverrideSettingOrder {
				// We need to convert the reference into an order ID
				orderId, OK := refToOrderId[row.RisesAboveSizeOverrideReference()]
				if OK {
					sub.RisesAbove.SizeOverrideValue = &types.StopOrderSizeOverrideValue{OrderID: orderId}
				} else {
					return nil, errors.New("reference doesn't match to existing order")
				}
			}
		}
	}

	if row.row.HasColumn("fb size override setting") {
		value := row.FallsBelowSizeOverrideSetting()
		if value == types.StopOrderSizeOverrideSettingOrder {
			sub.FallsBelow.SizeOverrideSetting = value
			if value == types.StopOrderSizeOverrideSettingOrder {
				// We need to convert the reference into an order ID
				orderId, OK := refToOrderId[row.FallsBelowSizeOverrideReference()]
				if OK {
					sub.FallsBelow.SizeOverrideValue = &types.StopOrderSizeOverrideValue{OrderID: orderId}
				} else {
					return nil, errors.New("reference doesn't match to existing order")
				}
			}
		}
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
		"pegged reference",
		"pegged offset",
		"ra size override setting",
		"ra size override reference",
		"fb size override setting",
		"fb size override reference",
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

func (r submitOrderRow) FallsBelowPriceTrigger() *num.Uint {
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

func (r submitOrderRow) FallsBelowTrailing() num.Decimal {
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
	return now.Add(r.row.MustDurationSec2("so expires in")).Local().UnixNano()
}

func (r submitOrderRow) ExpiryStrategy() types.StopOrderExpiryStrategy {
	if !r.row.HasColumn("so expiry strategy") {
		return types.StopOrderExpiryStrategyCancels
	}
	return r.row.MustExpiryStrategy("so expiry strategy")
}

func (r submitOrderRow) PeggedReference() types.PeggedReference {
	if !r.row.HasColumn("pegged reference") {
		return types.PeggedReferenceUnspecified
	}
	return r.row.MustPeggedReference("pegged reference")
}

func (r submitOrderRow) PeggedOffset() *num.Uint {
	if !r.row.HasColumn("pegged offset") {
		return nil
	}
	return r.row.MustUint("pegged offset")
}

func (r submitOrderRow) RisesAboveSizeOverrideSetting() types.StopOrderSizeOverrideSetting {
	if !r.row.HasColumn("ra size override setting") {
		return types.StopOrderSizeOverrideSettingUnspecified
	}
	return r.row.MustSizeOverrideSetting("ra size override setting")
}

func (r submitOrderRow) RisesAboveSizeOverrideReference() string {
	return r.row.MustStr("ra size override reference")
}

func (r submitOrderRow) FallsBelowSizeOverrideSetting() types.StopOrderSizeOverrideSetting {
	if !r.row.HasColumn("fb size override setting") {
		return types.StopOrderSizeOverrideSettingUnspecified
	}
	return r.row.MustSizeOverrideSetting("fb size override setting")
}

func (r submitOrderRow) FallsBelowSizeOverrideReference() string {
	return r.row.MustStr("fb size override reference")
}
