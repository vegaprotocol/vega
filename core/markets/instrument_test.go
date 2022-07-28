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

package markets_test

import (
	"context"
	"testing"

	"code.vegaprotocol.io/vega/core/broker/mocks"
	emock "code.vegaprotocol.io/vega/core/execution/mocks"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/core/markets"
	"code.vegaprotocol.io/vega/core/oracles"

	oraclespb "code.vegaprotocol.io/protos/vega/oracles/v1"
	"code.vegaprotocol.io/vega/core/products"
	"code.vegaprotocol.io/vega/core/types"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInstrument(t *testing.T) {
	t.Run("Create a valid new instrument", func(t *testing.T) {
		pinst := getValidInstrumentProto()
		inst, err := markets.NewInstrument(context.Background(), logging.NewTestLogger(), pinst, newOracleEngine(t))
		assert.NotNil(t, inst)
		assert.Nil(t, err)
	})

	t.Run("nil product", func(t *testing.T) {
		pinst := getValidInstrumentProto()
		pinst.Product = nil
		inst, err := markets.NewInstrument(context.Background(), logging.NewTestLogger(), pinst, newOracleEngine(t))
		assert.Nil(t, inst)
		assert.NotNil(t, err)
		assert.Equal(t, err.Error(), "unable to instantiate product from instrument configuration: nil product")
	})

	t.Run("nil oracle spec", func(t *testing.T) {
		pinst := getValidInstrumentProto()
		pinst.Product = &types.InstrumentFuture{
			Future: &types.Future{
				SettlementAsset:                 "Ethereum/Ether",
				OracleSpecForSettlementPrice:    nil,
				OracleSpecForTradingTermination: nil,
				OracleSpecBinding: &types.OracleSpecBindingForFuture{
					SettlementPriceProperty:    "prices.ETH.value",
					TradingTerminationProperty: "trading.terminated",
				},
			},
		}
		inst, err := markets.NewInstrument(context.Background(), logging.NewTestLogger(), pinst, newOracleEngine(t))
		require.NotNil(t, err)
		assert.Nil(t, inst)
		assert.Equal(t, "unable to instantiate product from instrument configuration: an oracle spec and an oracle spec binding are required", err.Error())
	})

	t.Run("nil oracle spec binding", func(t *testing.T) {
		pinst := getValidInstrumentProto()
		pinst.Product = &types.InstrumentFuture{
			Future: &types.Future{
				SettlementAsset: "Ethereum/Ether",
				OracleSpecForSettlementPrice: &types.OracleSpec{
					PubKeys: []string{"0xDEADBEEF"},
					Filters: []*types.OracleSpecFilter{
						{
							Key: &types.OracleSpecPropertyKey{
								Name: "prices.ETH.value",
								Type: oraclespb.PropertyKey_TYPE_INTEGER,
							},
							Conditions: []*types.OracleSpecCondition{},
						},
					},
				},
				OracleSpecForTradingTermination: &types.OracleSpec{
					PubKeys: []string{"0xDEADBEEF"},
					Filters: []*types.OracleSpecFilter{
						{
							Key: &types.OracleSpecPropertyKey{
								Name: "trading.terminated",
								Type: oraclespb.PropertyKey_TYPE_BOOLEAN,
							},
							Conditions: []*types.OracleSpecCondition{},
						},
					},
				},
				OracleSpecBinding: nil,
			},
		}
		inst, err := markets.NewInstrument(context.Background(), logging.NewTestLogger(), pinst, newOracleEngine(t))
		require.NotNil(t, err)
		assert.Nil(t, inst)
		assert.Equal(t, "unable to instantiate product from instrument configuration: an oracle spec and an oracle spec binding are required", err.Error())
	})
}

func newOracleEngine(t *testing.T) products.OracleEngine {
	t.Helper()
	ctrl := gomock.NewController(t)
	broker := mocks.NewMockBroker(ctrl)
	broker.EXPECT().Send(gomock.Any()).AnyTimes()

	ts := emock.NewMockTimeService(ctrl)
	ts.EXPECT().GetTimeNow().AnyTimes()

	return oracles.NewEngine(
		logging.NewTestLogger(),
		oracles.NewDefaultConfig(),
		ts,
		broker,
	)
}

func getValidInstrumentProto() *types.Instrument {
	return &types.Instrument{
		ID:   "Crypto/BTCUSD/Futures/Dec19",
		Code: "FX:BTCUSD/DEC19",
		Name: "December 2019 BTC vs USD future",
		Metadata: &types.InstrumentMetadata{
			Tags: []string{
				"asset_class:fx/crypto",
				"product:futures",
			},
		},
		Product: &types.InstrumentFuture{
			Future: &types.Future{
				QuoteName:       "USD",
				SettlementAsset: "Ethereum/Ether",
				OracleSpecForSettlementPrice: &types.OracleSpec{
					PubKeys: []string{"0xDEADBEEF"},
					Filters: []*types.OracleSpecFilter{
						{
							Key: &types.OracleSpecPropertyKey{
								Name: "prices.ETH.value",
								Type: oraclespb.PropertyKey_TYPE_INTEGER,
							},
							Conditions: []*types.OracleSpecCondition{},
						},
					},
				},
				OracleSpecForTradingTermination: &types.OracleSpec{
					PubKeys: []string{"0xDEADBEEF"},
					Filters: []*types.OracleSpecFilter{
						{
							Key: &types.OracleSpecPropertyKey{
								Name: "trading.terminated",
								Type: oraclespb.PropertyKey_TYPE_BOOLEAN,
							},
							Conditions: []*types.OracleSpecCondition{},
						},
					},
				},
				OracleSpecBinding: &types.OracleSpecBindingForFuture{
					SettlementPriceProperty:    "prices.ETH.value",
					TradingTerminationProperty: "trading.terminated",
				},
			},
		},
	}
}
