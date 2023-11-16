// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package entities

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/ptr"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	"code.vegaprotocol.io/vega/protos/vega"

	"github.com/shopspring/decimal"
)

type _Market struct{}

type MarketID = ID[_Market]

type Market struct {
	ID                            MarketID
	TxHash                        TxHash
	VegaTime                      time.Time
	InstrumentID                  string
	TradableInstrument            TradableInstrument
	DecimalPlaces                 int
	Fees                          Fees
	OpeningAuction                AuctionDuration
	PriceMonitoringSettings       PriceMonitoringSettings
	LiquidityMonitoringParameters LiquidityMonitoringParameters
	TradingMode                   MarketTradingMode
	State                         MarketState
	MarketTimestamps              MarketTimestamps
	PositionDecimalPlaces         int
	LpPriceRange                  string
	LinearSlippageFactor          *decimal.Decimal
	QuadraticSlippageFactor       *decimal.Decimal
	ParentMarketID                MarketID
	InsurancePoolFraction         *decimal.Decimal
	LiquiditySLAParameters        LiquiditySLAParameters
	// Not saved in the market table, but used when retrieving data from the database.
	// This will be populated when a market has a successor
	SuccessorMarketID   MarketID
	LiquidationStrategy *LiquidationStrategy
}

type MarketCursor struct {
	VegaTime time.Time `json:"vegaTime"`
	ID       MarketID  `json:"id"`
}

func (mc MarketCursor) String() string {
	bs, err := json.Marshal(mc)
	if err != nil {
		panic(fmt.Errorf("could not marshal market cursor: %w", err))
	}
	return string(bs)
}

func (mc *MarketCursor) Parse(cursorString string) error {
	if cursorString == "" {
		return nil
	}

	return json.Unmarshal([]byte(cursorString), mc)
}

