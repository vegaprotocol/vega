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

	"code.vegaprotocol.io/vega/core/types"
	proto "code.vegaprotocol.io/vega/protos/vega"
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
	pk := types.CreateSignerFromString("pubkey", types.DataSignerTypePubKey)

	pMarket := &proto.Market{
		Id: "foo",
		TradableInstrument: &proto.TradableInstrument{
			Instrument: &proto.Instrument{
				Id:   "bar",
				Code: "FB",
				Name: "FooBar",
				Metadata: &proto.InstrumentMetadata{
					Tags: []string{"test", "foo", "bar", "foobar"},
				},
				Product: &proto.Instrument_Future{
					Future: &proto.Future{
						SettlementAsset: "GBP",
						QuoteName:       "USD",
						DataSourceSpecForSettlementData: &vegapb.DataSourceSpec{
							Id:        "os1",
							CreatedAt: 0,
							UpdatedAt: 1,
							Data: vegapb.NewDataSourceDefinition(
								vegapb.DataSourceDefinitionTypeExt,
							).SetOracleConfig(
								&vegapb.DataSourceSpecConfiguration{
									Signers: []*datapb.Signer{pk.IntoProto()},
									Filters: []*datapb.Filter{testFilter1},
								},
							),
							Status: vegapb.DataSourceSpec_STATUS_ACTIVE,
						},
						DataSourceSpecForTradingTermination: &vegapb.DataSourceSpec{
							Id:        "os1",
							CreatedAt: 0,
							UpdatedAt: 1,
							Data: vegapb.NewDataSourceDefinition(
								vegapb.DataSourceDefinitionTypeExt,
							).SetOracleConfig(
								&vegapb.DataSourceSpecConfiguration{
									Signers: []*datapb.Signer{pk.IntoProto()},
									Filters: []*datapb.Filter{},
								},
							),
							Status: vegapb.DataSourceSpec_STATUS_ACTIVE,
						},
						DataSourceSpecBinding: &proto.DataSourceSpecToFutureBinding{
							SettlementDataProperty: "something",
						},
					},
				},
			},
			MarginCalculator: &proto.MarginCalculator{
				ScalingFactors: &proto.ScalingFactors{
					SearchLevel:       0.02,
					InitialMargin:     0.05,
					CollateralRelease: 0.1,
				},
			},
			RiskModel: &proto.TradableInstrument_LogNormalRiskModel{
				LogNormalRiskModel: &proto.LogNormalRiskModel{
					RiskAversionParameter: 0.01,
					Tau:                   0.2,
					Params: &proto.LogNormalModelParams{
						Mu:    0.12323,
						R:     0.125,
						Sigma: 0.3,
					},
				},
			},
		},
		DecimalPlaces: 3,
		Fees: &proto.Fees{
			Factors: &proto.FeeFactors{
				MakerFee:          "0.002",
				InfrastructureFee: "0.001",
				LiquidityFee:      "0.003",
			},
		},
		OpeningAuction: &proto.AuctionDuration{
			Duration: 1,
			Volume:   0,
		},
		PriceMonitoringSettings: &proto.PriceMonitoringSettings{
			Parameters: &proto.PriceMonitoringParameters{
				Triggers: []*proto.PriceMonitoringTrigger{
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
		LiquidityMonitoringParameters: &proto.LiquidityMonitoringParameters{
			TargetStakeParameters: &proto.TargetStakeParameters{
				TimeWindow:    20,
				ScalingFactor: 0.7,
			},
			TriggeringRatio:  "0.8",
			AuctionExtension: 5,
		},
		TradingMode: proto.Market_TRADING_MODE_CONTINUOUS,
		State:       proto.Market_STATE_ACTIVE,
		MarketTimestamps: &proto.MarketTimestamps{
			Proposed: 0,
			Pending:  1,
			Open:     2,
			Close:    360,
		},
		LpPriceRange: "0.95",
	}

	domain, err := types.MarketFromProto(pMarket)
	require.NoError(t, err)

	// we can check equality of individual fields, but perhaps this is the easiest way:
	got := domain.IntoProto()
	require.EqualValues(t, pMarket, got)
}
