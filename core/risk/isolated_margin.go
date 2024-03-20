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
// For isolated margin search levels and release levels are set to 0.
// auctionPrice is nil if not in an auction, otherwise the max(markPrice, indicativePrice).
// NB: pure calculation, no events emitted, no state changed.
func (e *Engine) calculateIsolatedMargins(m events.Margin, marketObservable *num.Uint, inc num.Decimal, marginFactor num.Decimal, auctionPrice *num.Uint, orders []*types.Order) *types.MarginLevels {
	auction := e.as.InAuction() && !e.as.CanLeave()
	// NB:we don't include orders when calculating margin for isolated margin as they are margined separately!
	margins := e.calculateMargins(m, marketObservable, *e.factors, false, auction, inc)
	margins.OrderMargin = CalcOrderMargins(m.Size(), orders, e.positionFactor, marginFactor, auctionPrice)
	margins.CollateralReleaseLevel = num.UintZero()
	margins.SearchLevel = num.UintZero()
	margins.MarginMode = types.MarginModeIsolatedMargin
	margins.MarginFactor = marginFactor
	margins.Party = m.Party()
	margins.Asset = m.Asset()
	margins.MarketID = m.MarketID()
	margins.Timestamp = e.timeSvc.GetTimeNow().UnixNano()
	return margins
}

// ReleaseExcessMarginAfterAuctionUncrossing is called after auction uncrossing to release excess order margin due to orders placed during an auction
// when the price used for order margin is the auction price rather than the order price.
func (e *Engine) ReleaseExcessMarginAfterAuctionUncrossing(ctx context.Context, m events.Margin, marketObservable *num.Uint, increment num.Decimal, marginFactor num.Decimal, orders []*types.Order) events.Risk {
	margins := e.calculateIsolatedMargins(m, marketObservable, increment, marginFactor, nil, orders)
	e.updateMarginLevels(events.NewMarginLevelsEvent(ctx, *margins))
	if margins.OrderMargin.LT(m.OrderMarginBalance()) {
		amt := num.UintZero().Sub(m.OrderMarginBalance(), margins.OrderMargin)
		return &marginChange{
			Margin: m,
			transfer: &types.Transfer{
				Owner: m.Party(),
				Type:  types.TransferTypeOrderMarginHigh,
				Amount: &types.FinancialAmount{
					Asset:  m.Asset(),
					Amount: amt,
				},
				MinAmount: amt.Clone(),
			},
			margins: margins,
		}
	}
	return nil
}

