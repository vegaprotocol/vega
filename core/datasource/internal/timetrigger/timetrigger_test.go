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

//lint:file-ignore ST1003 Ignore underscores in names, this is straigh copied from the proto package to ease introducing the domain types

package timetrigger_test

import (
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/datasource"
	"code.vegaprotocol.io/vega/core/datasource/common"
	"code.vegaprotocol.io/vega/core/datasource/internal/timetrigger"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
	datapb "code.vegaprotocol.io/vega/protos/vega/data/v1"

	"github.com/stretchr/testify/assert"
)

func TestSpecConfigurationIntoProto(t *testing.T) {
	t.Run("non-empty time source with empty lists", func(t *testing.T) {
		ds := datasource.NewDefinitionWith(timetrigger.SpecConfiguration{})
		protoDs := ds.IntoProto()
		assert.IsType(t, &vegapb.DataSourceDefinition{}, protoDs)
		assert.NotNil(t, protoDs.SourceType)
		ext := protoDs.GetInternal()
		assert.NotNil(t, ext)
		o := ext.GetTimeTrigger()
		assert.Equal(t, 0, len(o.Conditions))
	})

	t.Run("non-empty time source with data", func(t *testing.T) {
		timeNow := time.Now()
		ds := datasource.NewDefinitionWith(timetrigger.SpecConfiguration{
			Triggers: common.InternalTimeTriggers{
				{
					Initial: &timeNow,
					Every:   int64(15),
				},
			},
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
		internal := protoDs.GetInternal()
		assert.NotNil(t, internal)
		o := internal.GetTimeTrigger()
		assert.Equal(t, 3, len(o.Conditions))
		assert.Equal(t, datapb.Condition_Operator(0), o.Conditions[0].Operator)
		assert.Equal(t, "", o.Conditions[0].Value)
		assert.Equal(t, datapb.Condition_OPERATOR_EQUALS, o.Conditions[1].Operator)
		assert.Equal(t, "14", o.Conditions[1].Value)
		assert.Equal(t, datapb.Condition_OPERATOR_GREATER_THAN, o.Conditions[2].Operator)
		assert.Equal(t, "9", o.Conditions[2].Value)
		assert.IsType(t, &datapb.InternalTimeTrigger{}, o.Triggers[0])
		assert.Equal(t, timeNow.Unix(), *o.Triggers[0].Initial)
		assert.Equal(t, int64(15), o.Triggers[0].Every)
	})
}

func TestSpecConfigurationGetFilters(t *testing.T) {
	timeNow := time.Now()
	ds := datasource.NewDefinitionWith(timetrigger.SpecConfiguration{
		Triggers: common.InternalTimeTriggers{
			{
				Initial: &timeNow,
				Every:   int64(15),
			},
		},
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

	filters := ds.GetFilters()
	assert.NotNil(t, filters)
	assert.Equal(t, 1, len(filters))
	assert.IsType(t, &common.SpecFilter{}, filters[0])
	assert.Equal(t, timetrigger.InternalTimeTriggerKey, filters[0].Key.Name)
	assert.Equal(t, datapb.PropertyKey_TYPE_TIMESTAMP, filters[0].Key.Type)
	assert.Equal(t, 3, len(filters[0].Conditions))
	assert.Equal(t, datapb.Condition_OPERATOR_EQUALS, filters[0].Conditions[1].Operator)
	assert.Equal(t, "14", filters[0].Conditions[1].Value)
	assert.Equal(t, datapb.Condition_OPERATOR_GREATER_THAN, filters[0].Conditions[2].Operator)
	assert.Equal(t, "14", filters[0].Conditions[1].Value)
}

func TestSpecConfigurationGetTimeTriggers(t *testing.T) {
	timeNow := time.Now()
	ds := datasource.NewDefinitionWith(timetrigger.SpecConfiguration{
		Triggers: common.InternalTimeTriggers{
			{
				Initial: &timeNow,
				Every:   int64(15),
			},
		},
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

	triggers := ds.GetTimeTriggers()
	assert.NotNil(t, triggers)
	assert.Equal(t, 1, len(triggers))
	assert.IsType(t, &common.InternalTimeTrigger{}, triggers[0])
	assert.Equal(t, timeNow, *triggers[0].Initial)
	assert.Equal(t, int64(15), triggers[0].Every)
}
