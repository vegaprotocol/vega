//lint:file-ignore ST1003 Ignore underscores in names, this is straigh copied from the proto package to ease introducing the domain types

package types

import (
	"errors"

	proto "code.vegaprotocol.io/protos/vega"
	v1 "code.vegaprotocol.io/protos/vega/oracles/v1"
	"code.vegaprotocol.io/vega/types/num"
)

type LiquidityProviderFeeShare = proto.LiquidityProviderFeeShare

type MarketTradingConfigType int

const (
	MarketTradingConfigUndefined MarketTradingConfigType = iota
	MarketTradingConfigContinuous
	MarketTradingConfigDiscrete
)

var (
	ErrNilTradableInstrument = errors.New("nil tradable instrument")
	ErrNilInstrument         = errors.New("nil instrument")
	ErrNilProduct            = errors.New("nil product")
	ErrUnknownAsset          = errors.New("unknown asset")
)

type MarketTimestamps struct {
	Proposed int64
	Pending  int64
	Open     int64
	Close    int64
}

func MarketTimestampsFromProto(p *proto.MarketTimestamps) *MarketTimestamps {
	var ts MarketTimestamps
	if p != nil {
		ts = MarketTimestamps{
			Proposed: p.Proposed,
			Pending:  p.Pending,
			Open:     p.Open,
			Close:    p.Close,
		}
	}
	return &ts
}

func (m MarketTimestamps) IntoProto() *proto.MarketTimestamps {
	return &proto.MarketTimestamps{
		Proposed: m.Proposed,
		Pending:  m.Pending,
		Open:     m.Open,
		Close:    m.Close,
	}
}

type MarketTradingMode = proto.Market_TradingMode

const (
	// Default value, this is invalid
	MarketTradingModeUnspecified MarketTradingMode = proto.Market_TRADING_MODE_UNSPECIFIED
	// Normal trading
	MarketTradingModeContinuous MarketTradingMode = proto.Market_TRADING_MODE_CONTINUOUS
	// Auction trading (FBA)
	MarketTradingModeBatchAuction MarketTradingMode = proto.Market_TRADING_MODE_BATCH_AUCTION
	// Opening auction
	MarketTradingModeOpeningAuction MarketTradingMode = proto.Market_TRADING_MODE_OPENING_AUCTION
	// Auction triggered by monitoring
	MarketTradingModeMonitoringAuction MarketTradingMode = proto.Market_TRADING_MODE_MONITORING_AUCTION
)

type MarketState = proto.Market_State

const (
	// Default value, invalid
	MarketStateUnspecified MarketState = proto.Market_STATE_UNSPECIFIED
	// The Governance proposal valid and accepted
	MarketStateProposed MarketState = proto.Market_STATE_PROPOSED
	// Outcome of governance votes is to reject the market
	MarketStateRejected MarketState = proto.Market_STATE_REJECTED
	// Governance vote passes/wins
	MarketStatePending MarketState = proto.Market_STATE_PENDING
	// Market triggers cancellation condition or governance
	// votes to close before market becomes Active
	MarketStateCancelled MarketState = proto.Market_STATE_CANCELLED
	// Enactment date reached and usual auction exit checks pass
	MarketStateActive MarketState = proto.Market_STATE_ACTIVE
	// Price monitoring or liquidity monitoring trigger
	MarketStateSuspended MarketState = proto.Market_STATE_SUSPENDED
	// Governance vote (to close)
	MarketStateClosed MarketState = proto.Market_STATE_CLOSED
	// Defined by the product (i.e. from a product parameter,
	// specified in market definition, giving close date/time)
	MarketStateTradingTerminated MarketState = proto.Market_STATE_TRADING_TERMINATED
	// Settlement triggered and completed as defined by product
	MarketStateSettled MarketState = proto.Market_STATE_SETTLED
)

type AuctionTrigger = proto.AuctionTrigger

