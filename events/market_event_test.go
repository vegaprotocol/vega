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

package events_test

import (
	"context"
	"testing"

	vegapb "code.vegaprotocol.io/protos/vega"
	oraclespb "code.vegaprotocol.io/protos/vega/oracles/v1"
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
	"github.com/stretchr/testify/assert"
)

func changeOracleSpec(spec *types.OracleSpec) {
	spec.ID = "Changed"
	spec.CreatedAt = 999
	spec.UpdatedAt = 999
	spec.PubKeys[0] = "Changed"
	spec.Filters[0].Key.Name = "Changed"
	spec.Filters[0].Key.Type = oraclespb.PropertyKey_TYPE_UNSPECIFIED
	spec.Filters[0].Conditions[0].Operator = oraclespb.Condition_OPERATOR_UNSPECIFIED
	spec.Filters[0].Conditions[0].Value = "Changed"
	spec.Status = oraclespb.OracleSpec_STATUS_UNSPECIFIED
}

func assertSpecsNotEqual(t *testing.T, spec1 *types.OracleSpec, spec2 *types.OracleSpec) {
	t.Helper()
	assert.NotEqual(t, spec1.ID, spec2.ID)
	assert.NotEqual(t, spec1.CreatedAt, spec2.CreatedAt)
	assert.NotEqual(t, spec1.UpdatedAt, spec2.UpdatedAt)
	assert.NotEqual(t, spec1.PubKeys[0], spec2.PubKeys[0])
	assert.NotEqual(t, spec1.Filters[0].Key.Name, spec2.Filters[0].Key.Name)
	assert.NotEqual(t, spec1.Filters[0].Key.Type, spec2.Filters[0].Key.Type)
	assert.NotEqual(t, spec1.Filters[0].Conditions[0].Operator, spec2.Filters[0].Conditions[0].Operator)
	assert.NotEqual(t, spec1.Filters[0].Conditions[0].Value, spec2.Filters[0].Conditions[0].Value)
	assert.NotEqual(t, spec1.Status, spec2.Status)
}

