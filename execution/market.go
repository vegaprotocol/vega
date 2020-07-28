package execution

import (
	"context"
	"crypto/sha256"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"math"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/collateral"
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/fee"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/markets"
	"code.vegaprotocol.io/vega/matching"
	"code.vegaprotocol.io/vega/metrics"
	"code.vegaprotocol.io/vega/positions"
	types "code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/risk"
	"code.vegaprotocol.io/vega/settlement"

	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
)

// InitialOrderVersion is set on `Version` field for every new order submission read from the network
const InitialOrderVersion = 1

var (
	// ErrMarketClosed signals that an action have been tried to be applied on a closed market
	ErrMarketClosed = errors.New("market closed")
	// ErrTraderDoNotExists signals that the trader used does not exists
	ErrTraderDoNotExists = errors.New("trader does not exist")
	// ErrMarginCheckFailed signals that a margin check for a position failed
	ErrMarginCheckFailed = errors.New("margin check failed")
	// ErrMarginCheckInsufficient signals that a margin had not enough funds
	ErrMarginCheckInsufficient = errors.New("insufficient margin")
	// ErrInvalidInitialMarkPrice signals that the initial mark price for a market is invalid
	ErrInvalidInitialMarkPrice = errors.New("invalid initial mark price (mkprice <= 0)")
	// ErrMissingGeneralAccountForParty ...
	ErrMissingGeneralAccountForParty = errors.New("missing general account for party")
	// ErrNotEnoughVolumeToZeroOutNetworkOrder ...
	ErrNotEnoughVolumeToZeroOutNetworkOrder = errors.New("not enough volume to zero out network order")
	// ErrInvalidAmendRemainQuantity signals incorrect remaining qty for a reduce by amend
	ErrInvalidAmendRemainQuantity = errors.New("incorrect remaining qty for a reduce by amend")
	// ErrEmptyMarketID is returned if processed market has an empty id
	ErrEmptyMarketID = errors.New("invalid market id (empty)")
	// ErrInvalidOrderType is returned if processed order has an invalid order type
	ErrInvalidOrderType = errors.New("invalid order type")
	// ErrInvalidExpiresAtTime is returned if the expire time is before the createdAt time
	ErrInvalidExpiresAtTime = errors.New("invalid expiresAt time")

	networkPartyID = "network"
)

// Market represents an instance of a market in vega and is in charge of calling
// the engines in order to process all transctiona
type Market struct {
	log   *logging.Logger
	idgen *IDgenerator

	matchingConfig matching.Config

	mkt         *types.Market
	closingAt   time.Time
	currentTime time.Time
	mu          sync.Mutex

	markPrice uint64

	// own engines
	matching           *matching.OrderBook
	tradableInstrument *markets.TradableInstrument
	risk               *risk.Engine
	position           *positions.Engine
	settlement         *settlement.Engine
	fee                *fee.Engine

	// deps engines
	collateral  *collateral.Engine
	partyEngine *Party

	broker Broker
	closed bool

	auction    bool
	auctionEnd time.Time
}

// SetMarketID assigns a deterministic pseudo-random ID to a Market
func SetMarketID(marketcfg *types.Market, seq uint64) error {
	marketcfg.Id = ""
	marketbytes, err := proto.Marshal(marketcfg)
	if err != nil {
		return err
	}
	if len(marketbytes) == 0 {
		return errors.New("failed to marshal market")
	}

	seqbytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(seqbytes, seq)

	h := sha256.New()
	h.Write(marketbytes)
	h.Write(seqbytes)

	d := h.Sum(nil)
	d = d[:20]
	marketcfg.Id = base32.StdEncoding.EncodeToString(d)
	return nil
}

// NewMarket creates a new market using the market framework configuration and creates underlying engines.
func NewMarket(
	log *logging.Logger,
	riskConfig risk.Config,
	positionConfig positions.Config,
	settlementConfig settlement.Config,
	matchingConfig matching.Config,
	feeConfig fee.Config,
	collateralEngine *collateral.Engine,
	partyEngine *Party,
	mkt *types.Market,
	now time.Time,
	broker Broker,
	idgen *IDgenerator,
) (*Market, error) {

	if len(mkt.Id) == 0 {
		return nil, ErrEmptyMarketID
	}

	tradableInstrument, err := markets.NewTradableInstrument(log, mkt.TradableInstrument)
	if err != nil {
		return nil, errors.Wrap(err, "unable to instantiate a new market")
	}

	if tradableInstrument.Instrument.InitialMarkPrice == 0 {
		return nil, ErrInvalidInitialMarkPrice
	}

	closingAt, err := tradableInstrument.Instrument.GetMarketClosingTime()
	if err != nil {
		return nil, errors.Wrap(err, "unable to get market closing time")
	}

	book := matching.NewOrderBook(log, matchingConfig, mkt.Id,
		tradableInstrument.Instrument.InitialMarkPrice)
	asset := tradableInstrument.Instrument.Product.GetAsset()
	riskEngine := risk.NewEngine(
		log,
		riskConfig,
		tradableInstrument.MarginCalculator,
		tradableInstrument.RiskModel,
		getInitialFactors(log, mkt, asset),
		book,
		broker,
		now.UnixNano(),
		mkt.GetId(),
	)
	settleEngine := settlement.New(
		log,
		settlementConfig,
		tradableInstrument.Instrument.Product,
		mkt.Id,
		broker,
	)
	positionEngine := positions.New(log, positionConfig)

	feeEngine, err := fee.New(log, feeConfig, *mkt.Fees, asset)
	if err != nil {
		return nil, errors.Wrap(err, "unable to instanciate fee engine")
	}

	market := &Market{
		log:                log,
		idgen:              idgen,
		mkt:                mkt,
		closingAt:          closingAt,
		currentTime:        now,
		markPrice:          tradableInstrument.Instrument.InitialMarkPrice,
		matching:           book,
		tradableInstrument: tradableInstrument,
		risk:               riskEngine,
		position:           positionEngine,
		settlement:         settleEngine,
		collateral:         collateralEngine,
		partyEngine:        partyEngine,
		broker:             broker,
		fee:                feeEngine,
		auctionEnd:         now.Add(time.Duration(mkt.OpeningAuction.Duration) * time.Second),
	}
	if market.auctionEnd.After(now) {
		market.auction = true
	}
	return market, nil
}

func (m *Market) GetMarketData() types.MarketData {
	bestBidPrice, bestBidVolume := m.matching.BestBidPriceAndVolume()
	bestOfferPrice, bestOfferVolume := m.matching.BestOfferPriceAndVolume()
	return types.MarketData{
		Market:          m.GetID(),
		BestBidPrice:    bestBidPrice,
		BestBidVolume:   bestBidVolume,
		BestOfferPrice:  bestOfferPrice,
		BestOfferVolume: bestOfferVolume,
		MidPrice:        (bestBidPrice + bestOfferPrice) / 2,
		MarkPrice:       m.markPrice,
		Timestamp:       m.currentTime.UnixNano(),
		OpenInterest:    m.position.GetOpenInterest(),
	}
}

// ReloadConf will trigger a reload of all the config settings in the market and all underlying engines
// this is required when hot-reloading any config changes, eg. logger level.
func (m *Market) ReloadConf(
	matchingConfig matching.Config,
	riskConfig risk.Config,
	positionConfig positions.Config,
	settlementConfig settlement.Config,
	feeConfig fee.Config,
) {
	m.log.Info("reloading configuration")
	m.matching.ReloadConf(matchingConfig)
	m.risk.ReloadConf(riskConfig)
	m.position.ReloadConf(positionConfig)
	m.settlement.ReloadConf(settlementConfig)
	m.fee.ReloadConf(feeConfig)
}

// GetID returns the id of the given market
func (m *Market) GetID() string {
	return m.mkt.Id
}

