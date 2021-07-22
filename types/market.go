//lint:file-ignore ST1003 Ignore underscores in names, this is straigh copied from the proto package to ease introducing the domain types

package types

import (
	"errors"

	proto "code.vegaprotocol.io/data-node/proto/vega"
	v1 "code.vegaprotocol.io/data-node/proto/vega/oracles/v1"
	"code.vegaprotocol.io/data-node/types/num"
)

type LiquidityProviderFeeShare = proto.LiquidityProviderFeeShare

type MarketTradingConfigType int

const (
	MARKET_TRADING_CONFIG_UNDEFINED MarketTradingConfigType = iota
	MARKET_TRADING_CONFIG_CONTINUOUS
	MARKET_TRADING_CONFIG_DISCRETE
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

type Market_TradingMode = proto.Market_TradingMode

const (
	// Default value, this is invalid
	Market_TRADING_MODE_UNSPECIFIED Market_TradingMode = 0
	// Normal trading
	Market_TRADING_MODE_CONTINUOUS Market_TradingMode = 1
	// Auction trading (FBA)
	Market_TRADING_MODE_BATCH_AUCTION Market_TradingMode = 2
	// Opening auction
	Market_TRADING_MODE_OPENING_AUCTION Market_TradingMode = 3
	// Auction triggered by monitoring
	Market_TRADING_MODE_MONITORING_AUCTION Market_TradingMode = 4
)

type Market_State = proto.Market_State

const (
	// Default value, invalid
	Market_STATE_UNSPECIFIED Market_State = 0
	// The Governance proposal valid and accepted
	Market_STATE_PROPOSED Market_State = 1
	// Outcome of governance votes is to reject the market
	Market_STATE_REJECTED Market_State = 2
	// Governance vote passes/wins
	Market_STATE_PENDING Market_State = 3
	// Market triggers cancellation condition or governance
	// votes to close before market becomes Active
	Market_STATE_CANCELLED Market_State = 4
	// Enactment date reached and usual auction exit checks pass
	Market_STATE_ACTIVE Market_State = 5
	// Price monitoring or liquidity monitoring trigger
	Market_STATE_SUSPENDED Market_State = 6
	// Governance vote (to close)
	Market_STATE_CLOSED Market_State = 7
	// Defined by the product (i.e. from a product parameter,
	// specified in market definition, giving close date/time)
	Market_STATE_TRADING_TERMINATED Market_State = 8
	// Settlement triggered and completed as defined by product
	Market_STATE_SETTLED Market_State = 9
)

type AuctionTrigger = proto.AuctionTrigger

const (
	// Default value for AuctionTrigger, no auction triggered
	AuctionTrigger_AUCTION_TRIGGER_UNSPECIFIED AuctionTrigger = 0
	// Batch auction
	AuctionTrigger_AUCTION_TRIGGER_BATCH AuctionTrigger = 1
	// Opening auction
	AuctionTrigger_AUCTION_TRIGGER_OPENING AuctionTrigger = 2
	// Price monitoring trigger
	AuctionTrigger_AUCTION_TRIGGER_PRICE AuctionTrigger = 3
	// Liquidity monitoring trigger
	AuctionTrigger_AUCTION_TRIGGER_LIQUIDITY AuctionTrigger = 4
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
		Value: p.Value.Uint64(),
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
		srm, ok := t.RiskModel.(*TradableInstrument_SimpleRiskModel)
		if !ok || srm == nil {
			return nil
		}
		return srm.SimpleRiskModel
	}
	return nil
}

