package liquidity

import (
	"context"
	"errors"
	"sort"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/liquidity/supplied"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
)

var ErrPartyHaveNoLiquidityProvision = errors.New("party have no liquidity provision")

func (e *Engine) CanAmend(
	lps *types.LiquidityProvisionAmendment,
	party string,
) error {
	// does the party is an LP
	_, ok := e.provisions.Get(party)
	if !ok {
		return ErrPartyHaveNoLiquidityProvision
	}

	// is the new amendment valid?
	if err := e.ValidateLiquidityProvisionAmendment(lps); err != nil {
		return err
	}

	// yes
	return nil
}

func (e *Engine) AmendLiquidityProvision(
	ctx context.Context,
	lps *types.LiquidityProvisionAmendment,
	party string,
) ([]*types.Order, error) {
	if err := e.CanAmend(lps, party); err != nil {
		return nil, err
	}

	// LP exists, checked in the previous func
	lp, _ := e.provisions.Get(party)

	// first we get all orders from this party to be cancelled
	// get the liquidity order to be cancelled
	// NOTE: safe to iterate over the map straight away here  as
	// no operation is done on orders
	cancels := make([]*types.Order, 0, len(e.liquidityOrders.m[party]))
	for _, o := range e.liquidityOrders.m[party] {
		cancels = append(cancels, o)
	}

	sort.Slice(cancels, func(i, j int) bool {
		return cancels[i].ID < cancels[j].ID
	})

	// now let's apply all changes
	// first reset the lp orders map
	e.liquidityOrders.ResetForParty(party)
	// then update the LP
	lp.UpdatedAt = e.currentTime.UnixNano()
	lp.CommitmentAmount = lps.CommitmentAmount.Clone()
	lp.Fee = lps.Fee
	lp.Reference = lps.Reference
	// only if it's active, we don't want to loose a PENDING
	// status here.
	if lp.Status == types.LiquidityProvisionStatusActive {
		lp.Status = types.LiquidityProvisionStatusUndeployed
	}

	e.buildLiquidityProvisionShapesReferences(lp, lps.Buys, lps.Sells)
	e.broker.Send(events.NewLiquidityProvisionEvent(ctx, lp))
	e.provisions.Set(party, lp)
	return cancels, nil
}

// GetPotentialShapeOrders is used to create orders from
// shape when amending a liquidity provision this allows us to
// ensure enough funds can be taken from the margin account in orders
// to submit orders later on.
func (e *Engine) GetPotentialShapeOrders(
	party string,
	bestBidPrice, bestAskPrice *num.Uint,
	lps *types.LiquidityProvisionAmendment,
	repriceFn RepricePeggedOrder,
) ([]*types.Order, error) {
	if err := e.ValidateLiquidityProvisionAmendment(lps); err != nil {
		return nil, err
	}

	priceShape := func(loShape []*types.LiquidityOrder, side types.Side) ([]*supplied.LiquidityOrder, bool) {
		shape := make([]*supplied.LiquidityOrder, 0, len(loShape))
		for _, lorder := range loShape {
			pegged := &types.PeggedOrder{
				Reference: lorder.Reference,
				Offset:    lorder.Offset,
			}
			order := &supplied.LiquidityOrder{
				Proportion: uint64(lorder.Proportion),
				Peg:        pegged,
			}
			price, _, err := repriceFn(pegged, side)
			if err != nil {
				return nil, false
			}
			order.Price = price
			shape = append(shape, order)
		}
		return shape, true
	}

	buyShape, ok := priceShape(lps.Buys, types.SideBuy)
	if !ok {
		return nil, errors.New("unable to price buy shape")
	}
	sellShape, ok := priceShape(lps.Sells, types.SideSell)
	if !ok {
		return nil, errors.New("unable to price sell shape")
	}

	// Update this once we have updated the commitment value to use Uint TODO UINT
	obligation, _ := num.UintFromDecimal(lps.CommitmentAmount.ToDecimal().Mul(e.stakeToObligationFactor).Round(0))
	// Create a slice shaped copy of the orders
	orders := make([]*types.Order, 0, len(e.orders.m[party]))
	for _, order := range e.orders.m[party] {
		orders = append(orders, order)
	}

	// now try to calculate the implied volume for our shape,
	// any error would exit straight away
	if err := e.suppliedEngine.CalculateLiquidityImpliedVolumes(
		bestBidPrice, bestAskPrice,
		obligation,
		orders,
		buyShape, sellShape,
	); err != nil {
		return nil, err
	}

	// from this point we should have no error possible, let's just
	// make the order shapes
	toCreate := e.buildPotentialShapeOrders(party, buyShape, types.SideBuy)
	toCreate = append(toCreate,
		e.buildPotentialShapeOrders(party, sellShape, types.SideSell)...)

	return toCreate, nil
}

func (e *Engine) buildPotentialShapeOrders(party string, supplied []*supplied.LiquidityOrder, side types.Side) []*types.Order {
	orders := make([]*types.Order, 0, len(supplied))

	for _, o := range supplied {
		// only add order with non = volume to the list
		if o.LiquidityImpliedVolume == 0 {
			continue
		}

		// no need to make it a proper pegged order, set an actual ID etc here
		// as we actually just return this order as a template for margin
		// calculation
		order := e.buildOrder(side, o.Price, party, e.marketID, o.LiquidityImpliedVolume, "", "")
		orders = append(orders, order)
	}

	return orders
}
