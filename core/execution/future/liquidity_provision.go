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

package future

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"code.vegaprotocol.io/vega/core/execution/common"
	"code.vegaprotocol.io/vega/core/idgeneration"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/positions"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
)

var ErrCommitmentAmountTooLow = errors.New("commitment amount is too low")

// SubmitLiquidityProvision forwards a LiquidityProvisionSubmission to the Liquidity Engine.
func (m *Market) SubmitLiquidityProvision(
	ctx context.Context,
	sub *types.LiquidityProvisionSubmission,
	party, deterministicID string,
) (err error,
) {
	defer m.onTxProcessed()

	m.idgen = idgeneration.New(deterministicID)
	defer func() { m.idgen = nil }()

	if !m.canSubmitCommitment() {
		return common.ErrCommitmentSubmissionNotAllowed
	}

	var (
		// this is use to specified that the lp may need to be cancelled
		needsCancel bool
		// his specifies that the changes on the bond account have to be
		// rolled back
		needsBondRollback bool
	)

	if err := m.ensureLPCommitmentAmount(sub.CommitmentAmount); err != nil {
		return err
	}

	if err := m.liquidity.SubmitLiquidityProvision(ctx, sub, party, m.idgen); err != nil {
		return err
	}

	// add the party to the list of all parties involved with
	// this market
	m.addParty(party)

	defer func() {
		if err == nil || !needsCancel {
			return
		}
		if newerr := m.liquidity.RejectLiquidityProvision(ctx, party); newerr != nil {
			m.log.Debug("unable to submit cancel liquidity provision submission",
				logging.String("party", party),
				logging.String("id", deterministicID),
				logging.Error(newerr))
			err = fmt.Errorf("%v, %w", err, newerr)
		}
	}()

	// we will need both bond account and the margin account, let's create
	// them now
	bondAcc, err := m.collateral.GetOrCreatePartyBondAccount(ctx, party, m.GetID(), m.settlementAsset)
	if err != nil {
		// error happen, we can't even have the bond account taken
		// if this is not an amendment, we cancel the liquidity provision
		needsCancel = true
		return err
	}
	_, err = m.collateral.CreatePartyMarginAccount(ctx, party, m.GetID(), m.settlementAsset)
	if err != nil {
		needsCancel = true
		return err
	}

	// now we calculate the amount that needs to be moved into the account

	amount, neg := num.UintZero().Delta(sub.CommitmentAmount, bondAcc.Balance)
	ty := types.TransferTypeBondLow
	if neg {
		ty = types.TransferTypeBondHigh
	}
	transfer := &types.Transfer{
		Owner: party,
		Amount: &types.FinancialAmount{
			Amount: amount.Clone(), // clone here, we're using amount again in case of rollback
			Asset:  m.settlementAsset,
		},
		Type:      ty,
		MinAmount: amount.Clone(),
	}

	tresp, err := m.collateral.BondUpdate(ctx, m.GetID(), transfer)
	if err != nil {
		// error happen, we cannot move the funds in the bond account
		// this mean there's either an error in the collateral engine,
		// or even the party have not enough funds,
		// if this was not an amend, we'll want to delete the liquidity
		// submission
		needsCancel = true
		m.log.Debug("bond update error", logging.Error(err))
		return err
	}
	m.broker.Send(events.NewLedgerMovements(ctx, []*types.LedgerMovement{tresp}))

	// if something happen, rollback the transfer
	defer func() {
		if err == nil || !needsBondRollback {
			return
		}
		// ensure the amount is correct
		transfer.Amount.Amount = amount
		transfer.MinAmount = amount.Clone()
		if transfer.Type == types.TransferTypeBondHigh {
			transfer.Type = types.TransferTypeBondLow
		} else {
			transfer.Type = types.TransferTypeBondHigh
		}

		tresp, newerr := m.collateral.BondUpdate(ctx, m.GetID(), transfer)
		if newerr != nil {
			m.log.Debug("unable to rollback bon account topup",
				logging.String("party", party),
				logging.BigUint("amount", amount),
				logging.Error(err))
			err = fmt.Errorf("%v, %w", err, newerr)
		}
		m.broker.Send(events.NewLedgerMovements(ctx, []*types.LedgerMovement{tresp}))
	}()

	defer func() {
		// so here we check if at least we were able to get hte
		// liquidity provision in, even if orders are not deployed, we should
		// be able to calculate the shares etc
		if !needsCancel && !needsBondRollback {
			// update the MVP, if we are in opening auction, this is the total
			// amount of stake, and it'll be setup properly
			m.updateMarketValueProxy()
			// now we can update the liquidity fee to be taken
			m.updateLiquidityFee(ctx)
			// now we can setup our party stake to calculate equities
			m.equityShares.SetPartyStake(party, sub.CommitmentAmount.Clone())
			// force update of shares so they are updated for all
			_ = m.equityShares.SharesExcept(m.liquidity.GetInactiveParties())
		}
	}()

	minLpPrice, maxLpPrice, err := m.getValidLPVolumeRange()
	if err != nil {
		m.log.Debug("could not get valid lp range to call liquidity",
			logging.String("market-id", m.GetID()),
			logging.String("party", party),
			logging.Error(err),
		)
		// at this point, we were able to take the bond from the party
		// but were not able to generate the orders
		// this is likely due to the market not being ready and the liquidity
		// engine not being able to price the orders
		// we do not want to rollback anything then
		needsBondRollback = false
		needsCancel = false
		return nil
	}
	newOrders := m.liquidity.CreateInitialOrders(ctx, minLpPrice, maxLpPrice, party, m.repriceLiquidityOrder)

	if err := m.createInitialLPOrders(ctx, newOrders); err != nil {
		m.log.Debug("Could not create or update orders for a liquidity provision",
			logging.String("market-id", m.GetID()),
			logging.String("party", party),
			logging.Error(err),
		)
		// at this point we could not create or update some order for this LP
		// in the case this was a new order, we will want to cancel all that happen
		// in the case it was an amend, we'll want to do nothing
		needsBondRollback = true
		needsCancel = true
		return err
	}

	// all went well, we can remove the pending state from the
	// liquidity engine
	m.liquidity.RemovePending(party)

	return nil
}

