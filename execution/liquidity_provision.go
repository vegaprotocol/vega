package execution

import (
	"context"
	"fmt"
	"sort"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/liquidity"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/positions"
	types "code.vegaprotocol.io/vega/proto"
	commandspb "code.vegaprotocol.io/vega/proto/commands/v1"
)

// SubmitLiquidityProvision forwards a LiquidityProvisionSubmission to the Liquidity Engine.
func (m *Market) SubmitLiquidityProvision(ctx context.Context, sub *commandspb.LiquidityProvisionSubmission, party, id string) (err error) {
	defer func() {
		if err != nil {
			m.broker.Send(events.NewTxErrEvent(ctx, err, party, sub))
		}
	}()
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

	// if the party is amending an existing LP
	// we go done the path of amending
	if m.liquidity.IsLiquidityProvider(party) {
		return m.amendOrCancelLiquidityProvision(ctx, sub, party, id)
	}

	if err := m.liquidity.SubmitLiquidityProvision(ctx, sub, party, id); err != nil {
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
				logging.String("id", id),
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
	amount := int64(sub.CommitmentAmount - bondAcc.Balance)
	ty := types.TransferType_TRANSFER_TYPE_BOND_LOW
	if amount < 0 {
		ty = types.TransferType_TRANSFER_TYPE_BOND_HIGH
		amount = -amount
	}
	transfer := &types.Transfer{
		Owner: party,
		Amount: &types.FinancialAmount{
			Amount: uint64(amount),
			Asset:  asset,
		},
		Type:      ty,
		MinAmount: uint64(amount),
	}

	tresp, err := m.collateral.BondUpdate(ctx, m.GetID(), party, transfer)
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
		if transfer.Type == types.TransferType_TRANSFER_TYPE_BOND_HIGH {
			transfer.Type = types.TransferType_TRANSFER_TYPE_BOND_LOW
		} else {
			transfer.Type = types.TransferType_TRANSFER_TYPE_BOND_HIGH
		}

		tresp, newerr := m.collateral.BondUpdate(ctx, m.GetID(), party, transfer)
		if newerr != nil {
			m.log.Debug("unable to rollback bon account topup",
				logging.String("party", party),
				logging.Int64("amount", amount),
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
			m.equityShares.SetPartyStake(party, sub.CommitmentAmount)
			// force update of shares so they are updated for all
			_ = m.equityShares.Shares(m.liquidity.GetInactiveParties())

			m.checkLiquidity(ctx, nil)
			m.commandLiquidityAuction(ctx)

		}
	}()

	existingOrders := m.matching.GetOrdersPerParty(party)
	midPriceBid, midPriceAsk, err := m.getStaticMidPrices()
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
	newOrders, err := m.liquidity.CreateInitialOrders(ctx, midPriceBid, midPriceAsk, party, existingOrders, m.repriceFuncW)
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
) error {

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
	initialMargins := map[string]uint64{}

	mktID := m.GetID()
	asset, _ := m.mkt.GetAsset()
	for _, order := range newOrders {
		// before we submit orders, we check if the party was pending
		// and save the amount of the margin balance.
		// so we can roll back to this state later on
		if m.liquidity.IsPending(order.PartyId) {
			if _, ok := initialMargins[order.PartyId]; !ok {
				marginAcc, _ := m.collateral.GetPartyMarginAccount(
					mktID, order.PartyId, asset)
				initialMargins[order.PartyId] = marginAcc.Balance
			}
		}

		if faulty, ok := faultyLPs[order.PartyId]; ok && faulty {
			// we already tried to submit an lp order which failed
			// for this party. we'll cancel them just in a bit
			// be patient...
			continue
		}
		if _, err := m.submitOrder(ctx, order, false); err != nil {
			m.log.Debug("could not submit liquidity provision order, scheduling for closeout",
				logging.OrderID(order.Id),
				logging.PartyID(order.PartyId),
				logging.MarketID(order.MarketId),
				logging.Error(err))
			// set the party as faulty
			faultyLPs[order.PartyId] = true
			faultyLPOrders[order.PartyId] = order
			continue
		}
		faultyLPs[order.PartyId] = false
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
		err := m.cancelDistressedLiquidityProvision(
			ctx, v.Party, faultyLPOrders[v.Party])
		if err != nil {
			m.log.Debug("issue cancelling liquidity provision",
				logging.Error(err),
				logging.MarketID(m.GetID()),
				logging.PartyID(v.Party))
		}
		// update shares to remove this party from the shares
		updateShares = true
	}

	if updateShares {
		// force update of shares so they are updated for all
		_ = m.equityShares.Shares(m.liquidity.GetInactiveParties())
	}

	return nil
}

func (m *Market) cancelPendingLiquidityProvision(
	ctx context.Context,
	party string,
	initialMargin uint64,
) error {
	// we will just cancel the party,
	// no bond slashing applied
	if err := m.cancelLiquidityProvision(ctx, party, false, false); err != nil {
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
) error {
	mktID := m.GetID()
	asset, _ := m.mkt.GetAsset()

	mpos, ok := m.position.GetPositionByPartyID(party)
	if !ok {
		m.log.Debug("error getting party position",
			logging.PartyID(party),
			logging.MarketID(mktID))
		return nil
	}

	margin, perr := m.collateral.GetPartyMargin(mpos, asset, mktID)
	if perr != nil {
		m.log.Debug("error getting party margin",
			logging.PartyID(party),
			logging.MarketID(mktID),
			logging.Error(perr))
		return perr
	}
	err := m.resolveClosedOutTraders(ctx, []events.Margin{margin}, order)
	if err != nil {
		m.log.Error("could not resolve out traders",
			logging.MarketID(mktID),
			logging.PartyID(party),
			logging.Error(err))
		return err
	}

	return nil
}

func (m *Market) createInitialLPOrders(ctx context.Context, newOrders []*types.Order) (err error) {
	if len(newOrders) <= 0 {
		return nil
	}

	asset, _ := m.mkt.GetAsset()
	party := newOrders[0].PartyId
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
		party := newOrders[0].PartyId
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
			_, ok := mappedIDs[v.Id]
			if !ok && failedOnID != v.Id {
				// this was not handled before, we need to send an
				v.Status = types.Order_STATUS_REJECTED
				// set margin check failed, it's the only reason we could
				// not place the order at this point
				v.Reason = types.OrderError_ORDER_ERROR_MARGIN_CHECK_FAILED
				m.broker.Send(events.NewOrderEvent(ctx, v))
			}
		}
	}()

	for _, order := range newOrders {
		if _, err := m.submitOrder(ctx, order, false); err != nil {
			failedOnID = order.Id
			m.log.Debug("unable to submit liquidity provision order",
				logging.MarketID(m.GetID()),
				logging.OrderID(order.Id),
				logging.PartyID(order.PartyId),
				logging.Error(err))
			return err
		}
		m.log.Debug("new liquidity order submitted successfully",
			logging.MarketID(m.GetID()),
			logging.OrderID(order.Id),
			logging.PartyID(order.PartyId))
		submittedIDs = append(submittedIDs, order.Id)
	}

	return nil
}