const (
	// Default value for AuctionTrigger, no auction triggered
	AuctionTriggerUnspecified AuctionTrigger = proto.AuctionTrigger_AUCTION_TRIGGER_UNSPECIFIED
	// Batch auction
	AuctionTriggerBatch AuctionTrigger = proto.AuctionTrigger_AUCTION_TRIGGER_BATCH
	// Opening auction
	AuctionTriggerOpening AuctionTrigger = proto.AuctionTrigger_AUCTION_TRIGGER_OPENING
	// Price monitoring trigger
	AuctionTriggerPrice AuctionTrigger = proto.AuctionTrigger_AUCTION_TRIGGER_PRICE
	// Liquidity monitoring trigger
	AuctionTriggerLiquidity AuctionTrigger = proto.AuctionTrigger_AUCTION_TRIGGER_LIQUIDITY
)

type InstrumentMetadata struct {
	Tags []string
}

func InstrumentMetadataFromProto(m *proto.InstrumentMetadata) *InstrumentMetadata {
	return &InstrumentMetadata{
		Tags: append([]string{}, m.Tags...),
	}
}

func (i InstrumentMetadata) IntoProto() *proto.InstrumentMetadata {
	tags := make([]string, 0, len(i.Tags))
	return &proto.InstrumentMetadata{
		Tags: append(tags, i.Tags...),
	}
}

func (i InstrumentMetadata) String() string {
	return i.IntoProto().String()
}

type Timestamp struct {
	Value int64
}

type Price struct {
	Value *num.Uint
}

type AuctionDuration struct {
	Duration int64
	Volume   uint64
}

func AuctionDurationFromProto(ad *proto.AuctionDuration) *AuctionDuration {
	if ad == nil {
		return nil
	}
	return &AuctionDuration{
		Duration: ad.Duration,
		Volume:   ad.Volume,
	}
}

func (a AuctionDuration) IntoProto() *proto.AuctionDuration {
	return &proto.AuctionDuration{
		Duration: a.Duration,
		Volume:   a.Volume,
	}
}

func (a AuctionDuration) String() string {
	return a.IntoProto().String()
}

func (p Price) IntoProto() *proto.Price {
	return &proto.Price{
		Value: num.UintToString(p.Value),
	}
}

func (p Price) String() string {
	return p.IntoProto().String()
}

func (t Timestamp) IntoProto() *proto.Timestamp {
	return &proto.Timestamp{
		Value: t.Value,
	}
}

func (t Timestamp) String() string {
	return t.IntoProto().String()
}

type rmType int

const (
	SIMPLE_RISK_MODEL rmType = iota
	LOGNORMAL_RISK_MODEL
)

type TradableInstrument struct {
	Instrument       *Instrument
	MarginCalculator *MarginCalculator
	RiskModel        isTRM
	rmt              rmType
}

type isTRM interface {
	isTRM()
	trmIntoProto() interface{}
	rmType() rmType
}

func TradableInstrumentFromProto(ti *proto.TradableInstrument) *TradableInstrument {
	if ti == nil {
		return nil
	}
	rm := isTRMFromProto(ti.RiskModel)
	return &TradableInstrument{
		Instrument:       InstrumentFromProto(ti.Instrument),
		MarginCalculator: MarginCalculatorFromProto(ti.MarginCalculator),
		RiskModel:        rm,
		rmt:              rm.rmType(),
	}
}

func (t TradableInstrument) IntoProto() *proto.TradableInstrument {
	var (
		i *proto.Instrument
		m *proto.MarginCalculator
	)
	if t.Instrument != nil {
		i = t.Instrument.IntoProto()
	}
	if t.MarginCalculator != nil {
		m = t.MarginCalculator.IntoProto()
	}
	r := &proto.TradableInstrument{
		Instrument:       i,
		MarginCalculator: m,
	}
	if t.RiskModel == nil {
		return r
	}
	rmp := t.RiskModel.trmIntoProto()
	switch rm := rmp.(type) {
	case *proto.TradableInstrument_SimpleRiskModel:
		r.RiskModel = rm
	case *proto.TradableInstrument_LogNormalRiskModel:
		r.RiskModel = rm
	}
	return r
}