// AmendLiquidityProvision forwards a LiquidityProvisionAmendment to the Liquidity Engine.
func (m *Market) AmendLiquidityProvision(ctx context.Context, lpa *types.LiquidityProvisionAmendment, party string, deterministicID string) (err error) {
	defer m.onTxProcessed()

	m.idgen = idgeneration.New(deterministicID)
	defer func() { m.idgen = nil }()

	if !m.canSubmitCommitment() {
		return common.ErrCommitmentSubmissionNotAllowed
	}

	if err := m.liquidity.ValidateLiquidityProvisionAmendment(lpa); err != nil {
		return err
	}

	if lpa.CommitmentAmount != nil {
		if err := m.ensureLPCommitmentAmount(lpa.CommitmentAmount); err != nil {
			return err
		}
	}

	if !m.liquidity.IsLiquidityProvider(party) {
		return common.ErrPartyNotLiquidityProvider
	}

	lp := m.liquidity.LiquidityProvisionByPartyID(party)
	if lp == nil {
		return fmt.Errorf("cannot edit liquidity provision from a non liquidity provider party (%v)", party)
	}

	// If commitment amount is not provided we keep the same
	if lpa.CommitmentAmount == nil || lpa.CommitmentAmount.IsZero() {
		lpa.CommitmentAmount = lp.CommitmentAmount
	}

	// If commitment amount is not provided we keep the same
	if lpa.Fee.IsZero() {
		lpa.Fee = lp.Fee
	}

	// If commitment amount is not provided we keep the same
	if lpa.Reference == "" {
		lpa.Reference = lp.Reference
	}

	// If orders shapes are not provided, keep the current LP orders
	if lpa.Sells == nil {
		lpa.Sells = make([]*types.LiquidityOrder, 0, len(lp.Sells))
	}
	if len(lpa.Sells) == 0 {
		for _, sell := range lp.Sells {
			lpa.Sells = append(lpa.Sells, sell.LiquidityOrder)
		}
	}
	if lpa.Buys == nil {
		lpa.Buys = make([]*types.LiquidityOrder, 0, len(lp.Buys))
	}
	if len(lpa.Buys) == 0 {
		for _, buy := range lp.Buys {
			lpa.Buys = append(lpa.Buys, buy.LiquidityOrder)
		}
	}

	// Increasing the commitment should always be allowed, but decreasing is
	// only valid if the resulting amount still allows the market as a whole
	// to reach it's commitment level. Otherwise the commitment reduction is
	// rejected.
	if lpa.CommitmentAmount.LT(lp.CommitmentAmount) {
		// first - does the market have enough stake
		supplied := m.getSuppliedStake()
		if m.getTargetStake().GTE(supplied) {
			return common.ErrNotEnoughStake
		}

		// now if the stake surplus is > than the change we are OK
		surplus := supplied.Sub(supplied, m.getTargetStake())
		diff := num.UintZero().Sub(lp.CommitmentAmount, lpa.CommitmentAmount)
		if surplus.LT(diff) {
			return common.ErrNotEnoughStake
		}
	}

	return m.amendLiquidityProvision(ctx, lpa, party)
}

