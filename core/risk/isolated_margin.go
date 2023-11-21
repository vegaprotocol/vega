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

package risk

import (
	"context"
	"fmt"
	"sort"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
)

// calculateIsolatedMargins calculates the required margins for a party in isolated margin mode.
// It is calculating margin the same way as the cross margin mode does and then enriches the result with the order margin requirement.
// auctionPrice is nil if not in an auction, otherwise the max(markPrice, indicativePrice).
func (e *Engine) calculateIsolatedMargins(m events.Margin, marketObservable *num.Uint, rf types.RiskFactor, auction bool, inc num.Decimal, orders []*types.Order, marginFactor, positionFactor num.Decimal, auctionPrice *num.Uint) *types.MarginLevels {
	margins := e.calculateMargins(m, marketObservable, rf, true, auction, inc)
	margins.OrderMargin = calcOrderMargins(m.Size(), orders, marginFactor, positionFactor, auctionPrice)
	margins.MarginMode = types.MarginModeIsolatedMargin
	margins.MarginFactor = marginFactor
	return margins
}

// CheckIsolatedMarginAuction checks that the party has sufficient cover in their order margin account during an auction for
// their orders. It returns an error if the party doesn't have sufficient cover, and the necessary transfers otherwise.
// NB: auctionPrice should be nil in continuous mode.
// NB: price is the marketObservable.
func (e *Engine) CheckIsolatedMargins(ctx context.Context, evt events.Margin, orders []*types.Order, marketObservable *num.Uint, auctionPrice *num.Uint, increment num.Decimal, marginFactor, positionFactor num.Decimal) (events.Risk, error) {
	margins := e.calculateIsolatedMargins(evt, marketObservable, *e.factors, auctionPrice != nil, increment, orders, marginFactor, positionFactor, auctionPrice)
	return checkIsolatedMargins(ctx, evt, margins, e.updateMarginLevels)
}

// checkIsolatedMargins checks that the order margin is valid (i.e. >= maintenance) and that the party has sufficient balance to cover them.
// If the party doesn't have sufficient cover, an error is returned, otherwise the transfers to top up or free funds
// to/from the order margin account are setup.
func checkIsolatedMargins(ctx context.Context, evt events.Margin, margins *types.MarginLevels, updateMarginLevels func(...*events.MarginLevels)) (events.Risk, error) {
	// if the margin account balance + the required order margin is less than the maintenance margin, return error
	if num.Sum(evt.MarginBalance(), margins.OrderMargin).LT(margins.MaintenanceMargin) {
		return nil, ErrInsufficientFundsForMaintenanceMargin
	}
	if margins.OrderMargin.GT(evt.OrderMarginBalance()) && num.UintZero().Sub(margins.OrderMargin, evt.OrderMarginBalance()).GT(evt.GeneralBalance()) {
		return nil, ErrInsufficientFundsForMarginInGeneralAccount
	}

	var amt *num.Uint
	tp := types.TransferTypeOrderMarginLow
	if margins.OrderMargin.GT(evt.OrderMarginBalance()) {
		amt = num.UintZero().Sub(margins.OrderMargin, evt.OrderMarginBalance())
	} else {
		amt = num.UintZero().Sub(evt.OrderMarginBalance(), margins.OrderMargin)
		tp = types.TransferTypeOrderMarginHigh
	}

	trnsfr := &types.Transfer{
		Owner: evt.Party(),
		Type:  tp,
		Amount: &types.FinancialAmount{
			Asset:  evt.Asset(),
			Amount: amt,
		},
		MinAmount: amt.Clone(),
	}
	// propagate margins levels to the buffer
	updateMarginLevels(events.NewMarginLevelsEvent(ctx, *margins))
	change := &marginChange{
		Margin:   evt,
		transfer: trnsfr,
		margins:  margins,
	}
	return change, nil
}

