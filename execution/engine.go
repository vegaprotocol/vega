package execution

import (
	"time"

	"code.vegaprotocol.io/vega/collateral"
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/metrics"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/pkg/errors"
)

var (
	// ErrMarketAlreadyExist signals that a market already exist
	ErrMarketAlreadyExist = errors.New("market already exist")

	// ErrUnknownProposalChange is returned if passed proposal cannot be enacted
	// because proposed changes cannot be processed by the system
	ErrUnknownProposalChange = errors.New("unknown proposal change")
)

// OrderBuf ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/order_buf_mock.go -package mocks code.vegaprotocol.io/vega/execution OrderBuf
type OrderBuf interface {
	Add(types.Order)
	Flush() error
}

// TradeBuf ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/trade_buf_mock.go -package mocks code.vegaprotocol.io/vega/execution TradeBuf
type TradeBuf interface {
	Add(types.Trade)
	Flush() error
}

// CandleBuf ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/candle_buf_mock.go -package mocks code.vegaprotocol.io/vega/execution CandleBuf
type CandleBuf interface {
	AddTrade(types.Trade) error
	Flush(marketID string, t time.Time) error
	Start(marketID string, t time.Time) (map[string]types.Candle, error)
}

// MarketBuf ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/market_buf_mock.go -package mocks code.vegaprotocol.io/vega/execution MarketBuf
type MarketBuf interface {
	Add(types.Market)
	Flush() error
}

// PartyBuf ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/party_buf_mock.go -package mocks code.vegaprotocol.io/vega/execution PartyBuf
type PartyBuf interface {
	Add(types.Party)
	Flush() error
}

// SettlementBuf ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/settlement_buf_mock.go -package mocks code.vegaprotocol.io/vega/execution SettlementBuf
type SettlementBuf interface {
	Add([]events.SettlePosition)
	Flush()
}

// TimeService ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/time_service_mock.go -package mocks code.vegaprotocol.io/vega/execution TimeService
type TimeService interface {
	GetTimeNow() (time.Time, error)
	NotifyOnTick(f func(time.Time))
}

// TransferBuf ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/transfer_buf_mock.go -package mocks code.vegaprotocol.io/vega/execution TransferBuf
type TransferBuf interface {
	Add([]*types.TransferResponse)
	Flush() error
}

// AccountBuf ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/account_buf_mock.go -package mocks code.vegaprotocol.io/vega/execution AccountBuf
type AccountBuf interface {
	Add(types.Account)
	Flush() error
}

// MarketDataBuf ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/market_data_buf_mock.go -package mocks code.vegaprotocol.io/vega/execution MarketDataBuf
type MarketDataBuf interface {
	Add(types.MarketData)
	Flush()
}

// MarginLevelsBuf ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/margin_levels_buf_mock.go -package mocks code.vegaprotocol.io/vega/execution MarginLevelsBuf
type MarginLevelsBuf interface {
	Add(types.MarginLevels)
	Flush()
}

// LossSocializationBuf ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/loss_socialization_buf_mock.go -package mocks code.vegaprotocol.io/vega/execution LossSocializationBuf
type LossSocializationBuf interface {
	Add([]events.LossSocialization)
	Flush()
}

// Engine is the execution engine
type Engine struct {
	Config
	log *logging.Logger

	markets    map[string]*Market
	party      *Party
	collateral *collateral.Engine
	idgen      *IDgenerator

	orderBuf        OrderBuf
	tradeBuf        TradeBuf
	candleBuf       CandleBuf
	marketBuf       MarketBuf
	partyBuf        PartyBuf
	accountBuf      AccountBuf
	transferBuf     TransferBuf
	marketDataBuf   MarketDataBuf
	marginLevelsBuf MarginLevelsBuf
	settleBuf       SettlementBuf
	lossSocBuf      LossSocializationBuf

	time TimeService
}

