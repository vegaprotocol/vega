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
	"time"

	"code.vegaprotocol.io/vega/libs/ptr"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
	datapb "code.vegaprotocol.io/vega/protos/vega/data/v1"
)

type TimeTrigger struct {
	// this is optional to reflect the proto
	// but it's actually always gonna be set by the governance
	From               *time.Time
	RepeatEverySeconds int64
	nextTrigger        *time.Time
}

func (t TimeTrigger) String() string {
	return fmt.Sprintf(
		"from(%v) repeatEvery(%d) nextTrigger(%v)",
		t.From,
		t.RepeatEverySeconds,
		t.nextTrigger,
	)
}

func (t TimeTrigger) IntoProto() *vegapb.InternalTimeTrigger {
	var from *int64
	var every int64
	if t.From != nil {
		from = ptr.From(t.From.Unix())
	}

	return &vegapb.InternalTimeTrigger{
		Initial: from,
		Every:   every,
	}
}

func (t *TimeTrigger) DeepClone() *TimeTrigger {
	var from *time.Time
	if from != nil {
		from = ptr.From(*t.From)
	}
	var lastTrigger *time.Time
	if t.nextTrigger != nil {
		lastTrigger = ptr.From(*t.nextTrigger)
	}

	return &TimeTrigger{
		From:               from,
		RepeatEverySeconds: t.RepeatEverySeconds,
		nextTrigger:        lastTrigger,
	}
}

func TimeTriggerFromProto(
	protoTrigger *vegapb.InternalTimeTrigger,
	timeNow time.Time,
) *TimeTrigger {
	var from *time.Time
	if protoTrigger.Initial != nil {
		from = ptr.From(time.Unix(*protoTrigger.Initial, 0))
	}
	tt := &TimeTrigger{
		From:               from,
		RepeatEverySeconds: protoTrigger.Every,
	}

	tt.setNextTrigger(timeNow)
	return tt
}

func (t *TimeTrigger) IsTriggered(timeNow time.Time) bool {
	if t.nextTrigger.Before(timeNow) {
		t.nextTrigger.Add(time.Duration(t.RepeatEverySeconds) * time.Second)
		return true
	}

	return false
}

func (t *TimeTrigger) setNextTrigger(timeNow time.Time) {
	if t.From == nil {
		panic("from time is nil")
	}

	t.nextTrigger = ptr.From(*t.From)

	// if from > now, we never been triggered, so we can set
	// nextTrigger to the from
	if t.From.After(timeNow) {
		return
	}

	// if from is in the past though, that means that we
	// have been triggered already, and we need to find
	// when is the next trigger
	for t.nextTrigger.Before(timeNow) {
		t.nextTrigger.Add(time.Duration(t.RepeatEverySeconds) * time.Second)
	}

}

type TimeTriggers []*TimeTrigger

func (t TimeTriggers) IntoProto() []*vegapb.InternalTimeTrigger {
	protos := make([]*vegapb.InternalTimeTrigger, 0, len(t))
	for _, v := range t {
		protos = append(protos, v.IntoProto())
	}

	return protos
}

func (t TimeTriggers) AnyTriggered(timeNow time.Time) bool {
	var ret bool
	for _, v := range t {
		ret = ret || v.IsTriggered(timeNow)
	}

	return ret
}

func (sc TimeTriggers) String() string {
	if sc == nil {
		return "[]"
	}
	strs := []string{}
	for _, c := range sc {
		strs = append(strs, c.String())
	}
	return "[" + strings.Join(strs, ", ") + "]"
}

func TimeTriggersFromProto(
	protos []*vegapb.InternalTimeTrigger,
	timeNow time.Time,
) []*TimeTrigger {
	triggers := make([]*TimeTrigger, 0, len(protos))
	for _, v := range protos {
		triggers = append(triggers, TimeTriggerFromProto(v, timeNow))
	}
	return triggers
}

func DeepCloneTimeTriggers(protos []*TimeTrigger) []*TimeTrigger {
	oth := make([]*TimeTrigger, 0, len(protos))
	for _, v := range protos {
		oth = append(oth, v.DeepClone())
	}
	return oth
}

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
