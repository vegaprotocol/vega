package execution

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/positions"
	types "code.vegaprotocol.io/vega/proto"
	commandspb "code.vegaprotocol.io/vega/proto/commands/v1"
)

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
