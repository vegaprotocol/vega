package types

import (
	"errors"
	"fmt"

	vegapb "code.vegaprotocol.io/vega/protos/vega"
	datapb "code.vegaprotocol.io/vega/protos/vega/data/v1"
)

var ErrInternalTimeDataSourceMissingConditions = errors.New("internal time based data source must have at least one condition")

// DataSourceSpecConfigurationTime is used internally.
type DataSourceSpecConfigurationTime struct {
	Conditions []*DataSourceSpecCondition
}

func (s *DataSourceSpecConfigurationTime) isDataSourceType() {}

func (s *DataSourceSpecConfigurationTime) oneOfProto() interface{} {
	return s
}

// /
// String returns the content of DataSourceSpecConfigurationTime as a string.
func (s *DataSourceSpecConfigurationTime) String() string {
	return fmt.Sprintf(
		"conditions(%s)", DataSourceSpecConditions(s.Conditions).String(),
	)
}

func (s *DataSourceSpecConfigurationTime) IntoProto() *vegapb.DataSourceSpecConfigurationTime {
	return &vegapb.DataSourceSpecConfigurationTime{
		Conditions: DataSourceSpecConditions(s.Conditions).IntoProto(),
	}
}

func (s *DataSourceSpecConfigurationTime) DeepClone() dataSourceType {
	conditions := []*DataSourceSpecCondition{}
	conditions = append(conditions, s.Conditions...)

	return &DataSourceSpecConfigurationTime{
		Conditions: conditions,
	}
}

func DataSourceSpecConfigurationTimeFromProto(protoConfig *vegapb.DataSourceSpecConfigurationTime) *DataSourceSpecConfigurationTime {
	dst := &DataSourceSpecConfigurationTime{
		Conditions: []*DataSourceSpecCondition{},
	}
	if protoConfig != nil {
		dst.Conditions = DataSourceSpecConditionsFromProto(protoConfig.Conditions)
	}

	return dst
}

type DataSourceDefinitionInternalTime struct {
	Time *DataSourceSpecConfigurationTime
}

func (i *DataSourceDefinitionInternalTime) isDataSourceType() {}

func (i *DataSourceDefinitionInternalTime) oneOfProto() interface{} {
	return i.IntoProto()
}

func (i *DataSourceDefinitionInternalTime) IntoProto() *vegapb.DataSourceDefinitionInternal_Time {
	ids := &vegapb.DataSourceSpecConfigurationTime{
		Conditions: []*datapb.Condition{},
	}

	if i.Time != nil {
		ids = i.Time.IntoProto()
	}

	return &vegapb.DataSourceDefinitionInternal_Time{
		Time: ids,
	}
}

func (i *DataSourceDefinitionInternalTime) DeepClone() dataSourceType {
	if i.Time == nil {
		return &DataSourceDefinitionInternalTime{
			Time: &DataSourceSpecConfigurationTime{},
		}
	}

	return nil
}

func (i *DataSourceDefinitionInternalTime) String() string {
	if i.Time == nil {
		return ""
	}
	return i.Time.String()
}

func DataSourceDefinitionInternalTimeFromProto(protoConfig *vegapb.DataSourceDefinitionInternal_Time) *DataSourceDefinitionInternalTime {
	ids := &DataSourceDefinitionInternalTime{
		Time: &DataSourceSpecConfigurationTime{},
	}

	if protoConfig != nil {
		if protoConfig.Time != nil {
			ids.Time = DataSourceSpecConfigurationTimeFromProto(protoConfig.Time)
		}
	}

	return ids
}
