package gql_test

import (
	"testing"

	gql "code.vegaprotocol.io/vega/gateway/graphql"
	"code.vegaprotocol.io/vega/proto"
	"github.com/stretchr/testify/assert"
)

func TestModelConverters(t *testing.T) {

	t.Run("TradingModeFromProto unimplemented", func(t *testing.T) {
		ptm := int(0)
		tm, err := gql.TradingModeConfigFromProto(ptm)
		assert.Nil(t, tm)
		assert.NotNil(t, err)
		assert.Equal(t, gql.ErrUnimplementedTradingMode, err)
	})

	t.Run("TradingModeFromProto nil", func(t *testing.T) {
		tm, err := gql.TradingModeConfigFromProto(nil)
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
		tm, err := gql.TradingModeConfigFromProto(ptm)
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
		tm, err := gql.TradingModeConfigFromProto(ptm)
		assert.NotNil(t, tm)
		assert.Nil(t, err)
		_, ok := tm.(*gql.DiscreteTrading)
		assert.True(t, ok)
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

	t.Run("NewMarketInput.IntoProto", func(t *testing.T) {

		mkt := gql.NewMarketInput{
			Instrument: &gql.InstrumentConfigurationInput{
				Name: "abcXyz",
				Code: "abccode",
				FutureProduct: &gql.FutureProductInput{
					Maturity:        "asdasdas",
					SettlementAsset: "Ethereum/Ether",
					QuoteName:       "Xyz",
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
				Code: "abccode",
				Name: "abcXyz",
				FutureProduct: &gql.FutureProductInput{
					Maturity:        "asdasdas",
					SettlementAsset: "Ethereum/Ether",
					QuoteName:       "Xyz",
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

func stringptr(s string) *string {
	return &s
}