func (t TradableInstrument) GetLogNormalRiskModel() *LogNormalRiskModel {
	if t.rmt == LOGNORMAL_RISK_MODEL {
		lrm, ok := t.RiskModel.(*TradableInstrument_LogNormalRiskModel)
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

type Market_Discrete struct {
	Discrete *DiscreteTrading
}

func (m Market_Discrete) IntoProto() *proto.Market_Discrete {
	return &proto.Market_Discrete{
		Discrete: m.Discrete.IntoProto(),
	}
}

func (Market_Discrete) istmc() {}

func (m Market_Discrete) tmcIntoProto() interface{} {
	return m.IntoProto()
}

func MarketDiscreteFromProto(m *proto.Market_Discrete) *Market_Discrete {
	return &Market_Discrete{
		Discrete: DiscreteTradingFromProto(m.Discrete),
	}
}

func (Market_Discrete) tmcType() MarketTradingConfigType {
	return MARKET_TRADING_CONFIG_DISCRETE
}

type Market_Continuous struct {
	Continuous *ContinuousTrading
}

func MarketContinuousFromProto(c *proto.Market_Continuous) *Market_Continuous {
	return &Market_Continuous{
		Continuous: ContinuousTradingFromProto(c.Continuous),
	}
}

func (m Market_Continuous) IntoProto() *proto.Market_Continuous {
	return &proto.Market_Continuous{
		Continuous: m.Continuous.IntoProto(),
	}
}

func (Market_Continuous) tmcType() MarketTradingConfigType {
	return MARKET_TRADING_CONFIG_CONTINUOUS
}

func (Market_Continuous) istmc() {}

func (m Market_Continuous) tmcIntoProto() interface{} {
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
	Maturity          string
	SettlementAsset   string
	QuoteName         string
	OracleSpec        *v1.OracleSpec
	OracleSpecBinding *OracleSpecToFutureBinding
}

func FutureFromProto(f *proto.Future) *Future {
	return &Future{
		Maturity:          f.Maturity,
		SettlementAsset:   f.SettlementAsset,
		QuoteName:         f.QuoteName,
		OracleSpec:        f.OracleSpec.DeepClone(),
		OracleSpecBinding: OracleSpecToFutureBindingFromProto(f.OracleSpecBinding),
	}
}

func (f Future) IntoProto() *proto.Future {
	return &proto.Future{
		Maturity:          f.Maturity,
		SettlementAsset:   f.SettlementAsset,
		QuoteName:         f.QuoteName,
		OracleSpec:        f.OracleSpec.DeepClone(),
		OracleSpecBinding: f.OracleSpecBinding.IntoProto(),
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
	Id       string
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
		Id:       i.Id,
		Code:     i.Code,
		Name:     i.Name,
		Metadata: InstrumentMetadataFromProto(i.Metadata),
		Product:  iInstrumentFromProto(i.Product),
	}
}

func (i Instrument) IntoProto() *proto.Instrument {
	p := i.Product.iIntoProto()
	r := &proto.Instrument{
		Id:       i.Id,
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
	MarketTradingMode         Market_TradingMode
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
	var mp uint64
	if m.MarkPrice != nil {
		mp = m.MarkPrice.Uint64()
	}
	r := &proto.MarketData{
		MarkPrice:                 mp,
		BestBidPrice:              m.BestBidPrice.Uint64(),
		BestBidVolume:             m.BestBidVolume,
		BestOfferPrice:            m.BestOfferPrice.Uint64(),
		BestOfferVolume:           m.BestOfferVolume,
		BestStaticBidPrice:        m.BestStaticBidPrice.Uint64(),
		BestStaticBidVolume:       m.BestStaticBidVolume,
		BestStaticOfferPrice:      m.BestStaticOfferPrice.Uint64(),
		BestStaticOfferVolume:     m.BestStaticOfferVolume,
		MidPrice:                  m.MidPrice.Uint64(),
		StaticMidPrice:            m.StaticMidPrice.Uint64(),
		Market:                    m.Market,
		Timestamp:                 m.Timestamp,
		OpenInterest:              m.OpenInterest,
		AuctionEnd:                m.AuctionEnd,
		AuctionStart:              m.AuctionStart,
		IndicativePrice:           m.IndicativePrice.Uint64(),
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
	Id                            string
	TradableInstrument            *TradableInstrument
	DecimalPlaces                 uint64
	Fees                          *Fees
	OpeningAuction                *AuctionDuration
	TradingModeConfig             istmc
	PriceMonitoringSettings       *PriceMonitoringSettings
	LiquidityMonitoringParameters *LiquidityMonitoringParameters
	TradingMode                   Market_TradingMode
	State                         Market_State
	MarketTimestamps              *MarketTimestamps
	tmc                           MarketTradingConfigType
	asset                         string
}

func MarketFromProto(mkt *proto.Market) *Market {
	asset, _ := mkt.GetAsset()
	m := &Market{
		Id:                            mkt.Id,
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
	m.tmc = m.TradingModeConfig.tmcType()
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
		Id:                            m.Id,
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

func (m Market) GetId() string {
	return m.Id
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

func (m Market) GetContinuous() *Market_Continuous {
	if m.tmc == MARKET_TRADING_CONFIG_UNDEFINED && m.TradingModeConfig != nil {
		m.tmc = m.TradingModeConfig.tmcType()
	}
	if m.tmc != MARKET_TRADING_CONFIG_CONTINUOUS {
		return nil
	}
	r, _ := m.TradingModeConfig.(*Market_Continuous)
	return r
}

func (m Market) GetDiscrete() *Market_Discrete {
	if m.tmc == MARKET_TRADING_CONFIG_UNDEFINED && m.TradingModeConfig != nil {
		m.tmc = m.TradingModeConfig.tmcType()
	}
	if m.tmc != MARKET_TRADING_CONFIG_DISCRETE {
		return nil
	}
	r, _ := m.TradingModeConfig.(*Market_Discrete)
	return r
}

func (m Market) String() string {
	return m.IntoProto().String()
}

func (m Market) DeepClone() *Market {
	return nil
}