func NewMarketFromProto(market *vega.Market, txHash TxHash, vegaTime time.Time) (*Market, error) {
	var (
		err                           error
		liquidityMonitoringParameters LiquidityMonitoringParameters
		marketTimestamps              MarketTimestamps
		priceMonitoringSettings       PriceMonitoringSettings
		openingAuction                AuctionDuration
		fees                          Fees
		liqStrat                      *LiquidationStrategy
	)

	if fees, err = feesFromProto(market.Fees); err != nil {
		return nil, err
	}

	if market.OpeningAuction != nil {
		openingAuction.Duration = market.OpeningAuction.Duration
		openingAuction.Volume = market.OpeningAuction.Volume
	}

	if priceMonitoringSettings, err = priceMonitoringSettingsFromProto(market.PriceMonitoringSettings); err != nil {
		return nil, err
	}

	if liquidityMonitoringParameters, err = liquidityMonitoringParametersFromProto(market.LiquidityMonitoringParameters); err != nil {
		return nil, err
	}

	if marketTimestamps, err = marketTimestampsFromProto(market.MarketTimestamps); err != nil {
		return nil, err
	}

	if market.DecimalPlaces > math.MaxInt {
		return nil, fmt.Errorf("%d is not a valid number for decimal places", market.DecimalPlaces)
	}

	if market.PositionDecimalPlaces > math.MaxInt {
		return nil, fmt.Errorf("%d is not a valid number for position decimal places", market.PositionDecimalPlaces)
	}

	dps := int(market.DecimalPlaces)
	positionDps := int(market.PositionDecimalPlaces)

	linearSlippageFactor := (*num.Decimal)(nil)
	if market.LinearSlippageFactor != "" {
		factor, err := num.DecimalFromString(market.LinearSlippageFactor)
		if err != nil {
			return nil, fmt.Errorf("'%v' is not a valid number for linear slippage factor", market.LinearSlippageFactor)
		}
		linearSlippageFactor = &factor
	}

	quadraticSlippageFactor := (*num.Decimal)(nil)
	if market.QuadraticSlippageFactor != "" {
		factor, err := num.DecimalFromString(market.QuadraticSlippageFactor)
		if err != nil {
			return nil, fmt.Errorf("'%v' is not a valid number for quadratic slippage factor", market.QuadraticSlippageFactor)
		}
		quadraticSlippageFactor = &factor
	}

	parentMarketID := MarketID("")
	if market.ParentMarketId != nil && *market.ParentMarketId != "" {
		parent := MarketID(*market.ParentMarketId)
		parentMarketID = parent
	}

	var insurancePoolFraction *num.Decimal
	if market.InsurancePoolFraction != nil && *market.InsurancePoolFraction != "" {
		insurance, err := num.DecimalFromString(*market.InsurancePoolFraction)
		if err != nil {
			return nil, fmt.Errorf("'%v' is not a valid number for insurance pool fraction", market.InsurancePoolFraction)
		}
		insurancePoolFraction = &insurance
	}

	var sla LiquiditySLAParameters
	if market.LiquiditySlaParams != nil {
		sla, err = LiquiditySLAParametersFromProto(market.LiquiditySlaParams)
		if err != nil {
			return nil, err
		}
	}
	if market.LiquidationStrategy != nil {
		liqStrat = LiquidationStrategyFromProto(market.LiquidationStrategy)
	}

	return &Market{
		ID:                            MarketID(market.Id),
		TxHash:                        txHash,
		VegaTime:                      vegaTime,
		InstrumentID:                  market.TradableInstrument.Instrument.Id,
		TradableInstrument:            TradableInstrument{market.TradableInstrument},
		DecimalPlaces:                 dps,
		Fees:                          fees,
		OpeningAuction:                openingAuction,
		PriceMonitoringSettings:       priceMonitoringSettings,
		LiquidityMonitoringParameters: liquidityMonitoringParameters,
		TradingMode:                   MarketTradingMode(market.TradingMode),
		State:                         MarketState(market.State),
		MarketTimestamps:              marketTimestamps,
		PositionDecimalPlaces:         positionDps,
		LpPriceRange:                  market.LpPriceRange,
		LinearSlippageFactor:          linearSlippageFactor,
		QuadraticSlippageFactor:       quadraticSlippageFactor,
		ParentMarketID:                parentMarketID,
		InsurancePoolFraction:         insurancePoolFraction,
		LiquiditySLAParameters:        sla,
		LiquidationStrategy:           liqStrat,
	}, nil
}

func (m Market) ToProto() *vega.Market {
	linearSlippageFactor := ""
	if m.LinearSlippageFactor != nil {
		linearSlippageFactor = m.LinearSlippageFactor.String()
	}

	quadraticSlippageFactor := ""
	if m.QuadraticSlippageFactor != nil {
		quadraticSlippageFactor = m.QuadraticSlippageFactor.String()
	}

	var parentMarketID, insurancePoolFraction *string

	if m.ParentMarketID != "" {
		parentMarketID = ptr.From(m.ParentMarketID.String())
	}

	if m.InsurancePoolFraction != nil {
		insurancePoolFraction = ptr.From(m.InsurancePoolFraction.String())
	}

	var successorMarketID *string
	if m.SuccessorMarketID != "" {
		successorMarketID = ptr.From(m.SuccessorMarketID.String())
	}

	return &vega.Market{
		Id:                 m.ID.String(),
		TradableInstrument: m.TradableInstrument.ToProto(),
		DecimalPlaces:      uint64(m.DecimalPlaces),
		Fees:               m.Fees.ToProto(),
		OpeningAuction: &vega.AuctionDuration{
			Duration: m.OpeningAuction.Duration,
			Volume:   m.OpeningAuction.Volume,
		},
		PriceMonitoringSettings:       m.PriceMonitoringSettings.ToProto(),
		LiquidityMonitoringParameters: m.LiquidityMonitoringParameters.ToProto(),
		TradingMode:                   vega.Market_TradingMode(m.TradingMode),
		State:                         vega.Market_State(m.State),
		MarketTimestamps:              m.MarketTimestamps.ToProto(),
		PositionDecimalPlaces:         int64(m.PositionDecimalPlaces),
		LpPriceRange:                  m.LpPriceRange,
		LinearSlippageFactor:          linearSlippageFactor,
		QuadraticSlippageFactor:       quadraticSlippageFactor,
		ParentMarketId:                parentMarketID,
		InsurancePoolFraction:         insurancePoolFraction,
		SuccessorMarketId:             successorMarketID,
		LiquiditySlaParams:            m.LiquiditySLAParameters.IntoProto(),
	}
}

