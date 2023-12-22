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

package settlement

import (
	"context"
	"sort"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/metrics"
	"code.vegaprotocol.io/vega/core/products"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
)

// MarketPosition ...
//
//go:generate go run github.com/golang/mock/mockgen -destination mocks/market_position_mock.go -package mocks code.vegaprotocol.io/vega/core/settlement MarketPosition
type MarketPosition interface {
	Party() string
	Size() int64
	Buy() int64
	Sell() int64
	Price() *num.Uint
	BuySumProduct() *num.Uint
	SellSumProduct() *num.Uint
	VWBuy() *num.Uint
	VWSell() *num.Uint
	ClearPotentials()
	AverageEntryPrice() *num.Uint
}

// Product ...
//
//go:generate go run github.com/golang/mock/mockgen -destination mocks/settlement_product_mock.go -package mocks code.vegaprotocol.io/vega/core/settlement Product
type Product interface {
	Settle(*num.Uint, *num.Uint, num.Decimal) (*types.FinancialAmount, bool, num.Decimal, error)
	GetAsset() string
}

// TimeService.
//
//go:generate go run github.com/golang/mock/mockgen -destination mocks/time_service_mock.go -package mocks code.vegaprotocol.io/vega/core/settlement TimeService
type TimeService interface {
	GetTimeNow() time.Time
}

// Broker - the event bus broker, send events here.
type Broker interface {
	Send(event events.Event)
	SendBatch(events []events.Event)
}

// Engine - the main type (of course).
type Engine struct {
	Config
	log *logging.Logger

	market          string
	product         Product
	settledPosition map[string]int64 // party -> last mark-to-market position
	mu              *sync.Mutex
	trades          map[string][]*settlementTrade
	timeService     TimeService
	broker          Broker
	positionFactor  num.Decimal
	lastMarkPrice   *num.Uint // price at last mark to market
}

// New instantiates a new instance of the settlement engine.
func New(log *logging.Logger, conf Config, product Product, market string, timeService TimeService, broker Broker, positionFactor num.Decimal) *Engine {
	// setup logger
	log = log.Named(namedLogger)
	log.SetLevel(conf.Level.Get())

	return &Engine{
		Config:          conf,
		log:             log,
		market:          market,
		product:         product,
		settledPosition: map[string]int64{},
		mu:              &sync.Mutex{},
		trades:          map[string][]*settlementTrade{},
		timeService:     timeService,
		broker:          broker,
		positionFactor:  positionFactor,
	}
}

func (e *Engine) UpdateProduct(product products.Product) {
	e.product = product
}

// ReloadConf update the internal configuration of the settlement engined.
func (e *Engine) ReloadConf(cfg Config) {
	e.log.Info("reloading configuration")
	if e.log.GetLevel() != cfg.Level.Get() {
		e.log.Info("updating log level",
			logging.String("old", e.log.GetLevel().String()),
			logging.String("new", cfg.Level.String()),
		)
		e.log.SetLevel(cfg.Level.Get())
	}

	e.Config = cfg
}

// Update merely adds positions to the settlement engine, and won't be useful for a MTM settlement
// this function is mainly used for testing, and should be used with extreme caution as a result
// perhaps the tests should be refactored to use the Settle call to create positions first.
func (e *Engine) Update(positions []events.MarketPosition) {
	e.mu.Lock()
	for _, p := range positions {
		party := p.Party()
		e.settledPosition[party] = p.Size()
		e.lastMarkPrice = p.Price()
	}
	e.mu.Unlock()
}

// Settle run settlement over all the positions.
func (e *Engine) Settle(t time.Time, settlementData *num.Uint) ([]*types.Transfer, *num.Uint, error) {
	e.log.Debugf("Settling market, closed at %s", t.Format(time.RFC3339))
	positions, round, err := e.settleAll(settlementData)
	if err != nil {
		e.log.Error(
			"Something went wrong trying to settle positions",
			logging.Error(err),
		)
		return nil, nil, err
	}
	return positions, round, nil
}

