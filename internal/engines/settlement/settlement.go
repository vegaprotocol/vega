package settlement

import (
	"sync"
	"time"

	"code.vegaprotocol.io/vega/internal/logging"
	types "code.vegaprotocol.io/vega/proto"
)

// We should really use a type from the proto package for this, although, these mocks are kind of easy to set up :)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/market_position_mock.go -package mocks code.vegaprotocol.io/vega/internal/engines/settlement MarketPosition
type MarketPosition interface {
	Party() string
	Size() int64
	Price() uint64
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/settlement_product_mock.go -package mocks code.vegaprotocol.io/vega/internal/engines/settlement Product
type Product interface {
	Settle(entryPrice uint64, netPosition int64) (*types.FinancialAmount, error)
}

// Engine - the main type (of course)
type Engine struct {
	log *logging.Logger

	Config
	mu      *sync.Mutex
	product Product
	pos     map[string]*pos
}

func New(log *logging.Logger, conf Config, product Product) *Engine {
	// setup logger
	log = log.Named(namedLogger)
	log.SetLevel(conf.Level.Get())

	return &Engine{
		log:     log,
		Config:  conf,
		mu:      &sync.Mutex{},
		product: product,
		pos:     map[string]*pos{},
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

func (e *Engine) Settle(t time.Time) ([]*types.SettlePosition, error) {
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
func (e *Engine) SettlePreTrade(markPrice uint64, trade *types.Trade) []*types.SettlePosition {
	result := make([]*types.SettlePosition, 0, 2)
	winSlice := make([]*types.SettlePosition, 0, 1) // expect 1 loss, 1 win (worst case 2 wins)
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
		// perhaps the mark price check should be separated out in case size == 0
		mtmShare := (int64(markPrice) - int64(ps.price)) * ps.size
		// update the positions after calculating mark-to-market share
		if !seller {
			ps.size += int64(trade.Size)
			e.updatePosition(ps, markPrice)
			seller = true
		} else {
			ps.size -= int64(trade.Size)
			e.updatePosition(ps, markPrice)
		}
		// nothing to mark to market, continue
		if mtmShare == 0 {
			continue
		}
		settle := &types.SettlePosition{
			Owner: owner,
			Size:  1,
			Amount: &types.FinancialAmount{
				Amount: mtmShare,
			},
		}
		if mtmShare > 0 {
			settle.Type = types.SettleType_MTM_WIN
			winSlice = append(winSlice, settle)
		} else {
			settle.Type = types.SettleType_MTM_LOSS
			result = append(result, settle)
		}
	}
	e.mu.Unlock()
	if len(winSlice) > 0 {
		// append wins if any...
		result = append(result, winSlice...)
	}
	return result
}

func (e *Engine) SettleMTM(trade *types.Trade, markPrice uint64, ch <-chan MarketPosition) <-chan []*types.SettlePosition {
	// put the positions on here once we've worked out what all we need to settle
	sch := make(chan []*types.SettlePosition)
	tradePos := e.SettlePreTrade(markPrice, trade)
	go func() {
		posSlice := make([]*types.SettlePosition, 0, cap(ch))
		winSlice := make([]*types.SettlePosition, 0, cap(ch)/2)
		// ensure we've got the MTM for buyer/seller _before_ trade was applied
		// makes sure the order is preserved, too
		for _, sp := range tradePos {
			if sp.Type == types.SettleType_MTM_WIN {
				winSlice = append(winSlice, sp)
			} else {
				posSlice = append(posSlice, sp)
			}
		}
		e.mu.Lock()
		for pos := range ch {
			if pos == nil {
				break
			}
			// MTM on traders in a trade -> TODO
			// 1st trade for person -> trade entry price might not be mark price
			// trade comes in, can be a MTM immediately needed
			//
			// @TODO -> trader net pos == 0, but closes out in trade
			// trade.Buyer == owner || trade.Seller == owner
			// update position for trader - always keep track of latest position
			pp := pos.Price()
			ps := pos.Size()
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
			mtmShare := (int64(markPrice) - int64(pp)) * ps
			if mtmShare == 0 {
				continue
			}
			settle := &types.SettlePosition{
				Owner: pos.Party(),
				Size:  1, // this is an absolute delta based on volume, so size is always 1
				Amount: &types.FinancialAmount{
					Amount: mtmShare, // current delta -> mark price minus current position average
				},
			}
			if mtmShare > 0 {
				settle.Type = types.SettleType_MTM_WIN
				winSlice = append(winSlice, settle)
			} else {
				settle.Type = types.SettleType_MTM_LOSS
				posSlice = append(posSlice, settle)
			}
		}
		e.mu.Unlock()
		posSlice = append(posSlice, winSlice...)
		sch <- posSlice
		close(sch)
	}()
	return sch
}

// simplified settle call
func (e *Engine) settleAll() ([]*types.SettlePosition, error) {
	// there should be as many positions as there are traders (obviously)
	aggregated := make([]*types.SettlePosition, 0, len(e.pos))
	// traders who are in the black should be appended (collect first) obvioulsy.
	// The split won't always be 50-50, but it's a reasonable approximation
	owed := make([]*types.SettlePosition, 0, len(e.pos)/2)
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
		settlePos := &types.SettlePosition{
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
			settlePos.Type = types.SettleType_LOSS
			aggregated = append(aggregated, settlePos)
		} else {
			// bad name again, but SELL means trader is owed money
			settlePos.Type = types.SettleType_WIN
			owed = append(owed, settlePos)
		}
	}
	// append the traders in the black to the end
	aggregated = append(aggregated, owed...)
	return aggregated, nil
}
