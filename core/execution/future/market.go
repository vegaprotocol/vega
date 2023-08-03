// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package future

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/core/assets"
	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/execution/common"
	"code.vegaprotocol.io/vega/core/execution/stoporders"
	"code.vegaprotocol.io/vega/core/fee"
	"code.vegaprotocol.io/vega/core/idgeneration"
	liquiditytarget "code.vegaprotocol.io/vega/core/liquidity/target"
	"code.vegaprotocol.io/vega/core/liquidity/v2"
	"code.vegaprotocol.io/vega/core/markets"
	"code.vegaprotocol.io/vega/core/matching"
	"code.vegaprotocol.io/vega/core/metrics"
	"code.vegaprotocol.io/vega/core/monitor"
	lmon "code.vegaprotocol.io/vega/core/monitor/liquidity"
	"code.vegaprotocol.io/vega/core/monitor/price"
	"code.vegaprotocol.io/vega/core/positions"
	"code.vegaprotocol.io/vega/core/products"
	"code.vegaprotocol.io/vega/core/risk"
	"code.vegaprotocol.io/vega/core/settlement"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/core/types/statevar"
	vegacontext "code.vegaprotocol.io/vega/libs/context"
	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/ptr"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/protos/vega"
)

// LiquidityMonitor.
type LiquidityMonitor interface {
	CheckLiquidity(as lmon.AuctionState, t time.Time, currentStake *num.Uint, trades []*types.Trade, rf types.RiskFactor, markPrice *num.Uint, bestStaticBidVolume, bestStaticAskVolume uint64, persistent bool) bool
	SetMinDuration(d time.Duration)
	UpdateTargetStakeTriggerRatio(ctx context.Context, ratio num.Decimal)
	UpdateParameters(*types.LiquidityMonitoringParameters)
}

// TargetStakeCalculator interface.
type TargetStakeCalculator interface {
	types.StateProvider
	RecordOpenInterest(oi uint64, now time.Time) error
	GetTargetStake(rf types.RiskFactor, now time.Time, markPrice *num.Uint) *num.Uint
	GetTheoreticalTargetStake(rf types.RiskFactor, now time.Time, markPrice *num.Uint, trades []*types.Trade) *num.Uint
	UpdateScalingFactor(sFactor num.Decimal) error
	UpdateTimeWindow(tWindow time.Duration)
	StopSnapshots()
	UpdateParameters(types.TargetStakeParameters)
}

// Market represents an instance of a market in vega and is in charge of calling
// the engines in order to process all transactions.
type Market struct {
	log   *logging.Logger
	idgen common.IDGenerator

	mkt *types.Market

	closingAt   time.Time
	timeService common.TimeService

	mu sync.Mutex

	lastTradedPrice *num.Uint
	markPrice       *num.Uint
	priceFactor     *num.Uint

	// own engines
	matching           *matching.CachedOrderBook
	tradableInstrument *markets.TradableInstrument
	risk               *risk.Engine
	position           *positions.SnapshotEngine
	settlement         *settlement.SnapshotEngine
	fee                *fee.Engine
	liquidity          *common.MarketLiquidity
	liquidityEngine    common.LiquidityEngine

	// deps engines
	collateral common.Collateral

	broker               common.Broker
	closed               bool
	finalFeesDistributed bool

	parties map[string]struct{}

	pMonitor common.PriceMonitor
	lMonitor LiquidityMonitor

	linearSlippageFactor    num.Decimal
	quadraticSlippageFactor num.Decimal

	tsCalc TargetStakeCalculator

	as common.AuctionState

	peggedOrders   *common.PeggedOrders
	expiringOrders *common.ExpiringOrders

	// Store the previous price values so we can see what has changed
	lastBestBidPrice *num.Uint
	lastBestAskPrice *num.Uint
	lastMidBuyPrice  *num.Uint
	lastMidSellPrice *num.Uint

	lastMarketValueProxy    num.Decimal
	bondPenaltyFactor       num.Decimal
	marketValueWindowLength time.Duration

	// Liquidity Fee
	feeSplitter  *common.FeeSplitter
	equityShares *common.EquityShares

	stateVarEngine        common.StateVarEngine
	marketActivityTracker *common.MarketActivityTracker
	positionFactor        num.Decimal // 10^pdp
	assetDP               uint32

	settlementDataInMarket *num.Numeric
	nextMTM                time.Time
	mtmDelta               time.Duration

	settlementAsset string
	succeeded       bool

	maxStopOrdersPerParties *num.Uint
	stopOrders              *stoporders.Pool
	expiringStopOrders      *common.ExpiringOrders

	minDuration time.Duration
	perp        bool
}

// NewMarket creates a new market using the market framework configuration and creates underlying engines.
func NewMarket(
	ctx context.Context,
	log *logging.Logger,
	riskConfig risk.Config,
	positionConfig positions.Config,
	settlementConfig settlement.Config,
	matchingConfig matching.Config,
	feeConfig fee.Config,
	liquidityConfig liquidity.Config,
	collateralEngine common.Collateral,
	oracleEngine products.OracleEngine,
	mkt *types.Market,
	timeService common.TimeService,
	broker common.Broker,
	auctionState *monitor.AuctionState,
	stateVarEngine common.StateVarEngine,
	marketActivityTracker *common.MarketActivityTracker,
	assetDetails *assets.Asset,
	peggedOrderNotify func(int64),
) (*Market, error) {
	if len(mkt.ID) == 0 {
		return nil, common.ErrEmptyMarketID
	}

	assetDecimals := assetDetails.DecimalPlaces()
	positionFactor := num.DecimalFromFloat(10).Pow(num.DecimalFromInt64(mkt.PositionDecimalPlaces))

	tradableInstrument, err := markets.NewTradableInstrument(ctx, log, mkt.TradableInstrument, mkt.ID, oracleEngine, broker, uint32(assetDecimals))
	if err != nil {
		return nil, fmt.Errorf("unable to instantiate a new market: %w", err)
	}
	priceFactor := num.NewUint(1)
	if exp := assetDecimals - mkt.DecimalPlaces; exp != 0 {
		priceFactor.Exp(num.NewUint(10), num.NewUint(exp))
	}

	// @TODO -> the raw auctionstate shouldn't be something exposed to the matching engine
	// as far as matching goes: it's either an auction or not
	book := matching.NewCachedOrderBook(log, matchingConfig, mkt.ID, auctionState.InAuction(), peggedOrderNotify)
	asset := tradableInstrument.Instrument.Product.GetAsset()

	riskEngine := risk.NewEngine(log,
		riskConfig,
		tradableInstrument.MarginCalculator,
		tradableInstrument.RiskModel,
		book,
		auctionState,
		timeService,
		broker,
		mkt.ID,
		asset,
		stateVarEngine,
		positionFactor,
		false,
		nil,
		mkt.LinearSlippageFactor,
		mkt.QuadraticSlippageFactor,
	)

	settleEngine := settlement.NewSnapshotEngine(
		log,
		settlementConfig,
		tradableInstrument.Instrument.Product,
		mkt.ID,
		timeService,
		broker,
		positionFactor,
	)
	positionEngine := positions.NewSnapshotEngine(log, positionConfig, mkt.ID, broker)

	feeEngine, err := fee.New(log, feeConfig, *mkt.Fees, asset, positionFactor)
	if err != nil {
		return nil, fmt.Errorf("unable to instantiate fee engine: %w", err)
	}

	tsCalc := liquiditytarget.NewSnapshotEngine(*mkt.LiquidityMonitoringParameters.TargetStakeParameters, positionEngine, mkt.ID, positionFactor)

	pMonitor, err := price.NewMonitor(asset, mkt.ID, tradableInstrument.RiskModel, auctionState, mkt.PriceMonitoringSettings, stateVarEngine, log)
	if err != nil {
		return nil, fmt.Errorf("unable to instantiate price monitoring engine: %w", err)
	}

	lMonitor := lmon.NewMonitor(tsCalc, mkt.LiquidityMonitoringParameters)

	now := timeService.GetTimeNow()

	liquidityEngine := liquidity.NewSnapshotEngine(
		liquidityConfig, log, timeService, broker, tradableInstrument.RiskModel,
		pMonitor, book, auctionState, asset, mkt.ID, stateVarEngine, positionFactor, mkt.LiquiditySLAParams,
	)

	equityShares := common.NewEquityShares(num.DecimalZero())

	marketLiquidity := common.NewMarketLiquidity(
		log, liquidityEngine, collateralEngine, broker, book, equityShares, marketActivityTracker,
		feeEngine, common.FutureMarketType, mkt.ID, asset, priceFactor, mkt.LiquiditySLAParams.PriceRange,
		mkt.LiquiditySLAParams.ProvidersFeeCalculationTimeStep,
	)

	// The market is initially created in a proposed state
	mkt.State = types.MarketStateProposed
	mkt.TradingMode = types.MarketTradingModeNoTrading

	pending, open := auctionState.GetAuctionBegin(), auctionState.GetAuctionEnd()
	// Populate the market timestamps
	ts := &types.MarketTimestamps{
		Proposed: now.UnixNano(),
		Pending:  now.UnixNano(),
	}
	if pending != nil {
		ts.Pending = pending.UnixNano()
	}
	if open != nil {
		ts.Open = open.UnixNano()
	}

	mkt.MarketTimestamps = ts

	marketType := mkt.MarketType()
	market := &Market{
		log:                     log,
		idgen:                   nil,
		mkt:                     mkt,
		matching:                book,
		tradableInstrument:      tradableInstrument,
		risk:                    riskEngine,
		position:                positionEngine,
		settlement:              settleEngine,
		collateral:              collateralEngine,
		timeService:             timeService,
		broker:                  broker,
		fee:                     feeEngine,
		liquidity:               marketLiquidity,
		liquidityEngine:         liquidityEngine, // TODO karel - consider not having this
		parties:                 map[string]struct{}{},
		as:                      auctionState,
		pMonitor:                pMonitor,
		lMonitor:                lMonitor,
		tsCalc:                  tsCalc,
		peggedOrders:            common.NewPeggedOrders(log, timeService),
		expiringOrders:          common.NewExpiringOrders(),
		feeSplitter:             common.NewFeeSplitter(),
		equityShares:            equityShares,
		lastBestAskPrice:        num.UintZero(),
		lastMidSellPrice:        num.UintZero(),
		lastMidBuyPrice:         num.UintZero(),
		lastBestBidPrice:        num.UintZero(),
		stateVarEngine:          stateVarEngine,
		marketActivityTracker:   marketActivityTracker,
		priceFactor:             priceFactor,
		positionFactor:          positionFactor,
		nextMTM:                 time.Time{}, // default to zero time
		linearSlippageFactor:    mkt.LinearSlippageFactor,
		quadraticSlippageFactor: mkt.QuadraticSlippageFactor,
		maxStopOrdersPerParties: num.UintZero(),
		stopOrders:              stoporders.New(log),
		expiringStopOrders:      common.NewExpiringOrders(),
		perp:                    marketType == types.MarketTypePerp,
	}

	assets, _ := mkt.GetAssets()
	market.settlementAsset = assets[0]

	liquidityEngine.SetGetStaticPricesFunc(market.getBestStaticPricesDecimal)

	switch marketType {
	case types.MarketTypeFuture:
		market.tradableInstrument.Instrument.Product.NotifyOnTradingTerminated(market.tradingTerminated)
		market.tradableInstrument.Instrument.Product.NotifyOnSettlementData(market.settlementData)
	case types.MarketTypePerp:
		market.tradableInstrument.Instrument.Product.NotifyOnSettlementData(market.settlementDataPerp)
	case types.MarketTypeSpot:
	default:
		log.Panic("unexpected market type", logging.Int("type", int(marketType)))
	}
	market.assetDP = uint32(assetDecimals)
	return market, nil
}

func (m *Market) OnEpochEvent(ctx context.Context, epoch types.Epoch) {
	if m.closed {
		return
	}
	if epoch.Action == vega.EpochAction_EPOCH_ACTION_START {
		m.liquidity.OnEpochStart(ctx, m.timeService.GetTimeNow(), m.markPrice, m.midPrice(), m.getTargetStake(), m.positionFactor)
	} else if epoch.Action == vega.EpochAction_EPOCH_ACTION_END && !m.finalFeesDistributed {
		m.liquidity.OnEpochEnd(ctx, m.timeService.GetTimeNow())
	}

	m.updateLiquidityFee(ctx)
}

func (m *Market) OnEpochRestore(ctx context.Context, epoch types.Epoch) {
	// TODO karel - implement
}

func (m *Market) IsSucceeded() bool {
	return m.succeeded
}

func (m *Market) IsPerp() bool {
	return m.perp
}

func (m *Market) StopSnapshots() {
	m.matching.StopSnapshots()
	m.position.StopSnapshots()
	m.liquidityEngine.StopSnapshots()
	m.settlement.StopSnapshots()
	m.tsCalc.StopSnapshots()
}

func (m *Market) Mkt() *types.Market {
	return m.mkt
}

func (m *Market) GetEquityShares() *common.EquityShares {
	return m.equityShares
}

func (m *Market) ResetParentIDAndInsurancePoolFraction() {
	m.mkt.ParentMarketID = ""
	m.mkt.InsurancePoolFraction = num.DecimalZero()
}

func (m *Market) GetParentMarketID() string {
	return m.mkt.ParentMarketID
}

func (m *Market) GetInsurancePoolFraction() num.Decimal {
	return m.mkt.InsurancePoolFraction
}

func (m *Market) SetSucceeded() {
	m.succeeded = true
}

func (m *Market) SetNextMTM(tm time.Time) {
	m.nextMTM = tm
}

func (m *Market) GetNextMTM() time.Time {
	return m.nextMTM
}

func (m *Market) GetSettlementAsset() string {
	return m.settlementAsset
}

func (m *Market) Update(ctx context.Context, config *types.Market, oracleEngine products.OracleEngine) error {
	config.TradingMode = m.mkt.TradingMode
	config.State = m.mkt.State
	config.MarketTimestamps = m.mkt.MarketTimestamps
	recalcMargins := !config.TradableInstrument.RiskModel.Equal(m.mkt.TradableInstrument.RiskModel)
	m.mkt = config
	assets, _ := config.GetAssets()
	m.settlementAsset = assets[0]

	if m.mkt.State == types.MarketStateTradingTerminated {
		m.tradableInstrument.Instrument.UnsubscribeSettlementData(ctx)
	} else {
		m.tradableInstrument.Instrument.Unsubscribe(ctx)
	}
	if err := m.tradableInstrument.UpdateInstrument(ctx, m.log, m.mkt.TradableInstrument, m.GetID(), oracleEngine, m.broker); err != nil {
		return err
	}
	m.risk.UpdateModel(m.stateVarEngine, m.tradableInstrument.MarginCalculator, m.tradableInstrument.RiskModel)
	m.settlement.UpdateProduct(m.tradableInstrument.Instrument.Product)
	m.tsCalc.UpdateParameters(*m.mkt.LiquidityMonitoringParameters.TargetStakeParameters)
	m.pMonitor.UpdateSettings(m.tradableInstrument.RiskModel, m.mkt.PriceMonitoringSettings)
	m.linearSlippageFactor = m.mkt.LinearSlippageFactor
	m.quadraticSlippageFactor = m.mkt.QuadraticSlippageFactor
	m.lMonitor.UpdateParameters(m.mkt.LiquidityMonitoringParameters)
	m.liquidity.UpdateMarketConfig(m.tradableInstrument.RiskModel, m.pMonitor, m.mkt.LiquiditySLAParams)

	// if we're already in trading terminated, not point to listen to trading termination oracle
	if m.perp {
		m.tradableInstrument.Instrument.Product.NotifyOnSettlementData(m.settlementDataPerp)
	} else if m.mkt.State != types.MarketStateTradingTerminated {
		m.tradableInstrument.Instrument.Product.NotifyOnTradingTerminated(m.tradingTerminated)
	} else {
		m.tradableInstrument.Instrument.UnsubscribeTradingTerminated(ctx)
	}
	if !m.perp {
		m.tradableInstrument.Instrument.Product.NotifyOnSettlementData(m.settlementData)
	}

	m.updateLiquidityFee(ctx)
	// risk model hasn't changed -> return
	if !recalcMargins {
		return nil
	}
	// We know the risk model has been updated, so we have to recalculate margin requirements
	m.recheckMargin(ctx, m.position.Positions())

	return nil
}

func (m *Market) IntoType() types.Market {
	return *m.mkt.DeepClone()
}

func (m *Market) Hash() []byte {
	mID := logging.String("market-id", m.GetID())
	matchingHash := m.matching.Hash()
	m.log.Debug("orderbook state hash", logging.Hash(matchingHash), mID)

	positionHash := m.position.Hash()
	m.log.Debug("positions state hash", logging.Hash(positionHash), mID)

	return crypto.Hash(append(matchingHash, positionHash...))
}

func (m *Market) GetMarketState() types.MarketState {
	return m.mkt.State
}

// priceToMarketPrecision
// It should never return a nil pointer.
func (m *Market) priceToMarketPrecision(price *num.Uint) *num.Uint {
	// we assume the price is cloned correctly already
	return price.Div(price, m.priceFactor)
}

func (m *Market) midPrice() *num.Uint {
	bestBidPrice, _, _ := m.matching.BestBidPriceAndVolume()
	bestOfferPrice, _, _ := m.matching.BestOfferPriceAndVolume()
	two := num.NewUint(2)
	midPrice := num.UintZero()
	if !bestBidPrice.IsZero() && !bestOfferPrice.IsZero() {
		midPrice = midPrice.Div(num.Sum(bestBidPrice, bestOfferPrice), two)
	}
	return midPrice
}