// AddTrade - this call is required to get the correct MTM settlement values
// each change in position has to be calculated using the exact price of the trade.
func (e *Engine) AddTrade(trade *types.Trade) {
	e.mu.Lock()
	defer e.mu.Unlock()
	// network registers a wash trade to update its position
	var buyerSize, sellerSize int64
	// checking the len of cd shouldn't be required here, but it is needed in the second if
	// in case the buyer and seller are one and the same...
	if cd, ok := e.trades[trade.Buyer]; !ok || len(cd) == 0 {
		e.trades[trade.Buyer] = []*settlementTrade{}
		// check if the buyer already has a known position
		if pos, ok := e.settledPosition[trade.Buyer]; ok {
			buyerSize = pos
		}
	} else {
		buyerSize = cd[len(cd)-1].newSize
	}
	if cd, ok := e.trades[trade.Seller]; !ok || len(cd) == 0 {
		e.trades[trade.Seller] = []*settlementTrade{}
		// check if seller has a known position
		if pos, ok := e.settledPosition[trade.Seller]; ok {
			sellerSize = pos
		}
	} else {
		sellerSize = cd[len(cd)-1].newSize
	}
	size := int64(trade.Size)
	// the parties both need to get a MTM settlement on the traded volume
	// and this MTM part has to be based on the _actual_ trade value
	price := trade.Price.Clone()
	e.trades[trade.Buyer] = append(e.trades[trade.Buyer], &settlementTrade{
		price:       price,
		marketPrice: trade.MarketPrice,
		size:        size,
		newSize:     buyerSize + size,
	})
	e.trades[trade.Seller] = append(e.trades[trade.Seller], &settlementTrade{
		price:       price.Clone(),
		marketPrice: trade.MarketPrice,
		size:        -size,
		newSize:     sellerSize - size,
	})
}

func (e *Engine) HasTraded() bool {
	return len(e.trades) > 0
}

func (e *Engine) getFundingTransfer(mtmShare *num.Uint, neg bool, mpos events.MarketPosition, owner string) (*mtmTransfer, bool) {
	tf := e.getMtmTransfer(mtmShare, neg, mpos, owner)
	if tf.transfer == nil {
		tf.transfer = &types.Transfer{
			Type:  types.TransferTypePerpFundingWin,
			Owner: owner,
			Amount: &types.FinancialAmount{
				Amount: mtmShare,
				Asset:  e.product.GetAsset(),
			},
		}
		return tf, false
	}
	if tf.transfer.Type == types.TransferTypeMTMLoss {
		tf.transfer.Type = types.TransferTypePerpFundingLoss
	} else {
		tf.transfer.Type = types.TransferTypePerpFundingWin
	}
	return tf, true
}

func (e *Engine) getMtmTransfer(mtmShare *num.Uint, neg bool, mpos events.MarketPosition, owner string) *mtmTransfer {
	if mtmShare.IsZero() {
		return &mtmTransfer{
			MarketPosition: mpos,
			transfer:       nil,
		}
	}
	typ := types.TransferTypeMTMWin
	if neg {
		typ = types.TransferTypeMTMLoss
	}
	return &mtmTransfer{
		MarketPosition: mpos,
		transfer: &types.Transfer{
			Type:  typ,
			Owner: owner,
			Amount: &types.FinancialAmount{
				Amount: mtmShare,
				Asset:  e.product.GetAsset(),
			},
		},
	}
}

func (e *Engine) winSocialisationUpdate(transfer *mtmTransfer, amt *num.Uint) {
	if amt.IsZero() {
		return
	}
	if transfer.transfer == nil {
		transfer.transfer = &types.Transfer{
			Type:  types.TransferTypeMTMWin,
			Owner: transfer.Party(),
			Amount: &types.FinancialAmount{
				Amount: num.UintZero(),
				Asset:  e.product.GetAsset(),
			},
		}
	}
	transfer.transfer.Amount.Amount.AddSum(amt)
}

