package markets_test

import (
	"testing"

	"code.vegaprotocol.io/vega/markets"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/stretchr/testify/assert"
)

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
				Oracle: &types.Future_EthereumEvent{
					EthereumEvent: &types.EthereumEvent{
						ContractID: "0x0B484706fdAF3A4F24b2266446B1cb6d648E3cC1",
						Event:      "price_changed",
					},
				},
			},
		},
	}
}

func TestInstrument(t *testing.T) {
	t.Run("Create a valid new instrument", func(t *testing.T) {
		pinst := getValidInstrumentProto()
		inst, err := markets.NewInstrument(pinst)
		assert.NotNil(t, inst)
		assert.Nil(t, err)
	})

	t.Run("Invalid future maturity", func(t *testing.T) {
		pinst := getValidInstrumentProto()
		pinst.Product = &types.Instrument_Future{
			Future: &types.Future{
				Maturity:        "notavaliddate",
				SettlementAsset: "Ethereum/Ether",
				Oracle: &types.Future_EthereumEvent{
					EthereumEvent: &types.EthereumEvent{
						ContractID: "0x0B484706fdAF3A4F24b2266446B1cb6d648E3cC1",
						Event:      "price_changed",
					},
				},
			},
		}
		inst, err := markets.NewInstrument(pinst)
		assert.Nil(t, inst)
		assert.NotNil(t, err)
	})

	t.Run("nil product", func(t *testing.T) {
		pinst := getValidInstrumentProto()
		pinst.Product = nil
		inst, err := markets.NewInstrument(pinst)
		assert.Nil(t, inst)
		assert.NotNil(t, err)
		assert.Equal(t, err.Error(), "unable to instantiate product from instrument configuration: nil product")
	})

	t.Run("nil oracle", func(t *testing.T) {
		pinst := getValidInstrumentProto()
		pinst.Product = &types.Instrument_Future{
			Future: &types.Future{
				Maturity:        "2019-12-31T00:00:00Z",
				SettlementAsset: "Ethereum/Ether",
				Oracle:          nil,
			},
		}
		inst, err := markets.NewInstrument(pinst)
		assert.Nil(t, inst)
		assert.NotNil(t, err)
		assert.Equal(t, err.Error(), "unable to instantiate product from instrument configuration: nil oracle")
	})

}
