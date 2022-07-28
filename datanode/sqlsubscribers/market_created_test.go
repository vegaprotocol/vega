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

package sqlsubscribers_test

import (
	"context"
	"testing"

	"code.vegaprotocol.io/data-node/datanode/sqlsubscribers"
	"code.vegaprotocol.io/data-node/datanode/sqlsubscribers/mocks"
	"code.vegaprotocol.io/data-node/logging"
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

	store.EXPECT().Upsert(context.Background(), gomock.Any()).Times(1)
	subscriber := sqlsubscribers.NewMarketCreated(store, logging.NewTestLogger())
	subscriber.Flush(context.Background())
	subscriber.Push(context.Background(), events.NewMarketCreatedEvent(context.Background(), getTestMarket()))
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
				Product: types.InstrumentFuture{
					Future: &types.Future{
						SettlementAsset: "",
						QuoteName:       "",
						OracleSpecForSettlementPrice: &types.OracleSpec{
							ID:        "",
							CreatedAt: 0,
							UpdatedAt: 0,
							PubKeys:   nil,
							Filters:   nil,
							Status:    0,
						},
						OracleSpecForTradingTermination: &types.OracleSpec{
							ID:        "",
							CreatedAt: 0,
							UpdatedAt: 0,
							PubKeys:   nil,
							Filters:   nil,
							Status:    0,
						},
						OracleSpecBinding: &types.OracleSpecBindingForFuture{
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
		OpeningAuction: nil,
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