// CancelLiquidityProvision forwards a LiquidityProvisionCancel to the Liquidity Engine.
func (m *Market) CancelLiquidityProvision(ctx context.Context, cancel *types.LiquidityProvisionCancellation, party string) (err error) {
	defer m.onTxProcessed()

	if !m.canSubmitCommitment() {
		return common.ErrCommitmentSubmissionNotAllowed
	}

	if !m.liquidity.IsLiquidityProvider(party) {
		return common.ErrPartyNotLiquidityProvider
	}

	lp := m.liquidity.LiquidityProvisionByPartyID(party)
	if lp == nil {
		return fmt.Errorf("cannot edit liquidity provision from a non liquidity provider party (%v)", party)
	}

	supplied := m.getSuppliedStake()
	if m.getTargetStake().GTE(supplied) {
		return common.ErrNotEnoughStake
	}

	// now if the stake surplus is > than the change we are OK
	surplus := supplied.Sub(supplied, m.getTargetStake())
	if surplus.LT(lp.CommitmentAmount) {
		return common.ErrNotEnoughStake
	}

	defer m.releaseMarginExcess(ctx, party)

	return m.cancelLiquidityProvision(ctx, party, false)
}

func (m *Market) cancelPendingLiquidityProvision(
	ctx context.Context,
	party string,
	initialMargin *num.Uint,
) error {
	// we will just cancel the party,
	// no bond slashing applied
	if err := m.cancelLiquidityProvision(ctx, party, false); err != nil {
		m.log.Debug("error cancelling liquidity provision commitment",
			logging.PartyID(party),
			logging.MarketID(m.GetID()),
			logging.Error(err))
		return err
	}

	return m.rollBackMargin(ctx, party, initialMargin)
}

func (m *Market) createInitialLPOrders(ctx context.Context, newOrders []*types.Order) (err error) {
	if len(newOrders) <= 0 {
		return nil
	}

	party := newOrders[0].Party
	// get the new balance
	marginAcc, _ := m.collateral.GetPartyMarginAccount(
		m.GetID(), party, m.settlementAsset)
	initialMargin := marginAcc.Balance

	submittedIDs := []string{}
	failedOnID := ""
	// submitted order rollback
	defer func() {
		if err == nil || len(newOrders) <= 0 {
			return
		}
		party := newOrders[0].Party
		mappedIDs := map[string]struct{}{} // just a map to access them easily
		// first we cancel all order which we  were able to submit
		for _, v := range submittedIDs {
			mappedIDs[v] = struct{}{}
			_, newerr := m.cancelOrder(ctx, party, v)
			if newerr != nil {
				m.log.Error("unable to rollback order via cancel",
					logging.Error(newerr),
					logging.String("party", party),
					logging.String("order-id", v))
				err = fmt.Errorf("%v, %w", err, newerr)
			}
		}
		// then we release any margin excess
		if rerr := m.rollBackMargin(ctx, party, initialMargin); rerr != nil {
			err = fmt.Errorf("%v, %w", err, rerr)
		}

		// the we just send through the bus all order
		// we were not even able to submit with a rejected event
		for _, v := range newOrders {
			_, ok := mappedIDs[v.ID]
			if !ok && failedOnID != v.ID {
				// this was not handled before, we need to send an
				v.Status = types.OrderStatusRejected
				// set margin check failed, it's the only reason we could
				// not place the order at this point
				v.Reason = types.OrderErrorMarginCheckFailed
				m.broker.Send(events.NewOrderEvent(ctx, v))
			}
		}
	}()

	for _, order := range newOrders {
		// ignoring updated orders as we expect
		// no updates there as the party should ever be able to
		// submit without issues or not at all.
		if order.OriginalPrice == nil {
			order.OriginalPrice = order.Price.Clone()
			order.Price.Mul(order.Price, m.priceFactor)
		}
		if conf, _, err := m.submitOrder(ctx, order); err != nil {
			failedOnID = order.ID
			m.log.Debug("unable to submit liquidity provision order",
				logging.MarketID(m.GetID()),
				logging.OrderID(order.ID),
				logging.PartyID(order.Party),
				logging.Error(err))
			return err
		} else if len(conf.Trades) > 0 {
			m.log.Panic("liquidity provision initial submission should never trade",
				logging.Error(err))
		}
		m.log.Debug("new liquidity order submitted successfully",
			logging.MarketID(m.GetID()),
			logging.OrderID(order.ID),
			logging.PartyID(order.Party))
		submittedIDs = append(submittedIDs, order.ID)
	}

	return nil
}

