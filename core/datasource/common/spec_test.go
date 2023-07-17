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

package common_test

import (
	"testing"

	"code.vegaprotocol.io/vega/core/datasource/common"
	datapb "code.vegaprotocol.io/vega/protos/vega/data/v1"
	"github.com/stretchr/testify/assert"
)

func TestSpecConditionDeepClone(t *testing.T) {
	sc := common.SpecCondition{
		Operator: common.SpecConditionOperator(int32(5)),
		Value:    "12",
	}

	res := sc.DeepClone()
	assert.NotNil(t, res)
	assert.IsType(t, &common.SpecCondition{}, res)
	assert.Equal(t, common.SpecConditionOperator(int32(5)), res.Operator)
	assert.Equal(t, "12", res.Value)
}

func TestSpecConditionsString(t *testing.T) {
	sc := common.SpecConditions{
		{
			Operator: common.SpecConditionOperator(int32(5)),
			Value:    "12",
		},
		{
			Operator: common.SpecConditionOperator(int32(58)),
			Value:    "17",
		},
	}

	assert.Equal(t, "[value(12) operator(OPERATOR_LESS_THAN_OR_EQUAL), value(17) operator(58)]", sc.String())
}

func TestSpecConditionsIntoProto(t *testing.T) {
	sc := common.SpecConditions{
		{
			Operator: common.SpecConditionOperator(int32(5)),
			Value:    "12",
		},
		{
			Operator: common.SpecConditionOperator(int32(58)),
			Value:    "17",
		},
	}

	res := sc.IntoProto()
	assert.NotNil(t, res)
	assert.IsType(t, []*datapb.Condition{}, res)
	assert.Equal(t, 2, len(res))
	assert.Equal(t, datapb.Condition_Operator(5), res[0].Operator)
	assert.Equal(t, "12", res[0].Value)
	assert.Equal(t, datapb.Condition_Operator(58), res[1].Operator)
	assert.Equal(t, "17", res[1].Value)
}

func TestSpecConditionsFromProto(t *testing.T) {
	pc := []*datapb.Condition{
		{
			Operator: datapb.Condition_OPERATOR_GREATER_THAN,
			Value:    "test-value-0",
		},
		{
			Operator: datapb.Condition_OPERATOR_GREATER_THAN_OR_EQUAL,
			Value:    "test-value-1",
		},
	}

	res := common.SpecConditionsFromProto(pc)
	assert.NotNil(t, res)
	assert.IsType(t, []*common.SpecCondition{}, res)

	assert.Equal(t, 2, len(res))
	assert.Equal(t, datapb.Condition_OPERATOR_GREATER_THAN, res[0].Operator)
	assert.Equal(t, "test-value-0", res[0].Value)
	assert.Equal(t, datapb.Condition_OPERATOR_GREATER_THAN_OR_EQUAL, res[1].Operator)
	assert.Equal(t, "test-value-1", res[1].Value)
}

func TestSpecFiltersString(t *testing.T) {
	dp := uint64(5)
	sf := common.SpecFilters{
		{
			Key: &common.SpecPropertyKey{
				Name:                "test-name-0",
				Type:                common.SpecPropertyKeyType(4),
				NumberDecimalPlaces: &dp,
			},
		},
		{
			Key: &common.SpecPropertyKey{
				Name:                "test-name-1",
				Type:                common.SpecPropertyKeyType(9),
				NumberDecimalPlaces: &dp,
			},
		},
	}

	assert.Equal(t, "[key(name(test-name-0) type(TYPE_BOOLEAN) decimals(5)) conditions([]), key(name(test-name-1) type(9) decimals(5)) conditions([])]", sf.String())
}

