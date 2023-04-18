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

package liquidity

import (
	"context"
	"errors"
	"sort"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/liquidity/supplied"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/protos/vega"
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
	lpa *types.LiquidityProvisionAmendment,
	party string,
	idGen IDGen,
) ([]*types.Order, error) {
	if err := e.CanAmend(lpa, party); err != nil {
		return nil, err
	}

	// LP exists, checked in the previous func
	lp, _ := e.provisions.Get(party)

	// first we get all orders from this party to be cancelled
	// get the liquidity order to be cancelled
	// NOTE: safe to iterate over the map straight away here  as
	// no operation is done on orders
	cancels := e.orderBook.GetLiquidityOrders(party)
	sort.Slice(cancels, func(i, j int) bool {
		return cancels[i].ID < cancels[j].ID
	})

	cancelsM := map[string]struct{}{}
	for _, c := range cancels {
		cancelsM[c.ID] = struct{}{}
	}

	now := e.timeService.GetTimeNow().UnixNano()
	orderEvts := e.getCancelAllLiquidityOrders(
		ctx, lp, cancelsM, types.OrderStatusStopped, now)

	// update the LP
	lp.UpdatedAt = now
	lp.CommitmentAmount = lpa.CommitmentAmount.Clone()
	lp.Fee = lpa.Fee
	lp.Reference = lpa.Reference
	// only if it's active, we don't want to loose a PENDING
	// status here.
	if lp.Status == types.LiquidityProvisionStatusActive {
		lp.Status = types.LiquidityProvisionStatusUndeployed
	}
	// update version
	lp.Version++

	orderEvts = append(orderEvts, e.SetShapesReferencesOnLiquidityProvision(ctx, lp, lpa.Buys, lpa.Sells, idGen)...)
	// seed the dummy orders with the generated IDs in order to avoid broken references
	e.broker.SendBatch(orderEvts)
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
	minLpPrice, maxLpPrice *num.Uint,
	lps *types.LiquidityProvisionAmendment,
	repriceFn RepriceOrder,
) ([]*types.Order, error) {
	if err := e.ValidateLiquidityProvisionAmendment(lps); err != nil {
		return nil, err
	}

	priceShape := func(loShape []*types.LiquidityOrder, side types.Side) ([]*supplied.LiquidityOrder, bool) {
		shape := make([]*supplied.LiquidityOrder, 0, len(loShape))
		for _, lorder := range loShape {
			order := &supplied.LiquidityOrder{
				Details: lorder,
			}
			price, err := repriceFn(side, lorder.Reference, lorder.Offset.Clone())
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
	partyOrders := e.orderBook.GetOrdersPerParty(party)
	orders := make([]*types.Order, 0, len(partyOrders))
	for _, order := range partyOrders {
		if !order.IsLiquidityOrder() && order.Status == vega.Order_STATUS_ACTIVE {
			orders = append(orders, order)
		}
	}

	// now calculate the implied volume for our shape
	e.suppliedEngine.CalculateLiquidityImpliedVolumes(
		obligation,
		orders,
		minLpPrice, maxLpPrice,
		buyShape, sellShape,
	)

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
