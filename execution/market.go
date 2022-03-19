package execution

import (
	"context"
	"crypto/sha256"
	"encoding/base32"
	"encoding/binary"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/idgeneration"
	vegacontext "code.vegaprotocol.io/vega/libs/context"

	"code.vegaprotocol.io/vega/assets"
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/fee"
	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/liquidity"
	liquiditytarget "code.vegaprotocol.io/vega/liquidity/target"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/markets"
	"code.vegaprotocol.io/vega/matching"
	"code.vegaprotocol.io/vega/metrics"
	"code.vegaprotocol.io/vega/monitor"
	lmon "code.vegaprotocol.io/vega/monitor/liquidity"
	"code.vegaprotocol.io/vega/monitor/price"
	"code.vegaprotocol.io/vega/positions"
	"code.vegaprotocol.io/vega/products"
	"code.vegaprotocol.io/vega/risk"
	"code.vegaprotocol.io/vega/settlement"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
	"code.vegaprotocol.io/vega/types/statevar"

	"code.vegaprotocol.io/vega/libs/proto"
)

// InitialOrderVersion is set on `Version` field for every new order submission read from the network.
const InitialOrderVersion = 1

var (
	// ErrMarketClosed signals that an action have been tried to be applied on a closed market.
	ErrMarketClosed = errors.New("market closed")
	// ErrPartyDoNotExists signals that the party used does not exists.
	ErrPartyDoNotExists = errors.New("party does not exist")
	// ErrMarginCheckFailed signals that a margin check for a position failed.
	ErrMarginCheckFailed = errors.New("margin check failed")
	// ErrMarginCheckInsufficient signals that a margin had not enough funds.
	ErrMarginCheckInsufficient = errors.New("insufficient margin")
	// ErrMissingGeneralAccountForParty ...
	ErrMissingGeneralAccountForParty = errors.New("missing general account for party")
	// ErrNotEnoughVolumeToZeroOutNetworkOrder ...
	ErrNotEnoughVolumeToZeroOutNetworkOrder = errors.New("not enough volume to zero out network order")
	// ErrInvalidAmendRemainQuantity signals incorrect remaining qty for a reduce by amend.
	ErrInvalidAmendRemainQuantity = errors.New("incorrect remaining qty for a reduce by amend")
	// ErrEmptyMarketID is returned if processed market has an empty id.
	ErrEmptyMarketID = errors.New("invalid market id (empty)")
	// ErrInvalidOrderType is returned if processed order has an invalid order type.
	ErrInvalidOrderType = errors.New("invalid order type")
	// ErrInvalidExpiresAtTime is returned if the expire time is before the createdAt time.
	ErrInvalidExpiresAtTime = errors.New("invalid expiresAt time")
	// ErrGFAOrderReceivedDuringContinuousTrading is returned is a gfa order hits the market when the market is in continuous trading state.
	ErrGFAOrderReceivedDuringContinuousTrading = errors.New("gfa order received during continuous trading")
	// ErrGFNOrderReceivedAuctionTrading is returned if a gfn order hits the market when in auction state.
	ErrGFNOrderReceivedAuctionTrading = errors.New("gfn order received during auction trading")
	// ErrIOCOrderReceivedAuctionTrading is returned if a ioc order hits the market when in auction state.
	ErrIOCOrderReceivedAuctionTrading = errors.New("ioc order received during auction trading")
	// ErrFOKOrderReceivedAuctionTrading is returned if a fok order hits the market when in auction state.
	ErrFOKOrderReceivedAuctionTrading = errors.New("fok order received during auction trading")
	// ErrUnableToReprice we are unable to get a price required to reprice.
	ErrUnableToReprice = errors.New("unable to reprice")
	// ErrOrderNotFound we cannot find the order in the market.
	ErrOrderNotFound = errors.New("unable to find the order in the market")
	// ErrTradingNotAllowed no trading related functionalities are allowed in the current state.
	ErrTradingNotAllowed = errors.New("trading not allowed")
	// ErrCommitmentSubmissionNotAllowed no commitment submission are permitted in the current state.
	ErrCommitmentSubmissionNotAllowed = errors.New("commitment submission not allowed")
	// ErrNotEnoughStake is returned when a LP update results in not enough commitment.
	ErrNotEnoughStake = errors.New("commitment submission rejected, not enough stake")
	// ErrPartyNotLiquidityProvider is returned when a LP update or cancel does not match an LP party.
	ErrPartyNotLiquidityProvider = errors.New("party is not a liquidity provider")
	// ErrPartyAlreadyLiquidityProvider is returned when a LP is submitted by a party which is already LP.
	ErrPartyAlreadyLiquidityProvider = errors.New("party is already a liquidity provider")
	// ErrCannotRejectMarketNotInProposedState.
	ErrCannotRejectMarketNotInProposedState = errors.New("cannot reject a market not in proposed state")
	// ErrCannotStateOpeningAuctionForMarketNotInProposedState.
	ErrCannotStartOpeningAuctionForMarketNotInProposedState = errors.New("cannot start the opening auction for a market not in proposed state")
	// ErrCannotRepriceDuringAuction.
	ErrCannotRepriceDuringAuction = errors.New("cannot reprice during auction")

	one = num.One()
)

// PriceMonitor interface to handle price monitoring/auction triggers
// @TODO the interface shouldn't be imported here.
type PriceMonitor interface {
	CheckPrice(ctx context.Context, as price.AuctionState, p *num.Uint, v uint64, now time.Time, persistent bool) error
	GetCurrentBounds() []*types.PriceMonitoringBounds
	SetMinDuration(d time.Duration)
	GetValidPriceRange() (num.WrappedDecimal, num.WrappedDecimal)
	// Snapshot
	GetState() *types.PriceMonitor
	Changed() bool
	IsBoundFactorsInitialised() bool
}

// LiquidityMonitor.
type LiquidityMonitor interface {
	CheckLiquidity(as lmon.AuctionState, t time.Time, currentStake *num.Uint, trades []*types.Trade, rf types.RiskFactor, markPrice *num.Uint, bestStaticBidVolume, bestStaticAskVolume uint64)
	SetMinDuration(d time.Duration)
	UpdateTargetStakeTriggerRatio(ctx context.Context, ratio num.Decimal)
}

// TargetStakeCalculator interface.
type TargetStakeCalculator interface {
	types.StateProvider
	RecordOpenInterest(oi uint64, now time.Time) error
	GetTargetStake(rf types.RiskFactor, now time.Time, markPrice *num.Uint) *num.Uint
	GetTheoreticalTargetStake(rf types.RiskFactor, now time.Time, markPrice *num.Uint, trades []*types.Trade) *num.Uint
	UpdateScalingFactor(sFactor num.Decimal) error
	UpdateTimeWindow(tWindow time.Duration)
	Changed() bool
	StopSnapshots()
}

type MarketCollateral interface {
	Deposit(ctx context.Context, party, asset string, amount *num.Uint) (*types.TransferResponse, error)
	Withdraw(ctx context.Context, party, asset string, amount *num.Uint) (*types.TransferResponse, error)
	EnableAsset(ctx context.Context, asset types.Asset) error
	GetPartyGeneralAccount(party, asset string) (*types.Account, error)
	GetPartyBondAccount(market, partyID, asset string) (*types.Account, error)
	BondUpdate(ctx context.Context, market string, transfer *types.Transfer) (*types.TransferResponse, error)
	MarginUpdateOnOrder(ctx context.Context, marketID string, update events.Risk) (*types.TransferResponse, events.Margin, error)
	GetPartyMargin(pos events.MarketPosition, asset, marketID string) (events.Margin, error)
	GetPartyMarginAccount(market, party, asset string) (*types.Account, error)
	RollbackMarginUpdateOnOrder(ctx context.Context, marketID string, assetID string, transfer *types.Transfer) (*types.TransferResponse, error)
	GetOrCreatePartyBondAccount(ctx context.Context, partyID, marketID, asset string) (*types.Account, error)
	CreatePartyMarginAccount(ctx context.Context, partyID, marketID, asset string) (string, error)
	FinalSettlement(ctx context.Context, marketID string, transfers []*types.Transfer) ([]*types.TransferResponse, error)
	ClearMarket(ctx context.Context, mktID, asset string, parties []string) ([]*types.TransferResponse, error)
	HasGeneralAccount(party, asset string) bool
	ClearPartyMarginAccount(ctx context.Context, party, market, asset string) (*types.TransferResponse, error)
	CanCoverBond(market, party, asset string, amount *num.Uint) bool
	Hash() []byte
	TransferFeesContinuousTrading(ctx context.Context, marketID string, assetID string, ft events.FeesTransfer) ([]*types.TransferResponse, error)
	TransferFees(ctx context.Context, marketID string, assetID string, ft events.FeesTransfer) ([]*types.TransferResponse, error)
	MarginUpdate(ctx context.Context, marketID string, updates []events.Risk) ([]*types.TransferResponse, []events.Margin, []events.Margin, error)
	MarkToMarket(ctx context.Context, marketID string, transfers []events.Transfer, asset string) ([]events.Margin, []*types.TransferResponse, error)
	RemoveDistressed(ctx context.Context, parties []events.MarketPosition, marketID, asset string) (*types.TransferResponse, error)
	GetMarketLiquidityFeeAccount(market, asset string) (*types.Account, error)
	GetAssetQuantum(asset string) (num.Decimal, error)
}

// AuctionState ...
type AuctionState interface {
	price.AuctionState
	lmon.AuctionState
	// are we in auction, and what auction are we in?
	InAuction() bool
	IsOpeningAuction() bool
	IsPriceAuction() bool
	IsLiquidityAuction() bool
	IsFBA() bool
	IsMonitorAuction() bool
	// is it the start/end of an auction
	AuctionStart() bool
	CanLeave() bool
	// when does the auction start/end
	ExpiresAt() *time.Time
	Start() time.Time
	// signal we've started/ended the auction
	AuctionStarted(ctx context.Context, time time.Time) *events.Auction
	AuctionExtended(ctx context.Context, time time.Time) *events.Auction
	ExtendAuction(delta types.AuctionDuration)
	Left(ctx context.Context, now time.Time) *events.Auction
	// get some data
	Mode() types.MarketTradingMode
	Trigger() types.AuctionTrigger
	ExtensionTrigger() types.AuctionTrigger
	// UpdateMinDuration works out whether or not the current auction period (if applicable) should be extended
	UpdateMinDuration(ctx context.Context, d time.Duration) *events.Auction
	// Snapshot
	GetState() *types.AuctionState
	Changed() bool
}

// Market represents an instance of a market in vega and is in charge of calling
// the engines in order to process all transactions.
type Market struct {
	log   *logging.Logger
	idgen IDGenerator

	mkt *types.Market

	closingAt   time.Time
	currentTime time.Time

	mu sync.Mutex

	markPrice   *num.Uint
	priceFactor *num.Uint

	// own engines
	matching           *matching.CachedOrderBook
	tradableInstrument *markets.TradableInstrument
	risk               *risk.Engine
	position           *positions.SnapshotEngine
	settlement         *settlement.Engine
	fee                *fee.Engine
	liquidity          *liquidity.SnapshotEngine

	// deps engines
	collateral MarketCollateral

	broker Broker
	closed bool

	parties map[string]struct{}

	pMonitor PriceMonitor
	lMonitor LiquidityMonitor

	tsCalc TargetStakeCalculator

	as AuctionState

	peggedOrders   *PeggedOrders
	expiringOrders *ExpiringOrders

	// Store the previous price values so we can see what has changed
	lastBestBidPrice *num.Uint
	lastBestAskPrice *num.Uint
	lastMidBuyPrice  *num.Uint
	lastMidSellPrice *num.Uint

	lastMarketValueProxy    num.Decimal
	bondPenaltyFactor       num.Decimal
	marketValueWindowLength time.Duration

	// Liquidity Fee
	feeSplitter                *FeeSplitter
	lpFeeDistributionTimeStep  time.Duration
	lastEquityShareDistributed time.Time
	equityShares               *EquityShares
	minLPStakeQuantumMultiple  num.Decimal

	stateVarEngine StateVarEngine
	stateChanged   bool
	feesTracker    *FeesTracker
	marketTracker  *MarketTracker
	positionFactor num.Decimal // 10^pdp
}

