package settlement

import (
	"sync"
	"time"

	"code.vegaprotocol.io/vega/internal/events"
	"code.vegaprotocol.io/vega/internal/logging"

	types "code.vegaprotocol.io/vega/proto"
)

// We should really use a type from the proto package for this, although, these mocks are kind of easy to set up :)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/market_position_mock.go -package mocks code.vegaprotocol.io/vega/internal/settlement MarketPosition
type MarketPosition interface {
	Party() string
	Size() int64
	Price() uint64
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/settlement_product_mock.go -package mocks code.vegaprotocol.io/vega/internal/settlement Product
type Product interface {
	Settle(entryPrice uint64, netPosition int64) (*types.FinancialAmount, error)
	GetAsset() string
}

// Engine - the main type (of course)
type Engine struct {
	log *logging.Logger

	Config
	mu      *sync.Mutex
	product Product
	pos     map[string]*pos
	closed  map[string][]*pos
	market  string
}

func New(log *logging.Logger, conf Config, product Product, market string) *Engine {
	// setup logger
	log = log.Named(namedLogger)
	log.SetLevel(conf.Level.Get())

	return &Engine{
		log:     log,
		Config:  conf,
		mu:      &sync.Mutex{},
		product: product,
		pos:     map[string]*pos{},
		market:  market,
	}
}

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

// Update - takes market positions, keeps track of things
func (e *Engine) Update(positions []MarketPosition) {
	e.mu.Lock()
	for _, p := range positions {
		e.updatePosition(p, p.Price())
	}
	e.mu.Unlock()
}

func (e *Engine) updatePosition(p MarketPosition, price uint64) {
	party := p.Party()
	ps, ok := e.pos[party]
	if !ok {
		ps = &pos{
			party: party,
		}
		e.pos[party] = ps
	}
	// price can come from either position, or trade (Update vs SettleMTM)
	ps.price = price
	ps.size = p.Size()
}

func (e *Engine) getPosition(party string) pos {
	ps, ok := e.pos[party]
	if !ok {
		ps = &pos{
			party: party,
		}
		e.pos[party] = ps
	}
	return *ps
}

func (e *Engine) Settle(t time.Time) ([]*types.Transfer, error) {
	e.mu.Lock()
	e.log.Debugf("Settling market, closed at %s", t.Format(time.RFC3339))
	positions, err := e.settleAll()
	if err != nil {
		e.log.Error(
			"Something went wrong trying to settle positions",
			logging.Error(err),
		)
		return nil, err
	}
	e.mu.Unlock()
	return positions, nil
}

// SettlePreTrade ensures that the MTM for traders involved in the trade are applied before closing out
// this applies the MTM for traders _before_ the trade happened
func (e *Engine) settlePreTrade(markPrice uint64, trade types.Trade) []*mtmTransfer {
	res := make([]*mtmTransfer, 0, 2)
	winS := make([]*mtmTransfer, 0, 1)
	e.mu.Lock()
	positions := map[string]pos{
		trade.Buyer:  e.getPosition(trade.Buyer),
		trade.Seller: e.getPosition(trade.Seller),
	}
	// buyer -> seller, know which to update
	var seller bool
	for owner, ps := range positions {
		// if markPrice == position price, or position size == 0
		// this won't really do anything, other than making sure the mark price matches
		// but the trade itself needn't have happened at the mark price, so we carry on regardless
		// because we add/subtract later on
		mtmShare := (int64(markPrice) - int64(ps.price)) * ps.size
		// update the positions after calculating mark-to-market share
		if !seller {
			// account for the trade size being different to the mark price:
			// subtract the (trade price - mark price) * trade volume from mtmShare
			// old pos +10@1k -> trade at 2K (new pos +11), mark price 1.5K
			// this would mean mtmShare == (1500 - 1000) * 10 => 5K
			// mtmShare -= (2000 - 1500) * 1 == 4.5K
			// perfectly valid again
			mtmShare -= (int64(trade.Price) - int64(markPrice)) * int64(trade.Size)
			ps.size += int64(trade.Size)
			e.updatePosition(ps, markPrice)
			seller = true
		} else {
			// add (trade price - mark price) * trade volume to the MTM share
			// if old pos was +1 @1K, trade at 2K (net pos 0) -> mark price 1.5K
			// this will yield a mtmShare of 500, + (2000 - 1500)*1 -> 1K MTM share, perfectly valid
			mtmShare += (int64(trade.Price) - int64(markPrice)) * int64(trade.Size)
			ps.size -= int64(trade.Size)
			e.updatePosition(ps, markPrice)
		}
		// nothing to mark to market, continue
		if mtmShare == 0 {
			continue
		}
		settle := &types.Transfer{
			Owner: owner,
			Size:  1,
			Amount: &types.FinancialAmount{
				Amount: mtmShare,
				Asset:  e.product.GetAsset(),
			},
		}
		if mtmShare > 0 {
			settle.Type = types.TransferType_MTM_WIN
			winS = append(winS, &mtmTransfer{
				MarketPosition: ps,
				transfer:       settle,
			})
		} else {
			settle.Type = types.TransferType_MTM_LOSS
			res = append(res, &mtmTransfer{
				MarketPosition: ps,
				transfer:       settle,
			})
		}
	}
	e.mu.Unlock()
	if len(winS) > 0 {
		// append wins if any...
		res = append(res, winS...)
	}
	return res
}

func (e *Engine) ListenClosed(ch <-chan events.MarketPosition) {
	// lock before we can start
	e.mu.Lock()
	go func() {
		// e.mu.Lock()
		// wipe closed map
		e.closed = map[string][]*pos{}
		for ps := range ch {
			trader := ps.Party()
			size := ps.Size()
			price := ps.Price()
			updatePrice := price
			// check current position to see if trade closed out some position
			current := e.getCurrentPosition(trader)
			// if trader is long, and trade closed out (part of) long position, or trader was short, and is now "less short"
			if (current.size > 0 && size < current.size) || (current.size < 0 && size > current.size) {
				closed := current.size
				// trader was long, and still is || trader was short && still is
				if (current.size > 0 && size > 0) || (current.size < 0 && size < 0) {
					// trader was +10, now +5 -> +10 - +5 == MTM on +5 closed positions --> good
					// trader was -10, now -5 -> -10 - -5 == MTM on -5 closed positions --> good
					closed -= size
					updatePrice = current.price
				}
				// let's add this change to the traders' closed positions to be added to the MTM settlement later on
				trades, ok := e.closed[trader]
				if !ok {
					trades = []*pos{}
				}
				e.closed[trader] = append(trades, &pos{
					party: trader,
					size:  closed,
					price: current.price, // we closed out at the old price vs mark price
				})
			}
			// we've taken the closed out stuff into account, so we can freely update the size here
			current.size = size
			// the position price is possibly updated (e.g. if there was no open position prior to this, or trader went from long to short or vice-versa)
			current.price = updatePrice
		}
		e.mu.Unlock()
	}()
}

// SettleOrder - settlements based on order-level, can take several update positions, and marks all to market
// if party size and price were both updated (ie party was a trader), we're combining both MTM's (old + new position)
// and creating a single transfer from that
func (e *Engine) SettleOrder(markPrice uint64, positions []events.MarketPosition) []events.Transfer {
	transfers := make([]events.Transfer, 0, len(positions))
	winTransfers := make([]events.Transfer, 0, len(positions)/2)
	// see if we've got closed out positions
	e.mu.Lock()
	closed := e.closed
	// reset map here in case we're going to call this with just an updated mark price
	e.closed = map[string][]*pos{}
	e.mu.Unlock()
	for _, pos := range positions {
		size := pos.Size()
		price := pos.Price()
		trader := pos.Party()
		current := e.getCurrentPosition(trader)
		// markPrice was already set by positions engine
		// e.g. position avg -> 90, mark price 100:
		// short -> (100 - 90) * -10 => -100 ==> MTM_LOSS
		// long -> (100-90) * 10 => 100 ==> MTM_WIN
		// short -> (100 - 110) * -10 => 100 ==> MTM_WIN
		// long -> (100 - 110) * 10 => -100 ==> MTM_LOSS
		closedOut, _ := closed[trader]
		// updated price is mark price, mark against that using current known price
		if price == markPrice {
			price = current.price
		}
		mtmShare := calcMTM(int64(markPrice), size, int64(price), closedOut)
		// update position
		current.size = size
		current.price = markPrice
		// there's nothing to mark to market
		if mtmShare == 0 {
			continue
		}
		settle := &types.Transfer{
			Owner: current.party,
			Size:  1, // this is an absolute delta based on volume, so size is always 1
			Amount: &types.FinancialAmount{
				Amount: mtmShare, // current delta -> mark price minus current position average
				Asset:  e.product.GetAsset(),
			},
		}
		if mtmShare > 0 {
			settle.Type = types.TransferType_MTM_WIN
			winTransfers = append(winTransfers, &mtmTransfer{
				MarketPosition: pos,
				transfer:       settle,
			})
		} else {
			// losses are prepended
			settle.Type = types.TransferType_MTM_LOSS
			transfers = append(transfers, &mtmTransfer{
				MarketPosition: pos,
				transfer:       settle,
			})
		}
	}
	transfers = append(transfers, winTransfers...)
	return transfers
}

func calcMTM(markPrice, size, price int64, closed []*pos) (mtmShare int64) {
	mtmShare = (markPrice - price) * size
	for _, c := range closed {
		// add MTM compared to trade price for the positions that were closed out
		mtmShare += (markPrice - int64(c.price)) * c.size
	}
	return
}

func getMTMAmount(markPrice, currentPrice, currentSize, newSize, newPrice int64) (mtmShare int64) {
	mtmShare = (markPrice - newPrice) * newSize
	if currentSize == newSize {
		return
	}
	// trader was long
	if currentSize > 0 {
		// trader increased overall long position, no separate MTM required
		if newSize > currentSize {
			return
		}
		// the trader went from long to short, the only part that we need to MTM is the long position held
		if newSize < 0 {
			newSize = 0
		}
		// the trader was long, and is now "less long", we need to add the MTM of the delta
		// so trader went from +10 to +5 => add MTM for 5
		mtmShare += (markPrice - currentPrice) * (currentSize - newSize)
		return
	}
	// now in case trader was short
	if newSize < currentSize {
		// trader went "even more short", nothing needs to be done
		return
	}
	// new size is either zero, or trader went long
	// we need to MTM the previously held short position
	if newSize > 0 {
		newSize = 0
	}
	// newSize is still short, but less short than the previous position
	// add MTM share
	mtmShare += (markPrice - currentPrice) * (currentSize - newSize)
	return
}

func (e *Engine) getCurrentPosition(trader string) *pos {
	p, ok := e.pos[trader]
	if !ok {
		p = &pos{
			party: trader,
		}
		e.pos[trader] = p
	}
	return p
}

func (e *Engine) SettleMTM(trade types.Trade, markPrice uint64, ch <-chan events.MarketPosition) <-chan []events.Transfer {
	// put the positions on here once we've worked out what all we need to settle
	sch := make(chan []events.Transfer)
	// sch := make(chan []*types.Transfer)
	tradePos := e.settlePreTrade(markPrice, trade)
	go func() {
		posE := make([]events.Transfer, 0, cap(ch))
		winE := make([]events.Transfer, 0, cap(ch)/2)
		// ensure we've got the MTM for buyer/seller _before_ trade was applied
		// makes sure the order is preserved, too
		for _, sp := range tradePos {
			if sp.transfer.Type == types.TransferType_MTM_WIN {
				winE = append(winE, sp)
			} else {
				posE = append(posE, sp)
			}
		}
		e.mu.Lock()
		for pos := range ch {
			if pos == nil {
				break
			}
			// trade.Buyer == owner || trade.Seller == owner
			// update position for trader - always keep track of latest position
			ps := pos.Size()
			pp := pos.Price()
			// all positions need to be updated to the new mark price
			e.updatePosition(pos, markPrice)
			if pp == markPrice || ps == 0 {
				// nothing has changed or there's no position to settle
				continue
			}
			// e.g. position avg -> 90, mark price 100:
			// short -> (100 - 90) * -10 => -100 ==> MTM_LOSS
			// long -> (100-90) * 10 => 100 ==> MTM_WIN
			// short -> (100 - 110) * -10 => 100 ==> MTM_WIN
			// long -> (100 - 110) * 10 => -100 ==> MTM_LOSS
			// @TODO -> move to product level (same as prod.Settle)
			mtmShare := (int64(markPrice) - int64(pp)) * ps
			if mtmShare == 0 {
				continue
			}
			settle := &types.Transfer{
				Owner: pos.Party(),
				Size:  1, // this is an absolute delta based on volume, so size is always 1
				Amount: &types.FinancialAmount{
					Amount: mtmShare, // current delta -> mark price minus current position average
					Asset:  e.product.GetAsset(),
				},
			}
			if mtmShare > 0 {
				settle.Type = types.TransferType_MTM_WIN
				winE = append(winE, &mtmTransfer{
					MarketPosition: pos,
					transfer:       settle,
				})
			} else {
				settle.Type = types.TransferType_MTM_LOSS
				posE = append(posE, &mtmTransfer{
					MarketPosition: pos,
					transfer:       settle,
				})
			}
		}
		e.mu.Unlock()
		posE = append(posE, winE...)
		sch <- posE
		close(sch)
	}()
	return sch
}

// simplified settle call
func (e *Engine) settleAll() ([]*types.Transfer, error) {
	// there should be as many positions as there are traders (obviously)
	aggregated := make([]*types.Transfer, 0, len(e.pos))
	// traders who are in the black should be appended (collect first) obvioulsy.
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
		amt, err := e.product.Settle(pos.price, pos.size)
		// for now, product.Settle returns the total value, we need to only settle the delta between a traders current position
		// and the final price coming from the oracle, so oracle_price - mark_price * volume (check with Tamlyn whether this should be absolute or not)
		if err != nil {
			e.log.Error(
				"Failed to settle position for trader",
				logging.String("trader-id", party),
				logging.Error(err),
			)
			return nil, err
		}
		settlePos := &types.Transfer{
			Owner:  party,
			Size:   1,
			Amount: amt,
		}
		e.log.Debug(
			"Settled position for trader",
			logging.String("trader-id", party),
			logging.Int64("amount", amt.Amount),
		)
		if amt.Amount < 0 {
			// trader is winning...
			settlePos.Type = types.TransferType_LOSS
			aggregated = append(aggregated, settlePos)
		} else {
			// bad name again, but SELL means trader is owed money
			settlePos.Type = types.TransferType_WIN
			owed = append(owed, settlePos)
		}
	}
	// append the traders in the black to the end
	aggregated = append(aggregated, owed...)
	return aggregated, nil
}
