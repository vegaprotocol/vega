package banking

import (
	"context"
	"errors"
	"fmt"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
)

var ErrStartEpochInThePast = errors.New("start epoch in the past")

type transferForEpoch struct {
	// transfers using the same account type
	transfer              *types.Transfer
	accountType           types.AccountType
	feeTransfer           *types.Transfer
	feeTransferAccounType types.AccountType
}

type recurringTransfer struct {
	// to send events
	recurring *types.RecurringTransfer
	// epoch to transfers
	transfers map[uint64]*transferForEpoch
	reference string
}

func (e *Engine) recurringTransfer(
	ctx context.Context,
	transfer *types.RecurringTransfer,
) error {
	defer func() {
		e.broker.Send(events.NewRecurringTransferFundsEvent(ctx, transfer))
	}()

	// ensure asset exists
	if _, err := e.assets.Get(transfer.Asset); err != nil {
		transfer.Status = types.TransferStatusRejected
		e.log.Debug("cannot transfer funds, invalid asset", logging.Error(err))
		return fmt.Errorf("could not transfer funds: %w", err)
	}

	if err := transfer.IsValid(); err != nil {
		transfer.Status = types.TransferStatusRejected
		return err
	}

	// can't create transfer with start epoch in the past
	if transfer.StartEpoch <= e.currentEpoch {
		transfer.Status = types.TransferStatusRejected
		return ErrStartEpochInThePast
	}

	// now calculate the transfers
	var (
		tfe            = map[uint64]*transferForEpoch{}
		startAmount    = transfer.Amount.ToDecimal()
		startEpoch     = num.NewUint(transfer.StartEpoch).ToDecimal()
		payTransfer    *types.Transfer
		payFeeTransfer *types.Transfer
	)

	for i := transfer.StartEpoch; transfer.EndEpoch > i; i++ {
		currentEpoch := num.NewUint(i).ToDecimal()
		amount, _ := num.UintFromDecimal(startAmount.Mul(transfer.Factor.Pow(currentEpoch.Sub(startEpoch))))

		fmt.Printf("AMOUNT: %v\n", amount.String())

		fromTransfer, toTransfer := e.makeTransfers(
			transfer.From, transfer.To, transfer.Asset, amount)

		if payTransfer != nil {
			payTransfer = payTransfer.Merge(fromTransfer)
		} else {
			payTransfer = fromTransfer
		}

		// build the fee, we ensure the party have the
		// amount down the line
		feeTransfer := e.makeFeeTransferForTransferFunds(
			amount.Clone(), transfer.From, transfer.Asset, transfer.FromAccountType,
		)

		if payFeeTransfer != nil {
			payFeeTransfer = payFeeTransfer.Merge(feeTransfer)
		} else {
			payFeeTransfer = payFeeTransfer.Clone()
		}

		// add the new fee to the total amount
		totalFee.Add(totalFee, feeTransfer.Amount.Amount)

		// add the map
		tfe[i] = &transferForEpoch{
			transfer:              toTransfer,
			accountType:           transfer.ToAccountType,
			feeTransfer:           feeTransfer,
			feeTransferAccounType: transfer.FromAccountType,
		}
	}

	// now all our epoch are processed, we can
	err := e.ensureRecurringTransferFee(
		payTransfer.Amount.Amount.Clone(),
		totalFee,
		transfer.From,
		transfer.Asset,
		transfer.FromAccountType,
	)
	if err != nil {
		transfer.Status = types.TransferStatusRejected
		return err
	}

	// now pay the funds in the pool
	tresps, err := e.col.TransferFunds(
		ctx,
		[]*types.Transfer{payTransfer},
		[]types.AccountType{transfer.FromAccountType},
		[]string{transfer.Reference},
		nil,
		nil,
	)
	if err != nil {
		transfer.Status = types.TransferStatusRejected
		return err
	}

	// now all is good, we can just save this transfer, and send events
	e.recurringTransfers[transfer.ID] = &recurringTransfer{
		recurring: transfer,
		transfers: tfe,
		reference: transfer.Reference,
	}

	// send events
	e.broker.Send(events.NewTransferResponse(ctx, tresps))

	return nil
}