// SetMarketID assigns a deterministic pseudo-random ID to a Market.
func SetMarketID(marketcfg *types.Market, seq uint64) error {
	marketcfg.ID = ""
	marketbytes, err := proto.Marshal(marketcfg.IntoProto())
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
	marketcfg.ID = base32.StdEncoding.EncodeToString(d)
	return nil
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
	collateralEngine MarketCollateral,
	oracleEngine products.OracleEngine,
	mkt *types.Market,
	now time.Time,
	broker Broker,
	as *monitor.AuctionState,
	stateVarEngine StateVarEngine,
	feesTracker *FeesTracker,
	assetDetails *assets.Asset,
	marketTracker *MarketTracker,
) (*Market, error) {
	if len(mkt.ID) == 0 {
		return nil, ErrEmptyMarketID
	}

	positionFactor := num.DecimalFromFloat(10).Pow(num.DecimalFromInt64(int64(mkt.PositionDecimalPlaces)))

	tradableInstrument, err := markets.NewTradableInstrument(ctx, log, mkt.TradableInstrument, oracleEngine)
	if err != nil {
		return nil, fmt.Errorf("unable to instantiate a new market: %w", err)
	}
	priceFactor := num.NewUint(1)
	if exp := assetDetails.DecimalPlaces() - mkt.DecimalPlaces; exp != 0 {
		priceFactor.Exp(num.NewUint(10), num.NewUint(exp))
	}

	// @TODO -> the raw auctionstate shouldn't be something exposed to the matching engine
	// as far as matching goes: it's either an auction or not
	book := matching.NewCachedOrderBook(
		log, matchingConfig, mkt.ID, as.InAuction())
	asset := tradableInstrument.Instrument.Product.GetAsset()

	riskEngine := risk.NewEngine(
		log,
		riskConfig,
		tradableInstrument.MarginCalculator,
		tradableInstrument.RiskModel,
		book,
		as,
		broker,
		now.UnixNano(),
		mkt.ID,
		asset,
		stateVarEngine,
		tradableInstrument.RiskModel.DefaultRiskFactors(),
		false,
		positionFactor,
	)

	settleEngine := settlement.New(
		log,
		settlementConfig,
		tradableInstrument.Instrument.Product,
		mkt.ID,
		broker,
		positionFactor,
	)
	positionEngine := positions.NewSnapshotEngine(log, positionConfig, mkt.ID, broker)

	feeEngine, err := fee.New(log, feeConfig, *mkt.Fees, asset, positionFactor)
	if err != nil {
		return nil, fmt.Errorf("unable to instantiate fee engine: %w", err)
	}

	tsCalc := liquiditytarget.NewSnapshotEngine(*mkt.LiquidityMonitoringParameters.TargetStakeParameters, positionEngine, mkt.ID, positionFactor)

	pMonitor, err := price.NewMonitor(asset, mkt.ID, tradableInstrument.RiskModel, mkt.PriceMonitoringSettings, stateVarEngine, log)
	if err != nil {
		return nil, fmt.Errorf("unable to instantiate price monitoring engine: %w", err)
	}

	lMonitor := lmon.NewMonitor(tsCalc, mkt.LiquidityMonitoringParameters)

	liqEngine := liquidity.NewSnapshotEngine(
		liquidityConfig, log, broker, tradableInstrument.RiskModel, pMonitor, asset, mkt.ID, stateVarEngine, priceFactor.Clone(), positionFactor)
	// call on chain time update straight away, so
	// the time in the engine is being updatedat creation
	liqEngine.OnChainTimeUpdate(ctx, now)

	// The market is initially create in a proposed state
	mkt.State = types.MarketStateProposed
	mkt.TradingMode = types.MarketTradingModeContinuous

	// Populate the market timestamps
	ts := &types.MarketTimestamps{
		Proposed: now.UnixNano(),
	}

	if mkt.OpeningAuction != nil {
		ts.Open = now.Add(time.Duration(mkt.OpeningAuction.Duration)).UnixNano()
	} else {
		ts.Open = now.UnixNano()
	}

	mkt.MarketTimestamps = ts

	market := &Market{
		log:                       log,
		idgen:                     nil,
		mkt:                       mkt,
		currentTime:               now,
		matching:                  book,
		tradableInstrument:        tradableInstrument,
		risk:                      riskEngine,
		position:                  positionEngine,
		settlement:                settleEngine,
		collateral:                collateralEngine,
		broker:                    broker,
		fee:                       feeEngine,
		liquidity:                 liqEngine,
		parties:                   map[string]struct{}{},
		as:                        as,
		pMonitor:                  pMonitor,
		lMonitor:                  lMonitor,
		tsCalc:                    tsCalc,
		peggedOrders:              NewPeggedOrders(),
		expiringOrders:            NewExpiringOrders(),
		feeSplitter:               NewFeeSplitter(),
		equityShares:              NewEquityShares(num.DecimalZero()),
		lastBestAskPrice:          num.Zero(),
		lastMidSellPrice:          num.Zero(),
		lastMidBuyPrice:           num.Zero(),
		lastBestBidPrice:          num.Zero(),
		stateChanged:              true,
		stateVarEngine:            stateVarEngine,
		feesTracker:               feesTracker,
		priceFactor:               priceFactor,
		minLPStakeQuantumMultiple: num.MustDecimalFromString("1"),
		marketTracker:             marketTracker,
		positionFactor:            positionFactor,
	}

	liqEngine.SetGetStaticPricesFunc(market.getBestStaticPricesDecimal)
	market.tradableInstrument.Instrument.Product.NotifyOnTradingTerminated(market.tradingTerminated)
	market.tradableInstrument.Instrument.Product.NotifyOnSettlementPrice(market.settlementPrice)
	return market, nil
}

func appendBytes(bz ...[]byte) []byte {
	var out []byte
	for _, b := range bz {
		out = append(out, b...)
	}
	return out
}

func (m *Market) IntoType() types.Market {
	return *m.mkt.DeepClone()
}

// UpdateRiskFactorsForTest is a hack for setting the risk factors for tests directly rather than through the consensus engine.
// Never use this for anything functional.
func (m *Market) UpdateRiskFactorsForTest() {
	m.risk.CalculateRiskFactorsForTest()
}

func (m *Market) Hash() []byte {
	mID := logging.String("market-id", m.GetID())
	matchingHash := m.matching.Hash()
	m.log.Debug("orderbook state hash", logging.Hash(matchingHash), mID)

	positionHash := m.position.Hash()
	m.log.Debug("positions state hash", logging.Hash(positionHash), mID)

	accountsHash := m.collateral.Hash()
	m.log.Debug("accounts state hash", logging.Hash(accountsHash), mID)

	return crypto.Hash(appendBytes(
		matchingHash, positionHash, accountsHash,
	))
}

func (m *Market) GetMarketState() types.MarketState {
	return m.mkt.State
}

func (m *Market) priceToMarketPrecision(price *num.Uint) *num.Uint {
	// we assume the price is cloned correctly already
	return price.Div(price, m.priceFactor)
}