func (e *Engine) SettleMTM(ctx context.Context, markPrice *num.Uint, positions []events.MarketPosition) []events.Transfer {
	timer := metrics.NewTimeCounter("-", "settlement", "SettleOrder")
	defer func() { e.lastMarkPrice = markPrice.Clone() }()
	e.mu.Lock()
	tCap := e.transferCap(positions)
	transfers := make([]events.Transfer, 0, tCap)
	// roughly half of the transfers should be wins, half losses
	wins := make([]events.Transfer, 0, tCap/2)
	trades := e.trades
	e.trades = map[string][]*settlementTrade{} // remove here, once we've processed it all here, we're done
	evts := make([]events.Event, 0, len(positions))
	var (
		largestShare  *mtmTransfer       // pointer to whomever gets the last remaining amount from the loss
		zeroShares    = []*mtmTransfer{} // all zero shares for equal distribution if possible
		zeroAmts      = false
		mtmDec        = num.NewDecimalFromFloat(0)
		lossTotal     = num.UintZero()
		winTotal      = num.UintZero()
		lossTotalDec  = num.NewDecimalFromFloat(0)
		winTotalDec   = num.NewDecimalFromFloat(0)
		appendLargest = false
	)

	// network is treated as a regular party
	for _, evt := range positions {
		party := evt.Party()
		current, lastSettledPrice := e.getOrCreateCurrentPosition(party, evt.Size())
		traded, hasTraded := trades[party]
		tradeset := make([]events.TradeSettlement, 0, len(traded))
		// empty position
		skip := current == 0 && lastSettledPrice.IsZero() && evt.Buy() == 0 && evt.Sell() == 0
		for _, t := range traded {
			tradeset = append(tradeset, t)
		}
		// create (and add position to buffer)
		evts = append(evts, events.NewSettlePositionEvent(ctx, party, e.market, evt.Price(), tradeset, e.timeService.GetTimeNow().UnixNano(), e.positionFactor))
		// no changes in position, and the MTM price hasn't changed, we don't need to do anything
		// or an empty position that isn't the result of the party closing itself out
		if !hasTraded && (lastSettledPrice.EQ(markPrice) || skip) {
			// no changes in position and markPrice hasn't changed -> nothing needs to be marked
			continue
		}
		// calculate MTM value, we need the signed mark-price, the OLD open position/volume
		// the new position is either the same, or accounted for by the traded var (added trades)
		// and the old mark price at which the party held the position
		// the trades slice contains all trade positions (position changes for the party)
		// at their exact trade price, so we can MTM that volume correctly, too
		mtmShare, mtmDShare, neg := calcMTM(markPrice, lastSettledPrice, current, traded, e.positionFactor)
		// we've marked this party to market, their position can now reflect this
		e.settledPosition[party] = evt.Size()
		// we don't want to accidentally MTM a party who closed out completely when they open
		// a new position at a later point, so remove if size == 0
		if evt.Size() == 0 && evt.Buy() == 0 && evt.Sell() == 0 {
			// broke this up into its own func for symmetry
			e.rmPosition(party)
		}

		// there's still a subset of potential-only positions, their MTM will be zero
		// but they don't hold an open position, and are excluded from win-socialisation.
		skip = !hasTraded && evt.Size() == 0
		posEvent := newPos(evt, markPrice)
		mtmTransfer := e.getMtmTransfer(mtmShare.Clone(), neg, posEvent, party)

		if !neg {
			wins = append(wins, mtmTransfer)
			winTotal.AddSum(mtmShare)
			winTotalDec = winTotalDec.Add(mtmDShare)
			if !skip && mtmShare.IsZero() {
				zeroShares = append(zeroShares, mtmTransfer)
				zeroAmts = true
			}
			if mtmDShare.GreaterThan(mtmDec) {
				mtmDec = mtmDShare
				largestShare = mtmTransfer
			}
		} else if mtmShare.IsZero() {
			// zero value loss
			wins = append(wins, mtmTransfer)
			lossTotalDec = lossTotalDec.Add(mtmDShare)
		} else {
			transfers = append(transfers, mtmTransfer)
			lossTotal.AddSum(mtmShare)
			lossTotalDec = lossTotalDec.Add(mtmDShare)
		}
	}
	// no need for this lock anymore
	e.mu.Unlock()
	delta := num.UintZero().Sub(lossTotal, winTotal)
	// make sure largests share is never nil
	if largestShare == nil {
		largestShare = &mtmTransfer{
			MarketPosition: &npos{
				price: markPrice.Clone(),
			},
		}
		appendLargest = true
	}
	if !delta.IsZero() {
		if zeroAmts {
			if appendLargest {
				zeroShares = append(zeroShares, largestShare)
			}
			zRound := num.DecimalFromInt64(int64(len(zeroShares)))
			// there are more transfers from losses than we pay out to wins, but some winning parties have zero transfers
			// this delta should == combined win decimals, let's sanity check this!
			if winTotalDec.LessThan(lossTotalDec) && winTotalDec.LessThan(lossTotalDec.Sub(zRound)) {
				e.log.Panic("There's less MTM wins than losses, even accounting for decimals",
					logging.Decimal("total loss", lossTotalDec),
					logging.Decimal("total wins", winTotalDec),
				)
			}
			// parties with a zero win transfer should get AT MOST a transfer of value 1
			// any remainder after that should go to the largest win share, unless we only have parties
			// with a win share of 0. that shouldn't be possible however, and so we can ignore that case
			// should this happen at any point, the collateral engine will panic on settlement balance > 0
			// which is the correct behaviour

			// start distributing the delta
			one := num.NewUint(1)
			for _, transfer := range zeroShares {
				e.winSocialisationUpdate(transfer, one)
				delta.Sub(delta, one)
				if delta.IsZero() {
					break // all done
				}
			}
		}
		// delta is whatever amount the largest share win party gets, this shouldn't be too much
		// delta can be zero at this stage, which is fine
		e.winSocialisationUpdate(largestShare, delta)
	}
	// append wins after loss transfers
	transfers = append(transfers, wins...)
	if len(transfers) > 0 && appendLargest && largestShare.transfer != nil {
		transfers = append(transfers, largestShare)
	}
	if len(evts) > 0 {
		e.broker.SendBatch(evts)
	}
	timer.EngineTimeCounterAdd()
	return transfers
}

