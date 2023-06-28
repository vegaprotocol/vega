package banking

import (
	"context"
	"errors"
	"fmt"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
)

func (e *Engine) distributeScheduledGovernanceTransfers(ctx context.Context) {
	timepoints := []int64{}
	now := e.timeService.GetTimeNow()
	for k := range e.scheduledGovernanceTransfers {
		if now.UnixNano() >= k {
			timepoints = append(timepoints, k)
		}
	}

	for _, t := range timepoints {
		transfers := e.scheduledGovernanceTransfers[t]
		for _, gTransfer := range transfers {
			amt, err := e.processGovernanceTransfer(ctx, gTransfer)
			if err != nil {
				gTransfer.Status = types.TransferStatusStopped
				e.broker.Send(events.NewGovTransferFundsEventWithReason(ctx, gTransfer, amt, err.Error()))
			} else {
				gTransfer.Status = types.TransferStatusDone
				e.broker.Send(events.NewGovTransferFundsEvent(ctx, gTransfer, amt))
			}
		}
		delete(e.scheduledGovernanceTransfers, t)
	}
}

func (e *Engine) distributeRecurringGovernanceTransfers(ctx context.Context) {
	var (
		transfersDone = []events.Event{}
		doneIDs       = []string{}
	)

	for _, gTransfer := range e.recurringGovernanceTransfers {
		if gTransfer.Config.RecurringTransferConfig.StartEpoch > e.currentEpoch {
			continue
		}

		amount, err := e.processGovernanceTransfer(ctx, gTransfer)
		if err != nil {
			e.log.Error("error calculating transfer amount for governance transfer", logging.Error(err))
			gTransfer.Status = types.TransferStatusStopped
			transfersDone = append(transfersDone, events.NewGovTransferFundsEventWithReason(ctx, gTransfer, amount, err.Error()))
			doneIDs = append(doneIDs, gTransfer.ID)
			continue
		}

		if gTransfer.Config.RecurringTransferConfig.EndEpoch != nil && *gTransfer.Config.RecurringTransferConfig.EndEpoch == e.currentEpoch {
			gTransfer.Status = types.TransferStatusDone
			transfersDone = append(transfersDone, events.NewGovTransferFundsEvent(ctx, gTransfer, amount))
			doneIDs = append(doneIDs, gTransfer.ID)
			continue
		}
		e.broker.Send(events.NewGovTransferFundsEvent(ctx, gTransfer, amount))
	}

	if len(transfersDone) > 0 {
		for _, id := range doneIDs {
			e.deleteGovTransfer(id)
		}
		e.broker.SendBatch(transfersDone)
	}
}

func (e *Engine) deleteGovTransfer(ID string) {
	index := -1
	for i, rt := range e.recurringGovernanceTransfers {
		if rt.ID == ID {
			index = i
			break
		}
	}
	if index >= 0 {
		e.recurringGovernanceTransfers = append(e.recurringGovernanceTransfers[:index], e.recurringGovernanceTransfers[index+1:]...)
		delete(e.recurringGovernanceTransfersMap, ID)
	}
}