func (m Market) Cursor() *Cursor {
	mc := MarketCursor{
		VegaTime: m.VegaTime,
		ID:       m.ID,
	}
	return NewCursor(mc.String())
}

func (m Market) ToProtoEdge(_ ...any) (*v2.MarketEdge, error) {
	return &v2.MarketEdge{
		Node:   m.ToProto(),
		Cursor: m.Cursor().Encode(),
	}, nil
}

type MarketTimestamps struct {
	Proposed int64 `json:"proposed,omitempty"`
	Pending  int64 `json:"pending,omitempty"`
	Open     int64 `json:"open,omitempty"`
	Close    int64 `json:"close,omitempty"`
}

func (mt MarketTimestamps) ToProto() *vega.MarketTimestamps {
	return &vega.MarketTimestamps{
		Proposed: mt.Proposed,
		Pending:  mt.Pending,
		Open:     mt.Open,
		Close:    mt.Close,
	}
}

func marketTimestampsFromProto(ts *vega.MarketTimestamps) (MarketTimestamps, error) {
	if ts == nil {
		return MarketTimestamps{}, errors.New("market timestamps cannot be nil")
	}

	return MarketTimestamps{
		Proposed: ts.Proposed,
		Pending:  ts.Pending,
		Open:     ts.Open,
		Close:    ts.Close,
	}, nil
}

type TargetStakeParameters struct {
	TimeWindow     int64   `json:"timeWindow,omitempty"`
	ScalingFactors float64 `json:"scalingFactor,omitempty"`
}

func (tsp TargetStakeParameters) ToProto() *vega.TargetStakeParameters {
	return &vega.TargetStakeParameters{
		TimeWindow:    tsp.TimeWindow,
		ScalingFactor: tsp.ScalingFactors,
	}
}

type LiquidityMonitoringParameters struct {
	TargetStakeParameters *TargetStakeParameters `json:"targetStakeParameters,omitempty"`
	TriggeringRatio       string                 `json:"triggeringRatio,omitempty"`
	AuctionExtension      int64                  `json:"auctionExtension,omitempty"`
}

type LiquiditySLAParameters struct {
	PriceRange                  num.Decimal `json:"priceRange,omitempty"`
	CommitmentMinTimeFraction   num.Decimal `json:"commitmentMinTimeFraction,omitempty"`
	PerformanceHysteresisEpochs uint64      `json:"performanceHysteresisEpochs,omitempty"`
	SlaCompetitionFactor        num.Decimal `json:"slaCompetitionFactor,omitempty"`
}

type LiquidationStrategy struct {
	DisposalTimeStep    time.Duration `json:"disposalTimeStep"`
	DisposalFraction    num.Decimal   `json:"disposalFraction"`
	FullDisposalSize    uint64        `json:"fullDisposalSize"`
	MaxFractionConsumed num.Decimal   `json:"maxFractionConsumed"`
}

func LiquidationStrategyFromProto(ls *vega.LiquidationStrategy) *LiquidationStrategy {
	if ls == nil {
		return nil
	}
	df, _ := num.DecimalFromString(ls.DisposalFraction)
	mfc, _ := num.DecimalFromString(ls.MaxFractionConsumed)
	return &LiquidationStrategy{
		DisposalTimeStep:    time.Duration(ls.DisposalTimeStep) * time.Second,
		FullDisposalSize:    ls.FullDisposalSize,
		DisposalFraction:    df,
		MaxFractionConsumed: mfc,
	}
}