// OnChainTimeUpdate notifies the market of a new time event/update.
// todo: make this a more generic function name e.g. OnTimeUpdateEvent
func (m *Market) OnChainTimeUpdate(t time.Time) (closed bool) {
	ctx := context.TODO()
	timer := metrics.NewTimeCounter(m.mkt.Id, "market", "OnChainTimeUpdate")

	m.mu.Lock()
	defer m.mu.Unlock()

	m.risk.OnTimeUpdate(t)

	closed = t.After(m.closingAt)
	m.closed = closed
	m.currentTime = t
	// opening auction ended
	if !m.currentTime.Before(m.auctionEnd) {
		m.auction = false
	}

	// TODO(): handle market start time

	if m.log.GetLevel() == logging.DebugLevel {
		m.log.Debug("Calculating risk factors (if required)",
			logging.String("market-id", m.mkt.Id))
	}

	m.risk.CalculateFactors(t)

	if m.log.GetLevel() == logging.DebugLevel {
		m.log.Debug("Calculated risk factors and updated positions (maybe)",
			logging.String("market-id", m.mkt.Id))
	}

	timer.EngineTimeCounterAdd()

	if !closed {
		m.broker.Send(events.NewMarketTick(ctx, m.mkt.Id, t))
		return
	}
	// market is closed, final settlement
	// call settlement and stuff
	positions, err := m.settlement.Settle(t, m.markPrice)
	if err != nil {
		m.log.Error(
			"Failed to get settle positions on market close",
			logging.Error(err),
		)
	} else {
		transfers, err := m.collateral.FinalSettlement(ctx, m.GetID(), positions)
		if err != nil {
			m.log.Error(
				"Failed to get ledger movements after settling closed market",
				logging.String("market-id", m.GetID()),
				logging.Error(err),
			)
		} else {
			// @TODO pass in correct context -> Previous or next block? Which is most appropriate here?
			// this will be next block
			evt := events.NewTransferResponse(ctx, transfers)
			m.broker.Send(evt)
			if m.log.GetLevel() == logging.DebugLevel {
				// use transfers, unused var thingy
				for _, v := range transfers {
					if m.log.GetLevel() == logging.DebugLevel {
						m.log.Debug(
							"Got transfers on market close",
							logging.String("transfer", fmt.Sprintf("%v", *v)),
							logging.String("market-id", m.GetID()))
					}
				}
			}

			asset, _ := m.mkt.GetAsset()
			// FIXME(JEREMY): once deposit and withdrawal
			// are implemented with the new method, the partyEngine
			// will be removed, this call will need to be changed to
			// use a slice of parties stored in the current market
			// until we refactor collateral engine to work per market maybe
			parties := m.partyEngine.GetByMarket(m.GetID())
			clearMarketTransfers, err := m.collateral.ClearMarket(ctx, m.GetID(), asset, parties)
			if err != nil {
				m.log.Error("Clear market error",
					logging.String("market-id", m.GetID()),
					logging.Error(err))
			} else {
				evt := events.NewTransferResponse(ctx, clearMarketTransfers)
				m.broker.Send(evt)
				if m.log.GetLevel() == logging.DebugLevel {
					// use transfers, unused var thingy
					for _, v := range clearMarketTransfers {
						if m.log.GetLevel() == logging.DebugLevel {
							m.log.Debug(
								"Market cleared with success",
								logging.String("transfer", fmt.Sprintf("%v", *v)),
								logging.String("market-id", m.GetID()))
						}
					}
				}
			}
		}
	}
	return
}

func (m *Market) unregisterAndReject(ctx context.Context, order *types.Order, err error) error {
	_, perr := m.position.UnregisterOrder(order)
	if perr != nil {
		m.log.Error("Unable to unregister potential trader positions",
			logging.String("market-id", m.GetID()),
			logging.Error(err))
	}
	order.Status = types.Order_STATUS_REJECTED
	if oerr, ok := types.IsOrderError(err); ok {
		order.Reason = oerr
	} else {
		// should not happend but still...
		order.Reason = types.OrderError_ORDER_ERROR_INTERNAL_ERROR
	}
	m.broker.Send(events.NewOrderEvent(ctx, order))
	if m.log.GetLevel() == logging.DebugLevel {
		m.log.Debug("Failure after submitting order to matching engine",
			logging.Order(*order),
			logging.Error(err))
	}
	return err
}

