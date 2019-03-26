package internal

import (
	"strings"
	"sync"

	"code.vegaprotocol.io/vega/internal/blockchain"
	"code.vegaprotocol.io/vega/internal/candles"
	"code.vegaprotocol.io/vega/internal/logging"
	"code.vegaprotocol.io/vega/internal/markets"
	"code.vegaprotocol.io/vega/internal/orders"
	"code.vegaprotocol.io/vega/internal/parties"
	"code.vegaprotocol.io/vega/internal/storage"
	"code.vegaprotocol.io/vega/internal/trades"
	"code.vegaprotocol.io/vega/internal/vegatime"

	"github.com/pkg/errors"
)

// internal type used in CloseStores
type errStack []error

type Resolver struct {
	config          *Config
	onCriticalError func()

	candleStore storage.CandleStore
	orderStore  storage.OrderStore
	marketStore storage.MarketStore
	tradeStore  storage.TradeStore
	partyStore  storage.PartyStore
	riskStore   storage.RiskStore

	candleService candles.Service
	tradeService  trades.Service
	marketService markets.Service
	orderService  orders.Service
	partyService  parties.Service
	timeService   vegatime.Service

	blockchainClient blockchain.Client

	stMu sync.Mutex // Thread safety for singletons
	seMu sync.Mutex
	tsMu sync.Mutex
	bcMu sync.Mutex
}

// NewResolver initialises an instance of the VEGA resolver, this provides access to services and stores to help
// with a dependency graph. VEGA config implementation is required.
func NewResolver(config *Config, onCriticalError func()) (*Resolver, error) {
	if config == nil {
		return nil, errors.New("config instance is nil when calling NewResolver.")
	}
	return &Resolver{
		config:          config,
		onCriticalError: onCriticalError,
	}, nil
}

// ResolveLogger returns a pointer to a singleton instance of the debug/error logger. This instance of a logger is
// typically provided/injected into NewResolver at runtime.
func (r *Resolver) ResolveLogger() (*logging.Logger, error) {
	return r.config.log, nil
}

// -------------- Services/ --------------

// ResolveCandleService returns a pointer to a singleton instance of the candle service.
func (r *Resolver) ResolveCandleService() (candles.Service, error) {
	r.seMu.Lock()
	defer r.seMu.Unlock()

	if r.candleService != nil {
		return r.candleService, nil
	}

	candleStore, err := r.ResolveCandleStore()
	if err != nil {
		return nil, errors.Wrap(err, "error resolving candle store instance.")
	}

	candleService, err := candles.NewCandleService(
		r.config.Candles,
		candleStore,
	)
	if err != nil {
		return nil, err
	}

	r.candleService = candleService
	return r.candleService, nil
}

// ResolveMarketService returns a pointer to a singleton instance of the market service.
func (r *Resolver) ResolveMarketService() (markets.Service, error) {
	r.seMu.Lock()
	defer r.seMu.Unlock()

	if r.marketService != nil {
		return r.marketService, nil
	}

	marketStore, err := r.ResolveMarketStore()
	if err != nil {
		return nil, errors.Wrap(err, "error resolving market store instance.")
	}
	orderStore, err := r.ResolveOrderStore()
	if err != nil {
		return nil, errors.Wrap(err, "error resolving order store instance.")
	}

	marketService, err := markets.NewMarketService(
		r.config.Markets,
		marketStore,
		orderStore,
	)
	if err != nil {
		return nil, err
	}

	r.marketService = marketService
	return r.marketService, nil
}

// ResolvePartyService returns a pointer to a singleton instance of the party service.
func (r *Resolver) ResolvePartyService() (parties.Service, error) {
	r.seMu.Lock()
	defer r.seMu.Unlock()

	if r.partyService != nil {
		return r.partyService, nil
	}

	partyStore, err := r.ResolvePartyStore()
	if err != nil {
		return nil, errors.Wrap(err, "error resolving party store instance.")
	}

	partyService, err := parties.NewPartyService(
		r.config.Parties,
		partyStore,
	)
	if err != nil {
		return nil, err
	}

	r.partyService = partyService
	return r.partyService, nil
}

