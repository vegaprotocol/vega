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

package banking

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/proto"
	"code.vegaprotocol.io/vega/logging"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
)

var (
	ErrStartEpochInThePast                                     = errors.New("start epoch in the past")
	ErrCannotSubmitDuplicateRecurringTransferWithSameFromAndTo = errors.New("cannot submit duplicate recurring transfer with same from and to")
)

func (e *Engine) recurringTransfer(
	ctx context.Context,
	transfer *types.RecurringTransfer,
) (err error) {
	defer func() {
		if err != nil {
			e.broker.Send(events.NewRecurringTransferFundsEventWithReason(ctx, transfer, err.Error()))
		} else {
			e.broker.Send(events.NewRecurringTransferFundsEvent(ctx, transfer))
		}
	}()

	// ensure asset exists
	a, err := e.assets.Get(transfer.Asset)
	if err != nil {
		transfer.Status = types.TransferStatusRejected
		e.log.Debug("cannot transfer funds, invalid asset", logging.Error(err))
		return fmt.Errorf("could not transfer funds: %w", err)
	}

	if transfer.DispatchStrategy != nil {
		hasAsset := len(transfer.DispatchStrategy.AssetForMetric) > 0
		// ensure the asset transfer is correct
		if hasAsset {
			_, err := e.assets.Get(transfer.DispatchStrategy.AssetForMetric)
			if err != nil {
				transfer.Status = types.TransferStatusRejected
				e.log.Debug("cannot transfer funds, invalid asset for metric", logging.Error(err))
				return fmt.Errorf("could not transfer funds, invalid asset for metric: %w", err)
			}
		}
	}

	if err := transfer.IsValid(); err != nil {
		transfer.Status = types.TransferStatusRejected
		return err
	}

	if err := e.ensureMinimalTransferAmount(a, transfer.Amount); err != nil {
		transfer.Status = types.TransferStatusRejected
		return err
	}

	if err := e.ensureNoRecurringTransferDuplicates(transfer); err != nil {
		transfer.Status = types.TransferStatusRejected
		return err
	}

	// can't create transfer with start epoch in the past
	if transfer.StartEpoch < e.currentEpoch {
		transfer.Status = types.TransferStatusRejected
		return ErrStartEpochInThePast
	}

	// from here all sounds OK, we can add the transfer
	// in the recurringTransfer map/slice
	e.recurringTransfers = append(e.recurringTransfers, transfer)
	e.recurringTransfersMap[transfer.ID] = transfer

	if transfer.DispatchStrategy != nil {
		hash := e.hashDispatchStrategy(transfer.DispatchStrategy)
		if _, ok := e.hashToStrategy[hash]; !ok {
			e.hashToStrategy[hash] = &dispatchStrategyCacheEntry{ds: transfer.DispatchStrategy, refCount: 1}
		} else {
			e.hashToStrategy[hash].refCount++
		}
	}

	return nil
}

func (e *Engine) hashDispatchStrategy(ds *vegapb.DispatchStrategy) string {
	p, err := proto.Marshal(ds)
	if err != nil {
		e.log.Panic("failed to marshal dispatch strategy", logging.String("dispatch-strategy", ds.String()))
	}
	return hex.EncodeToString(crypto.Hash(p))
}

func (e *Engine) registerDispatchStrategy(ds *vegapb.DispatchStrategy) {
	if ds == nil {
		return
	}
	hash := e.hashDispatchStrategy(ds)
	if _, ok := e.hashToStrategy[hash]; !ok {
		e.hashToStrategy[hash] = &dispatchStrategyCacheEntry{ds: ds, refCount: 1}
	} else {
		e.hashToStrategy[hash].refCount++
	}
}

func (e *Engine) unregisterDispatchStrategy(ds *vegapb.DispatchStrategy) {
	if ds == nil {
		return
	}
	hash := e.hashDispatchStrategy(ds)
	e.hashToStrategy[hash].refCount--
}

func (e *Engine) cleanupStaleDispatchStrategies() {
	for hash, dsc := range e.hashToStrategy {
		if dsc.refCount == 0 {
			delete(e.hashToStrategy, hash)
		}
	}
}

func compareStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