// SubmitOrder submits the given order
func (m *Market) SubmitOrder(ctx context.Context, order *types.Order) (*types.OrderConfirmation, error) {
	timer := metrics.NewTimeCounter(m.mkt.Id, "market", "SubmitOrder")
	orderValidity := "invalid"
	defer func() {
		timer.EngineTimeCounterAdd()
		metrics.OrderCounterInc(m.mkt.Id, orderValidity)
	}()

	// set those at the begining as even rejected order get through the buffers
	m.idgen.SetID(order)
	order.Version = InitialOrderVersion
	order.Status = types.Order_STATUS_ACTIVE

	// Check the expiry time is valid
	if order.ExpiresAt > 0 && order.ExpiresAt < order.CreatedAt {
		order.Status = types.Order_STATUS_REJECTED
		order.Reason = types.OrderError_ORDER_ERROR_INVALID_EXPIRATION_DATETIME
		m.broker.Send(events.NewOrderEvent(ctx, order))
		return nil, ErrInvalidExpiresAtTime
	}

	if m.closed {
		// adding order to the buffer first
		order.Status = types.Order_STATUS_REJECTED
		order.Reason = types.OrderError_ORDER_ERROR_MARKET_CLOSED
		m.broker.Send(events.NewOrderEvent(ctx, order))
		return nil, ErrMarketClosed
	}

	if order.Type == types.Order_TYPE_NETWORK {
		// adding order to the buffer first
		order.Status = types.Order_STATUS_REJECTED
		order.Reason = types.OrderError_ORDER_ERROR_INVALID_TYPE
		m.broker.Send(events.NewOrderEvent(ctx, order))
		return nil, ErrInvalidOrderType
	}

	// Validate market
	if order.MarketID != m.mkt.Id {
		// adding order to the buffer first
		order.Status = types.Order_STATUS_REJECTED
		order.Reason = types.OrderError_ORDER_ERROR_INVALID_MARKET_ID
		m.broker.Send(events.NewOrderEvent(ctx, order))

		if m.log.GetLevel() == logging.DebugLevel {
			m.log.Debug("Market ID mismatch",
				logging.Order(*order),
				logging.String("market", m.mkt.Id))
		}

		return nil, types.ErrInvalidMarketID
	}

	// TODO(): jeremy
	// when new withdrawals and deposits are used only
	// we will need to use the collateral engine
	// and add a check in there to get the General account for
	// this party and the market Asset
	// Verify and add new parties
	// party, _ := m.parties.GetByID(order.PartyID)
	party, _ := m.partyEngine.GetByMarketAndID(m.GetID(), order.PartyID)
	if party == nil {
		// adding order to the buffer first
		order.Status = types.Order_STATUS_REJECTED
		order.Reason = types.OrderError_ORDER_ERROR_INVALID_PARTY_ID
		m.broker.Send(events.NewOrderEvent(ctx, order))

		// trader should be created before even trying to post order
		return nil, ErrTraderDoNotExists
	}

	// ensure party have a general account, and margin account is / can be created
	asset, _ := m.mkt.GetAsset()
	_, err := m.collateral.CreatePartyMarginAccount(ctx, order.PartyID, order.MarketID, asset)
	if err != nil {
		m.log.Error("Margin account verification failed",
			logging.String("party-id", order.PartyID),
			logging.String("market-id", m.GetID()),
			logging.String("asset", asset),
		)
		// adding order to the buffer first
		order.Status = types.Order_STATUS_REJECTED
		order.Reason = types.OrderError_ORDER_ERROR_MISSING_GENERAL_ACCOUNT
		m.broker.Send(events.NewOrderEvent(ctx, order))
		return nil, ErrMissingGeneralAccountForParty
	}

	// Register order as potential positions
	pos, err := m.position.RegisterOrder(order)
	if err != nil {
		// adding order to the buffer first
		order.Status = types.Order_STATUS_REJECTED
		order.Reason = types.OrderError_ORDER_ERROR_INTERNAL_ERROR
		m.broker.Send(events.NewOrderEvent(ctx, order))

		if m.log.GetLevel() == logging.DebugLevel {
			m.log.Debug("Unable to register potential trader position",
				logging.String("market-id", m.GetID()),
				logging.Error(err))
		}
		return nil, ErrMarginCheckFailed
	}

	// Perform check and allocate margin
	newOrderMarginRiskRollback, err := m.checkMarginForOrder(ctx, pos, order, nil)
	if err != nil {
		_, err1 := m.position.UnregisterOrder(order)
		if err1 != nil {
			m.log.Error("Unable to unregister potential trader positions",
				logging.String("market-id", m.GetID()),
				logging.Error(err1))
		}

		// adding order to the buffer first
		order.Status = types.Order_STATUS_REJECTED
		order.Reason = types.OrderError_ORDER_ERROR_MARGIN_CHECK_FAILED
		m.broker.Send(events.NewOrderEvent(ctx, order))

		m.log.Error("Unable to check/add margin for trader",
			logging.String("market-id", m.GetID()),
			logging.Error(err))
		return nil, ErrMarginCheckFailed
	}

	// first we call the order book to get the list of trades
	trades, err := m.matching.GetTrades(order)
	if err != nil {
		return nil, m.unregisterAndReject(ctx, order, err)
	}

	// try to apply fees on the trade
	err = m.applyFees(ctx, order, trades)
	if err != nil {
		return nil, err
	}

	// Send the aggressive order into matching engine
	confirmation, err := m.matching.SubmitOrder(order)
	if confirmation == nil || err != nil {
		_, err := m.position.UnregisterOrder(order)
		if err != nil {
			m.log.Error("Unable to unregister potential trader positions",
				logging.String("market-id", m.GetID()),
				logging.Error(err))
		}
		order.Status = types.Order_STATUS_REJECTED
		if oerr, ok := types.IsOrderError(err); ok {
			order.Reason = oerr
		} else {
			// should not happend but still...
			order.Reason = types.OrderError_ORDER_ERROR_INTERNAL_ERROR
		}
		m.broker.Send(events.NewOrderEvent(ctx, order))
		if m.log.GetLevel() == logging.DebugLevel {
			m.log.Debug("Failure after submitting order to matching engine",
				logging.Order(*order),
				logging.Error(err))
		}

		return nil, err
	}

	// if order was FOK or IOC some or all of it may have not be consumed, so we need to
	// or if the order was stopped because of a wash trade
	// remove them from the potential orders,
	// then we should be able to process the rest of the order properly.
	if (order.TimeInForce == types.Order_TIF_FOK || order.TimeInForce == types.Order_TIF_IOC || order.Status == types.Order_STATUS_STOPPED) &&
		confirmation.Order.Remaining != 0 {
		_, err := m.position.UnregisterOrder(order)
		if err != nil {
			m.log.Error("Unable to unregister potential trader positions",
				logging.String("market-id", m.GetID()),
				logging.Error(err))
		}

		// if the specific case we are in and FOK or IOC
		// we moved some margin, which may never be released
		// as we never create a position in the settlement engine.
		// to be fair the monies would be released later on if the
		// party place an order which stay in the book / trade.
		// but in the case the party is actually neve used again
		// the funds woulds be locked on the margin account until
		// the market is being closed.
		// we also check the margin risk update was not nil, as it's not
		// guaranteed the trader had to pay any margin
		if order.Remaining == order.Size && newOrderMarginRiskRollback != nil {
			transfers, err := m.collateral.RollbackMarginUpdateOnOrder(
				ctx, m.GetID(), asset, newOrderMarginRiskRollback)
			if err != nil {
				m.log.Error("Unable to rollback risk updates",
					logging.String("market-id", m.GetID()),
					logging.Error(err))
			}
			evt := events.NewTransferResponse(
				ctx, []*types.TransferResponse{transfers})
			m.broker.Send(evt)
		}

	}

	// Insert aggressive remaining order
	m.broker.Send(events.NewOrderEvent(ctx, order))

	// we replace the trades in the confirmation with the one we got initially
	// the contains the fees informations
	confirmation.Trades = trades

	m.handleConfirmation(ctx, order, confirmation)

	orderValidity = "valid" // used in deferred func.
	return confirmation, nil
}

func (m *Market) applyFees(ctx context.Context, order *types.Order, trades []*types.Trade) error {
	// if we have some trades, let's try to get the fees
	// FIXME(): change the following code with this check:
	// we de not take any fees if the market was on a open auction
	// if len(trades) <= 0 || m.IsInMarketOpenAuctionMode {
	if len(trades) <= 0 {
		return nil
	}

	// first we get the fees for these trades
	var (
		fees events.FeesTransfer
		err  error
	)

	switch m.mkt.TradingMode.(type) {
	case *types.Market_Continuous:
		// change this by the following check:
		// if m.NotInMonitoringAuctionMode {
		if true {
			fees, err = m.fee.CalculateForContinuousMode(trades)
		}
		// FIXME(): uncomment this once we have implemented
		// monitoring auction mode
		//  Use this once we implemented Opening auction
		// } else {
		// 	fees, err = m.fee.CalculateForAuctionMode(trades)
		// }

		// FIXME(): uncomment this once we have implemented
		// FrequentBatchesAuctions
		// case *types.Market_FrequentBatchesAuctions:
		// 	fees, err = m.fee.CalculateForFrequentBatchesAuctionMode(trades)
	}

	if err != nil {
		return m.unregisterAndReject(ctx, order, err)
	}
	_ = fees

	var (
		transfers []*types.TransferResponse
		asset, _  = m.mkt.GetAsset()
	)
	switch m.mkt.TradingMode.(type) {
	case *types.Market_Continuous:
		// change this by the following check:
		// if m.NotInMonitoringAuctionMode {
		if true {
			transfers, err = m.collateral.TransferFeesContinuousTrading(
				ctx, m.GetID(), asset, fees)
		}
		// FIXME():
		//  Use this once we implemented Opening auction
		// } else {
		// 	transfers, err = m.collateral.TransferFeesAuctionModes(
		// 		ctx, m.GetID(), asset, fees)
		// }

		// FIXME(): uncomment once Frequent batches auctions is implemented
		// case *types.Market_FrequentBatchesAuctions:
		// 	transfers, err = m.collateral.TransferFeesAuctionModes(
		// 		ctx, m.GetID(), asset, fees)
		// }
	}

	if err != nil {
		m.log.Error("unable to transfer fees for trades",
			logging.String("order-id", order.Id),
			logging.String("market-id", m.GetID()),
			logging.Error(err))
		return m.unregisterAndReject(ctx,
			order, types.OrderError_ORDER_ERROR_INSUFFICIENT_FUNDS_TO_PAY_FEES)
	}

	// send transfers through the broker
	if err == nil && len(transfers) > 0 {
		evt := events.NewTransferResponse(ctx, transfers)
		m.broker.Send(evt)
	}

	return nil
}

