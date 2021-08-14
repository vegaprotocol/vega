package gql_test

import (
	"testing"

	gql "code.vegaprotocol.io/data-node/gateway/graphql"
	proto "code.vegaprotocol.io/protos/vega"

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

}

func stringptr(s string) *string {
	return &s
}
