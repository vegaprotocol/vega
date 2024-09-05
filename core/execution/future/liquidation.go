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

package future

import (
	"context"
	"time"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
)

func (m *Market) checkNetwork(ctx context.Context, now time.Time) error {
	if m.as.InAuction() {
		return nil
	}
	// this only returns an error if we couldn't get the price range, incidating no orders on book
	order, _ := m.liquidation.OnTick(ctx, now, m.midPrice())
	if order == nil {
		return nil
	}
	// register the network order on the positions engine
	_ = m.position.RegisterOrder(ctx, order)
	order.OriginalPrice, _ = num.UintFromDecimal(order.Price.ToDecimal().Div(m.priceFactor))
	m.broker.Send(events.NewOrderEvent(ctx, order))
	conf, err := m.matching.SubmitOrder(order)
	if err != nil {
		// order failed to uncross, reject and done
		return m.unregisterAndReject(ctx, order, err)
	}
	order.ClearUpExtraRemaining()
	// this should not be possible (network position can't really flip)
	if order.ReduceOnly && order.Remaining > 0 {
		order.Status = types.OrderStatusStopped
	}

	// if the order is not staying in the book, then we remove it
	// from the potential positions
	if order.IsFinished() && order.Remaining > 0 {
		_ = m.position.UnregisterOrder(ctx, order)
	}
	// send the event with the order in its final state
	m.broker.Send(events.NewOrderEvent(ctx, order))

	// no trades...
	if len(conf.Trades) == 0 {
		return nil
	}
	// transfer fees to the good party -> fees are now taken from the insurance pool
	fees, _ := m.fee.GetFeeForPositionResolution(conf.Trades, m.referralDiscountRewardService, m.volumeDiscountService, m.volumeRebateService)
	tresps, err := m.collateral.TransferFees(ctx, m.GetID(), m.settlementAsset, fees)
	if err != nil {
		// we probably should reject the order, although if we end up here we have a massive problem.
		_ = m.position.UnregisterOrder(ctx, order)
		// if we get an eror transfer fees, we are missing accounts, and something is terribly wrong.
		// the fees we get from the fee engine result in transfers with the minimum amount set to 0,
		// so the only thing that could go wrong is missing accounts.
		m.log.Panic("unable to transfer fees for positions resolution",
			logging.Error(err),
			logging.String("market-id", m.GetID()))
		return err
	}
	if len(tresps) > 0 {
		m.broker.Send(events.NewLedgerMovements(ctx, tresps))
	}
	// Now that the fees have been taken care of, get the current last traded price:
	lastTraded := m.getLastTradedPrice()
	tradeType := types.TradeTypeNetworkCloseOutGood
	// now handle the confirmation like you would any other order/trade confirmation
	m.handleConfirmation(ctx, conf, &tradeType)
	// restore the last traded price, the network trades do not count towards the mark price
	// nor do they factor in to the price monitoring logic.
	m.lastTradedPrice = lastTraded
	// update the liquidation engine to reflect the trades have happened
	m.liquidation.UpdateNetworkPosition(conf.Trades)

	// check for reference moves again? We should've already done this
	// This can probably be removed
	m.checkForReferenceMoves(ctx, conf.PassiveOrdersAffected, false)
	return nil
}
