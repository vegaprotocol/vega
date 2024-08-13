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

func PartiesPlaceTheFollowingHackedOrders(
	exec Execution,
	ts *stubs.TimeStub,
	table *godog.Table,
) error {
	now := ts.GetTimeNow()
	for _, r := range parseSubmitHackedOrderTable(table) {
		row := newHackedOrderRow(r)
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
		party := row.Party()
		if row.IsAMM() {
			if ammP, ok := exec.GetAMMSubAccountID(party); ok {
				party = ammP
			}
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
		stopOrderSubmission, err := buildStopOrder(&orderSubmission, row.submitOrderRow, now)
		if err != nil {
			return err
		}

		var resp *types.OrderConfirmation
		if stopOrderSubmission != nil {
			resp, err = exec.SubmitStopOrder(
				context.Background(),
				stopOrderSubmission,
				party,
			)
		} else {
			resp, err = exec.SubmitOrder(context.Background(), &orderSubmission, party)
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

func PartyAddsTheFollowingCancelsToABatch(party string, exec Execution, time *stubs.TimeStub, table *godog.Table) error {
	// ensure time is set + idgen is not nil
	now := time.GetTimeNow()
	time.SetTime(now)
	for _, r := range parseAddCancelToBatchTable(table) {
		row := newCancelOrderInBatchRow(r)

		// Convert the reference into a orderID
		orderID := refToOrderId[row.Reference()]
		orderCancel := types.OrderCancellation{
			MarketID: row.MarketID(),
			OrderID:  orderID,
		}
		if err := exec.AddCancelOrderToBatch(&orderCancel, party); err != nil {
			return err
		}
	}
	return nil
}

func PartyAddsTheFollowingAmendsToABatch(party string, exec Execution, time *stubs.TimeStub, table *godog.Table) error {
	// ensure time is set + idgen is not nil
	now := time.GetTimeNow()
	time.SetTime(now)
	for _, r := range parseAddAmendToBatchTable(table) {
		row := newAmendOrderInBatchRow(r)

		// Convert the reference into a orderID
		orderID := refToOrderId[row.Reference()]
		orderAmend := types.OrderAmendment{
			OrderID:         orderID,
			Price:           row.Price(),
			SizeDelta:       row.SizeDelta(),
			MarketID:        row.MarketID(),
			Size:            row.Size(),
			ExpiresAt:       row.ExpiresAt(),
			TimeInForce:     row.TimeInForce(),
			PeggedOffset:    row.PeggedOffset(),
			PeggedReference: row.PeggedReference(),
		}
		if err := exec.AddAmendOrderToBatch(&orderAmend, party); err != nil {
			return err
		}
	}
	return nil
}

func PartySubmitsTheirBatchInstruction(party string, exec Execution) error {
	return exec.ProcessBatch(context.Background(), party)
}

func PartySubmitsTheirBatchInstructionWithError(party, err string, exec Execution) error {
	retErr := exec.ProcessBatch(context.Background(), party)

	err = fmt.Sprintf("1 (%s)", err)
	re := retErr.Error()
	if re != err {
		return retErr
	}
	return nil
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

	var (
		fbStrategy        *types.StopOrderExpiryStrategy
		fbStopOrderExpiry *time.Time
		raStrategy        *types.StopOrderExpiryStrategy
		raStopOrderExpiry *time.Time
	)
	if stopOrderExp := row.StopOrderFBExpirationDate(now); stopOrderExp != 0 {
		fbStrategy = ptr.From(row.ExpiryStrategyFB())
		fbStopOrderExpiry = ptr.From(time.Unix(0, stopOrderExp))
	}
	if stopOrderExp := row.StopOrderRAExpirationDate(now); stopOrderExp != 0 {
		raStrategy = ptr.From(row.ExpiryStrategyRA())
		raStopOrderExpiry = ptr.From(time.Unix(0, stopOrderExp))
	}

	switch {
	case fbPriced != nil:
		sub.FallsBelow = &types.StopOrderSetup{
			OrderSubmission: submission,
			Expiry: &types.StopOrderExpiry{
				ExpiresAt:      fbStopOrderExpiry,
				ExpiryStrategy: fbStrategy,
			},
			Trigger: types.NewPriceStopOrderTrigger(
				types.StopOrderTriggerDirectionFallsBelow,
				fbPriced.Clone(),
			),
		}
	case !fbTrailing.IsZero():
		sub.FallsBelow = &types.StopOrderSetup{
			OrderSubmission: submission,
			Expiry: &types.StopOrderExpiry{
				ExpiresAt:      fbStopOrderExpiry,
				ExpiryStrategy: fbStrategy,
			},
			Trigger: types.NewTrailingStopOrderTrigger(
				types.StopOrderTriggerDirectionFallsBelow,
				fbTrailing,
			),
		}
	}

	switch {
	case raPriced != nil:
		sub.RisesAbove = &types.StopOrderSetup{
			OrderSubmission: ptr.From(*submission),
			Expiry: &types.StopOrderExpiry{
				ExpiryStrategy: raStrategy,
				ExpiresAt:      raStopOrderExpiry,
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
				ExpiryStrategy: raStrategy,
				ExpiresAt:      raStopOrderExpiry,
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
		sub.RisesAbove.SizeOverrideSetting = value

		if row.row.HasColumn("ra size override percentage") {
			percentage := row.RisesAboveSizeOverridePercentage()
			percentageValue := num.MustDecimalFromString(percentage)
			sub.RisesAbove.SizeOverrideValue = &types.StopOrderSizeOverrideValue{PercentageSize: percentageValue}
		}
	}

	if row.row.HasColumn("fb size override setting") {
		value := row.FallsBelowSizeOverrideSetting()
		sub.FallsBelow.SizeOverrideSetting = value

		if row.row.HasColumn("fb size override percentage") {
			percentage := row.FallsBelowSizeOverridePercentage()
			percentageValue := num.MustDecimalFromString(percentage)
			sub.FallsBelow.SizeOverrideValue = &types.StopOrderSizeOverrideValue{PercentageSize: percentageValue}
		}
	}

	return sub, nil
}

func parseSubmitHackedOrderTable(table *godog.Table) []RowWrapper {
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
		"ra expires in",
		"ra expiry strategy",
		"fb expires in",
		"fb expiry strategy",
		"pegged reference",
		"pegged offset",
		"ra size override setting",
		"ra size override percentage",
		"fb size override setting",
		"fb size override percentage",
		"is amm",
	})
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
		"ra expires in",
		"ra expiry strategy",
		"fb expires in",
		"fb expiry strategy",
		"pegged reference",
		"pegged offset",
		"ra size override setting",
		"ra size override percentage",
		"fb size override setting",
		"fb size override percentage",
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

func parseAddCancelToBatchTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"market id",
		"reference",
	}, []string{})
}

func parseAddAmendToBatchTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"market id",
		"reference",
	}, []string{
		"price",
		"size",
		"size delta",
		"expires at",
		"tif",
		"pegged offset",
		"pegged reference",
	})
}

