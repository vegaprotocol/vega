package types

import (
	"errors"
	"fmt"

	vegapb "code.vegaprotocol.io/vega/protos/vega"
)

var ErrInternalTimeDataSourceMissingConditions = errors.New("internal time based data source must have at least one condition")

// DataSourceSpecConfigurationTime is used internally.
type DataSourceSpecConfigurationTime struct {
	Conditions []*DataSourceSpecCondition
}

// String returns the content of DataSourceSpecConfigurationTime as a string.
func (s DataSourceSpecConfigurationTime) String() string {
	return fmt.Sprintf(
		"conditions(%s)", DataSourceSpecConditions(s.Conditions).String(),
	)
}

func (s DataSourceSpecConfigurationTime) IntoProto() *vegapb.DataSourceSpecConfigurationTime {
	return &vegapb.DataSourceSpecConfigurationTime{
		Conditions: DataSourceSpecConditions(s.Conditions).IntoProto(),
	}
}

func (s DataSourceSpecConfigurationTime) DeepClone() dataSourceType {
	conditions := []*DataSourceSpecCondition{}
	conditions = append(conditions, s.Conditions...)

	return DataSourceSpecConfigurationTime{
		Conditions: conditions,
	}
}

func DataSourceSpecConfigurationTimeFromProto(protoConfig *vegapb.DataSourceSpecConfigurationTime) DataSourceSpecConfigurationTime {
	dst := DataSourceSpecConfigurationTime{
		Conditions: []*DataSourceSpecCondition{},
	}
	if protoConfig != nil {
		dst.Conditions = DataSourceSpecConditionsFromProto(protoConfig.Conditions)
	}

	return dst
}

func (s DataSourceSpecConfigurationTime) ToDataSourceDefinitionProto() (*vegapb.DataSourceDefinition, error) {
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