func (m *Market) GetMarketData() types.MarketData {
	bestBidPrice, bestBidVolume, _ := m.matching.BestBidPriceAndVolume()
	bestOfferPrice, bestOfferVolume, _ := m.matching.BestOfferPriceAndVolume()
	bestStaticBidPrice, bestStaticBidVolume, _ := m.getBestStaticBidPriceAndVolume()
	bestStaticOfferPrice, bestStaticOfferVolume, _ := m.getBestStaticAskPriceAndVolume()

	// Auction related values
	indicativePrice := num.Zero()
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
	midPrice := num.Zero()
	if !bestBidPrice.IsZero() && !bestOfferPrice.IsZero() {
		midPrice = midPrice.Div(num.Sum(bestBidPrice, bestOfferPrice), two)
	}

	staticMidPrice := num.Zero()
	if !bestStaticBidPrice.IsZero() && !bestStaticOfferPrice.IsZero() {
		staticMidPrice = staticMidPrice.Div(num.Sum(bestStaticBidPrice, bestStaticOfferPrice), two)
	}

	var targetStake string
	if m.as.InAuction() {
		targetStake = m.priceToMarketPrecision(m.getTheoreticalTargetStake()).String()
	} else {
		targetStake = m.priceToMarketPrecision(m.getTargetStake()).String()
	}
	bounds := m.pMonitor.GetCurrentBounds()
	for _, b := range bounds {
		m.priceToMarketPrecision(b.MaxValidPrice) // effictively floors this
		m.priceToMarketPrecision(b.MinValidPrice)
		if m.priceFactor.NEQ(one) {
			b.MinValidPrice.AddSum(one) // ceil
		}
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
		Timestamp:                 m.currentTime.UnixNano(),
		OpenInterest:              m.position.GetOpenInterest(),
		IndicativePrice:           m.priceToMarketPrecision(indicativePrice),
		IndicativeVolume:          indicativeVolume,
		AuctionStart:              auctionStart,
		AuctionEnd:                auctionEnd,
		MarketTradingMode:         m.as.Mode(),
		Trigger:                   m.as.Trigger(),
		ExtensionTrigger:          m.as.ExtensionTrigger(),
		TargetStake:               targetStake,
		SuppliedStake:             m.getSuppliedStake().String(),
		PriceMonitoringBounds:     bounds,
		MarketValueProxy:          m.lastMarketValueProxy.BigInt().String(),
		LiquidityProviderFeeShare: lpsToLiquidityProviderFeeShare(m.equityShares.lps),
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
	if m.mkt.State != types.MarketStateProposed {
		return ErrCannotRejectMarketNotInProposedState
	}

	// we closed all parties accounts
	m.cleanupOnReject(ctx)
	m.mkt.State = types.MarketStateRejected
	m.broker.Send(events.NewMarketUpdatedEvent(ctx, *m.mkt))
	m.stateChanged = true

	return nil
}

// CanLeaveOpeningAuction checks if the market can leave the opening auction based on whether floating point consensus has been reached on all 3 vars.
func (m *Market) CanLeaveOpeningAuction() bool {
	boundFactorsInitialised := m.pMonitor.IsBoundFactorsInitialised()
	potInitialised := m.liquidity.IsPoTInitialised()
	riskFactorsInitialised := m.risk.IsRiskFactorInitialised()
	canLeave := boundFactorsInitialised && potInitialised && riskFactorsInitialised
	if !canLeave {
		m.log.Info("Cannot leave opening auction", logging.String("market", m.mkt.ID), logging.Bool("bound-factors-initialised", boundFactorsInitialised), logging.Bool("pot-initialised", potInitialised), logging.Bool("risk-factors-initialised", riskFactorsInitialised))
	}
	return canLeave
}

func (m *Market) StartOpeningAuction(ctx context.Context) error {
	if m.mkt.State != types.MarketStateProposed {
		return ErrCannotStartOpeningAuctionForMarketNotInProposedState
	}

	// now we start the opening auction
	if m.as.AuctionStart() {
		// we are now in a pending state
		m.mkt.State = types.MarketStatePending
		m.mkt.MarketTimestamps.Pending = m.currentTime.UnixNano()
		m.enterAuction(ctx)
	} else {
		// TODO(): to be removed once we don't have market starting
		// without an opening auction
		m.mkt.State = types.MarketStateActive
	}

	m.broker.Send(events.NewMarketUpdatedEvent(ctx, *m.mkt))
	return nil
}

// GetID returns the id of the given market.
func (m *Market) GetID() string {
	return m.mkt.ID
}

func (m *Market) PostRestore(ctx context.Context) error {
	m.settlement.Update(m.position.Positions())

	pps := m.position.Parties()
	peggedOrder := m.peggedOrders.GetAll()
	parties := make(map[string]struct{}, len(pps)+len(peggedOrder))

	for _, p := range pps {
		parties[p] = struct{}{}
	}

	for _, o := range m.peggedOrders.GetAll() {
		parties[o.Party] = struct{}{}
	}
	return nil
}

// OnChainTimeUpdate notifies the market of a new time event/update.
// todo: make this a more generic function name e.g. OnTimeUpdateEvent
func (m *Market) OnChainTimeUpdate(ctx context.Context, t time.Time) bool {
	timer := metrics.NewTimeCounter(m.mkt.ID, "market", "OnChainTimeUpdate")
	m.mu.Lock()
	defer m.mu.Unlock()

	_, blockHash := vegacontext.TraceIDFromContext(ctx)
	m.idgen = idgeneration.New(blockHash)
	defer func() { m.idgen = nil }()

	if m.closed {
		return true
	}

	// some engines still needs to get updates:
	m.currentTime = t
	m.peggedOrders.OnTimeUpdate(t)
	m.liquidity.OnChainTimeUpdate(ctx, t)
	m.risk.OnTimeUpdate(t)
	m.settlement.OnTick(t)
	m.feeSplitter.SetCurrentTime(t)

	m.stateChanged = true

	// TODO(): This also assume that the market is not
	// being closed before the market is leaving
	// the opening auction, but settlement at expiry is
	// not even specced or implemented as of now...
	// if the state of the market is just PROPOSED,
	// we will just skip everything there as nothing apply.
	if m.mkt.State == types.MarketStateProposed {
		return false
	}

	// if in somce case we're still in trading terminated state and tried to settle but failed,
	// as long as there's settlement price we can retry
	settlementPrice, _ := m.tradableInstrument.Instrument.Product.SettlementPrice()

	if m.mkt.State == types.MarketStateTradingTerminated {
		// if we now have settlement price - try to settle and close the market
		if settlementPrice != nil {
			m.closeMarket(ctx, t)
		}
		m.closed = m.mkt.State == types.MarketStateSettled
		return m.closed
	}

	// distribute liquidity fees each `m.lpFeeDistributionTimeStep`
	if t.Sub(m.lastEquityShareDistributed) > m.lpFeeDistributionTimeStep {
		m.lastEquityShareDistributed = t
		m.stateChanged = true

		if err := m.distributeLiquidityFees(ctx); err != nil {
			m.log.Panic("liquidity fee distribution error", logging.Error(err))
		}
	}

	// check auction, if any
	m.checkAuction(ctx, t)
	timer.EngineTimeCounterAdd()

	m.updateMarketValueProxy()
	m.broker.Send(events.NewMarketTick(ctx, m.mkt.ID, t))
	return m.closed
}

func (m *Market) updateMarketValueProxy() {
	// if windows length is reached, reset fee splitter
	if mvwl := m.marketValueWindowLength; m.feeSplitter.Elapsed() > mvwl {
		m.feeSplitter.TimeWindowStart(m.currentTime)
	}

	// these need to happen every block
	// but also when new LP is submitted just so we are sure we do
	// not have a mvp of 0
	ts := m.liquidity.ProvisionsPerParty().TotalStake()
	m.lastMarketValueProxy = m.feeSplitter.MarketValueProxy(
		m.marketValueWindowLength, ts)
	m.equityShares.WithMVP(m.lastMarketValueProxy)
	m.stateChanged = true
}

func (m *Market) closeMarket(ctx context.Context, t time.Time) error {
	// market is closed, final settlement
	// call settlement and stuff
	positions, err := m.settlement.Settle(t)
	if err != nil {
		m.log.Error("Failed to get settle positions on market closed",
			logging.Error(err))

		return err
	}

	transfers, err := m.collateral.FinalSettlement(ctx, m.GetID(), positions)
	if err != nil {
		m.log.Error("Failed to get ledger movements after settling closed market",
			logging.MarketID(m.GetID()),
			logging.Error(err))
		return err
	}

	// @TODO pass in correct context -> Previous or next block?
	// Which is most appropriate here?
	// this will be next block
	m.broker.Send(events.NewTransferResponse(ctx, transfers))

	asset, _ := m.mkt.GetAsset()
	parties := make([]string, 0, len(m.parties))
	for k := range m.parties {
		parties = append(parties, k)
	}

	clearMarketTransfers, err := m.collateral.ClearMarket(ctx, m.GetID(), asset, parties)
	if err != nil {
		m.log.Error("Clear market error",
			logging.MarketID(m.GetID()),
			logging.Error(err))
		return err
	}

	m.broker.Send(events.NewTransferResponse(ctx, clearMarketTransfers))
	m.mkt.State = types.MarketStateSettled
	m.broker.Send(events.NewMarketUpdatedEvent(ctx, *m.mkt))

	m.stateChanged = true
	return nil
}

func (m *Market) unregisterAndReject(ctx context.Context, order *types.Order, err error) error {
	_ = m.position.UnregisterOrder(ctx, order)
	order.UpdatedAt = m.currentTime.UnixNano()
	order.Status = types.OrderStatusRejected
	if oerr, ok := types.IsOrderError(err); ok {
		// the order wasn't invalid, so stopped is a better status, rather than rejected.
		if oerr == types.OrderErrorNonPersistentOrderOutOfPriceBounds {
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
		return num.Zero(), ErrCannotRepriceDuringAuction
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
		return num.Zero(), ErrUnableToReprice
	}

	offset := num.Zero().Mul(order.PeggedOrder.Offset, m.priceFactor)
	if order.Side == types.SideSell {
		return price.AddSum(offset), nil
	}

	if price.LTE(offset) {
		return num.Zero(), ErrUnableToReprice
	}

	return num.Zero().Sub(price, offset), nil
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
	toPark := m.peggedOrders.GetAllActiveOrders()
	for _, order := range toPark {
		m.parkOrder(ctx, order)
	}
	return toPark
}

// EnterAuction : Prepare the order book to be run as an auction.
func (m *Market) enterAuction(ctx context.Context) {
	// Change market type to auction
	ordersToCancel := m.matching.EnterAuction()

	// Move into auction mode to prevent pegged order repricing
	event := m.as.AuctionStarted(ctx, m.currentTime)

	// this is at least the size of the orders to be cancelled
	updatedOrders := make([]*types.Order, 0, len(ordersToCancel))

	// Cancel all the orders that were invalid
	for _, order := range ordersToCancel {
		_, err := m.cancelOrder(ctx, order.Party, order.ID)
		if err != nil {
			m.log.Debug("error cancelling order when entering auction",
				logging.MarketID(m.GetID()),
				logging.OrderID(order.ID),
				logging.Error(err))
		}
		updatedOrders = append(updatedOrders, order)
	}

	// now update all special orders
	m.enterAuctionSpecialOrders(ctx, updatedOrders)

	// Send an event bus update
	m.broker.Send(event)

	if m.as.InAuction() && (m.as.IsLiquidityAuction() || m.as.IsPriceAuction()) {
		m.mkt.State = types.MarketStateSuspended
		m.broker.Send(events.NewMarketUpdatedEvent(ctx, *m.mkt))
		m.stateChanged = true
	}
}

// OnOpeningAuctionFirstUncrossingPrice is triggered when the opening auction sees an uncrossing price for the first time and emits
// an event to the state variable engine.
func (m *Market) OnOpeningAuctionFirstUncrossingPrice() {
	m.log.Info("OnOpeningAuctionFirstUncrossingPrice event fired", logging.String("market", m.mkt.ID))
	asset, _ := m.mkt.GetAsset()
	m.stateVarEngine.ReadyForTimeTrigger(asset, m.mkt.ID)
	m.stateVarEngine.NewEvent(asset, m.mkt.ID, statevar.StateVarEventTypeOpeningAuctionFirstUncrossingPrice)
}

// OnAuctionEnded is called whenever an auction is ended and emits an event to the state var engine.
func (m *Market) OnAuctionEnded() {
	m.log.Info("OnAuctionEnded event fired", logging.String("market", m.mkt.ID))
	asset, _ := m.mkt.GetAsset()
	m.stateVarEngine.NewEvent(asset, m.mkt.ID, statevar.StateVarEventTypeAuctionEnded)
}

// leaveAuction : Return the orderbook and market to continuous trading.
func (m *Market) leaveAuction(ctx context.Context, now time.Time) {
	defer func() {
		if !m.as.InAuction() && m.mkt.State == types.MarketStateSuspended {
			m.mkt.State = types.MarketStateActive
			m.broker.Send(events.NewMarketUpdatedEvent(ctx, *m.mkt))
			m.stateChanged = true
		}
	}()

	// Change market type to continuous trading
	uncrossedOrders, ordersToCancel, err := m.matching.LeaveAuction(m.currentTime)
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

	// now that we're left the auction, we can mark all positions
	// in case any party is distressed (Which shouldn't be possible)
	// we'll fall back to the a network order at the new mark price (mid-price)
	cmp := m.getCurrentMarkPrice()
	mcmp := num.Zero().Div(cmp, m.priceFactor) // create the market representation of the price
	m.confirmMTM(ctx, &types.Order{
		ID:            m.idgen.NextID(),
		Price:         cmp,
		OriginalPrice: mcmp,
	})

	// keep var to see if we're leaving opening auction
	isOpening := m.as.IsOpeningAuction()
	// update auction state, so we know what the new tradeMode ought to be
	endEvt := m.as.Left(ctx, now)

	for _, uncrossedOrder := range uncrossedOrders {
		if !isOpening {
			// @TODO we should update this once
			for _, trade := range uncrossedOrder.Trades {
				err := m.pMonitor.CheckPrice(
					ctx, m.as, trade.Price.Clone(), trade.Size, now, true,
				)
				if err != nil {
					m.log.Panic("unable to run check price with price monitor",
						logging.String("market-id", m.GetID()),
						logging.Error(err))
				}
			}
		}

		updatedOrders = append(updatedOrders, uncrossedOrder.Order)
		updatedOrders = append(
			updatedOrders, uncrossedOrder.PassiveOrdersAffected...)
	}

	// Send an event bus update
	m.broker.Send(endEvt)
	m.checkForReferenceMoves(ctx, updatedOrders, true)
	m.checkLiquidity(ctx, nil)
	m.commandLiquidityAuction(ctx)
	m.updateLiquidityFee(ctx)
	m.OnAuctionEnded()
}

func (m *Market) validatePeggedOrder(order *types.Order) types.OrderError {
	if order.Type != types.OrderTypeLimit {
		// All pegged orders must be LIMIT orders
		return types.ErrPeggedOrderMustBeLimitOrder
	}

	if order.TimeInForce != types.OrderTimeInForceGTT && order.TimeInForce != types.OrderTimeInForceGTC {
		// Pegged orders can only be GTC or GTT
		return types.ErrPeggedOrderMustBeGTTOrGTC
	}

	if order.PeggedOrder.Reference == types.PeggedReferenceUnspecified {
		// We must specify a valid reference
		return types.ErrPeggedOrderWithoutReferencePrice
	}

	if order.Side == types.SideBuy {
		switch order.PeggedOrder.Reference {
		case types.PeggedReferenceBestAsk:
			return types.ErrPeggedOrderBuyCannotReferenceBestAskPrice
		case types.PeggedReferenceMid:
			if order.PeggedOrder.Offset.IsZero() {
				return types.ErrPeggedOrderOffsetMustBeGreaterThanZero
			}
		}
	} else {
		switch order.PeggedOrder.Reference {
		case types.PeggedReferenceBestBid:
			return types.ErrPeggedOrderSellCannotReferenceBestBidPrice
		case types.PeggedReferenceMid:
			if order.PeggedOrder.Offset.IsZero() {
				return types.ErrPeggedOrderOffsetMustBeGreaterThanZero
			}
		}
	}
	return types.OrderErrorUnspecified
}

func (m *Market) validateOrder(ctx context.Context, order *types.Order) error {
	// Check we are allowed to handle this order type with the current market status
	isAuction := m.as.InAuction()
	if isAuction && order.TimeInForce == types.OrderTimeInForceGFN {
		order.Status = types.OrderStatusRejected
		order.Reason = types.OrderErrorGFNOrderDuringAnAuction
		m.broker.Send(events.NewOrderEvent(ctx, order))
		return ErrGFNOrderReceivedAuctionTrading
	}

	if isAuction && order.TimeInForce == types.OrderTimeInForceIOC {
		order.Status = types.OrderStatusRejected
		order.Reason = types.OrderErrorCannotSendIOCOrderDuringAuction
		m.broker.Send(events.NewOrderEvent(ctx, order))
		return ErrIOCOrderReceivedAuctionTrading
	}

	if isAuction && order.TimeInForce == types.OrderTimeInForceFOK {
		order.Status = types.OrderStatusRejected
		order.Reason = types.OrderErrorCannotSendFOKOrderDurinAuction
		m.broker.Send(events.NewOrderEvent(ctx, order))
		return ErrFOKOrderReceivedAuctionTrading
	}

	if !isAuction && order.TimeInForce == types.OrderTimeInForceGFA {
		order.Status = types.OrderStatusRejected
		order.Reason = types.OrderErrorGFAOrderDuringContinuousTrading
		m.broker.Send(events.NewOrderEvent(ctx, order))
		return ErrGFAOrderReceivedDuringContinuousTrading
	}

	// Check the expiry time is valid
	if order.ExpiresAt > 0 && order.ExpiresAt < order.CreatedAt {
		order.Status = types.OrderStatusRejected
		order.Reason = types.OrderErrorInvalidExpirationDatetime
		m.broker.Send(events.NewOrderEvent(ctx, order))
		return ErrInvalidExpiresAtTime
	}

	if m.closed {
		// adding order to the buffer first
		order.Status = types.OrderStatusRejected
		order.Reason = types.OrderErrorMarketClosed
		m.broker.Send(events.NewOrderEvent(ctx, order))
		return ErrMarketClosed
	}

	if order.Type == types.OrderTypeNetwork {
		// adding order to the buffer first
		order.Status = types.OrderStatusRejected
		order.Reason = types.OrderErrorInvalidType
		m.broker.Send(events.NewOrderEvent(ctx, order))
		return ErrInvalidOrderType
	}

	// Validate market
	if order.MarketID != m.mkt.ID {
		// adding order to the buffer first
		order.Status = types.OrderStatusRejected
		order.Reason = types.OrderErrorInvalidMarketID
		m.broker.Send(events.NewOrderEvent(ctx, order))

		if m.log.GetLevel() == logging.DebugLevel {
			m.log.Debug("Market ID mismatch",
				logging.Order(*order),
				logging.String("market", m.mkt.ID))
		}
		return types.ErrInvalidMarketID
	}

	// Validate pegged orders
	if order.PeggedOrder != nil {
		reason := m.validatePeggedOrder(order)
		if reason != types.OrderErrorUnspecified {
			order.Status = types.OrderStatusRejected
			order.Reason = reason

			m.broker.Send(events.NewOrderEvent(ctx, order))

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
	asset, _ := m.mkt.GetAsset()
	if !m.collateral.HasGeneralAccount(order.Party, asset) {
		// adding order to the buffer first
		order.Status = types.OrderStatusRejected
		order.Reason = types.OrderErrorInsufficientAssetBalance
		m.broker.Send(events.NewOrderEvent(ctx, order))

		// party should be created before even trying to post order
		return ErrPartyDoNotExists
	}

	// ensure party have a general account, and margin account is / can be created
	_, err := m.collateral.CreatePartyMarginAccount(ctx, order.Party, order.MarketID, asset)
	if err != nil {
		m.log.Error("Margin account verification failed",
			logging.String("party-id", order.Party),
			logging.String("market-id", m.GetID()),
			logging.String("asset", asset),
		)
		// adding order to the buffer first
		order.Status = types.OrderStatusRejected
		order.Reason = types.OrderErrorMissingGeneralAccount
		m.broker.Send(events.NewOrderEvent(ctx, order))
		return ErrMissingGeneralAccountForParty
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
		// position was never created or party went distressed and don't exist
		// all good we can return
		return
	}

	// now check if all buy/sell/size are 0
	if pos.Buy() != 0 || pos.Sell() != 0 || pos.Size() != 0 || !pos.VWBuy().IsZero() || !pos.VWSell().IsZero() {
		// position is not 0, nothing to release surely
		return
	}

	asset, _ := m.mkt.GetAsset()
	transfers, err := m.collateral.ClearPartyMarginAccount(
		ctx, partyID, m.GetID(), asset)
	if err != nil {
		m.log.Error("unable to clear party margin account", logging.Error(err))
		return
	}
	evt := events.NewTransferResponse(
		ctx, []*types.TransferResponse{transfers})
	m.broker.Send(evt)
}

// SubmitOrder submits the given order.
func (m *Market) SubmitOrder(
	ctx context.Context,
	orderSubmission *types.OrderSubmission,
	party string,
	deterministicId string,
) (oc *types.OrderConfirmation, _ error) {
	defer func() {
		if oc != nil {
			party := ""
			if oc.Order.Reference == "jeremy-debug" {
				fmt.Printf("SUBMIT ORDER  : %v\n", oc.Order.String())
				party = oc.Order.Party
			}
			for _, v := range oc.PassiveOrdersAffected {
				if v.Reference == "jeremy-debug" {
					fmt.Printf("SUBMIT PASSIVE: %v\n", v.String())
					party = v.Party
				}
			}

			if party != "" {
				for _, v := range oc.Trades {
					if v.Buyer == "jeremy-debug" || v.SellOrder == "jeremy-debug" {
						fmt.Printf("SUBMIT TRADE  : %v\n", v.String())
					}
				}
			}
		}
	}()

	m.idgen = idgeneration.New(deterministicId)
	defer func() { m.idgen = nil }()

	order := orderSubmission.IntoOrder(party)
	if order.Price != nil {
		order.OriginalPrice = order.Price.Clone()
		order.Price.Mul(order.Price, m.priceFactor)
	}
	order.CreatedAt = m.currentTime.UnixNano()

	if !m.canTrade() {
		order.Status = types.OrderStatusRejected
		order.Reason = types.OrderErrorMarketClosed
		m.broker.Send(events.NewOrderEvent(ctx, order))
		return nil, ErrTradingNotAllowed
	}

	order.ID = m.idgen.NextID()
	conf, orderUpdates, err := m.submitOrder(ctx, order)
	if err != nil {
		return nil, err
	}

	allUpdatedOrders := append(
		[]*types.Order{conf.Order}, conf.PassiveOrdersAffected...)
	allUpdatedOrders = append(allUpdatedOrders, orderUpdates...)

	m.checkForReferenceMoves(
		ctx, allUpdatedOrders, false)
	m.checkLiquidity(ctx, nil)
	m.commandLiquidityAuction(ctx)

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
	order.Version = InitialOrderVersion
	order.Status = types.OrderStatusActive

	if err := m.validateOrder(ctx, order); err != nil {
		return nil, nil, err
	}

	if err := m.validateAccounts(ctx, order); err != nil {
		return nil, nil, err
	}

	if order.PeggedOrder != nil {
		// Add pegged order to time sorted list
		m.peggedOrders.Add(order)
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
			// If we are in an auction, we don't insert this order into the book
			// Maybe should return an orderConfirmation with order state PARKED
			m.broker.Send(events.NewOrderEvent(ctx, order))
			return &types.OrderConfirmation{Order: order}, nil, nil
		} else {
			// Reprice
			err := m.repricePeggedOrder(order)
			if err != nil {
				fmt.Printf("ERROR: %v\n\n", err)
				m.broker.Send(events.NewOrderEvent(ctx, order))
				return &types.OrderConfirmation{Order: order}, nil, nil // nolint
			}
		}
	}

	oldPos, ok := m.position.GetPositionByPartyID(order.Party)
	// Register order as potential positions
	pos := m.position.RegisterOrder(ctx, order)
	checkMargin := true
	if !isPegged && ok {
		oldVol, newVol := pos.Size()+pos.Buy()-pos.Sell(), oldPos.Size()+pos.Buy()-pos.Sell()
		if oldVol < 0 {
			oldVol = -oldVol
		}
		if newVol < 0 {
			newVol = -newVol
		}
		// check margin if the new volume is greater, or the same (implying long to short, or short to long)
		checkMargin = oldVol <= newVol
	}

	// Perform check and allocate margin unless the order is (partially) closing the party position
	if checkMargin {
		if err := m.checkMarginForOrder(ctx, pos, order); err != nil {
			if m.log.GetLevel() <= logging.DebugLevel {
				m.log.Debug("Unable to check/add margin for party",
					logging.Order(*order), logging.Error(err))
			}
			_ = m.unregisterAndReject(
				ctx, order, types.OrderErrorMarginCheckFailed)
			return nil, nil, ErrMarginCheckFailed
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
	if m.as.InAuction() && (isPegged || order.IsLiquidityOrder()) {
		return &types.OrderConfirmation{Order: order}, nil, nil
	}

	order.Status = types.OrderStatusActive

	// Send the aggressive order into matching engine
	confirmation, err := m.matching.SubmitOrder(order)
	if err != nil {
		return nil, nil, m.unregisterAndReject(ctx, order, err)
	}

	// if order was FOK or IOC some or all of it may have not be consumed, so we need to
	// remove them from the potential orders,
	// then we should be able to process the rest of the order properly.
	if ((order.TimeInForce == types.OrderTimeInForceFOK ||
		order.TimeInForce == types.OrderTimeInForceIOC ||
		order.Status == types.OrderStatusStopped) &&
		confirmation.Order.Remaining != 0) ||
		// Also do it if specifically we went against a wash trade
		(order.Status == types.OrderStatusRejected &&
			order.Reason == types.OrderErrorSelfTrading) {
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
	persistent := true
	switch order.TimeInForce {
	case types.OrderTimeInForceFOK, types.OrderTimeInForceGFN, types.OrderTimeInForceIOC:
		persistent = false
	}

	for _, t := range trades {
		if merr := m.pMonitor.CheckPrice(ctx, m.as, t.Price.Clone(), t.Size, m.currentTime, persistent); merr != nil {
			// a specific order error
			if err, ok := merr.(types.OrderError); ok {
				return nil, err
			}
			m.log.Panic("unable to run check price with price monitor",
				logging.String("market-id", m.GetID()),
				logging.Error(merr))
		}
	}

	if evt := m.as.AuctionExtended(ctx, m.currentTime); evt != nil {
		m.broker.Send(evt)
	}
	m.checkLiquidity(ctx, trades)

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

	var (
		transfers []*types.TransferResponse
		asset, _  = m.mkt.GetAsset()
	)

	if !m.as.InAuction() {
		transfers, err = m.collateral.TransferFeesContinuousTrading(ctx, m.GetID(), asset, fees)
	} else if m.as.IsMonitorAuction() {
		// @TODO handle this properly
		transfers, err = m.collateral.TransferFees(ctx, m.GetID(), asset, fees)
	} else if m.as.IsFBA() {
		// @TODO implement transfer for auction types
		transfers, err = m.collateral.TransferFees(ctx, m.GetID(), asset, fees)
	}

	if err != nil {
		m.log.Error("unable to transfer fees for trades",
			logging.String("order-id", order.ID),
			logging.String("market-id", m.GetID()),
			logging.Error(err))
		return types.OrderErrorInsufficientFundsToPayFees
	}

	// send transfers through the broker
	if err == nil && len(transfers) > 0 {
		evt := events.NewTransferResponse(ctx, transfers)
		m.broker.Send(evt)
	}

	m.feesTracker.UpdateFeesFromTransfers(fees.Transfers())

	return nil
}

func (m *Market) handleConfirmationPassiveOrders(
	ctx context.Context,
	conf *types.OrderConfirmation) {
	if conf.PassiveOrdersAffected != nil {
		var (
			evts        = make([]events.Event, 0, len(conf.PassiveOrdersAffected))
			currentTime = m.currentTime.UnixNano()
		)

		// Insert or update passive orders siting on the book
		for _, order := range conf.PassiveOrdersAffected {
			// set the `updatedAt` value as these orders have changed
			order.UpdatedAt = currentTime
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

func (m *Market) handleConfirmation(ctx context.Context, conf *types.OrderConfirmation) (orderUpdates []*types.Order) {
	// When re-submitting liquidity order, it happen that the pricing is putting
	// the order at a price which makes it uncross straight away.
	// then triggering this handleConfirmation flow, etc.
	// Although the order is considered aggressive, and we never expect in the flow
	// for an aggressive order to be pegged, so we never remove them from the pegged
	// list. All this impact the float of EnterAuction, which if triggered from there
	// will try to park all pegged orders, including this order which have never been
	// remove from the pegged list. We add this check to make sure  that if the
	// aggressive order is pegged, we then do remove it from the list.
	if conf.Order.PeggedOrder != nil {
		if conf.Order.Remaining == 0 || conf.Order.Status != types.OrderStatusActive {
			m.removePeggedOrder(conf.Order)
		}
	}

	m.handleConfirmationPassiveOrders(ctx, conf)
	end := m.as.CanLeave()

	if len(conf.Trades) > 0 {
		// Calculate and set current mark price
		m.setMarkPrice(conf.Trades[len(conf.Trades)-1])

		// Insert all trades resulted from the executed order
		tradeEvts := make([]events.Event, 0, len(conf.Trades))
		for idx, trade := range conf.Trades {
			trade.SetIDs(m.idgen.NextID(), conf.Order, conf.PassiveOrdersAffected[idx])

			tradeEvts = append(tradeEvts, events.NewTradeEvent(ctx, *trade))

			// Update positions (this communicates with settlement via channel)
			m.position.Update(trade)
			// Record open interest change
			if err := m.tsCalc.RecordOpenInterest(m.position.GetOpenInterest(), m.currentTime); err != nil {
				m.log.Debug("unable record open interest",
					logging.String("market-id", m.GetID()),
					logging.Error(err))
			}
			// add trade to settlement engine for correct MTM settlement of individual trades
			m.settlement.AddTrade(trade)
			tradeValue, _ := num.UintFromDecimal(num.DecimalFromInt64(int64(trade.Size)).Mul(trade.Price.ToDecimal()).Div(m.positionFactor))
			m.feeSplitter.AddTradeValue(tradeValue)
			m.marketTracker.AddValueTraded(m.mkt.ID, tradeValue)
		}
		m.broker.SendBatch(tradeEvts)

		if !end {
			orderUpdates = m.confirmMTM(ctx, conf.Order)
		}
	} else {
		// we had no trade, but still want to register this position in the settlement
		// engine I guess
		party := conf.Order.Party
		if pos, ok := m.position.GetPositionByPartyID(party); ok {
			m.settlement.AddPosition(party, pos)
		}
	}

	return orderUpdates
}

func (m *Market) confirmMTM(
	ctx context.Context, order *types.Order) (orderUpdates []*types.Order) {
	// now let's get the transfers for MTM settlement
	markPrice := m.getCurrentMarkPrice()
	evts := m.position.UpdateMarkPrice(markPrice)
	settle := m.settlement.SettleMTM(ctx, markPrice, evts)

	// Only process collateral and risk once per order, not for every trade
	margins := m.collateralAndRisk(ctx, settle)
	if len(margins) > 0 {
		transfers, closed, bondPenalties, err := m.collateral.MarginUpdate(ctx, m.GetID(), margins)
		if err == nil && len(transfers) > 0 {
			evt := events.NewTransferResponse(ctx, transfers)
			m.broker.Send(evt)
		}
		if len(bondPenalties) > 0 {
			transfers, err := m.bondSlashing(ctx, bondPenalties...)
			if err != nil {
				m.log.Error("Failed to perform bond slashing",
					logging.Error(err))
			}
			m.broker.Send(events.NewTransferResponse(ctx, transfers))
		}
		if len(closed) > 0 {
			orderUpdates, err = m.resolveClosedOutParties(ctx, closed, order)
			if err != nil {
				m.log.Error("unable to closed out parties",
					logging.String("market-id", m.GetID()),
					logging.Error(err))
			}
		}
		m.updateLiquidityFee(ctx)
	}

	return orderUpdates
}

// updateLiquidityFee computes the current LiquidityProvision fee and updates
// the fee engine.
func (m *Market) updateLiquidityFee(ctx context.Context) {
	stake := m.getTargetStake()
	fee := m.liquidity.ProvisionsPerParty().FeeForTarget(stake)
	if !fee.Equals(m.getLiquidityFee()) {
		m.fee.SetLiquidityFee(fee)
		m.setLiquidityFee(fee)
		m.broker.Send(
			events.NewMarketUpdatedEvent(ctx, *m.mkt),
		)
		m.stateChanged = true
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
func (m *Market) resolveClosedOutParties(ctx context.Context, distressedMarginEvts []events.Margin, o *types.Order) ([]*types.Order, error) {
	if len(distressedMarginEvts) == 0 {
		return nil, nil
	}
	timer := metrics.NewTimeCounter(m.mkt.ID, "market", "resolveClosedOutParties")
	defer timer.EngineTimeCounterAdd()

	// this is going to be run after the the closed out routines
	// are finished, in order to notify the liquidity engine of
	// any changes in the book / orders owned by the lp providers
	orderUpdates := []*types.Order{}
	distressedParties := []string{}
	defer func() {
		// First we check for all distressed parties if they are liquidity
		// providers, and if yea cancel their commitments
		for _, party := range distressedParties {
			if m.liquidity.IsLiquidityProvider(party) {
				if err := m.cancelLiquidityProvision(ctx, party, true); err != nil {
					m.log.Debug("could not cancel liquidity provision",
						logging.MarketID(m.GetID()),
						logging.PartyID(party),
						logging.Error(err))
				}
			}
		}
	}()

	distressedPos := make([]events.MarketPosition, 0, len(distressedMarginEvts))
	for _, v := range distressedMarginEvts {
		if m.log.GetLevel() == logging.DebugLevel {
			m.log.Debug("closing out party",
				logging.PartyID(v.Party()),
				logging.MarketID(m.GetID()))
		}
		distressedPos = append(distressedPos, v)
		distressedParties = append(distressedParties, v.Party())
	}
	// cancel pending orders for parties
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
		o.UpdatedAt = m.currentTime.UnixNano()
		evts = append(evts, events.NewOrderEvent(ctx, o))
		_ = m.position.UnregisterOrder(ctx, o)
	}

	// add the orders remove from the book to the orders
	// to be sent to the liquidity engine
	orderUpdates = append(orderUpdates, rmorders...)

	// now we also remove ALL parked order for the different parties
	for _, v := range distressedPos {
		orders, oevts := m.peggedOrders.RemoveAllParkedForParty(
			ctx, v.Party(), types.OrderStatusStopped)

		for _, v := range orders {
			m.expiringOrders.RemoveOrder(v.ExpiresAt, v.ID)
		}

		// add all pegged orders too to the orderUpdates
		orderUpdates = append(orderUpdates, orders...)
		// add all events to evts list
		evts = append(evts, oevts...)

		if m.liquidity.IsLiquidityProvider(v.Party()) {
			if err := m.confiscateBondAccount(ctx, v.Party()); err != nil {
				m.log.Error("unable to confiscate liquidity provision for a distressed party",
					logging.String("party-id", o.Party),
					logging.String("market-id", mktID),
					logging.Error(err),
				)
			}
		}
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
		okPos, closed = m.risk.ExpectMargins(distressedMarginEvts, m.markPrice.Clone())

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
		return orderUpdates, nil
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
		CreatedAt:   m.currentTime.UnixNano(),
		Reference:   fmt.Sprintf("LS-%s", o.ID), // liquidity sourcing, reference the order which caused the problem
		TimeInForce: types.OrderTimeInForceFOK,  // this is an all-or-nothing order, so TIME_IN_FORCE == FOK
		Type:        types.OrderTypeNetwork,
		Price:       num.Zero(),
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

	// FIXME(j): this is a temporary measure for the case where we do not have enough orders
	// in the book to 0 out the positions.
	// in this case we will just return now, cutting off the position resolution
	// this means that party still being distressed will stay distressed,
	// then when a new order is placed, the distressed parties will go again through positions resolution
	// and if the volume of the book is acceptable, we will then process positions resolutions
	if no.Remaining == no.Size {
		return orderUpdates, ErrNotEnoughVolumeToZeroOutNetworkOrder
	}

	// @NOTE: At this point, the network order was updated by the orderbook
	// the price field now contains the average trade price at which the order was fulfilled
	m.broker.Send(events.NewOrderEvent(ctx, &no))

	m.handleConfirmationPassiveOrders(ctx, confirmation)

	// also add the passive orders from the book into the list
	// of updated orders to send to liquidity engine
	orderUpdates = append(orderUpdates, confirmation.PassiveOrdersAffected...)

	asset, _ := m.mkt.GetAsset()

	// pay the fees now
	fees, distressedPartiesFees := m.fee.CalculateFeeForPositionResolution(
		confirmation.Trades, closedMPs)

	tresps, err := m.collateral.TransferFees(ctx, m.GetID(), asset, fees)
	if err != nil {
		// FIXME(): we may figure a better error handling in here
		m.log.Error("unable to transfer fees for positions resolutions",
			logging.Error(err),
			logging.String("market-id", m.GetID()))
		return orderUpdates, err
	}
	// send transfer to buffer
	m.broker.Send(events.NewTransferResponse(ctx, tresps))

	if len(confirmation.Trades) > 0 {
		// Insert all trades resulted from the executed order
		tradeEvts := make([]events.Event, 0, len(confirmation.Trades))
		for idx, trade := range confirmation.Trades {
			trade.SetIDs(m.idgen.NextID(), &no, confirmation.PassiveOrdersAffected[idx])

			// setup the type of the trade to network
			// this trade did happen with a GOOD trader to
			// 0 out the BAD trader position
			trade.Type = types.TradeTypeNetworkCloseOutGood
			tradeEvts = append(tradeEvts, events.NewTradeEvent(ctx, *trade))

			// Update positions - this is a special trade involving the network as party
			// so rather than checking this every time we call Update, call special UpdateNetwork
			m.position.UpdateNetwork(trade)
			if err := m.tsCalc.RecordOpenInterest(m.position.GetOpenInterest(), m.currentTime); err != nil {
				m.log.Debug("unable record open interest",
					logging.String("market-id", m.GetID()),
					logging.Error(err))
			}

			m.settlement.AddTrade(trade)
			tradeValue, _ := num.UintFromDecimal(num.DecimalFromInt64(int64(trade.Size)).Mul(trade.Price.ToDecimal()).Div(m.positionFactor))
			m.feeSplitter.AddTradeValue(tradeValue)
			m.marketTracker.AddValueTraded(m.mkt.ID, tradeValue)
		}
		m.broker.SendBatch(tradeEvts)
	}

	m.zeroOutNetwork(ctx, closedMPs, &no, o, distressedPartiesFees)

	// swipe all accounts and stuff
	m.finalizePartiesCloseOut(ctx, closed, closedMPs)

	// get the updated positions
	evt := m.position.Positions()

	// settle MTM, the positions have changed
	settle := m.settlement.SettleMTM(ctx, m.markPrice.Clone(), evt)
	// we're not interested in the events here, they're used for margin updates
	// we know the margin requirements will be met, and come the next block
	// margins will automatically be checked anyway

	_, responses, err := m.collateral.MarkToMarket(ctx, m.GetID(), settle, asset)
	if m.log.GetLevel() == logging.DebugLevel {
		m.log.Debug(
			"ledger movements after MTM on parties who closed out distressed",
			logging.Int("response-count", len(responses)),
			logging.String("raw", fmt.Sprintf("%#v", responses)),
		)
	}
	// send transfer to buffer
	m.broker.Send(events.NewTransferResponse(ctx, responses))
	// lastly, recalculate margins for the non-distressed parties
	if err != nil {
		return orderUpdates, err
	}
	// Only check margins if MTM was successful.
	err = m.recheckMargin(ctx, evt)

	return orderUpdates, err
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
	asset, _ := m.mkt.GetAsset()
	// finally remove from collateral (moving funds where needed)
	movements, err := m.collateral.RemoveDistressed(
		ctx, closedMPs, m.GetID(), asset)
	if err != nil {
		m.log.Panic(
			"Failed to remove distressed accounts cleanly",
			logging.Error(err))
	}

	if len(movements.Transfers) > 0 {
		m.broker.Send(
			events.NewTransferResponse(
				ctx, []*types.TransferResponse{movements}),
		)
	}
}

func (m *Market) confiscateBondAccount(ctx context.Context, partyID string) error {
	asset, err := m.mkt.GetAsset()
	if err != nil {
		return err
	}
	bacc, err := m.collateral.GetOrCreatePartyBondAccount(ctx, partyID, m.mkt.ID, asset)
	if err != nil {
		return err
	}

	// we may alreadu have confiscated all funds
	if bacc.Balance.IsZero() {
		return nil
	}

	transfer := &types.Transfer{
		Owner: partyID,
		Amount: &types.FinancialAmount{
			Amount: bacc.Balance, // no need to clone, bacc isn't used after this
			Asset:  asset,
		},
		Type:      types.TransferTypeBondSlashing,
		MinAmount: bacc.Balance.Clone(),
	}
	tresp, err := m.collateral.BondUpdate(ctx, m.mkt.ID, transfer)
	if err != nil {
		return err
	}
	m.broker.Send(events.NewTransferResponse(ctx, []*types.TransferResponse{tresp}))

	return nil
}

func (m *Market) zeroOutNetwork(ctx context.Context, parties []events.MarketPosition, settleOrder, initial *types.Order, fees map[string]*types.Fee) {
	timer := metrics.NewTimeCounter(m.mkt.ID, "market", "zeroOutNetwork")
	defer timer.EngineTimeCounterAdd()

	// ensure an original price is set
	if settleOrder.OriginalPrice == nil {
		settleOrder.OriginalPrice = num.Zero().Div(settleOrder.Price, m.priceFactor)
	}
	marketID := m.GetID()
	order := types.Order{
		MarketID:      marketID,
		Status:        types.OrderStatusFilled,
		Party:         types.NetworkParty,
		Price:         settleOrder.Price.Clone(),
		OriginalPrice: settleOrder.OriginalPrice.Clone(),
		CreatedAt:     m.currentTime.UnixNano(),
		Reference:     "close-out distressed",
		TimeInForce:   types.OrderTimeInForceFOK, // this is an all-or-nothing order, so TIME_IN_FORCE == FOK
		Type:          types.OrderTypeNetwork,
	}

	asset, _ := m.mkt.GetAsset()
	marginLevels := types.MarginLevels{
		MarketID:  m.mkt.GetID(),
		Asset:     asset,
		Timestamp: m.currentTime.UnixNano(),
	}

	tradeEvts := make([]events.Event, 0, len(parties))
	for i, party := range parties {
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
			CreatedAt:     m.currentTime.UnixNano(),
			Reference:     fmt.Sprintf("distressed-%d-%s", i, initial.ID),
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

func (m *Market) recheckMargin(ctx context.Context, pos []events.MarketPosition) error {
	risk, err := m.updateMargin(ctx, pos)
	if err != nil {
		return err
	}
	if len(risk) == 0 {
		return nil
	}
	// now transfer margins, ignore closed because we're only recalculating for non-distressed parties.
	return m.transferRecheckMargins(ctx, risk)
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

func (m *Market) setMarkPrice(trade *types.Trade) {
	// The current mark price calculation is simply the last trade
	// in the future this will use varying logic based on market config
	// the responsibility for calculation could be elsewhere for testability
	m.markPrice = trade.Price.Clone()
	m.stateChanged = true
}

// this function handles moving money after settle MTM + risk margin updates
// but does not move the money between party accounts (ie not to/from margin accounts after risk).
func (m *Market) collateralAndRisk(ctx context.Context, settle []events.Transfer) []events.Risk {
	timer := metrics.NewTimeCounter(m.mkt.ID, "market", "collateralAndRisk")
	defer timer.EngineTimeCounterAdd()
	asset, _ := m.mkt.GetAsset()
	evts, response, err := m.collateral.MarkToMarket(ctx, m.GetID(), settle, asset)
	if err != nil {
		m.log.Error(
			"Failed to process mark to market settlement (collateral)",
			logging.Error(err),
		)
		return nil
	}
	// sending response to buffer
	if response != nil {
		m.broker.Send(events.NewTransferResponse(ctx, response))
	}

	// let risk engine do its thing here - it returns a slice of money that needs
	// to be moved to and from margin accounts
	riskUpdates := m.risk.UpdateMarginsOnSettlement(ctx, evts, m.getCurrentMarkPrice())
	if len(riskUpdates) == 0 {
		return nil
	}

	return riskUpdates
}

func (m *Market) CancelAllOrders(ctx context.Context, partyID string) ([]*types.OrderCancellationConfirmation, error) {
	if !m.canTrade() {
		return nil, ErrTradingNotAllowed
	}

	// get all order for this party in the book
	orders := m.matching.GetOrdersPerParty(partyID)

	// add all orders being eventually parked
	orders = append(orders, m.peggedOrders.GetAllForParty(partyID)...)

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

	// now we extract all liquidity provision order out of the list.
	// cancelling some order may trigger repricing, and repricing
	// liquidity order, which also trigger cancelling...
	// by filtering the list now, we are sure that we will
	// never try to
	// 1. remove a lp order
	// 2. have invalid order referencing lp order which have been canceleld
	okOrders := []*types.Order{}
	for _, order := range orders {
		if m.liquidity.IsLiquidityOrder(partyID, order.ID) {
			continue
		}
		okOrders = append(okOrders, order)
	}

	cancellations := make([]*types.OrderCancellationConfirmation, 0, len(orders))

	// now iterate over all orders and cancel one by one.
	cancelledOrders := make([]*types.Order, 0, len(okOrders))
	for _, order := range okOrders {
		cancellation, err := m.cancelOrder(ctx, partyID, order.ID)
		if err != nil {
			return nil, err
		}
		cancellations = append(cancellations, cancellation)
		cancelledOrders = append(cancelledOrders, cancellation.Order)
	}

	m.checkForReferenceMoves(ctx, cancelledOrders, false)
	m.checkLiquidity(ctx, nil)
	m.commandLiquidityAuction(ctx)

	return cancellations, nil
}

func (m *Market) CancelOrder(ctx context.Context, partyID, orderID string, deterministicId string) (oc *types.OrderCancellationConfirmation, _ error) {
	defer func() {
		if oc != nil {
			if oc.Order.Reference == "jeremy-debug" {
				fmt.Printf("CANCEL ORDER  : %v\n", oc.Order.String())
			}
		}
	}()
	m.idgen = idgeneration.New(deterministicId)
	defer func() { m.idgen = nil }()

	if !m.canTrade() {
		return nil, ErrTradingNotAllowed
	}

	// cancelling and amending an order that is part of the LP commitment isn't allowed
	if m.liquidity.IsLiquidityOrder(partyID, orderID) {
		return nil, types.ErrEditNotAllowed
	}

	conf, err := m.cancelOrder(ctx, partyID, orderID)
	if err != nil {
		return conf, err
	}

	m.checkForReferenceMoves(ctx, []*types.Order{conf.Order}, false)
	m.checkLiquidity(ctx, nil)
	m.commandLiquidityAuction(ctx)

	return conf, nil
}

// CancelOrder cancels the given order.
func (m *Market) cancelOrder(ctx context.Context, partyID, orderID string) (*types.OrderCancellationConfirmation, error) {
	timer := metrics.NewTimeCounter(m.mkt.ID, "market", "CancelOrder")
	defer timer.EngineTimeCounterAdd()

	if m.closed {
		return nil, ErrMarketClosed
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
	order.UpdatedAt = m.currentTime.UnixNano()
	m.broker.Send(events.NewOrderEvent(ctx, order))

	return &types.OrderCancellationConfirmation{Order: order}, nil
}

// parkOrder removes the given order from the orderbook
// parkOrder will panic if it encounters errors, which means that it reached an
// invalid state.
func (m *Market) parkOrder(ctx context.Context, order *types.Order) {
	defer m.releaseMarginExcess(ctx, order.Party)

	if err := m.matching.RemoveOrder(order); err != nil {
		m.log.Panic("Failure to remove order from matching engine",
			logging.Order(*order),
			logging.Error(err))
	}

	m.peggedOrders.Park(order)
	m.broker.Send(events.NewOrderEvent(ctx, order))
	_ = m.position.UnregisterOrder(ctx, order)
}

// AmendOrder amend an existing order from the order book.
func (m *Market) AmendOrder(ctx context.Context, orderAmendment *types.OrderAmendment, party string,
	deterministicId string) (oc *types.OrderConfirmation, _ error,
) {
	defer func() {
		if oc != nil {
			party := ""
			if oc.Order.Reference == "jeremy-debug" {
				fmt.Printf("AMEND ORDER  : %v\n", oc.Order.String())
				party = oc.Order.Party
			}
			for _, v := range oc.PassiveOrdersAffected {
				if v.Reference == "jeremy-debug" {
					fmt.Printf("AMEND PASSIVE: %v\n", v.String())
					party = v.Party
				}
			}

			if party != "" {
				for _, v := range oc.Trades {
					if v.Buyer == "jeremy-debug" || v.SellOrder == "jeremy-debug" {
						fmt.Printf("AMEND TRADE: %v\n", v.String())
					}
				}
			}
		}
	}()

	m.idgen = idgeneration.New(deterministicId)
	defer func() { m.idgen = nil }()

	if !m.canTrade() {
		return nil, ErrTradingNotAllowed
	}

	// explicitly/directly ordering an LP commitment order is not allowed
	if m.liquidity.IsLiquidityOrder(party, orderAmendment.OrderID) {
		return nil, types.ErrEditNotAllowed
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
	m.checkForReferenceMoves(ctx, allUpdatedOrders, false)
	m.checkLiquidity(ctx, nil)
	m.commandLiquidityAuction(ctx)

	return conf, nil
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
		return nil, nil, ErrMarketClosed
	}

	// Try and locate the existing order specified on the
	// order book in the matching engine for this market
	existingOrder, foundOnBook, err := m.getOrderByID(orderAmendment.OrderID)
	if err != nil {
		if m.log.GetLevel() == logging.DebugLevel {
			m.log.Debug("Invalid order ID",
				logging.OrderID(orderAmendment.GetOrderId()),
				logging.PartyID(party),
				logging.MarketID(orderAmendment.GetMarketId()),
				logging.Error(err))
		}
		return nil, nil, types.ErrInvalidOrderID
	}

	// We can only amend this order if we created it
	if existingOrder.Party != party {
		if m.log.GetLevel() == logging.DebugLevel {
			m.log.Debug("Invalid party ID",
				logging.String("original party id:", existingOrder.Party),
				logging.PartyID(party))
		}
		return nil, nil, types.ErrInvalidPartyID
	}

	// Validate Market
	if existingOrder.MarketID != m.mkt.ID {
		if m.log.GetLevel() == logging.DebugLevel {
			m.log.Debug("Market ID mismatch",
				logging.MarketID(m.mkt.ID),
				logging.Order(*existingOrder))
		}
		return nil, nil, types.ErrInvalidMarketID
	}

	if err := m.validateOrderAmendment(existingOrder, orderAmendment); err != nil {
		return nil, nil, err
	}

	amendedOrder, err := m.applyOrderAmendment(existingOrder, orderAmendment)
	if err != nil {
		return nil, nil, err
	}

	// If we have a pegged order that is no longer expiring, we need to remove it
	var (
		needToRemoveExpiry       = false
		needToAddExpiry          = false
		expiresAt          int64 = 0
	)
	defer func() {
		// no errors, amend most likely happened properly
		if returnedErr == nil {
			if needToRemoveExpiry {
				m.expiringOrders.RemoveOrder(expiresAt, existingOrder.ID)
			}
			if needToAddExpiry {
				m.expiringOrders.Insert(amendedOrder.ID, amendedOrder.ExpiresAt)
			}
		}
	}()

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

	// if we are amending from GTT to GTC, flag ready to remove from expiry list
	if existingOrder.IsExpireable() &&
		!amendedOrder.IsExpireable() {
		// We no longer need to handle the expiry
		needToRemoveExpiry = true
		expiresAt = existingOrder.ExpiresAt
	}

	// if we are amending from GTC to GTT, flag ready to add to expiry list
	if !existingOrder.IsExpireable() &&
		amendedOrder.IsExpireable() {
		// We need to handle the expiry
		needToAddExpiry = true
	}

	// if both where expireable but we changed the duration
	// then we need to remove, then reinsert...
	if existingOrder.IsExpireable() &&
		amendedOrder.IsExpireable() &&
		existingOrder.ExpiresAt != amendedOrder.ExpiresAt {
		// We no longer need to handle the expiry
		needToRemoveExpiry = true
		needToAddExpiry = true
		expiresAt = existingOrder.ExpiresAt
	}

	// if expiration has changed and is before the original creation time, reject this amend
	if amendedOrder.ExpiresAt != 0 && amendedOrder.ExpiresAt < existingOrder.CreatedAt {
		if m.log.GetLevel() == logging.DebugLevel {
			m.log.Debug("Amended expiry before original creation time",
				logging.Int64("original order created at ts:", existingOrder.CreatedAt),
				logging.Int64("amended expiry ts:", amendedOrder.ExpiresAt),
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
			m.orderAmendInPlace(existingOrder, amendedOrder)
			cancellation, err := m.matching.CancelOrder(amendedOrder)
			if cancellation == nil || err != nil {
				m.log.Panic("Failure to cancel order from matching engine",
					logging.String("party-id", amendedOrder.Party),
					logging.String("order-id", amendedOrder.ID),
					logging.String("market", m.mkt.ID),
					logging.Error(err))
				return nil, nil, err
			}

			_ = m.position.UnregisterOrder(ctx, cancellation.Order)
			amendedOrder = cancellation.Order
		}

		// Update the order in our stores (will be marked as cancelled)
		// set the proper status
		amendedOrder.Status = types.OrderStatusExpired
		m.broker.Send(events.NewOrderEvent(ctx, amendedOrder))

		m.removePeggedOrder(amendedOrder)

		// m.checkForReferenceMoves(ctx, []*types.Order{}, false)

		return &types.OrderConfirmation{
			Order: amendedOrder,
		}, nil, nil
	}

	// TODO: This can be simplified by:
	// - amending the order in the peggedList first
	// - applying the changed based on auction / repricing
	if existingOrder.PeggedOrder != nil {
		// Amend in place during an auction
		if m.as.InAuction() {
			ret := m.orderAmendWhenParked(existingOrder, amendedOrder)
			m.broker.Send(events.NewOrderEvent(ctx, amendedOrder))
			return ret, nil, nil
		}
		err := m.repricePeggedOrder(amendedOrder)
		if err != nil {
			// Failed to reprice so we have to park the order
			if amendedOrder.Status != types.OrderStatusParked {
				// If we are live then park
				m.parkOrder(ctx, existingOrder)
			}
			ret := m.orderAmendWhenParked(existingOrder, amendedOrder)
			m.broker.Send(events.NewOrderEvent(ctx, amendedOrder))
			return ret, nil, nil
		} else {
			// We got a new valid price, if we are parked we need to unpark
			if amendedOrder.Status == types.OrderStatusParked {
				orderConf, orderUpdts, err := m.submitValidatedOrder(ctx, amendedOrder)
				if err != nil {
					// If we cannot submit a new order then the amend has failed, return the error
					return nil, orderUpdts, err
				}
				// Update pegged order with new amended version
				m.peggedOrders.Amend(amendedOrder)
				return orderConf, orderUpdts, err
			}
		}
	}

	// from here these are the normal amendment
	var priceIncrease, priceShift, sizeIncrease, sizeDecrease, expiryChange, timeInForceChange bool

	if amendedOrder.Price.NEQ(existingOrder.Price) {
		priceShift = true
		priceIncrease = existingOrder.Price.LT(amendedOrder.Price)
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
			// m.checkForReferenceMoves(ctx, []*types.Order{}, false)
		}
		return ret, nil, err
	}

	// Update potential new position after the amend
	pos := m.position.AmendOrder(ctx, existingOrder, amendedOrder)

	// Perform check and allocate margin if price or order size is increased
	// ignore rollback return here, as if we amend it means the order
	// is already on the book, not rollback will be needed, the margin
	// will be updated later on for sure.

	if priceIncrease || sizeIncrease {
		if err = m.checkMarginForOrder(ctx, pos, amendedOrder); err != nil {
			// Undo the position registering
			_ = m.position.AmendOrder(ctx, amendedOrder, existingOrder)

			if m.log.GetLevel() == logging.DebugLevel {
				m.log.Debug("Unable to check/add margin for party",
					logging.String("market-id", m.GetID()),
					logging.Error(err))
			}
			return nil, nil, ErrMarginCheckFailed
		}
	}

	// if increase in size or change in price
	// ---> DO atomic cancel and submit
	if priceShift || sizeIncrease {
		confirmation, err := m.orderCancelReplace(ctx, existingOrder, amendedOrder)
		var orders []*types.Order
		if err != nil {
			// if an error happen, the order never hit the book, so we can
			// just rollback the position size
			_ = m.position.AmendOrder(ctx, amendedOrder, existingOrder)

		} else {
			orders = m.handleConfirmation(ctx, confirmation)
			m.broker.Send(events.NewOrderEvent(ctx, confirmation.Order))
		}
		return confirmation, orders, err
	}

	// if decrease in size or change in expiration date
	// ---> DO amend in place in matching engine
	if expiryChange || sizeDecrease || timeInForceChange {
		if sizeDecrease && amendedOrder.Remaining >= existingOrder.Remaining {
			_ = m.position.AmendOrder(ctx, amendedOrder, existingOrder)

			if m.log.GetLevel() == logging.DebugLevel {
				m.log.Debug("Order amendment not allowed when reducing to a larger amount", logging.Order(*existingOrder))
			}
			return nil, nil, ErrInvalidAmendRemainQuantity
		}
		// we not doing anything in case of error here as its
		// pretty much impossible at this point for an order not to be
		// amended in place. Maybe a panic would be better
		ret, err := m.orderAmendInPlace(existingOrder, amendedOrder)
		if err == nil {
			m.broker.Send(events.NewOrderEvent(ctx, amendedOrder))
			// m.checkForReferenceMoves(ctx, []*types.Order{}, false)
		}
		return ret, nil, err
	}

	if m.log.GetLevel() == logging.DebugLevel {
		m.log.Debug("Order amendment not allowed", logging.Order(*existingOrder))
	}
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

// this function assume the amendment have been validated before.
func (m *Market) applyOrderAmendment(
	existingOrder *types.Order,
	amendment *types.OrderAmendment,
) (order *types.Order, err error) {
	order = existingOrder.Clone()
	order.UpdatedAt = m.currentTime.UnixNano()
	order.Version += 1

	if existingOrder.PeggedOrder != nil {
		order.PeggedOrder = &types.PeggedOrder{
			Reference: existingOrder.PeggedOrder.Reference,
			Offset:    existingOrder.PeggedOrder.Offset,
		}
	}

	var amendPrice *num.Uint
	if amendment.Price != nil {
		amendPrice = amendment.Price.Clone()
		amendPrice.Mul(amendPrice, m.priceFactor)
	}
	// apply price changes
	if amendment.Price != nil && existingOrder.Price.NEQ(amendPrice) {
		order.Price = amendPrice.Clone()
		order.OriginalPrice = amendment.Price.Clone()
	}

	// apply size changes
	if amendment.SizeDelta != 0 {
		if amendment.SizeDelta > 0 {
			order.Size += uint64(amendment.SizeDelta)
		} else {
			order.Size -= uint64(-amendment.SizeDelta)
		}

		newRemaining := int64(existingOrder.Remaining) + amendment.SizeDelta
		if newRemaining <= 0 {
			newRemaining = 0
		}
		order.Remaining = uint64(newRemaining)
	}

	// apply tif
	if amendment.TimeInForce != types.OrderTimeInForceUnspecified {
		order.TimeInForce = amendment.TimeInForce
		if amendment.TimeInForce != types.OrderTimeInForceGTT {
			order.ExpiresAt = 0
		}
	}
	if amendment.ExpiresAt != nil {
		order.ExpiresAt = *amendment.ExpiresAt
	}

	// apply pegged order values
	if order.PeggedOrder != nil {
		if amendment.PeggedOffset != nil {
			order.PeggedOrder.Offset = amendment.PeggedOffset.Clone()
		}

		if amendment.PeggedReference != types.PeggedReferenceUnspecified {
			order.PeggedOrder.Reference = amendment.PeggedReference
		}
		if verr := m.validatePeggedOrder(order); verr != types.OrderErrorUnspecified {
			err = verr
		}
	}

	return order, err
}

func (m *Market) orderCancelReplace(ctx context.Context, existingOrder, newOrder *types.Order) (conf *types.OrderConfirmation, err error) {
	timer := metrics.NewTimeCounter(m.mkt.ID, "market", "orderCancelReplace")

	// make sure the order is on the book, this was done by canceling the order initially, but that could
	// trigger an auction in some cases.
	if o, err := m.matching.GetOrderByID(existingOrder.ID); err != nil || o == nil {
		m.log.Panic("Can't CancelReplace, the original order was not found",
			logging.OrderWithTag(*existingOrder, "existing-order"),
			logging.Error(err))
	}
	// first we call the order book to evaluate auction triggers and get the list of trades
	trades, err := m.checkPriceAndGetTrades(ctx, newOrder)
	if err != nil {
		return nil, errors.New("couldn't insert order in book")
	}

	// try to apply fees on the trade
	if err := m.applyFees(ctx, newOrder, trades); err != nil {
		return nil, errors.New("could not apply fees for order")
	}

	// "hot-swap" of the orders
	conf, err = m.matching.ReplaceOrder(existingOrder, newOrder)
	if err != nil {
		m.log.Panic("unable to submit order", logging.Error(err))
	}
	// replace the trades in the confirmation to have
	// the ones with the fees embedded
	conf.Trades = trades

	timer.EngineTimeCounterAdd()

	return conf, nil
}

func (m *Market) orderAmendInPlace(originalOrder, amendOrder *types.Order) (*types.OrderConfirmation, error) {
	timer := metrics.NewTimeCounter(m.mkt.ID, "market", "orderAmendInPlace")
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

func (m *Market) orderAmendWhenParked(originalOrder, amendOrder *types.Order) *types.OrderConfirmation {
	amendOrder.Status = types.OrderStatusParked
	amendOrder.Price = num.Zero()
	amendOrder.OriginalPrice = num.Zero()
	*originalOrder = *amendOrder

	return &types.OrderConfirmation{
		Order: amendOrder,
	}
}

// RemoveExpiredOrders remove all expired orders from the order book
// and also any pegged orders that are parked.
func (m *Market) RemoveExpiredOrders(
	ctx context.Context, timestamp int64) ([]*types.Order, error) {
	timer := metrics.NewTimeCounter(m.mkt.ID, "market", "RemoveExpiredOrders")
	defer timer.EngineTimeCounterAdd()

	if m.closed {
		return nil, ErrMarketClosed
	}

	expired := []*types.Order{}
	evts := []events.Event{}
	for _, orderID := range m.expiringOrders.Expire(timestamp) {
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
		order.UpdatedAt = m.currentTime.UnixNano()
		order.Status = types.OrderStatusExpired
		expired = append(expired, order)
		evts = append(evts, events.NewOrderEvent(ctx, order))
	}
	m.broker.SendBatch(evts)

	// If we have removed an expired order, do we need to reprice any
	// or maybe notify the liquidity engine
	if len(expired) > 0 {
		m.checkForReferenceMoves(ctx, expired, false)
		m.checkLiquidity(ctx, nil)
		m.commandLiquidityAuction(ctx)
	}

	return expired, nil
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
		return num.Zero(), err
	}
	ask, err := m.matching.GetBestStaticAskPrice()
	if err != nil {
		return num.Zero(), err
	}
	mid := num.Zero()
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
	m.peggedOrders.Remove(order)
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
	if o := m.peggedOrders.GetByID(orderID); o != nil {
		return o, false, nil
	}

	// We couldn't find it
	return nil, false, ErrOrderNotFound
}

func (m *Market) getRiskFactors() (*types.RiskFactor, error) {
	rf, err := m.risk.GetRiskFactors()
	if err != nil {
		return nil, err
	}
	return rf, nil
}

func (m *Market) getTheoreticalTargetStake() *num.Uint {
	rf, err := m.getRiskFactors()
	if err != nil {
		logging.Error(err)
		m.log.Debug("unable to get risk factors, can't calculate target")
		return num.Zero()
	}
	return m.tsCalc.GetTheoreticalTargetStake(
		*rf, m.currentTime, m.getCurrentMarkPrice(), nil)
}

func (m *Market) getTargetStake() *num.Uint {
	rf, err := m.getRiskFactors()
	if err != nil {
		logging.Error(err)
		m.log.Debug("unable to get risk factors, can't calculate target")
		return num.Zero()
	}
	return m.tsCalc.GetTargetStake(*rf, m.currentTime, m.getCurrentMarkPrice())
}

func (m *Market) getSuppliedStake() *num.Uint {
	return m.liquidity.CalculateSuppliedStake()
}

func (m *Market) checkLiquidity(ctx context.Context, trades []*types.Trade) {
	// before we check liquidity, ensure we've moved all funds that can go towards
	// provided stake to the bond accounts so we don't trigger liquidity auction for no reason
	m.checkBondBalance(ctx)
	_, vBid, _ := m.getBestStaticBidPriceAndVolume()
	_, vAsk, _ := m.getBestStaticAskPriceAndVolume()

	rf, err := m.getRiskFactors()
	if err != nil {
		m.log.Panic("unable to get risk factors, can't check liquidity",
			logging.String("market-id", m.GetID()),
			logging.Error(err))
	}

	m.lMonitor.CheckLiquidity(
		m.as, m.currentTime,
		m.getSuppliedStake(),
		trades,
		*rf,
		m.getCurrentMarkPrice(),
		vBid, vAsk)
	if evt := m.as.AuctionExtended(ctx, m.currentTime); evt != nil {
		m.broker.Send(evt)
	}
}

// command liquidity auction checks if liquidity auction should be entered and if it can end.
func (m *Market) commandLiquidityAuction(ctx context.Context) {
	// start the liquidity monitoring auction if required
	if !m.as.InAuction() && m.as.AuctionStart() {
		m.enterAuction(ctx)
	}
	// end the liquidity monitoring auction if possible
	if m.as.InAuction() && m.as.CanLeave() && !m.as.IsOpeningAuction() {
		p, v, _ := m.matching.GetIndicativePriceAndVolume()
		// no need to clone here, we're getting indicative price once for this call
		if err := m.pMonitor.CheckPrice(ctx, m.as, p, v, m.currentTime, true); err != nil {
			m.log.Panic("unable to run check price with price monitor",
				logging.String("market-id", m.GetID()),
				logging.Error(err))
		}
		// TODO: Need to also get indicative trades and check how they'd impact target stake,
		// see  https://github.com/vegaprotocol/vega/issues/3047
		// If price monitoring doesn't trigger auction than leave it
		if evt := m.as.AuctionExtended(ctx, m.currentTime); evt != nil {
			m.broker.Send(evt)
		}
	}
}

func (m *Market) tradingTerminated(ctx context.Context, tt bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.mkt.State = types.MarketStateTradingTerminated
	m.broker.Send(events.NewMarketUpdatedEvent(ctx, *m.mkt))

	sp, _ := m.tradableInstrument.Instrument.Product.SettlementPrice()
	if sp != nil {
		m.settlementPriceWithLock(ctx, sp)
	}
	m.stateChanged = true
}

func (m *Market) settlementPrice(ctx context.Context, settlementPrice *num.Uint) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.settlementPriceWithLock(ctx, settlementPrice)
}

// NB this musy be called with the lock already acquired.
func (m *Market) settlementPriceWithLock(ctx context.Context, settlementPrice *num.Uint) {
	if m.closed {
		return
	}
	if m.mkt.State == types.MarketStateTradingTerminated && settlementPrice != nil {
		m.closeMarket(ctx, m.currentTime)
		m.closed = m.mkt.State == types.MarketStateSettled
	}
}

func (m *Market) canTrade() bool {
	return m.mkt.State == types.MarketStateActive ||
		m.mkt.State == types.MarketStatePending ||
		m.mkt.State == types.MarketStateSuspended
}

func (m *Market) canSubmitCommitment() bool {
	return m.canTrade() || m.mkt.State == types.MarketStateProposed
}

// cleanupOnReject remove all resources created while the
// market was on PREPARED state.
// we'll need to remove all accounts related to the market
// all margin accounts for this market
// all bond accounts for this market too.
// at this point no fees would have been collected or anything
// like this.
func (m *Market) cleanupOnReject(ctx context.Context) {
	// get the list of all parties in this market
	parties := make([]string, 0, len(m.parties))
	for k := range m.parties {
		parties = append(parties, k)
	}

	err := m.stopAllLiquidityProvisionOnReject(ctx)
	if err != nil {
		m.log.Debug("could not stop all liquidity provision on market rejection",
			logging.MarketID(m.GetID()),
			logging.Error(err))
	}

	asset, _ := m.mkt.GetAsset()
	tresps, err := m.collateral.ClearMarket(ctx, m.GetID(), asset, parties)
	if err != nil {
		m.log.Panic("unable to cleanup a rejected market",
			logging.String("market-id", m.GetID()),
			logging.Error(err))
		return
	}

	m.matching.StopSnapshots()
	m.position.StopSnapshots()
	m.liquidity.StopSnapshots()
	m.tsCalc.StopSnapshots()

	// then send the responses
	m.broker.Send(events.NewTransferResponse(ctx, tresps))
}

func (m *Market) stopAllLiquidityProvisionOnReject(ctx context.Context) error {
	for party := range m.liquidity.ProvisionsPerParty() {
		// here we ignore  the list of orders that could have been
		// created with this party liquidity provision. At this point
		// if we are calling this function, the market is in a PENDING
		// state, which means that liquidity provision can be submitted
		// but orders would never be able to be deployed, so it's safe
		// to ignorethe second return as it shall be an empty slice.
		_, err := m.liquidity.StopLiquidityProvision(ctx, party)
		if err != nil {
			return err
		}
	}

	return nil
}

func lpsToLiquidityProviderFeeShare(lps map[string]*lp) []*types.LiquidityProviderFeeShare {
	out := make([]*types.LiquidityProviderFeeShare, 0, len(lps))
	for k, v := range lps {
		out = append(out, &types.LiquidityProviderFeeShare{
			Party:                 k,
			EquityLikeShare:       v.share.String(),
			AverageEntryValuation: v.avg.String(),
		})
	}

	// sort then so we produce the same output on all nodes
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Party < out[j].Party
	})

	return out
}

func (m *Market) distributeLiquidityFees(ctx context.Context) error {
	asset, err := m.mkt.GetAsset()
	if err != nil {
		return fmt.Errorf("failed to get asset: %w", err)
	}

	acc, err := m.collateral.GetMarketLiquidityFeeAccount(m.mkt.GetID(), asset)
	if err != nil {
		return fmt.Errorf("failed to get market liquidity fee account: %w", err)
	}

	// We can't distribute any share when no balance.
	if acc.Balance.IsZero() {
		return nil
	}

	shares := m.equityShares.SharesExcept(m.liquidity.GetInactiveParties())
	if len(shares) == 0 {
		return nil
	}

	feeTransfer := m.fee.BuildLiquidityFeeDistributionTransfer(shares, acc)
	if feeTransfer == nil {
		return nil
	}

	m.feesTracker.UpdateFeesFromTransfers(feeTransfer.Transfers())
	resp, err := m.collateral.TransferFees(ctx, m.GetID(), asset, feeTransfer)
	if err != nil {
		return fmt.Errorf("failed to transfer fees: %w", err)
	}

	m.broker.Send(events.NewTransferResponse(ctx, resp))
	return nil
}