// RemoveDistressed - remove whatever settlement data we have for distressed parties
// they are being closed out, and shouldn't be part of any MTM settlement or closing settlement.
func (e *Engine) RemoveDistressed(ctx context.Context, evts []events.Margin) {
	devts := make([]events.Event, 0, len(evts))
	e.mu.Lock()
	for _, v := range evts {
		key := v.Party()
		margin := num.Sum(v.MarginBalance(), v.GeneralBalance())
		devts = append(devts, events.NewSettleDistressed(ctx, key, e.market, v.Price(), margin, e.timeService.GetTimeNow().UnixNano()))
		// @TODO check network trades, resolve to drop from MTM settlement
		delete(e.settledPosition, key)
		delete(e.trades, key)
	}
	e.mu.Unlock()
	e.broker.SendBatch(devts)
}

// simplified settle call.
func (e *Engine) settleAll(settlementData *num.Uint) ([]*types.Transfer, *num.Uint, error) {
	e.mu.Lock()

	// there should be as many positions as there are parties (obviously)
	aggregated := make([]*types.Transfer, 0, len(e.settledPosition))
	// parties who are in profit should be appended (collect first).
	// The split won't always be 50-50, but it's a reasonable approximation
	owed := make([]*types.Transfer, 0, len(e.settledPosition)/2)
	// ensure we iterate over the positions in the same way by getting all the parties (keys)
	// and sort them
	keys := make([]string, 0, len(e.settledPosition))
	for p := range e.settledPosition {
		keys = append(keys, p)
	}
	sort.Strings(keys)
	var delta num.Decimal
	for _, party := range keys {
		pos := e.settledPosition[party]
		// this is possible now, with the Mark to Market stuff, it's possible we've settled any and all positions for a given party
		if pos == 0 {
			continue
		}
		e.log.Debug("Settling position for party", logging.String("party-id", party))
		// @TODO - there was something here... the final amount had to be oracle - market or something
		amt, neg, rem, err := e.product.Settle(e.lastMarkPrice, settlementData.Clone(), num.DecimalFromInt64(pos).Div(e.positionFactor))
		// for now, product.Settle returns the total value, we need to only settle the delta between a parties current position
		// and the final price coming from the oracle, so oracle_price - mark_price * volume (check with Tamlyn whether this should be absolute or not)
		if err != nil {
			e.log.Error(
				"Failed to settle position for party",
				logging.String("party-id", party),
				logging.Error(err),
			)
			e.mu.Unlock()
			return nil, nil, err
		}
		settlePos := &types.Transfer{
			Owner:  party,
			Amount: amt,
		}
		e.log.Debug(
			"Settled position for party",
			logging.String("party-id", party),
			logging.String("amount", amt.Amount.String()),
		)

		if neg { // this is a loss transfer
			settlePos.Type = types.TransferTypeLoss
			aggregated = append(aggregated, settlePos)
			// truncated loss amount will not be transferred to the settlement balance
			// so remove it from the total delta (aka rounding)
			delta = delta.Sub(rem)
		} else { // this is a win transfer
			settlePos.Type = types.TransferTypeWin
			owed = append(owed, settlePos)
			// Truncated win transfer won't be withdrawn from the settlement balance
			// so add it to the total delta (aka rounding)
			delta = delta.Add(rem)
		}
	}
	// we only care about the int part
	round := num.UintZero()
	// if delta > 0, the settlement account will have a non-zero balance at the end
	if !delta.IsNegative() {
		round, _ = num.UintFromDecimal(delta)
	}
	// append the parties in profit to the end
	aggregated = append(aggregated, owed...)
	e.mu.Unlock()
	return aggregated, round, nil
}

