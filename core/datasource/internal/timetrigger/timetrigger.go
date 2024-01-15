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

package timetrigger

import (
	"fmt"
	"time"

	"code.vegaprotocol.io/vega/core/datasource/common"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
	datapb "code.vegaprotocol.io/vega/protos/vega/data/v1"
)

const InternalTimeTriggerKey = "vegaprotocol.builtin.timetrigger"

type SpecConfiguration struct {
	Triggers   common.InternalTimeTriggers
	Conditions []*common.SpecCondition
}

func (s *SpecConfiguration) SetInitial(initial, timeNow time.Time) error {
	if err := s.Triggers.Empty(); err != nil {
		return err
	}

	s.Triggers[0].SetInitial(initial, timeNow)
	return nil
}

func (s *SpecConfiguration) SetNextTrigger(timeNow time.Time) error {
	if err := s.Triggers.Empty(); err != nil {
		return err
	}

	s.Triggers[0].SetNextTrigger(timeNow)
	return nil
}

func (s SpecConfiguration) String() string {
	return fmt.Sprintf(
		"trigger(%s), conditions(%s)",
		s.Triggers.String(),
		common.SpecConditions(s.Conditions).String(),
	)
}

func (s SpecConfiguration) IntoProto() *vegapb.DataSourceSpecConfigurationTimeTrigger {
	return &vegapb.DataSourceSpecConfigurationTimeTrigger{
		Triggers:   s.Triggers.IntoProto(),
		Conditions: common.SpecConditions(s.Conditions).IntoProto(),
	}
}

func (s SpecConfiguration) DeepClone() common.DataSourceType {
	condition := make([]*common.SpecCondition, 0, len(s.Conditions))
	for _, c := range s.Conditions {
		condition = append(condition, c.DeepClone())
	}
	trigs := s.Triggers.DeepClone()
	return &SpecConfiguration{
		Triggers:   *trigs,
		Conditions: condition,
	}
}

func (s SpecConfiguration) GetFilters() []*common.SpecFilter {
	filters := []*common.SpecFilter{}

	conditions := []*common.SpecCondition{}
	if s.Conditions != nil {
		conditions = s.Conditions
	}

	// For the case the internal data source is time based
	// (https://github.com/vegaprotocol/specs/blob/master/protocol/0048-DSRI-data_source_internal.md#12-time-triggered)
	// We add the filter key values manually to match a time based data source
	// if len(s.Conditions) > 0 {
	filters = append(
		filters,
		&common.SpecFilter{
			Key: &common.SpecPropertyKey{
				Name: InternalTimeTriggerKey,
				Type: datapb.PropertyKey_TYPE_TIMESTAMP,
			},
			Conditions: conditions,
		},
	)
	//}
	return filters
}

func (s SpecConfiguration) GetTimeTriggers() *common.InternalTimeTriggers {
	return s.Triggers.DeepClone()
}

func (s SpecConfiguration) IsTriggered(tm time.Time) bool {
	return s.Triggers.IsTriggered(tm)
}

func SpecConfigurationFromProto(protoConfig *vegapb.DataSourceSpecConfigurationTimeTrigger, tm *time.Time) (*SpecConfiguration, error) {
	if protoConfig == nil {
		return &SpecConfiguration{}, nil
	}

	return &SpecConfiguration{
		Triggers:   common.InternalTimeTriggersFromProto(protoConfig.Triggers),
		Conditions: common.SpecConditionsFromProto(protoConfig.Conditions),
	}, nil
}

func (s *SpecConfiguration) ToDefinitionProto(_ uint64) (*vegapb.DataSourceDefinition, error) {
	return &vegapb.DataSourceDefinition{
		SourceType: &vegapb.DataSourceDefinition_Internal{
			Internal: &vegapb.DataSourceDefinitionInternal{
				SourceType: &vegapb.DataSourceDefinitionInternal_TimeTrigger{
					TimeTrigger: s.IntoProto(),
				},
			},
		},
	}, nil
}
