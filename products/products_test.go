package products_test

import (
	"context"
	"testing"

	oraclesv1 "code.vegaprotocol.io/protos/vega/oracles/v1"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/products"
	"code.vegaprotocol.io/vega/products/mocks"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
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
		Product: &types.Instrument_Future{
			Future: &types.Future{
				QuoteName:       "USD",
				Maturity:        "2019-12-31T00:00:00Z",
				SettlementAsset: SettlementAssetStr,
				OracleSpecForSettlementPrice: &oraclesv1.OracleSpec{
					PubKeys: []string{"0xDEADBEEF"},
					Filters: []*oraclesv1.Filter{
						{
							Key: &oraclesv1.PropertyKey{
								Name: "prices.ETH.value",
								Type: oraclesv1.PropertyKey_TYPE_INTEGER,
							},
							Conditions: []*oraclesv1.Condition{},
						},
					},
				},
				OracleSpecForTradingTermination: &oraclesv1.OracleSpec{
					PubKeys: []string{"0xDEADBEEF"},
					Filters: []*oraclesv1.Filter{
						{
							Key: &oraclesv1.PropertyKey{
								Name: "trading.terminated",
								Type: oraclesv1.PropertyKey_TYPE_BOOLEAN,
							},
							Conditions: []*oraclesv1.Condition{},
						},
					},
				},
				OracleSpecBinding: &types.OracleSpecToFutureBinding{
					SettlementPriceProperty:    "prices.ETH.value",
					TradingTerminationProperty: "trading.terminated",
				},
			},
		},
	}
}

func TestFuture(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	oe := mocks.NewMockOracleEngine(ctrl)

	oe.EXPECT().Subscribe(ctx, gomock.Any(), gomock.Any()).AnyTimes()

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
		fa, _, err := prod.Settle(ep, num.DecimalFromInt64(param.position))
		assert.NoError(t, err)
		assert.EqualValues(t, param.result, fa.Amount.Uint64())
	}
}