func (e *Engine) getOrCreateCurrentPosition(party string, size int64) (int64, *num.Uint) {
	p, ok := e.settledPosition[party]
	if !ok {
		e.settledPosition[party] = size
		return 0, num.UintZero()
	}
	return p, e.lastMarkPrice
}

func (e *Engine) HasPosition(party string) bool {
	_, okPos := e.settledPosition[party]
	_, okTrades := e.trades[party]

	return okPos && okTrades
}

func (e *Engine) rmPosition(party string) {
	delete(e.settledPosition, party)
}

// just get the max len as cap.
func (e *Engine) transferCap(evts []events.MarketPosition) int {
	curLen, evtLen := len(e.settledPosition), len(evts)
	if curLen >= evtLen {
		return curLen
	}
	return evtLen
}

// party.PREV_OPEN_VOLUME * (product.value(current_price) - product.value(prev_mark_price)) + SUM(from i=1 to new_trades.length)( new_trade(i).volume(party) * (product.value(current_price) - new_trade(i).price ) )
// the sum bit is a worry, we do not have all the trades available at this point...

// calcMTM only handles futures ATM. The formula is simple:
// amount =  prev_vol * (current_price - prev_mark_price) + SUM(new_trade := range trades)( new_trade(i).volume(party)*(current_price - new_trade(i).price )
// given that the new trades price will equal new mark price,  the sum(trades) bit will probably == 0 for nicenet
// the size here is the _new_ position size, the price is the OLD price!!
func calcMTM(markPrice, price *num.Uint, size int64, trades []*settlementTrade, positionFactor num.Decimal) (*num.Uint, num.Decimal, bool) {
	delta, sign := num.UintZero().Delta(markPrice, price)
	// this shouldn't be possible I don't think, but just in case
	if size < 0 {
		size = -size
		// swap sign
		sign = !sign
	}
	mtmShare := delta.Mul(delta, num.NewUint(uint64(size)))
	for _, c := range trades {
		delta, neg := num.UintZero().Delta(markPrice, c.price)
		size := num.NewUint(uint64(c.size))
		if c.size < 0 {
			size = size.SetUint64(uint64(-c.size))
			neg = !neg
		}
		add := delta.Mul(delta, size)
		if mtmShare.IsZero() {
			mtmShare.Set(add)
			sign = neg
		} else if neg == sign {
			// both mtmShare and add are the same sign
			mtmShare = mtmShare.Add(mtmShare, add)
		} else if mtmShare.GTE(add) {
			// regardless of sign, we just have to subtract
			mtmShare = mtmShare.Sub(mtmShare, add)
		} else {
			// add > mtmShare, we don't care about signs here
			// just subtract mtmShare and switch signs
			mtmShare = add.Sub(add, mtmShare)
			sign = neg
		}
	}

	// as mtmShare was calculated with the volumes as integers (not decimals in pdp space) we need to divide by position factor
	decShare := mtmShare.ToDecimal().Div(positionFactor)
	res, _ := num.UintFromDecimal(decShare)
	return res, decShare, sign
}