type cancelOrderInBatchRow struct {
	row RowWrapper
}

func newCancelOrderInBatchRow(r RowWrapper) cancelOrderInBatchRow {
	row := cancelOrderInBatchRow{
		row: r,
	}
	return row
}

func (r cancelOrderInBatchRow) MarketID() string {
	return r.row.MustStr("market id")
}

func (r cancelOrderInBatchRow) Reference() string {
	return r.row.MustStr("reference")
}

type amendOrderInBatchRow struct {
	row RowWrapper
}

func newAmendOrderInBatchRow(r RowWrapper) amendOrderInBatchRow {
	row := amendOrderInBatchRow{
		row: r,
	}
	return row
}

func (r amendOrderInBatchRow) MarketID() string {
	return r.row.MustStr("market id")
}

func (r amendOrderInBatchRow) Price() *num.Uint {
	return r.row.MustUint("price")
}

func (r amendOrderInBatchRow) PeggedOffset() *num.Uint {
	if !r.row.HasColumn("pegged offset") {
		return nil
	}
	return r.row.MustUint("pegged offset")
}

func (r amendOrderInBatchRow) PeggedReference() types.PeggedReference {
	if !r.row.HasColumn("pegged reference") {
		return types.PeggedReferenceUnspecified
	}
	return r.row.MustPeggedReference("pegged reference")
}

func (r amendOrderInBatchRow) SizeDelta() int64 {
	if !r.row.HasColumn("size delta") {
		return 0
	}
	return r.row.MustI64("size delta")
}

func (r amendOrderInBatchRow) Size() *uint64 {
	if !r.row.HasColumn("size") {
		return nil
	}
	size := r.row.MustU64("size")
	return &size
}

func (r amendOrderInBatchRow) ExpiresAt() *int64 {
	if !r.row.HasColumn("expires at") {
		return nil
	}
	expires := r.row.MustI64("expires at")
	return &expires
}

func (r amendOrderInBatchRow) TimeInForce() types.OrderTimeInForce {
	return r.row.MustTIF("tif")
}

func (r amendOrderInBatchRow) Reference() string {
	return r.row.MustStr("reference")
}

type submitOrderRow struct {
	row RowWrapper
}

type submitHackedRow struct {
	submitOrderRow
	row RowWrapper
}

func newHackedOrderRow(r RowWrapper) submitHackedRow {
	return submitHackedRow{
		submitOrderRow: newSubmitOrderRow(r),
		row:            r,
	}
}

func (h submitHackedRow) IsAMM() bool {
	if !h.row.HasColumn("is amm") {
		return false
	}
	return h.row.MustBool("is amm")
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
		// Allow negative expires in seconds for testing purposes
		return now.Add(r.row.MustDurationSec2("expires in")).Local().UnixNano()
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

func (r submitOrderRow) StopOrderRAExpirationDate(now time.Time) int64 {
	if !r.row.HasColumn("ra expires in") {
		return 0
	}
	return now.Add(r.row.MustDurationSec2("ra expires in")).Local().UnixNano()
}

func (r submitOrderRow) StopOrderFBExpirationDate(now time.Time) int64 {
	if !r.row.HasColumn("fb expires in") {
		return 0
	}
	return now.Add(r.row.MustDurationSec2("fb expires in")).Local().UnixNano()
}

func (r submitOrderRow) ExpiryStrategyRA() types.StopOrderExpiryStrategy {
	if !r.row.HasColumn("ra expiry strategy") {
		return types.StopOrderExpiryStrategyCancels
	}
	return r.row.MustExpiryStrategy("ra expiry strategy")
}

func (r submitOrderRow) ExpiryStrategyFB() types.StopOrderExpiryStrategy {
	if !r.row.HasColumn("fb expiry strategy") {
		return types.StopOrderExpiryStrategyCancels
	}
	return r.row.MustExpiryStrategy("fb expiry strategy")
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

func (r submitOrderRow) RisesAboveSizeOverridePercentage() string {
	return r.row.MustStr("ra size override percentage")
}

func (r submitOrderRow) FallsBelowSizeOverrideSetting() types.StopOrderSizeOverrideSetting {
	if !r.row.HasColumn("fb size override setting") {
		return types.StopOrderSizeOverrideSettingUnspecified
	}
	return r.row.MustSizeOverrideSetting("fb size override setting")
}

func (r submitOrderRow) FallsBelowSizeOverridePercentage() string {
	return r.row.MustStr("fb size override percentage")
}
