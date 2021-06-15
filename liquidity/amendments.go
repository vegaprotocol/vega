package liquidity

import (
	"context"
	"errors"
	"sort"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/liquidity/supplied"
	commandspb "code.vegaprotocol.io/vega/proto/commands/v1"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
)

var (
	ErrPartyHaveNoLiquidityProvision = errors.New("party have no liquidity provision")
)

func (e *Engine) CanAmend(
	lps *commandspb.LiquidityProvisionSubmission,
	party string,
) error {
	// does the party is an LP
	_, ok := e.provisions[party]
	if !ok {
		return ErrPartyHaveNoLiquidityProvision
	}

	// is the new submission valid?
	if err := e.ValidateLiquidityProvisionSubmission(lps, false); err != nil {
		return err
	}

	// yes
	return nil
}

func (e *Engine) AmendLiquidityProvision(
	ctx context.Context,
	lps *commandspb.LiquidityProvisionSubmission,
	party string,
) ([]*types.Order, error) {
	if err := e.CanAmend(lps, party); err != nil {
		return nil, err
	}

	// LP exists, checked in the previous func
	lp := e.provisions[party]

	// first we get all orders from this party to be cancelled
	// get the liquidity order to be cancelled
	cancels := make([]*types.Order, 0, len(e.liquidityOrders[party]))
	for _, o := range e.liquidityOrders[party] {
		cancels = append(cancels, o)
	}

	sort.Slice(cancels, func(i, j int) bool {
		return cancels[i].Id < cancels[j].Id
	})

	// now let's apply all changes
	// first reset the lp orders map
	e.liquidityOrders[party] = map[string]*types.Order{}
	// then update the LP
	lp.UpdatedAt = e.currentTime.UnixNano()
	lp.CommitmentAmount = num.NewUint(lps.CommitmentAmount)
	lp.Fee = lps.Fee
	lp.Reference = lps.Reference
	// only if it's active, we don't want to loose a PENDING
	// status here.
	if lp.Status == types.LiquidityProvision_STATUS_ACTIVE {
		lp.Status = types.LiquidityProvision_STATUS_UNDEPLOYED
	}

	e.buildLiquidityProvisionShapesReferences(lp, lps)

	e.broker.Send(events.NewLiquidityProvisionEvent(ctx, lp))
	return cancels, nil
}

// GetPotentialShapeOrders is used to create orders from
// shape when amending a liquidity provision this allows us to
// ensure enough funds can be taken from the margin account in orders
// to submit orders later on.
func (e *Engine) GetPotentialShapeOrders(
	party string,
	bestBidPrice, bestAskPrice *num.Uint,
	lps *commandspb.LiquidityProvisionSubmission,
	repriceFn RepricePeggedOrder,
) ([]*types.Order, error) {
	if err := e.ValidateLiquidityProvisionSubmission(lps, false); err != nil {
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
			order.Price.Copy(price)
			shape = append(shape, order)
		}
		return shape, true
	}

	buyShape, ok := priceShape(lps.Buys, types.Side_SIDE_BUY)
	if !ok {
		return nil, errors.New("unable to price buy shape")
	}
	sellShape, ok := priceShape(lps.Sells, types.Side_SIDE_SELL)
	if !ok {
		return nil, errors.New("unable to price sell shape")
	}

	// Update this once we have updated the commitment value to use Uint TODO UINT
	ob := float64(lps.CommitmentAmount) * e.stakeToObligationFactor
	obligation, _ := num.UintFromDecimal(num.DecimalFromFloat(ob))
	// Create a slice shaped copy of the orders
	orders := make([]*types.Order, 0, len(e.orders[party]))
	for _, order := range e.orders[party] {
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
	toCreate := e.buildPotentialShapeOrders(party, buyShape, types.Side_SIDE_BUY)
	toCreate = append(toCreate,
		e.buildPotentialShapeOrders(party, sellShape, types.Side_SIDE_SELL)...)

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
		order := e.buildOrder(side, o.Peg, o.Price, party, e.marketID, o.LiquidityImpliedVolume, "", "")
		orders = append(orders, order)
	}

	return orders
}