func (m *Market) rollBackMargin(
	ctx context.Context,
	party string,
	initialMargin *num.Uint,
) error {
	// get the new balance
	marginAcc, err := m.collateral.GetPartyMarginAccount(
		m.GetID(), party, m.settlementAsset)
	if err != nil {
		m.log.Error("could not get margin account",
			logging.PartyID(party),
			logging.MarketID(m.GetID()),
			logging.AssetID(m.settlementAsset),
			logging.Error(err))
		return err
	}

	if marginAcc.Balance.LT(initialMargin) {
		// nothing to rollback
		return nil
	}

	amount := num.UintZero().Sub(marginAcc.Balance, initialMargin)
	// now create the rollback to transfer
	transfer := types.Transfer{
		Owner: party,
		Amount: &types.FinancialAmount{
			Amount: amount,
			Asset:  m.settlementAsset,
		},
		Type:      types.TransferTypeMarginHigh,
		MinAmount: amount.Clone(),
	}

	// then trigger the rollback
	resp, err := m.collateral.RollbackMarginUpdateOnOrder(
		ctx, m.GetID(), m.settlementAsset, &transfer)
	if err != nil {
		m.log.Debug("error rolling back party margin",
			logging.PartyID(party),
			logging.MarketID(m.GetID()),
			logging.Error(err))
		return err
	}

	// then send the event for the transfer request
	m.broker.Send(events.NewLedgerMovements(
		ctx, []*types.LedgerMovement{resp}))
	return nil
}

func (m *Market) repriceLiquidityOrder(side types.Side, reference types.PeggedReference, offset *num.Uint) (*num.Uint, error) {
	if m.as.InAuction() {
		return num.UintZero(), common.ErrCannotRepriceDuringAuction
	}
	var (
		err      error
		refPrice *num.Uint
	)

	switch reference {
	case types.PeggedReferenceMid:
		refPrice, err = m.getStaticMidPrice(side)
	case types.PeggedReferenceBestBid:
		refPrice, err = m.getBestStaticBidPrice()
	case types.PeggedReferenceBestAsk:
		refPrice, err = m.getBestStaticAskPrice()
	}
	if err != nil {
		return num.UintZero(), common.ErrUnableToReprice
	}
	minLpPrice, maxLpPrice, err := m.getValidLPVolumeRange()
	if err != nil {
		return num.UintZero(), err
	}
	return m.adjustPrice(side, refPrice, offset, minLpPrice, maxLpPrice), nil
}

func (m *Market) adjustPrice(side types.Side, referencePrice, offset, minLpPrice, maxLpPrice *num.Uint) *num.Uint {
	offsetPrice := m.applyOffset(side, referencePrice, offset)
	if offsetPrice.GTE(minLpPrice) && offsetPrice.LTE(maxLpPrice) {
		return offsetPrice
	}

	if side == types.SideBuy {
		return minLpPrice
	}
	return maxLpPrice
}

func (m *Market) applyOffset(side types.Side, referencePrice, offset *num.Uint) *num.Uint {
	// scale offset by tick size
	ofst := num.UintZero().Mul(offset, m.priceFactor)
	if side == types.SideSell {
		return num.UintZero().Add(referencePrice, ofst)
	}
	// prevent underflow
	if referencePrice.LTE(ofst) {
		return m.minValidPrice()
	}
	return num.UintZero().Sub(referencePrice, ofst)
}