func (t TradableInstrument) GetSimpleRiskModel() *SimpleRiskModel {
	if t.rmt == SIMPLE_RISK_MODEL {
		srm, ok := t.RiskModel.(*TradableInstrumentSimpleRiskModel)
		if !ok || srm == nil {
			return nil
		}
		return srm.SimpleRiskModel
	}
	return nil
}

func (t TradableInstrument) GetLogNormalRiskModel() *LogNormalRiskModel {
	if t.rmt == LOGNORMAL_RISK_MODEL {
		lrm, ok := t.RiskModel.(*TradableInstrumentLogNormalRiskModel)
		if !ok || lrm == nil {
			return nil
		}
		return lrm.LogNormalRiskModel
	}
	return nil
}

func (t TradableInstrument) String() string {
	return t.IntoProto().String()
}

type MarketDiscrete struct {
	Discrete *DiscreteTrading
}

func (m MarketDiscrete) IntoProto() *proto.Market_Discrete {
	return &proto.Market_Discrete{
		Discrete: m.Discrete.IntoProto(),
	}
}

func (MarketDiscrete) istmc() {}

func (m MarketDiscrete) tmcIntoProto() interface{} {
	return m.IntoProto()
}

func MarketDiscreteFromProto(m *proto.Market_Discrete) *MarketDiscrete {
	return &MarketDiscrete{
		Discrete: DiscreteTradingFromProto(m.Discrete),
	}
}

func (MarketDiscrete) tmcType() MarketTradingConfigType {
	return MarketTradingConfigDiscrete
}

type MarketContinuous struct {
	Continuous *ContinuousTrading
}

func MarketContinuousFromProto(c *proto.Market_Continuous) *MarketContinuous {
	return &MarketContinuous{
		Continuous: ContinuousTradingFromProto(c.Continuous),
	}
}

func (m MarketContinuous) IntoProto() *proto.Market_Continuous {
	return &proto.Market_Continuous{
		Continuous: m.Continuous.IntoProto(),
	}
}

func (MarketContinuous) tmcType() MarketTradingConfigType {
	return MarketTradingConfigContinuous
}

func (MarketContinuous) istmc() {}

func (m MarketContinuous) tmcIntoProto() interface{} {
	return m.IntoProto()
}

func tmcFromProto(tm interface{}) istmc {
	switch tmc := tm.(type) {
	case *proto.Market_Continuous:
		return MarketContinuousFromProto(tmc)
	case *proto.Market_Discrete:
		return MarketDiscreteFromProto(tmc)
	}
	return nil
}

type Instrument_Future struct {
	Future *Future
}

type Future struct {
	Maturity                        string
	SettlementAsset                 string
	QuoteName                       string
	OracleSpecForSettlementPrice    *v1.OracleSpec
	OracleSpecForTradingTermination *v1.OracleSpec
	OracleSpecBinding               *OracleSpecToFutureBinding
}

func FutureFromProto(f *proto.Future) *Future {
	return &Future{
		Maturity:                        f.Maturity,
		SettlementAsset:                 f.SettlementAsset,
		QuoteName:                       f.QuoteName,
		OracleSpecForSettlementPrice:    f.OracleSpecForSettlementPrice.DeepClone(),
		OracleSpecForTradingTermination: f.OracleSpecForTradingTermination.DeepClone(),
		OracleSpecBinding:               OracleSpecToFutureBindingFromProto(f.OracleSpecBinding),
	}
}

func (f Future) IntoProto() *proto.Future {
	return &proto.Future{
		Maturity:                        f.Maturity,
		SettlementAsset:                 f.SettlementAsset,
		QuoteName:                       f.QuoteName,
		OracleSpecForSettlementPrice:    f.OracleSpecForSettlementPrice.DeepClone(),
		OracleSpecForTradingTermination: f.OracleSpecForTradingTermination.DeepClone(),
		OracleSpecBinding:               f.OracleSpecBinding.IntoProto(),
	}
}