func (e *Engine) NewGovernanceTransfer(ctx context.Context, ID, reference string, config *types.NewTransferConfiguration) error {
	var err error
	var amount *num.Uint
	var gTransfer *types.GovernanceTransfer

	defer func() {
		if err != nil {
			e.broker.Send(events.NewGovTransferFundsEventWithReason(ctx, gTransfer, amount, err.Error()))
		} else {
			e.broker.Send(events.NewGovTransferFundsEvent(ctx, gTransfer, amount))
		}
	}()
	now := e.timeService.GetTimeNow()
	gTransfer = &types.GovernanceTransfer{
		ID:        ID,
		Reference: reference,
		Config:    config,
		Status:    types.TransferStatusPending,
		Timestamp: now,
	}
	if config.Kind == types.TransferKindOneOff {
		// one off governance transfer to be executed straight away
		if config.OneOffTransferConfig.DeliverOn == 0 || config.OneOffTransferConfig.DeliverOn < now.UnixNano() {
			amount, err = e.processGovernanceTransfer(ctx, gTransfer)
			if err != nil {
				gTransfer.Status = types.TransferStatusRejected
				return err
			}
			gTransfer.Status = types.TransferStatusDone
			return nil
		}
		// scheduled one off governance transfer
		if _, ok := e.scheduledGovernanceTransfers[config.OneOffTransferConfig.DeliverOn]; !ok {
			e.scheduledGovernanceTransfers[config.OneOffTransferConfig.DeliverOn] = []*types.GovernanceTransfer{}
		}
		e.scheduledGovernanceTransfers[config.OneOffTransferConfig.DeliverOn] = append(e.scheduledGovernanceTransfers[config.OneOffTransferConfig.DeliverOn], gTransfer)
		amount = num.UintZero()
		gTransfer.Status = types.TransferStatusPending
		return nil
	}
	// recurring governance transfer
	amount = num.UintZero()
	e.recurringGovernanceTransfers = append(e.recurringGovernanceTransfers, gTransfer)
	e.recurringGovernanceTransfersMap[ID] = gTransfer
	return nil
}

// processGovernanceTransfer process a governance transfer and emit ledger movement events.
func (e *Engine) processGovernanceTransfer(
	ctx context.Context,
	gTransfer *types.GovernanceTransfer,
) (*num.Uint, error) {
	transferAmount, err := e.CalculateGovernanceTransferAmount(gTransfer.Config.Asset, gTransfer.Config.Source, gTransfer.Config.SourceType, gTransfer.Config.FractionOfBalance, gTransfer.Config.MaxAmount, gTransfer.Config.TransferType)
	if err != nil {
		e.log.Error("failed to calculate amount for governance transfer", logging.String("proposal", gTransfer.ID), logging.String("error", err.Error()))
		return num.UintZero(), err
	}

	from := "*"
	fromMarket := gTransfer.Config.Source

	toMarket := ""
	to := gTransfer.Config.Destination
	if gTransfer.Config.DestinationType == types.AccountTypeGlobalReward {
		to = "*"
	} else if gTransfer.Config.DestinationType == types.AccountTypeInsurance {
		toMarket = to
		to = "*"
	}

	fromTransfer, toTransfer := e.makeTransfers(from, to, gTransfer.Config.Asset, fromMarket, toMarket, transferAmount)
	transfers := []*types.Transfer{fromTransfer, toTransfer}
	accountTypes := []types.AccountType{gTransfer.Config.SourceType, gTransfer.Config.DestinationType}
	references := []string{gTransfer.Reference, gTransfer.Reference}
	tresps, err := e.col.GovernanceTransferFunds(ctx, transfers, accountTypes, references)
	if err != nil {
		return num.UintZero(), err
	}

	e.broker.Send(events.NewLedgerMovements(ctx, tresps))
	return transferAmount, nil
}