func (m *Market) GetMarketData() types.MarketData {
	bestBidPrice, bestBidVolume, _ := m.matching.BestBidPriceAndVolume()
	bestOfferPrice, bestOfferVolume, _ := m.matching.BestOfferPriceAndVolume()
	bestStaticBidPrice, bestStaticBidVolume, _ := m.getBestStaticBidPriceAndVolume()
	bestStaticOfferPrice, bestStaticOfferVolume, _ := m.getBestStaticAskPriceAndVolume()

	// Auction related values
	indicativePrice := num.UintZero()
	indicativeVolume := uint64(0)
	var auctionStart, auctionEnd int64
	if m.as.InAuction() {
		indicativePrice, indicativeVolume, _ = m.matching.GetIndicativePriceAndVolume()
		if t := m.as.Start(); !t.IsZero() {
			auctionStart = t.UnixNano()
		}
		if t := m.as.ExpiresAt(); t != nil {
			auctionEnd = t.UnixNano()
		}
	}

	// If we do not have one of the best_* prices, leave the mid price as zero
	two := num.NewUint(2)
	midPrice := num.UintZero()
	if !bestBidPrice.IsZero() && !bestOfferPrice.IsZero() {
		midPrice = midPrice.Div(num.Sum(bestBidPrice, bestOfferPrice), two)
	}

	staticMidPrice := num.UintZero()
	if !bestStaticBidPrice.IsZero() && !bestStaticOfferPrice.IsZero() {
		staticMidPrice = staticMidPrice.Div(num.Sum(bestStaticBidPrice, bestStaticOfferPrice), two)
	}

	var targetStake string
	if m.as.InAuction() {
		targetStake = m.getTheoreticalTargetStake().String()
	} else {
		targetStake = m.getTargetStake().String()
	}
	bounds := m.pMonitor.GetCurrentBounds()
	for _, b := range bounds {
		m.priceToMarketPrecision(b.MaxValidPrice) // effictively floors this
		m.priceToMarketPrecision(b.MinValidPrice)

		rp, _ := num.UintFromDecimal(b.ReferencePrice)
		m.priceToMarketPrecision(rp)
		b.ReferencePrice = num.DecimalFromUint(rp)

		if m.priceFactor.NEQ(common.One) {
			b.MinValidPrice.AddSum(common.One) // ceil
		}
	}
	mode := m.as.Mode()
	if m.mkt.TradingMode == types.MarketTradingModeNoTrading {
		mode = m.mkt.TradingMode
	}

	return types.MarketData{
		Market:                    m.GetID(),
		BestBidPrice:              m.priceToMarketPrecision(bestBidPrice),
		BestBidVolume:             bestBidVolume,
		BestOfferPrice:            m.priceToMarketPrecision(bestOfferPrice),
		BestOfferVolume:           bestOfferVolume,
		BestStaticBidPrice:        m.priceToMarketPrecision(bestStaticBidPrice),
		BestStaticBidVolume:       bestStaticBidVolume,
		BestStaticOfferPrice:      m.priceToMarketPrecision(bestStaticOfferPrice),
		BestStaticOfferVolume:     bestStaticOfferVolume,
		MidPrice:                  m.priceToMarketPrecision(midPrice),
		StaticMidPrice:            m.priceToMarketPrecision(staticMidPrice),
		MarkPrice:                 m.priceToMarketPrecision(m.getCurrentMarkPrice()),
		LastTradedPrice:           m.priceToMarketPrecision(m.getLastTradedPrice()),
		Timestamp:                 m.timeService.GetTimeNow().UnixNano(),
		OpenInterest:              m.position.GetOpenInterest(),
		IndicativePrice:           m.priceToMarketPrecision(indicativePrice),
		IndicativeVolume:          indicativeVolume,
		AuctionStart:              auctionStart,
		AuctionEnd:                auctionEnd,
		MarketTradingMode:         mode,
		MarketState:               m.mkt.State,
		Trigger:                   m.as.Trigger(),
		ExtensionTrigger:          m.as.ExtensionTrigger(),
		TargetStake:               targetStake,
		SuppliedStake:             m.getSuppliedStake().String(),
		PriceMonitoringBounds:     bounds,
		MarketValueProxy:          m.lastMarketValueProxy.BigInt().String(),
		LiquidityProviderFeeShare: m.equityShares.LpsToLiquidityProviderFeeShare(m.liquidityEngine.GetAverageLiquidityScores()),
		NextMTM:                   m.nextMTM.UnixNano(),
		MarketGrowth:              m.equityShares.GetMarketGrowth(),
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

func (m *Market) Reject(ctx context.Context) error {
	if !m.canReject() {
		return common.ErrCannotRejectMarketNotInProposedState
	}

	// we closed all parties accounts
	m.cleanupOnReject(ctx)
	m.mkt.State = types.MarketStateRejected
	m.mkt.TradingMode = types.MarketTradingModeNoTrading
	m.broker.Send(events.NewMarketUpdatedEvent(ctx, *m.mkt))

	return nil
}

func (m *Market) canReject() bool {
	if m.mkt.State == types.MarketStateProposed {
		return true
	}
	if len(m.mkt.ParentMarketID) == 0 {
		return false
	}
	// parent market is set, market can be in pending state when it is rejected.
	return m.mkt.State == types.MarketStatePending
}

func (m *Market) onTxProcessed() {
	m.risk.FlushMarginLevelsEvents()
}

// CanLeaveOpeningAuction checks if the market can leave the opening auction based on whether floating point consensus has been reached on all 3 vars.
func (m *Market) CanLeaveOpeningAuction() bool {
	boundFactorsInitialised := m.pMonitor.IsBoundFactorsInitialised()
	potInitialised := m.liquidity.IsProbabilityOfTradingInitialised()
	riskFactorsInitialised := m.risk.IsRiskFactorInitialised()
	canLeave := boundFactorsInitialised && riskFactorsInitialised && potInitialised
	if !canLeave {
		m.log.Info("Cannot leave opening auction", logging.String("market", m.mkt.ID), logging.Bool("bound-factors-initialised", boundFactorsInitialised), logging.Bool("risk-factors-initialised", riskFactorsInitialised))
	}
	return canLeave
}

func (m *Market) InheritParent(ctx context.Context, pstate *types.CPMarketState) {
	// add the trade value from the parent
	m.feeSplitter.SetTradeValue(pstate.LastTradeValue)
	m.equityShares.InheritELS(pstate.Shares)
}

func (m *Market) RestoreELS(ctx context.Context, pstate *types.CPMarketState) {
	m.equityShares.RestoreELS(pstate.Shares)
}

func (m *Market) RollbackInherit(ctx context.Context) {
	// the InheritParent call has to be made before checking if the market can leave opening auction
	// if the market did not leave opening auction, market state needs to be resored to what it was
	// before the call to InheritParent was made. Market is still in opening auction, therefore
	// feeSplitter trade value is zero, and equity shares are linear stake/vstake/ELS
	// do make sure this call is not made when the market is active
	if m.mkt.State == types.MarketStatePending {
		m.feeSplitter.SetTradeValue(num.UintZero())
		m.equityShares.RollbackParentELS()
	}
}

func (m *Market) StartOpeningAuction(ctx context.Context) error {
	if m.mkt.State != types.MarketStateProposed {
		return common.ErrCannotStartOpeningAuctionForMarketNotInProposedState
	}

	defer m.onTxProcessed()

	// now we start the opening auction
	if m.as.AuctionStart() {
		// we are now in a pending state
		m.mkt.State = types.MarketStatePending
		// this should no longer be needed
		// m.mkt.MarketTimestamps.Pending = m.timeService.GetTimeNow().UnixNano()
		m.mkt.TradingMode = types.MarketTradingModeOpeningAuction
		m.enterAuction(ctx)
	} else {
		// TODO(): to be removed once we don't have market starting
		// without an opening auction - this is only used in unit tests
		// validation on the proposal ensures opening auction duration is always >= 1 (or whatever the min duration is)
		m.mkt.State = types.MarketStateActive
		m.mkt.TradingMode = types.MarketTradingModeContinuous
	}

	m.broker.Send(events.NewMarketUpdatedEvent(ctx, *m.mkt))
	return nil
}

// GetID returns the id of the given market.
func (m *Market) GetID() string {
	return m.mkt.ID
}

func (m *Market) PostRestore(ctx context.Context) error {
	// tell the matching engine about the markets price factor so it can finish restoring orders
	m.matching.RestoreWithMarketPriceFactor(m.priceFactor)
	return nil
}

// OnTick notifies the market of a new time event/update.
// todo: make this a more generic function name e.g. OnTimeUpdateEvent
func (m *Market) OnTick(ctx context.Context, t time.Time) bool {
	defer m.onTxProcessed()

	timer := metrics.NewTimeCounter(m.mkt.ID, "market", "OnTick")
	m.mu.Lock()
	defer m.mu.Unlock()

	_, blockHash := vegacontext.TraceIDFromContext(ctx)
	// make deterministic ID for this market, concatenate
	// the block hash and the market ID
	m.idgen = idgeneration.New(blockHash + crypto.HashStrToHex(m.GetID()))
	// and we call next ID on this directly just so we don't have an ID which have
	// a different from others, we basically burn the first ID.
	_ = m.idgen.NextID()
	defer func() { m.idgen = nil }()

	if m.closed {
		return true
	}

	// first we expire orders
	if !m.closed && m.canTrade() {
		expired := m.removeExpiredOrders(ctx, t.UnixNano())
		metrics.OrderGaugeAdd(-len(expired), m.GetID())
		confirmations := m.removeExpiredStopOrders(ctx, t.UnixNano(), m.idgen)

		stopsExpired := 0
		for _, v := range confirmations {
			stopsExpired++
			for _, v := range v.PassiveOrdersAffected {
				if v.Status != types.OrderStatusActive {
					stopsExpired++
				}
			}
		}
		metrics.OrderGaugeAdd(-stopsExpired, m.GetID())
	}

	// some engines still needs to get updates:
	m.pMonitor.OnTimeUpdate(t)
	m.feeSplitter.SetCurrentTime(t)

	// TODO(): This also assume that the market is not
	// being closed before the market is leaving
	// the opening auction, but settlement at expiry is
	// not even specced or implemented as of now...
	// if the state of the market is just PROPOSED,
	// we will just skip everything there as nothing apply.
	if m.mkt.State == types.MarketStateProposed {
		return false
	}

	// if trading is terminated, we have nothing to do here.
	// we just need to wait for the settlementData to arrive through oracle
	if m.mkt.State == types.MarketStateTradingTerminated {
		return false
	}

	m.liquidity.OnTick(ctx, t)

	// check auction, if any. If we leave auction, MTM is performed in this call
	m.checkAuction(ctx, t, m.idgen)
	timer.EngineTimeCounterAdd()

	m.updateMarketValueProxy()
	m.updateLiquidityFee(ctx)
	m.broker.Send(events.NewMarketTick(ctx, m.mkt.ID, t))
	return m.closed
}

// BlockEnd notifies the market of the end of the block.
func (m *Market) BlockEnd(ctx context.Context) {
	defer m.onTxProcessed()

	// MTM if enough time has elapsed, we are not in auction, and we have a non-zero mark price.
	// we MTM in leaveAuction before deploying LP orders like we did before, but we do update nextMTM there
	var tID string
	ctx, tID = vegacontext.TraceIDFromContext(ctx)
	m.idgen = idgeneration.New(tID + crypto.HashStrToHex("blockend"+m.GetID()))
	defer func() {
		m.idgen = nil
	}()
	hasTraded := m.settlement.HasTraded()
	mp := m.getLastTradedPrice()
	if !hasTraded && m.markPrice != nil {
		// no trades happened, make sure we're just using the current mark price
		mp = m.markPrice.Clone()
	}
	t := m.timeService.GetTimeNow()
	if mp != nil && !mp.IsZero() && !m.as.InAuction() && (m.nextMTM.IsZero() || !m.nextMTM.After(t)) {
		m.markPrice = mp
		m.nextMTM = t.Add(m.mtmDelta) // add delta here
		if hasTraded {
			// only MTM if we have traded
			m.confirmMTM(ctx, false)
		}

		closedPositions := m.position.GetClosedPositions()

		if len(closedPositions) > 0 {
			m.releaseExcessMargin(ctx, closedPositions...)
			// also remove all stop orders
			m.removeAllStopOrders(ctx, closedPositions...)
		}
		// last traded price should not reflect the closeout trades
		m.lastTradedPrice = mp.Clone()
	}
	m.releaseExcessMargin(ctx, m.position.Positions()...)
	// send position events
	m.position.FlushPositionEvents(ctx)

	m.liquidity.EndBlock(m.markPrice, m.midPrice(), m.positionFactor)
}

func (m *Market) removeAllStopOrders(
	ctx context.Context,
	positions ...events.MarketPosition,
) {
	evts := []events.Event{}

	for _, v := range positions {
		sos, _ := m.stopOrders.Cancel(v.Party(), "")
		for _, so := range sos {
			if so.Expiry.Expires() {
				_ = m.expiringOrders.RemoveOrder(so.Expiry.ExpiresAt.UnixNano(), so.ID)
			}
			evts = append(evts, events.NewStopOrderEvent(ctx, so))
		}
	}

	if len(evts) > 0 {
		m.broker.SendBatch(evts)
	}
}

func (m *Market) updateMarketValueProxy() {
	// if windows length is reached, reset fee splitter
	if mvwl := m.marketValueWindowLength; m.feeSplitter.Elapsed() > mvwl {
		// AvgTradeValue calculates the rolling average trade value to include the current window (which is ending)
		m.equityShares.AvgTradeValue(m.feeSplitter.AvgTradeValue())
		// this increments the internal window counter
		m.feeSplitter.TimeWindowStart(m.timeService.GetTimeNow())
		// m.equityShares.UpdateVirtualStake() // this should always set the vStake >= physical stake?
	}

	// TODO karel - do we still need to calculate the market value proxy????
	// these need to happen every block
	// but also when new LP is submitted just so we are sure we do
	// not have a mvp of 0
	// ts := m.liquidity.Stake
	// m.lastMarketValueProxy = m.feeSplitter.MarketValueProxy(
	// 	m.marketValueWindowLength, ts)
}

func (m *Market) removeOrders(ctx context.Context) {
	// remove all order from the book
	// and send events with the stopped status
	orders := append(m.matching.Settled(), m.peggedOrders.Settled()...)
	orderEvents := make([]events.Event, 0, len(orders))
	for _, v := range orders {
		orderEvents = append(orderEvents, events.NewOrderEvent(ctx, v))
	}

	m.broker.SendBatch(orderEvents)
}

func (m *Market) cleanMarketWithState(ctx context.Context, mktState types.MarketState) error {
	parties := make([]string, 0, len(m.parties))
	for k := range m.parties {
		parties = append(parties, k)
	}

	// insurance pool has to be preserved in case a successor market leaves opening auction
	// the insurance pool must be preserved if a market is settled or was closed through governance
	keepInsurance := (mktState == types.MarketStateSettled || mktState == types.MarketStateClosed) && !m.succeeded
	sort.Strings(parties)
	clearMarketTransfers, err := m.collateral.ClearMarket(ctx, m.GetID(), m.settlementAsset, parties, keepInsurance)
	if err != nil {
		m.log.Error("Clear market error",
			logging.MarketID(m.GetID()),
			logging.Error(err))
		return err
	}

	// unregister state-variables
	m.stateVarEngine.UnregisterStateVariable(m.settlementAsset, m.mkt.ID)

	if len(clearMarketTransfers) > 0 {
		m.broker.Send(events.NewLedgerMovements(ctx, clearMarketTransfers))
	}

	m.mkt.State = mktState
	m.mkt.TradingMode = types.MarketTradingModeNoTrading
	m.mkt.MarketTimestamps.Close = m.timeService.GetTimeNow().UnixNano()
	m.broker.Send(events.NewMarketUpdatedEvent(ctx, *m.mkt))

	return nil
}

func (m *Market) closeCancelledMarket(ctx context.Context) error {
	// we got here because trading was terminated, so we've already unsubscribed that oracle data source.
	m.tradableInstrument.Instrument.UnsubscribeSettlementData(ctx)

	if err := m.cleanMarketWithState(ctx, types.MarketStateCancelled); err != nil {
		return err
	}

	m.liquidity.StopAllLiquidityProvision(ctx)

	m.closed = true

	return nil
}

func (m *Market) closeMarket(ctx context.Context, t time.Time, finalState types.MarketState, settlementPriceInAsset *num.Uint) error {
	positions, round, err := m.settlement.Settle(t, settlementPriceInAsset)
	if err != nil {
		m.log.Error("Failed to get settle positions on market closed",
			logging.Error(err))

		return err
	}

	transfers, err := m.collateral.FinalSettlement(ctx, m.GetID(), positions, round)
	if err != nil {
		m.log.Error("Failed to get ledger movements after settling closed market",
			logging.MarketID(m.GetID()),
			logging.Error(err))
		return err
	}

	m.tradableInstrument.Instrument.UnsubscribeSettlementData(ctx)
	// @TODO pass in correct context -> Previous or next block?
	// Which is most appropriate here?
	// this will be next block
	if len(transfers) > 0 {
		m.broker.Send(events.NewLedgerMovements(ctx, transfers))
	}

	// final distribution of liquidity fees
	if !m.finalFeesDistributed {
		if err := m.liquidity.AllocateFees(ctx); err != nil {
			m.log.Panic("failed to allocate liquidity provision fees", logging.Error(err))
		}

		m.liquidity.OnEpochEnd(ctx, t)
		m.finalFeesDistributed = true
	}

	err = m.cleanMarketWithState(ctx, finalState)
	if err != nil {
		return err
	}

	m.removeOrders(ctx)

	m.liquidity.StopAllLiquidityProvision(ctx)

	return nil
}

func (m *Market) unregisterAndReject(ctx context.Context, order *types.Order, err error) error {
	// in case the order was reduce only
	order.ClearUpExtraRemaining()

	_ = m.position.UnregisterOrder(ctx, order)
	order.UpdatedAt = m.timeService.GetTimeNow().UnixNano()
	order.Status = types.OrderStatusRejected
	if oerr, ok := types.IsOrderError(err); ok {
		// the order wasn't invalid, so stopped is a better status, rather than rejected.
		if types.IsStoppingOrder(oerr) {
			order.Status = types.OrderStatusStopped
		}
		order.Reason = oerr
	} else {
		// should not happened but still...
		order.Reason = types.OrderErrorInternalError
	}
	m.broker.Send(events.NewOrderEvent(ctx, order))
	if m.log.GetLevel() == logging.DebugLevel {
		m.log.Debug("Failure after submitting order to matching engine",
			logging.Order(*order),
			logging.Error(err))
	}
	return err
}

func (m *Market) getNewPeggedPrice(order *types.Order) (*num.Uint, error) {
	if m.as.InAuction() {
		return num.UintZero(), common.ErrCannotRepriceDuringAuction
	}

	var (
		err   error
		price *num.Uint
	)

	switch order.PeggedOrder.Reference {
	case types.PeggedReferenceMid:
		price, err = m.getStaticMidPrice(order.Side)
	case types.PeggedReferenceBestBid:
		price, err = m.getBestStaticBidPrice()
	case types.PeggedReferenceBestAsk:
		price, err = m.getBestStaticAskPrice()
	}
	if err != nil {
		return num.UintZero(), common.ErrUnableToReprice
	}

	offset := num.UintZero().Mul(order.PeggedOrder.Offset, m.priceFactor)
	if order.Side == types.SideSell {
		return price.AddSum(offset), nil
	}

	if price.LTE(offset) {
		return num.UintZero(), common.ErrUnableToReprice
	}

	return num.UintZero().Sub(price, offset), nil
}

// Reprice a pegged order. This only updates the price on the order.
func (m *Market) repricePeggedOrder(order *types.Order) error {
	// Work out the new price of the order
	price, err := m.getNewPeggedPrice(order)
	if err != nil {
		return err
	}
	original := price.Clone()
	order.OriginalPrice = original.Div(original, m.priceFactor) // set original price in market precision
	order.Price = price
	return nil
}

func (m *Market) parkAllPeggedOrders(ctx context.Context) []*types.Order {
	toParkIDs := m.matching.GetActivePeggedOrderIDs()

	parked := make([]*types.Order, 0, len(toParkIDs))
	for _, order := range toParkIDs {
		parked = append(parked, m.parkOrder(ctx, order))
	}
	return parked
}

func (m *Market) uncrossOrderAtAuctionEnd(ctx context.Context) {
	if !m.as.InAuction() || m.as.IsOpeningAuction() {
		return
	}
	m.uncrossOnLeaveAuction(ctx)
}

func (m *Market) UpdateMarketState(ctx context.Context, changes *types.MarketStateUpdateConfiguration) error {
	_, blockHash := vegacontext.TraceIDFromContext(ctx)
	// make deterministic ID for this market, concatenate
	// the block hash and the market ID
	m.idgen = idgeneration.New(blockHash + crypto.HashStrToHex(m.GetID()))
	// and we call next ID on this directly just so we don't have an ID which have
	// a different from others, we basically burn the first ID.
	_ = m.idgen.NextID()
	defer func() { m.idgen = nil }()
	if changes.UpdateType == types.MarketStateUpdateTypeTerminate {
		m.uncrossOrderAtAuctionEnd(ctx)
		// terminate and settle data (either last traded price for perp, or settlement data provided via governance
		m.tradingTerminatedWithFinalState(ctx, types.MarketStateClosed, num.UintZero().Mul(changes.SettlementPrice, m.priceFactor))
	} else if changes.UpdateType == types.MarketStateUpdateTypeSuspend {
		m.mkt.State = types.MarketStateSuspendedViaGovernance
		m.mkt.TradingMode = types.MarketTradingModeSuspendedViaGovernance
		if m.as.InAuction() {
			m.as.ExtendAuctionSuspension(types.AuctionDuration{Duration: int64(m.minDuration)})
			evt := m.as.AuctionExtended(ctx, m.timeService.GetTimeNow())
			if evt != nil {
				m.broker.Send(evt)
			}
		} else {
			m.as.StartGovernanceSuspensionAuction(m.timeService.GetTimeNow())
			m.enterAuction(ctx)
			m.broker.Send(events.NewMarketUpdatedEvent(ctx, *m.mkt))
		}
	} else if changes.UpdateType == types.MarketStateUpdateTypeResume && m.mkt.State == types.MarketStateSuspendedViaGovernance {
		if m.as.GetState().Trigger == types.AuctionTriggerGovernanceSuspension && m.as.GetState().Extension == types.AuctionTriggerUnspecified {
			m.as.EndGovernanceSuspensionAuction()
			m.leaveAuction(ctx, m.timeService.GetTimeNow())
		} else {
			if m.as.GetState().Trigger == types.AuctionTriggerOpening {
				m.mkt.State = types.MarketStatePending
				m.mkt.TradingMode = types.MarketTradingModeOpeningAuction
			} else {
				m.mkt.State = types.MarketStateSuspended
				m.mkt.TradingMode = types.MarketTradingModeMonitoringAuction
			}
			m.checkAuction(ctx, m.timeService.GetTimeNow(), m.idgen)
			m.broker.Send(events.NewMarketUpdatedEvent(ctx, *m.mkt))
		}
	}
	return nil
}

// EnterAuction : Prepare the order book to be run as an auction.
func (m *Market) enterAuction(ctx context.Context) {
	// Change market type to auction
	ordersToCancel := m.matching.EnterAuction()

	// Move into auction mode to prevent pegged order repricing
	event := m.as.AuctionStarted(ctx, m.timeService.GetTimeNow())

	// Cancel all the orders that were invalid
	for _, order := range ordersToCancel {
		_, err := m.cancelOrder(ctx, order.Party, order.ID)
		if err != nil {
			m.log.Debug("error cancelling order when entering auction",
				logging.MarketID(m.GetID()),
				logging.OrderID(order.ID),
				logging.Error(err))
		}
	}

	// now update all special orders
	m.enterAuctionSpecialOrders(ctx)

	// Send an event bus update
	m.broker.Send(event)

	if m.as.InAuction() && (m.as.IsLiquidityAuction() || m.as.IsPriceAuction()) {
		m.mkt.State = types.MarketStateSuspended
		m.mkt.TradingMode = types.MarketTradingModeMonitoringAuction
		m.broker.Send(events.NewMarketUpdatedEvent(ctx, *m.mkt))
	}
}

func (m *Market) uncrossOnLeaveAuction(ctx context.Context) ([]*types.OrderConfirmation, []*types.Order) {
	uncrossedOrders, ordersToCancel, err := m.matching.LeaveAuction(m.timeService.GetTimeNow())
	if err != nil {
		m.log.Error("Error leaving auction", logging.Error(err))
	}

	// Process each confirmation & apply fee calculations to each trade
	evts := make([]events.Event, 0, len(uncrossedOrders))
	for _, uncrossedOrder := range uncrossedOrders {
		// handle fees first
		err := m.applyFees(ctx, uncrossedOrder.Order, uncrossedOrder.Trades)
		if err != nil {
			// @TODO this ought to be an event
			m.log.Error("Unable to apply fees to order",
				logging.String("OrderID", uncrossedOrder.Order.ID))
		}

		// then do the confirmation
		m.handleConfirmation(ctx, uncrossedOrder)

		if uncrossedOrder.Order.Remaining == 0 {
			uncrossedOrder.Order.Status = types.OrderStatusFilled
		}
		evts = append(evts, events.NewOrderEvent(ctx, uncrossedOrder.Order))
	}

	// send order events in a single batch, it's more efficient
	m.broker.SendBatch(evts)
	return uncrossedOrders, ordersToCancel
}

// OnOpeningAuctionFirstUncrossingPrice is triggered when the opening auction sees an uncrossing price for the first time and emits
// an event to the state variable engine.
func (m *Market) OnOpeningAuctionFirstUncrossingPrice() {
	m.log.Info("OnOpeningAuctionFirstUncrossingPrice event fired", logging.String("market", m.mkt.ID))
	m.stateVarEngine.ReadyForTimeTrigger(m.settlementAsset, m.mkt.ID)
	m.stateVarEngine.NewEvent(m.settlementAsset, m.mkt.ID, statevar.EventTypeOpeningAuctionFirstUncrossingPrice)
}

// OnAuctionEnded is called whenever an auction is ended and emits an event to the state var engine.
func (m *Market) OnAuctionEnded() {
	m.log.Info("OnAuctionEnded event fired", logging.String("market", m.mkt.ID))
	m.stateVarEngine.NewEvent(m.settlementAsset, m.mkt.ID, statevar.EventTypeAuctionEnded)
}

// leaveAuction : Return the orderbook and market to continuous trading.
func (m *Market) leaveAuction(ctx context.Context, now time.Time) {
	defer func() {
		if !m.as.InAuction() && (m.mkt.State == types.MarketStateSuspended || m.mkt.State == types.MarketStatePending || m.mkt.State == types.MarketStateSuspendedViaGovernance) {
			if m.mkt.State == types.MarketStatePending {
				// the market is now properly open,
				// so set the timestamp to when the opening auction actually ended
				m.mkt.MarketTimestamps.Open = now.UnixNano()
			}

			m.mkt.State = types.MarketStateActive
			m.mkt.TradingMode = types.MarketTradingModeContinuous
			m.broker.Send(events.NewMarketUpdatedEvent(ctx, *m.mkt))

			m.updateLiquidityFee(ctx)
			m.OnAuctionEnded()
		}
	}()

	uncrossedOrders, ordersToCancel := m.uncrossOnLeaveAuction(ctx)
	// will hold all orders which have been updated by the uncrossing
	// or which were cancelled at end of auction
	updatedOrders := []*types.Order{}

	// Process each order we have to cancel
	for _, order := range ordersToCancel {
		conf, err := m.cancelOrder(ctx, order.Party, order.ID)
		if err != nil {
			m.log.Panic("Failed to cancel order",
				logging.Error(err),
				logging.String("OrderID", order.ID))
		}

		updatedOrders = append(updatedOrders, conf.Order)
	}

	// update auction state, so we know what the new tradeMode ought to be
	endEvt := m.as.Left(ctx, now)

	for _, uncrossedOrder := range uncrossedOrders {
		updatedOrders = append(updatedOrders, uncrossedOrder.Order)
		updatedOrders = append(
			updatedOrders, uncrossedOrder.PassiveOrdersAffected...)
	}

	previousMarkPrice := m.getCurrentMarkPrice()
	// set the mark price here so that margins checks for special orders use the correct value
	m.markPrice = m.getLastTradedPrice()

	m.checkForReferenceMoves(ctx, updatedOrders, true)

	m.checkLiquidity(ctx, nil, true)
	m.commandLiquidityAuction(ctx)

	if !m.as.InAuction() {
		// only send the auction-left event if we actually *left* the auction.
		m.broker.Send(endEvt)
		// now that we've left the auction and all the orders have been unparked,
		// we can mark all positions using the margin calculation method appropriate
		// for non-auction mode and carry out any closeouts if need be
		m.confirmMTM(ctx, false)
		// set next MTM
		m.nextMTM = m.timeService.GetTimeNow().Add(m.mtmDelta)
	} else {
		// revert to old mark price if we're not leaving the auction after all
		m.markPrice = previousMarkPrice
	}
}

func (m *Market) validateOrder(ctx context.Context, order *types.Order) (err error) {
	defer func() {
		if err != nil {
			order.Status = types.OrderStatusRejected
			m.broker.Send(events.NewOrderEvent(ctx, order))
		}
	}()

	// Check we are allowed to handle this order type with the current market status
	isAuction := m.as.InAuction()
	if isAuction && order.TimeInForce == types.OrderTimeInForceGFN {
		order.Status = types.OrderStatusRejected
		order.Reason = types.OrderErrorCannotSendGFNOrderDuringAnAuction
		return common.ErrGFNOrderReceivedAuctionTrading
	}

	if isAuction && order.TimeInForce == types.OrderTimeInForceIOC {
		order.Reason = types.OrderErrorCannotSendIOCOrderDuringAuction
		return common.ErrIOCOrderReceivedAuctionTrading
	}

	if isAuction && order.TimeInForce == types.OrderTimeInForceFOK {
		order.Reason = types.OrderErrorCannotSendFOKOrderDurinAuction
		return common.ErrFOKOrderReceivedAuctionTrading
	}

	if !isAuction && order.TimeInForce == types.OrderTimeInForceGFA {
		order.Reason = types.OrderErrorGFAOrderDuringContinuousTrading
		return common.ErrGFAOrderReceivedDuringContinuousTrading
	}

	// Check the expiry time is valid
	if order.ExpiresAt > 0 && order.ExpiresAt < order.CreatedAt {
		order.Reason = types.OrderErrorInvalidExpirationDatetime
		return common.ErrInvalidExpiresAtTime
	}

	if m.closed {
		// adding order to the buffer first
		order.Reason = types.OrderErrorMarketClosed
		return common.ErrMarketClosed
	}

	if order.Type == types.OrderTypeNetwork {
		order.Reason = types.OrderErrorInvalidType
		return common.ErrInvalidOrderType
	}

	// Validate market
	if order.MarketID != m.mkt.ID {
		// adding order to the buffer first
		order.Reason = types.OrderErrorInvalidMarketID
		if m.log.GetLevel() == logging.DebugLevel {
			m.log.Debug("Market ID mismatch",
				logging.Order(*order),
				logging.String("market", m.mkt.ID))
		}
		return types.ErrInvalidMarketID
	}

	// Validate pegged orders
	if order.PeggedOrder != nil {
		if reason := order.ValidatePeggedOrder(); reason != types.OrderErrorUnspecified {
			order.Reason = reason
			if m.log.GetLevel() == logging.DebugLevel {
				m.log.Debug("Failed to validate pegged order details",
					logging.Order(*order),
					logging.String("market", m.mkt.ID))
			}
			return reason
		}
	}

	return nil
}

func (m *Market) validateAccounts(ctx context.Context, order *types.Order) error {
	if !m.collateral.HasGeneralAccount(order.Party, m.settlementAsset) {
		// adding order to the buffer first
		order.Status = types.OrderStatusRejected
		order.Reason = types.OrderErrorInsufficientAssetBalance
		m.broker.Send(events.NewOrderEvent(ctx, order))

		// party should be created before even trying to post order
		return common.ErrPartyInsufficientAssetBalance
	}

	// ensure party have a general account, and margin account is / can be created
	_, err := m.collateral.CreatePartyMarginAccount(ctx, order.Party, order.MarketID, m.settlementAsset)
	if err != nil {
		m.log.Error("Margin account verification failed",
			logging.String("party-id", order.Party),
			logging.String("market-id", m.GetID()),
			logging.String("asset", m.settlementAsset),
		)
		// adding order to the buffer first
		order.Status = types.OrderStatusRejected
		order.Reason = types.OrderErrorMissingGeneralAccount
		m.broker.Send(events.NewOrderEvent(ctx, order))
		return common.ErrMissingGeneralAccountForParty
	}

	// from this point we know the party have a margin account
	// we had it to the list of parties.
	m.addParty(order.Party)
	return nil
}

func (m *Market) releaseMarginExcess(ctx context.Context, partyID string) {
	// if this position went 0
	pos, ok := m.position.GetPositionByPartyID(partyID)
	if !ok {
		// the party has closed their position and it's been removed from the
		// position engine. Let's just create an empty one, so it can be cleared
		// down the line.
		pos = positions.NewMarketPosition(partyID)
	}
	m.releaseExcessMargin(ctx, pos)
}

// releaseExcessMargin does what releaseMarginExcess does. Added this function to be able to release
// all excess margin on MTM without having to call the latter by iterating all positions, and then
// fetching said position again my party.
func (m *Market) releaseExcessMargin(ctx context.Context, positions ...events.MarketPosition) {
	evts := make([]events.Event, 0, len(positions))
	for _, pos := range positions {
		// if the party still have a position in the settlement engine,
		// do not remove them for now
		if m.settlement.HasPosition(pos.Party()) {
			continue
		}

		// now check if all buy/sell/size are 0
		if pos.Buy() != 0 || pos.Sell() != 0 || pos.Size() != 0 {
			// position is not 0, nothing to release surely
			continue
		}

		transfers, err := m.collateral.ClearPartyMarginAccount(
			ctx, pos.Party(), m.GetID(), m.settlementAsset)
		if err != nil {
			m.log.Error("unable to clear party margin account", logging.Error(err))
			return
		}

		if transfers != nil {
			evts = append(evts, events.NewLedgerMovements(
				ctx, []*types.LedgerMovement{transfers}),
			)
		}

		// we can delete the party from the map here
		delete(m.parties, pos.Party())
	}
	m.broker.SendBatch(evts)
}

func rejectStopOrders(rejectionReason types.StopOrderRejectionReason, orders ...*types.StopOrder) {
	for _, o := range orders {
		if o != nil {
			o.Status = types.StopOrderStatusRejected
			o.RejectionReason = ptr.From(rejectionReason)
		}
	}
}

func (m *Market) SubmitStopOrdersWithIDGeneratorAndOrderIDs(
	ctx context.Context,
	submission *types.StopOrdersSubmission,
	party string,
	idgen common.IDGenerator,
	fallsBelowID, risesAboveID *string,
) (*types.OrderConfirmation, error) {
	m.idgen = idgen
	defer func() { m.idgen = nil }()

	fallsBelow, risesAbove := submission.IntoStopOrders(
		party, ptr.UnBox(fallsBelowID), ptr.UnBox(risesAboveID), m.timeService.GetTimeNow())

	defer func() {
		evts := []events.Event{}
		if fallsBelow != nil {
			evts = append(evts, events.NewStopOrderEvent(ctx, fallsBelow))
		}
		if risesAbove != nil {
			evts = append(evts, events.NewStopOrderEvent(ctx, risesAbove))
		}

		if len(evts) > 0 {
			m.broker.SendBatch(evts)
		}
	}()

	if !m.canTrade() {
		rejectStopOrders(types.StopOrderRejectionTradingNotAllowed, fallsBelow, risesAbove)
		return nil, common.ErrTradingNotAllowed
	}

	now := m.timeService.GetTimeNow()
	orderCnt := 0
	if fallsBelow != nil {
		if fallsBelow.Expiry.Expires() && fallsBelow.Expiry.ExpiresAt.Before(now) {
			rejectStopOrders(types.StopOrderRejectionExpiryInThePast, fallsBelow, risesAbove)
			return nil, common.ErrStopOrderExpiryInThePast
		}
		if !fallsBelow.OrderSubmission.ReduceOnly {
			rejectStopOrders(types.StopOrderRejectionMustBeReduceOnly, fallsBelow, risesAbove)
			return nil, common.ErrStopOrderMustBeReduceOnly
		}
		orderCnt++
	}
	if risesAbove != nil {
		if risesAbove.Expiry.Expires() && risesAbove.Expiry.ExpiresAt.Before(now) {
			rejectStopOrders(types.StopOrderRejectionExpiryInThePast, fallsBelow, risesAbove)
			return nil, common.ErrStopOrderExpiryInThePast
		}
		if !risesAbove.OrderSubmission.ReduceOnly {
			rejectStopOrders(types.StopOrderRejectionMustBeReduceOnly, fallsBelow, risesAbove)
			return nil, common.ErrStopOrderMustBeReduceOnly
		}
		orderCnt++
	}

	// now check if that party hasn't exceeded the max amount per market
	if m.stopOrders.CountForParty(party)+uint64(orderCnt) > m.maxStopOrdersPerParties.Uint64() {
		rejectStopOrders(types.StopOrderRejectionMaxStopOrdersPerPartyReached, fallsBelow, risesAbove)
		return nil, common.ErrMaxStopOrdersPerPartyReached
	}

	// now check for the parties position
	positions := m.position.GetPositionsByParty(party)
	if len(positions) > 1 {
		m.log.Panic("only one position expected", logging.Int("got", len(positions)))
	}

	if len(positions) < 1 {
		rejectStopOrders(types.StopOrderRejectionNotAllowedWithoutAPosition, fallsBelow, risesAbove)
		return nil, common.ErrStopOrderSubmissionNotAllowedWithoutExistingPosition
	}

	pos := positions[0]

	// now we will reject if the direction of order if is not
	// going to close the position or potential position
	potentialSize := pos.Size() - pos.Sell() + pos.Buy()
	size := pos.Size()

	var stopOrderSide types.Side
	if fallsBelow != nil {
		stopOrderSide = fallsBelow.OrderSubmission.Side
	} else {
		stopOrderSide = risesAbove.OrderSubmission.Side
	}

	switch stopOrderSide {
	case types.SideBuy:
		if potentialSize >= 0 && size >= 0 {
			rejectStopOrders(types.StopOrderRejectionNotClosingThePosition, fallsBelow, risesAbove)
			return nil, common.ErrStopOrderSideNotClosingThePosition
		}
	case types.SideSell:
		if potentialSize <= 0 && size <= 0 {
			rejectStopOrders(types.StopOrderRejectionNotClosingThePosition, fallsBelow, risesAbove)
			return nil, common.ErrStopOrderSideNotClosingThePosition
		}
	}

	fallsBelowTriggered, risesAboveTriggered := m.stopOrderWouldTriggerAtSubmission(fallsBelow),
		m.stopOrderWouldTriggerAtSubmission(risesAbove)
	triggered := fallsBelowTriggered || risesAboveTriggered

	// if we are in an auction
	// or no order is triggered
	// let's just submit it straight away
	if m.as.InAuction() || !triggered {
		m.poolStopOrders(fallsBelow, risesAbove)
		return nil, nil
	}

	var confirmation *types.OrderConfirmation
	var err error
	// now would the order get trigger straight away?
	switch {
	case fallsBelowTriggered:
		fallsBelow.Status = types.StopOrderStatusTriggered
		if risesAbove != nil {
			risesAbove.Status = types.StopOrderStatusStopped
		}
		fallsBelow.OrderID = idgen.NextID()
		confirmation, err = m.SubmitOrderWithIDGeneratorAndOrderID(
			ctx, fallsBelow.OrderSubmission, party, idgen, fallsBelow.OrderID, true,
		)
		if err != nil && confirmation != nil {
			fallsBelow.OrderID = confirmation.Order.ID
		}
	case risesAboveTriggered:
		risesAbove.Status = types.StopOrderStatusTriggered
		if fallsBelow != nil {
			fallsBelow.Status = types.StopOrderStatusStopped
		}
		risesAbove.OrderID = idgen.NextID()
		confirmation, err = m.SubmitOrderWithIDGeneratorAndOrderID(
			ctx, risesAbove.OrderSubmission, party, idgen, risesAbove.OrderID, true,
		)
		if err != nil && confirmation != nil {
			risesAbove.OrderID = confirmation.Order.ID
		}
	}

	return confirmation, err
}

func (m *Market) poolStopOrders(
	fallsBelow, risesAbove *types.StopOrder,
) {
	if fallsBelow != nil {
		m.stopOrders.Insert(fallsBelow)
		if fallsBelow.Expiry.Expires() {
			m.expiringStopOrders.Insert(fallsBelow.ID, fallsBelow.Expiry.ExpiresAt.UnixNano())
		}
	}
	if risesAbove != nil {
		m.stopOrders.Insert(risesAbove)
		if risesAbove.Expiry.Expires() {
			m.expiringStopOrders.Insert(risesAbove.ID, risesAbove.Expiry.ExpiresAt.UnixNano())
		}
	}
}

func (m *Market) stopOrderWouldTriggerAtSubmission(
	stopOrder *types.StopOrder,
) bool {
	if m.lastTradedPrice == nil || stopOrder == nil || stopOrder.Trigger.IsTrailingPercenOffset() {
		return false
	}

	lastTradedPrice := m.priceToMarketPrecision(m.getLastTradedPrice())

	switch stopOrder.Trigger.Direction {
	case types.StopOrderTriggerDirectionFallsBelow:
		if lastTradedPrice.LTE(stopOrder.Trigger.Price()) {
			return true
		}
	case types.StopOrderTriggerDirectionRisesAbove:
		if lastTradedPrice.GTE(stopOrder.Trigger.Price()) {
			return true
		}
	}
	return false
}

func (m *Market) triggerStopOrders(
	ctx context.Context,
	idgen common.IDGenerator,
) []*types.OrderConfirmation {
	if m.lastTradedPrice == nil {
		return nil
	}

	lastTradedPrice := m.priceToMarketPrecision(m.getLastTradedPrice())
	triggered, cancelled := m.stopOrders.PriceUpdated(lastTradedPrice)

	if len(triggered) <= 0 {
		return nil
	}

	now := m.timeService.GetTimeNow()
	// remove from expiring orders + set updatedAt
	for _, v := range append(triggered, cancelled...) {
		v.UpdatedAt = now
		if v.Expiry.Expires() {
			m.expiringStopOrders.RemoveOrder(v.Expiry.ExpiresAt.UnixNano(), v.ID)
		}
	}

	evts := make([]events.Event, 0, len(cancelled))
	for _, v := range cancelled {
		evts = append(evts, events.NewStopOrderEvent(ctx, v))
	}

	m.broker.SendBatch(evts)

	confirmations := m.submitStopOrders(ctx, triggered, types.StopOrderStatusTriggered, idgen)

	return append(m.triggerStopOrders(ctx, idgen), confirmations...)
}

// SubmitOrder submits the given order.
func (m *Market) SubmitOrder(
	ctx context.Context,
	orderSubmission *types.OrderSubmission,
	party string,
	deterministicID string,
) (oc *types.OrderConfirmation, _ error) {
	idgen := idgeneration.New(deterministicID)
	return m.SubmitOrderWithIDGeneratorAndOrderID(
		ctx, orderSubmission, party, idgen, idgen.NextID(), true,
	)
}

// SubmitOrder submits the given order.
func (m *Market) SubmitOrderWithIDGeneratorAndOrderID(
	ctx context.Context,
	orderSubmission *types.OrderSubmission,
	party string,
	idgen common.IDGenerator,
	orderID string,
	checkForTriggers bool,
) (oc *types.OrderConfirmation, _ error) {
	defer m.onTxProcessed()

	m.idgen = idgen
	defer func() { m.idgen = nil }()

	defer func() {
		if !checkForTriggers {
			return
		}

		m.triggerStopOrders(ctx, idgen)
	}()
	order := orderSubmission.IntoOrder(party)
	if order.Price != nil {
		order.OriginalPrice = order.Price.Clone()
		order.Price.Mul(order.Price, m.priceFactor)
	}
	order.CreatedAt = m.timeService.GetTimeNow().UnixNano()
	order.ID = orderID

	if !m.canTrade() {
		order.Status = types.OrderStatusRejected
		order.Reason = types.OrderErrorMarketClosed
		m.broker.Send(events.NewOrderEvent(ctx, order))
		return nil, common.ErrTradingNotAllowed
	}

	conf, orderUpdates, err := m.submitOrder(ctx, order)
	if err != nil {
		return nil, err
	}

	allUpdatedOrders := append(
		[]*types.Order{conf.Order}, conf.PassiveOrdersAffected...)
	allUpdatedOrders = append(allUpdatedOrders, orderUpdates...)

	if !m.as.InAuction() {
		m.checkForReferenceMoves(ctx, allUpdatedOrders, false)
	}

	return conf, nil
}

func (m *Market) submitOrder(ctx context.Context, order *types.Order) (*types.OrderConfirmation, []*types.Order, error) {
	timer := metrics.NewTimeCounter(m.mkt.ID, "market", "SubmitOrder")
	orderValidity := "invalid"
	defer func() {
		timer.EngineTimeCounterAdd()
		metrics.OrderCounterInc(m.mkt.ID, orderValidity)
	}()

	// set those at the beginning as even rejected order get through the buffers
	order.Version = common.InitialOrderVersion
	order.Status = types.OrderStatusActive

	if err := m.validateOrder(ctx, order); err != nil {
		return nil, nil, err
	}

	if err := m.validateAccounts(ctx, order); err != nil {
		return nil, nil, err
	}

	// Now that validation is handled, call the code to place the order
	orderConf, orderUpdates, err := m.submitValidatedOrder(ctx, order)
	if err == nil {
		orderValidity = "valid"
	}

	if order.PeggedOrder != nil && order.IsFinished() {
		// remove the pegged order from anywhere
		m.removePeggedOrder(order)
	}

	// insert an expiring order if it's either in the book
	// or in the parked list
	if order.IsExpireable() && !order.IsFinished() {
		m.expiringOrders.Insert(order.ID, order.ExpiresAt)
	}

	return orderConf, orderUpdates, err
}

func (m *Market) submitValidatedOrder(ctx context.Context, order *types.Order) (*types.OrderConfirmation, []*types.Order, error) {
	isPegged := order.PeggedOrder != nil
	if isPegged {
		order.Status = types.OrderStatusParked
		order.Reason = types.OrderErrorUnspecified

		if m.as.InAuction() {
			order.SetIcebergPeaks()

			m.peggedOrders.Park(order)
			// If we are in an auction, we don't insert this order into the book
			// Maybe should return an orderConfirmation with order state PARKED
			m.broker.Send(events.NewOrderEvent(ctx, order))
			return &types.OrderConfirmation{Order: order}, nil, nil
		}
		// Reprice
		err := m.repricePeggedOrder(order)
		if err != nil {
			order.SetIcebergPeaks()
			m.peggedOrders.Park(order)
			m.broker.Send(events.NewOrderEvent(ctx, order))
			return &types.OrderConfirmation{Order: order}, nil, nil // nolint
		}
	}

	// Register order as potential positions
	pos := m.position.RegisterOrder(ctx, order)

	// in case we have an IOC order, that would work but need to be stopped because
	// it'd be flipping the position of the party
	// first check if we have a reduce only order and make sure it can go through
	if order.ReduceOnly {
		reduce, extraSize := pos.OrderReducesOnlyExposure(order)
		// if we are not reducing, or if the position flips on a FOK, we short-circuit here.
		// in the case of a IOC, the order will be stopped once we reach 0
		if !reduce || (order.TimeInForce == types.OrderTimeInForceFOK && extraSize > 0) {
			return nil, nil, m.unregisterAndReject(
				ctx, order, types.ErrReduceOnlyOrderWouldNotReducePosition)
		}
		// keep track of the eventual reduce only size
		order.ReduceOnlyAdjustRemaining(extraSize)
	}

	// Perform check and allocate margin unless the order is (partially) closing the party position
	if !order.ReduceOnly && !pos.OrderReducesExposure(order) {
		if err := m.checkMarginForOrder(ctx, pos, order); err != nil {
			if m.log.GetLevel() <= logging.DebugLevel {
				m.log.Debug("Unable to check/add margin for party",
					logging.Order(*order), logging.Error(err))
			}
			_ = m.unregisterAndReject(
				ctx, order, types.OrderErrorMarginCheckFailed)
			return nil, nil, common.ErrMarginCheckFailed
		}
	}

	// from here we may have assigned some margin.
	// we add the check to roll it back in case we have a 0 positions after this
	defer m.releaseMarginExcess(ctx, order.Party)

	// If we are not in an opening auction, apply fees
	var trades []*types.Trade
	// we're not in auction (not opening, not any other auction
	if !m.as.InAuction() {
		// first we call the order book to evaluate auction triggers and get the list of trades
		var err error
		trades, err = m.checkPriceAndGetTrades(ctx, order)
		if err != nil {
			return nil, nil, m.unregisterAndReject(ctx, order, err)
		}

		// try to apply fees on the trade
		err = m.applyFees(ctx, order, trades)
		if err != nil {
			return nil, nil, m.unregisterAndReject(ctx, order, err)
		}
	}

	// if an auction was trigger, and we are a pegged order
	// or a liquidity order, let's return now.
	if m.as.InAuction() && isPegged {
		if isPegged {
			m.peggedOrders.Park(order)
		}
		// parking the order, needs to unregister it first
		_ = m.position.UnregisterOrder(ctx, order)
		m.broker.Send(events.NewOrderEvent(ctx, order))
		return &types.OrderConfirmation{Order: order}, nil, nil
	}

	order.Status = types.OrderStatusActive

	// Send the aggressive order into matching engine
	confirmation, err := m.matching.SubmitOrder(order)
	if err != nil {
		return nil, nil, m.unregisterAndReject(ctx, order, err)
	}

	// this is no op for non reduce-only orders
	order.ClearUpExtraRemaining()

	// this means our reduce-only order (IOC) have been stopped
	// from trading to the point it would flip the position,
	// and successfully reduced the position to 0.
	// set the status to Stopped then.
	if order.ReduceOnly && order.Remaining > 0 {
		order.Status = types.OrderStatusStopped
	}

	// if the order is not staying in the book, then we remove it
	// from the potential positions
	if order.IsFinished() && order.Remaining > 0 {
		_ = m.position.UnregisterOrder(ctx, order)
	}

	// we replace the trades in the confirmation with the one we got initially
	// the contains the fees information
	confirmation.Trades = trades

	// Send out the order update here as handling the confirmation message
	// below might trigger an action that can change the order details.
	m.broker.Send(events.NewOrderEvent(ctx, order))

	orderUpdates := m.handleConfirmation(ctx, confirmation)
	return confirmation, orderUpdates, nil
}

func (m *Market) checkPriceAndGetTrades(ctx context.Context, order *types.Order) ([]*types.Trade, error) {
	trades, err := m.matching.GetTrades(order)
	if err != nil {
		return nil, err
	}

	if order.PostOnly && len(trades) > 0 {
		return nil, types.OrderErrorPostOnlyOrderWouldTrade
	}

	persistent := true
	switch order.TimeInForce {
	case types.OrderTimeInForceFOK, types.OrderTimeInForceGFN, types.OrderTimeInForceIOC:
		persistent = false
	}

	if m.pMonitor.CheckPrice(ctx, m.as, trades, persistent) {
		return nil, types.OrderErrorNonPersistentOrderOutOfPriceBounds
	}

	if evt := m.as.AuctionExtended(ctx, m.timeService.GetTimeNow()); evt != nil {
		m.broker.Send(evt)
	}

	// start the  monitoring auction if required?
	if m.as.AuctionStart() {
		m.enterAuction(ctx)
		return nil, nil
	}

	return trades, nil
}

func (m *Market) addParty(party string) {
	if _, ok := m.parties[party]; !ok {
		m.parties[party] = struct{}{}
	}
}

func (m *Market) applyFees(ctx context.Context, order *types.Order, trades []*types.Trade) error {
	// if we have some trades, let's try to get the fees

	if len(trades) <= 0 || m.as.IsOpeningAuction() {
		return nil
	}

	// first we get the fees for these trades
	var (
		fees events.FeesTransfer
		err  error
	)

	if !m.as.InAuction() {
		fees, err = m.fee.CalculateForContinuousMode(trades)
	} else if m.as.IsMonitorAuction() {
		// we are in auction mode
		fees, err = m.fee.CalculateForAuctionMode(trades)
	} else if m.as.IsFBA() {
		fees, err = m.fee.CalculateForFrequentBatchesAuctionMode(trades)
	}

	if err != nil {
		return err
	}

	var transfers []*types.LedgerMovement

	if !m.as.InAuction() {
		transfers, err = m.collateral.TransferFeesContinuousTrading(ctx, m.GetID(), m.settlementAsset, fees)
	} else if m.as.IsMonitorAuction() {
		// @TODO handle this properly
		transfers, err = m.collateral.TransferFees(ctx, m.GetID(), m.settlementAsset, fees)
	} else if m.as.IsFBA() {
		// @TODO implement transfer for auction types
		transfers, err = m.collateral.TransferFees(ctx, m.GetID(), m.settlementAsset, fees)
	}

	if err != nil {
		m.log.Error("unable to transfer fees for trades",
			logging.String("order-id", order.ID),
			logging.String("market-id", m.GetID()),
			logging.Error(err))
		return types.OrderErrorInsufficientFundsToPayFees
	}

	// send transfers through the broker
	if len(transfers) > 0 {
		m.broker.Send(events.NewLedgerMovements(ctx, transfers))
	}

	m.marketActivityTracker.UpdateFeesFromTransfers(m.GetID(), fees.Transfers())

	return nil
}

func (m *Market) handleConfirmationPassiveOrders(
	ctx context.Context,
	conf *types.OrderConfirmation,
) {
	if conf.PassiveOrdersAffected != nil {
		evts := make([]events.Event, 0, len(conf.PassiveOrdersAffected))

		// Insert or update passive orders siting on the book
		for _, order := range conf.PassiveOrdersAffected {
			// set the `updatedAt` value as these orders have changed
			order.UpdatedAt = m.timeService.GetTimeNow().UnixNano()
			evts = append(evts, events.NewOrderEvent(ctx, order))

			// If the order is a pegged order and is complete we must remove it from the pegged list
			if order.PeggedOrder != nil {
				if order.Remaining == 0 || order.Status != types.OrderStatusActive {
					m.removePeggedOrder(order)
				}
			}

			// remove the order from the expiring list
			// if it was a GTT order
			if order.IsExpireable() && order.IsFinished() {
				m.expiringOrders.RemoveOrder(order.ExpiresAt, order.ID)
			}
		}

		m.broker.SendBatch(evts)
	}
}

func (m *Market) handleConfirmation(ctx context.Context, conf *types.OrderConfirmation) []*types.Order {
	// When re-submitting liquidity order, it happen that the pricing is putting
	// the order at a price which makes it uncross straight away.
	// then triggering this handleConfirmation flow, etc.
	// Although the order is considered aggressive, and we never expect in the flow
	// for an aggressive order to be pegged, so we never remove them from the pegged
	// list. All this impact the float of EnterAuction, which if triggered from there
	// will try to park all pegged orders, including this order which have never been
	// removed from the pegged list. We add this check to make sure  that if the
	// aggressive order is pegged, we then do remove it from the list.
	if conf.Order.PeggedOrder != nil {
		if conf.Order.Remaining == 0 || conf.Order.Status != types.OrderStatusActive {
			m.removePeggedOrder(conf.Order)
		}
	}

	m.handleConfirmationPassiveOrders(ctx, conf)
	orderUpdates := make([]*types.Order, 0, len(conf.PassiveOrdersAffected)+1)
	orderUpdates = append(orderUpdates, conf.Order)
	orderUpdates = append(orderUpdates, conf.PassiveOrdersAffected...)

	if len(conf.Trades) == 0 {
		return orderUpdates
	}
	m.setLastTradedPrice(conf.Trades[len(conf.Trades)-1])

	// Insert all trades resulted from the executed order
	tradeEvts := make([]events.Event, 0, len(conf.Trades))
	tradedValue, _ := num.UintFromDecimal(
		conf.TradedValue().ToDecimal().Div(m.positionFactor))
	for idx, trade := range conf.Trades {
		trade.SetIDs(m.idgen.NextID(), conf.Order, conf.PassiveOrdersAffected[idx])

		tradeEvts = append(tradeEvts, events.NewTradeEvent(ctx, *trade))

		m.position.Update(ctx, trade, conf.PassiveOrdersAffected[idx], conf.Order)

		// Record open interest change
		if err := m.tsCalc.RecordOpenInterest(m.position.GetOpenInterest(), m.timeService.GetTimeNow()); err != nil {
			m.log.Debug("unable record open interest",
				logging.String("market-id", m.GetID()),
				logging.Error(err))
		}
		// add trade to settlement engine for correct MTM settlement of individual trades
		m.settlement.AddTrade(trade)
	}
	m.feeSplitter.AddTradeValue(tradedValue)
	m.marketActivityTracker.AddValueTraded(m.mkt.ID, tradedValue)
	m.broker.SendBatch(tradeEvts)

	// check reference moves if we have order updates, and we are not in an auction (or leaving an auction)
	// we handle reference moves in confirmMTM when leaving an auction already
	if len(orderUpdates) > 0 && !m.as.CanLeave() && !m.as.InAuction() {
		m.checkForReferenceMoves(
			ctx, orderUpdates, false)
	}

	return orderUpdates
}

func (m *Market) confirmMTM(ctx context.Context, skipMargin bool) {
	// now let's get the transfers for MTM settlement
	mp := m.getCurrentMarkPrice()
	evts := m.position.UpdateMarkPrice(mp)
	settle := m.settlement.SettleMTM(ctx, mp, evts)

	// let the product know about the mark-price, incase its the sort of product that cares
	if m.perp && m.markPrice != nil {
		m.tradableInstrument.Instrument.Product.SubmitDataPoint(ctx, mp, m.timeService.GetTimeNow().UnixNano())
	}

	// Only process collateral and risk once per order, not for every trade
	margins := m.collateralAndRisk(ctx, settle)
	orderUpdates := m.handleRiskEvts(ctx, margins)

	// orders updated -> check reference moves
	// force check
	m.checkForReferenceMoves(ctx, orderUpdates, false)

	if !skipMargin {
		// release excess margin for all positions
		m.recheckMargin(ctx, m.position.Positions())
	}
}

func (m *Market) handleRiskEvts(ctx context.Context, margins []events.Risk) []*types.Order {
	if len(margins) == 0 {
		return nil
	}
	transfers, closed, bondPenalties, err := m.collateral.MarginUpdate(ctx, m.GetID(), margins)
	if err != nil {
		m.log.Error("margin update had issues", logging.Error(err))
	}
	if err == nil && len(transfers) > 0 {
		evt := events.NewLedgerMovements(ctx, transfers)
		m.broker.Send(evt)
	}
	if len(bondPenalties) > 0 {
		transfers, err := m.bondSlashing(ctx, bondPenalties...)
		if err != nil {
			m.log.Error("Failed to perform bond slashing",
				logging.Error(err))
		}
		// if bond slashing occurred then amounts in "closed" will not be accurate
		if len(transfers) > 0 {
			m.broker.Send(events.NewLedgerMovements(ctx, transfers))
			closedRecalculated := make([]events.Margin, 0, len(closed))
			for _, c := range closed {
				if pos, ok := m.position.GetPositionByPartyID(c.Party()); ok {
					margin, err := m.collateral.GetPartyMargin(pos, m.settlementAsset, m.mkt.ID)
					if err != nil {
						m.log.Error("couldn't get party margin",
							logging.PartyID(c.Party()),
							logging.Error(err))
						// keep old value if we weren't able to recalculate
						closedRecalculated = append(closedRecalculated, c)
						continue
					}
					closedRecalculated = append(closedRecalculated, margin)
				}
			}
			closed = closedRecalculated
		}
	}
	if len(closed) == 0 {
		m.updateLiquidityFee(ctx)
		return nil
	}
	var orderUpdates []*types.Order
	upd, err := m.resolveClosedOutParties(
		ctx, closed)
	if err != nil {
		m.log.Error("unable to closed out parties",
			logging.String("market-id", m.GetID()),
			logging.Error(err))
	}
	if len(upd) > 0 {
		orderUpdates = append(orderUpdates, upd...)
	}

	m.updateLiquidityFee(ctx)
	return orderUpdates
}

// updateLiquidityFee computes the current LiquidityProvision fee and updates
// the fee engine.
func (m *Market) updateLiquidityFee(ctx context.Context) {
	stake := m.getTargetStake()
	fee := m.liquidityEngine.ProvisionsPerParty().FeeForTarget(stake)
	if !fee.Equals(m.getLiquidityFee()) {
		m.fee.SetLiquidityFee(fee)
		m.setLiquidityFee(fee)
		m.broker.Send(
			events.NewMarketUpdatedEvent(ctx, *m.mkt),
		)
	}
}

func (m *Market) setLiquidityFee(fee num.Decimal) {
	m.mkt.Fees.Factors.LiquidityFee = fee
}

func (m *Market) getLiquidityFee() num.Decimal {
	return m.mkt.Fees.Factors.LiquidityFee
}

// resolveClosedOutParties - the parties with the given market position who haven't got sufficient collateral
// need to be closed out -> the network buys/sells the open volume, and trades with the rest of the network
// this flow is similar to the SubmitOrder bit where trades are made, with fewer checks (e.g. no MTM settlement, no risk checks)
// pass in the order which caused parties to be distressed.
func (m *Market) resolveClosedOutParties(ctx context.Context, distressedMarginEvts []events.Margin) ([]*types.Order, error) {
	if len(distressedMarginEvts) == 0 {
		return nil, nil
	}
	timer := metrics.NewTimeCounter(m.mkt.ID, "market", "resolveClosedOutParties")
	defer timer.EngineTimeCounterAdd()

	now := m.timeService.GetTimeNow()
	// this is going to be run after the closed out routines
	// are finished, in order to notify the liquidity engine of
	// any changes in the book
	orderUpdates := []*types.Order{}

	distressedPos := make([]events.MarketPosition, 0, len(distressedMarginEvts))
	for _, v := range distressedMarginEvts {
		if m.log.GetLevel() == logging.DebugLevel {
			m.log.Debug("closing out party",
				logging.PartyID(v.Party()),
				logging.MarketID(m.GetID()))
		}
		distressedPos = append(distressedPos, v)
	}

	rmorders, err := m.matching.RemoveDistressedOrders(distressedPos)
	if err != nil {
		m.log.Panic("Failed to remove distressed parties from the orderbook",
			logging.Error(err),
		)
	}

	mktID := m.GetID()
	// push rm orders into buf
	// and remove the orders from the positions engine
	evts := []events.Event{}
	for _, o := range rmorders {
		if o.IsExpireable() {
			m.expiringOrders.RemoveOrder(o.ExpiresAt, o.ID)
		}
		if o.PeggedOrder != nil {
			m.removePeggedOrder(o)
		}
		o.UpdatedAt = now.UnixNano()
		evts = append(evts, events.NewOrderEvent(ctx, o))
		_ = m.position.UnregisterOrder(ctx, o)
	}

	// add the orders remove from the book to the orders
	// to be sent to the liquidity engine
	orderUpdates = append(orderUpdates, rmorders...)

	// now we also remove ALL parked order for the different parties
	for _, v := range distressedPos {
		orders, oevts := m.peggedOrders.RemoveAllForParty(
			ctx, v.Party(), types.OrderStatusStopped)

		for _, v := range orders {
			m.expiringOrders.RemoveOrder(v.ExpiresAt, v.ID)
		}

		// add all pegged orders too to the orderUpdates
		orderUpdates = append(orderUpdates, orders...)
		// add all events to evts list
		evts = append(evts, oevts...)
	}

	// send all orders which got stopped through the event bus
	m.broker.SendBatch(evts)

	closed := distressedMarginEvts // default behaviour (ie if rmorders is empty) is to closed out all distressed positions we started out with

	// we need to check margin requirements again, it's possible for parties to no longer be distressed now that their orders have been removed
	if len(rmorders) != 0 {
		var okPos []events.Margin // need to declare this because we want to reassign closed
		// now that we closed orders, let's run the risk engine again
		// so it'll separate the positions still in distress from the
		// which have acceptable margins
		increment := m.tradableInstrument.Instrument.Product.GetMarginIncrease(m.timeService.GetTimeNow().UnixNano())
		okPos, closed = m.risk.ExpectMargins(distressedMarginEvts, m.lastTradedPrice.Clone(), increment)

		parties := make([]string, 0, len(okPos))
		for _, v := range okPos {
			parties = append(parties, v.Party())
		}
		if m.log.IsDebug() {
			for _, pID := range parties {
				m.log.Debug("previously distressed party have now an acceptable margin",
					logging.String("market-id", mktID),
					logging.String("party-id", pID))
			}
		}
		if len(parties) > 0 {
			// emit event indicating we had to close orders, but parties were not distressed anymore after doing so.
			m.broker.Send(events.NewDistressedOrdersEvent(ctx, mktID, parties))
		}
	}

	// if no position are meant to be closed, just return now.
	if len(closed) <= 0 {
		return orderUpdates, nil
	}

	// we only need the MarketPosition events here, and rather than changing all the calls
	// we can just keep the MarketPosition bit
	closedMPs := make([]events.MarketPosition, 0, len(closed))
	closedParties := make([]string, 0, len(closed))
	// get the actual position, so we can work out what the total position of the market is going to be
	var networkPos int64
	for _, pos := range closed {
		networkPos += pos.Size()
		closedMPs = append(closedMPs, pos)
		closedParties = append(closedParties, pos.Party())
	}
	if networkPos == 0 {
		m.log.Warn("Network positions is 0 after closing out parties, nothing more to do",
			logging.String("market-id", m.GetID()))
		m.finalizePartiesCloseOut(ctx, closed, closedMPs)
		return orderUpdates, nil
	}
	// network order
	// @TODO this order is more of a placeholder than an actual final version
	// of the network order we'll be using
	size := uint64(networkPos)
	if networkPos < 0 {
		size = uint64(-networkPos)
	}

	no := types.Order{
		MarketID:    m.GetID(),
		Remaining:   size,
		Status:      types.OrderStatusActive,
		Party:       types.NetworkParty, // network is not a party as such
		Side:        types.SideSell,     // assume sell, price is zero in that case anyway
		CreatedAt:   now.UnixNano(),
		Reference:   "LS",                      // liquidity sourcing, reference the order which caused the problem
		TimeInForce: types.OrderTimeInForceFOK, // this is an all-or-nothing order, so TIME_IN_FORCE == FOK
		Type:        types.OrderTypeNetwork,
		Price:       num.UintZero(),
	}
	no.Size = no.Remaining

	no.ID = m.idgen.NextID()
	// we need to buy, specify side + max price
	if networkPos < 0 {
		no.Side = types.SideBuy
	}
	// Send the aggressive order into matching engine
	confirmation, err := m.matching.SubmitOrder(&no)
	if err != nil {
		// we can safely panic here, only possibility of failure
		// with the orderbook is in case of order validation, it should
		// not be possible for us to submit an invalid order at this
		// point, and an invalid order would be a code error then.
		m.log.Panic("Failure after submitting order to matching engine",
			logging.Order(no),
			logging.Error(err))
	}
	// Whether we were able to uncross the network order or not, we have to update the positions
	// any closedParties element that wasn't previously marked as distressed should be marked as now
	// being distressed. Any previously distressed positions that are no longer distressed
	// should also be updated.
	// If the network order uncrosses, we can ignore the distressed parties (they are closed out)
	// but the safe parties should still be sent out.
	dp, sp := m.position.MarkDistressed(closedParties)
	// FIXME(j): this is a temporary measure for the case where we do not have enough orders
	// in the book to 0 out the positions.
	// in this case we will just return now, cutting off the position resolution
	// this means that party still being distressed will stay distressed,
	// then when a new order is placed, the distressed parties will go again through positions resolution
	// and if the volume of the book is acceptable, we will then process positions resolutions
	if no.Remaining == no.Size {
		// it is possible that we pass through here with the same distressed parties as before, no need
		// to send the event if both distressed and safe parties slices are nil
		if len(dp) != 0 || len(sp) != 0 {
			devt := events.NewDistressedPositionsEvent(ctx, m.GetID(), dp, sp)
			m.broker.Send(devt)
		}
		return orderUpdates, common.ErrNotEnoughVolumeToZeroOutNetworkOrder
	}
	// if we have any distressed positions that now no longer are distressed, emit the event
	// no point in sending an event unless there's data
	if len(sp) > 0 {
		devt := events.NewDistressedPositionsEvent(ctx, m.GetID(), nil, sp)
		m.broker.Send(devt)
	}

	// @NOTE: At this point, the network order was updated by the orderbook
	// the price field now contains the average trade price at which the order was fulfilled
	m.broker.Send(events.NewOrderEvent(ctx, &no))

	m.handleConfirmationPassiveOrders(ctx, confirmation)

	// also add the passive orders from the book into the list
	// of updated orders to send to liquidity engine
	orderUpdates = append(orderUpdates, confirmation.PassiveOrdersAffected...)

	// pay the fees now
	fees, distressedPartiesFees := m.fee.CalculateFeeForPositionResolution(
		confirmation.Trades, closedMPs)

	tresps, err := m.collateral.TransferFees(ctx, m.GetID(), m.settlementAsset, fees)
	if err != nil {
		// FIXME(): we may figure a better error handling in here
		m.log.Error("unable to transfer fees for positions resolutions",
			logging.Error(err),
			logging.String("market-id", m.GetID()))
		return orderUpdates, err
	}
	// send transfer to buffer
	if len(tresps) > 0 {
		m.broker.Send(events.NewLedgerMovements(ctx, tresps))
	}

	if len(confirmation.Trades) > 0 {
		// Insert all trades resulted from the executed order
		tradeEvts := make([]events.Event, 0, len(confirmation.Trades))
		// get total traded volume
		tradedValue, _ := num.UintFromDecimal(
			confirmation.TradedValue().ToDecimal().Div(m.positionFactor))
		for idx, trade := range confirmation.Trades {
			trade.SetIDs(m.idgen.NextID(), &no, confirmation.PassiveOrdersAffected[idx])

			// setup the type of the trade to network
			// this trade did happen with a GOOD trader to
			// 0 out the BAD trader position
			trade.Type = types.TradeTypeNetworkCloseOutGood
			tradeEvts = append(tradeEvts, events.NewTradeEvent(ctx, *trade))

			// Update positions - this is a special trade involving the network as party
			// so rather than checking this every time we call Update, call special UpdateNetwork
			m.position.UpdateNetwork(ctx, trade, confirmation.PassiveOrdersAffected[idx])
			if err := m.tsCalc.RecordOpenInterest(m.position.GetOpenInterest(), now); err != nil {
				m.log.Debug("unable record open interest",
					logging.String("market-id", m.GetID()),
					logging.Error(err))
			}

			m.settlement.AddTrade(trade)
		}
		m.feeSplitter.AddTradeValue(tradedValue)
		m.marketActivityTracker.AddValueTraded(m.mkt.ID, tradedValue)
		m.broker.SendBatch(tradeEvts)
	}

	m.zeroOutNetwork(ctx, closedMPs, &no, distressedPartiesFees)

	// swipe all accounts and stuff
	m.finalizePartiesCloseOut(ctx, closed, closedMPs)

	m.confirmMTM(ctx, false)

	return orderUpdates, nil
}

func (m *Market) finalizePartiesCloseOut(
	ctx context.Context,
	closed []events.Margin,
	closedMPs []events.MarketPosition,
) {
	// remove accounts, positions and return
	// from settlement engine first
	m.settlement.RemoveDistressed(ctx, closed)
	// then from positions
	closedMPs = m.position.RemoveDistressed(closedMPs)
	// finally remove from collateral (moving funds where needed)
	movements, err := m.collateral.RemoveDistressed(
		ctx, closedMPs, m.GetID(), m.settlementAsset)
	if err != nil {
		m.log.Panic(
			"Failed to remove distressed accounts cleanly",
			logging.Error(err))
	}

	if len(movements.Entries) > 0 {
		m.broker.Send(
			events.NewLedgerMovements(
				ctx, []*types.LedgerMovement{movements}),
		)
	}
}

func (m *Market) zeroOutNetwork(ctx context.Context, parties []events.MarketPosition, settleOrder *types.Order, fees map[string]*types.Fee) {
	timer := metrics.NewTimeCounter(m.mkt.ID, "market", "zeroOutNetwork")
	defer timer.EngineTimeCounterAdd()

	// ensure an original price is set
	if settleOrder.OriginalPrice == nil {
		settleOrder.OriginalPrice = num.UintZero().Div(settleOrder.Price, m.priceFactor)
	}
	marketID := m.GetID()
	now := m.timeService.GetTimeNow().UnixNano()
	order := types.Order{
		MarketID:      marketID,
		Status:        types.OrderStatusFilled,
		Party:         types.NetworkParty,
		Price:         settleOrder.Price.Clone(),
		OriginalPrice: settleOrder.OriginalPrice.Clone(),
		CreatedAt:     now,
		Reference:     "close-out distressed",
		TimeInForce:   types.OrderTimeInForceFOK, // this is an all-or-nothing order, so TIME_IN_FORCE == FOK
		Type:          types.OrderTypeNetwork,
	}

	marginLevels := types.MarginLevels{
		MarketID:  m.mkt.GetID(),
		Asset:     m.settlementAsset,
		Timestamp: now,
	}

	tradeEvts := make([]events.Event, 0, len(parties))
	for _, party := range parties {
		tSide, nSide := types.SideSell, types.SideSell // one of them will have to sell
		if party.Size() < 0 {
			tSide = types.SideBuy
		} else {
			nSide = types.SideBuy
		}
		tSize := party.Size()
		order.Size = uint64(tSize)
		if tSize < 0 {
			order.Size = uint64(-tSize)
		}

		// set order fields (network order)
		order.Remaining = 0
		order.Side = nSide
		order.Status = types.OrderStatusFilled // An order with no remaining must be filled

		order.ID = m.idgen.NextID()

		// this is the party order
		partyOrder := types.Order{
			MarketID:      marketID,
			Size:          order.Size,
			Remaining:     0,
			Status:        types.OrderStatusFilled,
			Party:         party.Party(),
			Side:          tSide,                     // assume sell, price is zero in that case anyway
			Price:         settleOrder.Price.Clone(), // average price
			OriginalPrice: settleOrder.OriginalPrice.Clone(),
			CreatedAt:     now,
			Reference:     fmt.Sprintf("distressed-%s", party.Party()),
			TimeInForce:   types.OrderTimeInForceFOK, // this is an all-or-nothing order, so TIME_IN_FORCE == FOK
			Type:          types.OrderTypeNetwork,
		}

		partyOrder.ID = m.idgen.NextID()

		// store the party order, too
		m.broker.Send(events.NewOrderEvent(ctx, &partyOrder))
		m.broker.Send(events.NewOrderEvent(ctx, &order))

		// now let's create the trade between the party and network
		var (
			buyOrder, sellOrder     *types.Order
			buySideFee, sellSideFee *types.Fee
		)
		if order.Side == types.SideBuy {
			buyOrder = &order
			sellOrder = &partyOrder
			sellSideFee = fees[party.Party()]
		} else {
			sellOrder = &order
			buyOrder = &partyOrder
			buySideFee = fees[party.Party()]
		}

		trade := types.Trade{
			ID:          m.idgen.NextID(),
			MarketID:    partyOrder.MarketID,
			Price:       partyOrder.Price.Clone(),
			MarketPrice: partyOrder.OriginalPrice.Clone(),
			Size:        partyOrder.Size,
			Aggressor:   order.Side, // we consider network to be aggressor
			BuyOrder:    buyOrder.ID,
			SellOrder:   sellOrder.ID,
			Buyer:       buyOrder.Party,
			Seller:      sellOrder.Party,
			Timestamp:   partyOrder.CreatedAt,
			Type:        types.TradeTypeNetworkCloseOutBad,
			SellerFee:   sellSideFee,
			BuyerFee:    buySideFee,
		}
		tradeEvts = append(tradeEvts, events.NewTradeEvent(ctx, trade))

		// 0 out margins levels for this trader
		marginLevels.Party = party.Party()
		m.broker.Send(events.NewMarginLevelsEvent(ctx, marginLevels))

		if m.log.GetLevel() == logging.DebugLevel {
			m.log.Debug("party closed-out with success",
				logging.String("party-id", party.Party()),
				logging.String("market-id", m.GetID()))
		}
	}
	if len(tradeEvts) > 0 {
		m.broker.SendBatch(tradeEvts)
	}
}

func (m *Market) recheckMargin(ctx context.Context, pos []events.MarketPosition) {
	risk := m.updateMargin(ctx, pos)
	if len(risk) == 0 {
		return
	}
	// now transfer margins, ignore closed because we're only recalculating for non-distressed parties.
	m.transferRecheckMargins(ctx, risk)
}

func (m *Market) checkMarginForOrder(ctx context.Context, pos *positions.MarketPosition, order *types.Order) error {
	risk, closed, err := m.calcMargins(ctx, pos, order)
	// margin error
	if err != nil {
		return err
	}

	// margins calculated, set about tranferring funds. At this point, if closed is not empty, those parties are distressed
	// the risk slice are risk events, that we must use to transfer funds
	return m.transferMargins(ctx, risk, closed)
}

func (m *Market) checkMarginForAmendOrder(ctx context.Context, existingOrder *types.Order, amendedOrder *types.Order) error {
	origPos, ok := m.position.GetPositionByPartyID(existingOrder.Party)
	if !ok {
		m.log.Panic("could not get position for party", logging.PartyID(existingOrder.Party))
	}

	pos := origPos.Clone()

	// if order was park we have nothing to do here
	if existingOrder.Status != types.OrderStatusParked {
		pos.UnregisterOrder(m.log, existingOrder)
	}

	pos.RegisterOrder(m.log, amendedOrder)

	// we are just checking here if we can pass the margin calls.
	_, _, err := m.calcMargins(ctx, pos, amendedOrder)
	return err
}

func (m *Market) setLastTradedPrice(trade *types.Trade) {
	m.lastTradedPrice = trade.Price.Clone()
}

// this function handles moving money after settle MTM + risk margin updates
// but does not move the money between party accounts (ie not to/from margin accounts after risk).
func (m *Market) collateralAndRisk(ctx context.Context, settle []events.Transfer) []events.Risk {
	timer := metrics.NewTimeCounter(m.mkt.ID, "market", "collateralAndRisk")
	defer timer.EngineTimeCounterAdd()
	evts, response, err := m.collateral.MarkToMarket(ctx, m.GetID(), settle, m.settlementAsset)
	if err != nil {
		m.log.Error(
			"Failed to process mark to market settlement (collateral)",
			logging.Error(err),
		)
		return nil
	}
	// sending response to buffer
	if len(response) > 0 {
		m.broker.Send(events.NewLedgerMovements(ctx, response))
	}

	// let risk engine do its thing here - it returns a slice of money that needs
	// to be moved to and from margin accounts
	increment := m.tradableInstrument.Instrument.Product.GetMarginIncrease(m.timeService.GetTimeNow().UnixNano())
	riskUpdates := m.risk.UpdateMarginsOnSettlement(ctx, evts, m.getCurrentMarkPrice(), increment)
	if len(riskUpdates) == 0 {
		return nil
	}

	return riskUpdates
}

func (m *Market) CancelAllStopOrders(ctx context.Context, partyID string) error {
	if !m.canTrade() {
		return common.ErrTradingNotAllowed
	}

	stopOrders, err := m.stopOrders.Cancel(partyID, "")
	if err != nil {
		return err
	}

	m.removeCancelledExpiringStopOrders(stopOrders)

	evts := make([]events.Event, 0, len(stopOrders))
	for _, v := range stopOrders {
		evts = append(evts, events.NewStopOrderEvent(ctx, v))
	}

	m.broker.SendBatch(evts)

	return nil
}

func (m *Market) CancelAllOrders(ctx context.Context, partyID string) ([]*types.OrderCancellationConfirmation, error) {
	defer m.onTxProcessed()

	if !m.canTrade() {
		return nil, common.ErrTradingNotAllowed
	}

	// get all order for this party in the book
	orders := m.matching.GetOrdersPerParty(partyID)

	// add all orders being eventually parked
	orders = append(orders, m.peggedOrders.GetAllParkedForParty(partyID)...)

	// just an early exit, there's just no orders...
	if len(orders) <= 0 {
		return nil, nil
	}

	// now we eventually dedup them
	uniq := map[string]*types.Order{}
	for _, v := range orders {
		uniq[v.ID] = v
	}

	// put them back in the slice, and sort them
	orders = make([]*types.Order, 0, len(uniq))
	for _, v := range uniq {
		orders = append(orders, v)
	}
	sort.Slice(orders, func(i, j int) bool {
		return orders[i].ID < orders[j].ID
	})

	cancellations := make([]*types.OrderCancellationConfirmation, 0, len(orders))

	// now iterate over all orders and cancel one by one.
	cancelledOrders := make([]*types.Order, 0, len(orders))
	for _, order := range orders {
		cancellation, err := m.cancelOrder(ctx, partyID, order.ID)
		if err != nil {
			return nil, err
		}
		cancellations = append(cancellations, cancellation)
		cancelledOrders = append(cancelledOrders, cancellation.Order)
	}

	m.checkForReferenceMoves(ctx, cancelledOrders, false)

	return cancellations, nil
}

func (m *Market) CancelOrder(
	ctx context.Context,
	partyID, orderID string, deterministicID string,
) (oc *types.OrderCancellationConfirmation, _ error) {
	idgen := idgeneration.New(deterministicID)
	return m.CancelOrderWithIDGenerator(ctx, partyID, orderID, idgen)
}

func (m *Market) CancelOrderWithIDGenerator(
	ctx context.Context,
	partyID, orderID string,
	idgen common.IDGenerator,
) (oc *types.OrderCancellationConfirmation, _ error) {
	defer m.onTxProcessed()

	m.idgen = idgen
	defer func() { m.idgen = nil }()

	if !m.canTrade() {
		return nil, common.ErrTradingNotAllowed
	}

	conf, err := m.cancelOrder(ctx, partyID, orderID)
	if err != nil {
		return conf, err
	}

	if !m.as.InAuction() {
		m.checkForReferenceMoves(ctx, []*types.Order{conf.Order}, false)
	}

	return conf, nil
}

func (m *Market) CancelStopOrder(
	ctx context.Context,
	partyID, orderID string,
) error {
	if !m.canTrade() {
		return common.ErrTradingNotAllowed
	}

	stopOrders, err := m.stopOrders.Cancel(partyID, orderID)
	if err != nil || len(stopOrders) <= 0 { // could return just an empty slice
		return err
	}

	m.removeCancelledExpiringStopOrders(stopOrders)

	evts := []events.Event{events.NewStopOrderEvent(ctx, stopOrders[0])}
	if len(stopOrders) > 1 {
		evts = append(evts, events.NewStopOrderEvent(ctx, stopOrders[1]))
	}

	m.broker.SendBatch(evts)

	return nil
}

func (m *Market) removeCancelledExpiringStopOrders(
	stopOrders []*types.StopOrder,
) {
	for _, o := range stopOrders {
		if o.Expiry.Expires() {
			m.expiringStopOrders.RemoveOrder(o.Expiry.ExpiresAt.UnixNano(), o.ID)
		}
	}
}

// CancelOrder cancels the given order.
func (m *Market) cancelOrder(ctx context.Context, partyID, orderID string) (*types.OrderCancellationConfirmation, error) {
	timer := metrics.NewTimeCounter(m.mkt.ID, "market", "CancelOrder")
	defer timer.EngineTimeCounterAdd()

	if m.closed {
		return nil, common.ErrMarketClosed
	}

	order, foundOnBook, err := m.getOrderByID(orderID)
	if err != nil {
		return nil, err
	}

	// Only allow the original order creator to cancel their order
	if order.Party != partyID {
		if m.log.GetLevel() == logging.DebugLevel {
			m.log.Debug("Party ID mismatch",
				logging.String("party-id", partyID),
				logging.String("order-id", orderID),
				logging.String("market", m.mkt.ID))
		}
		return nil, types.ErrInvalidPartyID
	}

	defer m.releaseMarginExcess(ctx, partyID)

	if foundOnBook {
		cancellation, err := m.matching.CancelOrder(order)
		if cancellation == nil || err != nil {
			if m.log.GetLevel() == logging.DebugLevel {
				m.log.Debug("Failure after cancel order from matching engine",
					logging.String("party-id", partyID),
					logging.String("order-id", orderID),
					logging.String("market", m.mkt.ID),
					logging.Error(err))
			}
			return nil, err
		}
		_ = m.position.UnregisterOrder(ctx, order)
	}

	if order.IsExpireable() {
		m.expiringOrders.RemoveOrder(order.ExpiresAt, order.ID)
	}

	// If this is a pegged order, remove from pegged and parked lists
	if order.PeggedOrder != nil {
		m.removePeggedOrder(order)
		order.Status = types.OrderStatusCancelled
	}

	// Publish the changed order details
	order.UpdatedAt = m.timeService.GetTimeNow().UnixNano()
	m.broker.Send(events.NewOrderEvent(ctx, order))

	return &types.OrderCancellationConfirmation{Order: order}, nil
}

// parkOrder removes the given order from the orderbook
// parkOrder will panic if it encounters errors, which means that it reached an
// invalid state.
func (m *Market) parkOrder(ctx context.Context, orderID string) *types.Order {
	order, err := m.matching.RemoveOrder(orderID)
	if err != nil {
		m.log.Panic("Failure to remove order from matching engine",
			logging.OrderID(orderID),
			logging.Error(err))
	}

	_ = m.position.UnregisterOrder(ctx, order)
	m.peggedOrders.Park(order)
	m.broker.Send(events.NewOrderEvent(ctx, order))
	m.releaseMarginExcess(ctx, order.Party)
	return order
}

// AmendOrder amend an existing order from the order book.
func (m *Market) AmendOrder(
	ctx context.Context,
	orderAmendment *types.OrderAmendment,
	party string,
	deterministicID string,
) (oc *types.OrderConfirmation, _ error,
) {
	idgen := idgeneration.New(deterministicID)
	return m.AmendOrderWithIDGenerator(ctx, orderAmendment, party, idgen)
}

func (m *Market) AmendOrderWithIDGenerator(
	ctx context.Context,
	orderAmendment *types.OrderAmendment,
	party string,
	idgen common.IDGenerator,
) (oc *types.OrderConfirmation, _ error,
) {
	defer m.onTxProcessed()

	m.idgen = idgen
	defer func() { m.idgen = nil }()

	defer func() {
		m.triggerStopOrders(ctx, idgen)
	}()

	if !m.canTrade() {
		return nil, common.ErrTradingNotAllowed
	}

	conf, updatedOrders, err := m.amendOrder(ctx, orderAmendment, party)
	if err != nil {
		return nil, err
	}

	allUpdatedOrders := append(
		[]*types.Order{conf.Order},
		conf.PassiveOrdersAffected...,
	)
	allUpdatedOrders = append(
		allUpdatedOrders,
		updatedOrders...,
	)

	if !m.as.InAuction() {
		m.checkForReferenceMoves(ctx, allUpdatedOrders, false)
	}

	return conf, nil
}

func (m *Market) findOrderAndEnsureOwnership(
	orderID, partyID, marketID string,
) (exitingOrder *types.Order, foundOnBook bool, err error) {
	// Try and locate the existing order specified on the
	// order book in the matching engine for this market
	existingOrder, foundOnBook, err := m.getOrderByID(orderID)
	if err != nil {
		if m.log.GetLevel() == logging.DebugLevel {
			m.log.Debug("Invalid order ID",
				logging.OrderID(orderID),
				logging.PartyID(partyID),
				logging.MarketID(marketID),
				logging.Error(err))
		}
		return nil, false, types.ErrInvalidOrderID
	}

	// We can only amend this order if we created it
	if existingOrder.Party != partyID {
		if m.log.GetLevel() == logging.DebugLevel {
			m.log.Debug("Invalid party ID",
				logging.String("original party id:", existingOrder.Party),
				logging.PartyID(partyID),
			)
		}
		return nil, false, types.ErrInvalidPartyID
	}

	// Validate Market
	if existingOrder.MarketID != marketID {
		// we should never reach this point
		m.log.Panic("Market ID mismatch",
			logging.MarketID(m.mkt.ID),
			logging.Order(*existingOrder),
			logging.Error(types.ErrInvalidMarketID),
		)
	}

	return existingOrder, foundOnBook, err
}

func (m *Market) amendOrder(
	ctx context.Context,
	orderAmendment *types.OrderAmendment,
	party string,
) (cnf *types.OrderConfirmation, orderUpdates []*types.Order, returnedErr error) {
	timer := metrics.NewTimeCounter(m.mkt.ID, "market", "AmendOrder")
	defer timer.EngineTimeCounterAdd()

	// Verify that the market is not closed
	if m.closed {
		return nil, nil, common.ErrMarketClosed
	}

	existingOrder, foundOnBook, err := m.findOrderAndEnsureOwnership(
		orderAmendment.OrderID, party, m.GetID())
	if err != nil {
		return nil, nil, err
	}

	if err := m.validateOrderAmendment(existingOrder, orderAmendment); err != nil {
		return nil, nil, err
	}

	amendedOrder, err := existingOrder.ApplyOrderAmendment(orderAmendment, m.timeService.GetTimeNow().UnixNano(), m.priceFactor)
	if err != nil {
		return nil, nil, err
	}

	// We do this first, just in case the party would also have
	// change the expiry, and that would have been catched by
	// the follow up checks, so we do not insert a non-existing
	// order in the expiring orders
	// if remaining is reduces <= 0, then order is cancelled
	if amendedOrder.Remaining <= 0 {
		confirm, err := m.cancelOrder(
			ctx, existingOrder.Party, existingOrder.ID)
		if err != nil {
			return nil, nil, err
		}
		return &types.OrderConfirmation{
			Order: confirm.Order,
		}, nil, nil
	}

	// If we have a pegged order that is no longer expiring, we need to remove it
	var (
		needToRemoveExpiry, needToAddExpiry bool
		expiresAt                           int64
	)

	defer func() {
		// no errors, amend most likely happened properly
		if returnedErr == nil {
			if needToRemoveExpiry {
				m.expiringOrders.RemoveOrder(expiresAt, existingOrder.ID)
			}
			// need to make sure the order haven't been matched with the
			// amend, consuming the remain volume as well or we would
			// add an order while it's not needed to the expiring list
			if needToAddExpiry && cnf != nil && !cnf.Order.IsFinished() {
				m.expiringOrders.Insert(amendedOrder.ID, amendedOrder.ExpiresAt)
			}
		}
	}()

	// if we are amending from GTT to GTC, flag ready to remove from expiry list
	if existingOrder.IsExpireable() && !amendedOrder.IsExpireable() {
		// We no longer need to handle the expiry
		needToRemoveExpiry = true
		expiresAt = existingOrder.ExpiresAt
	}

	// if we are amending from GTC to GTT, flag ready to add to expiry list
	if !existingOrder.IsExpireable() && amendedOrder.IsExpireable() {
		// We need to handle the expiry
		needToAddExpiry = true
	}

	// if both where expireable but we changed the duration
	// then we need to remove, then reinsert...
	if existingOrder.IsExpireable() && amendedOrder.IsExpireable() &&
		existingOrder.ExpiresAt != amendedOrder.ExpiresAt {
		// Still expiring but needs to be updated in the expiring
		// orders pool
		needToRemoveExpiry = true
		needToAddExpiry = true
		expiresAt = existingOrder.ExpiresAt
	}

	// if expiration has changed and is before the original creation time, reject this amend
	if amendedOrder.ExpiresAt != 0 && amendedOrder.ExpiresAt < existingOrder.CreatedAt {
		if m.log.GetLevel() == logging.DebugLevel {
			m.log.Debug("Amended expiry before original creation time",
				logging.Int64("existing-created-at", existingOrder.CreatedAt),
				logging.Int64("amended-expires-at", amendedOrder.ExpiresAt),
				logging.Order(*existingOrder))
		}
		return nil, nil, types.ErrInvalidExpirationDatetime
	}

	// if expiration has changed and is not 0, and is before currentTime
	// then we expire the order
	if amendedOrder.ExpiresAt != 0 && amendedOrder.ExpiresAt < amendedOrder.UpdatedAt {
		needToAddExpiry = false
		// remove the order from the expiring
		// at this point the order is still referenced at the time of expiry of the existingOrder
		if existingOrder.IsExpireable() {
			m.expiringOrders.RemoveOrder(existingOrder.ExpiresAt, amendedOrder.ID)
		}

		// Update the existing message in place before we cancel it
		if foundOnBook {
			// Do not amend in place, the amend could be something
			// not supported for an amend in place, and not pass
			// the validation of the order book
			cancellation, err := m.matching.CancelOrder(existingOrder)
			if cancellation == nil || err != nil {
				m.log.Panic("Failure to cancel order from matching engine",
					logging.String("party-id", amendedOrder.Party),
					logging.String("order-id", amendedOrder.ID),
					logging.String("market", m.mkt.ID),
					logging.Error(err))
			}

			// unregister the existing order
			_ = m.position.UnregisterOrder(ctx, existingOrder)
		}

		// Update the order in our stores (will be marked as cancelled)
		// set the proper status
		amendedOrder.Status = types.OrderStatusExpired
		m.broker.Send(events.NewOrderEvent(ctx, amendedOrder))
		m.removePeggedOrder(amendedOrder)

		return &types.OrderConfirmation{
			Order: amendedOrder,
		}, nil, nil
	}

	if existingOrder.PeggedOrder != nil {
		// Amend in place during an auction
		if m.as.InAuction() {
			ret := m.orderAmendWhenParked(amendedOrder)
			m.broker.Send(events.NewOrderEvent(ctx, amendedOrder))
			return ret, nil, nil
		}
		err := m.repricePeggedOrder(amendedOrder)
		if err != nil {
			// Failed to reprice so we have to park the order
			if amendedOrder.Status != types.OrderStatusParked {
				// If we are live then park
				m.parkOrder(ctx, existingOrder.ID)
			}
			ret := m.orderAmendWhenParked(amendedOrder)
			m.broker.Send(events.NewOrderEvent(ctx, amendedOrder))
			return ret, nil, nil
		}
		// We got a new valid price, if we are parked we need to unpark
		if amendedOrder.Status == types.OrderStatusParked {
			// is we cann pass the margin calls, then do nothing
			if err := m.checkMarginForAmendOrder(ctx, existingOrder, amendedOrder); err != nil {
				return nil, nil, err
			}
			// we were parked, need to unpark
			m.peggedOrders.Unpark(amendedOrder.ID)
			return m.submitValidatedOrder(ctx, amendedOrder)
		}
	}

	priceShift := amendedOrder.Price.NEQ(existingOrder.Price)
	sizeIncrease := amendedOrder.Size > existingOrder.Size
	sizeDecrease := amendedOrder.Size < existingOrder.Size
	expiryChange := amendedOrder.ExpiresAt != existingOrder.ExpiresAt
	timeInForceChange := amendedOrder.TimeInForce != existingOrder.TimeInForce

	// If nothing changed, amend in place to update updatedAt and version number
	if !priceShift && !sizeIncrease && !sizeDecrease && !expiryChange && !timeInForceChange {
		ret := m.orderAmendInPlace(existingOrder, amendedOrder)
		m.broker.Send(events.NewOrderEvent(ctx, amendedOrder))
		return ret, nil, nil
	}

	// Update potential new position after the amend
	pos := m.position.AmendOrder(ctx, existingOrder, amendedOrder)

	// Perform check and allocate margin if price or order size is increased
	// ignore rollback return here, as if we amend it means the order
	// is already on the book, not rollback will be needed, the margin
	// will be updated later on for sure.

	// always update margin, even for price/size decrease
	if err = m.checkMarginForOrder(ctx, pos, amendedOrder); err != nil {
		// Undo the position registering
		_ = m.position.AmendOrder(ctx, amendedOrder, existingOrder)

		if m.log.GetLevel() == logging.DebugLevel {
			m.log.Debug("Unable to check/add margin for party",
				logging.String("market-id", m.GetID()),
				logging.Error(err))
		}
		return nil, nil, common.ErrMarginCheckFailed
	}

	icebergSizeIncrease := false
	if amendedOrder.IcebergOrder != nil && sizeIncrease {
		// iceberg orders size changes can always be done in-place because they either:
		// 1) decrease the size, which is already done in-place for all orders
		// 2) increase the size, which only increases the reserved remaining and not the "active" remaining of the iceberg
		// so we set an icebergSizeIncrease to skip the cancel-replace flow.
		sizeIncrease = false
		icebergSizeIncrease = true
	}

	// if increase in size or change in price
	// ---> DO atomic cancel and submit
	if priceShift || sizeIncrease {
		return m.orderCancelReplace(ctx, existingOrder, amendedOrder)
	}

	// if decrease in size or change in expiration date
	// ---> DO amend in place in matching engine
	if expiryChange || sizeDecrease || timeInForceChange || icebergSizeIncrease {
		ret := m.orderAmendInPlace(existingOrder, amendedOrder)
		if sizeDecrease {
			// ensure we release excess if party reduced the size of their order
			m.recheckMargin(ctx, m.position.GetPositionsByParty(amendedOrder.Party))
		}
		m.broker.Send(events.NewOrderEvent(ctx, amendedOrder))
		return ret, nil, nil
	}

	// we should never reach this point as amendment was validated before
	// and every kind should have been handled down here.
	m.log.Panic(
		"invalid amend did not match any amendment combination",
		logging.String("amended-order", amendedOrder.String()),
		logging.String("existing-order", amendedOrder.String()),
	)

	return nil, nil, types.ErrEditNotAllowed
}

func (m *Market) validateOrderAmendment(
	order *types.Order,
	amendment *types.OrderAmendment,
) error {
	if err := amendment.Validate(); err != nil {
		return err
	}
	// check TIME_IN_FORCE and expiry
	if amendment.TimeInForce == types.OrderTimeInForceGTT {
		// if expiresAt is before or equal to created at
		// we return an error, we know ExpiresAt is set because of amendment.Validate
		if *amendment.ExpiresAt <= order.CreatedAt {
			return types.OrderErrorExpiryAtBeforeCreatedAt
		}
	}

	if (amendment.TimeInForce == types.OrderTimeInForceGFN ||
		amendment.TimeInForce == types.OrderTimeInForceGFA) &&
		amendment.TimeInForce != order.TimeInForce {
		// We cannot amend to a GFA/GFN orders
		return types.OrderErrorCannotAmendToGFAOrGFN
	}

	if (order.TimeInForce == types.OrderTimeInForceGFN ||
		order.TimeInForce == types.OrderTimeInForceGFA) &&
		(amendment.TimeInForce != order.TimeInForce &&
			amendment.TimeInForce != types.OrderTimeInForceUnspecified) {
		// We cannot amend from a GFA/GFN orders
		return types.OrderErrorCannotAmendFromGFAOrGFN
	}

	if order.PeggedOrder == nil {
		// We cannot change a pegged orders details on a non pegged order
		if amendment.PeggedOffset != nil ||
			amendment.PeggedReference != types.PeggedReferenceUnspecified {
			return types.OrderErrorCannotAmendPeggedOrderDetailsOnNonPeggedOrder
		}
	} else if amendment.Price != nil {
		// We cannot change the price on a pegged order
		return types.OrderErrorUnableToAmendPriceOnPeggedOrder
	}
	return nil
}

func (m *Market) orderCancelReplace(
	ctx context.Context,
	existingOrder, newOrder *types.Order,
) (conf *types.OrderConfirmation, orders []*types.Order, err error) {
	defer func() {
		if err != nil {
			// if an error happen, the order never hit the book, so we can
			// just rollback the position size
			_ = m.position.AmendOrder(ctx, newOrder, existingOrder)
			return
		}

		orders = m.handleConfirmation(ctx, conf)
		m.broker.Send(events.NewOrderEvent(ctx, conf.Order))
	}()

	timer := metrics.NewTimeCounter(m.mkt.ID, "market", "orderCancelReplace")
	defer timer.EngineTimeCounterAdd()

	// make sure the order is on the book, this was done by canceling the order initially, but that could
	// trigger an auction in some cases.
	if o, err := m.matching.GetOrderByID(existingOrder.ID); err != nil || o == nil {
		m.log.Panic("Can't CancelReplace, the original order was not found",
			logging.OrderWithTag(*existingOrder, "existing-order"),
			logging.Error(err))
	}
	// cancel-replace amend during auction is quite simple at this point
	if m.as.InAuction() {
		conf, err := m.matching.ReplaceOrder(existingOrder, newOrder)
		if err != nil {
			m.log.Panic("unable to submit order", logging.Error(err))
		}
		if newOrder.PeggedOrder != nil {
			m.log.Panic("should never reach this point")
		}

		return conf, nil, nil
	}

	// if its an iceberg order with a price change and it is being submitted aggressively
	// set the visible remaining to the full size
	if newOrder.IcebergOrder != nil {
		newOrder.Remaining += newOrder.IcebergOrder.ReservedRemaining
		newOrder.IcebergOrder.ReservedRemaining = 0
	}

	// first we call the order book to evaluate auction triggers and get the list of trades
	trades, err := m.checkPriceAndGetTrades(ctx, newOrder)
	if err != nil {
		return nil, nil, errors.New("couldn't insert order in book")
	}

	// try to apply fees on the trade
	if err := m.applyFees(ctx, newOrder, trades); err != nil {
		return nil, nil, errors.New("could not apply fees for order")
	}

	// "hot-swap" of the orders
	conf, err = m.matching.ReplaceOrder(existingOrder, newOrder)
	if err != nil {
		m.log.Panic("unable to submit order", logging.Error(err))
	}

	// replace the trades in the confirmation to have
	// the ones with the fees embedded
	conf.Trades = trades

	// if the order is not staying in the book, then we remove it
	// from the potential positions
	if conf.Order.IsFinished() && conf.Order.Remaining > 0 {
		_ = m.position.UnregisterOrder(ctx, conf.Order)
	}

	return conf, orders, nil
}

func (m *Market) orderAmendInPlace(
	originalOrder, amendOrder *types.Order,
) *types.OrderConfirmation {
	timer := metrics.NewTimeCounter(m.mkt.ID, "market", "orderAmendInPlace")
	defer timer.EngineTimeCounterAdd()

	err := m.matching.AmendOrder(originalOrder, amendOrder)
	if err != nil {
		// panic here, no good reason for a failure at this point
		m.log.Panic("Failure after amend order from matching engine (amend-in-place)",
			logging.OrderWithTag(*amendOrder, "new-order"),
			logging.OrderWithTag(*originalOrder, "old-order"),
			logging.Error(err))
	}

	return &types.OrderConfirmation{
		Order: amendOrder,
	}
}

func (m *Market) orderAmendWhenParked(amendOrder *types.Order) *types.OrderConfirmation {
	amendOrder.Status = types.OrderStatusParked
	amendOrder.Price = num.UintZero()
	amendOrder.OriginalPrice = num.UintZero()
	m.peggedOrders.AmendParked(amendOrder)

	return &types.OrderConfirmation{
		Order: amendOrder,
	}
}

// submitStopOrders gets a status as parameter.
// this function is used on trigger but also on submission
// at expiry, so just filters out with a parameter.
func (m *Market) submitStopOrders(
	ctx context.Context,
	stopOrders []*types.StopOrder,
	status types.StopOrderStatus,
	idgen common.IDGenerator,
) []*types.OrderConfirmation {
	confirmations := []*types.OrderConfirmation{}
	evts := make([]events.Event, 0, len(stopOrders))

	// might contains both the triggered orders and the expired OCO
	for _, v := range stopOrders {
		if v.Status == status {
			conf, err := m.SubmitOrderWithIDGeneratorAndOrderID(
				ctx, v.OrderSubmission, v.Party, idgen, idgen.NextID(), false,
			)
			if err != nil {
				// not much we can do at that point, let's log the error and move on?
				m.log.Error("could not submit stop order",
					logging.StopOrderSubmission(v),
					logging.Error(err))
			}
			if err == nil && conf != nil {
				v.OrderID = conf.Order.ID
				confirmations = append(confirmations, conf)
			}
		}

		evts = append(evts, events.NewStopOrderEvent(ctx, v))
	}

	m.broker.SendBatch(evts)

	return confirmations
}

// removeExpiredOrders remove all expired orders from the order book
// and also any pegged orders that are parked.
func (m *Market) removeExpiredStopOrders(
	ctx context.Context, timestamp int64, idgen common.IDGenerator,
) []*types.OrderConfirmation {
	timer := metrics.NewTimeCounter(m.mkt.ID, "market", "RemoveExpiredStopOrders")
	defer timer.EngineTimeCounterAdd()

	toExpire := m.expiringStopOrders.Expire(timestamp)
	stopOrders := m.stopOrders.RemoveExpired(toExpire)

	updatedAt := m.timeService.GetTimeNow()

	if m.as.InAuction() {
		m.removeExpiredStopOrdersInAuction(ctx, updatedAt, stopOrders)
		return nil
	}

	return m.removeExpiredStopOrdersInContinuous(ctx, updatedAt, stopOrders, idgen)
}

func (m *Market) removeExpiredStopOrdersInAuction(
	ctx context.Context,
	updatedAt time.Time,
	stopOrders []*types.StopOrder,
) {
	evts := []events.Event{}
	for _, v := range stopOrders {
		v.UpdatedAt = updatedAt
		v.Status = types.StopOrderStatusExpired
		// nothing to do, can send the event now
		evts = append(evts, events.NewStopOrderEvent(ctx, v))
	}

	m.broker.SendBatch(evts)
}

func (m *Market) removeExpiredStopOrdersInContinuous(
	ctx context.Context,
	updatedAt time.Time,
	stopOrders []*types.StopOrder,
	idgen common.IDGenerator,
) []*types.OrderConfirmation {
	evts := []events.Event{}
	filteredOCO := []*types.StopOrder{}
	for _, v := range stopOrders {
		v.UpdatedAt = updatedAt
		if v.Expiry.Expires() && *v.Expiry.ExpiryStrategy == types.StopOrderExpiryStrategySubmit && len(v.OCOLinkID) <= 0 {
			filteredOCO = append(filteredOCO, v)
			continue
		}
		// nothing to do, can send the event now
		evts = append(evts, events.NewStopOrderEvent(ctx, v))
	}

	m.broker.SendBatch(evts)

	return m.submitStopOrders(ctx, filteredOCO, types.StopOrderStatusExpired, idgen)
}

// removeExpiredOrders remove all expired orders from the order book
// and also any pegged orders that are parked.
func (m *Market) removeExpiredOrders(
	ctx context.Context, timestamp int64,
) []*types.Order {
	timer := metrics.NewTimeCounter(m.mkt.ID, "market", "RemoveExpiredOrders")
	defer timer.EngineTimeCounterAdd()

	expired := []*types.Order{}
	toExp := m.expiringOrders.Expire(timestamp)
	if len(toExp) == 0 {
		return expired
	}
	ids := make([]string, 0, len(toExp))
	for _, orderID := range toExp {
		var order *types.Order
		// The pegged expiry orders are copies and do not reflect the
		// current state of the order, therefore we look it up
		originalOrder, foundOnBook, err := m.getOrderByID(orderID)
		if err != nil {
			// nothing to do there.
			continue
		}
		// assign to the order the order from the book
		// so we get the most recent version from the book
		// to continue with
		order = originalOrder

		// if the order was on the book basically
		// either a pegged + non parked
		// or a non-pegged order
		if foundOnBook {
			m.position.UnregisterOrder(ctx, order)
			m.matching.DeleteOrder(order)
		}

		// if this was a pegged order
		// remove from the pegged / parked list
		if order.PeggedOrder != nil {
			m.removePeggedOrder(order)
		}

		// now we add to the list of expired orders
		// and assign the appropriate status
		order.UpdatedAt = m.timeService.GetTimeNow().UnixNano()
		order.Status = types.OrderStatusExpired
		expired = append(expired, order)
		ids = append(ids, orderID)
	}
	if len(ids) > 0 {
		m.broker.Send(events.NewExpiredOrdersEvent(ctx, m.mkt.ID, ids))
	}

	// If we have removed an expired order, do we need to reprice any
	// or maybe notify the liquidity engine
	if len(expired) > 0 && !m.as.InAuction() {
		m.checkForReferenceMoves(ctx, expired, false)
	}

	return expired
}

func (m *Market) getBestStaticAskPrice() (*num.Uint, error) {
	return m.matching.GetBestStaticAskPrice()
}

func (m *Market) getBestStaticAskPriceAndVolume() (*num.Uint, uint64, error) {
	return m.matching.GetBestStaticAskPriceAndVolume()
}

func (m *Market) getBestStaticBidPrice() (*num.Uint, error) {
	return m.matching.GetBestStaticBidPrice()
}

func (m *Market) getBestStaticBidPriceAndVolume() (*num.Uint, uint64, error) {
	return m.matching.GetBestStaticBidPriceAndVolume()
}

func (m *Market) getBestStaticPricesDecimal() (bid, ask num.Decimal, err error) {
	ask = num.DecimalZero()
	ubid, err := m.getBestStaticBidPrice()
	if err != nil {
		bid = num.DecimalZero()
		return
	}
	bid = ubid.ToDecimal()
	uask, err := m.getBestStaticAskPrice()
	if err != nil {
		ask = num.DecimalZero()
		return
	}
	ask = uask.ToDecimal()
	return
}

func (m *Market) getStaticMidPrice(side types.Side) (*num.Uint, error) {
	bid, err := m.matching.GetBestStaticBidPrice()
	if err != nil {
		return num.UintZero(), err
	}
	ask, err := m.matching.GetBestStaticAskPrice()
	if err != nil {
		return num.UintZero(), err
	}
	mid := num.UintZero()
	one := num.NewUint(1)
	two := num.Sum(one, one)
	one.Mul(one, m.priceFactor)
	if side == types.SideBuy {
		mid = mid.Div(num.Sum(bid, ask, one), two)
	} else {
		mid = mid.Div(num.Sum(bid, ask), two)
	}

	return mid, nil
}

// removePeggedOrder looks through the pegged and parked list
// and removes the matching order if found.
func (m *Market) removePeggedOrder(order *types.Order) {
	// remove if order was expiring
	m.expiringOrders.RemoveOrder(order.ExpiresAt, order.ID)
	// unpark will remove the order from the pegged orders data structure
	m.peggedOrders.Unpark(order.ID)
}

// getOrderBy looks for the order in the order book and in the list
// of pegged orders in the market. Returns the order if found, a bool
// representing if the order was found on the order book and any error code.
func (m *Market) getOrderByID(orderID string) (*types.Order, bool, error) {
	order, err := m.matching.GetOrderByID(orderID)
	if err == nil {
		return order, true, nil
	}

	// The pegged order list contains all the pegged orders in the system
	// whether they are parked or live. Check this list of a matching order
	if o := m.peggedOrders.GetParkedByID(orderID); o != nil {
		return o, false, nil
	}

	// We couldn't find it
	return nil, false, common.ErrOrderNotFound
}

func (m *Market) getTheoreticalTargetStake() *num.Uint {
	rf := m.risk.GetRiskFactors()

	// Ignoring the error as GetTheoreticalTargetStake handles trades==nil and len(trades)==0
	trades, _ := m.matching.GetIndicativeTrades()

	return m.tsCalc.GetTheoreticalTargetStake(
		*rf, m.timeService.GetTimeNow(), m.getReferencePrice(), trades)
}

func (m *Market) getTargetStake() *num.Uint {
	return m.tsCalc.GetTargetStake(*m.risk.GetRiskFactors(), m.timeService.GetTimeNow(), m.getCurrentMarkPrice())
}

func (m *Market) getSuppliedStake() *num.Uint {
	return m.liquidityEngine.CalculateSuppliedStake()
}

//nolint:unparam
func (m *Market) checkLiquidity(ctx context.Context, trades []*types.Trade, persistentOrder bool) bool {
	// before we check liquidity, ensure we've moved all funds that can go towards
	// provided stake to the bond accounts so we don't trigger liquidity auction for no reason
	m.checkBondBalance(ctx)
	var vBid, vAsk uint64
	// if we're not in auction, or we are checking liquidity when leaving opening auction, or we have best bid/ask volume
	if !m.as.InAuction() || m.matching.BidAndAskPresentAfterAuction() {
		_, vBid, _ = m.getBestStaticBidPriceAndVolume()
		_, vAsk, _ = m.getBestStaticAskPriceAndVolume()
	}

	return m.lMonitor.CheckLiquidity(
		m.as, m.timeService.GetTimeNow(),
		m.getSuppliedStake(),
		trades,
		*m.risk.GetRiskFactors(),
		m.getReferencePrice(),
		vBid, vAsk,
		persistentOrder)
}

// command liquidity auction checks if liquidity auction should be entered and if it can end.
func (m *Market) commandLiquidityAuction(ctx context.Context) {
	// start the liquidity monitoring auction if required
	if !m.as.InAuction() && m.as.AuctionStart() {
		m.enterAuction(ctx)
	}
	// end the liquidity monitoring auction if possible
	if m.as.InAuction() && m.as.CanLeave() && !m.as.IsOpeningAuction() {
		trades, err := m.matching.GetIndicativeTrades()
		if err != nil {
			m.log.Panic("Can't get indicative trades")
		}
		m.pMonitor.CheckPrice(ctx, m.as, trades, true)
		// TODO: Need to also get indicative trades and check how they'd impact target stake,
		// see  https://github.com/vegaprotocol/vega/issues/3047
		// If price monitoring doesn't trigger auction than leave it
		if evt := m.as.AuctionExtended(ctx, m.timeService.GetTimeNow()); evt != nil {
			m.broker.Send(evt)
		}
	}
}

func (m *Market) tradingTerminated(ctx context.Context, tt bool) {
	targetState := types.MarketStateSettled
	if m.mkt.State == types.MarketStatePending {
		targetState = types.MarketStateCancelled
	}
	m.tradingTerminatedWithFinalState(ctx, targetState, nil)
}

func (m *Market) tradingTerminatedWithFinalState(ctx context.Context, finalState types.MarketState, settlementDataInAsset *num.Uint) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// ignore trading termination while the governance proposal hasn't been enacted
	if m.mkt.State == types.MarketStateProposed {
		return
	}

	m.tradableInstrument.Instrument.Product.UnsubscribeTradingTerminated(ctx)

	if finalState != types.MarketStateCancelled {
		// we're either going to set state to trading terminated
		// or we'll be performing the final settlement (setting market status to settled)
		// in both cases, we want to MTM any pending trades
		if mp := m.getLastTradedPrice(); mp != nil && !mp.IsZero() && m.settlement.HasTraded() {
			// we need the ID-gen
			_, blockHash := vegacontext.TraceIDFromContext(ctx)
			m.idgen = idgeneration.New(blockHash + crypto.HashStrToHex("finalmtm"+m.GetID()))
			defer func() {
				m.idgen = nil
			}()
			// we have trades, and the market has been closed. Perform MTM sequence now so the final settlement
			// works as expected.
			m.markPrice = mp
			// send market data event with the updated mark price
			m.broker.Send(events.NewMarketDataEvent(ctx, m.GetMarketData()))
			m.confirmMTM(ctx, true)
		}
		m.mkt.State = types.MarketStateTradingTerminated
		m.mkt.TradingMode = types.MarketTradingModeNoTrading
		m.broker.Send(events.NewMarketUpdatedEvent(ctx, *m.mkt))
		var err error
		if settlementDataInAsset != nil {
			m.settlementDataWithLock(ctx, finalState, settlementDataInAsset)
		} else if m.settlementDataInMarket != nil {
			// because we need to be able to perform the MTM settlement, only update market state now
			settlementDataInAsset, err = m.tradableInstrument.Instrument.Product.ScaleSettlementDataToDecimalPlaces(m.settlementDataInMarket, m.assetDP)
			if err != nil {
				m.log.Error(err.Error())
			} else {
				m.settlementDataWithLock(ctx, finalState, settlementDataInAsset)
			}
		} else {
			m.log.Debug("no settlement data", logging.MarketID(m.GetID()))
		}
		return
	}
	for party := range m.parties {
		_, err := m.CancelAllOrders(ctx, party)
		if err != nil {
			m.log.Debug("could not cancel orders for party", logging.PartyID(party), logging.Error(err))
		}
	}
	err := m.closeCancelledMarket(ctx)
	if err != nil {
		m.log.Debug("could not close market", logging.MarketID(m.GetID()))
		return
	}

	m.log.Debug("market must not terminated before its enactment time", logging.MarketID(m.GetID()))
}

func (m *Market) settlementData(ctx context.Context, settlementData *num.Numeric) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.settlementDataInMarket = settlementData
	settlementDataInAsset, err := m.tradableInstrument.Instrument.Product.ScaleSettlementDataToDecimalPlaces(m.settlementDataInMarket, m.assetDP)
	if err != nil {
		m.log.Error(err.Error())
		return
	}
	m.settlementDataWithLock(ctx, types.MarketStateSettled, settlementDataInAsset)
}