func iInstrumentFromProto(pi interface{}) iProto {
	switch i := pi.(type) {
	case proto.Instrument_Future:
		return InstrumentFutureFromProto(&i)
	case *proto.Instrument_Future:
		return InstrumentFutureFromProto(i)
	}
	return nil
}

func InstrumentFutureFromProto(f *proto.Instrument_Future) *Instrument_Future {
	return &Instrument_Future{
		Future: FutureFromProto(f.Future),
	}
}

func (i Instrument_Future) IntoProto() *proto.Instrument_Future {
	return &proto.Instrument_Future{
		Future: i.Future.IntoProto(),
	}
}

func (i Instrument_Future) getAsset() (string, error) {
	if i.Future == nil {
		return "", ErrUnknownAsset
	}
	return i.Future.SettlementAsset, nil
}

func (i Instrument_Future) iIntoProto() interface{} {
	return i.IntoProto()
}

type iProto interface {
	iIntoProto() interface{}
	getAsset() (string, error)
}

type Instrument struct {
	ID       string
	Code     string
	Name     string
	Metadata *InstrumentMetadata
	// Types that are valid to be assigned to Product:
	//	*Instrument_Future
	Product iProto
}

func InstrumentFromProto(i *proto.Instrument) *Instrument {
	if i == nil {
		return nil
	}
	return &Instrument{
		ID:       i.Id,
		Code:     i.Code,
		Name:     i.Name,
		Metadata: InstrumentMetadataFromProto(i.Metadata),
		Product:  iInstrumentFromProto(i.Product),
	}
}

func (i Instrument) IntoProto() *proto.Instrument {
	p := i.Product.iIntoProto()
	r := &proto.Instrument{
		Id:       i.ID,
		Code:     i.Code,
		Name:     i.Name,
		Metadata: i.Metadata.IntoProto(),
	}
	switch pt := p.(type) {
	case *proto.Instrument_Future:
		r.Product = pt
	}
	return r
}

type MarketData struct {
	MarkPrice                 *num.Uint
	BestBidPrice              *num.Uint
	BestBidVolume             uint64
	BestOfferPrice            *num.Uint
	BestOfferVolume           uint64
	BestStaticBidPrice        *num.Uint
	BestStaticBidVolume       uint64
	BestStaticOfferPrice      *num.Uint
	BestStaticOfferVolume     uint64
	MidPrice                  *num.Uint
	StaticMidPrice            *num.Uint
	Market                    string
	Timestamp                 int64
	OpenInterest              uint64
	AuctionEnd                int64
	AuctionStart              int64
	IndicativePrice           *num.Uint
	IndicativeVolume          uint64
	MarketTradingMode         MarketTradingMode
	Trigger                   AuctionTrigger
	ExtensionTrigger          AuctionTrigger
	TargetStake               string
	SuppliedStake             string
	PriceMonitoringBounds     []*PriceMonitoringBounds
	MarketValueProxy          string
	LiquidityProviderFeeShare []*LiquidityProviderFeeShare
}

func (m MarketData) DeepClone() *MarketData {
	cpy := m
	if m.MarkPrice != nil {
		cpy.MarkPrice = m.MarkPrice.Clone()
	}
	if m.BestBidPrice != nil {
		cpy.BestBidPrice = m.BestBidPrice.Clone()
	}
	if m.BestOfferPrice != nil {
		cpy.BestOfferPrice = m.BestOfferPrice.Clone()
	}
	if m.BestStaticBidPrice != nil {
		cpy.BestStaticBidPrice = m.BestStaticBidPrice.Clone()
	}
	if m.BestStaticOfferPrice != nil {
		cpy.BestStaticOfferPrice = m.BestStaticOfferPrice.Clone()
	}
	if m.MidPrice != nil {
		cpy.MidPrice = m.MidPrice.Clone()
	}
	if m.StaticMidPrice != nil {
		cpy.StaticMidPrice = m.StaticMidPrice.Clone()
	}
	if m.IndicativePrice != nil {
		cpy.IndicativePrice = m.IndicativePrice.Clone()
	}
	cpy.PriceMonitoringBounds = make([]*PriceMonitoringBounds, 0, len(m.PriceMonitoringBounds))
	for _, pmb := range m.PriceMonitoringBounds {
		cpy.PriceMonitoringBounds = append(cpy.PriceMonitoringBounds, pmb.DeepClone())
	}
	lpfs := make([]*LiquidityProviderFeeShare, 0, len(m.LiquidityProviderFeeShare))
	for _, fs := range m.LiquidityProviderFeeShare {
		lpfs = append(lpfs, fs.DeepClone())
	}
	cpy.LiquidityProviderFeeShare = lpfs
	return &cpy
}