func (m *Market) handleConfirmation(ctx context.Context, order *types.Order, confirmation *types.OrderConfirmation) {
	if confirmation.PassiveOrdersAffected != nil {
		// Insert or update passive orders siting on the book
		for _, order := range confirmation.PassiveOrdersAffected {
			// set the `updatedAt` value as these orders have changed
			order.UpdatedAt = m.currentTime.UnixNano()
			m.broker.Send(events.NewOrderEvent(ctx, order))
		}
	}

	if len(confirmation.Trades) > 0 {

		// Calculate and set current mark price
		m.setMarkPrice(confirmation.Trades[len(confirmation.Trades)-1])

		// Insert all trades resulted from the executed order
		tradeEvts := make([]events.Event, 0, len(confirmation.Trades))
		for idx, trade := range confirmation.Trades {
			trade.Id = fmt.Sprintf("%s-%010d", order.Id, idx)
			if order.Side == types.Side_SIDE_BUY {
				trade.BuyOrder = order.Id
				trade.SellOrder = confirmation.PassiveOrdersAffected[idx].Id
			} else {
				trade.SellOrder = order.Id
				trade.BuyOrder = confirmation.PassiveOrdersAffected[idx].Id
			}

			tradeEvts = append(tradeEvts, events.NewTradeEvent(ctx, *trade))

			// Update positions (this communicates with settlement via channel)
			m.position.Update(trade)
			// add trade to settlement engine for correct MTM settlement of individual trades
			m.settlement.AddTrade(trade)
		}
		m.broker.SendBatch(tradeEvts)

		// now let's get the transfers for MTM settlement
		evts := m.position.UpdateMarkPrice(m.markPrice)
		settle := m.settlement.SettleMTM(ctx, m.markPrice, evts)

		// Only process collateral and risk once per order, not for every trade
		margins := m.collateralAndRisk(ctx, settle)
		if len(margins) > 0 {

			transfers, closed, err := m.collateral.MarginUpdate(ctx, m.GetID(), margins)
			if m.log.GetLevel() == logging.DebugLevel {
				m.log.Debug(
					"Updated margin balances",
					logging.Int("transfer-count", len(transfers)),
					logging.Int("closed-count", len(closed)),
					logging.Error(err),
				)
				for _, tr := range transfers {
					for _, v := range tr.GetTransfers() {
						if m.log.GetLevel() == logging.DebugLevel {
							m.log.Debug(
								"Ensured margin on order with success",
								logging.String("transfer", fmt.Sprintf("%v", *v)),
								logging.String("market-id", m.GetID()))
						}
					}
				}
			}
			if err == nil && len(transfers) > 0 {
				evt := events.NewTransferResponse(ctx, transfers)
				m.broker.Send(evt)
			}
			if len(closed) > 0 {
				err = m.resolveClosedOutTraders(ctx, closed, order)
				if err != nil {
					m.log.Error("unable to close out traders",
						logging.String("market-id", m.GetID()),
						logging.Error(err))
				}
			}
		}
	}
}

// resolveClosedOutTraders - the traders with the given market position who haven't got sufficient collateral
// need to be closed out -> the network buys/sells the open volume, and trades with the rest of the network
// this flow is similar to the SubmitOrder bit where trades are made, with fewer checks (e.g. no MTM settlement, no risk checks)
// pass in the order which caused traders to be distressed
func (m *Market) resolveClosedOutTraders(ctx context.Context, distressedMarginEvts []events.Margin, o *types.Order) error {
	if len(distressedMarginEvts) == 0 {
		return nil
	}
	timer := metrics.NewTimeCounter(m.mkt.Id, "market", "resolveClosedOutTraders")
	defer timer.EngineTimeCounterAdd()

	distressedPos := make([]events.MarketPosition, 0, len(distressedMarginEvts))
	for _, v := range distressedMarginEvts {
		if m.log.GetLevel() == logging.DebugLevel {
			m.log.Debug("closing out trader",
				logging.String("party-id", v.Party()),
				logging.String("market-id", m.GetID()))
		}
		distressedPos = append(distressedPos, v)
	}
	// cancel pending orders for traders
	rmorders, err := m.matching.RemoveDistressedOrders(distressedPos)
	if err != nil {
		m.log.Error(
			"Failed to remove distressed traders from the orderbook",
			logging.Error(err),
		)
		return err
	}
	mktID := m.GetID()
	// push rm orders into buf
	// and remove the orders from the positions engine
	for _, o := range rmorders {
		o.UpdatedAt = m.currentTime.UnixNano()
		m.broker.Send(events.NewOrderEvent(ctx, o))
		if _, err := m.position.UnregisterOrder(o); err != nil {
			m.log.Error("unable to unregister order for a distressed party",
				logging.String("party-id", o.PartyID),
				logging.String("market-id", mktID),
				logging.String("order-id", o.Id),
			)
		}
	}

	closed := distressedMarginEvts // default behaviour (ie if rmorders is empty) is to close out all distressed positions we started out with

	// we need to check margin requirements again, it's possible for traders to no longer be distressed now that their orders have been removed
	if len(rmorders) != 0 {
		var okPos []events.Margin // need to declare this because we want to reassign closed
		// now that we closed orders, let's run the risk engine again
		// so it'll separate the positions still in distress from the
		// which have acceptable margins
		okPos, closed = m.risk.ExpectMargins(distressedMarginEvts, m.markPrice)

		if m.log.GetLevel() == logging.DebugLevel {
			for _, v := range okPos {
				if m.log.GetLevel() == logging.DebugLevel {
					m.log.Debug("previously distressed party have now an acceptable margin",
						logging.String("market-id", mktID),
						logging.String("party-id", v.Party()))
				}
			}
		}
	}

	// if no position are meant to be closed, just return now.
	if len(closed) <= 0 {
		return nil
	}

	// we only need the MarketPosition events here, and rather than changing all the calls
	// we can just keep the MarketPosition bit
	closedMPs := make([]events.MarketPosition, 0, len(closed))
	// get the actual position, so we can work out what the total position of the market is going to be
	var networkPos int64
	for _, pos := range closed {
		networkPos += pos.Size()
		closedMPs = append(closedMPs, pos)
	}
	if networkPos == 0 {
		m.log.Warn("Network positions is 0 after closing out traders, nothing more to do",
			logging.String("market-id", m.GetID()))

		// remove accounts, positions and return
		// from settlement engine first
		m.settlement.RemoveDistressed(ctx, closed)
		// then from positions
		closedMPs = m.position.RemoveDistressed(closedMPs)
		asset, _ := m.mkt.GetAsset()
		// finally remove from collateral (moving funds where needed)
		var movements *types.TransferResponse
		movements, err = m.collateral.RemoveDistressed(ctx, closedMPs, m.GetID(), asset)
		if err != nil {
			m.log.Error(
				"Failed to remove distressed accounts cleanly",
				logging.Error(err),
			)
			return err
		}
		if len(movements.Transfers) > 0 {
			evt := events.NewTransferResponse(ctx, []*types.TransferResponse{movements})
			m.broker.Send(evt)
		}
		return nil
	}
	// network order
	// @TODO this order is more of a placeholder than an actual final version
	// of the network order we'll be using
	size := uint64(math.Abs(float64(networkPos)))
	no := types.Order{
		MarketID:    m.GetID(),
		Remaining:   size,
		Status:      types.Order_STATUS_ACTIVE,
		PartyID:     networkPartyID,       // network is not a party as such
		Side:        types.Side_SIDE_SELL, // assume sell, price is zero in that case anyway
		CreatedAt:   m.currentTime.UnixNano(),
		Reference:   fmt.Sprintf("LS-%s", o.Id), // liquidity sourcing, reference the order which caused the problem
		TimeInForce: types.Order_TIF_FOK,        // this is an all-or-nothing order, so TIF == FOK
		Type:        types.Order_TYPE_NETWORK,
	}
	no.Size = no.Remaining
	m.idgen.SetID(&no)
	// we need to buy, specify side + max price
	if networkPos < 0 {
		no.Side = types.Side_SIDE_BUY
	}
	// Send the aggressive order into matching engine
	confirmation, err := m.matching.SubmitOrder(&no)
	if err != nil {
		if m.log.GetLevel() == logging.DebugLevel {
			m.log.Debug("Failure after submitting order to matching engine",
				logging.Order(no),
				logging.Error(err))
		}
		return err
	}
	// @NOTE: At this point, the network order was updated by the orderbook
	// the price field now contains the average trade price at which the order was fulfilled
	m.broker.Send(events.NewOrderEvent(ctx, &no))

	// FIXME(j): this is a temporary measure for the case where we do not have enough orders
	// in the book to 0 out the positions.
	// in this case we will just return now, cutting off the position resolution
	// this means that trader still being distressed will stay distressed,
	// then when a new order is placed, the distressed traders will go again through positions resolution
	// and if the volume of the book is acceptable, we will then process positions resolutions
	if no.Remaining == no.Size {
		return ErrNotEnoughVolumeToZeroOutNetworkOrder
	}

	if confirmation.PassiveOrdersAffected != nil {
		// Insert or update passive orders siting on the book
		for _, order := range confirmation.PassiveOrdersAffected {
			order.UpdatedAt = m.currentTime.UnixNano()
			m.broker.Send(events.NewOrderEvent(ctx, order))
		}
	}

	asset, _ := m.mkt.GetAsset()

	// pay the fees now
	fees, err := m.fee.CalculateFeeForPositionResolution(
		confirmation.Trades, closedMPs)
	if err != nil {
		m.log.Error("unable to calculate fees for positions resolutions",
			logging.Error(err),
			logging.String("market-id", m.GetID()))
		return err
	}
	tresps, err := m.collateral.TransferFees(ctx, m.GetID(), asset, fees)
	if err != nil {
		m.log.Error("unable to transfer fees for positions resolutions",
			logging.Error(err),
			logging.String("market-id", m.GetID()))
		return err
	}
	// send transfer to buffer
	m.broker.Send(events.NewTransferResponse(ctx, tresps))

	if len(confirmation.Trades) > 0 {
		// Insert all trades resulted from the executed order
		tradeEvts := make([]events.Event, 0, len(confirmation.Trades))
		for idx, trade := range confirmation.Trades {
			trade.Id = fmt.Sprintf("%s-%010d", no.Id, idx)
			if no.Side == types.Side_SIDE_BUY {
				trade.BuyOrder = no.Id
				trade.SellOrder = confirmation.PassiveOrdersAffected[idx].Id
			} else {
				trade.SellOrder = no.Id
				trade.BuyOrder = confirmation.PassiveOrdersAffected[idx].Id
			}

			// setup the type of the trade to network
			// this trade did happen with a GOOD trader to
			// 0 out the BAD trader position
			trade.Type = types.Trade_TYPE_NETWORK_CLOSE_OUT_GOOD

			tradeEvts = append(tradeEvts, events.NewTradeEvent(ctx, *trade))

			// Update positions - this is a special trade involving the network as party
			// so rather than checking this every time we call Update, call special UpdateNetwork
			m.position.UpdateNetwork(trade)
			m.settlement.AddTrade(trade)
		}
		m.broker.SendBatch(tradeEvts)
	}

	if err = m.zeroOutNetwork(ctx, closedMPs, &no, o); err != nil {
		m.log.Error(
			"Failed to create closing order with distressed traders",
			logging.Error(err),
		)
		return err
	}
	// remove accounts, positions, any funds left on the distressed accounts will be moved to the
	// insurance pool, which needs to happen before we settle the non-distressed traders
	m.settlement.RemoveDistressed(ctx, closed)
	closedMPs = m.position.RemoveDistressed(closedMPs)
	movements, err := m.collateral.RemoveDistressed(ctx, closedMPs, m.GetID(), asset)
	if err != nil {
		m.log.Error(
			"Failed to remove distressed accounts cleanly",
			logging.Error(err),
		)
		return err
	}
	if len(movements.Transfers) > 0 {
		evt := events.NewTransferResponse(ctx, []*types.TransferResponse{movements})
		m.broker.Send(evt)
	}
	// get the updated positions
	evt := m.position.Positions()

	// settle MTM, the positions have changed
	settle := m.settlement.SettleMTM(ctx, m.markPrice, evt)
	// we're not interested in the events here, they're used for margin updates
	// we know the margin requirements will be met, and come the next block
	// margins will automatically be checked anyway

	_, responses, err := m.collateral.MarkToMarket(ctx, m.GetID(), settle, asset)
	if m.log.GetLevel() == logging.DebugLevel {
		m.log.Debug(
			"ledger movements after MTM on traders who closed out distressed",
			logging.Int("response-count", len(responses)),
			logging.String("raw", fmt.Sprintf("%#v", responses)),
		)
	}
	// send transfer to buffer
	m.broker.Send(events.NewTransferResponse(ctx, responses))
	return err
}

