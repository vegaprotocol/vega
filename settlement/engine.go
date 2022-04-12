package settlement

import (
	"context"
	"sort"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/metrics"
	"code.vegaprotocol.io/vega/products"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
)

// MarketPosition ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/market_position_mock.go -package mocks code.vegaprotocol.io/vega/settlement MarketPosition
type MarketPosition interface {
	Party() string
	Size() int64
	Buy() int64
	Sell() int64
	Price() *num.Uint
	VWBuy() *num.Uint
	VWSell() *num.Uint
	ClearPotentials()
}

// Product ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/settlement_product_mock.go -package mocks code.vegaprotocol.io/vega/settlement Product
type Product interface {
	Settle(entryPrice *num.Uint, assetDecimals uint32, netFractionalPosition num.Decimal) (amt *types.FinancialAmount, neg bool, err error)
	GetAsset() string
	SettlementPrice() (*num.Decimal, error)
}

// Broker - the event bus broker, send events here.
type Broker interface {
	SendBatch(events []events.Event)
}

// Engine - the main type (of course).
type Engine struct {
	Config
	log *logging.Logger

	market         string
	product        Product
	pos            map[string]*pos
	mu             *sync.Mutex
	trades         map[string][]*pos
	broker         Broker
	currentTime    time.Time
	positionFactor num.Decimal
}

func (e *Engine) OnTick(t time.Time) {
	e.currentTime = t
}

// New instantiates a new instance of the settlement engine.
func New(log *logging.Logger, conf Config, product Product, market string, broker Broker, positionFactor num.Decimal) *Engine {
	// setup logger
	log = log.Named(namedLogger)
	log.SetLevel(conf.Level.Get())

	return &Engine{
		Config:         conf,
		log:            log,
		market:         market,
		product:        product,
		pos:            map[string]*pos{},
		mu:             &sync.Mutex{},
		trades:         map[string][]*pos{},
		broker:         broker,
		positionFactor: positionFactor,
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
		if ps, ok := e.pos[party]; ok {
			// ATM, this can't possibly return an error, hence we're ignoring it
			_ = ps.update(p)
		} else {
			e.pos[party] = newPos(p)
		}
	}
	e.mu.Unlock()
}

// Settle run settlement over all the positions.
func (e *Engine) Settle(t time.Time, assetDecimals uint32) ([]*types.Transfer, error) {
	e.log.Debugf("Settling market, closed at %s", t.Format(time.RFC3339))
	positions, err := e.settleAll(assetDecimals)
	if err != nil {
		e.log.Error(
			"Something went wrong trying to settle positions",
			logging.Error(err),
		)
		return nil, err
	}
	return positions, nil
}

// AddTrade - this call is required to get the correct MTM settlement values
// each change in position has to be calculated using the exact price of the trade.
func (e *Engine) AddTrade(trade *types.Trade) {
	e.mu.Lock()
	var buyerSize, sellerSize int64
	// checking the len of cd shouldn't be required here, but it is needed in the second if
	// in case the buyer and seller are one and the same...
	if cd, ok := e.trades[trade.Buyer]; !ok || len(cd) == 0 {
		e.trades[trade.Buyer] = []*pos{}
		// check if the buyer already has a known position
		if pos, ok := e.pos[trade.Buyer]; ok {
			buyerSize = pos.size
		}
	} else {
		buyerSize = cd[len(cd)-1].newSize
	}
	if cd, ok := e.trades[trade.Seller]; !ok || len(cd) == 0 {
		e.trades[trade.Seller] = []*pos{}
		// check if seller has a known position
		if pos, ok := e.pos[trade.Seller]; ok {
			sellerSize = pos.size
		}
	} else {
		sellerSize = cd[len(cd)-1].newSize
	}
	size := int64(trade.Size)
	// the parties both need to get a MTM settlement on the traded volume
	// and this MTM part has to be based on the _actual_ trade value
	price := trade.Price.Clone()
	e.trades[trade.Buyer] = append(e.trades[trade.Buyer], &pos{
		price:   price,
		size:    size,
		newSize: buyerSize + size,
	})
	e.trades[trade.Seller] = append(e.trades[trade.Seller], &pos{
		price:   price.Clone(),
		size:    -size,
		newSize: sellerSize - size,
	})
	e.mu.Unlock()
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
				Amount: num.Zero(),
				Asset:  e.product.GetAsset(),
			},
		}
	}
	transfer.transfer.Amount.Amount.AddSum(amt)
}