// UpdateIsolatedMarginOnAggressor is called when a new order comes in and is matched immediately.
// NB: evt has the position after the trades + orders need to include the new order with the updated remaining.
// returns an error if the new margin is invalid or if the margin account cannot be topped up from general account.
// if successful it updates the margin level and returns the transfer that is needed for the topup of the margin account or release from the margin account excess.
func (e *Engine) UpdateIsolatedMarginOnAggressor(ctx context.Context, evt events.Margin, marketObservable *num.Uint, increment num.Decimal, orders []*types.Order, trades []*types.Trade, marginFactor num.Decimal, traderSide types.Side, isAmend bool, fees *num.Uint) ([]events.Risk, error) {
	if evt == nil {
		return nil, nil
	}
	margins := e.calculateIsolatedMargins(evt, marketObservable, increment, marginFactor, nil, orders)
	tradedSize := int64(0)
	side := trades[0].Aggressor
	requiredMargin := num.UintZero()
	for _, t := range trades {
		tradedSize += int64(t.Size)
		requiredMargin.AddSum(num.UintZero().Mul(t.Price, num.NewUint(t.Size)))
	}
	if side == types.SideSell {
		tradedSize = -tradedSize
	}
	oldPosition := evt.Size() - tradedSize
	if evt.Size()*oldPosition >= 0 { // position didn't switch sides
		if int64Abs(oldPosition) < int64Abs(evt.Size()) { // position increased
			requiredMargin, _ = num.UintFromDecimal(requiredMargin.ToDecimal().Div(e.positionFactor).Mul(marginFactor))
			if num.Sum(requiredMargin, evt.MarginBalance()).LT(margins.MaintenanceMargin) {
				return nil, ErrInsufficientFundsForMaintenanceMargin
			}
			if !isAmend && requiredMargin.GT(evt.GeneralAccountBalance()) {
				return nil, ErrInsufficientFundsForMarginInGeneralAccount
			}
			if isAmend && requiredMargin.GT(num.Sum(evt.GeneralAccountBalance(), evt.OrderMarginBalance())) {
				return nil, ErrInsufficientFundsForMarginInGeneralAccount
			}
			// new order, given that they can cover for the trade, do they have enough left to cover the fees?
			if !isAmend && num.Sum(requiredMargin, fees).GT(num.Sum(evt.GeneralAccountBalance(), evt.MarginBalance())) {
				return nil, ErrInsufficientFundsToCoverTradeFees
			}
			// amended order, given that they can cover for the trade, do they have enough left to cover the fees for the amended order's trade?
			if isAmend && num.Sum(requiredMargin, fees).GT(num.Sum(evt.GeneralAccountBalance(), evt.MarginBalance(), evt.OrderMarginBalance())) {
				return nil, ErrInsufficientFundsToCoverTradeFees
			}
		}
	} else {
		// position did switch sides
		requiredMargin = num.UintZero()
		pos := int64Abs(oldPosition)
		totalSize := uint64(0)
		for _, t := range trades {
			if pos >= t.Size {
				pos -= t.Size
				totalSize += t.Size
			} else if pos == 0 {
				requiredMargin.AddSum(num.UintZero().Mul(t.Price, num.NewUint(t.Size)))
			} else {
				size := t.Size - pos
				requiredMargin.AddSum(num.UintZero().Mul(t.Price, num.NewUint(size)))
				pos = 0
			}
		}
		// The new margin required balance is requiredMargin, so we need to check that:
		// 1) it's greater than maintenance margin to keep the invariant
		// 2) there are sufficient funds in what's currently in the margin account + general account to cover for the new required margin
		requiredMargin, _ = num.UintFromDecimal(requiredMargin.ToDecimal().Div(e.positionFactor).Mul(marginFactor))
		if num.Sum(requiredMargin, evt.MarginBalance()).LT(margins.MaintenanceMargin) {
			return nil, ErrInsufficientFundsForMaintenanceMargin
		}
		if requiredMargin.GT(num.Sum(evt.GeneralAccountBalance(), evt.MarginBalance())) {
			return nil, ErrInsufficientFundsForMarginInGeneralAccount
		}
		if num.Sum(requiredMargin, fees).GT(num.Sum(evt.GeneralAccountBalance(), evt.MarginBalance())) {
			return nil, ErrInsufficientFundsToCoverTradeFees
		}
	}

	e.updateMarginLevels(events.NewMarginLevelsEvent(ctx, *margins))
	transfers := getIsolatedMarginTransfersOnPositionChange(evt.Party(), evt.Asset(), trades, traderSide, evt.Size(), e.positionFactor, marginFactor, evt.MarginBalance(), evt.OrderMarginBalance(), marketObservable, true, isAmend)
	if transfers == nil {
		return nil, nil
	}
	ret := []events.Risk{}
	for _, t := range transfers {
		ret = append(ret, &marginChange{
			Margin:   evt,
			transfer: t,
			margins:  margins,
		})
	}
	return ret, nil
}

