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
			e.broker.Send(events.NewRecurringTransferFundsEventWithReason(ctx, transfer, err.Error(), e.getGameID(transfer)))
		} else {
			e.broker.Send(events.NewRecurringTransferFundsEvent(ctx, transfer, e.getGameID(transfer)))
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

		if hasAsset && len(transfer.DispatchStrategy.Markets) > 0 {
			asset := transfer.DispatchStrategy.AssetForMetric
			for _, mid := range transfer.DispatchStrategy.Markets {
				if !e.marketActivityTracker.MarketTrackedForAsset(mid, asset) {
					transfer.Status = types.TransferStatusRejected
					e.log.Debug("cannot transfer funds, invalid market for dispatch asset",
						logging.String("mid", mid),
						logging.String("asset", asset),
					)
					return errors.New("could not transfer funds, invalid market for dispatch asset")
				}
			}
		}
	}

	if err := transfer.IsValid(); err != nil {
		transfer.Status = types.TransferStatusRejected
		return err
	}

	if err := e.ensureMinimalTransferAmount(a, transfer.Amount, transfer.FromAccountType, transfer.From, transfer.FromDerivedKey); err != nil {
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
	e.registerDispatchStrategy(transfer.DispatchStrategy)

	return nil
}

func (e *Engine) getGameID(transfer *types.RecurringTransfer) *string {
	if transfer.DispatchStrategy == nil {
		return nil
	}
	gameID := e.hashDispatchStrategy(transfer.DispatchStrategy)
	return &gameID
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
			e.marketActivityTracker.GameFinished(hash)
		}
	}
}