func (l LiquidationStrategy) IntoProto() *vega.LiquidationStrategy {
	return &vega.LiquidationStrategy{
		DisposalTimeStep:    int64(l.DisposalTimeStep / time.Second),
		DisposalFraction:    l.DisposalFraction.String(),
		FullDisposalSize:    l.FullDisposalSize,
		MaxFractionConsumed: l.MaxFractionConsumed.String(),
	}
}

func (lsp LiquiditySLAParameters) IntoProto() *vega.LiquiditySLAParameters {
	return &vega.LiquiditySLAParameters{
		PriceRange:                  lsp.PriceRange.String(),
		CommitmentMinTimeFraction:   lsp.CommitmentMinTimeFraction.String(),
		SlaCompetitionFactor:        lsp.SlaCompetitionFactor.String(),
		PerformanceHysteresisEpochs: lsp.PerformanceHysteresisEpochs,
	}
}

func LiquiditySLAParametersFromProto(sla *vega.LiquiditySLAParameters) (LiquiditySLAParameters, error) {
	// SLA can be nil for futures for NOW
	if sla == nil {
		return LiquiditySLAParameters{}, nil
	}
	priceRange, err := num.DecimalFromString(sla.PriceRange)
	if err != nil {
		return LiquiditySLAParameters{}, errors.New("invalid price range in liquidity sla parameters")
	}
	commitmentMinTimeFraction, err := num.DecimalFromString(sla.CommitmentMinTimeFraction)
	if err != nil {
		return LiquiditySLAParameters{}, errors.New("invalid commitment min time fraction in liquidity sla parameters")
	}
	slaCompetitionFactor, err := num.DecimalFromString(sla.SlaCompetitionFactor)
	if err != nil {
		return LiquiditySLAParameters{}, errors.New("invalid commitment sla competition factor in liquidity sla parameters")
	}
	return LiquiditySLAParameters{
		PriceRange:                  priceRange,
		CommitmentMinTimeFraction:   commitmentMinTimeFraction,
		SlaCompetitionFactor:        slaCompetitionFactor,
		PerformanceHysteresisEpochs: sla.PerformanceHysteresisEpochs,
	}, nil
}

func (lmp LiquidityMonitoringParameters) ToProto() *vega.LiquidityMonitoringParameters {
	if lmp.TargetStakeParameters == nil {
		return nil
	}
	return &vega.LiquidityMonitoringParameters{
		TargetStakeParameters: lmp.TargetStakeParameters.ToProto(),
		TriggeringRatio:       lmp.TriggeringRatio,
		AuctionExtension:      lmp.AuctionExtension,
	}
}

func liquidityMonitoringParametersFromProto(lmp *vega.LiquidityMonitoringParameters) (LiquidityMonitoringParameters, error) {
	if lmp == nil {
		return LiquidityMonitoringParameters{}, errors.New("liquidity monitoring parameters cannot be Nil")
	}

	var tsp *TargetStakeParameters

	if lmp.TargetStakeParameters != nil {
		tsp = &TargetStakeParameters{
			TimeWindow:     lmp.TargetStakeParameters.TimeWindow,
			ScalingFactors: lmp.TargetStakeParameters.ScalingFactor,
		}
	}

	return LiquidityMonitoringParameters{
		TargetStakeParameters: tsp,
		TriggeringRatio:       lmp.TriggeringRatio,
		AuctionExtension:      lmp.AuctionExtension,
	}, nil
}

type PriceMonitoringParameters struct {
	Triggers []*PriceMonitoringTrigger `json:"triggers,omitempty"`
}

func priceMonitoringParametersFromProto(pmp *vega.PriceMonitoringParameters) PriceMonitoringParameters {
	if len(pmp.Triggers) == 0 {
		return PriceMonitoringParameters{}
	}

	triggers := make([]*PriceMonitoringTrigger, 0, len(pmp.Triggers))

	for _, trigger := range pmp.Triggers {
		probability, _ := decimal.NewFromString(trigger.Probability)
		triggers = append(triggers, &PriceMonitoringTrigger{
			Horizon:          uint64(trigger.Horizon),
			Probability:      probability,
			AuctionExtension: uint64(trigger.AuctionExtension),
		})
	}

	return PriceMonitoringParameters{
		Triggers: triggers,
	}
}