// UpdateIsolatedMarginOnOrder checks that the party has sufficient cover for the given orders including the new one. It returns an error if the party doesn't have sufficient cover and the necessary transfers otherwise.
// NB: auctionPrice should be nil in continuous mode.
func (e *Engine) UpdateIsolatedMarginOnOrder(ctx context.Context, evt events.Margin, orders []*types.Order, marketObservable *num.Uint, auctionPrice *num.Uint, increment num.Decimal, marginFactor num.Decimal) (events.Risk, error) {
	auction := e.as.InAuction() && !e.as.CanLeave()
	var ap *num.Uint
	if auction {
		ap = auctionPrice
	}
	margins := e.calculateIsolatedMargins(evt, marketObservable, increment, marginFactor, ap, orders)

	// if the margin account balance + the required order margin is less than the maintenance margin, return error
	if margins.OrderMargin.GT(evt.OrderMarginBalance()) && num.UintZero().Sub(margins.OrderMargin, evt.OrderMarginBalance()).GT(evt.GeneralAccountBalance()) {
		return nil, ErrInsufficientFundsForMarginInGeneralAccount
	}

	e.updateMarginLevels(events.NewMarginLevelsEvent(ctx, *margins))
	var amt *num.Uint
	tp := types.TransferTypeOrderMarginLow
	if margins.OrderMargin.GT(evt.OrderMarginBalance()) {
		amt = num.UintZero().Sub(margins.OrderMargin, evt.OrderMarginBalance())
	} else {
		amt = num.UintZero().Sub(evt.OrderMarginBalance(), margins.OrderMargin)
		tp = types.TransferTypeOrderMarginHigh
	}

	var trnsfr *types.Transfer
	if amt.IsZero() {
		return nil, nil
	}

	trnsfr = &types.Transfer{
		Owner: evt.Party(),
		Type:  tp,
		Amount: &types.FinancialAmount{
			Asset:  evt.Asset(),
			Amount: amt,
		},
		MinAmount: amt.Clone(),
	}

	change := &marginChange{
		Margin:   evt,
		transfer: trnsfr,
		margins:  margins,
	}
	return change, nil
}

func (e *Engine) UpdateIsolatedMarginOnOrderCancel(ctx context.Context, evt events.Margin, orders []*types.Order, marketObservable *num.Uint, auctionPrice *num.Uint, increment num.Decimal, marginFactor num.Decimal) (events.Risk, error) {
	auction := e.as.InAuction() && !e.as.CanLeave()
	var ap *num.Uint
	if auction {
		ap = auctionPrice
	}
	margins := e.calculateIsolatedMargins(evt, marketObservable, increment, marginFactor, ap, orders)
	if margins.OrderMargin.GT(evt.OrderMarginBalance()) {
		return nil, ErrInsufficientFundsForOrderMargin
	}

	e.updateMarginLevels(events.NewMarginLevelsEvent(ctx, *margins))
	var amt *num.Uint
	tp := types.TransferTypeOrderMarginHigh
	amt = num.UintZero().Sub(evt.OrderMarginBalance(), margins.OrderMargin)

	var trnsfr *types.Transfer
	if amt.IsZero() {
		return nil, nil
	}

	trnsfr = &types.Transfer{
		Owner: evt.Party(),
		Type:  tp,
		Amount: &types.FinancialAmount{
			Asset:  evt.Asset(),
			Amount: amt,
		},
		MinAmount: amt.Clone(),
	}

	change := &marginChange{
		Margin:   evt,
		transfer: trnsfr,
		margins:  margins,
	}
	return change, nil
}

