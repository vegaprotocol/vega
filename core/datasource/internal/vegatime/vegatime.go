// Copyright (c) 2023 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

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