// ResolveOrderService returns a pointer to a singleton instance of the order service.
func (r *Resolver) ResolveOrderService() (orders.Service, error) {
	r.seMu.Lock()
	defer r.seMu.Unlock()

	if r.orderService != nil {
		return r.orderService, nil
	}

	orderStore, err := r.ResolveOrderStore()
	if err != nil {
		return nil, errors.Wrap(err, "error resolving order store instance.")
	}
	timeService, err := r.ResolveTimeService()
	if err != nil {
		return nil, errors.Wrap(err, "error resolving vega-time service instance.")
	}
	client, err := r.ResolveBlockchainClient()
	if err != nil {
		return nil, errors.Wrap(err, "error resolving blockchain client instance.")
	}

	orderService, err := orders.NewOrderService(
		r.config.Orders,
		orderStore,
		timeService,
		client,
	)
	if err != nil {
		return nil, err
	}

	r.orderService = orderService
	return r.orderService, nil
}

// ResolveTradeService returns a pointer to a singleton instance of the trade service.
func (r *Resolver) ResolveTradeService() (trades.Service, error) {
	r.seMu.Lock()
	defer r.seMu.Unlock()

	if r.tradeService != nil {
		return r.tradeService, nil
	}

	tradeStore, err := r.ResolveTradeStore()
	if err != nil {
		return nil, errors.Wrap(err, "error resolving trade store instance.")
	}
	riskStore, err := r.ResolveRiskStore()
	if err != nil {
		return nil, errors.Wrap(err, "error resolving risk store instance.")
	}

	tradeService, err := trades.NewTradeService(
		r.config.Trades,
		tradeStore,
		riskStore,
	)
	if err != nil {
		return nil, err
	}

	r.tradeService = tradeService
	return r.tradeService, nil
}

// ResolveTimeService returns a pointer to a singleton instance of the VEGA time service.
func (r *Resolver) ResolveTimeService() (vegatime.Service, error) {
	r.tsMu.Lock()
	defer r.tsMu.Unlock()

	if r.timeService != nil {
		return r.timeService, nil
	}

	r.timeService = vegatime.NewTimeService(
		r.config.Time,
	)
	return r.timeService, nil
}

// -------------- /Services --------------

// --------------- Storage/ --------------

// ResolveCandleStore returns a pointer to a singleton instance of the candle store.
func (r *Resolver) ResolveCandleStore() (storage.CandleStore, error) {
	r.stMu.Lock()
	defer r.stMu.Unlock()

	if r.candleStore != nil {
		return r.candleStore, nil
	}

	candleStore, err := storage.NewCandleStore(
		r.config.Storage,
	)
	if err != nil {
		return nil, err
	}

	r.candleStore = candleStore
	return r.candleStore, nil
}

// ResolveOrderStore returns a pointer to a singleton instance of the order store.
func (r *Resolver) ResolveOrderStore() (storage.OrderStore, error) {
	r.stMu.Lock()
	defer r.stMu.Unlock()

	if r.orderStore != nil {
		return r.orderStore, nil
	}

	orderStore, err := storage.NewOrderStore(
		r.config.Storage, r.onCriticalError,
	)
	if err != nil {
		return nil, err
	}

	r.orderStore = orderStore
	return r.orderStore, nil
}

// ResolveTradeStore returns a pointer to a singleton instance of the trade store.
func (r *Resolver) ResolveTradeStore() (storage.TradeStore, error) {
	r.stMu.Lock()
	defer r.stMu.Unlock()

	if r.tradeStore != nil {
		return r.tradeStore, nil
	}

	tradeStore, err := storage.NewTradeStore(
		r.config.Storage, r.onCriticalError,
	)
	if err != nil {
		return nil, err
	}

	r.tradeStore = tradeStore
	return r.tradeStore, nil
}