// UpdateIsolatedMarginOnPositionChanged is called upon changes to the position of a party in isolated margin mode.
// Depending on the nature of the change it checks if it needs to move funds into our out of the margin account from the
// order margin account or to the general account.
// At this point we don't enforce any invariants just calculate transfers.
func (e *Engine) UpdateIsolatedMarginsOnPositionChange(ctx context.Context, evt events.Margin, marketObservable *num.Uint, increment num.Decimal, orders []*types.Order, trades []*types.Trade, traderSide types.Side, marginFactor num.Decimal) ([]events.Risk, error) {
	if evt == nil {
		return nil, nil
	}
	margins := e.calculateIsolatedMargins(evt, marketObservable, increment, marginFactor, nil, orders)
	ret := []events.Risk{}
	transfer := getIsolatedMarginTransfersOnPositionChange(evt.Party(), evt.Asset(), trades, traderSide, evt.Size(), e.positionFactor, marginFactor, evt.MarginBalance(), evt.OrderMarginBalance(), marketObservable, false, false)
	e.updateMarginLevels(events.NewMarginLevelsEvent(ctx, *margins))
	if transfer != nil {
		ret = append(ret, &marginChange{
			Margin:   evt,
			transfer: transfer[0],
			margins:  margins,
		})
	}
	var amtForRelease *num.Uint
	if !evt.OrderMarginBalance().IsZero() && margins.OrderMargin.IsZero() && transfer != nil && evt.OrderMarginBalance().GT(transfer[0].Amount.Amount) {
		amtForRelease = num.UintZero().Sub(evt.OrderMarginBalance(), transfer[0].Amount.Amount)
	}

	// if there's no more order margin requirement, release remaining order margin
	if amtForRelease != nil && !amtForRelease.IsZero() {
		ret = append(ret,
			&marginChange{
				Margin: evt,
				transfer: &types.Transfer{
					Owner: evt.Party(),
					Type:  types.TransferTypeOrderMarginHigh,
					Amount: &types.FinancialAmount{
						Asset:  evt.Asset(),
						Amount: amtForRelease,
					},
					MinAmount: amtForRelease.Clone(),
				},
				margins: margins,
			},
		)
	}
	return ret, nil
}

func (e *Engine) CheckMarginInvariants(ctx context.Context, evt events.Margin, marketObservable *num.Uint, increment num.Decimal, orders []*types.Order, marginFactor num.Decimal) (events.Risk, error) {
	margins := e.calculateIsolatedMargins(evt, marketObservable, increment, marginFactor, nil, orders)
	e.updateMarginLevels(events.NewMarginLevelsEvent(ctx, *margins))
	return e.checkMarginInvariants(evt, margins)
}

// CheckMarginInvariants returns an error if the margin invariants are invalidated, i.e. if margin balance < margin level or order margin balance < order margin level.
func (e *Engine) checkMarginInvariants(evt events.Margin, margins *types.MarginLevels) (events.Risk, error) {
	ret := &marginChange{
		Margin:   evt,
		transfer: nil,
		margins:  margins,
	}
	if evt.MarginBalance().LT(margins.MaintenanceMargin) {
		return ret, ErrInsufficientFundsForMaintenanceMargin
	}
	if evt.OrderMarginBalance().LT(margins.OrderMargin) {
		return ret, ErrInsufficientFundsForOrderMargin
	}
	return ret, nil
}

// SwitchToIsolatedMargin attempts to switch the party from cross margin mode to isolated mode.
// Error can be returned if it is not possible for the party to switch at this moment.
// If successful the new margin levels are buffered and the required margin level, margin balances, and transfers (aka events.risk) is returned.
func (e *Engine) SwitchToIsolatedMargin(ctx context.Context, evt events.Margin, marketObservable *num.Uint, inc num.Decimal, orders []*types.Order, marginFactor num.Decimal, auctionPrice *num.Uint) ([]events.Risk, error) {
	margins := e.calculateIsolatedMargins(evt, marketObservable, inc, marginFactor, auctionPrice, orders)
	risk, err := switchToIsolatedMargin(evt, margins, orders, marginFactor, e.positionFactor)
	if err != nil {
		return nil, err
	}

	e.updateMarginLevels(events.NewMarginLevelsEvent(ctx, *margins))
	return risk, nil
}