func (m MarketData) IntoProto() *proto.MarketData {
	r := &proto.MarketData{
		MarkPrice:                 num.UintToString(m.MarkPrice),
		BestBidPrice:              num.UintToString(m.BestBidPrice),
		BestBidVolume:             m.BestBidVolume,
		BestOfferPrice:            num.UintToString(m.BestOfferPrice),
		BestOfferVolume:           m.BestOfferVolume,
		BestStaticBidPrice:        num.UintToString(m.BestStaticBidPrice),
		BestStaticBidVolume:       m.BestStaticBidVolume,
		BestStaticOfferPrice:      num.UintToString(m.BestStaticOfferPrice),
		BestStaticOfferVolume:     m.BestStaticOfferVolume,
		MidPrice:                  num.UintToString(m.MidPrice),
		StaticMidPrice:            num.UintToString(m.StaticMidPrice),
		Market:                    m.Market,
		Timestamp:                 m.Timestamp,
		OpenInterest:              m.OpenInterest,
		AuctionEnd:                m.AuctionEnd,
		AuctionStart:              m.AuctionStart,
		IndicativePrice:           num.UintToString(m.IndicativePrice),
		IndicativeVolume:          m.IndicativeVolume,
		MarketTradingMode:         m.MarketTradingMode,
		Trigger:                   m.Trigger,
		ExtensionTrigger:          m.ExtensionTrigger,
		TargetStake:               m.TargetStake,
		SuppliedStake:             m.SuppliedStake,
		PriceMonitoringBounds:     make([]*proto.PriceMonitoringBounds, 0, len(m.PriceMonitoringBounds)),
		MarketValueProxy:          m.MarketValueProxy,
		LiquidityProviderFeeShare: make([]*proto.LiquidityProviderFeeShare, 0, len(m.LiquidityProviderFeeShare)),
	}
	for _, pmb := range m.PriceMonitoringBounds {
		r.PriceMonitoringBounds = append(r.PriceMonitoringBounds, pmb.IntoProto())
	}
	for _, lpfs := range m.LiquidityProviderFeeShare {
		r.LiquidityProviderFeeShare = append(r.LiquidityProviderFeeShare, lpfs.DeepClone()) // call IntoProto if this type gets updated
	}
	return r
}

func (m MarketData) String() string {
	return m.IntoProto().String()
}

type istmc interface {
	istmc()
	tmcIntoProto() interface{}
	tmcType() MarketTradingConfigType
}

type Market struct {
	ID                            string
	TradableInstrument            *TradableInstrument
	DecimalPlaces                 uint64
	Fees                          *Fees
	OpeningAuction                *AuctionDuration
	TradingModeConfig             istmc
	PriceMonitoringSettings       *PriceMonitoringSettings
	LiquidityMonitoringParameters *LiquidityMonitoringParameters
	TradingMode                   MarketTradingMode
	State                         MarketState
	MarketTimestamps              *MarketTimestamps
	tmc                           MarketTradingConfigType
	asset                         string
}