// ResolveRiskStore returns a pointer to a singleton instance of the risk store.
func (r *Resolver) ResolveRiskStore() (storage.RiskStore, error) {
	r.stMu.Lock()
	defer r.stMu.Unlock()

	if r.riskStore != nil {
		return r.riskStore, nil
	}

	riskStore, err := storage.NewRiskStore(
		r.config.Storage,
	)
	if err != nil {
		return nil, err
	}

	r.riskStore = riskStore
	return r.riskStore, nil
}

// ResolveMarketStore returns a pointer to a singleton instance of the market store.
func (r *Resolver) ResolveMarketStore() (storage.MarketStore, error) {
	r.stMu.Lock()
	defer r.stMu.Unlock()

	if r.marketStore != nil {
		return r.marketStore, nil
	}

	marketStore, err := storage.NewMarketStore(
		r.config.Storage,
	)
	if err != nil {
		return nil, err
	}

	r.marketStore = marketStore
	return r.marketStore, nil
}

// ResolvePartyStore returns a pointer to a singleton instance of the party store.
func (r *Resolver) ResolvePartyStore() (storage.PartyStore, error) {
	r.stMu.Lock()
	defer r.stMu.Unlock()

	if r.partyStore != nil {
		return r.partyStore, nil
	}

	partyStore, err := storage.NewPartyStore(
		r.config.Storage,
	)
	if err != nil {
		return nil, err
	}

	r.partyStore = partyStore
	return r.partyStore, nil
}

// CloseStores is a helper utility that aids in cleaning up the storage layer on application shutdown etc.
// Typically run with defer, or at the end of the app lifecycle.
func (r *Resolver) CloseStores() error {
	var werr errStack
	r.stMu.Lock()

	if r.candleStore != nil {
		if err := r.candleStore.Close(); err != nil {
			werr = append(werr, errors.Wrap(err, "error closing candle store in resolver."))
		}
	}
	if r.riskStore != nil {
		if err := r.riskStore.Close(); err != nil {
			werr = append(werr, errors.Wrap(err, "error closing risk store in resolver."))
		}
	}
	if r.tradeStore != nil {
		if err := r.tradeStore.Close(); err != nil {
			werr = append(werr, errors.Wrap(err, "error closing trade store in resolver."))
		}
	}
	if r.orderStore != nil {
		if err := r.orderStore.Close(); err != nil {
			werr = append(werr, errors.Wrap(err, "error closing order store in resolver."))
		}
	}
	if r.marketStore != nil {
		if err := r.marketStore.Close(); err != nil {
			werr = append(werr, errors.Wrap(err, "error closing market store in resolver."))
		}
	}
	if r.partyStore != nil {
		if err := r.partyStore.Close(); err != nil {
			werr = append(werr, errors.Wrap(err, "error closing party store in resolver."))
		}
	}

	r.stMu.Unlock()

	return werr
}

// --------------- /Storage --------------

// -------------- Blockchain/ -------------

// ResolveBlockchainClient returns a a singleton instance of the (Tendermint) blockchain client.
func (r *Resolver) ResolveBlockchainClient() (blockchain.Client, error) {
	r.bcMu.Lock()
	defer r.bcMu.Unlock()

	if r.blockchainClient != nil {
		return r.blockchainClient, nil
	}

	client, err := blockchain.NewClient(
		r.config.Blockchain,
	)
	if err != nil {
		return nil, err
	}

	r.blockchainClient = client
	return r.blockchainClient, nil
}

// -------------- /Blockchain -------------

// Error - implement the error interface on the errStack type
func (e errStack) Error() string {
	s := make([]string, 0, len(e))
	for _, err := range e {
		s = append(s, err.Error())
	}
	return strings.Join(s, "\n")
}