// CalculateGovernanceTransferAmount calculates the balance of a governance transfer as follows:
//
// transfer_amount = min(
//
//	proposal.fraction_of_balance * source.balance,
//	proposal.amount,
//	NETWORK_MAX_AMOUNT,
//	NETWORK_MAX_FRACTION * source.balance
//
// )
// where
// NETWORK_MAX_AMOUNT is a network parameter specifying the maximum absolute amount that can be transferred by governance for the source account type
// NETWORK_MAX_FRACTION is a network parameter specifying the maximum fraction of the balance that can be transferred by governance for the source account type (must be <= 1)
//
// If type is "all or nothing" then the transfer will only proceed if:
//
//	transfer_amount == min(proposal.fraction_of_balance * source.balance,proposal.amount).
func (e *Engine) CalculateGovernanceTransferAmount(asset string, market string, accountType types.AccountType, fraction num.Decimal, amount *num.Uint, transferType vegapb.GovernanceTransferType) (*num.Uint, error) {
	balance, err := e.col.GetSystemAccountBalance(asset, market, accountType)
	if err != nil {
		e.log.Error("could not find system account balance for", logging.String("asset", asset), logging.String("market", market), logging.String("account-type", accountType.String()))
		return nil, err
	}

	amountFromMaxFraction, _ := num.UintFromDecimal(e.maxGovTransferFraction.Mul(balance.ToDecimal()))
	amountFromProposalFraction, _ := num.UintFromDecimal(fraction.Mul(balance.ToDecimal()))
	min1 := num.Min(amountFromMaxFraction, amountFromProposalFraction)
	min2 := num.Min(amount, e.maxGovTransferAmount)
	amt := num.Min(min1, min2)

	if transferType == vegapb.GovernanceTransferType_GOVERNANCE_TRANSFER_TYPE_ALL_OR_NOTHING && amt.NEQ(num.Min(amountFromProposalFraction, amount)) {
		e.log.Error("could not process governance transfer with type all of nothing", logging.String("transfer-amount", amt.String()), logging.String("fraction-of-balance", amountFromProposalFraction.String()), logging.String("amount", amount.String()))
		return nil, errors.New("invalid transfer amount for transfer type all or nothing")
	}

	return amt, nil
}

func (e *Engine) VerifyGovernanceTransfer(transfer *types.NewTransferConfiguration) error {
	if transfer == nil {
		return errors.New("missing transfer configuration")
	}

	// check source type is valid
	if transfer.SourceType != types.AccountTypeInsurance &&
		transfer.SourceType != types.AccountTypeGlobalReward {
		return errors.New("invalid source type for governance transfer")
	}

	// check destination type is valid
	if transfer.DestinationType != types.AccountTypeGeneral &&
		transfer.DestinationType != types.AccountTypeGlobalReward &&
		transfer.DestinationType != types.AccountTypeInsurance {
		return errors.New("invalid destination for governance transfer")
	}

	// check asset is not empty
	if len(transfer.Asset) == 0 {
		return errors.New("missing asset for governance transfer")
	}

	// check source account exists
	if _, err := e.col.GetSystemAccountBalance(transfer.Asset, transfer.Source, transfer.SourceType); err != nil {
		return err
	}

	// check destination account exists
	if transfer.DestinationType == types.AccountTypeGeneral {
		if _, err := e.col.GetPartyGeneralAccount(transfer.Destination, transfer.Asset); err != nil {
			return err
		}
	} else {
		if _, err := e.col.GetSystemAccountBalance(transfer.Asset, transfer.Destination, transfer.DestinationType); err != nil {
			return err
		}
	}

	// check transfer type is specified
	if transfer.TransferType == vegapb.GovernanceTransferType_GOVERNANCE_TRANSFER_TYPE_UNSPECIFIED {
		return errors.New("invalid governance transfer type")
	}

	// check max amount is positive
	if transfer.MaxAmount == nil || transfer.MaxAmount.IsNegative() || transfer.MaxAmount.IsZero() {
		return errors.New("invalid max amount for governance transfer")
	}

	// check fraction of balance is positive
	if !transfer.FractionOfBalance.IsPositive() {
		return errors.New("invalid fraction of balance for governance transfer")
	}

	// verify recurring transfer starting epoch is not in the past
	if transfer.RecurringTransferConfig != nil && transfer.RecurringTransferConfig.StartEpoch < e.currentEpoch {
		return ErrStartEpochInThePast
	}

	return nil
}

func (e *Engine) VerifyCancelGovernanceTransfer(transferID string) error {
	if _, ok := e.recurringGovernanceTransfersMap[transferID]; !ok {
		return fmt.Errorf("Governance transfer %s not found", transferID)
	}
	return nil
}
