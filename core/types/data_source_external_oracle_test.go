package types_test

import (
	"testing"

	"code.vegaprotocol.io/vega/core/types"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
	v1 "code.vegaprotocol.io/vega/protos/vega/data/v1"
	"github.com/stretchr/testify/assert"
)

func TestDataSourceSpecConfigurationIntoProto(t *testing.T) {
	t.Run("non-empty oracle with empty lists", func(t *testing.T) {
		ds := types.NewDataSourceDefinitionWith(types.DataSourceSpecConfiguration{})
		protoDs := ds.IntoProto()
		assert.IsType(t, &vegapb.DataSourceDefinition{}, protoDs)
		assert.NotNil(t, protoDs.SourceType)
		ext := protoDs.GetExternal()
		assert.NotNil(t, ext)
		o := ext.GetOracle()
		assert.Equal(t, 0, len(o.Signers))
		assert.Equal(t, 0, len(o.Filters))
	})

	t.Run("non-empty oracle with data", func(t *testing.T) {
		ds := types.NewDataSourceDefinitionWith(types.DataSourceSpecConfiguration{
			Signers: []*types.Signer{
				{},
			},
			Filters: []*types.DataSourceSpecFilter{
				{
					Key: &types.DataSourceSpecPropertyKey{
						Name: "test-name",
						Type: types.DataSourceSpecPropertyKeyType(0),
					},
				},
			},
		})

		protoDs := ds.IntoProto()
		assert.IsType(t, &vegapb.DataSourceDefinition{}, protoDs)
		assert.NotNil(t, protoDs.SourceType)
		ext := protoDs.GetExternal()
		assert.NotNil(t, ext)
		o := ext.GetOracle()
		assert.Equal(t, 1, len(o.Signers))
		assert.Nil(t, o.Signers[0].Signer)
		assert.Equal(t, 1, len(o.Filters))
		assert.NotNil(t, o.Filters[0].Conditions)
		assert.NotNil(t, o.Filters[0].Key)
	})
}

func TestDataSourceSpecConfigurationFromProto(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		s := types.DataSourceSpecConfigurationFromProto(nil)

		assert.NotNil(t, s)
		assert.IsType(t, types.DataSourceSpecConfiguration{}, s)

		assert.Nil(t, s.Signers)
		assert.Nil(t, s.Filters)
	})

	t.Run("non-empty with empty lists", func(t *testing.T) {
		protoSource := &vegapb.DataSourceSpecConfiguration{
			Signers: nil,
			Filters: nil,
		}
		s := types.DataSourceSpecConfigurationFromProto(protoSource)

		assert.NotNil(t, s)
		assert.NotNil(t, &types.DataSourceSpecConfiguration{}, s)

		assert.NotNil(t, s.Signers)
		assert.Equal(t, 0, len(s.Signers))
		assert.NotNil(t, s.Filters)
		assert.Equal(t, 0, len(s.Filters))
	})

	t.Run("non-empty", func(t *testing.T) {
		protoSource := &vegapb.DataSourceSpecConfiguration{
			Signers: []*v1.Signer{
				{
					Signer: &v1.Signer_EthAddress{
						EthAddress: &v1.ETHAddress{
							Address: "some-address",
						},
					},
				},
			},
			Filters: []*v1.Filter{
				{
					Key: &v1.PropertyKey{
						Name: "test-key",
						Type: v1.PropertyKey_Type(1),
					},
				},
			},
		}

		ds := types.DataSourceSpecConfigurationFromProto(protoSource)
		assert.NotNil(t, ds)
		assert.Equal(t, 1, len(ds.Signers))
		assert.IsType(t, &types.SignerETHAddress{}, ds.Signers[0].Signer)
		assert.Equal(t, "some-address", ds.Signers[0].GetSignerETHAddress().Address)
		assert.Equal(t, 1, len(ds.Filters))
		assert.Equal(t, "test-key", ds.Filters[0].Key.Name)
		assert.Equal(t, types.DataSourceSpecPropertyKeyType(1), ds.Filters[0].Key.Type)
	})
}

func TestDataSourceSpecConfigurationString(t *testing.T) {
	t.Run("non-empty oracle with empty lists", func(t *testing.T) {
		ds := types.NewDataSourceDefinitionWith(types.DataSourceSpecConfiguration{}).String()

		assert.Equal(t, "signers() filters()", ds)
	})

	t.Run("non-empty oracle with data", func(t *testing.T) {
		ds := types.NewDataSourceDefinitionWith(types.DataSourceSpecConfiguration{
			Signers: []*types.Signer{
				{},
			},
			Filters: []*types.DataSourceSpecFilter{
				{
					Key: &types.DataSourceSpecPropertyKey{
						Name: "test-name",
						Type: types.DataSourceSpecPropertyKeyType(0),
					},
					Conditions: []*types.DataSourceSpecCondition{
						{
							Operator: 8,
							Value:    "12",
						},
					},
				},
			},
		}).String()

		assert.Equal(t, "signers(nil) filters(key(name(test-name) type(TYPE_UNSPECIFIED) decimals()) conditions([value(12) operator(8)]))", ds)
	})
}

func TestToDataSourceDefinitionProto(t *testing.T) {
}
