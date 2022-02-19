package steps

import (
	"context"
	"errors"
	"time"

	proto "code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/banking"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
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
	}, []string{"error"})
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
		transfer, _ := rowToRecurringTransfer(r)
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
	}, []string{"error"})
}

func rowToRecurringTransfer(r RowWrapper) (*types.RecurringTransfer, error) {
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
	start_epoch, _ := num.UintFromString(r.MustStr("start_epoch"), 10)
	end_epoch := r.MustStr("end_epoch")
	var end_epoch_ptr *uint64
	if len(end_epoch) > 0 {
		end_epoch_uint, _ := num.UintFromString(r.MustStr("end_epoch"), 10)
		end_epoch_uint64 := end_epoch_uint.Uint64()
		end_epoch_ptr = &end_epoch_uint64
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
		StartEpoch: start_epoch.Uint64(),
		EndEpoch:   end_epoch_ptr,
		Factor:     factor,
	}
	return recurring, nil
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