func (m *Market) zeroOutNetwork(ctx context.Context, traders []events.MarketPosition, settleOrder, initial *types.Order) error {
	timer := metrics.NewTimeCounter(m.mkt.Id, "market", "zeroOutNetwork")
	defer timer.EngineTimeCounterAdd()

	marketID := m.GetID()
	order := types.Order{
		MarketID:    marketID,
		Status:      types.Order_STATUS_FILLED,
		PartyID:     networkPartyID,
		Price:       settleOrder.Price,
		CreatedAt:   m.currentTime.UnixNano(),
		Reference:   "close-out distressed",
		TimeInForce: types.Order_TIF_FOK, // this is an all-or-nothing order, so TIF == FOK
		Type:        types.Order_TYPE_NETWORK,
	}

	asset, _ := m.mkt.GetAsset()
	marginLevels := types.MarginLevels{
		MarketID:  m.mkt.GetId(),
		Asset:     asset,
		Timestamp: m.currentTime.UnixNano(),
	}

	tradeEvts := make([]events.Event, 0, len(traders))
	for i, trader := range traders {
		tSide, nSide := types.Side_SIDE_SELL, types.Side_SIDE_SELL // one of them will have to sell
		if trader.Size() < 0 {
			tSide = types.Side_SIDE_BUY
		} else {
			nSide = types.Side_SIDE_BUY
		}
		tSize := uint64(math.Abs(float64(trader.Size())))

		// set order fields (network order)
		order.Size = tSize
		order.Remaining = 0
		order.Side = nSide
		order.Status = types.Order_STATUS_FILLED // An order with no remaining must be filled
		m.idgen.SetID(&order)

		// this is the party order
		partyOrder := types.Order{
			MarketID:    marketID,
			Size:        tSize,
			Remaining:   0,
			Status:      types.Order_STATUS_FILLED,
			PartyID:     trader.Party(),
			Side:        tSide,             // assume sell, price is zero in that case anyway
			Price:       settleOrder.Price, // average price
			CreatedAt:   m.currentTime.UnixNano(),
			Reference:   fmt.Sprintf("distressed-%d-%s", i, initial.Id),
			TimeInForce: types.Order_TIF_FOK, // this is an all-or-nothing order, so TIF == FOK
			Type:        types.Order_TYPE_NETWORK,
		}
		m.idgen.SetID(&partyOrder)

		// store the trader order, too
		m.broker.Send(events.NewOrderEvent(ctx, &partyOrder))
		m.broker.Send(events.NewOrderEvent(ctx, &order))

		// now let's create the trade between the party and network
		var (
			buyOrder  *types.Order
			sellOrder *types.Order
		)
		if order.Side == types.Side_SIDE_BUY {
			buyOrder = &order
			sellOrder = &partyOrder
		} else {
			sellOrder = &order
			buyOrder = &partyOrder
		}

		trade := types.Trade{
			Id:        fmt.Sprintf("%s-%010d", partyOrder.Id, 1),
			MarketID:  partyOrder.MarketID,
			Price:     partyOrder.Price,
			Size:      partyOrder.Size,
			Aggressor: order.Side, // we consider network to be agressor
			BuyOrder:  buyOrder.Id,
			SellOrder: sellOrder.Id,
			Buyer:     buyOrder.PartyID,
			Seller:    sellOrder.PartyID,
			Timestamp: partyOrder.CreatedAt,
			Type:      types.Trade_TYPE_NETWORK_CLOSE_OUT_BAD,
		}
		tradeEvts = append(tradeEvts, events.NewTradeEvent(ctx, trade))

		// 0 out margins levels for this trader
		marginLevels.PartyID = trader.Party()
		m.broker.Send(events.NewMarginLevelsEvent(ctx, marginLevels))

		if m.log.GetLevel() == logging.DebugLevel {
			m.log.Debug("trader closed-out with success",
				logging.String("party-id", trader.Party()),
				logging.String("market-id", m.GetID()))
		}
	}
	if len(tradeEvts) > 0 {
		m.broker.SendBatch(tradeEvts)
	}
	return nil
}

