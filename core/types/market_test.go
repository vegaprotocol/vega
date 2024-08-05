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

package types_test

import (
	"testing"

	dstypes "code.vegaprotocol.io/vega/core/datasource/common"
	"code.vegaprotocol.io/vega/core/types"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
	datapb "code.vegaprotocol.io/vega/protos/vega/data/v1"

	"github.com/stretchr/testify/require"
)

var testFilter1 = &datapb.Filter{
	Key: &datapb.PropertyKey{
		Name: "filter1",
		Type: datapb.PropertyKey_TYPE_STRING,
	},
	Conditions: []*datapb.Condition{
		{
			Operator: datapb.Condition_OPERATOR_EQUALS,
			Value:    "true",
		},
	},
}

func TestMarketFromIntoProto(t *testing.T) {
	pk := dstypes.CreateSignerFromString("pubkey", dstypes.SignerTypePubKey)
	fPtr := false

	pMarket := &vegapb.Market{
		Id: "foo",
		TradableInstrument: &vegapb.TradableInstrument{
			Instrument: &vegapb.Instrument{
				Id:   "bar",
				Code: "FB",
				Name: "FooBar",
				Metadata: &vegapb.InstrumentMetadata{
					Tags: []string{"test", "foo", "bar", "foobar"},
				},
				Product: &vegapb.Instrument_Future{
					Future: &vegapb.Future{
						SettlementAsset: "GBP",
						QuoteName:       "USD",
						DataSourceSpecForSettlementData: &vegapb.DataSourceSpec{
							Id:        "os1",
							CreatedAt: 0,
							UpdatedAt: 1,
							Data: vegapb.NewDataSourceDefinition(
								vegapb.DataSourceContentTypeOracle,
							).SetOracleConfig(
								&vegapb.DataSourceDefinitionExternal_Oracle{
									Oracle: &vegapb.DataSourceSpecConfiguration{
										Signers: []*datapb.Signer{pk.IntoProto()},
										Filters: []*datapb.Filter{testFilter1},
									},
								},
							),
							Status: vegapb.DataSourceSpec_STATUS_ACTIVE,
						},
						DataSourceSpecForTradingTermination: &vegapb.DataSourceSpec{
							Id:        "os1",
							CreatedAt: 0,
							UpdatedAt: 1,
							Data: vegapb.NewDataSourceDefinition(
								vegapb.DataSourceContentTypeOracle,
							).SetOracleConfig(
								&vegapb.DataSourceDefinitionExternal_Oracle{
									Oracle: &vegapb.DataSourceSpecConfiguration{
										Signers: []*datapb.Signer{pk.IntoProto()},
										Filters: []*datapb.Filter{},
									},
								},
							),
							Status: vegapb.DataSourceSpec_STATUS_ACTIVE,
						},
						DataSourceSpecBinding: &vegapb.DataSourceSpecToFutureBinding{
							SettlementDataProperty: "something",
						},
					},
				},
			},
			MarginCalculator: &vegapb.MarginCalculator{
				ScalingFactors: &vegapb.ScalingFactors{
					SearchLevel:       0.02,
					InitialMargin:     0.05,
					CollateralRelease: 0.1,
				},
				FullyCollateralised: &fPtr,
			},
			RiskModel: &vegapb.TradableInstrument_LogNormalRiskModel{
				LogNormalRiskModel: &vegapb.LogNormalRiskModel{
					RiskAversionParameter: 0.01,
					Tau:                   0.2,
					Params: &vegapb.LogNormalModelParams{
						Mu:    0.12323,
						R:     0.125,
						Sigma: 0.3,
					},
				},
			},
		},
		DecimalPlaces: 3,
		Fees: &vegapb.Fees{
			Factors: &vegapb.FeeFactors{
				MakerFee:          "0.002",
				InfrastructureFee: "0.001",
				LiquidityFee:      "0.003",
				BuyBackFee:        "0.1",
				TreasuryFee:       "0.2",
			},
			LiquidityFeeSettings: &vegapb.LiquidityFeeSettings{
				Method: vegapb.LiquidityFeeSettings_METHOD_WEIGHTED_AVERAGE,
			},
		},
		OpeningAuction: &vegapb.AuctionDuration{
			Duration: 1,
			Volume:   0,
		},
		PriceMonitoringSettings: &vegapb.PriceMonitoringSettings{
			Parameters: &vegapb.PriceMonitoringParameters{
				Triggers: []*vegapb.PriceMonitoringTrigger{
					{
						Horizon:          5,
						Probability:      "0.99",
						AuctionExtension: 4,
					},
					{
						Horizon:          10,
						Probability:      "0.95",
						AuctionExtension: 6,
					},
				},
			},
		},
		LiquidityMonitoringParameters: &vegapb.LiquidityMonitoringParameters{
			TargetStakeParameters: &vegapb.TargetStakeParameters{
				TimeWindow:    20,
				ScalingFactor: 0.7,
			},
		},
		TradingMode: vegapb.Market_TRADING_MODE_CONTINUOUS,
		State:       vegapb.Market_STATE_ACTIVE,
		MarketTimestamps: &vegapb.MarketTimestamps{
			Proposed: 0,
			Pending:  1,
			Open:     2,
			Close:    360,
		},
		LiquiditySlaParams: &vegapb.LiquiditySLAParameters{
			PriceRange:                  "0.95",
			CommitmentMinTimeFraction:   "0.5",
			PerformanceHysteresisEpochs: 4,
			SlaCompetitionFactor:        "0.5",
		},
		LinearSlippageFactor:    "0.1",
		QuadraticSlippageFactor: "0.1",
		MarkPriceConfiguration: &vegapb.CompositePriceConfiguration{
			DecayWeight:              "0.5",
			DecayPower:               1,
			CashAmount:               "0",
			CompositePriceType:       2,
			SourceWeights:            []string{"0.2", "0.3", "0.4", "0.5"},
			SourceStalenessTolerance: []string{"3h0m0s", "2s", "24h0m0s", "1h25m0s"},
		},
		TickSize:                    "1",
		EnableTransactionReordering: true,
	}

	domain, err := types.MarketFromProto(pMarket)
	require.NoError(t, err)

	// we can check equality of individual fields, but perhaps this is the easiest way:
	got := domain.IntoProto()
	require.EqualValues(t, pMarket, got)
}