func (m *Market) computeValidLPVolumeRange(bestStaticBid, bestStaticAsk *num.Uint) (*num.Uint, *num.Uint) {
	mid := bestStaticBid.ToDecimal().Add(bestStaticAsk.ToDecimal()).Div(num.DecimalFromFloat(2))

	lbD := num.DecimalOne().Sub(m.lpPriceRange).Mul(mid)
	ubD := num.DecimalOne().Add(m.lpPriceRange).Mul(mid)

	tick := m.priceFactor.ToDecimal()

	// ceil lower bound
	qL, rL := lbD.QuoRem(tick, int32(0))
	if !rL.IsZero() {
		qL = qL.Add(num.DecimalOne())
	}
	lbD = qL.Mul(tick)

	// floor upper bound
	qU, _ := ubD.QuoRem(tick, int32(0))
	ubD = qU.Mul(tick)

	lb, _ := num.UintFromDecimal(lbD)
	ub, _ := num.UintFromDecimal(ubD)

	// floor at 1 to avoid non-positive value
	if lb.IsNegative() || lb.IsZero() {
		lb = m.minValidPrice()
	}
	if lb.GTE(ub) {
		// if we ended up with overlapping upper and lower bound we set the upper bound to lower bound plus one tick.
		ub = ub.Add(lb, m.priceFactor)
	}

	// we can't have lower bound >= best static ask as then a buy order with that price would trade on entry
	// so place it one tick to the left
	if lb.GTE(bestStaticAsk) {
		lb = num.UintZero().Sub(bestStaticAsk, m.priceFactor)
	}

	// we can't have upper bound <= best static bid as then a sell order with that price would trade on entry
	// so place it one tick to the right
	if ub.LTE(bestStaticBid) {
		ub = num.UintZero().Add(bestStaticBid, m.priceFactor)
	}

	return lb, ub
}

func (m *Market) getValidLPVolumeRange() (*num.Uint, *num.Uint, error) {
	bBid, err := m.getBestStaticBidPrice()
	if err != nil {
		return num.UintOne(), num.MaxUint(), err
	}

	bAsk, err := m.getBestStaticAskPrice()
	if err != nil {
		return num.UintOne(), num.MaxUint(), err
	}
	min, max := m.computeValidLPVolumeRange(bBid, bAsk)
	return min, max, nil
}

func (m *Market) cancelLiquidityProvision(
	ctx context.Context, party string, isDistressed bool,
) error {
	// cancel the liquidity provision
	err := m.liquidity.CancelLiquidityProvision(ctx, party)

	cancelOrders := m.matching.GetLiquidityOrders(party)
	if err != nil {
		m.log.Debug("unable to cancel liquidity provision",
			logging.String("party-id", party),
			logging.String("market-id", m.GetID()),
			logging.Error(err),
		)
		return err
	}

	sort.Slice(cancelOrders, func(i, j int) bool {
		return cancelOrders[i].ID < cancelOrders[j].ID
	})

	// is our party distressed?
	// if yes, the orders have been cancelled by the resolve
	// distressed parties flow.
	if !isDistressed {
		// now we cancel all existing orders
		for _, order := range cancelOrders {
			if _, err := m.cancelOrder(ctx, party, order.ID); err != nil {
				// nothing much we can do here, I suppose
				// something wrong might have happen...
				// does this need a panic? need to think about it...
				m.log.Debug("unable cancel liquidity order",
					logging.String("party", party),
					logging.String("order-id", order.ID),
					logging.Error(err))
			}
		}
	}

	// now we move back the funds from the bond account to the general account
	// of the party
	bondAcc, err := m.collateral.GetOrCreatePartyBondAccount(
		ctx, party, m.GetID(), m.settlementAsset)
	if err != nil {
		m.log.Debug("could not get the party bond account",
			logging.String("party-id", party),
			logging.Error(err))
	}

	// now if our bondAccount is nil
	// it just mean that the party my have gone the distressed path
	// also if the balance is already 0, let's not bother create a
	// transfer request
	if err == nil && !bondAcc.Balance.IsZero() {
		transfer := &types.Transfer{
			Owner: party,
			Amount: &types.FinancialAmount{
				Amount: bondAcc.Balance,
				Asset:  m.settlementAsset,
			},
			Type:      types.TransferTypeBondHigh,
			MinAmount: bondAcc.Balance.Clone(),
		}

		tresp, err := m.collateral.BondUpdate(ctx, m.GetID(), transfer)
		if err != nil {
			m.log.Debug("bond update error", logging.Error(err))
			return err
		}
		m.broker.Send(events.NewLedgerMovements(ctx, []*types.LedgerMovement{tresp}))
		m.collateral.RemoveBondAccount(party, m.GetID(), m.settlementAsset)
	}

	// now let's update the fee selection
	m.updateLiquidityFee(ctx)
	// and remove the party from the equity share like calculation
	m.equityShares.SetPartyStake(party, nil)
	// force update of shares so they are updated for all
	_ = m.equityShares.SharesExcept(m.liquidity.GetInactiveParties())

	m.checkForReferenceMoves(ctx, []*types.Order{}, true)
	return nil
}

