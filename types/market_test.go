package types_test

import (
	"testing"

	"code.vegaprotocol.io/vega/proto"
	v1 "code.vegaprotocol.io/vega/proto/oracles/v1"
	"code.vegaprotocol.io/vega/types"
	"github.com/stretchr/testify/require"
)

func TestMarketFromIntoProto(t *testing.T) {
	filter := &v1.Filter{
		Key: &v1.PropertyKey{
			Name: "filter1",
			Type: v1.PropertyKey_TYPE_STRING,
		},
		Conditions: []*v1.Condition{
			{
				Operator: v1.Condition_OPERATOR_EQUALS,
				Value:    "true",
			},
		},
	}
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
						Maturity:        "very",
						SettlementAsset: "GBP",
						QuoteName:       "USD",
						OracleSpec: &v1.OracleSpec{
							Id:        "os1",
							CreatedAt: 0,
							UpdatedAt: 1,
							PubKeys:   []string{"pubkey"},
							Filters:   []*v1.Filter{filter},
							Status:    v1.OracleSpec_STATUS_ACTIVE,
						},
						OracleSpecBinding: &proto.OracleSpecToFutureBinding{
							SettlementPriceProperty: "something",
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
		TradingModeConfig: &proto.Market_Continuous{
			Continuous: &proto.ContinuousTrading{
				TickSize: "1",
			},
		},
		PriceMonitoringSettings: &proto.PriceMonitoringSettings{
			Parameters: &proto.PriceMonitoringParameters{
				Triggers: []*proto.PriceMonitoringTrigger{
					{
						Horizon:          5,
						Probability:      .99,
						AuctionExtension: 4,
					},
					{
						Horizon:          10,
						Probability:      .95,
						AuctionExtension: 6,
					},
				},
			},
			UpdateFrequency: 20,
		},
		LiquidityMonitoringParameters: &proto.LiquidityMonitoringParameters{
			TargetStakeParameters: &proto.TargetStakeParameters{
				TimeWindow:    20,
				ScalingFactor: 0.7,
			},
			TriggeringRatio:  0.8,
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
	}
	domain := types.MarketFromProto(pMarket)
	// we can check equality of individual fields, but perhaps this is the easiest way:
	got := domain.IntoProto()
	require.EqualValues(t, pMarket, got)
}
