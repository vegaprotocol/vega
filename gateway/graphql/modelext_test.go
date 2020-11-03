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
			TickSize: "0.1",
		}
		pdt, err := dt.IntoProto()
		assert.NotNil(t, pdt)
		assert.Nil(t, err)
		assert.Equal(t, int64(dt.Duration), pdt.Discrete.DurationNs)
	})

	t.Run("InstrumentMetadata.IntoProto", func(t *testing.T) {
		im := gql.InstrumentMetadata{Tags: []string{"tag:1", "tag:2"}}
		pim, err := im.IntoProto()
		assert.Nil(t, err)
		assert.NotNil(t, pim)
		assert.NotNil(t, pim.Tags)
		assert.Len(t, pim.Tags, 2)
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
				TickSize: "0.1",
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
		assert.Equal(t, pim.Tags[0], (im.Tags[0]))
		assert.Equal(t, pim.Tags[1], (im.Tags[1]))
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

	t.Run("NewMarketInput.IntoProto", func(t *testing.T) {

		mkt := gql.NewMarketInput{
			Instrument: &gql.InstrumentConfigurationInput{
				Name:      "abcXyz",
				Code:      "abccode",
				QuoteName: "Xyz",
				FutureProduct: &gql.FutureProductInput{
					Maturity: "asdasdas",
					Asset:    "Ethereum/Ether",
				},
			},
			RiskParameters: &gql.RiskParametersInput{
				Simple: &gql.SimpleRiskModelParamsInput{
					FactorLong:  0.1,
					FactorShort: 0.2,
				},
			},
			Metadata: []string{"tag:1", "tag:2"},
			ContinuousTrading: &gql.ContinuousTradingInput{
				TickSize: stringptr("0.1"),
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
			Instrument: &gql.InstrumentConfigurationInput{
				Code:      "abccode",
				Name:      "abcXyz",
				QuoteName: "Xyz",
				FutureProduct: &gql.FutureProductInput{
					Maturity: "asdasdas",
					Asset:    "Ethereum/Ether",
				},
			},
			RiskParameters: &gql.RiskParametersInput{
				Simple: &gql.SimpleRiskModelParamsInput{
					FactorLong:  0.1,
					FactorShort: 0.2,
				},
			},
			Metadata: []string{"tag:1", "tag:2"},
			DiscreteTrading: &gql.DiscreteTradingInput{
				Duration: 100,
				TickSize: stringptr("0.1"),
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

}

func intptr(i int) *int {
	return &i
}

func stringptr(s string) *string {
	return &s
}