func (m *Market) amendLiquidityProvision(
	ctx context.Context, sub *types.LiquidityProvisionAmendment, party string,
) (err error) {
	bondRollback, err := m.ensureLiquidityProvisionBond(ctx, sub, party)
	if err != nil {
		m.log.Debug("could not submit update bond for lp amendment",
			logging.PartyID(party),
			logging.MarketID(m.GetID()),
			logging.Error(err))
		return err
	}

	// in case something goes wrong, we defer rolling back the bond account.
	// in any case from here any here would mean one of these things:
	// - party could not pay the margins
	// - orders could not priced / sized
	defer func() {
		if err != nil {
			tresp, newerr := m.collateral.BondUpdate(
				ctx, m.GetID(), bondRollback)
			if newerr != nil {
				m.log.Debug("unable to rollback bond account topup",
					logging.String("party", party),
					logging.BigUint("amount", bondRollback.Amount.Amount),
					logging.Error(err))
				err = fmt.Errorf("%v: %w", err, newerr)
			}
			if tresp != nil {
				m.broker.Send(events.NewLedgerMovements(
					ctx, []*types.LedgerMovement{tresp}))
			}
		}
	}()

	if m.as.InAuction() {
		return m.amendLiquidityProvisionAuction(ctx, sub, party)
	}
	return m.amendLiquidityProvisionContinuous(ctx, sub, party)
}

// When amending LP during an auction a few different thing can happen
// - first we can get the an indicative uncrossing price, then orders
// will need to use that to be priced, and size
// - second we do not have a indicative uncrossing price, then same thing
// is done with the mark price (if available from previous the state of the
// auction
// - third, none of them are available, which just accept the change, all
// hel may break loose when coming out of auction, but we know this.
func (m *Market) amendLiquidityProvisionAuction(
	ctx context.Context, sub *types.LiquidityProvisionAmendment, party string,
) error {
	price := m.getMarketObservable(num.UintZero())

	// now let's check if we are still at 0, if yes, it means we are in the
	// third condition from before, no price available, we just accept the
	// amendment without deploying any orders, so no need to check any margin etc
	if !price.IsZero() {
		if err := m.calcLiquidityProvisionPotentialMarginsAuction(
			ctx, sub, party, price); err != nil {
			return err
		}
	}

	return m.finalizeLiquidityProvisionAmendmentAuction(ctx, sub, party)
}

// in here we will calculate the liquidity provision potential margin for
// this amendment, this is all happening during auction, so no LP order
// from the party should be in the book, we will just get a list of order
// from the liquidity engine, and try to calculate the potential position
// from there, then move the funds in the party margin account.
func (m *Market) calcLiquidityProvisionPotentialMarginsAuction(
	ctx context.Context,
	sub *types.LiquidityProvisionAmendment,
	party string,
	price *num.Uint,
) error {
	minLpPrice, maxLpPrice := m.minValidPrice(), num.MaxUint()
	repriceFn := func(side types.Side, reference types.PeggedReference, offset *num.Uint) (*num.Uint, error) {
		return m.adjustPrice(side, price.Clone(), offset, minLpPrice, maxLpPrice), nil
	}

	// first lets get the protential shape for this submission
	orders, err := m.liquidity.GetPotentialShapeOrders(
		party, minLpPrice, maxLpPrice, sub, repriceFn)
	if err != nil {
		// any error here means:
		// - the submission was invalid
		// - order(s) in the shapes where not priceable / sizeable
		return err
	}

	// if we have no orders, this might not be an error
	// the commitment can be fulfilled by all the limit orders already
	// submitted by the party into the book
	if len(orders) <= 0 {
		return nil
	}

	// then let's get the margins checked
	// first let's build the position
	pos, ok := m.position.GetPositionByPartyID(party)
	if !ok {
		// this is not an error here, that would just mean the party
		// never had a position open before that, we may be in the auction
		// the party join, and never had the chance to get anything deployed
		// so not positions exists
		pos = positions.NewMarketPosition(party)
	}

	// now we register all these orders as potential positions
	// which we will use to calculate the margin just after
	for _, order := range orders {
		pos.RegisterOrder(m.log, order)
	}

	// then calculate the margins,
	// any shortfall is a blocker here.
	risk, err := m.calcMarginsLiquidityProvisionAmendAuction(ctx, pos, price.Clone())
	if err != nil {
		return err
	}

	// so far all is ok, just one last step, if a risk event
	// was returned let's move the funds
	if risk != nil {
		return m.transferMarginsLiquidityProvisionAmendAuction(ctx, risk)
	}

	// nothing left to do
	return nil
}

