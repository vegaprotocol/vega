package products_test

import (
	"context"
	"testing"

	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/products"
	"code.vegaprotocol.io/vega/products/mocks"
	types "code.vegaprotocol.io/vega/proto"
	oraclesv1 "code.vegaprotocol.io/vega/proto/oracles/v1"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const SettlementAssetStr = "Ethereum/Ether"

func getValidInstrumentProto() *types.Instrument {
	return &types.Instrument{
		Id:   "Crypto/BTCUSD/Futures/Dec19",
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
				OracleSpec: &oraclesv1.OracleSpec{
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
				OracleSpecBinding: &types.OracleSpecToFutureBinding{
					SettlementPriceProperty: "prices.ETH.value",
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

	prodSpec := proto.GetProduct()
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
	value, err := prod.Value(1000)
	assert.NoError(t, err)
	assert.EqualValues(t, 1000, value)

	var params = []struct {
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
		fa, err := prod.Settle(param.entryPrice, param.position)
		assert.NoError(t, err)
		assert.EqualValues(t, param.result, fa.Amount)
	}
}