func (m *Market) settlementDataPerp(ctx context.Context, settlementData *num.Numeric) {
	m.mu.Lock()
	defer m.mu.Unlock()

	_, blockHash := vegacontext.TraceIDFromContext(ctx)
	m.idgen = idgeneration.New(blockHash + crypto.HashStrToHex("perpsettlement"+m.GetID()))
	defer func() {
		m.idgen = nil
	}()

	// take all positions, get funding transfers
	sdi := settlementData.Int()
	if !settlementData.IsInt() && settlementData.Decimal() != nil {
		sdi = num.NewInt(settlementData.Decimal().IntPart())
	}
	if sdi == nil {
		return
	}
	transfers, round := m.settlement.SettleFundingPeriod(ctx, m.position.Positions(), settlementData.Int())
	if len(transfers) == 0 {
		m.log.Debug("Failed to get settle positions for funding period")
		return
	}

	margins, ledgerMovements, err := m.collateral.PerpsFundingSettlement(ctx, m.GetID(), transfers, m.settlementAsset, round)
	if err != nil {
		m.log.Error("Failed to get ledger movements when performing the funding settlement",
			logging.MarketID(m.GetID()),
			logging.Error(err))
		return
	}

	if len(ledgerMovements) > 0 {
		m.broker.Send(events.NewLedgerMovements(ctx, ledgerMovements))
	}
	// no margin events, no margin stuff to check
	if len(margins) == 0 {
		return
	}

	// check margin balances
	increment := m.tradableInstrument.Instrument.Product.GetMarginIncrease(m.timeService.GetTimeNow().UnixNano())
	riskUpdates := m.risk.UpdateMarginsOnSettlement(ctx, margins, m.getCurrentMarkPrice(), increment)
	// no margin accounts need updating...
	if len(riskUpdates) == 0 {
		return
	}
	// update margins, close-out any positions that don't have the required margin
	orderUpdates := m.handleRiskEvts(ctx, riskUpdates)
	m.checkForReferenceMoves(ctx, orderUpdates, false)
}