func TestSpecFiltersIntoProto(t *testing.T) {
	dp := uint64(7)
	sf := common.SpecFilters{
		{
			Key: &common.SpecPropertyKey{
				Name:                "test-name-0",
				Type:                common.SpecPropertyKeyType(4),
				NumberDecimalPlaces: &dp,
			},
			Conditions: []*common.SpecCondition{
				{},
			},
		},
		{
			Key: &common.SpecPropertyKey{
				Name:                "test-name-1",
				Type:                common.SpecPropertyKeyType(9),
				NumberDecimalPlaces: &dp,
			},
			Conditions: []*common.SpecCondition{
				{
					Operator: common.SpecConditionOperator(5),
					Value:    "25",
				},
			},
		},
	}

	res := sf.IntoProto()
	assert.NotNil(t, res)
	assert.IsType(t, []*datapb.Filter{}, res)
	assert.Equal(t, 2, len(res))
	assert.NotNil(t, res[0].Key)
	assert.IsType(t, &datapb.PropertyKey{}, res[0].Key)
	assert.Equal(t, "test-name-0", res[0].Key.Name)
	assert.Equal(t, datapb.PropertyKey_Type(4), res[0].Key.Type)
	assert.Equal(t, &dp, res[0].Key.NumberDecimalPlaces)
	assert.Equal(t, 1, len(res[0].Conditions))
	assert.NotNil(t, res[1].Key)
	assert.IsType(t, &datapb.PropertyKey{}, res[1].Key)
	assert.Equal(t, "test-name-1", res[1].Key.Name)
	assert.Equal(t, datapb.PropertyKey_Type(9), res[1].Key.Type)
	assert.Equal(t, &dp, res[1].Key.NumberDecimalPlaces)
	assert.Equal(t, 1, len(res[1].Conditions))
	assert.Equal(t, datapb.Condition_OPERATOR_LESS_THAN_OR_EQUAL, res[1].Conditions[0].Operator)
	assert.Equal(t, "25", res[1].Conditions[0].Value)
}

func TestSpecFiltersFromProto(t *testing.T) {
	dp := uint64(12)
	pf := []*datapb.Filter{
		{
			Key: &datapb.PropertyKey{
				Name:                "test-proto-name-0",
				Type:                datapb.PropertyKey_TYPE_EMPTY,
				NumberDecimalPlaces: &dp,
			},
			Conditions: []*datapb.Condition{
				{
					Operator: datapb.Condition_OPERATOR_EQUALS,
					Value:    "21",
				},
				{
					Operator: datapb.Condition_OPERATOR_UNSPECIFIED,
					Value:    "17",
				},
			},
		},
	}

	res := common.SpecFiltersFromProto(pf)

	assert.NotNil(t, res)
	assert.IsType(t, []*common.SpecFilter{}, res)
	assert.Equal(t, 1, len(res))
	assert.Equal(t, "test-proto-name-0", res[0].Key.Name)
	assert.Equal(t, common.SpecPropertyKeyType(1), res[0].Key.Type)
	assert.Equal(t, dp, *res[0].Key.NumberDecimalPlaces)
	assert.Equal(t, 2, len(res[0].Conditions))
	assert.Equal(t, datapb.Condition_Operator(1), res[0].Conditions[0].Operator)
	assert.Equal(t, "21", res[0].Conditions[0].Value)
	assert.Equal(t, datapb.Condition_Operator(0), res[0].Conditions[1].Operator)
	assert.Equal(t, "17", res[0].Conditions[1].Value)
}

func TestSpecFiltersDeepClone(t *testing.T) {
	dp := uint64(12)
	sf := common.SpecFilter{
		Key: &common.SpecPropertyKey{
			Name:                "test-name-1",
			Type:                common.SpecPropertyKeyType(9),
			NumberDecimalPlaces: &dp,
		},
		Conditions: []*common.SpecCondition{
			{
				Operator: common.SpecConditionOperator(5),
				Value:    "25",
			},
		},
	}

	res := sf.DeepClone()
	assert.NotNil(t, res)
	assert.IsType(t, &common.SpecFilter{}, res)
	assert.IsType(t, &common.SpecPropertyKey{}, res.Key)
	assert.Equal(t, "test-name-1", res.Key.Name)
	assert.Equal(t, common.SpecPropertyKeyType(9), res.Key.Type)
	assert.Equal(t, &dp, res.Key.NumberDecimalPlaces)
	assert.Equal(t, 1, len(res.Conditions))
	assert.Equal(t, common.SpecConditionOperator(5), res.Conditions[0].Operator)
	assert.Equal(t, "25", res.Conditions[0].Value)
}
