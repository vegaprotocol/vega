package gql_test

import (
	"testing"

	gql "code.vegaprotocol.io/vega/gateway/graphql"
	"code.vegaprotocol.io/vega/proto"

	"github.com/stretchr/testify/assert"
)

func TestModelConverters(t *testing.T) {

	t.Run("TradingModeFromProto unimplemented", func(t *testing.T) {
		ptm := 0
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

	t.Run("NewMarketInput.IntoProto", func(t *testing.T) {

		mkt := gql.NewMarketInput{
			Instrument: &gql.InstrumentConfigurationInput{
				Name: "abcXyz",
				Code: "abccode",
				FutureProduct: &gql.FutureProductInput{
					Maturity:        "asdasdas",
					SettlementAsset: "Ethereum/Ether",
					QuoteName:       "Xyz",
					OracleSpec: &gql.OracleSpecConfigurationInput{
						PubKeys: []string{
							"0xDEADBEEF",
						},
						Filters: []*gql.FilterInput{
							{
								Key: &gql.PropertyKeyInput{
									Name: "prices.BTC.value",
									Type: gql.PropertyKeyTypeTypeInteger,
								},
								Conditions: []*gql.ConditionInput{
									{
										Operator: gql.ConditionOperatorOperatorEquals,
										Value:    "42",
									},
								},
							},
						},
					},
					OracleSpecBinding: &gql.OracleSpecToFutureBindingInput{
						SettlementPriceProperty: "prices.BTC.value",
					},
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
					OracleSpec: &gql.OracleSpecConfigurationInput{
						PubKeys: []string{
							"0xDEADBEEF",
						},
						Filters: []*gql.FilterInput{
							{
								Key: &gql.PropertyKeyInput{
									Name: "prices.BTC.value",
									Type: gql.PropertyKeyTypeTypeInteger,
								},
								Conditions: []*gql.ConditionInput{
									{
										Operator: gql.ConditionOperatorOperatorEquals,
										Value:    "42",
									},
								},
							},
						},
					},
					OracleSpecBinding: &gql.OracleSpecToFutureBindingInput{
						SettlementPriceProperty: "prices.BTC.value",
					},
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