func (m *Market) finalizeLiquidityProvisionAmendmentAuction(
	ctx context.Context, sub *types.LiquidityProvisionAmendment, party string,
) error {
	// first parameter is the update to the orders, but we know that during
	// auction no orders shall be return, so let's just look at the error
	_, err := m.liquidity.AmendLiquidityProvision(ctx, sub, party, m.idgen)
	if err != nil {
		m.log.Panic("error while amending liquidity provision, this should not happen at this point, the LP was validated earlier",
			logging.Error(err))
	}

	m.updateMarketValueProxy()
	// now we can update the liquidity fee to be taken
	m.updateLiquidityFee(ctx)
	// now we can setup our party stake to calculate equities
	m.equityShares.SetPartyStake(party, sub.CommitmentAmount.Clone())
	// force update of shares so they are updated for all
	_ = m.equityShares.SharesExcept(m.liquidity.GetInactiveParties())

	return nil
}

func (m *Market) amendLiquidityProvisionContinuous(
	ctx context.Context, sub *types.LiquidityProvisionAmendment, party string,
) error {
	minLpPrice, maxLpPrice, err := m.getValidLPVolumeRange()
	if err != nil {
		m.log.Debug("could not get valid lp range to call liquidity",
			logging.String("market-id", m.GetID()),
			logging.String("party", party),
			logging.Error(err),
		)
		return err
	}

	// first lets get the protential shape for this submission
	orders, err := m.liquidity.GetPotentialShapeOrders(
		party, minLpPrice, maxLpPrice, sub, m.repriceLiquidityOrder)
	if err != nil {
		// any error here means:
		// - the submission was invalid
		// - order(s) in the shapes where not priceable / sizeable
		return err
	}

	pos, ok := m.position.GetPositionByPartyID(party)
	if !ok {
		// this is not an error here, that would just mean the party
		// never had a position open before that, we may be in the auction
		// the party join, and never had the chance to get anything deployed
		// so not positions exists
		pos = positions.NewMarketPosition(party)
	}

	// first remove all existing orders from the potential positions
	lorders := m.matching.GetLiquidityOrders(party)
	for _, v := range lorders {
		// ensure the order is on the actual potential position first
		if order, foundOnBook, _ := m.getOrderByID(v.ID); foundOnBook {
			pos.UnregisterOrder(m.log, order)
		}
	}

	// then add all the newly created ones
	for _, v := range orders {
		pos.RegisterOrder(m.log, v)
	}

	// now we calculate the margin as if we were submitting these orders
	// any error here means we cannot amend,
	err = m.calcMarginsLiquidityProvisionAmendContinuous(ctx, pos)
	if err != nil {
		return err
	}

	// then we do not actually move the monies in this case
	// this will be done naturally when finalizing the amendment

	return m.finalizeLiquidityProvisionAmendmentContinuous(ctx, sub, party)
}

