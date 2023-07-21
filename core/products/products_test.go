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

	"code.vegaprotocol.io/vega/core/oracles"
	"code.vegaprotocol.io/vega/core/products"
	"code.vegaprotocol.io/vega/core/products/mocks"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
	datapb "code.vegaprotocol.io/vega/protos/vega/data/v1"
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
				DataSourceSpecForSettlementData: &types.DataSourceSpec{
					Data: types.NewDataSourceDefinition(
						vegapb.DataSourceDefinitionTypeExt,
					).SetOracleConfig(
						&types.DataSourceSpecConfiguration{
							Signers: []*types.Signer{
								types.CreateSignerFromString("0xDEADBEEF", types.DataSignerTypePubKey),
							},
							Filters: []*types.DataSourceSpecFilter{
								{
									Key: &types.DataSourceSpecPropertyKey{
										Name: "prices.ETH.value",
										Type: datapb.PropertyKey_TYPE_INTEGER,
									},
									Conditions: []*types.DataSourceSpecCondition{},
								},
							},
						},
					),
				},
				DataSourceSpecForTradingTermination: &types.DataSourceSpec{
					Data: types.NewDataSourceDefinition(
						vegapb.DataSourceDefinitionTypeExt,
					).SetOracleConfig(
						&types.DataSourceSpecConfiguration{
							Signers: []*types.Signer{
								types.CreateSignerFromString("0xDEADBEEF", types.DataSignerTypePubKey),
							},
							Filters: []*types.DataSourceSpecFilter{
								{
									Key: &types.DataSourceSpecPropertyKey{
										Name: "trading.terminated",
										Type: datapb.PropertyKey_TYPE_BOOLEAN,
									},
									Conditions: []*types.DataSourceSpecCondition{},
								},
							},
						},
					),
				},
				DataSourceSpecBinding: &types.DataSourceSpecBindingForFuture{
					SettlementDataProperty:     "prices.ETH.value",
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
		entryPrice     uint64
		settlementData uint64
		position       int64
		result         int64
	}{
		{100, 200, 10, 1000},  // (200-100)*10 == 1000
		{200, 100, 10, 1000},  // (100-200)*10 == 1000
		{100, 200, -10, 1000}, // (200-100)*-10 == 1000
		{200, 100, -10, 1000}, // (100-200)*-10 == 1000
	}

	for _, param := range params {
		n := &num.Numeric{}
		n.SetUint(num.NewUint(param.settlementData))
		// Use debug function to update the settlement data as if from a Oracle
		f.SetSettlementData(ctx, "prices.ETH.value", n)
		ep := num.NewUint(param.entryPrice)
		fa, _, _, err := prod.Settle(ep, 0, num.DecimalFromInt64(param.position))
		assert.NoError(t, err)
		assert.EqualValues(t, param.result, fa.Amount.Uint64())
	}
}