func isSimilar(dispatchStrategy1, dispatchStrategy2 *vegapb.DispatchStrategy) bool {
	return (dispatchStrategy1 == nil && dispatchStrategy2 == nil) ||
		(dispatchStrategy1 != nil && dispatchStrategy2 != nil && dispatchStrategy1.AssetForMetric == dispatchStrategy2.AssetForMetric && dispatchStrategy1.Metric == dispatchStrategy2.Metric && compareStringSlices(dispatchStrategy1.Markets, dispatchStrategy2.Markets))
}

func (e *Engine) ensureNoRecurringTransferDuplicates(
	transfer *types.RecurringTransfer,
) error {
	for _, v := range e.recurringTransfers {
		// NB: 2 transfers are identical and not allowed if they have the same from, to, type AND the same dispatch strategy.
		// This is needed so that we can for example setup transfer of USDT from one PK to the reward account with type maker fees received with dispatch based on the asset ETH -
		// and then a similar transfer of USDT from the same PK to the same reward type but with different dispatch strategy - one tracking markets for the asset DAI.
		if v.From == transfer.From && v.To == transfer.To && v.Asset == transfer.Asset && v.FromAccountType == transfer.FromAccountType && v.ToAccountType == transfer.ToAccountType && isSimilar(v.DispatchStrategy, transfer.DispatchStrategy) {
			return ErrCannotSubmitDuplicateRecurringTransferWithSameFromAndTo
		}
	}

	return nil
}

// dispatchRequired returns true if the metric for any qualifying entity in scope none zero.
// NB1: the check for market value metric should be done separately
// NB2: for validator ranking this will always return true as it is assumed that for the network to resume there must always be
// a validator with non zero ranking.
func (e *Engine) dispatchRequired(ds *vegapb.DispatchStrategy) bool {
	switch ds.Metric {
	case vegapb.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_PAID,
		vegapb.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_RECEIVED,
		vegapb.DispatchMetric_DISPATCH_METRIC_LP_FEES_RECEIVED,
		vegapb.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION,
		vegapb.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN,
		vegapb.DispatchMetric_DISPATCH_METRIC_RETURN_VOLATILITY:
		if ds.EntityScope == vegapb.EntityScope_ENTITY_SCOPE_INDIVIDUALS {
			return len(e.marketActivityTracker.CalculateMetricForIndividuals(ds)) > 0
		} else {
			tcs, _ := e.marketActivityTracker.CalculateMetricForTeams(ds)
			return len(tcs) > 0
		}
	case vegapb.DispatchMetric_DISPATCH_METRIC_VALIDATOR_RANKING:
		return true
	}
	return false
}

