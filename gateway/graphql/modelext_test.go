package gql_test

import (
	"testing"

	gql "code.vegaprotocol.io/vega/gateway/graphql"
	"code.vegaprotocol.io/vega/proto"
	"github.com/stretchr/testify/assert"
)

func TestModelConverters(t *testing.T) {

	t.Run("DiscreteTrading.IntoProto", func(t *testing.T) {

		dt := &gql.DiscreteTrading{
			Duration: 123,
			TickSize: 42,
		}
		pdt, err := dt.IntoProto()
		assert.NotNil(t, pdt)
		assert.Nil(t, err)
		assert.Equal(t, int64(dt.Duration), pdt.Discrete.DurationNs)
	})

	t.Run("Future.IntoProto nil oracle", func(t *testing.T) {
		f := &gql.Future{Maturity: "12/31/19", Asset: "Ethereum/Ether"}
		pf, err := f.IntoProto()
		assert.Nil(t, pf)
		assert.NotNil(t, err)
		assert.Equal(t, gql.ErrNilOracle, err)
	})

	t.Run("Future.IntoProto", func(t *testing.T) {
		f := &gql.Future{
			Maturity: "12/31/19",
			Asset:    "Ethereum/Ether",
			Oracle: &gql.EthereumEvent{
				ContractID: "asdas",
				Event:      "aerasd",
			},
		}
		pf, err := f.IntoProto()
		assert.NotNil(t, pf)
		assert.Nil(t, err)
	})

	t.Run("InstrumentMetadata.IntoProto", func(t *testing.T) {
		im := gql.InstrumentMetadata{Tags: []*string{stringptr("tag:1"), stringptr("tag:2")}}
		pim, err := im.IntoProto()
		assert.Nil(t, err)
		assert.NotNil(t, pim)
		assert.NotNil(t, pim.Tags)
		assert.Len(t, pim.Tags, 2)
	})

	t.Run("Instrument.IntoProto nil product", func(t *testing.T) {
		i := gql.Instrument{}
		pi, err := i.IntoProto()
		assert.Nil(t, pi)
		assert.NotNil(t, err)
		assert.Equal(t, gql.ErrNilProduct, err)
	})

	t.Run("Instrument.IntoProto ", func(t *testing.T) {
		i := gql.Instrument{Product: &gql.Future{
			Maturity: "asdasdas",
			Asset:    "Ethereum/Ether",
			Oracle: &gql.EthereumEvent{
				ContractID: "asdas",
				Event:      "aerasd",
			},
		}}
		pi, err := i.IntoProto()
		assert.NotNil(t, pi)
		assert.Nil(t, err)
	})

	t.Run("TradableInstrument.IntoProto nil inners types", func(t *testing.T) {
		ti := gql.TradableInstrument{
			Instrument: &gql.Instrument{},
		}
		pti, err := ti.IntoProto()
		assert.Nil(t, pti)
		assert.NotNil(t, err)
		assert.Equal(t, gql.ErrNilProduct, err)

		ti.Instrument.Product = &gql.Future{
			Maturity: "asdasdas",
			Asset:    "Ethereum/Ether",
			Oracle: &gql.EthereumEvent{
				ContractID: "asdas",
				Event:      "aerasd",
			}}
		pti, err = ti.IntoProto()
		assert.Nil(t, pti)
		assert.NotNil(t, err)
		assert.Equal(t, gql.ErrNilRiskModel, err)
	})

	t.Run("TradableIstrument.IntoProto", func(t *testing.T) {
		ti := gql.TradableInstrument{
			Instrument: &gql.Instrument{
				Product: &gql.Future{
					Maturity: "asdasdas",
					Asset:    "Ethereum/Ether",
					Oracle: &gql.EthereumEvent{
						ContractID: "asdas",
						Event:      "aerasd",
					},
				},
			},
			RiskModel: &gql.LogNormalRiskModel{
				RiskAversionParameter: 0.01,
				Tau:                   1.0 / 365.25 / 24,
				Params: &gql.LogNormalModelParams{
					Mu:    0,
					R:     0.016,
					Sigma: 0.09,
				},
			},
		}
		pti, err := ti.IntoProto()
		assert.NotNil(t, pti)
		assert.Nil(t, err)
	})

	t.Run("Market.IntoProto", func(t *testing.T) {
		mkt := gql.Market{
			TradingMode: &gql.ContinuousTrading{TickSize: 123},
			TradableInstrument: &gql.TradableInstrument{
				Instrument: &gql.Instrument{
					Product: &gql.Future{
						Maturity: "asdasdas",
						Asset:    "Ethereum/Ether",
						Oracle: &gql.EthereumEvent{
							ContractID: "asdas",
							Event:      "aerasd",
						},
					},
				},
				RiskModel: &gql.LogNormalRiskModel{
					RiskAversionParameter: 0.01,
					Tau:                   1.0 / 365.25 / 24,
					Params: &gql.LogNormalModelParams{
						Mu:    0,
						R:     0.016,
						Sigma: 0.09,
					},
				},
			},
		}
		pmkt, err := mkt.IntoProto()
		assert.NoError(t, err)
		assert.NotNil(t, pmkt)
		assert.Nil(t, err)
	})

	t.Run("TradingModeFromProto unimplemented", func(t *testing.T) {
		ptm := int(0)
		tm, err := gql.TradingModeFromProto(ptm)
		assert.Nil(t, tm)
		assert.NotNil(t, err)
		assert.Equal(t, gql.ErrUnimplementedTradingMode, err)
	})

	t.Run("TradingModeFromProto nil", func(t *testing.T) {
		tm, err := gql.TradingModeFromProto(nil)
		assert.Nil(t, tm)
		assert.NotNil(t, err)
		assert.Equal(t, gql.ErrNilTradingMode, err)
	})

	t.Run("TradingModeFromProto Continuous", func(t *testing.T) {
		ptm := &proto.Market_Continuous{
			Continuous: &proto.ContinuousTrading{
				TickSize: 42,
			},
		}
		tm, err := gql.TradingModeFromProto(ptm)
		assert.NotNil(t, tm)
		assert.Nil(t, err)
		_, ok := tm.(*gql.ContinuousTrading)
		assert.True(t, ok)
	})

	t.Run("TradingModeFromProto Discrete", func(t *testing.T) {
		ptm := &proto.Market_Discrete{
			Discrete: &proto.DiscreteTrading{
				DurationNs: 42,
			},
		}
		tm, err := gql.TradingModeFromProto(ptm)
		assert.NotNil(t, tm)
		assert.Nil(t, err)
		_, ok := tm.(*gql.DiscreteTrading)
		assert.True(t, ok)
	})

	t.Run("InstrumentMetadataFromProto nil", func(t *testing.T) {
		im, err := gql.InstrumentMetadataFromProto(nil)
		assert.Nil(t, im)
		assert.NotNil(t, err)
		assert.Equal(t, gql.ErrNilInstrumentMetadata, err)
	})

	t.Run("InstrumentMetadataFromProto", func(t *testing.T) {
		pim := &proto.InstrumentMetadata{
			Tags: []string{"tag:1", "tag:2"},
		}
		im, err := gql.InstrumentMetadataFromProto(pim)
		assert.NotNil(t, im)
		assert.Nil(t, err)
		assert.Len(t, im.Tags, 2)
		assert.Equal(t, pim.Tags[0], *(im.Tags[0]))
		assert.Equal(t, pim.Tags[1], *(im.Tags[1]))
	})

	t.Run("EthereumEventFromproto nil", func(t *testing.T) {
		ee, err := gql.EthereumEventFromProto(nil)
		assert.Nil(t, ee)
		assert.NotNil(t, err)
		assert.Equal(t, err, gql.ErrNilEthereumEvent)
	})

	t.Run("EthereumEventFromproto", func(t *testing.T) {
		pee := &proto.EthereumEvent{}
		ee, err := gql.EthereumEventFromProto(pee)
		assert.NotNil(t, ee)
		assert.Nil(t, err)
	})

	t.Run("OracleFromProto nil", func(t *testing.T) {
		o, err := gql.OracleFromProto(nil)
		assert.Nil(t, o)
		assert.NotNil(t, err)
		assert.Equal(t, err, gql.ErrNilOracle)
	})

	t.Run("OracleFromProto unimplemented", func(t *testing.T) {
		o, err := gql.OracleFromProto(struct{}{})
		assert.Nil(t, o)
		assert.NotNil(t, err)
		assert.Equal(t, err, gql.ErrUnimplementedOracle)
	})

	t.Run("OracleFromProto EthereumEvent", func(t *testing.T) {
		po := &proto.Future_EthereumEvent{
			EthereumEvent: &proto.EthereumEvent{},
		}
		o, err := gql.OracleFromProto(po)
		assert.NotNil(t, o)
		assert.Nil(t, err)
		_, ok := o.(*gql.EthereumEvent)
		assert.True(t, ok)
	})

	t.Run("FutureFromProto", func(t *testing.T) {
		pf := &proto.Future{
			Oracle: &proto.Future_EthereumEvent{
				EthereumEvent: &proto.EthereumEvent{},
			},
		}
		f, err := gql.FutureFromProto(pf)
		assert.NotNil(t, f)
		assert.Nil(t, err)
	})

	t.Run("FutureFromProto nil", func(t *testing.T) {
		f, err := gql.FutureFromProto(nil)
		assert.Nil(t, f)
		assert.NotNil(t, err)
		assert.Equal(t, err, gql.ErrNilFuture)
	})

	t.Run("ProductFromProto nil", func(t *testing.T) {
		p, err := gql.ProductFromProto(nil)
		assert.Nil(t, p)
		assert.NotNil(t, err)
		assert.Equal(t, err, gql.ErrNilProduct)
	})

	t.Run("ProductFromProto unimplemented", func(t *testing.T) {
		p, err := gql.ProductFromProto(struct{}{})
		assert.Nil(t, p)
		assert.NotNil(t, err)
		assert.Equal(t, err, gql.ErrUnimplementedProduct)
	})

	t.Run("ProductFromProto", func(t *testing.T) {
		pp := &proto.Instrument_Future{
			Future: &proto.Future{
				Oracle: &proto.Future_EthereumEvent{
					EthereumEvent: &proto.EthereumEvent{},
				},
			},
		}
		p, err := gql.ProductFromProto(pp)
		assert.NotNil(t, p)
		assert.Nil(t, err)
		_, ok := p.(*gql.Future)
		assert.True(t, ok)
	})

	t.Run("InstrumentFromProto nil", func(t *testing.T) {
		i, err := gql.InstrumentFromProto(nil)
		assert.Nil(t, i)
		assert.NotNil(t, err)
		assert.Equal(t, err, gql.ErrNilInstrument)
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
		i, err := gql.InstrumentFromProto(pi)
		assert.NotNil(t, i)
		assert.Nil(t, err)
	})

	t.Run("RiskModelFromProto nil", func(t *testing.T) {
		rm, err := gql.RiskModelFromProto(nil)
		assert.Nil(t, rm)
		assert.NotNil(t, err)
		assert.Equal(t, err, gql.ErrNilRiskModel)
	})

	t.Run("RiskModelFromProto unimplemented", func(t *testing.T) {
		rm, err := gql.RiskModelFromProto(struct{}{})
		assert.Nil(t, rm)
		assert.NotNil(t, err)
		assert.Equal(t, err, gql.ErrUnimplementedRiskModel)
	})

	t.Run("RiskModelFromProto", func(t *testing.T) {
		prm := &proto.TradableInstrument_LogNormalRiskModel{
			LogNormalRiskModel: &proto.LogNormalRiskModel{
				RiskAversionParameter: 0.01,
				Tau:                   1.0 / 365.25 / 24,
				Params: &proto.LogNormalModelParams{
					Mu:    0,
					R:     0.016,
					Sigma: 0.09,
				},
			},
		}
		rm, err := gql.RiskModelFromProto(prm)
		assert.NotNil(t, rm)
		assert.Nil(t, err)
		_, ok := rm.(*gql.LogNormalRiskModel)
		assert.True(t, ok)
	})

	t.Run("TradableInstrumentFromProto nil", func(t *testing.T) {
		ti, err := gql.TradableInstrumentFromProto(nil)
		assert.Nil(t, ti)
		assert.NotNil(t, err)
		assert.Equal(t, err, gql.ErrNilTradableInstrument)
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
			MarginCalculator: &proto.MarginCalculator{
				ScalingFactors: &proto.ScalingFactors{
					SearchLevel:       1.1,
					InitialMargin:     1.2,
					CollateralRelease: 1.4,
				},
			},
			RiskModel: &proto.TradableInstrument_LogNormalRiskModel{
				LogNormalRiskModel: &proto.LogNormalRiskModel{
					RiskAversionParameter: 0.01,
					Tau:                   1.0 / 365.25 / 24,
					Params: &proto.LogNormalModelParams{
						Mu:    0,
						R:     0.016,
						Sigma: 0.09,
					},
				},
			},
		}

		ti, err := gql.TradableInstrumentFromProto(pti)
		assert.NotNil(t, ti)
		assert.Nil(t, err)
	})

	t.Run("MarketFromProto nil", func(t *testing.T) {
		m, err := gql.MarketFromProto(nil)
		assert.Nil(t, m)
		assert.NotNil(t, err)
		assert.Equal(t, err, gql.ErrNilMarket)
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
				MarginCalculator: &proto.MarginCalculator{
					ScalingFactors: &proto.ScalingFactors{
						SearchLevel:       1.1,
						InitialMargin:     1.2,
						CollateralRelease: 1.4,
					},
				},
				RiskModel: &proto.TradableInstrument_LogNormalRiskModel{
					LogNormalRiskModel: &proto.LogNormalRiskModel{
						RiskAversionParameter: 0.01,
						Tau:                   1.0 / 365.25 / 24,
						Params: &proto.LogNormalModelParams{
							Mu:    0,
							R:     0.016,
							Sigma: 0.09,
						},
					},
				},
			},
			TradingMode: &proto.Market_Continuous{
				Continuous: &proto.ContinuousTrading{
					TickSize: 42,
				},
			},
		}

		m, err := gql.MarketFromProto(pm)
		assert.NotNil(t, m)
		assert.Nil(t, err)
	})

	t.Run("NewMarketInput.IntoProto", func(t *testing.T) {

		mkt := gql.NewMarketInput{
			Instrument: &gql.IntrumentConfigurationInput{
				Name:      "abcXyz",
				Code:      "abccode",
				BaseName:  "abc",
				QuoteName: "Xyz",
				FutureProduct: &gql.FutureProductInput{
					Maturity: "asdasdas",
					Asset:    "Ethereum/Ether",
				},
			},
			Risk: &gql.RiskInput{
				Model: gql.RiskModelTypeSimple,
				SimpleParameters: &gql.SimpleRiskModelParamsInput{
					FactorLong:  0.1,
					FactorShort: 0.2,
				},
			},
			Metadata: []*string{stringptr("tag:1"), stringptr("tag:2")},
			ContinuousTrading: &gql.ContinuousTradingInput{
				TickSize: 10,
			},
			DecimalPlaces: 5,
		}
		pmkt, err := mkt.IntoProto()
		assert.NotNil(t, pmkt)
		assert.NoError(t, err)
	})

	t.Run("ProposalTermsInput.IntoProto nil change", func(t *testing.T) {

		proposal := &gql.ProposalTermsInput{
			ClosingDatetime:   "2020-09-30T07:28:06+00:00",
			EnactmentDatetime: "2020-10-30T07:28:06+00:00",
		}
		proposalProto, err := proposal.IntoProto()
		assert.Nil(t, proposalProto)
		assert.Error(t, err)
		assert.EqualError(t, gql.ErrInvalidChange, err.Error())
	})

	t.Run("ProposalTermsInput.IntoProto", func(t *testing.T) {

		mkt := gql.NewMarketInput{
			Instrument: &gql.IntrumentConfigurationInput{
				Code:      "abccode",
				Name:      "abcXyz",
				BaseName:  "abc",
				QuoteName: "Xyz",
				FutureProduct: &gql.FutureProductInput{
					Maturity: "asdasdas",
					Asset:    "Ethereum/Ether",
				},
			},
			Risk: &gql.RiskInput{
				Model: gql.RiskModelTypeSimple,
				SimpleParameters: &gql.SimpleRiskModelParamsInput{
					FactorLong:  0.1,
					FactorShort: 0.2,
				},
			},
			Metadata: []*string{stringptr("tag:1"), stringptr("tag:2")},
			DiscreteTrading: &gql.DiscreteTradingInput{
				Duration: 100,
				TickSize: 10,
			},
			DecimalPlaces: 5,
		}
		pmkt, err := mkt.IntoProto()
		assert.NotNil(t, pmkt)
		assert.Nil(t, err)

		proposal := &gql.ProposalTermsInput{
			ClosingDatetime:   "2020-09-30T07:28:06+00:00",
			EnactmentDatetime: "2020-10-30T07:28:06+00:00",
			NewMarket:         &mkt,
		}
		proposalProto, err := proposal.IntoProto()
		assert.NotNil(t, proposalProto)
		assert.NoError(t, err)
	})

	t.Run("ProposalTermsFromProto valid data", func(t *testing.T) {
		pm := &proto.NewMarketConfiguration{
			Instrument: &proto.IntrumentConfiguration{
				Name:      "December 2020 ETH vs VUSD future",
				Code:      "CRYPTO:ETHVUSD/DEC20",
				BaseName:  "ETH",
				QuoteName: "VUSD",
				Product: &proto.IntrumentConfiguration_Future{
					Future: &proto.FutureProduct{
						Asset:    "VUSD",
						Maturity: "2020-12-31T23:59:59Z",
					},
				},
			},
			Risk: &proto.RiskConfiguration{
				Model: proto.RiskConfiguration_MODEL_LOG_NORMAL,
				Parameters: &proto.RiskConfiguration_LogNormal{
					LogNormal: &proto.LogNormalRiskModel{
						RiskAversionParameter: 0.01,
						Tau:                   1.0 / 365.25 / 24,
						Params: &proto.LogNormalModelParams{
							Mu:    0,
							R:     0.016,
							Sigma: 0.09,
						},
					},
				},
			},
			Metadata: []string{
				"asset_class:fx/crypto",
				"product:futures",
			},
			DecimalPlaces: 5,
			TradingMode: &proto.NewMarketConfiguration_Continuous{
				Continuous: &proto.ContinuousTrading{
					TickSize: 42,
				},
			},
		}

		protoTerms, err := gql.ProposalTermsFromProto(&proto.ProposalTerms{
			ClosingTimestamp:   1234567,
			EnactmentTimestamp: 1234568,
			Change: &proto.ProposalTerms_NewMarket{
				NewMarket: &proto.NewMarket{
					Changes: pm,
				},
			},
		})
		assert.NoError(t, err)
		assert.NotNil(t, protoTerms)
	})
}

func intptr(i int) *int {
	return &i
}

func stringptr(s string) *string {
	return &s
}