func (m *Market) finalizeLiquidityProvisionAmendmentContinuous(
	ctx context.Context, sub *types.LiquidityProvisionAmendment, party string,
) error {
	cancels, err := m.liquidity.AmendLiquidityProvision(ctx, sub, party, m.idgen)
	if err != nil {
		m.log.Panic("error while amending liquidity provision, this should not happen at this point, the LP was validated earlier",
			logging.Error(err))
	}

	for _, order := range cancels {
		if _, err := m.cancelOrder(ctx, party, order.ID); err != nil {
			// nothing much we can do here, I suppose
			// something wrong might have happen...
			// does this need a panic? need to think about it...
			m.log.Debug("unable cancel liquidity order",
				logging.String("party", party),
				logging.String("order-id", order.ID),
				logging.Error(err))
		}
	}

	defer func() {
		m.updateMarketValueProxy()
		// now we can update the liquidity fee to be taken
		m.updateLiquidityFee(ctx)
		// now we can setup our party stake to calculate equities
		m.equityShares.SetPartyStake(party, sub.CommitmentAmount)
		// force update of shares so they are updated for all
		_ = m.equityShares.SharesExcept(m.liquidity.GetInactiveParties())
	}()

	// this workd but we definitely trigger some recursive loop which
	// are unlikely to be fine.
	m.checkForReferenceMoves(ctx, []*types.Order{}, true)

	return nil
}

// returns the rollback transfer in case of error.
func (m *Market) ensureLiquidityProvisionBond(
	ctx context.Context, sub *types.LiquidityProvisionAmendment, party string,
) (*types.Transfer, error) {
	bondAcc, err := m.collateral.GetOrCreatePartyBondAccount(
		ctx, party, m.GetID(), m.settlementAsset)
	if err != nil {
		return nil, err
	}

	// first check if there's enough funds in the gen + bond
	// account to cover the new commitment
	if !m.collateral.CanCoverBond(m.GetID(), party, m.settlementAsset, sub.CommitmentAmount.Clone()) {
		return nil, common.ErrCommitmentSubmissionNotAllowed
	}

	// build our transfer to be sent to collateral
	amount, neg := num.UintZero().Delta(sub.CommitmentAmount, bondAcc.Balance)
	ty := types.TransferTypeBondLow
	if neg {
		ty = types.TransferTypeBondHigh
	}
	transfer := &types.Transfer{
		Owner: party,
		Amount: &types.FinancialAmount{
			Amount: amount,
			Asset:  m.settlementAsset,
		},
		Type:      ty,
		MinAmount: amount.Clone(),
	}

	// move our bond
	tresp, err := m.collateral.BondUpdate(ctx, m.GetID(), transfer)
	if err != nil {
		return nil, err
	}
	m.broker.Send(events.NewLedgerMovements(
		ctx, []*types.LedgerMovement{tresp}))

	// now we will use the actuall transfer as a rollback later on eventually
	// so let's just change from HIGH to LOW and inverse
	if transfer.Type == types.TransferTypeBondHigh {
		transfer.Type = types.TransferTypeBondLow
	} else {
		transfer.Type = types.TransferTypeBondHigh
	}

	return transfer, nil
}

func (m *Market) ensureLPCommitmentAmount(amount *num.Uint) error {
	quantum, err := m.collateral.GetAssetQuantum(m.settlementAsset)
	if err != nil {
		m.log.Panic("could not get quantum for asset, this should never happen",
			logging.AssetID(m.settlementAsset),
			logging.Error(err),
		)
	}
	minStake := quantum.Mul(m.minLPStakeQuantumMultiple)
	if amount.ToDecimal().LessThan(minStake) {
		return ErrCommitmentAmountTooLow
	}

	return nil
}

func (m *Market) updateLiquidityScores() {
	minLpPrice, maxLpPrice, err := m.getValidLPVolumeRange()
	if err != nil {
		m.log.Debug("liquidity score update error", logging.Error(err))
		return
	}
	bid, ask, err := m.getBestStaticPricesDecimal()
	if err != nil {
		m.log.Debug("liquidity score update error", logging.Error(err))
		return
	}

	m.liquidity.UpdateAverageLiquidityScores(bid, ask, minLpPrice, maxLpPrice)
}

func (m *Market) updateSharesWithLiquidityScores(shares map[string]num.Decimal) map[string]num.Decimal {
	lScores := m.liquidity.GetAverageLiquidityScores()

	total := num.DecimalZero()
	for k, v := range shares {
		l, ok := lScores[k]
		if !ok {
			continue
		}
		adjusted := v.Mul(l)
		shares[k] = adjusted

		total = total.Add(adjusted)
	}

	// normalise
	if !total.IsZero() {
		for k, v := range shares {
			shares[k] = v.Div(total)
		}
	}

	// reset for next period
	m.liquidity.ResetAverageLiquidityScores()

	return shares
}