func (e *Engine) distributeRecurringTransfers(ctx context.Context, newEpoch uint64) {
	var (
		transfersDone = []events.Event{}
		doneIDs       = []string{}
		tresps        = []*types.LedgerMovement{}
		currentEpoch  = num.NewUint(newEpoch).ToDecimal()
	)

	// iterate over all transfers
	for _, v := range e.recurringTransfers {
		if v.StartEpoch > newEpoch {
			// not started
			continue
		}

		var (
			startEpoch  = num.NewUint(v.StartEpoch).ToDecimal()
			startAmount = v.Amount.ToDecimal()
			amount, _   = num.UintFromDecimal(
				startAmount.Mul(
					v.Factor.Pow(currentEpoch.Sub(startEpoch)),
				),
			)
		)

		// check if the amount is still enough
		// ensure asset exists
		a, err := e.assets.Get(v.Asset)
		if err != nil {
			// this should not be possible, asset was validated at first when
			// accepting the transfer
			e.log.Panic("this should never happen", logging.Error(err))
		}

		if err = e.ensureMinimalTransferAmount(a, amount); err != nil {
			v.Status = types.TransferStatusStopped
			transfersDone = append(transfersDone,
				events.NewRecurringTransferFundsEventWithReason(ctx, v, err.Error()))
			doneIDs = append(doneIDs, v.ID)
			continue
		}

		// NB: if no dispatch strategy is defined - the transfer is made to the account as defined in the transfer.
		// If a dispatch strategy is defined but there are no relevant markets in scope or no fees in scope then no transfer is made!
		var resps []*types.LedgerMovement
		var r []*types.LedgerMovement
		if v.DispatchStrategy == nil {
			resps, err = e.processTransfer(
				ctx, v.From, v.To, v.Asset, "", v.FromAccountType, v.ToAccountType, amount, v.Reference, nil, // last is eventual oneoff, which this is not
			)
		} else {
			// check if the amount + fees can be covered by the party issuing the transfer
			if _, err = e.ensureFeeForTransferFunds(amount, v.From, v.Asset, v.FromAccountType); err == nil {
				// NB: if the metric is market value we're going to transfer the bonus if any directly
				// to the market account of the asset/reward type - this is similar to previous behaviour and
				// different to how all other metric based rewards behave. The reason is that we need the context of the funder
				// and this context is lost when the transfer has already gone through
				if v.DispatchStrategy.Metric == vegapb.DispatchMetric_DISPATCH_METRIC_MARKET_VALUE {
					marketProposersScore := e.marketActivityTracker.GetMarketsWithEligibleProposer(v.DispatchStrategy.AssetForMetric, v.DispatchStrategy.Markets, v.Asset, v.From)
					for _, fms := range marketProposersScore {
						amt, _ := num.UintFromDecimal(amount.ToDecimal().Mul(fms.Score))
						if amt.IsZero() {
							continue
						}
						r, err = e.processTransfer(
							ctx, v.From, v.To, v.Asset, fms.Market, v.FromAccountType, v.ToAccountType, amt, v.Reference, nil, // last is eventual oneoff, which this is not
						)
						if err != nil {
							e.log.Error("failed to process transfer",
								logging.String("from", v.From),
								logging.String("to", v.To),
								logging.String("asset", v.Asset),
								logging.String("market", fms.Market),
								logging.String("from-account-type", v.FromAccountType.String()),
								logging.String("to-account-type", v.ToAccountType.String()),
								logging.String("amount", amt.String()),
								logging.String("reference", v.Reference),
								logging.Error(err))
							break
						}
						if fms.Score.IsPositive() {
							e.marketActivityTracker.MarkPaidProposer(v.DispatchStrategy.AssetForMetric, fms.Market, v.Asset, v.DispatchStrategy.Markets, v.From)
						}
						resps = append(resps, r...)
					}
				}
				// for any other metric, we transfer the funds (full amount) to the reward account of the asset/reward_type/market=hash(dispatch_strategy)
				if e.dispatchRequired(v.DispatchStrategy) {
					p, _ := proto.Marshal(v.DispatchStrategy)
					hash := hex.EncodeToString(crypto.Hash(p))
					r, err = e.processTransfer(
						ctx, v.From, v.To, v.Asset, hash, v.FromAccountType, v.ToAccountType, amount, v.Reference, nil, // last is eventual oneoff, which this is not
					)
					if err != nil {
						e.log.Error("failed to process transfer", logging.Error(err))
					}
					resps = append(resps, r...)
				}
			} else {
				err = fmt.Errorf("could not pay the fee for transfer: %w", err)
			}
		}
		if err != nil {
			e.log.Info("transferred stopped", logging.Error(err))
			v.Status = types.TransferStatusStopped
			transfersDone = append(transfersDone,
				events.NewRecurringTransferFundsEventWithReason(ctx, v, err.Error()))
			doneIDs = append(doneIDs, v.ID)
			continue
		}

		tresps = append(tresps, resps...)

		// if we don't have anymore
		if v.EndEpoch != nil && *v.EndEpoch == e.currentEpoch {
			v.Status = types.TransferStatusDone
			transfersDone = append(transfersDone, events.NewRecurringTransferFundsEvent(ctx, v))
			doneIDs = append(doneIDs, v.ID)
		}
	}

	// send events
	if len(tresps) > 0 {
		e.broker.Send(events.NewLedgerMovements(ctx, tresps))
	}
	if len(transfersDone) > 0 {
		for _, id := range doneIDs {
			e.deleteTransfer(id)
		}
		// also set the state change
		e.broker.SendBatch(transfersDone)
	}
}

func (e *Engine) deleteTransfer(ID string) {
	index := -1
	for i, rt := range e.recurringTransfers {
		if rt.ID == ID {
			index = i
			e.unregisterDispatchStrategy(rt.DispatchStrategy)
			break
		}
	}
	if index >= 0 {
		e.recurringTransfers = append(e.recurringTransfers[:index], e.recurringTransfers[index+1:]...)
		delete(e.recurringTransfersMap, ID)
	}
}
