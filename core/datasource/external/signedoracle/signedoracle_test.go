package signedoracle_test

import (
	"testing"

	"code.vegaprotocol.io/vega/core/datasource"
	"code.vegaprotocol.io/vega/core/datasource/common"
	"code.vegaprotocol.io/vega/core/datasource/external/signedoracle"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
	v1 "code.vegaprotocol.io/vega/protos/vega/data/v1"
	"github.com/stretchr/testify/assert"
)

func TestSpecConfigurationIntoProto(t *testing.T) {
	t.Run("non-empty oracle with empty lists", func(t *testing.T) {
		ds := datasource.NewDefinitionWith(signedoracle.SpecConfiguration{})
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
		ds := datasource.NewDefinitionWith(signedoracle.SpecConfiguration{
			Signers: []*common.Signer{
				{},
			},
			Filters: []*common.SpecFilter{
				{
					Key: &common.SpecPropertyKey{
						Name: "test-name",
						Type: common.SpecPropertyKeyType(0),
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

func TestSpecConfigurationFromProto(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		s := signedoracle.SpecConfigurationFromProto(nil)

		assert.NotNil(t, s)
		assert.IsType(t, signedoracle.SpecConfiguration{}, s)

		assert.Nil(t, s.Signers)
		assert.Nil(t, s.Filters)
	})

	t.Run("non-empty with empty lists", func(t *testing.T) {
		protoSource := &vegapb.DataSourceSpecConfiguration{
			Signers: nil,
			Filters: nil,
		}
		s := signedoracle.SpecConfigurationFromProto(protoSource)

		assert.NotNil(t, s)
		assert.NotNil(t, &signedoracle.SpecConfiguration{}, s)

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

		ds := signedoracle.SpecConfigurationFromProto(protoSource)
		assert.NotNil(t, ds)
		assert.Equal(t, 1, len(ds.Signers))
		assert.IsType(t, &common.SignerETHAddress{}, ds.Signers[0].Signer)
		assert.Equal(t, "some-address", ds.Signers[0].GetSignerETHAddress().Address)
		assert.Equal(t, 1, len(ds.Filters))
		assert.Equal(t, "test-key", ds.Filters[0].Key.Name)
		assert.Equal(t, common.SpecPropertyKeyType(1), ds.Filters[0].Key.Type)
	})
}

func TestSpecConfigurationString(t *testing.T) {
	t.Run("non-empty oracle with empty lists", func(t *testing.T) {
		ds := datasource.NewDefinitionWith(signedoracle.SpecConfiguration{}).String()

		assert.Equal(t, "signers() filters()", ds)
	})

	t.Run("non-empty oracle with data", func(t *testing.T) {
		ds := datasource.NewDefinitionWith(signedoracle.SpecConfiguration{
			Signers: []*common.Signer{
				{},
			},
			Filters: []*common.SpecFilter{
				{
					Key: &common.SpecPropertyKey{
						Name: "test-name",
						Type: common.SpecPropertyKeyType(0),
					},
					Conditions: []*common.SpecCondition{
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

func TestToDataSourceDefinitionProto(t *testing.T) {}

func TestSpecConfigurationGetTimeTriggers(t *testing.T) {
	ds := datasource.NewDefinitionWith(
		signedoracle.SpecConfiguration{
			Signers: []*common.Signer{
				{},
			},
			Filters: []*common.SpecFilter{
				{
					Key: &common.SpecPropertyKey{
						Name: "test-name",
						Type: common.SpecPropertyKeyType(0),
					},
					Conditions: []*common.SpecCondition{
						{
							Operator: 8,
							Value:    "12",
						},
					},
				},
			},
		})

	triggers := ds.GetTimeTriggers()
	assert.NotNil(t, triggers)
	assert.Equal(t, 1, len(triggers))
	assert.IsType(t, &common.InternalTimeTrigger{}, triggers[0])
	assert.Nil(t, triggers[0])
}
