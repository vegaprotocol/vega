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
	"strings"
	"time"

	"code.vegaprotocol.io/vega/core/banking"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	proto "code.vegaprotocol.io/vega/protos/vega"
	"github.com/cucumber/godog"
)

func PartiesSubmitTransfers(
	engine *banking.Engine,
	table *godog.Table,
) error {
	errs := []error{}
	for _, r := range parseOneOffTransferTable(table) {
		transfer, _ := rowToOneOffTransfer(r)
		err := engine.TransferFunds(context.Background(), &types.TransferFunds{
			Kind:   types.TransferCommandKindOneOff,
			OneOff: transfer,
		})
		if len(r.Str("error")) > 0 || err != nil {
			expected := r.Str("error")
			actual := ""
			if err != nil {
				actual = err.Error()
			}
			if expected != actual {
				errs = append(errs, errors.New(r.MustStr("id")+" expected "+expected+" but got "+actual))
			}
		}
	}
	if len(errs) > 0 {
		return ErrStack(errs)
	}
	return nil
}

func parseOneOffTransferTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"id", "from", "from_account_type", "to", "to_account_type", "asset", "amount", "delivery_time",
	}, []string{"market", "error"})
}

func rowToOneOffTransfer(r RowWrapper) (*types.OneOffTransfer, error) {
	id := r.MustStr("id")
	from := r.MustStr("from")
	fromAccountType := r.MustStr("from_account_type")
	fromAT := proto.AccountType_value[fromAccountType]
	to := r.MustStr("to")
	toAccuontType := r.MustStr("to_account_type")
	toAT := proto.AccountType_value[toAccuontType]
	asset := r.MustStr("asset")
	amount := r.MustStr("amount")
	amountUint, _ := num.UintFromString(amount, 10)
	deliveryTime, err := time.Parse("2006-01-02T15:04:05Z", r.MustStr("delivery_time"))
	if err != nil {
		return nil, err
	}

	oneOff := &types.OneOffTransfer{
		TransferBase: &types.TransferBase{
			ID:              id,
			From:            from,
			FromAccountType: types.AccountType(fromAT),
			To:              to,
			ToAccountType:   types.AccountType(toAT),
			Asset:           asset,
			Amount:          amountUint,
		},
		DeliverOn: &deliveryTime,
	}
	return oneOff, nil
}

func PartiesSubmitRecurringTransfers(
	engine *banking.Engine,
	table *godog.Table,
) error {
	errs := []error{}
	for _, r := range parseRecurringTransferTable(table) {
		transfer := rowToRecurringTransfer(r)
		err := engine.TransferFunds(context.Background(), &types.TransferFunds{
			Kind:      types.TransferCommandKindRecurring,
			Recurring: transfer,
		})
		if len(r.Str("error")) > 0 || err != nil {
			expected := r.Str("error")
			actual := ""
			if err != nil {
				actual = err.Error()
			}
			if expected != actual {
				errs = append(errs, errors.New(r.MustStr("id")+" expected "+expected+" but got "+actual))
			}
		}
	}
	if len(errs) > 0 {
		return ErrStack(errs)
	}
	return nil
}

func parseRecurringTransferTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"id", "from", "from_account_type", "to", "to_account_type", "asset", "amount", "start_epoch", "end_epoch", "factor",
	}, []string{"metric", "metric_asset", "markets", "error"})
}

func rowToRecurringTransfer(r RowWrapper) *types.RecurringTransfer {
	id := r.MustStr("id")
	from := r.MustStr("from")
	fromAccountType := r.MustStr("from_account_type")
	fromAT := proto.AccountType_value[fromAccountType]
	to := r.MustStr("to")
	toAccuontType := r.MustStr("to_account_type")
	toAT := proto.AccountType_value[toAccuontType]
	asset := r.MustStr("asset")
	amount := r.MustStr("amount")
	amountUint, _ := num.UintFromString(amount, 10)
	startEpoch, _ := num.UintFromString(r.MustStr("start_epoch"), 10)
	endEpoch := r.MustStr("end_epoch")
	var endEpochPtr *uint64
	if len(endEpoch) > 0 {
		endEpochUint, _ := num.UintFromString(r.MustStr("end_epoch"), 10)
		endEpochUint64 := endEpochUint.Uint64()
		endEpochPtr = &endEpochUint64
	}

	var dispatchStrategy *proto.DispatchStrategy
	if len(r.Str("metric")) > 0 {
		mkts := strings.Split(r.MustStr("markets"), ",")
		if len(mkts) == 1 && mkts[0] == "" {
			mkts = []string{}
		}
		dispatchStrategy = &proto.DispatchStrategy{
			AssetForMetric:       r.MustStr("metric_asset"),
			Markets:              mkts,
			Metric:               proto.DispatchMetric(proto.DispatchMetric_value[r.MustStr("metric")]),
			DistributionStrategy: proto.DistributionStrategy_DISTRIBUTION_STRATEGY_PRO_RATA,
			LockPeriod:           1,
			EntityScope:          proto.EntityScope_ENTITY_SCOPE_INDIVIDUALS,
			IndividualScope:      proto.IndividualScope_INDIVIDUAL_SCOPE_ALL,
			WindowLength:         1,
		}
	}

	factor := num.MustDecimalFromString(r.MustStr("factor"))
	recurring := &types.RecurringTransfer{
		TransferBase: &types.TransferBase{
			ID:              id,
			From:            from,
			FromAccountType: types.AccountType(fromAT),
			To:              to,
			ToAccountType:   types.AccountType(toAT),
			Asset:           asset,
			Amount:          amountUint,
		},
		StartEpoch:       startEpoch.Uint64(),
		EndEpoch:         endEpochPtr,
		Factor:           factor,
		DispatchStrategy: dispatchStrategy,
	}
	return recurring
}

func PartiesCancelTransfers(
	engine *banking.Engine,
	table *godog.Table,
) error {
	errs := []error{}
	for _, r := range parseOneOffCancellationTable(table) {
		err := engine.CancelTransferFunds(context.Background(), &types.CancelTransferFunds{
			Party:      r.MustStr("party"),
			TransferID: r.MustStr("transfer_id"),
		})
		if len(r.Str("error")) > 0 || err != nil {
			expected := r.Str("error")
			actual := ""
			if err != nil {
				actual = err.Error()
			}
			if expected != actual {
				errs = append(errs, errors.New(r.MustStr("transfer_id")+" expected "+expected+" but got "+actual))
			}
		}
	}
	if len(errs) > 0 {
		return ErrStack(errs)
	}
	return nil
}

func parseOneOffCancellationTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"party", "transfer_id",
	}, []string{"error"})
}
