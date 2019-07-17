package posres

import (
	"sync"

	"code.vegaprotocol.io/vega/internal/events"
	"code.vegaprotocol.io/vega/internal/logging"
	types "code.vegaprotocol.io/vega/proto"
)

type Engine struct {
	Config
	log      *logging.Logger
	mu       *sync.Mutex
	marketID string
}

func New(log *logging.Logger, conf Config, marketID string) *Engine {
	log = log.Named(namedLogger)
	log.SetLevel(conf.Level.Get())
	return &Engine{
		Config:   conf,
		log:      log,
		mu:       &sync.Mutex{},
		marketID: marketID,
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

	e.mu.Lock()
	e.Config = cfg
	e.mu.Unlock()
}

func (e *Engine) CloseDistressed(traders []events.MarketPosition) (*types.Order, error) {
	var (
		net, buy, sell int64
		avgPrice       uint64
	)
	for _, t := range traders {
		s := t.Size()
		p := t.Price() // price, will be summed by volume
		net += s
		if s > 0 {
			sell += s
			avgPrice += p * uint64(s)
		} else {
			buy += s
			avgPrice += p * uint64(-s)
		}
	}
	avgPrice /= (-buy + sell) // volume weighted average price
	if net == 0 {
		e.log.Debug(
			"No market trades with non-distressed traders required",
			logging.String("market-id", e.marketID),
		)
		return nil, nil
	}
	e.log.Debug(
		"A network trade with regular traders required",
		logging.Int64("order-volume", net),
		logging.Int64("total-buy", buy),
		logging.Int64("total-sell", sell),
		logging.String("market-id", e.marketID),
	)
	// no party ID or order ID -> this is a network order
	order := types.Order{
		MarketID:  e.marketID,
		Price:     avgPrice,
		Size:      uint64(net),
		Type:      types.Order_Type_NDT,
		Reference: "network trade",
	}
	if net > 0 {
		order.Side = types.Side_Sell
	} else {
		order.Side = types.Side_Buy
	}
	return &order, nil
}
