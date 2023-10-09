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

package vegatime

import (
	"fmt"

	"code.vegaprotocol.io/vega/core/datasource/common"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
	datapb "code.vegaprotocol.io/vega/protos/vega/data/v1"
)

const VegaTimeKey = "vegaprotocol.builtin.timestamp"

// SpecConfiguration is used internally.
type SpecConfiguration struct {
	Conditions []*common.SpecCondition
}

// String returns the content of DataSourceSpecConfigurationTime as a string.
func (s SpecConfiguration) String() string {
	return fmt.Sprintf(
		"conditions(%s)", common.SpecConditions(s.Conditions).String(),
	)
}

func (s SpecConfiguration) IntoProto() *vegapb.DataSourceSpecConfigurationTime {
	return &vegapb.DataSourceSpecConfigurationTime{
		Conditions: common.SpecConditions(s.Conditions).IntoProto(),
	}
}

func (s SpecConfiguration) DeepClone() common.DataSourceType {
	conditions := []*common.SpecCondition{}
	conditions = append(conditions, s.Conditions...)

	return SpecConfiguration{
		Conditions: conditions,
	}
}

func (s SpecConfiguration) GetFilters() []*common.SpecFilter {
	filters := []*common.SpecFilter{}
	// For the case the internal data source is time based
	// (https://github.com/vegaprotocol/specs/blob/master/protocol/0048-DSRI-data_source_internal.md#13-vega-time-changed)
	// We add the filter key values manually to match a time based data source
	// Ensure only a single filter has been created, that holds the first condition
	if len(s.Conditions) > 0 {
		filters = append(
			filters,
			&common.SpecFilter{
				Key: &common.SpecPropertyKey{
					Name: VegaTimeKey,
					Type: datapb.PropertyKey_TYPE_TIMESTAMP,
				},
				Conditions: []*common.SpecCondition{
					s.Conditions[0],
				},
			},
		)
	}
	return filters
}

func SpecConfigurationFromProto(protoConfig *vegapb.DataSourceSpecConfigurationTime) SpecConfiguration {
	if protoConfig == nil {
		return SpecConfiguration{}
	}

	return SpecConfiguration{
		Conditions: common.SpecConditionsFromProto(protoConfig.Conditions),
	}
}

func (s SpecConfiguration) ToDefinitionProto() (*vegapb.DataSourceDefinition, error) {
	return &vegapb.DataSourceDefinition{
		SourceType: &vegapb.DataSourceDefinition_Internal{
			Internal: &vegapb.DataSourceDefinitionInternal{
				SourceType: &vegapb.DataSourceDefinitionInternal_Time{
					Time: s.IntoProto(),
				},
			},
		},
	}, nil
}
