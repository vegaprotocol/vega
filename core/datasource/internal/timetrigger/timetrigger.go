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

package timetrigger

import (
	"fmt"
	"time"

	"code.vegaprotocol.io/vega/core/datasource/common"
	"code.vegaprotocol.io/vega/core/datasource/errors"
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
	return SpecConfiguration{
		Triggers:   s.Triggers.DeepClone(),
		Conditions: s.Conditions,
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

func (s SpecConfiguration) GetTimeTriggers() common.InternalTimeTriggers {
	return s.Triggers
}

func (s SpecConfiguration) IsTriggered(tm time.Time) bool {
	return s.Triggers.IsTriggered(tm)
}

func SpecConfigurationFromProto(protoConfig *vegapb.DataSourceSpecConfigurationTimeTrigger, tm *time.Time) (SpecConfiguration, error) {
	if tm == nil {
		return SpecConfiguration{}, errors.ErrMissingTimeForSettingTriggerRepetition
	}
	if protoConfig == nil {
		return SpecConfiguration{}, nil
	}

	return SpecConfiguration{
		Triggers:   common.InternalTimeTriggersFromProto(protoConfig.Triggers, *tm),
		Conditions: common.SpecConditionsFromProto(protoConfig.Conditions),
	}, nil
}

func (s SpecConfiguration) ToDefinitionProto() (*vegapb.DataSourceDefinition, error) {
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
