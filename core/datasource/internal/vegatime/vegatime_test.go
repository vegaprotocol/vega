// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

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
		assert.Equal(t, "conditions([])", ds)
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

		assert.Equal(t, "conditions([value() operator(OPERATOR_UNSPECIFIED), value(14) operator(OPERATOR_EQUALS), value(9) operator(OPERATOR_GREATER_THAN)])", ds)
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
