package sqlsubscribers_test

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/data-node/sqlsubscribers"
	"code.vegaprotocol.io/data-node/sqlsubscribers/mocks"
	v1 "code.vegaprotocol.io/protos/vega/oracles/v1"
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
	"github.com/golang/mock/gomock"
)

func Test_MarketCreated_Push(t *testing.T) {
	t.Run("MarketCreatedEvent should call market SQL store Add", shouldCallMarketSQLStoreAdd)
}

func shouldCallMarketSQLStoreAdd(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	store := mocks.NewMockMarketsStore(ctrl)

	store.EXPECT().Upsert(gomock.Any()).Times(1)
	subscriber := sqlsubscribers.NewMarketCreated(store, logging.NewTestLogger())
	subscriber.Push(events.NewTime(context.Background(), time.Now()))
	subscriber.Push(events.NewMarketCreatedEvent(context.Background(), getTestMarket()))
}

func getTestMarket() types.Market {
	return types.Market{
		ID: "DEADBEEF",
		TradableInstrument: &types.TradableInstrument{
			Instrument: &types.Instrument{
				ID:   "TEST_INSTRUMENT",
				Code: "TEST",
				Name: "Test Instrument",
				Metadata: &types.InstrumentMetadata{
					Tags: []string{"AAA", "BBB"},
				},
				Product: types.Instrument_Future{
					Future: &types.Future{
						Maturity:        "",
						SettlementAsset: "",
						QuoteName:       "",
						OracleSpecForSettlementPrice: &v1.OracleSpec{
							Id:        "",
							CreatedAt: 0,
							UpdatedAt: 0,
							PubKeys:   nil,
							Filters:   nil,
							Status:    0,
						},
						OracleSpecForTradingTermination: &v1.OracleSpec{
							Id:        "",
							CreatedAt: 0,
							UpdatedAt: 0,
							PubKeys:   nil,
							Filters:   nil,
							Status:    0,
						},
						OracleSpecBinding: &types.OracleSpecToFutureBinding{
							SettlementPriceProperty:    "",
							TradingTerminationProperty: "",
						},
					},
				},
			},
			MarginCalculator: &types.MarginCalculator{
				ScalingFactors: &types.ScalingFactors{
					SearchLevel:       num.DecimalZero(),
					InitialMargin:     num.DecimalZero(),
					CollateralRelease: num.DecimalZero(),
				},
			},
			RiskModel: &types.TradableInstrumentSimpleRiskModel{
				SimpleRiskModel: &types.SimpleRiskModel{
					Params: &types.SimpleModelParams{
						FactorLong:           num.DecimalZero(),
						FactorShort:          num.DecimalZero(),
						MaxMoveUp:            num.DecimalZero(),
						MinMoveDown:          num.DecimalZero(),
						ProbabilityOfTrading: num.DecimalZero(),
					},
				},
			},
		},
		DecimalPlaces:         16,
		PositionDecimalPlaces: 8,
		Fees: &types.Fees{
			Factors: &types.FeeFactors{
				MakerFee:          num.DecimalZero(),
				InfrastructureFee: num.DecimalZero(),
				LiquidityFee:      num.DecimalZero(),
			},
		},
		OpeningAuction:    nil,
		TradingModeConfig: nil,
		PriceMonitoringSettings: &types.PriceMonitoringSettings{
			Parameters: &types.PriceMonitoringParameters{
				Triggers: []*types.PriceMonitoringTrigger{
					{
						Horizon:          0,
						HorizonDec:       num.DecimalZero(),
						Probability:      num.NewDecimalFromFloat(0.99),
						AuctionExtension: 0,
					},
				},
			},
			UpdateFrequency: 0,
		},
		LiquidityMonitoringParameters: &types.LiquidityMonitoringParameters{
			TargetStakeParameters: &types.TargetStakeParameters{
				TimeWindow:    0,
				ScalingFactor: num.DecimalZero(),
			},
			TriggeringRatio:  num.DecimalZero(),
			AuctionExtension: 0,
		},
		TradingMode: 0,
		State:       0,
		MarketTimestamps: &types.MarketTimestamps{
			Proposed: 0,
			Pending:  0,
			Open:     0,
			Close:    0,
		},
	}
}