func TestMarketDeepClone(t *testing.T) {
	ctx := context.Background()

	pme := vegapb.Market{
		Id: "Id",
		TradableInstrument: &vegapb.TradableInstrument{
			Instrument: &vegapb.Instrument{
				Id:   "Id",
				Code: "Code",
				Name: "Name",
				Metadata: &vegapb.InstrumentMetadata{
					Tags: []string{"Tag1", "Tag2"},
				},
				Product: &vegapb.Instrument_Future{
					Future: &vegapb.Future{
						SettlementAsset: "Asset",
						QuoteName:       "QuoteName",
						OracleSpecForSettlementPrice: &oraclespb.OracleSpec{
							Id:        "Id",
							CreatedAt: 1000,
							UpdatedAt: 2000,
							PubKeys:   []string{"PubKey "},
							Filters: []*oraclespb.Filter{
								{
									Key: &oraclespb.PropertyKey{
										Name: "Name",
										Type: oraclespb.PropertyKey_TYPE_DECIMAL,
									},
									Conditions: []*oraclespb.Condition{
										{
											Operator: oraclespb.Condition_OPERATOR_EQUALS,
											Value:    "Value",
										},
									},
								},
							},
							Status: oraclespb.OracleSpec_STATUS_ACTIVE,
						},
						OracleSpecForTradingTermination: &oraclespb.OracleSpec{
							Id:        "Id2",
							CreatedAt: 1000,
							UpdatedAt: 2000,
							PubKeys:   []string{"PubKey "},
							Filters: []*oraclespb.Filter{
								{
									Key: &oraclespb.PropertyKey{
										Name: "Name",
										Type: oraclespb.PropertyKey_TYPE_BOOLEAN,
									},
									Conditions: []*oraclespb.Condition{
										{
											Operator: oraclespb.Condition_OPERATOR_EQUALS,
											Value:    "Value",
										},
									},
								},
							},
							Status: oraclespb.OracleSpec_STATUS_ACTIVE,
						},

						OracleSpecBinding: &vegapb.OracleSpecToFutureBinding{
							SettlementPriceProperty:    "SettlementPrice",
							TradingTerminationProperty: "trading.terminated",
						},
					},
				},
			},
			MarginCalculator: &vegapb.MarginCalculator{
				ScalingFactors: &vegapb.ScalingFactors{
					SearchLevel:       123.45,
					InitialMargin:     234.56,
					CollateralRelease: 345.67,
				},
			},
			RiskModel: &vegapb.TradableInstrument_SimpleRiskModel{
				SimpleRiskModel: &vegapb.SimpleRiskModel{
					Params: &vegapb.SimpleModelParams{
						FactorLong:           123.45,
						FactorShort:          234.56,
						MaxMoveUp:            345.67,
						MinMoveDown:          456.78,
						ProbabilityOfTrading: 567.89,
					},
				},
			},
		},
		DecimalPlaces: 5,
		Fees: &vegapb.Fees{
			Factors: &vegapb.FeeFactors{
				MakerFee:          "0.1",
				InfrastructureFee: "0.2",
				LiquidityFee:      "0.3",
			},
		},
		OpeningAuction: &vegapb.AuctionDuration{
			Duration: 1000,
			Volume:   2000,
		},
		PriceMonitoringSettings: &vegapb.PriceMonitoringSettings{
			Parameters: &vegapb.PriceMonitoringParameters{
				Triggers: []*vegapb.PriceMonitoringTrigger{
					{
						Horizon:          1000,
						Probability:      "123.45",
						AuctionExtension: 2000,
					},
				},
			},
			UpdateFrequency: 3000,
		},
		LiquidityMonitoringParameters: &vegapb.LiquidityMonitoringParameters{
			TargetStakeParameters: &vegapb.TargetStakeParameters{
				TimeWindow:    1000,
				ScalingFactor: 2.0,
			},
			TriggeringRatio:  123.45,
			AuctionExtension: 5000,
		},
		TradingMode: vegapb.Market_TRADING_MODE_CONTINUOUS,
		State:       vegapb.Market_STATE_ACTIVE,
		MarketTimestamps: &vegapb.MarketTimestamps{
			Proposed: 1000,
			Pending:  2000,
			Open:     3000,
			Close:    4000,
		},
	}

	me := types.MarketFromProto(&pme)
	marketEvent := events.NewMarketCreatedEvent(ctx, *me)
	mktProto := marketEvent.Market()
	me2 := types.MarketFromProto(&mktProto)

	// Change the original and check we are not updating the wrapped event
	me.ID = "Changed"
	me.TradableInstrument.Instrument.ID = "Changed"
	me.TradableInstrument.Instrument.Code = "Changed"
	me.TradableInstrument.Instrument.Name = "Changed"
	me.TradableInstrument.Instrument.Metadata.Tags[0] = "Changed1"
	me.TradableInstrument.Instrument.Metadata.Tags[1] = "Changed2"
	future := me.TradableInstrument.Instrument.Product.(*types.InstrumentFuture)
	future.Future.SettlementAsset = "Changed"
	future.Future.QuoteName = "Changed"
	changeOracleSpec(future.Future.OracleSpecForSettlementPrice)
	changeOracleSpec(future.Future.OracleSpecForTradingTermination)
	future.Future.OracleSpecBinding.SettlementPriceProperty = "Changed"
	future.Future.OracleSpecBinding.TradingTerminationProperty = "Changed"

	me.TradableInstrument.MarginCalculator.ScalingFactors.SearchLevel = num.DecimalFromFloat(99.9)
	me.TradableInstrument.MarginCalculator.ScalingFactors.InitialMargin = num.DecimalFromFloat(99.9)
	me.TradableInstrument.MarginCalculator.ScalingFactors.CollateralRelease = num.DecimalFromFloat(99.9)

	risk := me.TradableInstrument.RiskModel.(*types.TradableInstrumentSimpleRiskModel)
	risk.SimpleRiskModel.Params.FactorLong = num.DecimalFromFloat(99.9)
	risk.SimpleRiskModel.Params.FactorShort = num.DecimalFromFloat(99.9)
	risk.SimpleRiskModel.Params.MaxMoveUp = num.DecimalFromFloat(99.9)
	risk.SimpleRiskModel.Params.MinMoveDown = num.DecimalFromFloat(99.9)
	risk.SimpleRiskModel.Params.ProbabilityOfTrading = num.DecimalFromFloat(99.9)

	me.DecimalPlaces = 999
	me.Fees.Factors.MakerFee = num.DecimalFromFloat(1999.)
	me.Fees.Factors.InfrastructureFee = num.DecimalFromFloat(1999.)
	me.Fees.Factors.LiquidityFee = num.DecimalFromFloat(1999.)

	me.OpeningAuction.Duration = 999
	me.OpeningAuction.Volume = 999

	me.PriceMonitoringSettings.Parameters.Triggers[0].Horizon = 999
	me.PriceMonitoringSettings.Parameters.Triggers[0].Probability = num.DecimalFromFloat(99.9)
	me.PriceMonitoringSettings.Parameters.Triggers[0].AuctionExtension = 999
	me.PriceMonitoringSettings.UpdateFrequency = 999

	me.LiquidityMonitoringParameters.TargetStakeParameters.TimeWindow = 999
	me.LiquidityMonitoringParameters.TargetStakeParameters.ScalingFactor = num.DecimalFromFloat(99.9)
	me.LiquidityMonitoringParameters.TriggeringRatio = num.DecimalFromFloat(99.9)
	me.LiquidityMonitoringParameters.AuctionExtension = 999

	me.TradingMode = vegapb.Market_TRADING_MODE_UNSPECIFIED
	me.State = vegapb.Market_STATE_UNSPECIFIED
	me.MarketTimestamps.Proposed = 999
	me.MarketTimestamps.Pending = 999
	me.MarketTimestamps.Open = 999
	me.MarketTimestamps.Close = 999

	assert.NotEqual(t, me.ID, me2.ID)

	assert.NotEqual(t, me.TradableInstrument.Instrument.ID, me2.TradableInstrument.Instrument.ID)
	assert.NotEqual(t, me.TradableInstrument.Instrument.Code, me2.TradableInstrument.Instrument.Code)
	assert.NotEqual(t, me.TradableInstrument.Instrument.Name, me2.TradableInstrument.Instrument.Name)
	assert.NotEqual(t, me.TradableInstrument.Instrument.Metadata.Tags[0], me2.TradableInstrument.Instrument.Metadata.Tags[0])
	assert.NotEqual(t, me.TradableInstrument.Instrument.Metadata.Tags[1], me2.TradableInstrument.Instrument.Metadata.Tags[1])

	future2 := me2.TradableInstrument.Instrument.Product.(*types.InstrumentFuture)

	assert.NotEqual(t, future.Future.SettlementAsset, future2.Future.SettlementAsset)
	assert.NotEqual(t, future.Future.QuoteName, future2.Future.QuoteName)
	assertSpecsNotEqual(t, future.Future.OracleSpecForSettlementPrice, future2.Future.OracleSpecForSettlementPrice)
	assertSpecsNotEqual(t, future.Future.OracleSpecForTradingTermination, future2.Future.OracleSpecForTradingTermination)
	assert.NotEqual(t, future.Future.OracleSpecBinding.TradingTerminationProperty, future2.Future.OracleSpecBinding.TradingTerminationProperty)
	assert.NotEqual(t, future.Future.OracleSpecBinding.SettlementPriceProperty, future2.Future.OracleSpecBinding.SettlementPriceProperty)

	assert.NotEqual(t, me.TradableInstrument.MarginCalculator.ScalingFactors.SearchLevel, me2.TradableInstrument.MarginCalculator.ScalingFactors.SearchLevel)
	assert.NotEqual(t, me.TradableInstrument.MarginCalculator.ScalingFactors.InitialMargin, me2.TradableInstrument.MarginCalculator.ScalingFactors.InitialMargin)
	assert.NotEqual(t, me.TradableInstrument.MarginCalculator.ScalingFactors.CollateralRelease, me2.TradableInstrument.MarginCalculator.ScalingFactors.CollateralRelease)

	risk2 := me2.TradableInstrument.RiskModel.(*types.TradableInstrumentSimpleRiskModel)
	assert.NotEqual(t, risk.SimpleRiskModel.Params.FactorLong, risk2.SimpleRiskModel.Params.FactorLong)
	assert.NotEqual(t, risk.SimpleRiskModel.Params.FactorShort, risk2.SimpleRiskModel.Params.FactorShort)
	assert.NotEqual(t, risk.SimpleRiskModel.Params.MaxMoveUp, risk2.SimpleRiskModel.Params.MaxMoveUp)
	assert.NotEqual(t, risk.SimpleRiskModel.Params.MinMoveDown, risk2.SimpleRiskModel.Params.MinMoveDown)
	assert.NotEqual(t, risk.SimpleRiskModel.Params.ProbabilityOfTrading, risk2.SimpleRiskModel.Params.ProbabilityOfTrading)

	assert.NotEqual(t, me.DecimalPlaces, me2.DecimalPlaces)
	assert.NotEqual(t, me.Fees.Factors.MakerFee, me2.Fees.Factors.MakerFee)
	assert.NotEqual(t, me.Fees.Factors.InfrastructureFee, me2.Fees.Factors.InfrastructureFee)
	assert.NotEqual(t, me.Fees.Factors.LiquidityFee, me2.Fees.Factors.LiquidityFee)
	assert.NotEqual(t, me.OpeningAuction.Duration, me2.OpeningAuction.Duration)
	assert.NotEqual(t, me.OpeningAuction.Volume, me2.OpeningAuction.Volume)

	assert.NotEqual(t, me.PriceMonitoringSettings.Parameters.Triggers[0].Horizon, me2.PriceMonitoringSettings.Parameters.Triggers[0].Horizon)
	assert.NotEqual(t, me.PriceMonitoringSettings.Parameters.Triggers[0].Probability, me2.PriceMonitoringSettings.Parameters.Triggers[0].Probability)
	assert.NotEqual(t, me.PriceMonitoringSettings.Parameters.Triggers[0].AuctionExtension, me2.PriceMonitoringSettings.Parameters.Triggers[0].AuctionExtension)
	assert.NotEqual(t, me.PriceMonitoringSettings.UpdateFrequency, me2.PriceMonitoringSettings.UpdateFrequency)
	assert.NotEqual(t, me.LiquidityMonitoringParameters.TargetStakeParameters.TimeWindow, me2.LiquidityMonitoringParameters.TargetStakeParameters.TimeWindow)
	assert.NotEqual(t, me.LiquidityMonitoringParameters.TargetStakeParameters.ScalingFactor, me2.LiquidityMonitoringParameters.TargetStakeParameters.ScalingFactor)
	assert.NotEqual(t, me.LiquidityMonitoringParameters.TriggeringRatio, me2.LiquidityMonitoringParameters.TriggeringRatio)
	assert.NotEqual(t, me.LiquidityMonitoringParameters.AuctionExtension, me2.LiquidityMonitoringParameters.AuctionExtension)
	assert.NotEqual(t, me.TradingMode, me2.TradingMode)
	assert.NotEqual(t, me.State, me2.State)
	assert.NotEqual(t, me.MarketTimestamps.Proposed, me2.MarketTimestamps.Proposed)
	assert.NotEqual(t, me.MarketTimestamps.Pending, me2.MarketTimestamps.Pending)
	assert.NotEqual(t, me.MarketTimestamps.Open, me2.MarketTimestamps.Open)
	assert.NotEqual(t, me.MarketTimestamps.Close, me2.MarketTimestamps.Close)
}
