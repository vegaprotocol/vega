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

package types_test

import (
	"testing"
	"time"

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
			TriggeringRatio:  "0.8",
			AuctionExtension: 5,
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
			PriceRange:                      "0.95",
			CommitmentMinTimeFraction:       "0.5",
			ProvidersFeeCalculationTimeStep: (5 * time.Second).Nanoseconds(),
			PerformanceHysteresisEpochs:     4,
			SlaCompetitionFactor:            "0.5",
		},
		LinearSlippageFactor:    "0.1",
		QuadraticSlippageFactor: "0.1",
	}

	domain, err := types.MarketFromProto(pMarket)
	require.NoError(t, err)

	// we can check equality of individual fields, but perhaps this is the easiest way:
	got := domain.IntoProto()
	require.EqualValues(t, pMarket, got)
}
