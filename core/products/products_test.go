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

package products_test

import (
	"context"
	"testing"

	oraclespb "code.vegaprotocol.io/protos/vega/oracles/v1"
	"code.vegaprotocol.io/vega/core/oracles"
	"code.vegaprotocol.io/vega/core/products"
	"code.vegaprotocol.io/vega/core/products/mocks"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const SettlementAssetStr = "Ethereum/Ether"

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
				SettlementAsset: SettlementAssetStr,
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

func TestFutureSettlement(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	oe := mocks.NewMockOracleEngine(ctrl)

	sid1 := oracles.SubscriptionID(1)
	oe.EXPECT().Unsubscribe(ctx, sid1).AnyTimes()
	oe.EXPECT().
		Subscribe(ctx, gomock.Any(), gomock.Any()).
		Times(2).
		Return(sid1, func(ctx context.Context, sid oracles.SubscriptionID) {
			oe.Unsubscribe(ctx, sid)
		})

	proto := getValidInstrumentProto()

	prodSpec := proto.Product
	require.NotNil(t, prodSpec)
	prod, err := products.New(ctx, logging.NewTestLogger(), prodSpec, oe)

	// Cast back into a future so we can call future specific functions
	f, ok := prod.(*products.Future)
	require.True(t, ok)
	require.NotNil(t, prod)
	require.NoError(t, err)

	// Check the assert string is correct
	assert.Equal(t, SettlementAssetStr, prod.GetAsset())

	// Future values are the same as the mark price
	given := num.NewUint(1000)
	value, err := prod.Value(given)
	assert.NoError(t, err)
	assert.EqualValues(t, given.String(), value.String())

	params := []struct {
		entryPrice      uint64
		settlementPrice uint64
		position        int64
		result          int64
	}{
		{100, 200, 10, 1000},  // (200-100)*10 == 1000
		{200, 100, 10, 1000},  // (100-200)*10 == 1000
		{100, 200, -10, 1000}, // (200-100)*-10 == 1000
		{200, 100, -10, 1000}, // (100-200)*-10 == 1000
	}

	for _, param := range params {
		// Use debug function to update the settlement price as if from a Oracle
		f.SetSettlementPrice(ctx, "prices.ETH.value", param.settlementPrice)
		ep := num.NewUint(param.entryPrice)
		fa, _, err := prod.Settle(ep, 0, num.DecimalFromInt64(param.position))
		assert.NoError(t, err)
		assert.EqualValues(t, param.result, fa.Amount.Uint64())
	}
}
