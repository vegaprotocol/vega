package gql

import (
	"testing"

	"code.vegaprotocol.io/vega/proto"
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
			Instrument: Instrument{
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
		pti, err := ti.IntoProto()
		assert.NotNil(t, pti)
		assert.Nil(t, err)
	})

	t.Run("Market.IntoProto", func(t *testing.T) {
		mkt := Market{
			TradingMode: &ContinuousTrading{TickSize: intptr(123)},
			TradableInstrument: TradableInstrument{
				Instrument: Instrument{
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
			},
		}
		pmkt, err := mkt.IntoProto()
		assert.NotNil(t, pmkt)
		assert.Nil(t, err)
	})

	t.Run("TradingModeFromProto unimplemented", func(t *testing.T) {
		ptm := int(0)
		tm, err := TradingModeFromProto(ptm)
		assert.Nil(t, tm)
		assert.NotNil(t, err)
		assert.Equal(t, ErrUnimplementedTradingMode, err)
	})

	t.Run("TradingModeFromProto nil", func(t *testing.T) {
		tm, err := TradingModeFromProto(nil)
		assert.Nil(t, tm)
		assert.NotNil(t, err)
		assert.Equal(t, ErrNilTradingMode, err)
	})

	t.Run("TradingModeFromProto Continuous", func(t *testing.T) {
		ptm := &proto.Market_Continuous{
			Continuous: &proto.ContinuousTrading{
				TickSize: 42,
			},
		}
		tm, err := TradingModeFromProto(ptm)
		assert.NotNil(t, tm)
		assert.Nil(t, err)
		_, ok := tm.(*ContinuousTrading)
		assert.True(t, ok)
	})

	t.Run("TradingModeFromProto Discrete", func(t *testing.T) {
		ptm := &proto.Market_Discrete{
			Discrete: &proto.DiscreteTrading{
				Duration: 42,
			},
		}
		tm, err := TradingModeFromProto(ptm)
		assert.NotNil(t, tm)
		assert.Nil(t, err)
		_, ok := tm.(*DiscreteTrading)
		assert.True(t, ok)
	})

	t.Run("InstrumentMetadataFromProto nil", func(t *testing.T) {
		im, err := InstrumentMetadataFromProto(nil)
		assert.Nil(t, im)
		assert.NotNil(t, err)
		assert.Equal(t, ErrNilInstrumentMetadata, err)
	})

	t.Run("InstrumentMetadataFromProto", func(t *testing.T) {
		pim := &proto.InstrumentMetadata{
			Tags: []string{"tag:1", "tag:2"},
		}
		im, err := InstrumentMetadataFromProto(pim)
		assert.NotNil(t, im)
		assert.Nil(t, err)
		assert.Len(t, im.Tags, 2)
		assert.Equal(t, pim.Tags[0], *(im.Tags[0]))
		assert.Equal(t, pim.Tags[1], *(im.Tags[1]))
	})

	t.Run("EthereumEventFromproto nil", func(t *testing.T) {
		ee, err := EthereumEventFromProto(nil)
		assert.Nil(t, ee)
		assert.NotNil(t, err)
		assert.Equal(t, err, ErrNilEthereumEvent)
	})

	t.Run("EthereumEventFromproto", func(t *testing.T) {
		pee := &proto.EthereumEvent{}
		ee, err := EthereumEventFromProto(pee)
		assert.NotNil(t, ee)
		assert.Nil(t, err)
	})

	t.Run("OracleFromProto nil", func(t *testing.T) {
		o, err := OracleFromProto(nil)
		assert.Nil(t, o)
		assert.NotNil(t, err)
		assert.Equal(t, err, ErrNilOracle)
	})

	t.Run("OracleFromProto unimplemented", func(t *testing.T) {
		o, err := OracleFromProto(struct{}{})
		assert.Nil(t, o)
		assert.NotNil(t, err)
		assert.Equal(t, err, ErrUnimplementedOracle)
	})

	t.Run("OracleFromProto EthereumEvent", func(t *testing.T) {
		po := &proto.Future_EthereumEvent{
			EthereumEvent: &proto.EthereumEvent{},
		}
		o, err := OracleFromProto(po)
		assert.NotNil(t, o)
		assert.Nil(t, err)
		_, ok := o.(*EthereumEvent)
		assert.True(t, ok)
	})

	t.Run("FutureFromProto", func(t *testing.T) {
		pf := &proto.Future{
			Oracle: &proto.Future_EthereumEvent{
				EthereumEvent: &proto.EthereumEvent{},
			},
		}
		f, err := FutureFromProto(pf)
		assert.NotNil(t, f)
		assert.Nil(t, err)
	})

	t.Run("FutureFromProto nil", func(t *testing.T) {
		f, err := FutureFromProto(nil)
		assert.Nil(t, f)
		assert.NotNil(t, err)
		assert.Equal(t, err, ErrNilFuture)
	})

	t.Run("ProductFromProto nil", func(t *testing.T) {
		p, err := ProductFromProto(nil)
		assert.Nil(t, p)
		assert.NotNil(t, err)
		assert.Equal(t, err, ErrNilProduct)
	})

	t.Run("ProductFromProto unimplemented", func(t *testing.T) {
		p, err := ProductFromProto(struct{}{})
		assert.Nil(t, p)
		assert.NotNil(t, err)
		assert.Equal(t, err, ErrUnimplementedProduct)
	})

	t.Run("ProductFromProto", func(t *testing.T) {
		pp := &proto.Instrument_Future{
			Future: &proto.Future{
				Oracle: &proto.Future_EthereumEvent{
					EthereumEvent: &proto.EthereumEvent{},
				},
			},
		}
		p, err := ProductFromProto(pp)
		assert.NotNil(t, p)
		assert.Nil(t, err)
		_, ok := p.(*Future)
		assert.True(t, ok)
	})

	t.Run("InstrumentFromProto nil", func(t *testing.T) {
		i, err := InstrumentFromProto(nil)
		assert.Nil(t, i)
		assert.NotNil(t, err)
		assert.Equal(t, err, ErrNilInstrument)
	})

	t.Run("InstrumentFromProto", func(t *testing.T) {
		pi := &proto.Instrument{
			Metadata: &proto.InstrumentMetadata{},
			Product: &proto.Instrument_Future{
				Future: &proto.Future{
					Oracle: &proto.Future_EthereumEvent{
						EthereumEvent: &proto.EthereumEvent{},
					},
				},
			},
		}
		i, err := InstrumentFromProto(pi)
		assert.NotNil(t, i)
		assert.Nil(t, err)
	})

	t.Run("RiskModelFromProto nil", func(t *testing.T) {
		rm, err := RiskModelFromProto(nil)
		assert.Nil(t, rm)
		assert.NotNil(t, err)
		assert.Equal(t, err, ErrNilRiskModel)
	})

	t.Run("RiskModelFromProto unimplemented", func(t *testing.T) {
		rm, err := RiskModelFromProto(struct{}{})
		assert.Nil(t, rm)
		assert.NotNil(t, err)
		assert.Equal(t, err, ErrUnimplementedRiskModel)
	})

	t.Run("RiskModelFromProto", func(t *testing.T) {
		prm := &proto.TradableInstrument_BuiltinFutures{
			BuiltinFutures: &proto.BuiltinFutures{
				HistoricVolatility: 42.4,
			},
		}
		rm, err := RiskModelFromProto(prm)
		assert.NotNil(t, rm)
		assert.Nil(t, err)
		_, ok := rm.(*BuiltinFutures)
		assert.True(t, ok)
	})

	t.Run("TradableInstrumentFromProto nil", func(t *testing.T) {
		ti, err := TradableInstrumentFromProto(nil)
		assert.Nil(t, ti)
		assert.NotNil(t, err)
		assert.Equal(t, err, ErrNilTradableInstrument)
	})

	t.Run("TradableInstrumentFromProto nil", func(t *testing.T) {
		pti := &proto.TradableInstrument{
			Instrument: &proto.Instrument{
				Metadata: &proto.InstrumentMetadata{},
				Product: &proto.Instrument_Future{
					Future: &proto.Future{
						Oracle: &proto.Future_EthereumEvent{
							EthereumEvent: &proto.EthereumEvent{},
						},
					},
				},
			},
			RiskModel: &proto.TradableInstrument_BuiltinFutures{
				BuiltinFutures: &proto.BuiltinFutures{
					HistoricVolatility: 42.4,
				},
			},
		}

		ti, err := TradableInstrumentFromProto(pti)
		assert.NotNil(t, ti)
		assert.Nil(t, err)
	})

	t.Run("MarketFromProto nil", func(t *testing.T) {
		m, err := MarketFromProto(nil)
		assert.Nil(t, m)
		assert.NotNil(t, err)
		assert.Equal(t, err, ErrNilMarket)
	})

	t.Run("MarketFromProto", func(t *testing.T) {
		pm := &proto.Market{
			TradableInstrument: &proto.TradableInstrument{
				Instrument: &proto.Instrument{
					Metadata: &proto.InstrumentMetadata{},
					Product: &proto.Instrument_Future{
						Future: &proto.Future{
							Oracle: &proto.Future_EthereumEvent{
								EthereumEvent: &proto.EthereumEvent{},
							},
						},
					},
				},
				RiskModel: &proto.TradableInstrument_BuiltinFutures{
					BuiltinFutures: &proto.BuiltinFutures{
						HistoricVolatility: 42.4,
					},
				},
			},
			TradingMode: &proto.Market_Continuous{
				Continuous: &proto.ContinuousTrading{
					TickSize: 42,
				},
			},
		}

		m, err := MarketFromProto(pm)
		assert.NotNil(t, m)
		assert.Nil(t, err)
	})
}

func intptr(i int) *int {
	return &i
}

func stringptr(s string) *string {
	return &s
}
