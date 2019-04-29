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
	pos     map[string]*pos
}

func New(log *logging.Logger, conf Config) *Engine {
	// setup logger
	log = log.Named(namedLogger)
	log.SetLevel(conf.Level.Get())

	return &Engine{
		log:    log,
		Config: conf,
		mu:     &sync.Mutex{},
		pos:    map[string]*pos{},
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
		ps, ok := e.pos[party]
		// create entry for trader if needed
		// if not, just update with new net position
		if !ok {
			ps := &pos{}
			e.pos[party] = ps
		}
		ps.price = p.Price()
		ps.size = p.Size()
	}
	e.mu.Unlock()
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

// simplified settle call
func (e *Engine) settleAll() ([]*types.SettlePosition, error) {
	// there should be as many positions as there are traders (obviously)
	aggregated := make([]*types.SettlePosition, 0, len(e.pos))
	// traders who are in the black should be appended (collect first) obvioulsy.
	// The split won't always be 50-50, but it's a reasonable approximation
	owed := make([]*types.SettlePosition, 0, len(e.pos)/2)
	for party, pos := range e.pos {
		e.log.Debug("Settling position for trader", logging.String("trader-id", party))
		if pos.size == 0 {
			e.log.Debug(
				"Trader has a net size/position of 0, default to 1",
				logging.String("trader-id", party),
				logging.Uint64("price", pos.price),
			)
			// we should have this happen on close, or ever, really -> the amount could be - or +
			// this is just so we can ensure the division and multiplication isn't going to fail
			pos.size = 1
		}
		// @TODO positions should take care of this
		netPrice := int64(pos.price) / pos.size
		if netPrice < 0 {
			// get abs value
			netPrice *= -1
		}
		amt, err := e.product.Settle(uint64(netPrice), pos.size)
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
			Size:   uint64(pos.size),
			Amount: amt,
			Type:   types.SettleType_LOSS, // this is a poor name, will be changed later
		}
		e.log.Debug(
			"Settled position for trader",
			logging.String("trader-id", party),
			logging.Int64("amount", amt.Amount),
		)
		if amt.Amount < 0 {
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