func (e *Engine) makeRecurringFeeTransfer(
	amount *num.Uint,
	from, asset string,
	fromAccountType types.AccountType,
) *types.Transfer {
	// first we calculate the fee
	feeAmount, _ := num.UintFromDecimal(amount.ToDecimal().Mul(e.transferFeeFactor))

	switch fromAccountType {
	case types.AccountTypeGeneral:
	default:
		e.log.Panic("from account not supported",
			logging.String("account-type", fromAccountType.String()),
			logging.String("asset", asset),
			logging.String("from", from),
		)
	}

	return &types.Transfer{
		Owner: from,
		Amount: &types.FinancialAmount{
			Amount: feeAmount.Clone(),
			Asset:  asset,
		},
		Type:      types.TransferTypeInfrastructureFeePay,
		MinAmount: feeAmount,
	}
}

func (e *Engine) ensureRecurringTransferFee(
	amount, feeAmount *num.Uint,
	from, asset string,
	fromAccountType types.AccountType,
) error {
	var (
		totalAmount = num.Sum(amount, feeAmount)
		account     *types.Account
		err         error
	)
	switch fromAccountType {
	case types.AccountTypeGeneral:
		account, err = e.col.GetPartyGeneralAccount(from, asset)
		if err != nil {
			return err
		}

	default:
		e.log.Panic("from account not supported",
			logging.String("account-type", fromAccountType.String()),
			logging.String("asset", asset),
			logging.String("from", from),
		)
	}

	if account.Balance.LT(totalAmount) {
		e.log.Debug("not enough funds to transfer",
			logging.BigUint("amount", amount),
			logging.BigUint("fee", feeAmount),
			logging.BigUint("total-amount", totalAmount),
			logging.BigUint("account-balance", account.Balance),
			logging.String("account-type", fromAccountType.String()),
			logging.String("asset", asset),
			logging.String("from", from),
		)
		return ErrNotEnoughFundsToTransfer
	}

	return nil
}

func (e *Engine) distributeRecurringTransfers(
	ctx context.Context,
	newEpoch uint64,
) error {
	var (
		evts         = []events.Event{}
		transfers    = []*types.Transfer{}
		feeTransfers = []*types.Transfer{}
		accountTypes = []types.AccountType{}
		references   = []string{}
	)

	allIDs := make([]string, 0, len(e.recurringTransfers))
	for k := range e.recurringTransfers {
		allIDs = append(allIDs, k)
	}

	// iterate over all transfers
	for _, k := range allIDs {
		v := e.recurringTransfers[k]
		tfe, ok := v.transfers[newEpoch]
		if !ok {
			// no transfer for this epoch
			continue
		}
		reference := v.reference

		// delete the transfer from the map
		delete(v.transfers, newEpoch)

		// if we don't have anymore
		if len(v.transfers) <= 0 {
			v.recurring.Status = types.TransferStatusDone
			evts = append(evts, events.NewRecurringTransferFundsEvent(ctx, v.recurring))
			delete(e.recurringTransfers, k)
		}

		// add to slices
		transfers = append(transfers, tfe.transfer)
		feeTransfers = append(feeTransfers, tfe.feeTransfer)
		accountTypes = append(accountTypes, tfe.accountType)
		references = append(references, reference)
	}

	if len(transfers) <= 0 {
		return nil
	}

	// call collateral
	tresps, err := e.col.TransferFunds(
		ctx, transfers, accountTypes, references, feeTransfers, accountTypes,
	)
	if err != nil {
		return err
	}

	// send events
	e.broker.Send(events.NewTransferResponse(ctx, tresps))
	e.broker.SendBatch(evts)

	return nil
}
