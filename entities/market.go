// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package entities

import (
	"errors"
	"fmt"
	"math"
	"time"

	v2 "code.vegaprotocol.io/protos/data-node/api/v2"
	"code.vegaprotocol.io/protos/vega"
	"github.com/shopspring/decimal"
)

type MarketID struct{ ID }

func NewMarketID(id string) MarketID {
	return MarketID{ID: ID(id)}
}

type Market struct {
	ID                            MarketID
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
}

func NewMarketFromProto(market *vega.Market, vegaTime time.Time) (*Market, error) {
	var err error
	var liquidityMonitoringParameters LiquidityMonitoringParameters
	var marketTimestamps MarketTimestamps
	var priceMonitoringSettings PriceMonitoringSettings
	var openingAuction AuctionDuration
	var fees Fees

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

	return &Market{
		ID:                            NewMarketID(market.Id),
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
	}, nil
}

func (m Market) ToProto() *vega.Market {
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
		PositionDecimalPlaces:         uint64(m.PositionDecimalPlaces),
	}
}

func (m Market) Cursor() *Cursor {
	return NewCursor(m.VegaTime.In(time.UTC).Format(time.RFC3339Nano))
}

func (m Market) ToProtoEdge(_ ...any) *v2.MarketEdge {
	return &v2.MarketEdge{
		Node:   m.ToProto(),
		Cursor: m.Cursor().Encode(),
	}
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
	TriggeringRatio       float64                `json:"triggeringRatio,omitempty"`
	AuctionExtension      int64                  `json:"auctionExtension,omitempty"`
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
	Parameters      *PriceMonitoringParameters `json:"priceMonitoringParameters,omitempty"`
	UpdateFrequency int64                      `json:"updateFrequency,omitempty"`
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
		UpdateFrequency: 0,
	}
}

func priceMonitoringSettingsFromProto(pms *vega.PriceMonitoringSettings) (PriceMonitoringSettings, error) {
	if pms == nil {
		return PriceMonitoringSettings{}, errors.New("price monitoring settings cannot be nil")
	}

	parameters := priceMonitoringParametersFromProto(pms.Parameters)
	return PriceMonitoringSettings{
		Parameters:      &parameters,
		UpdateFrequency: pms.UpdateFrequency,
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

type Fees struct {
	Factors *FeeFactors `json:"factors,omitempty"`
}

func (f Fees) ToProto() *vega.Fees {
	if f.Factors == nil {
		return nil
	}

	return &vega.Fees{
		Factors: &vega.FeeFactors{
			MakerFee:          f.Factors.MakerFee,
			InfrastructureFee: f.Factors.InfrastructureFee,
			LiquidityFee:      f.Factors.LiquidityFee,
		},
	}
}

func feesFromProto(fees *vega.Fees) (Fees, error) {
	if fees == nil {
		return Fees{}, errors.New("fees cannot be Nil")
	}

	return Fees{
		Factors: &FeeFactors{
			MakerFee:          fees.Factors.MakerFee,
			InfrastructureFee: fees.Factors.InfrastructureFee,
			LiquidityFee:      fees.Factors.LiquidityFee,
		},
	}, nil
}