func isSimilar(dispatchStrategy1, dispatchStrategy2 *vegapb.DispatchStrategy) bool {
	p1, _ := proto.Marshal(dispatchStrategy1)
	hash1 := hex.EncodeToString(crypto.Hash(p1))

	p2, _ := proto.Marshal(dispatchStrategy2)
	hash2 := hex.EncodeToString(crypto.Hash(p2))
	return hash1 == hash2
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
func (e *Engine) dispatchRequired(ctx context.Context, ds *vegapb.DispatchStrategy) bool {
	required, ok := e.dispatchRequiredCache[e.hashDispatchStrategy(ds)]
	if ok {
		return required
	}
	defer func() { e.dispatchRequiredCache[e.hashDispatchStrategy(ds)] = required }()
	switch ds.Metric {
	case vegapb.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_PAID,
		vegapb.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_RECEIVED,
		vegapb.DispatchMetric_DISPATCH_METRIC_LP_FEES_RECEIVED,
		vegapb.DispatchMetric_DISPATCH_METRIC_AVERAGE_NOTIONAL,
		vegapb.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN,
		vegapb.DispatchMetric_DISPATCH_METRIC_RETURN_VOLATILITY,
		vegapb.DispatchMetric_DISPATCH_METRIC_REALISED_RETURN,
		vegapb.DispatchMetric_DISPATCH_METRIC_ELIGIBLE_ENTITIES:
		if ds.EntityScope == vegapb.EntityScope_ENTITY_SCOPE_INDIVIDUALS {
			hasNonZeroMetric := false
			partyMetrics := e.marketActivityTracker.CalculateMetricForIndividuals(ctx, ds)
			gs := events.NewPartyGameScoresEvent(ctx, int64(e.currentEpoch), e.hashDispatchStrategy(ds), e.timeService.GetTimeNow(), partyMetrics)
			e.broker.Send(gs)
			hasEligibleParties := false
			for _, pm := range partyMetrics {
				if !pm.Score.IsZero() {
					hasNonZeroMetric = true
				}
				if pm.IsEligible {
					hasEligibleParties = true
				}
				if hasNonZeroMetric && hasEligibleParties {
					break
				}
			}
			required = hasNonZeroMetric || (hasEligibleParties && (ds.DistributionStrategy == vegapb.DistributionStrategy_DISTRIBUTION_STRATEGY_RANK || ds.DistributionStrategy == vegapb.DistributionStrategy_DISTRIBUTION_STRATEGY_RANK_LOTTERY))
			return required
		} else {
			tcs, pcs := e.marketActivityTracker.CalculateMetricForTeams(ctx, ds)
			gs := events.NewTeamGameScoresEvent(ctx, int64(e.currentEpoch), e.hashDispatchStrategy(ds), e.timeService.GetTimeNow(), tcs, pcs)
			e.broker.Send(gs)
			required = len(tcs) > 0
			return required
		}
	case vegapb.DispatchMetric_DISPATCH_METRIC_VALIDATOR_RANKING:
		required = true
		return required
	}
	required = false
	return required
}

func (e *Engine) scaleAmountByTargetNotional(ds *vegapb.DispatchStrategy, amount *num.Uint) *num.Uint {
	if ds == nil {
		return amount
	}
	if ds.TargetNotionalVolume == nil {
		return amount
	}
	actualVolumeInWindow := e.marketActivityTracker.GetNotionalVolumeForAsset(ds.AssetForMetric, ds.Markets, int(ds.WindowLength))
	if actualVolumeInWindow.IsZero() {
		return num.UintZero()
	}
	targetNotional := num.MustUintFromString(*ds.TargetNotionalVolume, 10)
	ratio := num.MinD(actualVolumeInWindow.ToDecimal().Div(targetNotional.ToDecimal()), num.DecimalOne())
	amt, _ := num.UintFromDecimal(ratio.Mul(amount.ToDecimal()))
	return amt
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

		// if the transfer should have been ended and has not, end it now.
		if v.EndEpoch != nil && *v.EndEpoch < e.currentEpoch {
			v.Status = types.TransferStatusDone
			transfersDone = append(transfersDone, events.NewRecurringTransferFundsEvent(ctx, v, e.getGameID(v)))
			doneIDs = append(doneIDs, v.ID)
			continue
		}

		if v.DispatchStrategy != nil && v.DispatchStrategy.TransferInterval != nil &&
			((newEpoch-v.StartEpoch+1) < uint64(*v.DispatchStrategy.TransferInterval) ||
				(newEpoch-v.StartEpoch+1)%uint64(*v.DispatchStrategy.TransferInterval) != 0) {
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

		// scale transfer amount as necessary
		amount = e.scaleAmountByTargetNotional(v.DispatchStrategy, amount)

		// check if the amount is still enough
		// ensure asset exists
		a, err := e.assets.Get(v.Asset)
		if err != nil {
			// this should not be possible, asset was validated at first when
			// accepting the transfer
			e.log.Panic("this should never happen", logging.Error(err))
		}

		if err = e.ensureMinimalTransferAmount(a, amount, v.FromAccountType, v.From, v.FromDerivedKey); err != nil {
			v.Status = types.TransferStatusStopped
			transfersDone = append(transfersDone,
				events.NewRecurringTransferFundsEventWithReason(ctx, v, err.Error(), e.getGameID(v)))
			doneIDs = append(doneIDs, v.ID)
			continue
		}

		// NB: if no dispatch strategy is defined - the transfer is made to the account as defined in the transfer.
		// If a dispatch strategy is defined but there are no relevant markets in scope or no fees in scope then no transfer is made!
		var resps []*types.LedgerMovement
		var r []*types.LedgerMovement
		if v.DispatchStrategy == nil {
			resps, err = e.processTransfer(
				ctx, a, v.From, v.To, "", v.FromAccountType, v.ToAccountType, amount, v.Reference, v.ID, newEpoch,
				v.FromDerivedKey, nil, // last is eventual oneoff, which this is not
			)
		} else {
			// check if the amount + fees can be covered by the party issuing the transfer
			if err = e.ensureFeeForTransferFunds(a, amount, v.From, v.FromAccountType, v.FromDerivedKey, v.To); err == nil {
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
							ctx, a, v.From, v.To, fms.Market, v.FromAccountType, v.ToAccountType, amt, v.Reference, v.ID,
							newEpoch, v.FromDerivedKey, nil, // last is eventual oneoff, which this is not
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
				if e.dispatchRequired(ctx, v.DispatchStrategy) {
					p, _ := proto.Marshal(v.DispatchStrategy)
					hash := hex.EncodeToString(crypto.Hash(p))
					r, err = e.processTransfer(
						ctx, a, v.From, v.To, hash, v.FromAccountType, v.ToAccountType, amount, v.Reference, v.ID, newEpoch,
						v.FromDerivedKey, nil, // last is eventual oneoff, which this is not
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
				events.NewRecurringTransferFundsEventWithReason(ctx, v, err.Error(), e.getGameID(v)))
			doneIDs = append(doneIDs, v.ID)
			continue
		}

		tresps = append(tresps, resps...)

		// if we don't have anymore
		if v.EndEpoch != nil && *v.EndEpoch == e.currentEpoch {
			v.Status = types.TransferStatusDone
			transfersDone = append(transfersDone, events.NewRecurringTransferFundsEvent(ctx, v, e.getGameID(v)))
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