func (m *Market) checkMarginForOrder(ctx context.Context, pos *positions.MarketPosition, order, oldOrder *types.Order) (*types.Transfer, error) {
	timer := metrics.NewTimeCounter(m.mkt.Id, "market", "checkMarginForOrder")
	defer timer.EngineTimeCounterAdd()

	asset, err := m.mkt.GetAsset()
	if err != nil {
		return nil, errors.Wrap(err, "unable to get risk updates")
	}

	e, err := m.collateral.GetPartyMargin(pos, asset, m.GetID())
	if err != nil {
		return nil, err
	}
	var riskUpdate events.Risk

	if m.auction {
		riskUpdate, err = m.collateralAndRiskForAuctionOrder(ctx, e, order, oldOrder)
	} else {
		riskUpdate, err = m.collateralAndRiskForOrder(ctx, e, m.markPrice)
	}
	if err != nil {
		if m.log.GetLevel() == logging.DebugLevel {
			m.log.Debug("unable to top up margin on new order",
				logging.String("party-id", order.PartyID),
				logging.String("market-id", order.MarketID),
				logging.Error(err),
			)
		}
		return nil, ErrMarginCheckInsufficient
	} else if riskUpdate == nil {
		if m.log.GetLevel() == logging.DebugLevel {
			m.log.Debug("No risk updates",
				logging.String("market-id", m.GetID()))
		}
		return nil, nil
	}
	return m.riskUpdateTransfer(ctx, riskUpdate, order)
}

func (m *Market) riskUpdateTransfer(ctx context.Context, evt events.Risk, order *types.Order) (*types.Transfer, error) {
	// this is a rollback transfer to be used in case the order do not
	// trade and do not stay in the book to prevent for margin being
	// locked in the margin account forever
	var riskRollback *types.Transfer
	// this should always be a increase to the InitialMargin
	// if it does fail, we need to return an error straight away
	transfer, closePos, err := m.collateral.MarginUpdateOnOrder(ctx, m.GetID(), evt)
	if err != nil {
		return nil, errors.Wrap(err, "unable to get risk updates")
	}
	m.broker.Send(events.NewTransferResponse(ctx, []*types.TransferResponse{transfer}))

	if closePos != nil {
		// if closePose is not nil then we return an error as well, it means the trader did not have enough
		// monies to reach the InitialMargin
		return nil, ErrMarginCheckInsufficient
	}

	if m.log.GetLevel() == logging.DebugLevel {
		m.log.Debug("Transfers applied for ")
		for _, v := range transfer.GetTransfers() {
			m.log.Debug(
				"Ensured margin on order with success",
				logging.String("transfer", fmt.Sprintf("%v", *v)),
				logging.String("market-id", m.GetID()),
			)
		}
	}

	if len(transfer.Transfers) > 0 {
		// we create the rollback transfer here, so it can be used in case of.
		riskRollback = &types.Transfer{
			Owner: evt.Party(),
			Amount: &types.FinancialAmount{
				Amount: int64(transfer.Transfers[0].Amount),
				Asset:  evt.Asset(),
			},
			Type:      types.TransferType_TRANSFER_TYPE_MARGIN_HIGH,
			MinAmount: int64(transfer.Transfers[0].Amount),
		}
	}
	return riskRollback, nil
}

func (m *Market) collateralAndRiskForAuctionOrder(ctx context.Context, e events.Margin, o, old *types.Order) (events.Risk, error) {
	// get the balance, then get the margin for this order
	margins, err := m.risk.UpdateMarginOnAuctionOrder(ctx, e, o, old)
	if err != nil {
		return nil, err
	}
	return margins, nil
}

// this function handles moving money after settle MTM + risk margin updates
// but does not move the money between trader accounts (ie not to/from margin accounts after risk)
func (m *Market) collateralAndRiskForOrder(ctx context.Context, e events.Margin, price uint64) (events.Risk, error) {
	timer := metrics.NewTimeCounter(m.mkt.Id, "market", "collateralAndRiskForOrder")
	defer timer.EngineTimeCounterAdd()

	// let risk engine do its thing here - it returns a slice of money that needs
	// to be moved to and from margin accounts
	riskUpdate, err := m.risk.UpdateMarginOnNewOrder(ctx, e, price)
	if err != nil {
		return nil, err
	}
	if riskUpdate == nil {
		m.log.Debug("No risk updates after call to Update Margins in collateralAndRisk()")
		return nil, nil
	}

	if m.log.GetLevel() == logging.DebugLevel {
		m.log.Debug("Got margins transfer on new order")
		transfer := riskUpdate.Transfer()
		m.log.Debug(
			"New margin transfer on order submit",
			logging.String("transfer", fmt.Sprintf("%v", *transfer)),
			logging.String("market-id", m.GetID()),
		)
	}

	return riskUpdate, nil
}

func (m *Market) setMarkPrice(trade *types.Trade) {
	// The current mark price calculation is simply the last trade
	// in the future this will use varying logic based on market config
	// the responsibility for calculation could be elsewhere for testability
	m.markPrice = trade.Price
}

// this function handles moving money after settle MTM + risk margin updates
// but does not move the money between trader accounts (ie not to/from margin accounts after risk)
func (m *Market) collateralAndRisk(ctx context.Context, settle []events.Transfer) []events.Risk {
	timer := metrics.NewTimeCounter(m.mkt.Id, "market", "collateralAndRisk")
	asset, _ := m.mkt.GetAsset()
	evts, response, err := m.collateral.MarkToMarket(ctx, m.GetID(), settle, asset)
	if err != nil {
		m.log.Error(
			"Failed to process mark to market settlement (collateral)",
			logging.Error(err),
		)
		return nil
	}
	if m.log.GetLevel() == logging.DebugLevel {
		// @TODO stream the ledger movements here
		m.log.Debug(
			"transfer responses after MTM settlement",
			logging.Int("transfer-count", len(response)),
			logging.String("raw-dump", fmt.Sprintf("%#v", response)),
		)
	}
	// sending response to buffer
	evt := events.NewTransferResponse(ctx, response)
	m.broker.Send(evt)

	// let risk engine do its thing here - it returns a slice of money that needs
	// to be moved to and from margin accounts
	riskUpdates := m.risk.UpdateMarginsOnSettlement(ctx, evts, m.markPrice)
	if len(riskUpdates) == 0 {
		if m.log.GetLevel() == logging.DebugLevel {
			m.log.Debug("No risk updates after call to Update Margins in collateralAndRisk()")
		}
		return nil
	}

	timer.EngineTimeCounterAdd()
	return riskUpdates
}

// CancelOrder cancels the given order
func (m *Market) CancelOrder(ctx context.Context, oc *types.OrderCancellation) (*types.OrderCancellationConfirmation, error) {
	timer := metrics.NewTimeCounter(m.mkt.Id, "market", "CancelOrder")
	defer timer.EngineTimeCounterAdd()

	if m.closed {
		return nil, ErrMarketClosed
	}

	// Validate Market
	if oc.MarketID != m.mkt.Id {
		if m.log.GetLevel() == logging.DebugLevel {
			m.log.Debug("Market ID mismatch",
				logging.String("party-id", oc.PartyID),
				logging.String("order-id", oc.OrderID),
				logging.String("market", m.mkt.Id))
		}
		return nil, types.ErrInvalidMarketID
	}

	order, err := m.matching.GetOrderByID(oc.OrderID)
	if err != nil {
		return nil, err
	}

	// Only allow the original order creator to cancel their order
	if order.PartyID != oc.PartyID {
		if m.log.GetLevel() == logging.DebugLevel {
			m.log.Debug("Party ID mismatch",
				logging.String("party-id", oc.PartyID),
				logging.String("order-id", oc.OrderID),
				logging.String("market", m.mkt.Id))
		}
		return nil, types.ErrInvalidPartyID
	}

	cancellation, err := m.matching.CancelOrder(order)
	if cancellation == nil || err != nil {
		if m.log.GetLevel() == logging.DebugLevel {
			m.log.Debug("Failure after cancel order from matching engine",
				logging.String("party-id", oc.PartyID),
				logging.String("order-id", oc.OrderID),
				logging.String("market", m.mkt.Id),
				logging.Error(err))
		}
		return nil, err
	}

	// Update the order in our stores (will be marked as cancelled)
	cancellation.Order.UpdatedAt = m.currentTime.UnixNano()
	m.broker.Send(events.NewOrderEvent(ctx, cancellation.Order))
	pos, err := m.position.UnregisterOrder(cancellation.Order)
	if err != nil {
		m.log.Error("Failure unregistering order in positions engine (cancel)",
			logging.Order(*order),
			logging.Error(err))
	}
	if m.auction {
		asset, err := m.mkt.GetAsset()
		if err != nil {
			return nil, errors.Wrap(err, "unable to get risk updates")
		}

		evt, err := m.collateral.GetPartyMargin(pos, asset, m.GetID())
		if err != nil {
			return nil, err
		}
		risk := m.risk.UpdateMarginOnCancelAuctionOrder(ctx, evt, order)
		if _, err := m.riskUpdateTransfer(ctx, risk, order); err != nil {
			m.log.Error("Failed to release margin after order was cancelled", logging.Error(err))
		}
	}

	return cancellation, nil
}

