package types

import (
	"fmt"
	"strings"

	datapb "code.vegaprotocol.io/vega/protos/vega/data/v1"
)

type DataSourceSpecConditions []*DataSourceSpecCondition

func (sc DataSourceSpecConditions) IntoProto() []*datapb.Condition {
	protoConditions := make([]*datapb.Condition, 0, len(sc))
	for _, condition := range sc {
		protoConditions = append(protoConditions, condition.IntoProto())
	}
	return protoConditions
}

func (sc DataSourceSpecConditions) String() string {
	if sc == nil {
		return "[]"
	}
	strs := []string{}
	for _, c := range sc {
		strs = append(strs, c.String())
	}
	return "[" + strings.Join(strs, ", ") + "]"
}

type DataSourceSpecConditionOperator = datapb.Condition_Operator

// DataSourceSpecCondition mirrors datapb.Condition type.
type DataSourceSpecCondition struct {
	Value    string
	Operator DataSourceSpecConditionOperator
}

func (c DataSourceSpecCondition) isDataSourceType() {}

func (c DataSourceSpecCondition) oneOfProto() interface{} {
	return c
}

func (c DataSourceSpecCondition) String() string {
	return fmt.Sprintf(
		"value(%s) operator(%s)",
		c.Value,
		c.Operator.String(),
	)
}

func (c DataSourceSpecCondition) IntoProto() *datapb.Condition {
	return &datapb.Condition{
		Operator: c.Operator,
		Value:    c.Value,
	}
}

func (c *DataSourceSpecCondition) DeepClone() dataSourceType {
	return &DataSourceSpecCondition{
		Operator: c.Operator,
		Value:    c.Value,
	}
}

func DataSourceSpecConditionFromProto(protoCondition *datapb.Condition) *DataSourceSpecCondition {
	return &DataSourceSpecCondition{
		Operator: protoCondition.Operator,
		Value:    protoCondition.Value,
	}
}

func DataSourceSpecConditionsFromProto(protoConditions []*datapb.Condition) []*DataSourceSpecCondition {
	conditions := make([]*DataSourceSpecCondition, 0, len(protoConditions))
	for _, protoCondition := range protoConditions {
		conditions = append(conditions, DataSourceSpecConditionFromProto(protoCondition))
	}
	return conditions
}

func DeepCloneDataSourceSpecConditions(conditions []*DataSourceSpecCondition) []*DataSourceSpecCondition {
	othConditions := make([]*DataSourceSpecCondition, 0, len(conditions))
	for _, condition := range conditions {
		othConditions = append(othConditions, condition.DeepClone().(*DataSourceSpecCondition))
	}
	return othConditions
}