// CheckIsolatedMarginsOnNewOrderTraded is called when a new order comes in and is matched immediately. If it doesn't
// NB: evt has the expected position after the trades + orders need to include the new order with the updated remaining somehow.
func (e *Engine) CheckIsolatedMarginsOnNewOrderTraded(ctx context.Context, evt events.Margin, marketObservable, auctionPrice *num.Uint, increment num.Decimal, orders []*types.Order, trades []*types.Trade, marginFactor, positionFactor num.Decimal, traderSide types.Side) (events.Risk, error) {
	if evt == nil {
		return nil, nil
	}
	margins := e.calculateIsolatedMargins(evt, marketObservable, *e.factors, false, increment, orders, marginFactor, positionFactor, auctionPrice)

	// no margins updates, nothing to do then
	if margins == nil {
		return nil, nil
	}

	// update other fields for the margins
	margins.Party = evt.Party()
	margins.Asset = evt.Asset()
	margins.Timestamp = e.timeSvc.GetTimeNow().UnixNano()
	margins.MarketID = e.mktID

	tradedSize := int64(0)
	side := trades[0].Aggressor
	marginToAdd := num.UintZero()
	for _, t := range trades {
		tradedSize += int64(t.Size)
		marginToAdd.AddSum(num.UintZero().Mul(t.Price, num.NewUint(t.Size)))
	}
	if side == types.SideSell {
		tradedSize = -tradedSize
	}
	oldPosition := evt.Size() - tradedSize
	if evt.Size()*oldPosition > 0 { // position didn't switch sides
		if int64Abs(oldPosition) > int64Abs(evt.Size()) { // position increased
			marginToAdd, _ = num.UintFromDecimal(marginToAdd.ToDecimal().Div(positionFactor).Mul(marginFactor))
			if num.Sum(evt.MarginBalance(), marginToAdd).LT(margins.MaintenanceMargin) {
				return nil, ErrInsufficientFundsForMaintenanceMargin
			}
			if marginToAdd.GT(evt.GeneralBalance()) {
				return nil, ErrInsufficientFundsForMarginInGeneralAccount
			}
		}
	} else {
		// position did switch sides
		marginToAdd = num.UintZero()
		pos := int64Abs(oldPosition)
		totalSize := uint64(0)
		for _, t := range trades {
			if pos >= t.Size {
				pos -= t.Size
				totalSize += t.Size
			} else if pos == 0 {
				marginToAdd.AddSum(num.UintZero().Mul(t.Price, num.NewUint(t.Size)))
			} else {
				size := t.Size - pos
				marginToAdd.AddSum(num.UintZero().Mul(t.Price, num.NewUint(size)))
				pos = 0
			}
		}
		// The new margin required balance is marginToAdd, so we need to check that:
		// 1) it's greater than maintenance margin to keep the invariant
		// 2) there are sufficient funds in what's currently in the margin account + general account to cover for the new required margin
		marginToAdd, _ = num.UintFromDecimal(marginToAdd.ToDecimal().Div(positionFactor).Mul(marginFactor))
		if marginToAdd.LT(margins.MaintenanceMargin) {
			return nil, ErrInsufficientFundsForMaintenanceMargin
		}
		if marginToAdd.GT(num.Sum(evt.GeneralBalance(), evt.MarginBalance())) {
			return nil, ErrInsufficientFundsForMarginInGeneralAccount
		}
	}

	transfer := getIsolatedMarginTransfersOnPositionChange(evt.Party(), evt.Asset(), trades, traderSide, evt.Size(), e.positionFactor, marginFactor, evt.MarginBalance(), marketObservable)
	e.updateMarginLevels(events.NewMarginLevelsEvent(ctx, *margins))
	change := &marginChange{
		Margin:   evt,
		transfer: transfer,
		margins:  margins,
	}
	return change, nil
}

// UpdateIsolatedMarginOnPositionChanged is called upon changes to the position of a party in isolated margin mode.
// Depending on the nature of the change it checks if it needs to move funds into our out of the margin account from the
// order margin account or to the general account.
// At this point we don't enforce any invariants just calculate transfers.
func (e *Engine) UpdateIsolatedMarginsOnPositionChange(ctx context.Context, evt events.Margin, marketObservable *num.Uint, increment num.Decimal, orders []*types.Order, trades []*types.Trade, traderSide types.Side, marginFactor num.Decimal, auctionPrice *num.Uint) (events.Risk, error) {
	if evt == nil {
		return nil, nil
	}
	margins := e.calculateIsolatedMargins(evt, marketObservable, *e.factors, false, increment, orders, marginFactor, e.positionFactor, auctionPrice)
	if margins == nil {
		return nil, nil
	}

	margins.Party = evt.Party()
	margins.Asset = evt.Asset()
	margins.Timestamp = e.timeSvc.GetTimeNow().UnixNano()
	margins.MarketID = e.mktID

	transfer := getIsolatedMarginTransfersOnPositionChange(evt.Party(), evt.Asset(), trades, traderSide, evt.Size(), e.positionFactor, marginFactor, evt.MarginBalance(), marketObservable)
	e.updateMarginLevels(events.NewMarginLevelsEvent(ctx, *margins))
	change := &marginChange{
		Margin:   evt,
		transfer: transfer,
		margins:  margins,
	}
	return change, nil
}