// NB this must be called with the lock already acquired.
func (m *Market) settlementDataWithLock(ctx context.Context, finalState types.MarketState, settlementDataInAsset *num.Uint) {
	if m.closed {
		return
	}

	if m.mkt.State == types.MarketStateTradingTerminated && settlementDataInAsset != nil {
		err := m.closeMarket(ctx, m.timeService.GetTimeNow(), finalState, settlementDataInAsset)
		if err != nil {
			m.log.Error("could not close market", logging.Error(err))
		}
		m.closed = m.mkt.State == finalState

		// mark price should be updated here
		if settlementDataInAsset != nil {
			m.lastTradedPrice = settlementDataInAsset.Clone()
			m.markPrice = settlementDataInAsset.Clone()
		}

		// send the market data with all updated stuff
		m.broker.Send(events.NewMarketDataEvent(ctx, m.GetMarketData()))
		m.broker.Send(events.NewMarketSettled(ctx, m.GetID(), m.timeService.GetTimeNow().UnixNano(), m.lastTradedPrice, m.positionFactor))
	}
}

func (m *Market) canTrade() bool {
	return m.mkt.State == types.MarketStateActive ||
		m.mkt.State == types.MarketStatePending ||
		m.mkt.State == types.MarketStateSuspended
}

// cleanupOnReject remove all resources created while the
// market was on PREPARED state.
// we'll need to remove all accounts related to the market
// all margin accounts for this market
// all bond accounts for this market too.
// at this point no fees would have been collected or anything
// like this.
func (m *Market) cleanupOnReject(ctx context.Context) {
	m.tradableInstrument.Instrument.Unsubscribe(ctx)

	// get the list of all parties in this market
	parties := make([]string, 0, len(m.parties))
	for k := range m.parties {
		parties = append(parties, k)
	}

	m.liquidity.StopAllLiquidityProvision(ctx)

	// cancel all pending orders
	orders := m.matching.Settled()
	// stop all parkedPeggedOrders
	parkedPeggedOrders := m.peggedOrders.Settled()

	evts := make([]events.Event, 0, len(orders)+len(parkedPeggedOrders))
	for _, o := range append(orders, parkedPeggedOrders...) {
		evts = append(evts, events.NewOrderEvent(ctx, o))
	}
	if len(evts) > 0 {
		m.broker.SendBatch(evts)
	}

	// now we do stop orders
	stopOrders := m.stopOrders.Settled()
	evts = make([]events.Event, 0, len(stopOrders))
	for _, o := range stopOrders {
		evts = append(evts, events.NewStopOrderEvent(ctx, o))
	}
	if len(evts) > 0 {
		m.broker.SendBatch(evts)
	}

	// release margin balance
	tresps, err := m.collateral.ClearMarket(ctx, m.GetID(), m.settlementAsset, parties, false)
	if err != nil {
		m.log.Panic("unable to cleanup a rejected market",
			logging.String("market-id", m.GetID()),
			logging.Error(err))
		return
	}

	m.stateVarEngine.UnregisterStateVariable(m.settlementAsset, m.mkt.ID)

	// then send the responses
	if len(tresps) > 0 {
		m.broker.Send(events.NewLedgerMovements(ctx, tresps))
	}
}