// CancelOrderByID locates order by its Id and cancels it
// @TODO This function should not exist. Needs to be removed
func (m *Market) CancelOrderByID(orderID string) (*types.OrderCancellationConfirmation, error) {
	ctx := context.TODO()
	order, err := m.matching.GetOrderByID(orderID)
	if err != nil {
		return nil, err
	}
	cancellation := types.OrderCancellation{
		OrderID:  order.Id,
		PartyID:  order.PartyID,
		MarketID: order.MarketID,
	}
	return m.CancelOrder(ctx, &cancellation)
}

// AmendOrder amend an existing order from the order book
func (m *Market) AmendOrder(ctx context.Context, orderAmendment *types.OrderAmendment) (*types.OrderConfirmation, error) {
	timer := metrics.NewTimeCounter(m.mkt.Id, "market", "AmendOrder")
	defer timer.EngineTimeCounterAdd()

	// Verify that the market is not closed
	if m.closed {
		return nil, ErrMarketClosed
	}

	// Try and locate the existing order specified on the
	// order book in the matching engine for this market
	existingOrder, err := m.matching.GetOrderByID(orderAmendment.OrderID)
	if err != nil {
		if m.log.GetLevel() == logging.DebugLevel {
			m.log.Debug("Invalid order ID",
				logging.String("id", orderAmendment.GetOrderID()),
				logging.String("party", orderAmendment.GetPartyID()),
				logging.String("market", orderAmendment.GetMarketID()),
				logging.Error(err))
		}

		return nil, types.ErrInvalidOrderID
	}

	// We can only amend this order if we created it
	if existingOrder.PartyID != orderAmendment.PartyID {
		if m.log.GetLevel() == logging.DebugLevel {
			m.log.Debug("Invalid party ID",
				logging.String("original party id:", existingOrder.PartyID),
				logging.String("amend party id:", orderAmendment.PartyID))
		}
		return nil, types.ErrInvalidPartyID
	}

	// Validate Market
	if existingOrder.MarketID != m.mkt.Id {
		if m.log.GetLevel() == logging.DebugLevel {
			m.log.Debug("Market ID mismatch",
				logging.String("market-id", m.mkt.Id),
				logging.Order(*existingOrder))
		}
		return nil, types.ErrInvalidMarketID
	}

	if err := m.validateOrderAmendment(existingOrder, orderAmendment); err != nil {
		return nil, err
	}

	amendedOrder, err := m.applyOrderAmendment(existingOrder, orderAmendment)
	if err != nil {
		return nil, err
	}

	// if remaining is reduces <= 0, then order is cancelled
	if amendedOrder.Remaining <= 0 {
		orderCancel := types.OrderCancellation{
			OrderID:  existingOrder.Id,
			PartyID:  existingOrder.PartyID,
			MarketID: existingOrder.MarketID,
		}

		confirm, err := m.CancelOrder(ctx, &orderCancel)
		if err != nil {
			return nil, err
		}
		return &types.OrderConfirmation{
			Order: confirm.Order,
		}, nil
	}

	// if expiration has changed and is before the original creation time, reject this amend
	if amendedOrder.ExpiresAt != 0 && amendedOrder.ExpiresAt < existingOrder.CreatedAt {
		if m.log.GetLevel() == logging.DebugLevel {
			m.log.Debug("Amended expiry before original creation time",
				logging.Int64("original order created at ts:", existingOrder.CreatedAt),
				logging.Int64("amended expiry ts:", amendedOrder.ExpiresAt),
				logging.Order(*existingOrder))
		}
		return nil, types.ErrInvalidExpirationDatetime
	}

	// if expiration has changed and is not 0, and is before currentTime
	// then we expire the order
	if amendedOrder.ExpiresAt != 0 && amendedOrder.ExpiresAt < amendedOrder.UpdatedAt {
		// Update the exiting message in place before we cancel it
		m.orderAmendInPlace(existingOrder, amendedOrder)
		cancellation, err := m.matching.CancelOrder(amendedOrder)
		if cancellation == nil || err != nil {
			if m.log.GetLevel() == logging.DebugLevel {
				m.log.Debug("Failure to cancel order from matching engine",
					logging.String("party-id", amendedOrder.PartyID),
					logging.String("order-id", amendedOrder.Id),
					logging.String("market", m.mkt.Id),
					logging.Error(err))
			}
			return nil, err
		}

		// Update the order in our stores (will be marked as cancelled)
		// set the proper status
		cancellation.Order.Status = types.Order_STATUS_EXPIRED
		m.broker.Send(events.NewOrderEvent(ctx, cancellation.Order))
		_, err = m.position.UnregisterOrder(cancellation.Order)
		if err != nil {
			m.log.Error("Failure unregistering order in positions engine (amendOrder)",
				logging.Order(*amendedOrder),
				logging.Error(err))
		}

		return &types.OrderConfirmation{
			Order: cancellation.Order,
		}, nil
	}

	// from here these are the normal amendment

	var priceShift, sizeIncrease, sizeDecrease, expiryChange, timeInForceChange bool

	if amendedOrder.Price != existingOrder.Price {
		priceShift = true
	}

	if amendedOrder.Size > existingOrder.Size {
		sizeIncrease = true
	}
	if amendedOrder.Size < existingOrder.Size {
		sizeDecrease = true
	}

	if amendedOrder.ExpiresAt != existingOrder.ExpiresAt {
		expiryChange = true
	}

	if amendedOrder.TimeInForce != existingOrder.TimeInForce {
		timeInForceChange = true
	}

	// If nothing changed, amend in place to update updatedAt and version number
	if !priceShift && !sizeIncrease && !sizeDecrease && !expiryChange && !timeInForceChange {
		ret, err := m.orderAmendInPlace(existingOrder, amendedOrder)
		if err == nil {
			m.broker.Send(events.NewOrderEvent(ctx, amendedOrder))
		}
		return ret, err
	}

	// Update potential new position after the amend
	pos, err := m.position.AmendOrder(existingOrder, amendedOrder)
	if err != nil {
		// adding order to the buffer first
		amendedOrder.Status = types.Order_STATUS_REJECTED
		amendedOrder.Reason = types.OrderError_ORDER_ERROR_INTERNAL_ERROR
		m.broker.Send(events.NewOrderEvent(ctx, amendedOrder))

		if m.log.GetLevel() == logging.DebugLevel {
			m.log.Debug("Unable to amend potential trader position",
				logging.String("market-id", m.GetID()),
				logging.Error(err))
		}
		return nil, ErrMarginCheckFailed
	}

	// Perform check and allocate margin
	// ignore rollback return here, as if we amend it means the order
	// is already on the book, not rollback will be needed, the margin
	// will be updated later on for sure.

	if _, err = m.checkMarginForOrder(ctx, pos, amendedOrder, existingOrder); err != nil {
		// Undo the position registering
		_, err1 := m.position.AmendOrder(amendedOrder, existingOrder)
		if err1 != nil {
			m.log.Error("Unable to unregister potential amended trader position",
				logging.String("market-id", m.GetID()),
				logging.Error(err1))
		}

		if m.log.GetLevel() == logging.DebugLevel {
			m.log.Debug("Unable to check/add margin for trader",
				logging.String("market-id", m.GetID()),
				logging.Error(err))
		}
		return nil, ErrMarginCheckFailed
	}

	// if increase in size or change in price
	// ---> DO atomic cancel and submit
	if priceShift || sizeIncrease {
		confirmation, err := m.orderCancelReplace(ctx, existingOrder, amendedOrder)
		if err == nil {
			m.handleConfirmation(ctx, amendedOrder, confirmation)
			m.broker.Send(events.NewOrderEvent(ctx, amendedOrder))
		}
		return confirmation, err
	}

	// if decrease in size or change in expiration date
	// ---> DO amend in place in matching engine
	if expiryChange || sizeDecrease || timeInForceChange {
		if sizeDecrease && amendedOrder.Remaining >= existingOrder.Remaining {
			if m.log.GetLevel() == logging.DebugLevel {
				m.log.Debug("Order amendment not allowed when reducing to a larger amount", logging.Order(*existingOrder))
			}
			return nil, ErrInvalidAmendRemainQuantity
		}
		ret, err := m.orderAmendInPlace(existingOrder, amendedOrder)
		if err == nil {
			m.broker.Send(events.NewOrderEvent(ctx, amendedOrder))
		}
		return ret, err
	}

	if m.log.GetLevel() == logging.DebugLevel {
		m.log.Debug("Order amendment not allowed", logging.Order(*existingOrder))
	}
	return nil, types.ErrEditNotAllowed
}

