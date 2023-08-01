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

package common

import (
	"fmt"
	"strconv"
	"strings"

	vegapb "code.vegaprotocol.io/vega/protos/vega"
	datapb "code.vegaprotocol.io/vega/protos/vega/data/v1"
)

type SpecConditionOperator = datapb.Condition_Operator

// SpecCondition mirrors datapb.Condition type.
type SpecCondition struct {
	Operator SpecConditionOperator
	Value    string
}

func (c SpecCondition) String() string {
	return fmt.Sprintf(
		"value(%s) operator(%s)",
		c.Value,
		c.Operator.String(),
	)
}

func (c SpecCondition) IntoProto() *datapb.Condition {
	return &datapb.Condition{
		Operator: c.Operator,
		Value:    c.Value,
	}
}

func (c *SpecCondition) DeepClone() *SpecCondition {
	return &SpecCondition{
		Operator: c.Operator,
		Value:    c.Value,
	}
}

func SpecConditionFromProto(protoCondition *datapb.Condition) *SpecCondition {
	return &SpecCondition{
		Operator: protoCondition.Operator,
		Value:    protoCondition.Value,
	}
}

type SpecConditions []*SpecCondition

func (sc SpecConditions) IntoProto() []*datapb.Condition {
	protoConditions := make([]*datapb.Condition, 0, len(sc))
	for _, condition := range sc {
		protoConditions = append(protoConditions, condition.IntoProto())
	}
	return protoConditions
}

func (sc SpecConditions) String() string {
	if sc == nil {
		return "[]"
	}
	strs := []string{}
	for _, c := range sc {
		strs = append(strs, c.String())
	}
	return "[" + strings.Join(strs, ", ") + "]"
}

func SpecConditionsFromProto(protoConditions []*datapb.Condition) []*SpecCondition {
	conditions := make([]*SpecCondition, 0, len(protoConditions))
	for _, protoCondition := range protoConditions {
		conditions = append(conditions, SpecConditionFromProto(protoCondition))
	}
	return conditions
}

func DeepCloneSpecConditions(conditions []*SpecCondition) []*SpecCondition {
	othConditions := make([]*SpecCondition, 0, len(conditions))
	for _, condition := range conditions {
		othConditions = append(othConditions, condition.DeepClone())
	}
	return othConditions
}

type SpecFilter struct {
	Key        *SpecPropertyKey
	Conditions []*SpecCondition
}

func SpecFilterFromProto(protoFilter *datapb.Filter) *SpecFilter {
	filter := &SpecFilter{
		Key:        &SpecPropertyKey{},
		Conditions: []*SpecCondition{},
	}

	if protoFilter.Key != nil {
		*filter.Key = *SpecPropertyKeyFromProto(protoFilter.Key)
	}

	filter.Conditions = SpecConditionsFromProto(protoFilter.Conditions)
	return filter
}

func (f SpecFilter) String() string {
	return fmt.Sprintf(
		"key(%s) conditions(%v)",
		f.Key.String(),
		SpecConditions(f.Conditions).String(),
	)
}

// IntoProto return proto version of the filter receiver
// taking into account if its fields are empty.
func (f *SpecFilter) IntoProto() *datapb.Filter {
	filter := &datapb.Filter{
		Key:        &datapb.PropertyKey{},
		Conditions: []*datapb.Condition{},
	}
	if f.Key != nil {
		filter.Key = f.Key.IntoProto()
	}

	if len(f.Conditions) > 0 {
		filter.Conditions = SpecConditions(f.Conditions).IntoProto()
	}
	return filter
}

// DeepClone clones the filter receiver taking into account if its fields are empty.
func (f *SpecFilter) DeepClone() *SpecFilter {
	filter := &SpecFilter{
		Key:        &SpecPropertyKey{},
		Conditions: []*SpecCondition{},
	}
	if f.Key != nil {
		filter.Key = f.Key.DeepClone()
	}

	if len(f.Conditions) > 0 {
		filter.Conditions = DeepCloneSpecConditions(f.Conditions)
	}
	return filter
}

type SpecFilters []*SpecFilter

func (df SpecFilters) IntoProto() []*datapb.Filter {
	protoFilters := make([]*datapb.Filter, 0, len(df))
	for _, filter := range df {
		protoFilters = append(protoFilters, filter.IntoProto())
	}
	return protoFilters
}

func (df SpecFilters) String() string {
	if df == nil {
		return "[]"
	}
	strs := make([]string, 0, len(df))
	for _, f := range df {
		strs = append(strs, f.String())
	}
	return "[" + strings.Join(strs, ", ") + "]"
}

func SpecFiltersFromProto(protoFilters []*datapb.Filter) []*SpecFilter {
	dsf := make([]*SpecFilter, len(protoFilters))
	for i, protoFilter := range protoFilters {
		dsf[i] = SpecFilterFromProto(protoFilter)
	}
	return dsf
}

func DeepCloneSpecFilters(filters []*SpecFilter) []*SpecFilter {
	clonedFilters := make([]*SpecFilter, 0, len(filters))
	for _, filter := range filters {
		clonedFilters = append(clonedFilters, filter.DeepClone())
	}
	return clonedFilters
}

type SpecPropertyKeyType = datapb.PropertyKey_Type

type SpecStatus = vegapb.DataSourceSpec_Status

type SpecPropertyKey struct {
	Name                string
	Type                SpecPropertyKeyType
	NumberDecimalPlaces *uint64
}

func (k SpecPropertyKey) String() string {
	var dp string
	if k.NumberDecimalPlaces != nil {
		dp = strconv.FormatUint(*k.NumberDecimalPlaces, 10)
	}

	return fmt.Sprintf(
		"name(%s) type(%s) decimals(%s)",
		k.Name,
		k.Type.String(),
		dp,
	)
}

func (k SpecPropertyKey) IntoProto() *datapb.PropertyKey {
	pk := &datapb.PropertyKey{
		Name:                k.Name,
		Type:                k.Type,
		NumberDecimalPlaces: k.NumberDecimalPlaces,
	}

	return pk
}

func (k *SpecPropertyKey) DeepClone() *SpecPropertyKey {
	c := k
	return c
}

func SpecPropertyKeyFromProto(protoKey *datapb.PropertyKey) *SpecPropertyKey {
	return &SpecPropertyKey{
		Name:                protoKey.Name,
		Type:                protoKey.Type,
		NumberDecimalPlaces: protoKey.NumberDecimalPlaces,
	}
}

func SpecPropertyKeyIsEmpty(key *SpecPropertyKey) bool {
	if key == nil {
		return true
	}

	if key.Name == "" && key.Type == 0 {
		return true
	}

	return false
}
