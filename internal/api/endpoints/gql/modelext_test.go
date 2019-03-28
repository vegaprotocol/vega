package gql

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestModelConverters(t *testing.T) {

	t.Run("DiscreteTrading.IntoProto nil duration", func(t *testing.T) {
		dt := &DiscreteTrading{}
		pdt, err := dt.IntoProto()
		assert.Nil(t, pdt)
		assert.NotNil(t, err)
		assert.Equal(t, ErrNilDiscreteTradingDuration, err)
	})
	t.Run("DiscreteTrading.IntoProto", func(t *testing.T) {

		dt := &DiscreteTrading{Duration: intptr(123)}
		pdt, err := dt.IntoProto()
		assert.NotNil(t, pdt)
		assert.Nil(t, err)
		assert.Equal(t, int64(*dt.Duration), pdt.Discrete.Duration)
	})

	t.Run("Future.IntoProto nil oracle", func(t *testing.T) {
		f := &Future{Maturity: "12/31/19", Asset: "Ethereum/Ether"}
		pf, err := f.IntoProto()
		assert.Nil(t, pf)
		assert.NotNil(t, err)
		assert.Equal(t, ErrNilOracle, err)
	})

	t.Run("Future.IntoProto", func(t *testing.T) {
		f := &Future{
			Maturity: "12/31/19",
			Asset:    "Ethereum/Ether",
			Oracle: &EthereumEvent{
				ContractID: "asdas",
				Event:      "aerasd",
			},
		}
		pf, err := f.IntoProto()
		assert.NotNil(t, pf)
		assert.Nil(t, err)
	})

	t.Run("InstrumentMetadata.IntoProto", func(t *testing.T) {
		im := InstrumentMetadata{Tags: []*string{stringptr("tag:1"), stringptr("tag:2")}}
		pim, err := im.IntoProto()
		assert.Nil(t, err)
		assert.NotNil(t, pim)
		assert.NotNil(t, pim.Tags)
		assert.Len(t, pim.Tags, 2)
	})

	t.Run("Instrument.IntoProto nil product", func(t *testing.T) {
		i := Instrument{}
		pi, err := i.IntoProto()
		assert.Nil(t, pi)
		assert.NotNil(t, err)
		assert.Equal(t, ErrNilProduct, err)
	})

	t.Run("Instrument.IntoProto ", func(t *testing.T) {
		i := Instrument{Product: &Future{
			Maturity: "asdasdas",
			Asset:    "Ethereum/Ether",
			Oracle: &EthereumEvent{
				ContractID: "asdas",
				Event:      "aerasd",
			},
		}}
		pi, err := i.IntoProto()
		assert.NotNil(t, pi)
		assert.Nil(t, err)
	})

	t.Run("TradableInstrument.IntoProto nil inners types", func(t *testing.T) {
		ti := TradableInstrument{}
		pti, err := ti.IntoProto()
		assert.Nil(t, pti)
		assert.NotNil(t, err)
		assert.Equal(t, ErrNilProduct, err)

		ti.Instrument.Product = &Future{
			Maturity: "asdasdas",
			Asset:    "Ethereum/Ether",
			Oracle: &EthereumEvent{
				ContractID: "asdas",
				Event:      "aerasd",
			}}
		pti, err = ti.IntoProto()
		assert.Nil(t, pti)
		assert.NotNil(t, err)
		assert.Equal(t, ErrNilRiskModel, err)
	})

	t.Run("TradableIstrument.IntoProto", func(t *testing.T) {
		ti := TradableInstrument{
			Instrument: Instrumment{
				Product: &Future{
					Maturity: "asdasdas",
					Asset:    "Ethereum/Ether",
					Oracle: &EthereumEvent{
						ContractID: "asdas",
						Event:      "aerasd",
					},
				},
			},
			RiskModel: &BuiltinFutures{
				HistoricVolatility: 42.42,
			},
		}
	})
}

func intptr(i int) *int {
	return &i
}

func stringptr(s string) *string {
	return &s
}
