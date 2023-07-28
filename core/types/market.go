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

//lint:file-ignore ST1003 Ignore underscores in names, this is straigh copied from the proto package to ease introducing the domain types

package types

import (
	"errors"
	"fmt"
	"strings"

	"code.vegaprotocol.io/vega/core/datasource"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/stringer"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
	"google.golang.org/protobuf/proto"
)

type LiquidityProviderFeeShares []*LiquidityProviderFeeShare

func (ls LiquidityProviderFeeShares) String() string {
	if ls == nil {
		return "[]"
	}
	strs := make([]string, 0, len(ls))
	for _, l := range ls {
		strs = append(strs, l.String())
	}
	return "[" + strings.Join(strs, ", ") + "]"
}

type LiquidityProviderFeeShare = vegapb.LiquidityProviderFeeShare

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

func MarketTimestampsFromProto(p *vegapb.MarketTimestamps) *MarketTimestamps {
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

func (m MarketTimestamps) IntoProto() *vegapb.MarketTimestamps {
	return &vegapb.MarketTimestamps{
		Proposed: m.Proposed,
		Pending:  m.Pending,
		Open:     m.Open,
		Close:    m.Close,
	}
}

func (m MarketTimestamps) DeepClone() *MarketTimestamps {
	return &MarketTimestamps{
		Proposed: m.Proposed,
		Pending:  m.Pending,
		Open:     m.Open,
		Close:    m.Close,
	}
}

func (m MarketTimestamps) String() string {
	return fmt.Sprintf(
		"proposed(%v) open(%v) pending(%v) close(%v)",
		m.Proposed,
		m.Open,
		m.Pending,
		m.Close,
	)
}

type MarketTradingMode = vegapb.Market_TradingMode

const (
	// Default value, this is invalid.
	MarketTradingModeUnspecified MarketTradingMode = vegapb.Market_TRADING_MODE_UNSPECIFIED
	// Normal trading.
	MarketTradingModeContinuous MarketTradingMode = vegapb.Market_TRADING_MODE_CONTINUOUS
	// Auction trading (FBA).
	MarketTradingModeBatchAuction MarketTradingMode = vegapb.Market_TRADING_MODE_BATCH_AUCTION
	// Opening auction.
	MarketTradingModeOpeningAuction MarketTradingMode = vegapb.Market_TRADING_MODE_OPENING_AUCTION
	// Auction triggered by monitoring.
	MarketTradingModeMonitoringAuction MarketTradingMode = vegapb.Market_TRADING_MODE_MONITORING_AUCTION
	// No trading allowed.
	MarketTradingModeNoTrading MarketTradingMode = vegapb.Market_TRADING_MODE_NO_TRADING
	// Special auction mode for market suspended via governance.
	MarketTradingModeSuspendedViaGovernance MarketTradingMode = vegapb.Market_TRADING_MODE_SUSPENDED_VIA_GOVERNANCE
)

type MarketState = vegapb.Market_State

const (
	// Default value, invalid.
	MarketStateUnspecified MarketState = vegapb.Market_STATE_UNSPECIFIED
	// The Governance proposal valid and accepted.
	MarketStateProposed MarketState = vegapb.Market_STATE_PROPOSED
	// Outcome of governance votes is to reject the market.
	MarketStateRejected MarketState = vegapb.Market_STATE_REJECTED
	// Governance vote passes/wins.
	MarketStatePending MarketState = vegapb.Market_STATE_PENDING
	// Market triggers cancellation condition or governance
	// votes to close before market becomes Active.
	MarketStateCancelled MarketState = vegapb.Market_STATE_CANCELLED
	// Enactment date reached and usual auction exit checks pass.
	MarketStateActive MarketState = vegapb.Market_STATE_ACTIVE
	// Price monitoring or liquidity monitoring trigger.
	MarketStateSuspended MarketState = vegapb.Market_STATE_SUSPENDED
	// Governance vote (to close).
	MarketStateClosed MarketState = vegapb.Market_STATE_CLOSED
	// Defined by the product (i.e. from a product parameter,
	// specified in market definition, giving close date/time).
	MarketStateTradingTerminated MarketState = vegapb.Market_STATE_TRADING_TERMINATED
	// Settlement triggered and completed as defined by product.
	MarketStateSettled MarketState = vegapb.Market_STATE_SETTLED
	// Market has been suspended via a governance proposal.
	MarketStateSuspendedViaGovernance MarketState = vegapb.Market_STATE_SUSPENDED_VIA_GOVERNANCE
)

type AuctionTrigger = vegapb.AuctionTrigger

const (
	// Default value for AuctionTrigger, no auction triggered.
	AuctionTriggerUnspecified AuctionTrigger = vegapb.AuctionTrigger_AUCTION_TRIGGER_UNSPECIFIED
	// Batch auction.
	AuctionTriggerBatch AuctionTrigger = vegapb.AuctionTrigger_AUCTION_TRIGGER_BATCH
	// Opening auction.
	AuctionTriggerOpening AuctionTrigger = vegapb.AuctionTrigger_AUCTION_TRIGGER_OPENING
	// Price monitoring trigger.
	AuctionTriggerPrice AuctionTrigger = vegapb.AuctionTrigger_AUCTION_TRIGGER_PRICE
	// Liquidity monitoring due to unmet target trigger.
	AuctionTriggerLiquidityTargetNotMet AuctionTrigger = vegapb.AuctionTrigger_AUCTION_TRIGGER_LIQUIDITY_TARGET_NOT_MET
	// Liquidity monitoring due to being unable to deploy LP orders due to missing best bid or ask.
	AuctionTriggerUnableToDeployLPOrders AuctionTrigger = vegapb.AuctionTrigger_AUCTION_TRIGGER_UNABLE_TO_DEPLOY_LP_ORDERS
	// Governance triggered auction.
	AuctionTriggerGovernanceSuspension AuctionTrigger = vegapb.AuctionTrigger_AUCTION_TRIGGER_GOVERNANCE_SUSPENSION
)

type InstrumentMetadata struct {
	Tags []string
}

func InstrumentMetadataFromProto(m *vegapb.InstrumentMetadata) *InstrumentMetadata {
	return &InstrumentMetadata{
		Tags: append([]string{}, m.Tags...),
	}
}

func (i InstrumentMetadata) IntoProto() *vegapb.InstrumentMetadata {
	tags := make([]string, 0, len(i.Tags))
	return &vegapb.InstrumentMetadata{
		Tags: append(tags, i.Tags...),
	}
}

func (i InstrumentMetadata) String() string {
	return fmt.Sprintf(
		"tags(%v)",
		Tags(i.Tags).String(),
	)
}

func (i InstrumentMetadata) DeepClone() *InstrumentMetadata {
	ret := &InstrumentMetadata{
		Tags: make([]string, len(i.Tags)),
	}
	copy(ret.Tags, i.Tags)
	return ret
}

type AuctionDuration struct {
	Duration int64
	Volume   uint64
}

func AuctionDurationFromProto(ad *vegapb.AuctionDuration) *AuctionDuration {
	if ad == nil {
		return nil
	}
	return &AuctionDuration{
		Duration: ad.Duration,
		Volume:   ad.Volume,
	}
}

func (a AuctionDuration) IntoProto() *vegapb.AuctionDuration {
	return &vegapb.AuctionDuration{
		Duration: a.Duration,
		Volume:   a.Volume,
	}
}

func (a AuctionDuration) String() string {
	return fmt.Sprintf(
		"duration(%v) volume(%v)",
		a.Duration,
		a.Volume,
	)
}

func (a AuctionDuration) DeepClone() *AuctionDuration {
	return &AuctionDuration{
		Duration: a.Duration,
		Volume:   a.Volume,
	}
}

type rmType int

const (
	SimpleRiskModelType rmType = iota
	LogNormalRiskModelType
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
	String() string
	Equal(isTRM) bool
}

func TradableInstrumentFromProto(ti *vegapb.TradableInstrument) *TradableInstrument {
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

func (t TradableInstrument) IntoProto() *vegapb.TradableInstrument {
	var (
		i *vegapb.Instrument
		m *vegapb.MarginCalculator
	)
	if t.Instrument != nil {
		i = t.Instrument.IntoProto()
	}
	if t.MarginCalculator != nil {
		m = t.MarginCalculator.IntoProto()
	}
	r := &vegapb.TradableInstrument{
		Instrument:       i,
		MarginCalculator: m,
	}
	if t.RiskModel == nil {
		return r
	}
	rmp := t.RiskModel.trmIntoProto()
	switch rm := rmp.(type) {
	case *vegapb.TradableInstrument_SimpleRiskModel:
		r.RiskModel = rm
	case *vegapb.TradableInstrument_LogNormalRiskModel:
		r.RiskModel = rm
	}
	return r
}

func (t TradableInstrument) GetSimpleRiskModel() *SimpleRiskModel {
	if t.rmt == SimpleRiskModelType {
		srm, ok := t.RiskModel.(*TradableInstrumentSimpleRiskModel)
		if !ok || srm == nil {
			return nil
		}
		return srm.SimpleRiskModel
	}
	return nil
}

func (t TradableInstrument) GetLogNormalRiskModel() *LogNormalRiskModel {
	if t.rmt == LogNormalRiskModelType {
		lrm, ok := t.RiskModel.(*TradableInstrumentLogNormalRiskModel)
		if !ok || lrm == nil {
			return nil
		}
		return lrm.LogNormalRiskModel
	}
	return nil
}

func (t TradableInstrument) String() string {
	return fmt.Sprintf(
		"instrument(%s) marginCalculator(%s) riskModel(%s)",
		stringer.ReflectPointerToString(t.Instrument),
		stringer.ReflectPointerToString(t.MarginCalculator),
		stringer.ReflectPointerToString(t.RiskModel),
	)
}

func (t TradableInstrument) DeepClone() *TradableInstrument {
	ti := &TradableInstrument{
		Instrument: t.Instrument.DeepClone(),
		RiskModel:  t.RiskModel,
		rmt:        t.rmt,
	}
	if t.MarginCalculator != nil {
		ti.MarginCalculator = t.MarginCalculator.DeepClone()
	}
	return ti
}

type InstrumentSpot struct {
	Spot *Spot
}

func (InstrumentSpot) Type() ProductType {
	return ProductTypeSpot
}

func (i InstrumentSpot) String() string {
	return fmt.Sprintf(
		"spot(%s)",
		stringer.ReflectPointerToString(i.Spot),
	)
}

type Spot struct {
	Name       string
	BaseAsset  string
	QuoteAsset string
}

func SpotFromProto(s *vegapb.Spot) *Spot {
	return &Spot{
		Name:       s.Name,
		BaseAsset:  s.BaseAsset,
		QuoteAsset: s.QuoteAsset,
	}
}

func (s Spot) IntoProto() *vegapb.Spot {
	return &vegapb.Spot{
		Name:       s.Name,
		BaseAsset:  s.BaseAsset,
		QuoteAsset: s.QuoteAsset,
	}
}

func (s Spot) String() string {
	return fmt.Sprintf(
		"baseAsset(%s) quoteAsset(%s)",
		s.BaseAsset,
		s.QuoteAsset,
	)
}

type InstrumentPerpetual struct {
	Perpetual *Perpetual
}

type Perpetual struct {
	MarginFundingFactor *num.Decimal
	// TODO the plumbing
	// https://github.com/vegaprotocol/vega/issues/8756
	// https://github.com/vegaprotocol/vega/issues/8753
}

type InstrumentFuture struct {
	Future *Future
}

func (InstrumentFuture) Type() ProductType {
	return ProductTypeFuture
}

func (i InstrumentFuture) String() string {
	return fmt.Sprintf(
		"future(%s)",
		stringer.ReflectPointerToString(i.Future),
	)
}

type Future struct {
	SettlementAsset                     string
	QuoteName                           string
	DataSourceSpecForSettlementData     *datasource.Spec
	DataSourceSpecForTradingTermination *datasource.Spec
	DataSourceSpecBinding               *datasource.SpecBindingForFuture
}

func FutureFromProto(f *vegapb.Future) *Future {
	return &Future{
		SettlementAsset:                     f.SettlementAsset,
		QuoteName:                           f.QuoteName,
		DataSourceSpecForSettlementData:     datasource.SpecFromProto(f.DataSourceSpecForSettlementData),
		DataSourceSpecForTradingTermination: datasource.SpecFromProto(f.DataSourceSpecForTradingTermination),
		DataSourceSpecBinding:               datasource.SpecBindingForFutureFromProto(f.DataSourceSpecBinding),
	}
}

func (f Future) IntoProto() *vegapb.Future {
	return &vegapb.Future{
		SettlementAsset:                     f.SettlementAsset,
		QuoteName:                           f.QuoteName,
		DataSourceSpecForSettlementData:     f.DataSourceSpecForSettlementData.IntoProto(),
		DataSourceSpecForTradingTermination: f.DataSourceSpecForTradingTermination.IntoProto(),
		DataSourceSpecBinding:               f.DataSourceSpecBinding.IntoProto(),
	}
}

func (f Future) String() string {
	return fmt.Sprintf(
		"quoteName(%s) settlementAsset(%s) dataSourceSpec(settlementData(%s) tradingTermination(%s) binding(%s))",
		f.QuoteName,
		f.SettlementAsset,
		stringer.ReflectPointerToString(f.DataSourceSpecForSettlementData),
		stringer.ReflectPointerToString(f.DataSourceSpecForTradingTermination),
		stringer.ReflectPointerToString(f.DataSourceSpecBinding),
	)
}

type InstrumentPerps struct {
	Perps *Perps
}

func (InstrumentPerps) Type() ProductType {
	return ProductTypePerps
}

func (i InstrumentPerps) String() string {
	return fmt.Sprintf(
		"perps(%s)",
		stringer.ReflectPointerToString(i.Perps),
	)
}

type Perps struct {
	SettlementAsset string
	QuoteName       string

	MarginFundingFactor num.Decimal
	InterestRate        num.Decimal
	ClampLowerBound     num.Decimal
	ClampUpperBound     num.Decimal

	DataSourceSpecForSettlementData     *datasource.Spec
	DataSourceSpecForSettlementSchedule *datasource.Spec
	DataSourceSpecBinding               *datasource.SpecBindingForPerps
}

func PerpsFromProto(p *vegapb.Perps) *Perps {
	return &Perps{
		SettlementAsset:                     p.SettlementAsset,
		QuoteName:                           p.QuoteName,
		MarginFundingFactor:                 num.MustDecimalFromString(p.MarginFundingFactor),
		InterestRate:                        num.MustDecimalFromString(p.InterestRate),
		ClampLowerBound:                     num.MustDecimalFromString(p.ClampLowerBound),
		ClampUpperBound:                     num.MustDecimalFromString(p.ClampUpperBound),
		DataSourceSpecForSettlementData:     datasource.SpecFromProto(p.DataSourceSpecForSettlementData),
		DataSourceSpecForSettlementSchedule: datasource.SpecFromProto(p.DataSourceSpecForSettlementSchedule),
		DataSourceSpecBinding:               datasource.SpecBindingForPerpsFromProto(p.DataSourceSpecBinding),
	}
}

func (p Perps) IntoProto() *vegapb.Perps {
	return &vegapb.Perps{
		SettlementAsset:                     p.SettlementAsset,
		QuoteName:                           p.QuoteName,
		MarginFundingFactor:                 p.MarginFundingFactor.String(),
		InterestRate:                        p.InterestRate.String(),
		ClampLowerBound:                     p.ClampLowerBound.String(),
		ClampUpperBound:                     p.ClampUpperBound.String(),
		DataSourceSpecForSettlementData:     p.DataSourceSpecForSettlementData.IntoProto(),
		DataSourceSpecForSettlementSchedule: p.DataSourceSpecForSettlementSchedule.IntoProto(),
		DataSourceSpecBinding:               p.DataSourceSpecBinding.IntoProto(),
	}
}

func (p Perps) String() string {
	return fmt.Sprintf(
		"quoteName(%s) settlementAsset(%s) marginFundingFactore(%s) interestRate(%s) clampLowerBound(%s) clampUpperBound(%s) settlementData(%s) tradingTermination(%s) binding(%s)",
		p.QuoteName,
		p.SettlementAsset,
		p.MarginFundingFactor.String(),
		p.InterestRate.String(),
		p.ClampLowerBound.String(),
		p.ClampUpperBound.String(),
		stringer.ReflectPointerToString(p.DataSourceSpecForSettlementData),
		stringer.ReflectPointerToString(p.DataSourceSpecForSettlementSchedule),
		stringer.ReflectPointerToString(p.DataSourceSpecBinding),
	)
}

func iInstrumentFromProto(pi interface{}) iProto {
	switch i := pi.(type) {
	case vegapb.Instrument_Future:
		return InstrumentFutureFromProto(&i)
	case *vegapb.Instrument_Future:
		return InstrumentFutureFromProto(i)
	case vegapb.Instrument_Perps:
		return InstrumentPerpsFromProto(&i)
	case *vegapb.Instrument_Perps:
		return InstrumentPerpsFromProto(i)
	case vegapb.Instrument_Spot:
		return InstrumentSpotFromProto(&i)
	case *vegapb.Instrument_Spot:
		return InstrumentSpotFromProto(i)
	}
	return nil
}

func InstrumentSpotFromProto(f *vegapb.Instrument_Spot) *InstrumentSpot {
	return &InstrumentSpot{
		Spot: SpotFromProto(f.Spot),
	}
}

func (i InstrumentSpot) IntoProto() *vegapb.Instrument_Spot {
	return &vegapb.Instrument_Spot{
		Spot: i.Spot.IntoProto(),
	}
}

func (i InstrumentSpot) getAssets() ([]string, error) {
	if i.Spot == nil {
		return []string{}, ErrUnknownAsset
	}
	return []string{i.Spot.BaseAsset, i.Spot.QuoteAsset}, nil
}

func (i InstrumentSpot) iIntoProto() interface{} {
	return i.IntoProto()
}

func InstrumentFutureFromProto(f *vegapb.Instrument_Future) *InstrumentFuture {
	return &InstrumentFuture{
		Future: FutureFromProto(f.Future),
	}
}

func (i InstrumentFuture) IntoProto() *vegapb.Instrument_Future {
	return &vegapb.Instrument_Future{
		Future: i.Future.IntoProto(),
	}
}

func (i InstrumentFuture) getAssets() ([]string, error) {
	if i.Future == nil {
		return []string{}, ErrUnknownAsset
	}
	return []string{i.Future.SettlementAsset}, nil
}

func InstrumentPerpsFromProto(p *vegapb.Instrument_Perps) *InstrumentPerps {
	return &InstrumentPerps{
		Perps: PerpsFromProto(p.Perps),
	}
}

func (i InstrumentPerps) IntoProto() *vegapb.Instrument_Perps {
	return &vegapb.Instrument_Perps{
		Perps: i.Perps.IntoProto(),
	}
}

func (i InstrumentPerps) getAssets() ([]string, error) {
	if i.Perps == nil {
		return []string{}, ErrUnknownAsset
	}
	return []string{i.Perps.SettlementAsset}, nil
}

func (m *Market) GetAssets() ([]string, error) {
	if m.TradableInstrument == nil {
		return []string{}, ErrNilTradableInstrument
	}
	if m.TradableInstrument.Instrument == nil {
		return []string{}, ErrNilInstrument
	}
	if m.TradableInstrument.Instrument.Product == nil {
		return []string{}, ErrNilProduct
	}

	return m.TradableInstrument.Instrument.Product.getAssets()
}

func (m *Market) ProductType() ProductType {
	return m.TradableInstrument.Instrument.Product.Type()
}

func (m *Market) GetFuture() *InstrumentFuture {
	if m.ProductType() == ProductTypeFuture {
		f, _ := m.TradableInstrument.Instrument.Product.(*InstrumentFuture)
		return f
	}
	return nil
}

func (m *Market) GetPerps() *InstrumentPerps {
	if m.ProductType() == ProductTypePerps {
		p, _ := m.TradableInstrument.Instrument.Product.(*InstrumentPerps)
		return p
	}
	return nil
}

func (m *Market) GetSpot() *InstrumentSpot {
	if m.ProductType() == ProductTypeSpot {
		s, _ := m.TradableInstrument.Instrument.Product.(*InstrumentSpot)
		return s
	}
	return nil
}

func (i InstrumentFuture) iIntoProto() interface{} {
	return i.IntoProto()
}

func (i InstrumentPerps) iIntoProto() interface{} {
	return i.IntoProto()
}

type iProto interface {
	iIntoProto() interface{}
	getAssets() ([]string, error)
	String() string
	Type() ProductType
}

type Instrument struct {
	ID       string
	Code     string
	Name     string
	Metadata *InstrumentMetadata
	// Types that are valid to be assigned to Product:
	//	*InstrumentFuture
	//	*InstrumentSpot
	//  *InstrumentPerps
	Product iProto
}

func InstrumentFromProto(i *vegapb.Instrument) *Instrument {
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

func (i Instrument) GetSpot() *Spot {
	switch p := i.Product.(type) {
	case *InstrumentSpot:
		return p.Spot
	default:
		return nil
	}
}

func (i Instrument) GetFuture() *Future {
	switch p := i.Product.(type) {
	case *InstrumentFuture:
		return p.Future
	default:
		return nil
	}
}

func (i Instrument) GetPerps() *Perps {
	switch p := i.Product.(type) {
	case *InstrumentPerps:
		return p.Perps
	default:
		return nil
	}
}

func (i Instrument) IntoProto() *vegapb.Instrument {
	p := i.Product.iIntoProto()
	r := &vegapb.Instrument{
		Id:       i.ID,
		Code:     i.Code,
		Name:     i.Name,
		Metadata: i.Metadata.IntoProto(),
	}
	switch pt := p.(type) {
	case *vegapb.Instrument_Future:
		r.Product = pt
	case *vegapb.Instrument_Perps:
		r.Product = pt
	case *vegapb.Instrument_Spot:
		r.Product = pt
	}
	return r
}

func (i Instrument) DeepClone() *Instrument {
	cpy := &Instrument{
		ID:      i.ID,
		Code:    i.Code,
		Name:    i.Name,
		Product: i.Product,
	}

	if i.Metadata != nil {
		cpy.Metadata = i.Metadata.DeepClone()
	}
	return cpy
}

func (i Instrument) String() string {
	return fmt.Sprintf(
		"ID(%s) name(%s) code(%s) product(%s) metadata(%s)",
		i.ID,
		i.Name,
		i.Code,
		stringer.ReflectPointerToString(i.Product),
		stringer.ReflectPointerToString(i.Metadata),
	)
}

type MarketData struct {
	MarkPrice                 *num.Uint
	LastTradedPrice           *num.Uint
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
	MarketState               MarketState
	Trigger                   AuctionTrigger
	ExtensionTrigger          AuctionTrigger
	TargetStake               string
	SuppliedStake             string
	PriceMonitoringBounds     []*PriceMonitoringBounds
	MarketValueProxy          string
	LiquidityProviderFeeShare []*LiquidityProviderFeeShare
	NextMTM                   int64
	MarketGrowth              num.Decimal
}

func (m MarketData) DeepClone() *MarketData {
	cpy := m
	cpy.MarkPrice = m.MarkPrice.Clone()
	cpy.LastTradedPrice = m.LastTradedPrice.Clone()
	cpy.BestBidPrice = m.BestBidPrice.Clone()
	cpy.BestOfferPrice = m.BestOfferPrice.Clone()
	cpy.BestStaticBidPrice = m.BestStaticBidPrice.Clone()
	cpy.BestStaticOfferPrice = m.BestStaticOfferPrice.Clone()
	cpy.MidPrice = m.MidPrice.Clone()
	cpy.StaticMidPrice = m.StaticMidPrice.Clone()
	cpy.IndicativePrice = m.IndicativePrice.Clone()

	cpy.PriceMonitoringBounds = make([]*PriceMonitoringBounds, 0, len(m.PriceMonitoringBounds))
	for _, pmb := range m.PriceMonitoringBounds {
		cpy.PriceMonitoringBounds = append(cpy.PriceMonitoringBounds, pmb.DeepClone())
	}
	lpfs := make([]*LiquidityProviderFeeShare, 0, len(m.LiquidityProviderFeeShare))
	for _, fs := range m.LiquidityProviderFeeShare {
		lpfs = append(lpfs, proto.Clone(fs).(*LiquidityProviderFeeShare))
	}
	cpy.LiquidityProviderFeeShare = lpfs
	return &cpy
}

func (m MarketData) IntoProto() *vegapb.MarketData {
	r := &vegapb.MarketData{
		MarkPrice:                 num.UintToString(m.MarkPrice),
		LastTradedPrice:           num.UintToString(m.LastTradedPrice),
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
		MarketState:               m.MarketState,
		Trigger:                   m.Trigger,
		ExtensionTrigger:          m.ExtensionTrigger,
		TargetStake:               m.TargetStake,
		SuppliedStake:             m.SuppliedStake,
		PriceMonitoringBounds:     make([]*vegapb.PriceMonitoringBounds, 0, len(m.PriceMonitoringBounds)),
		MarketValueProxy:          m.MarketValueProxy,
		LiquidityProviderFeeShare: make([]*vegapb.LiquidityProviderFeeShare, 0, len(m.LiquidityProviderFeeShare)),
		NextMarkToMarket:          m.NextMTM,
		MarketGrowth:              m.MarketGrowth.String(),
	}
	for _, pmb := range m.PriceMonitoringBounds {
		r.PriceMonitoringBounds = append(r.PriceMonitoringBounds, pmb.IntoProto())
	}
	for _, lpfs := range m.LiquidityProviderFeeShare {
		r.LiquidityProviderFeeShare = append(r.LiquidityProviderFeeShare, proto.Clone(lpfs).(*vegapb.LiquidityProviderFeeShare)) // call IntoProto if this type gets updated
	}
	return r
}

func (m MarketData) String() string {
	return fmt.Sprintf(
		"markPrice(%s) lastTradedPrice(%s) bestBidPrice(%s) bestBidVolume(%v) bestOfferPrice(%s) bestOfferVolume(%v) bestStaticBidPrice(%s) bestStaticBidVolume(%v) bestStaticOfferPrice(%s) bestStaticOfferVolume(%v) midPrice(%s) staticMidPrice(%s) market(%s) timestamp(%v) openInterest(%v) auctionEnd(%v) auctionStart(%v) indicativePrice(%s) indicativeVolume(%v) marketTradingMode(%s) marketState(%s) trigger(%s) extensionTrigger(%s) targetStake(%s) suppliedStake(%s) priceMonitoringBounds(%s) marketValueProxy(%s) liquidityProviderFeeShare(%v) nextMTM(%v) marketGrowth(%v)",
		stringer.UintPointerToString(m.MarkPrice),
		stringer.UintPointerToString(m.LastTradedPrice),
		m.BestBidPrice.String(),
		m.BestBidVolume,
		stringer.UintPointerToString(m.BestOfferPrice),
		m.BestOfferVolume,
		stringer.UintPointerToString(m.BestStaticBidPrice),
		m.BestStaticBidVolume,
		stringer.UintPointerToString(m.BestStaticOfferPrice),
		m.BestStaticOfferVolume,
		stringer.UintPointerToString(m.MidPrice),
		stringer.UintPointerToString(m.StaticMidPrice),
		m.Market,
		m.Timestamp,
		m.OpenInterest,
		m.AuctionEnd,
		m.AuctionStart,
		stringer.UintPointerToString(m.IndicativePrice),
		m.IndicativeVolume,
		m.MarketTradingMode.String(),
		m.MarketState.String(),
		m.Trigger.String(),
		m.ExtensionTrigger.String(),
		m.TargetStake,
		m.SuppliedStake,
		PriceMonitoringBoundsList(m.PriceMonitoringBounds).String(),
		m.MarketValueProxy,
		LiquidityProviderFeeShares(m.LiquidityProviderFeeShare).String(),
		m.NextMTM,
		m.MarketGrowth,
	)
}

type MarketType uint32

const (
	MarketTypeFuture MarketType = iota
	MarketTypeSpot
	MarketTypePerp
)

type Market struct {
	ID                            string
	TradableInstrument            *TradableInstrument
	DecimalPlaces                 uint64
	PositionDecimalPlaces         int64
	Fees                          *Fees
	OpeningAuction                *AuctionDuration
	PriceMonitoringSettings       *PriceMonitoringSettings
	LiquidityMonitoringParameters *LiquidityMonitoringParameters
	LPPriceRange                  num.Decimal
	LinearSlippageFactor          num.Decimal
	QuadraticSlippageFactor       num.Decimal
	LiquiditySLAParams            *LiquiditySLAParams

	TradingMode           MarketTradingMode
	State                 MarketState
	MarketTimestamps      *MarketTimestamps
	ParentMarketID        string
	InsurancePoolFraction num.Decimal
	mType                 MarketType
}

func MarketFromProto(mkt *vegapb.Market) (*Market, error) {
	lppr, _ := num.DecimalFromString(mkt.LpPriceRange)
	linearSlippageFactor, _ := num.DecimalFromString(mkt.LinearSlippageFactor)
	quadraticSlippageFactor, _ := num.DecimalFromString(mkt.QuadraticSlippageFactor)
	liquidityParameters, err := LiquidityMonitoringParametersFromProto(mkt.LiquidityMonitoringParameters)
	if err != nil {
		return nil, err
	}
	insFraction := num.DecimalZero()
	if mkt.InsurancePoolFraction != nil && len(*mkt.InsurancePoolFraction) > 0 {
		insFraction = num.MustDecimalFromString(*mkt.InsurancePoolFraction)
	}
	parent := ""
	if mkt.ParentMarketId != nil {
		parent = *mkt.ParentMarketId
	}

	m := &Market{
		ID:                            mkt.Id,
		TradableInstrument:            TradableInstrumentFromProto(mkt.TradableInstrument),
		DecimalPlaces:                 mkt.DecimalPlaces,
		PositionDecimalPlaces:         mkt.PositionDecimalPlaces,
		Fees:                          FeesFromProto(mkt.Fees),
		OpeningAuction:                AuctionDurationFromProto(mkt.OpeningAuction),
		PriceMonitoringSettings:       PriceMonitoringSettingsFromProto(mkt.PriceMonitoringSettings),
		LiquidityMonitoringParameters: liquidityParameters,
		TradingMode:                   mkt.TradingMode,
		State:                         mkt.State,
		MarketTimestamps:              MarketTimestampsFromProto(mkt.MarketTimestamps),
		LPPriceRange:                  lppr,
		LinearSlippageFactor:          linearSlippageFactor,
		QuadraticSlippageFactor:       quadraticSlippageFactor,
		ParentMarketID:                parent,
		InsurancePoolFraction:         insFraction,
		mType:                         MarketTypeFuture,
	}
	if s := m.GetSpot(); s != nil {
		m.mType = MarketTypeSpot
	} else if p := m.GetPerps(); p != nil {
		m.mType = MarketTypePerp
	}

	if mkt.LiquiditySlaParams != nil {
		m.LiquiditySLAParams = LiquiditySLAParamsFromProto(mkt.LiquiditySlaParams)
	}

	return m, nil
}

func (m Market) IntoProto() *vegapb.Market {
	var (
		openAuct *vegapb.AuctionDuration
		mktTS    *vegapb.MarketTimestamps
		ti       *vegapb.TradableInstrument
		fees     *vegapb.Fees
		pms      *vegapb.PriceMonitoringSettings
		lms      *vegapb.LiquidityMonitoringParameters
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
	var parent, insPoolFrac *string
	if len(m.ParentMarketID) != 0 {
		pid, insf := m.ParentMarketID, m.InsurancePoolFraction.String()
		parent = &pid
		insPoolFrac = &insf
	}

	var lpSLA *vegapb.LiquiditySLAParameters
	if m.LiquiditySLAParams != nil {
		lpSLA = m.LiquiditySLAParams.IntoProto()
	}

	r := &vegapb.Market{
		Id:                            m.ID,
		TradableInstrument:            ti,
		DecimalPlaces:                 m.DecimalPlaces,
		PositionDecimalPlaces:         m.PositionDecimalPlaces,
		Fees:                          fees,
		OpeningAuction:                openAuct,
		PriceMonitoringSettings:       pms,
		LiquidityMonitoringParameters: lms,
		TradingMode:                   m.TradingMode,
		State:                         m.State,
		MarketTimestamps:              mktTS,
		LpPriceRange:                  m.LPPriceRange.String(),
		LiquiditySlaParams:            lpSLA,
		LinearSlippageFactor:          m.LinearSlippageFactor.String(),
		QuadraticSlippageFactor:       m.QuadraticSlippageFactor.String(),
		InsurancePoolFraction:         insPoolFrac,
		ParentMarketId:                parent,
	}
	return r
}

func (m Market) GetID() string {
	return m.ID
}

func (m Market) String() string {
	return fmt.Sprintf(
		"ID(%s) tradableInstrument(%s) decimalPlaces(%v) positionDecimalPlaces(%v) fees(%s) openingAuction(%s) priceMonitoringSettings(%s) liquidityMonitoringParameters(%s) tradingMode(%s) state(%s) marketTimestamps(%s)",
		m.ID,
		stringer.ReflectPointerToString(m.TradableInstrument),
		m.DecimalPlaces,
		m.PositionDecimalPlaces,
		stringer.ReflectPointerToString(m.Fees),
		stringer.ReflectPointerToString(m.OpeningAuction),
		stringer.ReflectPointerToString(m.PriceMonitoringSettings),
		stringer.ReflectPointerToString(m.LiquidityMonitoringParameters),
		m.TradingMode.String(),
		m.State.String(),
		stringer.ReflectPointerToString(m.MarketTimestamps),
	)
}

func (m Market) MarketType() MarketType {
	return m.mType
}

func (m Market) DeepClone() *Market {
	cpy := &Market{
		ID:                      m.ID,
		DecimalPlaces:           m.DecimalPlaces,
		PositionDecimalPlaces:   m.PositionDecimalPlaces,
		TradingMode:             m.TradingMode,
		State:                   m.State,
		LPPriceRange:            m.LPPriceRange,
		LinearSlippageFactor:    m.LinearSlippageFactor,
		QuadraticSlippageFactor: m.QuadraticSlippageFactor,
		ParentMarketID:          m.ParentMarketID,
		InsurancePoolFraction:   m.InsurancePoolFraction,
	}

	if m.LiquiditySLAParams != nil {
		cpy.LiquiditySLAParams = m.LiquiditySLAParams.DeepClone()
	}

	if m.TradableInstrument != nil {
		cpy.TradableInstrument = m.TradableInstrument.DeepClone()
	}

	if m.Fees != nil {
		cpy.Fees = m.Fees.DeepClone()
	}

	if m.OpeningAuction != nil {
		cpy.OpeningAuction = m.OpeningAuction.DeepClone()
	}

	if m.PriceMonitoringSettings != nil {
		cpy.PriceMonitoringSettings = m.PriceMonitoringSettings.DeepClone()
	}

	if m.LiquidityMonitoringParameters != nil {
		cpy.LiquidityMonitoringParameters = m.LiquidityMonitoringParameters.DeepClone()
	}

	if m.MarketTimestamps != nil {
		cpy.MarketTimestamps = m.MarketTimestamps.DeepClone()
	}
	return cpy
}

type Tags []string

func (t Tags) String() string {
	return "[" + strings.Join(t, ", ") + "]"
}

func toPtr[T any](t T) *T { return &t }

type MarketCounters struct {
	StopOrderCounter    uint64
	PeggedOrderCounter  uint64
	LPShapeCount        uint64
	PositionCount       uint64
	OrderbookLevelCount uint64
}
