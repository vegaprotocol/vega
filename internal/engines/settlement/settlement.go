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

// MarkToMarket - read settle positions from channel (populated by positions engine), then notify through done channel
// these settle positions aren't going to update the positions tracked here in the settlement engine
// instead the net result will be a a slice where losses are prepended, and wins appended
// this slice will be pushed onto the return channel
func (e *Engine) MarkToMarket(ch <-chan *types.SettlePosition) <-chan []*types.SettlePosition {
	// indicate the settlement engine has processed everything in the channel via waitgroup, market framework is closing the channel after the loop
	sch := make(chan []*types.SettlePosition) // no buffer, so the read on this channel will be blocking. Once we've read the slice, we know the work here is done
	// read the channel, writing to the return channel indicates the routine is done, and the channel is closed
	go func() {
		// set buffer of channel as cap of slice, reason for this pre-allocation is the same as why we buffer the channel to a given size
		posSlice := make([]*types.SettlePosition, 0, cap(ch))
		winSlice := make([]*types.SettlePosition, 0, cap(ch)/2) // assuming half of these will be wins (not a given, but it's a decent enough cap)
		for pos := range ch {
			if pos.Type == types.SettleType_MTM_LOSS {
				posSlice = append(posSlice, pos)
			} else {
				winSlice = append(winSlice, pos)
			}
		}
		// create a single slice here, losses first, wins after
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
		if pos.size < 0 {
			// ensure absolute value
			settlePos.Size = uint64(-pos.size)
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