// NewEngine takes stores and engines and returns
// a new execution engine to process new orders, etc.
func NewEngine(
	log *logging.Logger,
	executionConfig Config,
	time TimeService,
	orderBuf OrderBuf,
	tradeBuf TradeBuf,
	candleBuf CandleBuf,
	marketBuf MarketBuf,
	partyBuf PartyBuf,
	accountBuf AccountBuf,
	transferBuf TransferBuf,
	marketDataBuf MarketDataBuf,
	marginLevelsBuf MarginLevelsBuf,
	settleBuf SettlementBuf,
	lossSocBuf LossSocializationBuf,
	pmkts []types.Market,
	collateral *collateral.Engine,
) *Engine {
	// setup logger
	log = log.Named(namedLogger)
	log.SetLevel(executionConfig.Level.Get())

	e := &Engine{
		log:             log,
		Config:          executionConfig,
		markets:         map[string]*Market{},
		candleBuf:       candleBuf,
		orderBuf:        orderBuf,
		tradeBuf:        tradeBuf,
		marketBuf:       marketBuf,
		partyBuf:        partyBuf,
		time:            time,
		collateral:      collateral,
		party:           NewParty(log, collateral, pmkts, partyBuf),
		accountBuf:      accountBuf,
		transferBuf:     transferBuf,
		marketDataBuf:   marketDataBuf,
		marginLevelsBuf: marginLevelsBuf,
		settleBuf:       settleBuf,
		lossSocBuf:      lossSocBuf,
		idgen:           NewIDGen(),
	}

	var err error
	// Add initial markets and flush to stores (if they're configured)
	if len(pmkts) > 0 {
		for _, mkt := range pmkts {
			mkt := mkt
			err = e.SubmitMarket(&mkt)
			if err != nil {
				e.log.Panic("Unable to submit market",
					logging.Error(err))
			}
		}
		if err := e.marketBuf.Flush(); err != nil {
			e.log.Error("unable to flush markets", logging.Error(err))
			return nil
		}
	}

	// Add time change event handler
	e.time.NotifyOnTick(e.onChainTimeUpdate)

	return e
}

// ReloadConf updates the internal configuration of the execution
// engine and its dependencies
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
	for _, mkt := range e.markets {
		mkt.ReloadConf(e.Config.Matching, e.Config.Risk,
			e.Config.Collateral, e.Config.Position, e.Config.Settlement)
	}
}

// NotifyTraderAccount notify the engine to create a new account for a party
func (e *Engine) NotifyTraderAccount(notify *types.NotifyTraderAccount) error {
	return e.party.NotifyTraderAccount(notify)
}

// CreateGeneralAccounts creates new general accounts for a party
func (e *Engine) CreateGeneralAccounts(partyID string) error {
	_, err := e.party.MakeGeneralAccounts(partyID)
	return err
}

func (e *Engine) Withdraw(w *types.Withdraw) error {
	err := e.collateral.Withdraw(w.PartyID, w.Asset, w.Amount)
	if err != nil {
		e.log.Error("An error occurred during withdrawal",
			logging.String("party-id", w.PartyID),
			logging.Uint64("amount", w.Amount),
			logging.Error(err),
		)
	}
	return err
}

// SubmitMarket will submit a new market configuration to the network
func (e *Engine) SubmitMarket(marketConfig *types.Market) error {

	// TODO: Check for existing market in MarketStore by Name.
	// if __TBC_MarketExists__(marketConfig.Name) {
	// 	return ErrMarketAlreadyExist
	// }

	var mkt *Market
	var err error

	now, _ := e.time.GetTimeNow()
	mkt, err = NewMarket(
		e.log,
		e.Config.Risk,
		e.Config.Position,
		e.Config.Settlement,
		e.Config.Matching,
		e.collateral,
		e.party,
		marketConfig,
		e.candleBuf,
		e.orderBuf,
		e.partyBuf,
		e.tradeBuf,
		e.transferBuf,
		e.marginLevelsBuf,
		e.settleBuf,
		now,
		e.idgen,
	)
	if err != nil {
		e.log.Error("Failed to instantiate market",
			logging.String("market-name", marketConfig.GetName()),
			logging.Error(err),
		)
	}

	e.markets[marketConfig.Id] = mkt

	// create market accounts
	asset, err := marketConfig.GetAsset()
	if err != nil {
		return err
	}

	// ignore response ids here + this cannot fail
	_, _ = e.collateral.CreateMarketAccounts(marketConfig.Id, asset, e.Config.InsurancePoolInitialBalance)

	// wire up party engine to new market
	e.party.addMarket(*mkt.mkt)
	e.markets[mkt.mkt.Id].partyEngine = e.party

	// Save to market proto to buffer
	e.marketBuf.Add(*marketConfig)
	return nil
}

