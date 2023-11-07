package future

import (
	"context"
	"fmt"
	"time"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
)

func (m *Market) checkNetwork(ctx context.Context, now time.Time) error {
	// debug
	// this only returns an error if we couldn't get the price range, incidating no orders on book
	order, _ := m.liquidation.OnTick(ctx, now)
	if order == nil {
		return nil
	}
	fmt.Printf(">>> DEBUG ORDER: %#v\n", order)
	order.OriginalPrice = num.UintZero().Div(order.Price, m.priceFactor)
	conf, err := m.matching.SubmitOrder(order)
	if err != nil {
		m.log.Panic("Failure after submitting order to matching engine",
			logging.Order(order),
			logging.Error(err),
		)
	}
	// no order was submitted
	if conf == nil {
		return nil
	}
	m.broker.Send(events.NewOrderEvent(ctx, order))
	m.handleConfirmationPassiveOrders(ctx, conf)

	// no trades...
	if len(conf.Trades) == 0 {
		m.checkForReferenceMoves(ctx, conf.PassiveOrdersAffected, false)
		return nil
	}
	// transfer fees to the good party -> fees are now taken from the insurance pool
	fees, _ := m.fee.GetFeeForPositionResolution(conf.Trades)
	tresps, err := m.collateral.TransferFees(ctx, m.GetID(), m.settlementAsset, fees)
	if err != nil {
		// this is the current behaviour, even if this fails, we need to check for reference moves
		// FIXME(): we may figure a better error handling in here
		m.checkForReferenceMoves(ctx, conf.PassiveOrdersAffected, false)
		m.log.Error("unable to transfer fees for positions resolutions",
			logging.Error(err),
			logging.String("market-id", m.GetID()))
		return err
	}
	if len(tresps) > 0 {
		m.broker.Send(events.NewLedgerMovements(ctx, tresps))
	}
	// update the liquidation engine to reflect the trades have happened
	m.liquidation.UpdateNetworkPosition(conf.Trades)

	// now update the network position, finish up the trades, update positions, etc...
	// essentially, handle trades
	// Insert all trades resulted from the executed order
	tradeEvts := make([]events.Event, 0, len(conf.Trades))
	// get total traded volume
	tradedValue, _ := num.UintFromDecimal(
		conf.TradedValue().ToDecimal().Div(m.positionFactor))
	for idx, trade := range conf.Trades {
		trade.SetIDs(m.idgen.NextID(), conf.Order, conf.PassiveOrdersAffected[idx])

		// setup the type of the trade to network
		// this trade did happen with a GOOD trader to
		// 0 out the BAD trader position
		trade.Type = types.TradeTypeNetworkCloseOutGood
		tradeEvts = append(tradeEvts, events.NewTradeEvent(ctx, *trade))

		// Update positions - this is a special trade involving the network as party
		// so rather than checking this every time we call Update, call special UpdateNetwork
		m.position.UpdateNetwork(ctx, trade, conf.PassiveOrdersAffected[idx])
		// record the updated passive side's position
		partyPos, _ := m.position.GetPositionByPartyID(conf.PassiveOrdersAffected[idx].Party)
		m.marketActivityTracker.RecordPosition(m.settlementAsset, conf.PassiveOrdersAffected[idx].Party, m.mkt.ID, partyPos.Size(), trade.Price, m.positionFactor, m.timeService.GetTimeNow())

		if err := m.tsCalc.RecordOpenInterest(m.position.GetOpenInterest(), now); err != nil {
			m.log.Debug("unable record open interest",
				logging.String("market-id", m.GetID()),
				logging.Error(err))
		}

		m.settlement.AddTrade(trade)
	}
	m.feeSplitter.AddTradeValue(tradedValue)
	m.marketActivityTracker.AddValueTraded(m.settlementAsset, m.mkt.ID, tradedValue)
	if len(tradeEvts) > 0 {
		m.broker.SendBatch(tradeEvts)
	}
	// perform a MTM settlement after the trades
	m.confirmMTM(ctx, false)
	// check for reference moves
	m.checkForReferenceMoves(ctx, conf.PassiveOrdersAffected, false)
	return nil
}
