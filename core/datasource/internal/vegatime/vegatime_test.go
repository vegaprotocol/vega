package vegatime_test

import (
	"testing"

	"code.vegaprotocol.io/vega/core/datasource"
	"code.vegaprotocol.io/vega/core/datasource/common"
	"code.vegaprotocol.io/vega/core/datasource/internal/vegatime"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
	datapb "code.vegaprotocol.io/vega/protos/vega/data/v1"
	v1 "code.vegaprotocol.io/vega/protos/vega/data/v1"
	"github.com/stretchr/testify/assert"
)

func TestSpecConfigurationIntoProto(t *testing.T) {
	t.Run("non-empty time source with empty lists", func(t *testing.T) {
		ds := datasource.NewDefinitionWith(vegatime.SpecConfiguration{})
		protoDs := ds.IntoProto()
		assert.IsType(t, &vegapb.DataSourceDefinition{}, protoDs)
		assert.NotNil(t, protoDs.SourceType)
		ext := protoDs.GetInternal()
		assert.NotNil(t, ext)
		o := ext.GetTime()
		assert.Equal(t, 0, len(o.Conditions))
	})

	t.Run("non-empty time source with data", func(t *testing.T) {
		ds := datasource.NewDefinitionWith(vegatime.SpecConfiguration{
			Conditions: []*common.SpecCondition{
				{},
				{
					Operator: datapb.Condition_OPERATOR_EQUALS,
					Value:    "14",
				},
				{
					Operator: datapb.Condition_OPERATOR_GREATER_THAN,
					Value:    "9",
				},
			},
		})

		protoDs := ds.IntoProto()
		assert.IsType(t, &vegapb.DataSourceDefinition{}, protoDs)
		assert.NotNil(t, protoDs.SourceType)
		ext := protoDs.GetInternal()
		assert.NotNil(t, ext)
		o := ext.GetTime()
		assert.Equal(t, 3, len(o.Conditions))
		assert.Equal(t, datapb.Condition_Operator(0), o.Conditions[0].Operator)
		assert.Equal(t, "", o.Conditions[0].Value)
		assert.Equal(t, datapb.Condition_OPERATOR_EQUALS, o.Conditions[1].Operator)
		assert.Equal(t, "14", o.Conditions[1].Value)
		assert.Equal(t, datapb.Condition_OPERATOR_GREATER_THAN, o.Conditions[2].Operator)
		assert.Equal(t, "9", o.Conditions[2].Value)
	})
}

func TestSpecConfigurationString(t *testing.T) {
	t.Run("non-empty time source with empty lists", func(t *testing.T) {
		ds := datasource.NewDefinitionWith(vegatime.SpecConfiguration{}).String()
		assert.Equal(t, "conditions([]) timeTriggers([])", ds)
	})

	t.Run("non-empty time source with data", func(t *testing.T) {
		ds := datasource.NewDefinitionWith(vegatime.SpecConfiguration{
			Conditions: []*common.SpecCondition{
				{},
				{
					Operator: datapb.Condition_OPERATOR_EQUALS,
					Value:    "14",
				},
				{
					Operator: datapb.Condition_OPERATOR_GREATER_THAN,
					Value:    "9",
				},
			},
		}).String()

		assert.Equal(t, "conditions([value() operator(OPERATOR_UNSPECIFIED), value(14) operator(OPERATOR_EQUALS), value(9) operator(OPERATOR_GREATER_THAN)]) timeTriggers([])", ds)
	})
}

func TestSpecConfigurationFromProto(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		s := vegatime.SpecConfigurationFromProto(nil)

		assert.NotNil(t, s)
		assert.IsType(t, vegatime.SpecConfiguration{}, s)

		assert.Nil(t, s.Conditions)
	})

	t.Run("non-empty with empty lists", func(t *testing.T) {
		s := vegatime.SpecConfigurationFromProto(
			&vegapb.DataSourceSpecConfigurationTime{
				Conditions: nil,
			},
		)
		assert.NotNil(t, s)
		assert.IsType(t, vegatime.SpecConfiguration{}, s)
		assert.NotNil(t, s.Conditions)
		assert.Equal(t, 0, len(s.Conditions))
	})

	t.Run("non-empty with data", func(t *testing.T) {
		s := vegatime.SpecConfigurationFromProto(
			&vegapb.DataSourceSpecConfigurationTime{
				Conditions: []*v1.Condition{
					{},
					{
						Operator: datapb.Condition_OPERATOR_EQUALS,
						Value:    "14",
					},
					{
						Operator: datapb.Condition_OPERATOR_GREATER_THAN,
						Value:    "9",
					},
				},
			},
		)
		assert.NotNil(t, s)
		assert.Equal(t, 3, len(s.Conditions))
		assert.Equal(t, v1.Condition_Operator(1), s.Conditions[1].Operator)
		assert.Equal(t, "14", s.Conditions[1].Value)
		assert.Equal(t, v1.Condition_Operator(2), s.Conditions[2].Operator)
		assert.Equal(t, "9", s.Conditions[2].Value)
	})
}