// SubmitOrder checks the incoming order and submits it to a Vega market.
func (e *Engine) SubmitOrder(order *types.Order) (*types.OrderConfirmation, error) {
	timer := metrics.NewTimeCounter(order.MarketID, "execution", "SubmitOrder")

	if e.log.GetLevel() == logging.DebugLevel {
		e.log.Debug("Submit order", logging.Order(*order))
	}

	mkt, ok := e.markets[order.MarketID]
	if !ok {
		t, err := e.time.GetTimeNow()
		if err != nil {
			order.CreatedAt = t.UnixNano()
		}
		e.idgen.SetID(order)

		// adding rejected order to the buf
		order.Status = types.Order_Rejected
		order.Reason = types.OrderError_INVALID_MARKET_ID
		e.orderBuf.Add(*order)

		timer.EngineTimeCounterAdd()
		return nil, types.ErrInvalidMarketID
	}

	if order.Status == types.Order_Active {
		metrics.OrderGaugeAdd(1, order.MarketID)
	}

	conf, err := mkt.SubmitOrder(order)
	if err != nil {
		timer.EngineTimeCounterAdd()
		return nil, err
	}

	if conf.Order.Status == types.Order_Filled {
		metrics.OrderGaugeAdd(-1, order.MarketID)
	}

	timer.EngineTimeCounterAdd()
	return conf, nil
}

// AmendOrder takes order amendment details and attempts to amend the order
// if it exists and is in a editable state.
func (e *Engine) AmendOrder(orderAmendment *types.OrderAmendment) (*types.OrderConfirmation, error) {
	if e.log.GetLevel() == logging.DebugLevel {
		e.log.Debug("Amend order",
			logging.String("order-id", orderAmendment.GetOrderID()),
			logging.String("party-id", orderAmendment.GetPartyID()),
			logging.String("market-id", orderAmendment.GetMarketID()),
			logging.Uint64("price", orderAmendment.GetPrice().Value),
			logging.Int64("sizeDelta", orderAmendment.GetSizeDelta()),
			logging.String("tif", orderAmendment.GetTimeInForce().String()),
			logging.Int64("expires-at", orderAmendment.GetExpiresAt().Value),
		)
	}

	mkt, ok := e.markets[orderAmendment.MarketID]
	if !ok {
		return nil, types.ErrInvalidMarketID
	}

	// we're passing a pointer here, so we need the wasActive var to be certain we're checking the original
	// order status. It's possible order.Status will reflect the new status value if we don't
	conf, err := mkt.AmendOrder(orderAmendment)
	if err != nil {
		return nil, err
	}
	// order was active, not anymore -> decrement gauge
	if conf.Order.Status != types.Order_Active {
		metrics.OrderGaugeAdd(-1, orderAmendment.MarketID)
	}
	return conf, nil
}

// CancelOrder takes order details and attempts to cancel if it exists in matching engine, stores etc.
func (e *Engine) CancelOrder(order *types.OrderCancellation) (*types.OrderCancellationConfirmation, error) {
	if e.log.GetLevel() == logging.DebugLevel {
		e.log.Debug("Cancel order", logging.String("order-id", order.OrderID))
	}
	mkt, ok := e.markets[order.MarketID]
	if !ok {
		return nil, types.ErrInvalidMarketID
	}
	conf, err := mkt.CancelOrder(order)
	if err != nil {
		return nil, err
	}
	if conf.Order.Status == types.Order_Cancelled {
		metrics.OrderGaugeAdd(-1, order.MarketID)
	}
	return conf, nil
}

// CancelOrderByID attempts to locate order by its Id and cancel it if exists.
func (e *Engine) CancelOrderByID(orderID string, marketID string) (*types.OrderCancellationConfirmation, error) {
	if e.log.GetLevel() == logging.DebugLevel {
		e.log.Debug("Cancel order by id", logging.String("order-id", orderID))
	}
	mkt, ok := e.markets[marketID]
	if !ok {
		return nil, types.ErrInvalidMarketID
	}
	conf, err := mkt.CancelOrderByID(orderID)
	if err != nil {
		return nil, err
	}
	if conf.Order.Status == types.Order_Cancelled {
		metrics.OrderGaugeAdd(-1, marketID)
	}
	return conf, nil
}