func (e *Engine) SettleMTM(ctx context.Context, markPrice *num.Uint, positions []events.MarketPosition) []events.Transfer {
	timer := metrics.NewTimeCounter("-", "settlement", "SettleOrder")
	e.mu.Lock()
	tCap := e.transferCap(positions)
	transfers := make([]events.Transfer, 0, tCap)
	// roughly half of the transfers should be wins, half losses
	wins := make([]events.Transfer, 0, tCap/2)
	trades := e.trades
	e.trades = map[string][]*pos{} // remove here, once we've processed it all here, we're done
	evts := make([]events.Event, 0, len(positions))
	var (
		largestShare *mtmTransfer       // pointer to whomever gets the last remaining amount from the loss
		zeroShares   = []*mtmTransfer{} // all zero shares for equal distribution if possible
		zeroAmts     = false
		mtmDec       = num.NewDecimalFromFloat(0)
		lossTotal    = num.Zero()
		winTotal     = num.Zero()
		lossTotalDec = num.NewDecimalFromFloat(0)
		winTotalDec  = num.NewDecimalFromFloat(0)
	)

	// Process any network trades first
	traded, hasTraded := trades[types.NetworkParty]
	if hasTraded {
		// don't create an event for the network. Its position is irrelevant

		mtmShare, mtmDShare, neg := calcMTM(markPrice, markPrice, 0, traded, e.positionFactor)
		// MarketPosition stub for network
		netMPos := &npos{
			price: markPrice.Clone(),
		}

		mtmTransfer := e.getMtmTransfer(mtmShare.Clone(), neg, netMPos, types.NetworkParty)

		if !neg {
			wins = append(wins, mtmTransfer)
			winTotal.AddSum(mtmShare)
			winTotalDec = winTotalDec.Add(mtmDShare)
			mtmDec = mtmDShare
			largestShare = mtmTransfer
			// mtmDec is zero at this point, this will always be the largest winning party at this point
			if mtmShare.IsZero() {
				zeroShares = append(zeroShares, mtmTransfer)
				zeroAmts = true
			}
		} else if mtmShare.IsZero() {
			// This would be a zero-value loss, so not sure why this would be at the end of the slice
			// shouldn't really matter if this is in the wins or losses part of the slice, but
			// this was the previous behaviour, so let's keep it
			wins = append(wins, mtmTransfer)
			lossTotalDec = lossTotalDec.Add(mtmDShare)
		} else {
			transfers = append(transfers, mtmTransfer)
			lossTotal.AddSum(mtmShare)
			lossTotalDec = lossTotalDec.Add(mtmDShare)
		}
	}

	for _, evt := range positions {
		party := evt.Party()
		// get the current position, and all (if any) position changes because of trades
		current := e.getCurrentPosition(party, evt)
		// we don't care if this is a nil value
		traded, hasTraded = trades[party]
		tradeset := make([]events.TradeSettlement, 0, len(traded))
		for _, t := range traded {
			tradeset = append(tradeset, t)
		}
		// create (and add position to buffer)
		evts = append(evts, events.NewSettlePositionEvent(ctx, party, e.market, evt.Price(), tradeset, e.currentTime.UnixNano(), e.positionFactor))
		// no changes in position, and the MTM price hasn't changed, we don't need to do anything
		if !hasTraded && current.price.EQ(markPrice) {
			// no changes in position and markPrice hasn't changed -> nothing needs to be marked
			continue
		}
		// calculate MTM value, we need the signed mark-price, the OLD open position/volume
		// the new position is either the same, or accounted for by the traded var (added trades)
		// and the old mark price at which the party held the position
		// the trades slice contains all trade positions (position changes for the party)
		// at their exact trade price, so we can MTM that volume correctly, too
		mtmShare, mtmDShare, neg := calcMTM(markPrice, current.price, current.size, traded, e.positionFactor)
		// we've marked this party to market, their position can now reflect this
		_ = current.update(evt)
		current.price = markPrice
		// we don't want to accidentally MTM a party who closed out completely when they open
		// a new position at a later point, so remove if size == 0
		if current.size == 0 && current.Buy() == 0 && current.Sell() == 0 {
			// broke this up into its own func for symmetry
			e.rmPosition(party)
		}

		mtmTransfer := e.getMtmTransfer(mtmShare.Clone(), neg, current, current.Party())

		if !neg {
			wins = append(wins, mtmTransfer)
			winTotal.AddSum(mtmShare)
			winTotalDec = winTotalDec.Add(mtmDShare)
			if mtmShare.IsZero() {
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
	delta := num.Zero().Sub(lossTotal, winTotal)
	if !delta.IsZero() {
		if zeroAmts {
			// there are more transfers from losses than we pay out to wins, but some winning parties have zero transfers
			// this delta should == combined win decimals, let's sanity check this!
			if winTotalDec.LessThan(lossTotalDec) {
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
	e.broker.SendBatch(evts)
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
		devts = append(devts, events.NewSettleDistressed(ctx, key, e.market, v.Price(), margin, e.currentTime.UnixNano()))
		delete(e.pos, key)
		delete(e.trades, key)
	}
	e.mu.Unlock()
	e.broker.SendBatch(devts)
}

// simplified settle call.
func (e *Engine) settleAll(assetDecimals uint32) ([]*types.Transfer, error) {
	e.mu.Lock()

	// there should be as many positions as there are parties (obviously)
	aggregated := make([]*types.Transfer, 0, len(e.pos))
	// parties who are in profit should be appended (collect first).
	// The split won't always be 50-50, but it's a reasonable approximation
	owed := make([]*types.Transfer, 0, len(e.pos)/2)
	// ensure we iterate over the positions in the same way by getting all the parties (keys)
	// and sort them
	keys := make([]string, 0, len(e.pos))
	for p := range e.pos {
		keys = append(keys, p)
	}
	sort.Strings(keys)
	for _, party := range keys {
		pos := e.pos[party]
		// this is possible now, with the Mark to Market stuff, it's possible we've settled any and all positions for a given party
		if pos.size == 0 {
			continue
		}
		e.log.Debug("Settling position for party", logging.String("party-id", party))
		// @TODO - there was something here... the final amount had to be oracle - market or something
		// check with Tamlyn why that was, because we're only handling open positions here...
		amt, neg, err := e.product.Settle(pos.price, assetDecimals, num.DecimalFromInt64(pos.size).Div(e.positionFactor))
		// for now, product.Settle returns the total value, we need to only settle the delta between a parties current position
		// and the final price coming from the oracle, so oracle_price - mark_price * volume (check with Tamlyn whether this should be absolute or not)
		if err != nil {
			e.log.Error(
				"Failed to settle position for party",
				logging.String("party-id", party),
				logging.Error(err),
			)
			e.mu.Unlock()
			return nil, err
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
		} else { // this is a win transfer
			settlePos.Type = types.TransferTypeWin
			owed = append(owed, settlePos)
		}
	}
	// append the parties in profit to the end
	aggregated = append(aggregated, owed...)
	e.mu.Unlock()
	return aggregated, nil
}

// this doesn't need the mutex wrap because it's an internal call and the function that is being
// called already locks the positions map.
func (e *Engine) getCurrentPosition(party string, evt events.MarketPosition) *pos {
	p, ok := e.pos[party]
	if !ok {
		p = newPos(evt)
		e.pos[party] = p
	}
	return p
}

func (e *Engine) AddPosition(party string, evt events.MarketPosition) {
	_ = e.getCurrentPosition(party, evt)
}

func (e *Engine) rmPosition(party string) {
	delete(e.pos, party)
}

// just get the max len as cap.
func (e *Engine) transferCap(evts []events.MarketPosition) int {
	curLen, evtLen := len(e.pos), len(evts)
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
func calcMTM(markPrice, price *num.Uint, size int64, trades []*pos, positionFactor num.Decimal) (*num.Uint, num.Decimal, bool) {
	delta, sign := num.Zero().Delta(markPrice, price)
	// this shouldn't be possible I don't think, but just in case
	if size < 0 {
		size = -size
		// swap sign
		sign = !sign
	}
	mtmShare := delta.Mul(delta, num.NewUint(uint64(size)))
	for _, c := range trades {
		delta, neg := num.Zero().Delta(markPrice, c.price)
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
