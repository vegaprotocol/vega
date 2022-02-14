package execution

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"code.vegaprotocol.io/vega/idgeneration"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/liquidity"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/positions"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
)

var ErrCommitmentAmountTooLow = errors.New("commitment amount is too low")

// SubmitLiquidityProvision forwards a LiquidityProvisionSubmission to the Liquidity Engine.
func (m *Market) SubmitLiquidityProvision(
	ctx context.Context,
	sub *types.LiquidityProvisionSubmission,
	party, deterministicId string,
) (err error,
) {
	m.idgen = idgeneration.New(deterministicId)
	defer func() { m.idgen = nil }()

	if !m.canSubmitCommitment() {
		return ErrCommitmentSubmissionNotAllowed
	}

	var (
		// this is use to specified that the lp may need to be cancelled
		needsCancel bool
		// his specifies that the changes on the bond account have to be
		// rolled back
		needsBondRollback bool
	)

	if err := m.liquidity.ValidateLiquidityProvisionSubmission(sub, true); err != nil {
		return err
	}

	if err := m.ensureLPCommitmentAmount(sub.CommitmentAmount); err != nil {
		return err
	}

	// if the party is alrready an LP we reject the new submission
	if m.liquidity.IsLiquidityProvider(party) {
		return ErrPartyAlreadyLiquidityProvider
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
				logging.String("id", deterministicId),
				logging.Error(newerr))
			err = fmt.Errorf("%v, %w", err, newerr)
		}
	}()

	// we will need both bond account and the margin account, let's create
	// them now
	asset, _ := m.mkt.GetAsset()
	bondAcc, err := m.collateral.GetOrCreatePartyBondAccount(ctx, party, m.GetID(), asset)
	if err != nil {
		// error happen, we can't even have the bond account taken
		// if this is not an amendment, we cancel the liquidity provision
		needsCancel = true
		return err
	}
	_, err = m.collateral.CreatePartyMarginAccount(ctx, party, m.GetID(), asset)
	if err != nil {
		needsCancel = true
		return err
	}

	// now we calculate the amount that needs to be moved into the account

	amount, neg := num.Zero().Delta(sub.CommitmentAmount, bondAcc.Balance)
	ty := types.TransferTypeBondLow
	if neg {
		ty = types.TransferTypeBondHigh
	}
	transfer := &types.Transfer{
		Owner: party,
		Amount: &types.FinancialAmount{
			Amount: amount.Clone(), // clone here, we're using amount again in case of rollback
			Asset:  asset,
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
	m.broker.Send(events.NewTransferResponse(ctx, []*types.TransferResponse{tresp}))

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
		m.broker.Send(events.NewTransferResponse(ctx, []*types.TransferResponse{tresp}))
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
			_ = m.equityShares.Shares(m.liquidity.GetInactiveParties())

			m.checkLiquidity(ctx, nil)
			m.commandLiquidityAuction(ctx)
		}
	}()

	existingOrders := m.matching.GetOrdersPerParty(party)
	bestBidPrice, bestAskPrice, err := m.getBestStaticPrices()
	if err != nil {
		m.log.Debug("could not get mid prices to call liquidity",
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
	newOrders, err := m.liquidity.CreateInitialOrders(ctx, bestBidPrice.Clone(), bestAskPrice.Clone(), party, existingOrders, m.repriceLiquidityOrder)
	if err != nil {
		m.log.Debug("orders from liquidity provisions could not be generated by the liquidity engine",
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
func (m *Market) AmendLiquidityProvision(ctx context.Context, lpa *types.LiquidityProvisionAmendment, party string, deterministicId string) (err error) {
	m.idgen = idgeneration.New(deterministicId)
	defer func() { m.idgen = nil }()

	if !m.canSubmitCommitment() {
		return ErrCommitmentSubmissionNotAllowed
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
		return ErrPartyNotLiquidityProvider
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
			return ErrNotEnoughStake
		}

		// now if the stake surplus is > than the change we are OK
		surplus := supplied.Sub(supplied, m.getTargetStake())
		diff := num.Zero().Sub(lp.CommitmentAmount, lpa.CommitmentAmount)
		if surplus.LT(diff) {
			return ErrNotEnoughStake
		}
	}

	return m.amendLiquidityProvision(ctx, lpa, party)
}

// CancelLiquidityProvision forwards a LiquidityProvisionCancel to the Liquidity Engine.
func (m *Market) CancelLiquidityProvision(ctx context.Context, cancel *types.LiquidityProvisionCancellation, party string) (err error) {
	if !m.liquidity.IsLiquidityProvider(party) {
		return ErrPartyNotLiquidityProvider
	}

	lp := m.liquidity.LiquidityProvisionByPartyID(party)
	if lp == nil {
		return fmt.Errorf("cannot edit liquidity provision from a non liquidity provider party (%v)", party)
	}

	supplied := m.getSuppliedStake()
	if m.getTargetStake().GTE(supplied) {
		return ErrNotEnoughStake
	}

	// now if the stake surplus is > than the change we are OK
	surplus := supplied.Sub(supplied, m.getTargetStake())
	if surplus.LT(lp.CommitmentAmount) {
		return ErrNotEnoughStake
	}

	return m.cancelLiquidityProvision(ctx, party, false)
}

// this is a function to be called when orders already exists
// submitted by the liquidity provider.
// We will first update orders, which basically will trigger cancellation
// then place the new orders.
// this is done this way just so we maximise the changes for the margin
// calls to succeed.
func (m *Market) updateAndCreateLPOrders(
	ctx context.Context,
	newOrders []*types.Order,
	cancels []*liquidity.ToCancel,
	distressed []*types.Order,
) ([]*types.Order, error) {
	market := m.GetID()

	for _, cancel := range cancels {
		for _, orderID := range cancel.OrderIDs {
			if _, err := m.cancelOrder(ctx, cancel.Party, orderID); err != nil {
				// here we panic, an order which should be in a the market
				// appears not to be. there's either an issue in the liquidity
				// engine and we are trying to remove a non-existing order
				// or the market lost track of the order
				m.log.Debug("unable to amend a liquidity order",
					logging.OrderID(orderID),
					logging.PartyID(cancel.Party),
					logging.MarketID(market),
					logging.Error(err))
			}
		}
	}

	// this is set of all liquidity provider which
	// at after trying to cancel and replace their orders
	// cannot fullfil their margins anymore.
	faultyLPs := map[string]bool{}
	faultyLPOrders := map[string]*types.Order{}
	initialMargins := map[string]*num.Uint{}
	var orderUpdates []*types.Order

	// first add all party which are already distressed here
	for _, v := range distressed {
		faultyLPOrders[v.Party] = v
		faultyLPs[v.Party] = true
	}

	mktID := m.GetID()
	asset, _ := m.mkt.GetAsset()
	var enteredAuction bool
	for _, order := range newOrders {
		// before we submit orders, we check if the party was pending
		// and save the amount of the margin balance.
		// so we can roll back to this state later on
		if m.liquidity.IsPending(order.Party) {
			if _, ok := initialMargins[order.Party]; !ok {
				marginAcc, _ := m.collateral.GetPartyMarginAccount(
					mktID, order.Party, asset)
				initialMargins[order.Party] = marginAcc.Balance // no need to clone
			}
		}

		if faulty, ok := faultyLPs[order.Party]; ok && faulty {
			// we already tried to submit an lp order which failed
			// for this party. we'll cancel them just in a bit
			// be patient...
			continue
		}
		if order.OriginalPrice == nil {
			order.OriginalPrice = order.Price.Clone()
			order.Price.Mul(order.Price, m.priceFactor)
		}
		conf, orderUpdts, err := m.submitOrder(ctx, order)
		if err != nil {
			m.log.Debug("could not submit liquidity provision order, scheduling for closeout",
				logging.OrderID(order.ID),
				logging.PartyID(order.Party),
				logging.MarketID(order.MarketID),
				logging.Error(err))
			// set the party as faulty
			faultyLPs[order.Party] = true
			faultyLPOrders[order.Party] = order
			continue
		}
		if len(conf.Trades) > 0 {
			m.log.Panic("submitting liquidity orders after a reprice should never trade",
				logging.Order(*order))
		}

		// did we enter auction
		if m.as.InAuction() {
			enteredAuction = true
			break
		}

		orderUpdates = append(orderUpdates, orderUpdts...)
		faultyLPs[order.Party] = false
	}

	// now get all non faulty parties, and get them not pending
	// if they were
	parties := make([]struct {
		Party  string
		Faulty bool
	}, 0, len(faultyLPs))
	for k, v := range faultyLPs {
		parties = append(parties, struct {
			Party  string
			Faulty bool
		}{k, v})
	}

	// now just sort them to deterministically send them
	sort.Slice(parties, func(i, j int) bool {
		return parties[i].Party < parties[j].Party
	})

	var updateShares bool
	for _, v := range parties {
		if !v.Faulty {
			// update shares to add this party to the shares
			updateShares = true
			m.liquidity.RemovePending(v.Party)
			continue
		}

		// now if the party was pending, which means the
		// order was never submitted, which also means that the
		// margin were never calculated on submission
		if m.liquidity.IsPending(v.Party) {
			_ = m.cancelPendingLiquidityProvision(
				ctx, v.Party, initialMargins[v.Party])
			continue
		}

		// now the party had not enough enough funds to pay the margin
		orders, err := m.cancelDistressedLiquidityProvision(
			ctx, v.Party, faultyLPOrders[v.Party])
		if err != nil {
			m.log.Debug("issue cancelling liquidity provision",
				logging.Error(err),
				logging.MarketID(m.GetID()),
				logging.PartyID(v.Party))
		}
		orderUpdates = append(orderUpdates, orders...)

		// update shares to remove this party from the shares
		updateShares = true
	}

	if updateShares {
		// force update of shares so they are updated for all
		_ = m.equityShares.Shares(m.liquidity.GetInactiveParties())
	}

	// if we are in an option, there's nothing to be done with these
	// updates specifically, let's just return
	if enteredAuction {
		orderUpdates = nil
	}

	return orderUpdates, nil
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

func (m *Market) cancelDistressedLiquidityProvision(
	ctx context.Context,
	party string,
	order *types.Order,
) ([]*types.Order, error) {
	mktID := m.GetID()
	asset, _ := m.mkt.GetAsset()

	mpos, ok := m.position.GetPositionByPartyID(party)
	if !ok {
		m.log.Debug("error getting party position",
			logging.PartyID(party),
			logging.MarketID(mktID))
		return nil, nil
	}

	margin, perr := m.collateral.GetPartyMargin(mpos, asset, mktID)
	if perr != nil {
		m.log.Debug("error getting party margin",
			logging.PartyID(party),
			logging.MarketID(mktID),
			logging.Error(perr))
		return nil, perr
	}
	orderUpdates, err := m.resolveClosedOutParties(
		ctx, []events.Margin{margin}, order)
	if err != nil {
		m.log.Error("could not resolve out parties",
			logging.MarketID(mktID),
			logging.PartyID(party),
			logging.Error(err))
		return nil, err
	}

	return orderUpdates, nil
}

func (m *Market) createInitialLPOrders(ctx context.Context, newOrders []*types.Order) (err error) {
	if len(newOrders) <= 0 {
		return nil
	}

	asset, _ := m.mkt.GetAsset()
	party := newOrders[0].Party
	// get the new balance
	marginAcc, _ := m.collateral.GetPartyMarginAccount(
		m.GetID(), party, asset)
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
	asset, _ := m.mkt.GetAsset()
	// get the new balance
	marginAcc, err := m.collateral.GetPartyMarginAccount(
		m.GetID(), party, asset)
	if err != nil {
		m.log.Error("could not get margin account",
			logging.PartyID(party),
			logging.MarketID(m.GetID()),
			logging.AssetID(asset),
			logging.Error(err))
		return err
	}

	if marginAcc.Balance.LT(initialMargin) {
		// nothing to rollback
		return nil
	}

	amount := num.Zero().Sub(marginAcc.Balance, initialMargin)
	// now create the rollback to transfer
	transfer := types.Transfer{
		Owner: party,
		Amount: &types.FinancialAmount{
			Amount: amount,
			Asset:  asset,
		},
		Type:      types.TransferTypeMarginHigh,
		MinAmount: amount.Clone(),
	}

	// then trigger the rollback
	resp, err := m.collateral.RollbackMarginUpdateOnOrder(
		ctx, m.GetID(), asset, &transfer)
	if err != nil {
		m.log.Debug("error rolling back party margin",
			logging.PartyID(party),
			logging.MarketID(m.GetID()),
			logging.Error(err))
		return err
	}

	// then send the event for the transfer request
	m.broker.Send(events.NewTransferResponse(
		ctx, []*types.TransferResponse{resp}))
	return nil
}

// repriceFuncW is an adapter for getNewPeggedPrice.
func (m *Market) repriceLiquidityOrder(
	po *types.PeggedOrder, side types.Side) (*num.Uint, *types.PeggedOrder, error) {
	if m.as.InAuction() {
		return num.Zero(), nil, ErrCannotRepriceDuringAuction
	}

	var (
		err   error
		price *num.Uint
	)

	switch po.Reference {
	case types.PeggedReferenceMid:
		price, err = m.getStaticMidPrice(side)
	case types.PeggedReferenceBestBid:
		price, err = m.getBestStaticBidPrice()
	case types.PeggedReferenceBestAsk:
		price, err = m.getBestStaticAskPrice()
	}
	if err != nil {
		return num.Zero(), nil, ErrUnableToReprice
	}
	return m.adjustPriceRange(po, side, price)
}

func (m *Market) adjustPriceRange(po *types.PeggedOrder, side types.Side, price *num.Uint) (p *num.Uint, _ *types.PeggedOrder, _ error) {
	// now from here, we will be adjusting the offset
	// to ensure the price is always in the range [minPrice, maxPrice]
	// from the price monitoring engine.

	// we get the minPrice and maxPrice from the price
	// monitoring here. We will always want our price for
	// liquidity orders to be in the range [minPrice, maxPrice].
	// if one price was to be out of this range, then peg offset
	// and order price should be bounded to min/max price.
	minPrice, maxPrice := m.pMonitor.GetValidPriceRange()
	// minPrice can't be negative anymore
	minP := minPrice.Representation()
	maxP := maxPrice.Representation()
	// now we have to ensure that the min price is ceil'ed, and max price is floored
	// if the market decimal places != asset decimals (indicated by priceFactor == 1)
	if m.priceFactor.NEQ(m.one) {
		// if min == 0, don't add 1
		if !minP.IsZero() {
			minP.Div(minP, m.priceFactor)
			minP.Add(minP, num.NewUint(1)) // ceil
			minP.Mul(minP, m.priceFactor)
		}
		// floor max price: divide and multiply back
		maxP.Div(maxP, m.priceFactor)
		maxP.Mul(maxP, m.priceFactor)
	}

	// this is handling bestAsk / mid for ASK.
	if side == types.SideSell {
		// that's our initial price with our offset
		basePrice := num.Sum(price, po.Offset)
		// now if this price+offset is < to maxPrice,
		// nothing needs to be changed. we return
		// both the current price, and the offset
		if basePrice.LTE(maxP) {
			// now we also need to make sure we are > minPrice
			if basePrice.GTE(minP) {
				return basePrice, po, nil
			}

			// now we are in the case where the price we did
			// calculate was < minPrice, we now need
			// to place an offset which gets us at least to
			// bestAsk/Mid
			switch po.Reference {
			case types.PeggedReferenceBestAsk:
				po.Offset = num.Zero()
			case types.PeggedReferenceMid:
				po.Offset.SetUint64(1)
				if m.as.InAuction() {
					po.Offset = num.Zero()
				}
			}
			// ensure the offset takes into account the decimal places
			offset := num.Zero().Mul(po.Offset, m.priceFactor)
			return num.Sum(price, offset), po, nil
		}

		// now our basePrice is outside range.
		// we have two posibilitied now, maxPrice is
		// bigger than the price we got, then we use it
		// or we will use price if it's higher.
		if price.LT(maxP) {
			// this is the case where maxPrice is > to price,
			// then we need to adapt the offset
			po.Offset = num.Zero().Sub(maxP, price)
			// and our price is the maxPrice
			return maxP, po, nil
		}

		// then this is the last case, were maxPrice would be smaller
		// than our price.
		// then we're going to set our price to the calculated price,
		// and the offset to 0 or 1 dependingof the reference.
		switch po.Reference {
		case types.PeggedReferenceBestAsk:
			po.Offset = num.Zero()
		case types.PeggedReferenceMid:
			po.Offset.SetUint64(1)
			if m.as.InAuction() {
				po.Offset = num.Zero()
			}
		}
		offset := num.Zero().Mul(po.Offset, m.priceFactor)
		return num.Sum(price, offset), po, nil
	}

	// This is handling bestBid / mid for BID
	// first the case where we are sure to be able to price
	if price.GT(po.Offset) {
		basePrice := num.Zero().Sub(price, po.Offset)

		// this is the case where our price is correct
		// at this point our basePrice should not be 0
		// and this would cover anycase where minPrice
		// would be 0, it's safe to return this offset
		// minPrice <= basePrice <= price
		if basePrice.GTE(minP) {
			if basePrice.LTE(maxP) {
				return basePrice, po, nil
			}

			// now we are in the case where the price we did
			// calculate was > maxPrice too, we now need
			// to place an offset which gets us at max to
			// at bestBid/Mid
			switch po.Reference {
			case types.PeggedReferenceBestBid:
				po.Offset = num.Zero()
			case types.PeggedReferenceMid:
				po.Offset.SetUint64(1)
				if m.as.InAuction() {
					po.Offset = num.Zero()
				}
			}
			offset := num.Zero().Mul(po.Offset, m.priceFactor)
			return price.Sub(price, offset), po, nil
		}

		// now this is the case where basePrice is < minPrice
		// and minPrice is non-negative + inferior to bestBid
		if !minP.IsZero() && minP.LT(price) {
			po.Offset = po.Offset.Sub(price, minP)

			return price.Sub(price, po.Offset), po, nil
		}

		// now we are going to handle the case where
		// basePrice < price > minPrice
		// in that case we will just assign the offset
		// to the price
		// we also know the price here cannot be 0
		// so it's safe to have a 1 offset
		switch po.Reference {
		case types.PeggedReferenceBestBid:
			po.Offset = num.Zero()
		case types.PeggedReferenceMid:
			po.Offset.SetUint64(1)
			if m.as.InAuction() {
				po.Offset = num.Zero()
			}
		}
		offset := num.Zero().Mul(po.Offset, m.priceFactor)
		return price.Sub(price, offset), po, nil
	}

	// now at this point we know that price - offset
	// would be negative, so we need to handle 2 cases
	// either minPrice is a non-0 price after offset
	// and it's smaller that price, or we will use price
	if minP.IsZero() || minP.GT(price) {
		// here we use the price as both case are invalid
		// for using minPrice
		switch po.Reference {
		case types.PeggedReferenceBestBid:
			po.Offset = num.Zero()
		case types.PeggedReferenceMid:
			po.Offset.SetUint64(1)
		}
		offset := num.Zero().Mul(po.Offset, m.priceFactor)
		return price.Sub(price, offset), po, nil
	}

	// this is the last case where we can use the minPrice
	off := num.Zero().Sub(price, minP)
	po.Offset = off.Clone()
	return price.Sub(price, off), po, nil
}

func (m *Market) cancelLiquidityProvision(
	ctx context.Context, party string, isDistressed bool) error {
	// cancel the liquidity provision
	cancelOrders, err := m.liquidity.CancelLiquidityProvision(ctx, party)
	if err != nil {
		m.log.Debug("unable to cancel liquidity provision",
			logging.String("party-id", party),
			logging.String("market-id", m.GetID()),
			logging.Error(err),
		)
		return err
	}

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
	asset, _ := m.mkt.GetAsset()
	bondAcc, err := m.collateral.GetOrCreatePartyBondAccount(
		ctx, party, m.GetID(), asset)
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
				Asset:  asset,
			},
			Type:      types.TransferTypeBondHigh,
			MinAmount: bondAcc.Balance.Clone(),
		}

		tresp, err := m.collateral.BondUpdate(ctx, m.GetID(), transfer)
		if err != nil {
			m.log.Debug("bond update error", logging.Error(err))
			return err
		}
		m.broker.Send(events.NewTransferResponse(ctx, []*types.TransferResponse{tresp}))
	}

	// now let's update the fee selection
	m.updateLiquidityFee(ctx)
	// and remove the party from the equity share like calculation
	m.equityShares.SetPartyStake(party, num.Zero())
	// force update of shares so they are updated for all
	_ = m.equityShares.Shares(m.liquidity.GetInactiveParties())

	m.checkForReferenceMoves(ctx, []*types.Order{}, true)
	m.checkLiquidity(ctx, nil)
	m.commandLiquidityAuction(ctx)

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
				m.broker.Send(events.NewTransferResponse(
					ctx, []*types.TransferResponse{tresp}))
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
	// first try to get the indicative uncrossing price from the book
	price := m.matching.GetIndicativePrice()
	if price.IsZero() {
		// here it is 0 so we will use the mark price
		price = m.getCurrentMarkPrice()
	}

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
	repriceFn := func(o *types.PeggedOrder, side types.Side) (*num.Uint, *types.PeggedOrder, error) {
		return m.adjustPriceRange(o, side, price.Clone())
	}

	// first lets get the protential shape for this submission
	orders, err := m.liquidity.GetPotentialShapeOrders(
		party, price, price.Clone(), sub, repriceFn)
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
		pos.RegisterOrder(order)
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
	_ = m.equityShares.Shares(m.liquidity.GetInactiveParties())

	m.checkLiquidity(ctx, nil)
	m.commandLiquidityAuction(ctx)

	return nil
}

func (m *Market) amendLiquidityProvisionContinuous(
	ctx context.Context, sub *types.LiquidityProvisionAmendment, party string,
) error {
	bestBidPrice, bestAskPrice, err := m.getBestStaticPrices()
	if err != nil {
		m.log.Debug("could not get mid prices to call liquidity",
			logging.String("market-id", m.GetID()),
			logging.String("party", party),
			logging.Error(err),
		)
		return err
	}

	// first lets get the protential shape for this submission
	orders, err := m.liquidity.GetPotentialShapeOrders(
		party, bestBidPrice.Clone(), bestAskPrice.Clone(), sub, m.repriceLiquidityOrder)
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
	lorders := m.liquidity.GetLiquidityOrders(party)
	for _, v := range lorders {
		// ensure the order is on the actual potential position first
		if order, foundOnBook, _ := m.getOrderByID(v.ID); foundOnBook {
			pos.UnregisterOrder(m.log, order)
		}
	}

	// then add all the newly created ones
	for _, v := range orders {
		pos.RegisterOrder(v)
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
	// first parameter is the update to the orders, but we know that during
	// auction no orders shall be return, so let's just look at the error
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
		_ = m.equityShares.Shares(m.liquidity.GetInactiveParties())
	}()

	// this workd but we definitely trigger some recursive loop which
	// are unlikely to be fine.
	m.checkForReferenceMoves(ctx, []*types.Order{}, true)
	m.checkLiquidity(ctx, nil)
	m.commandLiquidityAuction(ctx)

	return nil
}

// returns the rollback transfer in case of error.
func (m *Market) ensureLiquidityProvisionBond(
	ctx context.Context, sub *types.LiquidityProvisionAmendment, party string,
) (*types.Transfer, error) {
	asset, _ := m.mkt.GetAsset()
	bondAcc, err := m.collateral.GetOrCreatePartyBondAccount(
		ctx, party, m.GetID(), asset)
	if err != nil {
		return nil, err
	}

	// first check if there's enough funds in the gen + bond
	// account to cover the new commitment
	if !m.collateral.CanCoverBond(m.GetID(), party, asset, sub.CommitmentAmount.Clone()) {
		return nil, ErrCommitmentSubmissionNotAllowed
	}

	// build our transfer to be sent to collateral
	amount, neg := num.Zero().Delta(sub.CommitmentAmount, bondAcc.Balance)
	ty := types.TransferTypeBondLow
	if neg {
		ty = types.TransferTypeBondHigh
	}
	transfer := &types.Transfer{
		Owner: party,
		Amount: &types.FinancialAmount{
			Amount: amount,
			Asset:  asset,
		},
		Type:      ty,
		MinAmount: amount.Clone(),
	}

	// move our bond
	tresp, err := m.collateral.BondUpdate(ctx, m.GetID(), transfer)
	if err != nil {
		return nil, err
	}
	m.broker.Send(events.NewTransferResponse(
		ctx, []*types.TransferResponse{tresp}))

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
	asset, _ := m.mkt.GetAsset()
	quantum, err := m.collateral.GetAssetQuantum(asset)
	if err != nil {
		m.log.Panic("could not get quantum for asset, this should never happen",
			logging.AssetID(asset),
			logging.Error(err),
		)
	}
	minStake := quantum.ToDecimal().Mul(m.minLPStakeQuantumMultiple)
	if amount.ToDecimal().LessThan(minStake) {
		return ErrCommitmentAmountTooLow
	}

	return nil
}