func (e *Engine) onChainTimeUpdate(t time.Time) {
	timer := metrics.NewTimeCounter("-", "execution", "onChainTimeUpdate")

	// update block time on id generator
	e.idgen.NewBatch()

	e.log.Debug("updating engine on new time update")

	// update collateral
	e.collateral.OnChainTimeUpdate(t)

	// remove expired orders
	// TODO(FIXME): this should be remove, and handled inside the market directly
	// when call with the new time (see the next for loop)
	e.removeExpiredOrders(t)

	// notify markets of the time expiration
	for mktID, mkt := range e.markets {
		mkt := mkt
		closing := mkt.OnChainTimeUpdate(t)
		if closing {
			e.log.Info("market is closed, removing from execution engine",
				logging.String("market-id", mktID))
			delete(e.markets, mktID)
		}
	}
	timer.EngineTimeCounterAdd()
}

func (e *Engine) EnactProposal(proposal *types.Proposal) error {
	if newMarket := proposal.Terms.GetNewMarket(); newMarket != nil {
		if e.log.GetLevel() == logging.DebugLevel {
			e.log.Debug("enacting proposal", logging.String("proposal-id", proposal.ID))
		}
		if err := e.SubmitMarket(newMarket.Changes); err != nil {
			return err
		}
		proposal.State = types.Proposal_ENACTED
		return nil
	} else if updateMarket := proposal.Terms.GetUpdateMarket(); updateMarket != nil {

		return errors.New("update market enactment is not implemented")
	} else if updateNetwork := proposal.Terms.GetUpdateNetwork(); updateNetwork != nil {

		return errors.New("update network enactment is not implemented")
	}
	// This error shouldn't be possible here,if we reach this point the governance engine
	// has failed to perform the correct validation on the proposal itself
	return ErrUnknownProposalChange
}

// Process any data updates (including state changes)
// e.g. removing expired orders from matching engine.
func (e *Engine) removeExpiredOrders(t time.Time) {
	timer := metrics.NewTimeCounter("-", "execution", "removeExpiredOrders")
	if e.log.GetLevel() == logging.DebugLevel {
		e.log.Debug("Removing expiring orders from matching engine")
	}
	expiringOrders := []types.Order{}
	timeNow := t.UnixNano()
	for _, mkt := range e.markets {
		orders, err := mkt.RemoveExpiredOrders(timeNow)
		if err != nil {
			e.log.Error("unable to get remove expired orders",
				logging.String("market-id", mkt.GetID()),
				logging.Error(err))
		}
		expiringOrders = append(
			expiringOrders, orders...)
	}
	if e.log.GetLevel() == logging.DebugLevel {
		e.log.Debug("Removed expired orders from matching engine",
			logging.Int("orders-removed", len(expiringOrders)))
	}
	for _, order := range expiringOrders {
		order := order
		e.orderBuf.Add(order)
		metrics.OrderGaugeAdd(-1, order.MarketID) // decrement gauge
	}
	if e.log.GetLevel() == logging.DebugLevel {
		e.log.Debug("Updated expired orders in stores",
			logging.Int("orders-removed", len(expiringOrders)))
	}
	timer.EngineTimeCounterAdd()
}

func (e *Engine) GetMarketData(mktid string) (types.MarketData, error) {
	mkt, ok := e.markets[mktid]
	if !ok {
		return types.MarketData{}, types.ErrInvalidMarketID
	}
	return mkt.GetMarketData(), nil
}

// Generate flushes any data (including storing state changes) to underlying stores (if configured).
func (e *Engine) Generate() error {

	// Accounts
	err := e.accountBuf.Flush()
	if err != nil {
		return errors.Wrap(err, "Failed to flush accounts buffer")
	}

	// margins levels
	e.marginLevelsBuf.Flush()
	// Orders
	err = e.orderBuf.Flush()
	if err != nil {
		return errors.Wrap(err, "Failed to flush orders buffer")
	}

	// Trades - flush after orders so the traders reference an existing order
	err = e.tradeBuf.Flush()
	if err != nil {
		return errors.Wrap(err, "Failed to flush trades buffer")
	}
	// Transfers
	err = e.transferBuf.Flush()
	if err != nil {
		return errors.Wrap(err, "Failed to flush transfers buffer")
	}
	// Markets
	err = e.marketBuf.Flush()
	if err != nil {
		return errors.Wrap(err, "Failed to flush markets buffer")
	}
	// Market data is added to buffer on Generate
	for _, v := range e.markets {
		e.marketDataBuf.Add(v.GetMarketData())
	}
	e.marketDataBuf.Flush()
	// Parties
	_ = e.partyBuf.Flush() // JL: do not check errors here as they only happened when a party is created

	return nil
}
