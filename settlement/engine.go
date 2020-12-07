package settlement

import (
	"context"
	"errors"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/metrics"
	types "code.vegaprotocol.io/vega/proto"
)

// MarketPosition ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/market_position_mock.go -package mocks code.vegaprotocol.io/vega/settlement MarketPosition
type MarketPosition interface {
	Party() string
	Size() int64
	Buy() int64
	Sell() int64
	Price() uint64
	VWBuy() uint64
	VWSell() uint64
	ClearPotentials()
}

// Product ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/settlement_product_mock.go -package mocks code.vegaprotocol.io/vega/settlement Product
type Product interface {
	Settle(entryPrice uint64, netPosition int64) (*types.FinancialAmount, error)
	GetAsset() string
}

// Broker - the event bus broker, send events here
//go:generate go run github.com/golang/mock/mockgen -destination mocks/broker_mock.go -package mocks code.vegaprotocol.io/vega/settlement Broker
type Broker interface {
	SendBatch(events []events.Event)
}

// Engine - the main type (of course)
type Engine struct {
	Config
	log *logging.Logger

	market      string
	product     Product
	pos         map[string]*pos
	mu          *sync.Mutex
	trades      map[string][]*pos
	broker      Broker
	currentTime time.Time
}

func (e *Engine) OnTick(t time.Time) {
	e.currentTime = t
}

// New instantiates a new instance of the settlement engine
func New(log *logging.Logger, conf Config, product Product, market string, broker Broker) *Engine {
	// setup logger
	log = log.Named(namedLogger)
	log.SetLevel(conf.Level.Get())

	return &Engine{
		Config:  conf,
		log:     log,
		market:  market,
		product: product,
		pos:     map[string]*pos{},
		mu:      &sync.Mutex{},
		trades:  map[string][]*pos{},
		broker:  broker,
	}
}

// ReloadConf update the internal configuration of the settlement engined
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
// perhaps the tests should be refactored to use the Settle call to create positions first
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