// SwitchFromIsolatedMargin switches the party from isolated margin mode to cross margin mode.
// This includes:
// 1. recalcualtion of the required margin in cross margin mode + margin levels are buffered
// 2. return a transfer of all the balance from order margin account to margin account
// NB: cannot fail.
func (e *Engine) SwitchFromIsolatedMargin(ctx context.Context, evt events.Margin, marketObservable *num.Uint, inc num.Decimal) events.Risk {
	amt := evt.OrderMarginBalance().Clone()
	auction := e.as.InAuction() && !e.as.CanLeave()
	margins := e.calculateMargins(evt, marketObservable, *e.factors, true, auction, inc)
	margins.Party = evt.Party()
	margins.Asset = evt.Asset()
	margins.MarketID = evt.MarketID()
	margins.Timestamp = e.timeSvc.GetTimeNow().UnixNano()
	e.updateMarginLevels(events.NewMarginLevelsEvent(ctx, *margins))

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

// getIsolatedMarginTransfersOnPositionChange returns the transfers that need to be made to/from the margin account in isolated margin mode
// when the position changes. This handles the 3 different cases of position change (increase, decrease, switch sides).
// NB: positionSize is *after* the trades.
func getIsolatedMarginTransfersOnPositionChange(party, asset string, trades []*types.Trade, traderSide types.Side, positionSize int64, positionFactor, marginFactor num.Decimal, curMarginBalance, orderMarginBalance, markPrice *num.Uint, aggressiveSide bool, isAmend bool) []*types.Transfer {
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
			if !isAmend {
				// need to top up the margin account from the order margin account
				var tp types.TransferType
				if aggressiveSide {
					tp = types.TransferTypeMarginLow
				} else {
					tp = types.TransferTypeIsolatedMarginLow
				}
				return []*types.Transfer{{
					Owner: party,
					Type:  tp,
					Amount: &types.FinancialAmount{
						Asset:  asset,
						Amount: marginToAdd,
					},
					MinAmount: marginToAdd,
				}}
			}
			if marginToAdd.LTE(orderMarginBalance) {
				return []*types.Transfer{{
					Owner: party,
					Type:  types.TransferTypeIsolatedMarginLow,
					Amount: &types.FinancialAmount{
						Asset:  asset,
						Amount: marginToAdd,
					},
					MinAmount: marginToAdd,
				}}
			}
			generalTopUp := num.UintZero().Sub(marginToAdd, orderMarginBalance)
			return []*types.Transfer{
				{
					Owner: party,
					Type:  types.TransferTypeIsolatedMarginLow,
					Amount: &types.FinancialAmount{
						Asset:  asset,
						Amount: orderMarginBalance,
					},
					MinAmount: orderMarginBalance,
				}, {
					Owner: party,
					Type:  types.TransferTypeMarginLow,
					Amount: &types.FinancialAmount{
						Asset:  asset,
						Amount: generalTopUp,
					},
					MinAmount: generalTopUp,
				},
			}
		}
		// position decreased
		// marginToRelease = balanceBefore + positionBefore x (newTradeVWAP - markPrice) x |totalTradeSize|/|positionBefore|
		theoreticalAccountBalance, _ := num.UintFromDecimal(vwap.ToDecimal().Sub(markPrice.ToDecimal()).Mul(num.DecimalFromInt64(int64(int64Abs(oldPosition)))).Div(positionFactor).Add(curMarginBalance.ToDecimal()))
		marginToRelease := num.UintZero().Div(num.UintZero().Mul(theoreticalAccountBalance, num.NewUint(int64Abs(positionDelta))), num.NewUint(int64Abs(oldPosition)))
		// need to top up the margin account
		return []*types.Transfer{{
			Owner: party,
			Type:  types.TransferTypeMarginHigh,
			Amount: &types.FinancialAmount{
				Asset:  asset,
				Amount: marginToRelease,
			},
			MinAmount: marginToRelease,
		}}
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
	if aggressiveSide {
		tp = types.TransferTypeMarginLow
	}
	if !release.IsZero() {
		amt = release
		tp = types.TransferTypeMarginHigh
	}

	if amt.IsZero() {
		return nil
	}

	return []*types.Transfer{{
		Owner: party,
		Type:  tp,
		Amount: &types.FinancialAmount{
			Asset:  asset,
			Amount: amt,
		},
		MinAmount: amt,
	}}
}

