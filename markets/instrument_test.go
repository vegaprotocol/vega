package markets_test

import (
	"testing"

	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/markets"
	"code.vegaprotocol.io/vega/oracles"
	"code.vegaprotocol.io/vega/products"
	types "code.vegaprotocol.io/vega/proto"
	oraclesv1 "code.vegaprotocol.io/vega/proto/oracles/v1"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInstrument(t *testing.T) {
	t.Run("Create a valid new instrument", func(t *testing.T) {
		pinst := getValidInstrumentProto()
		inst, err := markets.NewInstrument(logging.NewTestLogger(), pinst, newOracleEngine())
		assert.NotNil(t, inst)
		assert.Nil(t, err)
	})

	t.Run("Invalid future maturity", func(t *testing.T) {
		pinst := getValidInstrumentProto()
		pinst.Product = &types.Instrument_Future{
			Future: &types.Future{
				Maturity:        "notavaliddate",
				SettlementAsset: "Ethereum/Ether",
				OracleSpec: &oraclesv1.OracleSpecConfiguration{
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
		}
		inst, err := markets.NewInstrument(logging.NewTestLogger(), pinst, newOracleEngine())
		assert.Nil(t, inst)
		assert.NotNil(t, err)
	})

	t.Run("nil product", func(t *testing.T) {
		pinst := getValidInstrumentProto()
		pinst.Product = nil
		inst, err := markets.NewInstrument(logging.NewTestLogger(), pinst, newOracleEngine())
		assert.Nil(t, inst)
		assert.NotNil(t, err)
		assert.Equal(t, err.Error(), "unable to instantiate product from instrument configuration: nil product")
	})

	t.Run("nil oracle spec", func(t *testing.T) {
		pinst := getValidInstrumentProto()
		pinst.Product = &types.Instrument_Future{
			Future: &types.Future{
				Maturity:        "2019-12-31T00:00:00Z",
				SettlementAsset: "Ethereum/Ether",
				OracleSpec:      nil,
				OracleSpecBinding: &types.OracleSpecToFutureBinding{
					SettlementPriceProperty: "prices.ETH.value",
				},
			},
		}
		inst, err := markets.NewInstrument(logging.NewTestLogger(), pinst, newOracleEngine())
		require.NotNil(t, err)
		assert.Nil(t, inst)
		assert.Equal(t, "unable to instantiate product from instrument configuration: an oracle spec and an oracle spec binding are required", err.Error())
	})

	t.Run("nil oracle spec binding", func(t *testing.T) {
		pinst := getValidInstrumentProto()
		pinst.Product = &types.Instrument_Future{
			Future: &types.Future{
				Maturity:        "2019-12-31T00:00:00Z",
				SettlementAsset: "Ethereum/Ether",
				OracleSpec: &oraclesv1.OracleSpecConfiguration{
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
				OracleSpecBinding: nil,
			},
		}
		inst, err := markets.NewInstrument(logging.NewTestLogger(), pinst, newOracleEngine())
		require.NotNil(t, err)
		assert.Nil(t, inst)
		assert.Equal(t, "unable to instantiate product from instrument configuration: an oracle spec and an oracle spec binding are required", err.Error())
	})
}

func newOracleEngine() products.OracleEngine {
	return oracles.NewEngine(logging.NewTestLogger(), oracles.NewDefaultConfig())
}

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
				SettlementAsset: "Ethereum/Ether",
				OracleSpec: &oraclesv1.OracleSpecConfiguration{
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