func TestPerpMarketFromIntoProto(t *testing.T) {
	pk := dstypes.CreateSignerFromString("pubkey", dstypes.SignerTypePubKey)
	fPtr := false

	pMarket := &vegapb.Market{
		Id:       "foo",
		TickSize: "2",
		TradableInstrument: &vegapb.TradableInstrument{
			Instrument: &vegapb.Instrument{
				Id:   "bar",
				Code: "FB",
				Name: "FooBar",
				Metadata: &vegapb.InstrumentMetadata{
					Tags: []string{"test", "foo", "bar", "foobar"},
				},
				Product: &vegapb.Instrument_Perpetual{
					Perpetual: &vegapb.Perpetual{
						SettlementAsset:     "GBP",
						QuoteName:           "USD",
						MarginFundingFactor: "0.5",
						InterestRate:        "0.2",
						ClampLowerBound:     "0.1",
						ClampUpperBound:     "0.6",
						InternalCompositePriceConfig: &vegapb.CompositePriceConfiguration{
							DecayWeight:              "0.5",
							DecayPower:               1,
							CashAmount:               "0",
							CompositePriceType:       2,
							SourceWeights:            []string{"0.2", "0.3", "0.4", "0.5"},
							SourceStalenessTolerance: []string{"3h0m0s", "2s", "24h0m0s", "1h25m0s"},
						},
						DataSourceSpecForSettlementData: &vegapb.DataSourceSpec{
							Id:        "os1",
							CreatedAt: 0,
							UpdatedAt: 1,
							Data: vegapb.NewDataSourceDefinition(
								vegapb.DataSourceContentTypeOracle,
							).SetOracleConfig(
								&vegapb.DataSourceDefinitionExternal_Oracle{
									Oracle: &vegapb.DataSourceSpecConfiguration{
										Signers: []*datapb.Signer{pk.IntoProto()},
										Filters: []*datapb.Filter{testFilter1},
									},
								},
							),
							Status: vegapb.DataSourceSpec_STATUS_ACTIVE,
						},
						DataSourceSpecForSettlementSchedule: &vegapb.DataSourceSpec{
							Id:        "os1",
							CreatedAt: 0,
							UpdatedAt: 1,
							Data: vegapb.NewDataSourceDefinition(
								vegapb.DataSourceContentTypeOracle,
							).SetOracleConfig(
								&vegapb.DataSourceDefinitionExternal_Oracle{
									Oracle: &vegapb.DataSourceSpecConfiguration{
										Signers: []*datapb.Signer{pk.IntoProto()},
										Filters: []*datapb.Filter{testFilter1},
									},
								},
							),
							Status: vegapb.DataSourceSpec_STATUS_ACTIVE,
						},
						DataSourceSpecBinding: &vegapb.DataSourceSpecToPerpetualBinding{
							SettlementDataProperty: "something",
						},
					},
				},
			},
			MarginCalculator: &vegapb.MarginCalculator{
				ScalingFactors: &vegapb.ScalingFactors{
					SearchLevel:       0.02,
					InitialMargin:     0.05,
					CollateralRelease: 0.1,
				},
				FullyCollateralised: &fPtr,
			},
			RiskModel: &vegapb.TradableInstrument_LogNormalRiskModel{
				LogNormalRiskModel: &vegapb.LogNormalRiskModel{
					RiskAversionParameter: 0.01,
					Tau:                   0.2,
					Params: &vegapb.LogNormalModelParams{
						Mu:    0.12323,
						R:     0.125,
						Sigma: 0.3,
					},
				},
			},
		},
		DecimalPlaces: 3,
		Fees: &vegapb.Fees{
			Factors: &vegapb.FeeFactors{
				MakerFee:          "0.002",
				InfrastructureFee: "0.001",
				LiquidityFee:      "0.003",
				BuyBackFee:        "0.1",
				TreasuryFee:       "0.2",
			},
			LiquidityFeeSettings: &vegapb.LiquidityFeeSettings{
				Method: vegapb.LiquidityFeeSettings_METHOD_WEIGHTED_AVERAGE,
			},
		},
		OpeningAuction: &vegapb.AuctionDuration{
			Duration: 1,
			Volume:   0,
		},
		PriceMonitoringSettings: &vegapb.PriceMonitoringSettings{
			Parameters: &vegapb.PriceMonitoringParameters{
				Triggers: []*vegapb.PriceMonitoringTrigger{
					{
						Horizon:          5,
						Probability:      "0.99",
						AuctionExtension: 4,
					},
					{
						Horizon:          10,
						Probability:      "0.95",
						AuctionExtension: 6,
					},
				},
			},
		},
		LiquidityMonitoringParameters: &vegapb.LiquidityMonitoringParameters{
			TargetStakeParameters: &vegapb.TargetStakeParameters{
				TimeWindow:    20,
				ScalingFactor: 0.7,
			},
		},
		TradingMode: vegapb.Market_TRADING_MODE_CONTINUOUS,
		State:       vegapb.Market_STATE_ACTIVE,
		MarketTimestamps: &vegapb.MarketTimestamps{
			Proposed: 0,
			Pending:  1,
			Open:     2,
			Close:    360,
		},
		LiquiditySlaParams: &vegapb.LiquiditySLAParameters{
			PriceRange:                  "0.95",
			CommitmentMinTimeFraction:   "0.5",
			PerformanceHysteresisEpochs: 4,
			SlaCompetitionFactor:        "0.5",
		},
		LinearSlippageFactor:    "0.1",
		QuadraticSlippageFactor: "0.1",
		MarkPriceConfiguration: &vegapb.CompositePriceConfiguration{
			DecayWeight:              "0.7",
			DecayPower:               2,
			CashAmount:               "100",
			CompositePriceType:       3,
			SourceWeights:            []string{"0.5", "0.2", "0.3", "0.1"},
			SourceStalenessTolerance: []string{"3h0m1s", "3s", "25h0m0s", "2h25m0s"},
		},
	}

	domain, err := types.MarketFromProto(pMarket)
	require.NoError(t, err)

	// we can check equality of individual fields, but perhaps this is the easiest way:
	got := domain.IntoProto()

	require.EqualValues(t, pMarket.MarkPriceConfiguration, got.MarkPriceConfiguration)
	require.EqualValues(t, pMarket, got)
}