// SettleFundingPeriod takes positions and a funding-payement and returns a slice of transfers.
// returns the slice of transfers to perform, and the max remainder on the settlement account due to rounding issues.
func (e *Engine) SettleFundingPeriod(ctx context.Context, positions []events.MarketPosition, fundingPayment *num.Int) ([]events.Transfer, *num.Uint) {
	if fundingPayment.IsZero() || len(positions) == 0 {
		// nothing to do here
		return nil, nil
	}

	// colletral engine expects all the losses before the wins
	transfers := make([]events.Transfer, 0, len(positions))
	wins := make([]events.Transfer, 0, len(positions))
	zeroTransfers := make([]events.Transfer, 0, len(positions)/2)
	totalW, totalL := num.UintZero(), num.UintZero()
	var delta num.Decimal
	for _, p := range positions {
		// per-party cash flow is -openVolume * fundingPayment
		flow, rem, neg := calcFundingFlow(fundingPayment, p, e.positionFactor)
		if neg {
			// amount of loss not collected, this never gets added to the settlement account
			delta = delta.Sub(rem)
		} else {
			// amount of wins never collected, remains in the settlement account
			delta = delta.Add(rem)
		}

		if tf, valid := e.getFundingTransfer(flow, neg, p, p.Party()); valid {
			if tf.transfer.Type == types.TransferTypePerpFundingWin {
				wins = append(wins, tf)
				totalW.AddSum(flow)
			} else {
				transfers = append(transfers, tf)
				totalL.AddSum(flow)
			}
		} else {
			// we could use deltas to order these transfers to prioritise the right people
			zeroTransfers = append(zeroTransfers, tf)
		}
		if e.log.IsDebug() {
			e.log.Debug("cash flow", logging.String("mid", e.market), logging.String("pid", p.Party()), logging.String("flow", flow.String()))
		}
	}
	// account for cases where the winning side never even accounts for an amount of 1
	if len(wins) == 0 && len(zeroTransfers) > 0 {
		wins = zeroTransfers
	}
	// profit and loss balances out perfectly, or profit > loss
	if totalL.LTE(totalW) {
		// this rounding shouldn't be needed, losses will be distributed in their entirety
		round, _ := num.UintFromDecimal(delta.Abs())
		return append(transfers, wins...), round
	}
	round := totalL.Sub(totalL, totalW) // loss - win is what will be left over
	// we have a remainder, make sure it's an expected amount due to rounding
	if dU, _ := num.UintFromDecimal(delta.Ceil().Abs()); dU.LT(round) {
		e.log.Panic("Excess loss transfer amount found, cannot be explained by rounding",
			logging.String("loss-win delta", round.String()),
			logging.Decimal("rounding delta", delta.Abs()),
		)
	}
	return append(transfers, wins...), round
}

func calcFundingFlow(fp *num.Int, p events.MarketPosition, posFac num.Decimal) (*num.Uint, num.Decimal, bool) {
	// -openVolume * fundingPayment
	// divide by position factor to account for position decimal places
	flowD := num.DecimalFromInt64(-p.Size()).Mul(num.DecimalFromInt(fp)).Div(posFac)
	neg := flowD.IsNegative()
	flow, frac := num.UintFromDecimalWithFraction(flowD.Abs())
	return flow, frac, neg
}