func (e *Engine) CalcOrderMarginsForClosedOutParty(orders []*types.Order, marginFactor num.Decimal) *num.Uint {
	return CalcOrderMargins(0, orders, e.positionFactor, marginFactor, nil)
}

// CalcOrderMargins calculates the the order margin required for the party given their current orders and margin factor.
func CalcOrderMargins(positionSize int64, orders []*types.Order, positionFactor, marginFactor num.Decimal, auctionPrice *num.Uint) *num.Uint {
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
		if o.Status != types.OrderStatusActive || o.PeggedOrder != nil {
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

// switching from cross margin to isolated margin or changing margin factor
//  1. For any active position, calculate average entry price * abs(position) * margin factor.
//     Calculate the amount of funds which will be added to, or subtracted from, the general account in order to do this.
//     If additional funds must be added which are not available, reject the transaction immediately.
//  2. For any active orders, calculate the quantity limit price * remaining size * margin factor which needs to be placed
//     in the order margin account. Add this amount to the difference calculated in step 1.
//     If this amount is less than or equal to the amount in the general account,
//     perform the transfers (first move funds into/out of margin account, then move funds into the order margin account).
//     If there are insufficient funds, reject the transaction.
//  3. Move account to isolated margin mode on this market
//
// If a party has no position nore orders and switches to isolated margin the function returns an empty slice.
func switchToIsolatedMargin(evt events.Margin, margin *types.MarginLevels, orders []*types.Order, marginFactor, positionFactor num.Decimal) ([]events.Risk, error) {
	marginAccountBalance := evt.MarginBalance()
	generalAccountBalance := evt.GeneralAccountBalance()
	orderMarginAccountBalance := evt.OrderMarginBalance()
	if orderMarginAccountBalance == nil {
		orderMarginAccountBalance = num.UintZero()
	}
	totalOrderNotional := num.UintZero()
	for _, o := range orders {
		if o.Status == types.OrderStatusActive && o.PeggedOrder == nil {
			totalOrderNotional = totalOrderNotional.AddSum(num.UintZero().Mul(o.Price, num.NewUint(o.TrueRemaining())))
		}
	}

	positionSize := int64Abs(evt.Size())
	requiredPositionMargin := num.UintZero().Mul(evt.AverageEntryPrice(), num.NewUint(positionSize)).ToDecimal().Mul(marginFactor).Div(positionFactor)
	requireOrderMargin := totalOrderNotional.ToDecimal().Mul(marginFactor).Div(positionFactor)

	// check that we have enough in the general account for any top up needed, i.e.
	// topupNeeded = requiredPositionMargin + requireOrderMargin - marginAccountBalance
	// if topupNeeded > generalAccountBalance => fail
	if requiredPositionMargin.Add(requireOrderMargin).Sub(marginAccountBalance.ToDecimal()).Sub(orderMarginAccountBalance.ToDecimal()).GreaterThan(generalAccountBalance.ToDecimal()) {
		return nil, fmt.Errorf("insufficient balance in general account to cover for required order margin")
	}

	// average entry price * current position * new margin factor (aka requiredPositionMargin) must be above the initial margin for the current position or the transaction will be rejected
	if !requiredPositionMargin.IsZero() && !requiredPositionMargin.GreaterThan(margin.InitialMargin.ToDecimal()) {
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
			amt = num.UintZero().Sub(marginAccountBalance, uRequiredPositionMargin)
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
			tp = types.TransferTypeOrderMarginHigh
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

// int64Abs returns the absolute uint64 value of the given int64 n.
func int64Abs(n int64) uint64 {
	if n < 0 {
		return uint64(-n)
	}
	return uint64(n)
}