// Settle run settlement over all the positions
func (e *Engine) Settle(t time.Time, lastMarkPrice uint64) ([]*types.Transfer, error) {
	e.log.Debugf("Settling market, closed at %s", t.Format(time.RFC3339))
	positions, err := e.settleAll(lastMarkPrice)
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
// each change in position has to be calculated using the exact price of the trade
func (e *Engine) AddTrade(trade *types.Trade) {
	e.mu.Lock()
	var (
		buyerSize, sellerSize int64
	)
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
	// the traders both need to get a MTM settlement on the traded volume
	// and this MTM part has to be based on the _actual_ trade value
	e.trades[trade.Buyer] = append(e.trades[trade.Buyer], &pos{
		price:   trade.Price,
		size:    size,
		newSize: buyerSize + size,
	})
	e.trades[trade.Seller] = append(e.trades[trade.Seller], &pos{
		price:   trade.Price,
		size:    -size,
		newSize: sellerSize - size,
	})
	e.mu.Unlock()
}

func (e *Engine) SettleMTM(ctx context.Context, markPrice uint64, positions []events.MarketPosition) []events.Transfer {
	timer := metrics.NewTimeCounter("-", "settlement", "SettleOrder")
	e.mu.Lock()
	tCap := e.transferCap(positions)
	transfers := make([]events.Transfer, 0, tCap)
	// roughly half of the transfers should be wins, half losses
	wins := make([]events.Transfer, 0, tCap/2)
	trades := e.trades
	e.trades = map[string][]*pos{} // remove here, once we've processed it all here, we're done
	mpSigned := int64(markPrice)
	evts := make([]events.Event, 0, len(positions))
	for _, evt := range positions {
		party := evt.Party()
		// get the current position, and all (if any) position changes because of trades
		current := e.getCurrentPosition(party, evt)
		// we don't care if this is a nil value
		traded, hasTraded := trades[party]
		tradeset := make([]events.TradeSettlement, 0, len(traded))
		for _, t := range traded {
			tradeset = append(tradeset, t)
		}
		// create (and add position to buffer)
		evts = append(evts, events.NewSettlePositionEvent(ctx, party, e.market, evt.Price(), tradeset, e.currentTime.UnixNano()))
		// no changes in position, and the MTM price hasn't changed, we don't need to do anything
		if !hasTraded && current.price == markPrice {
			// no changes in position and markPrice hasn't changed -> nothing needs to be marked
			continue
		}
		// calculate MTM value, we need the signed mark-price, the OLD open position/volume
		// the new position is either the same, or accounted for by the traded var (added trades)
		// and the old mark price at which the trader held the position
		// the trades slice contains all trade positions (position changes for the trader)
		// at their exact trade price, so we can MTM that volume correctly, too
		mtmShare := calcMTM(mpSigned, current.size, int64(current.price), traded)
		// we've marked this trader to market, their position can now reflect this
		current.update(evt)
		current.price = markPrice
		// we don't want to accidentally MTM a trader who closed out completely when they open
		// a new position at a later point, so remove if size == 0
		if current.size == 0 {
			// broke this up into its own func for symmetry
			e.rmPosition(party)
		}
		// we don't need to create a transfer if there's no changes to the balance...
		if mtmShare == 0 {
			wins = append(wins, &mtmTransfer{
				MarketPosition: current,
				transfer:       nil,
			})

			continue
		}
		settle := &types.Transfer{
			Owner: party,
			Amount: &types.FinancialAmount{
				Amount: mtmShare, // current delta -> mark price minus current position average
				Asset:  e.product.GetAsset(),
			},
		}

		if mtmShare > 0 {
			settle.Type = types.TransferType_TRANSFER_TYPE_MTM_WIN
			wins = append(wins, &mtmTransfer{
				MarketPosition: current,
				transfer:       settle,
			})
		} else {
			// losses are prepended
			settle.Type = types.TransferType_TRANSFER_TYPE_MTM_LOSS
			transfers = append(transfers, &mtmTransfer{
				MarketPosition: current,
				transfer:       settle,
			})
		}
	}
	// append wins after loss transfers
	transfers = append(transfers, wins...)
	// whatever was added to the buffer is now ready to be flushed
	e.mu.Unlock()
	e.broker.SendBatch(evts)
	timer.EngineTimeCounterAdd()
	return transfers
}

// RemoveDistressed - remove whatever settlement data we have for distressed traders
// they are being closed out, and shouldn't be part of any MTM settlement or closing settlement
func (e *Engine) RemoveDistressed(ctx context.Context, evts []events.Margin) {
	devts := make([]events.Event, 0, len(evts))
	e.mu.Lock()
	for _, v := range evts {
		key := v.Party()
		devts = append(devts, events.NewSettleDistressed(ctx, key, e.market, v.Price(), v.MarginBalance()+v.GeneralBalance(), e.currentTime.UnixNano()))
		delete(e.pos, key)
		delete(e.trades, key)
	}
	e.mu.Unlock()
	e.broker.SendBatch(devts)
}

func (e *Engine) getSettlementProduct(lastMarkPrice uint64) (Product, error) {
	switch e.Config.FinalSettlement.Get() {
	case FinalSettlementOracle:
		return e.product, nil
	case FinalSettlementMarkPrice:
		return &lastMarkPriceSettlement{lastMarkPrice, e.product.GetAsset()}, nil
	default:
		// can't happen at this point but...
		return nil, errors.New("invalid configuration: unknow final settlement")
	}
}

// simplified settle call
func (e *Engine) settleAll(lastMarkPrice uint64) ([]*types.Transfer, error) {
	e.mu.Lock()

	settleProd, err := e.getSettlementProduct(lastMarkPrice)
	if err != nil {
		return nil, err
	}

	// there should be as many positions as there are traders (obviously)
	aggregated := make([]*types.Transfer, 0, len(e.pos))
	// traders who are in profit should be appended (collect first).
	// The split won't always be 50-50, but it's a reasonable approximation
	owed := make([]*types.Transfer, 0, len(e.pos)/2)
	for party, pos := range e.pos {
		// this is possible now, with the Mark to Market stuff, it's possible we've settled any and all positions for a given trader
		if pos.size == 0 {
			continue
		}
		e.log.Debug("Settling position for trader", logging.String("trader-id", party))
		// @TODO - there was something here... the final amount had to be oracle - market or something
		// check with Tamlyn why that was, because we're only handling open positions here...
		amt, err := settleProd.Settle(pos.price, pos.size)
		// for now, product.Settle returns the total value, we need to only settle the delta between a traders current position
		// and the final price coming from the oracle, so oracle_price - mark_price * volume (check with Tamlyn whether this should be absolute or not)
		if err != nil {
			e.log.Error(
				"Failed to settle position for trader",
				logging.String("trader-id", party),
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
			"Settled position for trader",
			logging.String("trader-id", party),
			logging.Int64("amount", amt.Amount),
		)
		if amt.Amount < 0 {
			// trader is winning...
			settlePos.Type = types.TransferType_TRANSFER_TYPE_LOSS
			aggregated = append(aggregated, settlePos)
		} else {
			// bad name again, but SELL means trader is owed money
			settlePos.Type = types.TransferType_TRANSFER_TYPE_WIN
			owed = append(owed, settlePos)
		}
	}
	// append the traders in profit to the end
	aggregated = append(aggregated, owed...)
	e.mu.Unlock()
	return aggregated, nil
}

// this doesn't need the mutex wrap because it's an internal call and the function that is being
// called already locks the positions map
func (e *Engine) getCurrentPosition(party string, evt events.MarketPosition) *pos {
	p, ok := e.pos[party]
	if !ok {
		p = newPos(evt)
		e.pos[party] = p
	}
	return p
}

func (e *Engine) rmPosition(party string) {
	delete(e.pos, party)
}

// just get the max len as cap
func (e *Engine) transferCap(evts []events.MarketPosition) int {
	curLen, evtLen := len(e.pos), len(evts)
	if curLen >= evtLen {
		return curLen
	}
	return evtLen
}

//party.PREV_OPEN_VOLUME * (product.value(current_price) - product.value(prev_mark_price)) + SUM(from i=1 to new_trades.length)( new_trade(i).volume(party) * (product.value(current_price) - new_trade(i).price ) )
// the sum bit is a worry, we do not have all the trades available at this point...

// calcMTM only handles futures ATM. The formula is simple:
// amount =  prev_vol * (current_price - prev_mark_price) + SUM(new_trade := range trades)( new_trade(i).volume(party)*(current_price - new_trade(i).price )
// given that the new trades price will equal new mark price,  the sum(trades) bit will probably == 0 for nicenet
// the size here is the _new_ position size, the price is the OLD price!!
func calcMTM(markPrice, size, price int64, trades []*pos) (mtmShare int64) {
	mtmShare = (markPrice - price) * size
	for _, c := range trades {
		// add MTM compared to trade price for the positions changes for trades
		mtmShare += (markPrice - int64(c.price)) * c.size
	}
	return
}

type lastMarkPriceSettlement struct {
	markPrice uint64
	asset     string
}

func (l *lastMarkPriceSettlement) Settle(entryPrice uint64, netPosition int64) (*types.FinancialAmount, error) {
	return &types.FinancialAmount{
		Asset:  l.asset,
		Amount: int64((l.markPrice - entryPrice)) * netPosition,
	}, nil
}

func (l *lastMarkPriceSettlement) GetAsset() string {
	return l.asset
}