// getIsolatedMarginTransfersOnPositionChange returns the transfers that need to be made to/from the margin account in isolated margin mode
// when the position changes. This handles the 3 different cases of position change (increase, decrease, switch sides).
// NB: positionSize is *after* the trades.
func getIsolatedMarginTransfersOnPositionChange(party, asset string, trades []*types.Trade, traderSide types.Side, positionSize int64, positionFactor, marginFactor num.Decimal, curMarginBalance, markPrice *num.Uint) *types.Transfer {
	positionDelta := int64(0)
	marginToAdd := num.UintZero()
	vwap := num.UintZero()
	for _, t := range trades {
		positionDelta += int64(t.Size)
		marginToAdd.AddSum(num.UintZero().Mul(t.Price, num.NewUint(t.Size)))
		vwap.AddSum(num.UintZero().Mul(t.Price, num.NewUint(t.Size)))
	}
	vwap = num.UintZero().Div(vwap, num.NewUint(uint64(positionDelta)))
	if traderSide == types.SideSell {
		positionDelta = -positionDelta
	}
	oldPosition := positionSize - positionDelta

	if positionSize*oldPosition >= 0 { // position didn't switch sides
		if int64Abs(oldPosition) < int64Abs(positionSize) { // position increased
			marginToAdd, _ = num.UintFromDecimal(marginToAdd.ToDecimal().Div(positionFactor).Mul(marginFactor))
			// need to top up the margin account from the order margin account
			return &types.Transfer{
				Owner: party,
				Type:  types.TransferTypeIsolatedMarginLow,
				Amount: &types.FinancialAmount{
					Asset:  asset,
					Amount: marginToAdd,
				},
				MinAmount: marginToAdd,
			}
		}
		// position decreased
		// marginToRelease = balanceBefore + positionBefore x (newTradeVWAP - markPrice) x |totalTradeSize|/|positionBefore|
		theoreticalAccountBalance, _ := num.UintFromDecimal(vwap.ToDecimal().Sub(markPrice.ToDecimal()).Mul(num.DecimalFromInt64(int64(int64Abs(oldPosition)))).Div(positionFactor).Add(curMarginBalance.ToDecimal()))
		marginToRelease := num.UintZero().Div(num.UintZero().Mul(theoreticalAccountBalance, num.NewUint(int64Abs(positionDelta))), num.NewUint(int64Abs(oldPosition)))
		// need to top up the margin account
		return &types.Transfer{
			Owner: party,
			Type:  types.TransferTypeMarginHigh,
			Amount: &types.FinancialAmount{
				Asset:  asset,
				Amount: marginToRelease,
			},
			MinAmount: marginToRelease,
		}
	}

	// position switched sides, we need to handles the two sides separately
	// first calculate the amount that would be released
	marginToRelease := curMarginBalance.Clone()
	marginToAdd = num.UintZero()
	pos := int64Abs(oldPosition)
	totalSize := uint64(0)
	for _, t := range trades {
		if pos >= t.Size {
			pos -= t.Size
			totalSize += t.Size
		} else if pos == 0 {
			marginToAdd.AddSum(num.UintZero().Mul(t.Price, num.NewUint(t.Size)))
		} else {
			size := t.Size - pos
			marginToAdd.AddSum(num.UintZero().Mul(t.Price, num.NewUint(size)))
			pos = 0
		}
	}
	marginToAdd, _ = num.UintFromDecimal(marginToAdd.ToDecimal().Div(positionFactor).Mul(marginFactor))
	topup := num.UintZero()
	release := num.UintZero()
	if marginToAdd.GT(marginToRelease) {
		topup = num.UintZero().Sub(marginToAdd, marginToRelease)
	} else {
		release = num.UintZero().Sub(marginToRelease, marginToAdd)
	}

	amt := topup
	tp := types.TransferTypeMarginLow
	if !release.IsZero() {
		amt = release
		tp = types.TransferTypeMarginHigh
	}

	return &types.Transfer{
		Owner: party,
		Type:  tp,
		Amount: &types.FinancialAmount{
			Asset:  asset,
			Amount: amt,
		},
		MinAmount: amt,
	}
}