func (m *Market) validateOrderAmendment(
	order *types.Order,
	amendment *types.OrderAmendment,
) error {
	// check TIF and expiracy
	if amendment.TimeInForce == types.Order_TIF_GTT {
		if amendment.ExpiresAt == nil {
			return errors.New("cannot amend to order type GTT without an expiryAt value")
		}
		// if expiresAt is before or equal to created at
		// we return an error
		if amendment.ExpiresAt.Value <= order.CreatedAt {
			return fmt.Errorf("amend order, ExpiresAt(%v) can't be <= CreatedAt(%v)", amendment.ExpiresAt, order.CreatedAt)
		}
	} else if amendment.TimeInForce == types.Order_TIF_GTC {
		// this is cool, but we need to ensure and expiry is not set
		if amendment.ExpiresAt != nil {
			return errors.New("amend order, TIF GTC cannot have ExpiresAt set")
		}
	} else if amendment.TimeInForce == types.Order_TIF_FOK ||
		amendment.TimeInForce == types.Order_TIF_IOC {
		// IOC and FOK are not acceptable for amend order
		return errors.New("amend order, TIF FOK and IOC are not allowed")
	}
	return nil
}

// this function assume the amendment have been validated before
func (m *Market) applyOrderAmendment(
	existingOrder *types.Order,
	amendment *types.OrderAmendment,
) (order *types.Order, err error) {
	m.mu.Lock()
	currentTime := m.currentTime
	m.mu.Unlock()

	// initialize order with the existing order data
	order = &types.Order{
		Type:        existingOrder.Type,
		Id:          existingOrder.Id,
		MarketID:    existingOrder.MarketID,
		PartyID:     existingOrder.PartyID,
		Side:        existingOrder.Side,
		Price:       existingOrder.Price,
		Size:        existingOrder.Size,
		Remaining:   existingOrder.Remaining,
		TimeInForce: existingOrder.TimeInForce,
		CreatedAt:   existingOrder.CreatedAt,
		Status:      existingOrder.Status,
		ExpiresAt:   existingOrder.ExpiresAt,
		Reference:   existingOrder.Reference,
		Version:     existingOrder.Version + 1,
		UpdatedAt:   currentTime.UnixNano(),
	}

	// apply price changes
	if amendment.Price != nil && existingOrder.Price != amendment.Price.Value {
		order.Price = amendment.Price.Value
	}

	// apply size changes
	if amendment.SizeDelta != 0 {
		order.Size += uint64(amendment.SizeDelta)
		newRemaining := int64(existingOrder.Remaining) + amendment.SizeDelta
		if newRemaining <= 0 {
			newRemaining = 0
		}
		order.Remaining = uint64(newRemaining)
	}

	// apply tif
	if amendment.TimeInForce != types.Order_TIF_UNSPECIFIED {
		order.TimeInForce = amendment.TimeInForce
		if amendment.TimeInForce != types.Order_TIF_GTT {
			order.ExpiresAt = 0
		}
	}
	if amendment.ExpiresAt != nil {
		order.ExpiresAt = amendment.ExpiresAt.Value
	}
	return
}

func (m *Market) orderCancelReplace(ctx context.Context, existingOrder, newOrder *types.Order) (conf *types.OrderConfirmation, err error) {
	timer := metrics.NewTimeCounter(m.mkt.Id, "market", "orderCancelReplace")

	if m.log.GetLevel() == logging.DebugLevel {
		m.log.Debug("Cancel/replace order")
	}

	cancellation, err := m.matching.CancelOrder(existingOrder)
	if cancellation == nil {
		if err != nil {
			if m.log.GetLevel() == logging.DebugLevel {
				m.log.Debug("Failed to cancel order from matching engine during CancelReplace",
					logging.OrderWithTag(*existingOrder, "existing-order"),
					logging.OrderWithTag(*newOrder, "new-order"),
					logging.Error(err))
			}
		} else {
			err = fmt.Errorf("order cancellation failed (no error given)")
		}
	} else {
		// calculates the fees
		trades, err := m.matching.GetTrades(newOrder)
		if err != nil {
			return nil, m.unregisterAndReject(ctx, newOrder, err)
		}

		// try to apply fees on the trade
		err = m.applyFees(ctx, newOrder, trades)
		if err != nil {
			return nil, err
		}

		conf, err = m.matching.SubmitOrder(newOrder)
		// replace the trades in the confirmation to have
		// the ones with the fees embbeded
		conf.Trades = trades
	}

	timer.EngineTimeCounterAdd()
	return
}

func (m *Market) orderAmendInPlace(originalOrder, amendOrder *types.Order) (*types.OrderConfirmation, error) {
	timer := metrics.NewTimeCounter(m.mkt.Id, "market", "orderAmendInPlace")
	defer timer.EngineTimeCounterAdd()

	err := m.matching.AmendOrder(originalOrder, amendOrder)
	if err != nil {
		if m.log.GetLevel() == logging.DebugLevel {
			m.log.Debug("Failure after amend order from matching engine (amend-in-place)",
				logging.OrderWithTag(*amendOrder, "new-order"),
				logging.Error(err))
		}
		return nil, err
	}
	return &types.OrderConfirmation{
		Order: amendOrder,
	}, nil
}

// RemoveExpiredOrders remove all expired orders from the order book
func (m *Market) RemoveExpiredOrders(timestamp int64) (orderList []types.Order, err error) {
	timer := metrics.NewTimeCounter(m.mkt.Id, "market", "RemoveExpiredOrders")

	if m.closed {
		err = ErrMarketClosed
	} else {
		orderList = m.matching.RemoveExpiredOrders(timestamp)
		// need to remove the expired orders from the potentials positions
		for _, order := range orderList {
			order := order
			_, err = m.position.UnregisterOrder(&order)
			if err != nil {
				if m.log.GetLevel() == logging.DebugLevel {
					m.log.Debug("Failure unregistering order in positions engine (cancel)",
						logging.Order(order),
						logging.Error(err))
				}
			}
		}
	}

	timer.EngineTimeCounterAdd()
	return
}

// create an actual risk model, and calculate the risk factors
// if something goes wrong, return the hard-coded values of old
func getInitialFactors(log *logging.Logger, mkt *types.Market, asset string) *types.RiskResult {
	rm, err := risk.NewModel(log, mkt.TradableInstrument.RiskModel, asset)
	// @TODO log this error
	if err != nil {
		return nil
	}
	if ok, fact := rm.CalculateRiskFactors(nil); ok {
		return fact
	}
	// default to hard-coded risk factors
	return &types.RiskResult{
		RiskFactors: map[string]*types.RiskFactor{
			asset: {Long: 0.15, Short: 0.25},
		},
		PredictedNextRiskFactors: map[string]*types.RiskFactor{
			asset: {Long: 0.15, Short: 0.25},
		},
	}
}
