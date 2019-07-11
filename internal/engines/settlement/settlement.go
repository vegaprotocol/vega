package settlement

import (
	"sync"
	"time"

	"code.vegaprotocol.io/vega/internal/engines/position"
	"code.vegaprotocol.io/vega/internal/logging"
	"code.vegaprotocol.io/vega/internal/products"
	types "code.vegaprotocol.io/vega/proto"
)

type pos struct {
	size  int64
	price uint64
}

type Engine struct {
	log *logging.Logger

	Config
	mu      *sync.Mutex
	product products.Product
	buys    map[string]*pos
	sells   map[string]*pos
}

func New(log *logging.Logger, conf Config) *Engine {
	// setup logger
	log = log.Named(namedLogger)
	log.SetLevel(conf.Level.Get())

	return &Engine{
		log:    log,
		Config: conf,
		mu:     &sync.Mutex{},
		buys:   map[string]*pos{},
		sells:  map[string]*pos{},
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
func (e *Engine) Update(positions []position.MarketPosition) {
	e.mu.Lock()
	for _, p := range positions {
		party := p.Party()
		size := p.Size()
		if size > 0 {
			ps, ok := e.buys[party]
			if !ok {
				ps = &pos{}
				e.buys[party] = ps
			}
			// price and size are running totals
			ps.size = size
			ps.price = p.Price()
		} else {
			ps, ok := e.sells[party]
			if !ok {
				ps = &pos{}
				e.sells[party] = ps
			}
			// price and size are running totals
			ps.size = size
			ps.price = p.Price()
		}
	}
	e.mu.Unlock()
}

func (e *Engine) Settle(t time.Time) ([]*types.SettlePosition, error) {
	e.mu.Lock()
	e.log.Debugf("Settling market, closed at %s", t.Format(time.RFC3339))
	buy, err := e.settleBuy()
	if err != nil {
		e.log.Error(
			"Something went wrong trying to settle buy positions",
			logging.Error(err),
		)
		return nil, err
	}
	sell, err := e.settleSell()
	if err != nil {
		e.log.Error(
			"Something went wrong trying to settle sell positions",
			logging.Error(err),
		)
		return nil, err
	}
	e.mu.Unlock()
	// alloc when needed, when buy + sell were both succesful, and we know the length
	// well, we know the length == len(e.buys) + len(e.sells), but hey...
	positions := make([]*types.SettlePosition, 0, len(buy)+len(sell))
	// merge into single slice
	positions = append(positions, buy...)
	positions = append(positions, sell...)
	return positions, nil
}

func (e *Engine) settleBuy() ([]*types.SettlePosition, error) {
	// mu is locked here
	r := make([]*types.SettlePosition, 0, len(e.buys))
	for party, bpos := range e.buys {
		e.log.Debugf("Settling buys for %s", party)
		netPrice := int64(bpos.price) / bpos.size
		amt, err := e.product.Settle(uint64(netPrice), uint64(bpos.size))
		if err != nil {
			e.log.Error(
				"Failed to settle buy position for trader",
				logging.String("trader-id", party),
				logging.Error(err),
			)
			return nil, err
		}
		r = append(r, &types.SettlePosition{
			Owner: party,
			Size:  uint64(bpos.size),
			Amount: &types.FinancialAmount{
				Amount: amt.Amount,
				Asset:  amt.Asset,
			},
			Type: types.SettleType_BUY,
		})
		e.log.Debugf("Settling %d buys at average price: %d", bpos.size, netPrice)
	}
	return r, nil
}

func (e *Engine) settleSell() ([]*types.SettlePosition, error) {
	r := make([]*types.SettlePosition, 0, len(e.sells))
	for party, spos := range e.sells {
		e.log.Debugf("Settling sales for %s", party)
		netPrice := int64(spos.price) / (-spos.size)
		amt, err := e.product.Settle(uint64(netPrice), uint64(-spos.size))
		if err != nil {
			e.log.Error(
				"Failed to settle sell position for trader",
				logging.String("trader-id", party),
				logging.Error(err),
			)
			return nil, err
		}
		r = append(r, &types.SettlePosition{
			Owner: party,
			Size:  uint64(-spos.size),
			Amount: &types.FinancialAmount{
				Amount: amt.Amount,
				Asset:  amt.Asset,
			},
			Type: types.SettleType_SELL,
		})
		e.log.Debugf("Settling %d sales at average price: %d", spos.size, netPrice)
	}
	return r, nil
}