func MarketFromProto(mkt *proto.Market) *Market {
	asset, _ := mkt.GetAsset()
	m := &Market{
		ID:                            mkt.Id,
		TradableInstrument:            TradableInstrumentFromProto(mkt.TradableInstrument),
		DecimalPlaces:                 mkt.DecimalPlaces,
		Fees:                          FeesFromProto(mkt.Fees),
		OpeningAuction:                AuctionDurationFromProto(mkt.OpeningAuction),
		TradingModeConfig:             tmcFromProto(mkt.TradingModeConfig),
		PriceMonitoringSettings:       PriceMonitoringSettingsFromProto(mkt.PriceMonitoringSettings),
		LiquidityMonitoringParameters: LiquidityMonitoringParametersFromProto(mkt.LiquidityMonitoringParameters),
		TradingMode:                   mkt.TradingMode,
		State:                         mkt.State,
		MarketTimestamps:              MarketTimestampsFromProto(mkt.MarketTimestamps),
		asset:                         asset,
	}
	if m.TradingModeConfig != nil {
		m.tmc = m.TradingModeConfig.tmcType()
	}
	return m
}

func (m Market) IntoProto() *proto.Market {
	var (
		openAuct *proto.AuctionDuration
		mktTS    *proto.MarketTimestamps
		ti       *proto.TradableInstrument
		fees     *proto.Fees
		pms      *proto.PriceMonitoringSettings
		lms      *proto.LiquidityMonitoringParameters
	)
	if m.OpeningAuction != nil {
		openAuct = m.OpeningAuction.IntoProto()
	}
	if m.MarketTimestamps != nil {
		mktTS = m.MarketTimestamps.IntoProto()
	}
	if m.TradableInstrument != nil {
		ti = m.TradableInstrument.IntoProto()
	}
	if m.Fees != nil {
		fees = m.Fees.IntoProto()
	}
	if m.PriceMonitoringSettings != nil {
		pms = m.PriceMonitoringSettings.IntoProto()
	}
	if m.LiquidityMonitoringParameters != nil {
		lms = m.LiquidityMonitoringParameters.IntoProto()
	}
	r := &proto.Market{
		Id:                            m.ID,
		TradableInstrument:            ti,
		DecimalPlaces:                 m.DecimalPlaces,
		Fees:                          fees,
		OpeningAuction:                openAuct,
		PriceMonitoringSettings:       pms,
		LiquidityMonitoringParameters: lms,
		TradingMode:                   m.TradingMode,
		State:                         m.State,
		MarketTimestamps:              mktTS,
	}
	if m.TradingModeConfig == nil {
		return r
	}
	tmc := m.TradingModeConfig.tmcIntoProto()
	switch tm := tmc.(type) {
	case *proto.Market_Continuous:
		r.TradingModeConfig = tm
	case *proto.Market_Discrete:
		r.TradingModeConfig = tm
	}
	return r
}

func (m Market) GetID() string {
	return m.ID
}

func (m *Market) getAsset() (string, error) {
	if m.TradableInstrument == nil {
		return "", ErrNilTradableInstrument
	}
	if m.TradableInstrument.Instrument == nil {
		return "", ErrNilInstrument
	}
	if m.TradableInstrument.Instrument.Product == nil {
		return "", ErrNilProduct
	}

	return m.TradableInstrument.Instrument.Product.getAsset()
}

func (m *Market) GetAsset() (string, error) {
	if m.asset == "" {
		asset, err := m.getAsset()
		if err != nil {
			return asset, err
		}
		m.asset = asset
	}
	return m.asset, nil
}

func (m Market) GetContinuous() *MarketContinuous {
	if m.tmc == MarketTradingConfigUndefined && m.TradingModeConfig != nil {
		m.tmc = m.TradingModeConfig.tmcType()
	}
	if m.tmc != MarketTradingConfigContinuous {
		return nil
	}
	r, _ := m.TradingModeConfig.(*MarketContinuous)
	return r
}

func (m Market) GetDiscrete() *MarketDiscrete {
	if m.tmc == MarketTradingConfigUndefined && m.TradingModeConfig != nil {
		m.tmc = m.TradingModeConfig.tmcType()
	}
	if m.tmc != MarketTradingConfigDiscrete {
		return nil
	}
	r, _ := m.TradingModeConfig.(*MarketDiscrete)
	return r
}

func (m Market) String() string {
	return m.IntoProto().String()
}

func (m Market) DeepClone() *Market {
	return nil
}