// calcOrderMargins calculates the the order margin required for the party given their current orders and margin factor.
func calcOrderMargins(positionSize int64, orders []*types.Order, positionFactor, marginFactor num.Decimal, auctionPrice *num.Uint) *num.Uint {
	if len(orders) == 0 {
		return num.UintZero()
	}
	buyOrders := []*types.Order{}
	sellOrders := []*types.Order{}
	// split orders by side
	for _, o := range orders {
		if o.Side == types.SideBuy {
			buyOrders = append(buyOrders, o)
		} else {
			sellOrders = append(sellOrders, o)
		}
	}
	// sort orders from best to worst
	sort.Slice(buyOrders, func(i, j int) bool { return buyOrders[i].Price.GT(buyOrders[j].Price) })
	sort.Slice(sellOrders, func(i, j int) bool { return sellOrders[i].Price.LT(sellOrders[j].Price) })

	// calc the side margin
	marginByBuy := calcOrderSideMargin(positionSize, buyOrders, positionFactor, marginFactor, auctionPrice)
	marginBySell := calcOrderSideMargin(positionSize, sellOrders, positionFactor, marginFactor, auctionPrice)
	orderMargin := marginByBuy
	if marginBySell.GT(orderMargin) {
		orderMargin = marginBySell
	}
	return orderMargin
}

// calcOrderSideMargin returns the amount of order margin needed given the current position and party orders.
// Given the sorted orders of the side for the party (sorted from best to worst)
// If the party currently has a position x, assign 0 margin requirement the first-to-trade x of volume on the opposite side as this
// would reduce their position (for example, if a party had a long position 10 and sell orders of 15 at a price of $100 and 10
// at a price of $150, the first 10 of the sell order at $100 would not require any order margin).
// For any remaining volume, sum side margin = limit price * size * margin factor for each price level, as this is
// the worst-case trade price of the remaining component.
func calcOrderSideMargin(currentPosition int64, orders []*types.Order, positionFactor, marginFactor num.Decimal, auctionPrice *num.Uint) *num.Uint {
	margin := num.UintZero()
	remainingCovered := int64Abs(currentPosition)
	for _, o := range orders {
		if o.Status != types.OrderStatusActive {
			continue
		}
		size := o.TrueRemaining()
		// for long position we don't need to count margin for the top <currentPosition> size for sell orders
		// for short position we don't need to count margin for the top <currentPosition> size for buy orders
		if remainingCovered != 0 && (o.Side == types.SideBuy && currentPosition < 0) || (o.Side == types.SideSell && currentPosition > 0) {
			if size >= remainingCovered { // part of the order doesn't require margin
				size = size - remainingCovered
				remainingCovered = 0
			} else { // the entire order doesn't require margin
				remainingCovered -= size
				size = 0
			}
		}
		if size > 0 {
			// if we're in auction we need to use the larger between auction price (which is the max(indicativePrice, markPrice)) and the order price
			p := o.Price
			if auctionPrice != nil && auctionPrice.GT(p) {
				p = auctionPrice
			}
			// add the margin for the given order
			margin.AddSum(num.UintZero().Mul(num.NewUint(size), p))
		}
	}
	// factor the margin by margin factor and divide by position factor to get to the right decimals
	margin, _ = num.UintFromDecimal(margin.ToDecimal().Mul(marginFactor).Div(positionFactor))
	return margin
}

// int64Abs returns the absolute uint64 value of the given int64 n.
func int64Abs(n int64) uint64 {
	if n < 0 {
		return uint64(-n)
	}
	return uint64(n)
}

func (e *Engine) SwitchToIsolatedMargin(evt events.Margin, marketObservable *num.Uint, inc num.Decimal, orders []*types.Order, marginFactor num.Decimal, auctionPrice *num.Uint) ([]events.Risk, error) {
	crossMargins := e.calculateIsolatedMargins(evt, marketObservable, *e.factors, auctionPrice != nil, inc, orders, marginFactor, e.positionFactor, auctionPrice)
	return switchToIsolatedMargin(evt, crossMargins, orders, marginFactor, e.positionFactor)
}