// GetTotalOrderBookLevelCount returns the total number of levels in the order book.
func (m *Market) GetTotalOrderBookLevelCount() uint64 {
	return m.matching.GetOrderBookLevelCount()
}

// GetTotalPeggedOrderCount returns the total number of pegged orders.
func (m *Market) GetTotalPeggedOrderCount() uint64 {
	return m.matching.GetPeggedOrdersCount()
}

// GetTotalStopOrderCount returns the total number of stop orders.
func (m *Market) GetTotalStopOrderCount() uint64 {
	return m.stopOrders.GetStopOrderCount()
}

// GetTotalOpenPositionCount returns the total number of open positions.
func (m *Market) GetTotalOpenPositionCount() uint64 {
	return m.position.GetOpenPositionCount()
}

// getMarketObservable returns current mark price once market is out of opening auction, during opening auction the indicative uncrossing price is returned.
func (m *Market) getMarketObservable(fallbackPrice *num.Uint) *num.Uint {
	// during opening auction we don't have a last traded price, so we use the indicative price instead
	if m.as.IsOpeningAuction() {
		if ip := m.matching.GetIndicativePrice(); !ip.IsZero() {
			return ip
		}
		// we don't have an indicative price yet so we use the supplied price
		return fallbackPrice
	}
	return m.getCurrentMarkPrice()
}

// Mark price gets returned when market is not in auction, otherwise indicative uncrossing price gets returned.
func (m *Market) getReferencePrice() *num.Uint {
	if !m.as.InAuction() {
		return m.getCurrentMarkPrice()
	}
	ip := m.matching.GetIndicativePrice() // can be zero
	if ip.IsZero() {
		return m.getCurrentMarkPrice()
	}
	return ip
}

func (m *Market) getCurrentMarkPrice() *num.Uint {
	if m.markPrice == nil {
		return num.UintZero()
	}
	return m.markPrice.Clone()
}

func (m *Market) getLastTradedPrice() *num.Uint {
	if m.lastTradedPrice == nil {
		return num.UintZero()
	}
	return m.lastTradedPrice.Clone()
}

func (m *Market) GetAssetForProposerBonus() string {
	return m.settlementAsset
}

func (m *Market) GetMarketCounters() *types.MarketCounters {
	return &types.MarketCounters{
		StopOrderCounter:    m.GetTotalStopOrderCount(),
		PeggedOrderCounter:  m.GetTotalPeggedOrderCount(),
		OrderbookLevelCount: m.GetTotalOrderBookLevelCount(),
		PositionCount:       m.GetTotalOpenPositionCount(),
	}
}