func (m *Market) rollBackMargin(
	ctx context.Context,
	party string,
	initialMargin uint64,
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

	if marginAcc.Balance < initialMargin {
		// nothing to rollback
		return nil
	}

	amount := marginAcc.Balance - initialMargin
	// now create the rollback to transfer
	transfer := types.Transfer{
		Owner: party,
		Amount: &types.FinancialAmount{
			Amount: amount,
			Asset:  asset,
		},
		Type:      types.TransferType_TRANSFER_TYPE_MARGIN_HIGH,
		MinAmount: amount,
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
func (m *Market) repriceFuncW(po *types.PeggedOrder) (uint64, error) {
	return m.getNewPeggedPrice(
		&types.Order{PeggedOrder: po},
	)
}

func (m *Market) cancelLiquidityProvision(
	ctx context.Context, party string, isDistressed, isReplace bool) error {

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
	// distressed traders flow.
	if !isDistressed {
		// now we cancel all existing orders
		for _, order := range cancelOrders {
			if _, err := m.cancelOrder(ctx, party, order.Id); err != nil {
				// nothing much we can do here, I suppose
				// something wrong might have happen...
				// does this need a panic? need to think about it...
				m.log.Debug("unable cancel liquidity order",
					logging.String("party", party),
					logging.String("order-id", order.Id),
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
	// it just mean that the trader my have gone the distressed path
	// also if the balance is already 0, let's not bother created a
	// transfer request
	if err == nil && bondAcc.Balance > 0 {
		transfer := &types.Transfer{
			Owner: party,
			Amount: &types.FinancialAmount{
				Amount: bondAcc.Balance,
				Asset:  asset,
			},
			Type:      types.TransferType_TRANSFER_TYPE_BOND_HIGH,
			MinAmount: bondAcc.Balance,
		}

		tresp, err := m.collateral.BondUpdate(ctx, m.GetID(), party, transfer)
		if err != nil {
			m.log.Debug("bond update error", logging.Error(err))
			return err
		}
		m.broker.Send(events.NewTransferResponse(ctx, []*types.TransferResponse{tresp}))
	}

	if !isReplace {
		// now let's update the fee selection
		m.updateLiquidityFee(ctx)
		// and remove the party from the equity share like calculation
		m.equityShares.SetPartyStake(party, 0)
		// force update of shares so they are updated for all
		_ = m.equityShares.Shares(m.liquidity.GetInactiveParties())
	}

	m.checkLiquidity(ctx, nil)
	m.commandLiquidityAuction(ctx)

	return nil
}

func (m *Market) amendOrCancelLiquidityProvision(
	ctx context.Context, sub *commandspb.LiquidityProvisionSubmission, party, id string,
) error {
	lp := m.liquidity.LiquidityProvisionByPartyID(party)
	if lp == nil {
		return fmt.Errorf("cannot edit liquidity provision from a non liquidity provider party (%v)", party)
	}

	// Increasing the commitment should always be allowed, but decreasing is
	// only valid if the resulting amount still allows the market as a whole
	// to reach it's commitment level. Otherwise the commitment reduction is
	// rejected.
	if sub.CommitmentAmount < lp.CommitmentAmount {
		// first - does the market have enough stake
		if uint64(m.getTargetStake()) >= m.getSuppliedStake() {
			return ErrNotEnoughStake
		}

		// now if the stake surplus is > than the change we are OK
		surplus := m.getSuppliedStake() - uint64(m.getTargetStake())
		diff := lp.CommitmentAmount - sub.CommitmentAmount
		if surplus < diff {
			return ErrNotEnoughStake
		}
	}

	// here, we now we have a amendment
	// if this amendment is to reduce the stake to 0, then we'll want to
	// cancel this lp submission
	if sub.CommitmentAmount == 0 {
		return m.cancelLiquidityProvision(ctx, party, false, false)
	}

	// if commitment != 0, then it's an amend
	return m.amendLiquidityProvision(ctx, sub, party)
}

func (m *Market) amendLiquidityProvision(
	ctx context.Context, sub *commandspb.LiquidityProvisionSubmission, party string,
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
				ctx, m.GetID(), party, bondRollback)
			if newerr != nil {
				m.log.Debug("unable to rollback bond account topup",
					logging.String("party", party),
					logging.Uint64("amount", bondRollback.Amount.Amount),
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
	ctx context.Context, sub *commandspb.LiquidityProvisionSubmission, party string,
) error {
	// first try to get the indicative uncrossing price from the book
	price := m.matching.GetIndicativePrice()
	if price == 0 {
		// here it is 0 so we will use the mark price
		price = m.markPrice
	}

	// now let's check if we are still at 0, if yes, it means we are in the
	// third condition from before, no price available, we just accept the
	// amendment without deploying any orders, so no need to check any margin etc
	if price > 0 {
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
// from there, then move the funds in the party margin account
func (m *Market) calcLiquidityProvisionPotentialMarginsAuction(
	ctx context.Context,
	sub *commandspb.LiquidityProvisionSubmission,
	party string,
	price uint64,
) error {
	repriceFn := func(o *types.PeggedOrder) (uint64, error) {
		if o.Offset >= 0 {
			return price + uint64(o.Offset), nil
		}

		// At this stage offset is negative so we change it's sign to cast it to an
		// unsigned type
		offset := uint64(-o.Offset)
		if price <= offset {
			return 0, ErrUnableToReprice
		}

		return price - offset, nil
	}

	// first lets get the protential shape for this submission
	orders, err := m.liquidity.GetPotentialShapeOrders(
		party, price, sub, repriceFn)
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
		pos = &positions.MarketPosition{}
		pos.SetParty(party)
	}

	// now we register all these orders as potential positions
	// which we will use to calculate the margin just after
	for _, order := range orders {
		pos.RegisterOrder(order)
	}

	// then calculate the margins,
	// any shortfall is a blocker here.
	risk, err := m.calcMarginsLiquidityProvisionAmendAuction(ctx, pos, price)
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
	ctx context.Context, sub *commandspb.LiquidityProvisionSubmission, party string,
) error {
	// first parameter is the update to the orders, but we know that during
	// auction no orders shall be return, so let's just look at the error
	_, err := m.liquidity.AmendLiquidityProvision(ctx, sub, party)
	if err != nil {
		m.log.Panic("error while amending liquidity provision, this should not happen at this point, the LP was validated earlier",
			logging.Error(err))
	}

	m.updateMarketValueProxy()
	// now we can update the liquidity fee to be taken
	m.updateLiquidityFee(ctx)
	// now we can setup our party stake to calculate equities
	m.equityShares.SetPartyStake(party, sub.CommitmentAmount)
	// force update of shares so they are updated for all
	_ = m.equityShares.Shares(m.liquidity.GetInactiveParties())

	m.checkLiquidity(ctx, nil)
	m.commandLiquidityAuction(ctx)

	return nil
}

func (m *Market) amendLiquidityProvisionContinuous(
	ctx context.Context, sub *commandspb.LiquidityProvisionSubmission, party string,
) error {
	midPriceBid, _, err := m.getStaticMidPrices()
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
		party, midPriceBid, sub, m.repriceFuncW)
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
		pos = &positions.MarketPosition{}
		pos.SetParty(party)
	}

	// first remove all existing orders from the potential positions
	lorders := m.liquidity.GetLiquidityOrders(party)
	for _, v := range lorders {
		// ensure the order is on the actual potential position first
		if order, foundOnBook, _ := m.getOrderByID(v.Id); foundOnBook {
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
	ctx context.Context, sub *commandspb.LiquidityProvisionSubmission, party string,
) error {
	// first parameter is the update to the orders, but we know that during
	// auction no orders shall be return, so let's just look at the error
	cancels, err := m.liquidity.AmendLiquidityProvision(ctx, sub, party)
	if err != nil {
		m.log.Panic("error while amending liquidity provision, this should not happen at this point, the LP was validated earlier",
			logging.Error(err))
	}

	for _, order := range cancels {
		if _, err := m.cancelOrder(ctx, party, order.Id); err != nil {
			// nothing much we can do here, I suppose
			// something wrong might have happen...
			// does this need a panic? need to think about it...
			m.log.Debug("unable cancel liquidity order",
				logging.String("party", party),
				logging.String("order-id", order.Id),
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

		m.checkLiquidity(ctx, nil)
		m.commandLiquidityAuction(ctx)

	}()

	// this workd but we definitely trigger some recursive loop which
	// are unlikely to be fine.
	m.liquidityUpdate(ctx, nil)

	return nil
}

// returns the rollback transfer in case of error
func (m *Market) ensureLiquidityProvisionBond(
	ctx context.Context, sub *commandspb.LiquidityProvisionSubmission, party string,
) (*types.Transfer, error) {
	asset, _ := m.mkt.GetAsset()
	bondAcc, err := m.collateral.GetOrCreatePartyBondAccount(
		ctx, party, m.GetID(), asset)
	if err != nil {
		return nil, err
	}

	// first check if there's enough funds in the gen + bond
	// account to cover the new commitment
	if !m.collateral.CanCoverBond(m.GetID(), party, asset, sub.CommitmentAmount) {
		return nil, ErrCommitmentSubmissionNotAllowed
	}

	// build our transfer to be sent to collateral
	amount := int64(sub.CommitmentAmount - bondAcc.Balance)
	ty := types.TransferType_TRANSFER_TYPE_BOND_LOW
	if amount < 0 {
		ty = types.TransferType_TRANSFER_TYPE_BOND_HIGH
		amount = -amount
	}
	transfer := &types.Transfer{
		Owner: party,
		Amount: &types.FinancialAmount{
			Amount: uint64(amount),
			Asset:  asset,
		},
		Type:      ty,
		MinAmount: uint64(amount),
	}

	// move our bond
	tresp, err := m.collateral.BondUpdate(ctx, m.GetID(), party, transfer)
	if err != nil {
		return nil, err
	}
	m.broker.Send(events.NewTransferResponse(
		ctx, []*types.TransferResponse{tresp}))

	// now we will use the actuall transfer as a rollback later on eventually
	// so let's just change from HIGH to LOW and inverse
	if transfer.Type == types.TransferType_TRANSFER_TYPE_BOND_HIGH {
		transfer.Type = types.TransferType_TRANSFER_TYPE_BOND_LOW
	} else {
		transfer.Type = types.TransferType_TRANSFER_TYPE_BOND_HIGH
	}

	return transfer, nil
}