func switchToIsolatedMargin(evt events.Margin, margin *types.MarginLevels, orders []*types.Order, marginFactor, positionFactor num.Decimal) ([]events.Risk, error) {
	// switching from cross margin to isolated margin or changing margin factor
	// 1. For any active position, calculate average entry price * abs(position) * margin factor.
	//    Calculate the amount of funds which will be added to, or subtracted from, the general account in order to do this.
	//    If additional funds must be added which are not available, reject the transaction immediately.
	// 2. For any active orders, calculate the quantity limit price * remaining size * margin factor which needs to be placed
	//    in the order margin account. Add this amount to the difference calculated in step 1.
	//    If this amount is less than or equal to the amount in the general account,
	//    perform the transfers (first move funds into/out of margin account, then move funds into the order margin account).
	//    If there are insufficient funds, reject the transaction.
	// 3. Move account to isolated margin mode on this market
	marginAccountBalance := evt.MarginBalance()
	generalAccountBalance := evt.GeneralBalance()
	orderMarginAccountBalance := evt.OrderMarginBalance()
	totalOrderNotional := num.UintZero()
	for _, o := range orders {
		totalOrderNotional = totalOrderNotional.AddSum(num.UintZero().Mul(o.Price, num.NewUint(o.TrueRemaining())))
	}

	positionSize := int64Abs(evt.Size())
	requiredPositionMargin := num.UintZero().Mul(evt.AverageEntryPrice(), num.NewUint(positionSize)).ToDecimal().Mul(marginFactor).Div(positionFactor)
	requireOrderMargin := totalOrderNotional.ToDecimal().Mul(marginFactor).Div(positionFactor)

	// check that we have enough in the general account for any top up needed, i.e.
	// topupNeeded = requiredPositionMargin + requireOrderMargin - marginAccountBalance
	// if topupNeeded > generalAccountBalance => fail
	if requiredPositionMargin.Add(requireOrderMargin).Sub(marginAccountBalance.ToDecimal()).GreaterThan(generalAccountBalance.ToDecimal()) {
		return nil, fmt.Errorf("insufficient balance in general account to cover for required order margin")
	}

	// average entry price * current position * new margin factor (aka requiredPositionMargin) must be above the initial margin for the current position or the transaction will be rejected
	if !requiredPositionMargin.GreaterThan(margin.InitialMargin.ToDecimal()) {
		return nil, fmt.Errorf("required position margin must be greater than initial margin")
	}

	// we're all good, just need to setup the transfers for collateral topup/release
	uRequiredPositionMargin, _ := num.UintFromDecimal(requiredPositionMargin)
	riskEvents := []events.Risk{}
	if !uRequiredPositionMargin.EQ(marginAccountBalance) {
		// need to topup or release margin <-> general
		var amt *num.Uint
		var tp types.TransferType
		if uRequiredPositionMargin.GT(marginAccountBalance) {
			amt = num.UintZero().Sub(uRequiredPositionMargin, marginAccountBalance)
			tp = types.TransferTypeMarginLow
		} else {
			amt = num.UintZero().Sub(uRequiredPositionMargin, marginAccountBalance)
			tp = types.TransferTypeMarginHigh
		}
		riskEvents = append(riskEvents, &marginChange{
			Margin: evt,
			transfer: &types.Transfer{
				Owner:     evt.Party(),
				Type:      tp,
				MinAmount: amt,
				Amount: &types.FinancialAmount{
					Asset:  evt.Asset(),
					Amount: amt.Clone(),
				},
			},
			margins: margin,
		})
	}
	uRequireOrderMargin, _ := num.UintFromDecimal(requireOrderMargin)
	if !uRequireOrderMargin.EQ(orderMarginAccountBalance) {
		// need to topup or release orderMargin <-> general
		var amt *num.Uint
		var tp types.TransferType
		if requireOrderMargin.GreaterThan(orderMarginAccountBalance.ToDecimal()) {
			amt = num.UintZero().Sub(uRequireOrderMargin, orderMarginAccountBalance)
			tp = types.TransferTypeOrderMarginLow
		} else {
			amt = num.UintZero().Sub(orderMarginAccountBalance, uRequireOrderMargin)
			tp = types.TransferTypeOrderMarginLow
		}
		riskEvents = append(riskEvents, &marginChange{
			Margin: evt,
			transfer: &types.Transfer{
				Owner:     evt.Party(),
				Type:      tp,
				MinAmount: amt,
				Amount: &types.FinancialAmount{
					Asset:  evt.Asset(),
					Amount: amt.Clone(),
				},
			},
			margins: margin,
		})
	}
	return riskEvents, nil
}

func (e *Engine) SwitchFromIsolatedMargin(evt events.Margin, marketObservable *num.Uint, inc num.Decimal) events.Risk {
	amt := evt.OrderMarginBalance().Clone()
	auction := e.as.InAuction() && !e.as.CanLeave()
	margins := e.calculateMargins(evt, marketObservable, *e.factors, true, auction, inc)
	return &marginChange{
		Margin: evt,
		transfer: &types.Transfer{
			Owner:     evt.Party(),
			Type:      types.TransferTypeIsolatedMarginLow,
			MinAmount: amt,
			Amount: &types.FinancialAmount{
				Asset:  evt.Asset(),
				Amount: amt.Clone(),
			},
		},
		margins: margins,
	}
}