type PriceMonitoringSettings struct {
	Parameters *PriceMonitoringParameters `json:"priceMonitoringParameters,omitempty"`
}

func (s PriceMonitoringSettings) ToProto() *vega.PriceMonitoringSettings {
	if s.Parameters == nil {
		return nil
	}
	triggers := make([]*vega.PriceMonitoringTrigger, 0, len(s.Parameters.Triggers))

	if len(s.Parameters.Triggers) > 0 {
		for _, trigger := range s.Parameters.Triggers {
			triggers = append(triggers, trigger.ToProto())
		}
	}

	return &vega.PriceMonitoringSettings{
		Parameters: &vega.PriceMonitoringParameters{
			Triggers: triggers,
		},
	}
}

func priceMonitoringSettingsFromProto(pms *vega.PriceMonitoringSettings) (PriceMonitoringSettings, error) {
	if pms == nil {
		return PriceMonitoringSettings{}, errors.New("price monitoring settings cannot be nil")
	}

	parameters := priceMonitoringParametersFromProto(pms.Parameters)
	return PriceMonitoringSettings{
		Parameters: &parameters,
	}, nil
}

type AuctionDuration struct {
	Duration int64  `json:"duration,omitempty"`
	Volume   uint64 `json:"volume,omitempty"`
}

type FeeFactors struct {
	MakerFee          string `json:"makerFee,omitempty"`
	InfrastructureFee string `json:"infrastructureFee,omitempty"`
	LiquidityFee      string `json:"liquidityFee,omitempty"`
}

type LiquidityFeeSettings struct {
	Method      LiquidityFeeSettingsMethod `json:"makerFee,omitempty"`
	FeeConstant *string                    `json:"feeConstant,omitempty"`
}

type Fees struct {
	Factors              *FeeFactors           `json:"factors,omitempty"`
	LiquidityFeeSettings *LiquidityFeeSettings `json:"liquidityFeeSettings,omitempty"`
}

func (f Fees) ToProto() *vega.Fees {
	if f.Factors == nil {
		return nil
	}

	var liquidityFeeSettings *vega.LiquidityFeeSettings
	if f.LiquidityFeeSettings != nil {
		liquidityFeeSettings = &vega.LiquidityFeeSettings{
			Method:      vega.LiquidityFeeSettings_Method(f.LiquidityFeeSettings.Method),
			FeeConstant: f.LiquidityFeeSettings.FeeConstant,
		}
	}

	return &vega.Fees{
		Factors: &vega.FeeFactors{
			MakerFee:          f.Factors.MakerFee,
			InfrastructureFee: f.Factors.InfrastructureFee,
			LiquidityFee:      f.Factors.LiquidityFee,
		},
		LiquidityFeeSettings: liquidityFeeSettings,
	}
}

func feesFromProto(fees *vega.Fees) (Fees, error) {
	if fees == nil {
		return Fees{}, errors.New("fees cannot be Nil")
	}

	var liquidityFeeSettings *LiquidityFeeSettings
	if fees.LiquidityFeeSettings != nil {
		liquidityFeeSettings = &LiquidityFeeSettings{
			Method:      LiquidityFeeSettingsMethod(fees.LiquidityFeeSettings.Method),
			FeeConstant: fees.LiquidityFeeSettings.FeeConstant,
		}
	}

	return Fees{
		Factors: &FeeFactors{
			MakerFee:          fees.Factors.MakerFee,
			InfrastructureFee: fees.Factors.InfrastructureFee,
			LiquidityFee:      fees.Factors.LiquidityFee,
		},
		LiquidityFeeSettings: liquidityFeeSettings,
	}, nil
}
