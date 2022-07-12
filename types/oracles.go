// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package types

import (
	"fmt"
	"strings"

	vegapb "code.vegaprotocol.io/protos/vega"
	oraclespb "code.vegaprotocol.io/protos/vega/oracles/v1"
)

type OracleSpecConfiguration struct {
	PubKeys []string
	Filters []*OracleSpecFilter
}

func (c OracleSpecConfiguration) String() string {
	return fmt.Sprintf(
		"pubKeys(%v) filters(%v)",
		c.PubKeys,
		c.Filters,
	)
}

func (c *OracleSpecConfiguration) IntoProto() *oraclespb.OracleSpecConfiguration {
	return &oraclespb.OracleSpecConfiguration{
		PubKeys: c.PubKeys,
		Filters: OracleSpecFilters(c.Filters).IntoProto(),
	}
}

func (c *OracleSpecConfiguration) DeepClone() *OracleSpecConfiguration {
	return &OracleSpecConfiguration{
		PubKeys: c.PubKeys,
		Filters: DeepCloneOracleSpecFilters(c.Filters),
	}
}

func (c OracleSpecConfiguration) ToOracleSpec() *OracleSpec {
	return &OracleSpec{
		ID:      oraclespb.NewID(c.PubKeys, OracleSpecFilters(c.Filters).IntoProto()),
		PubKeys: c.PubKeys,
		Filters: c.Filters,
	}
}

func OracleSpecConfigurationFromProto(protoConfig *oraclespb.OracleSpecConfiguration) *OracleSpecConfiguration {
	return &OracleSpecConfiguration{
		PubKeys: protoConfig.PubKeys,
		Filters: OracleSpecFiltersFromProto(protoConfig.Filters),
	}
}

type OracleSpecFilters []*OracleSpecFilter

func (os OracleSpecFilters) IntoProto() []*oraclespb.Filter {
	protoFilters := make([]*oraclespb.Filter, 0, len(os))
	for _, filter := range os {
		protoFilters = append(protoFilters, filter.IntoProto())
	}
	return protoFilters
}

func (os OracleSpecFilters) String() string {
	if os == nil {
		return "[]"
	}
	strs := make([]string, 0, len(os))
	for _, f := range os {
		strs = append(strs, f.String())
	}
	return "[" + strings.Join(strs, ", ") + "]"
}

type OracleSpecFilter struct {
	Key        *OracleSpecPropertyKey
	Conditions []*OracleSpecCondition
}

func (f OracleSpecFilter) String() string {
	return fmt.Sprintf(
		"key(%s) conditions(%v)",
		f.Key.String(),
		OracleSpecConditions(f.Conditions).String(),
	)
}

func (f *OracleSpecFilter) IntoProto() *oraclespb.Filter {
	return &oraclespb.Filter{
		Key:        f.Key.IntoProto(),
		Conditions: OracleSpecConditions(f.Conditions).IntoProto(),
	}
}

func (f *OracleSpecFilter) DeepClone() *OracleSpecFilter {
	return &OracleSpecFilter{
		Key:        f.Key.DeepClone(),
		Conditions: DeepCloneOracleSpecConditions(f.Conditions),
	}
}

func OracleSpecFilterFromProto(protoFilter *oraclespb.Filter) *OracleSpecFilter {
	return &OracleSpecFilter{
		Key:        OracleSpecPropertyKeyFromProto(protoFilter.Key),
		Conditions: OracleSpecConditionsFromProto(protoFilter.Conditions),
	}
}

func OracleSpecFiltersFromProto(protoFilters []*oraclespb.Filter) []*OracleSpecFilter {
	filters := make([]*OracleSpecFilter, 0, len(protoFilters))
	for _, protoFilter := range protoFilters {
		filters = append(filters, OracleSpecFilterFromProto(protoFilter))
	}
	return filters
}

func DeepCloneOracleSpecFilters(filters []*OracleSpecFilter) []*OracleSpecFilter {
	othFilters := make([]*OracleSpecFilter, 0, len(filters))
	for _, filter := range filters {
		othFilters = append(othFilters, filter.DeepClone())
	}
	return othFilters
}

type OracleSpecConditions []*OracleSpecCondition

func (cs OracleSpecConditions) IntoProto() []*oraclespb.Condition {
	protoConditions := make([]*oraclespb.Condition, 0, len(cs))
	for _, condition := range cs {
		protoConditions = append(protoConditions, condition.IntoProto())
	}
	return protoConditions
}

func (cs OracleSpecConditions) String() string {
	if cs == nil {
		return "[]"
	}
	strs := make([]string, 0, len(cs))
	for _, c := range cs {
		strs = append(strs, c.String())
	}
	return "[" + strings.Join(strs, ", ") + "]"
}

type OracleSpecCondition struct {
	Operator OracleSpecConditionOperator
	Value    string
}

func (c OracleSpecCondition) String() string {
	return fmt.Sprintf(
		"value(%s) operator(%s)",
		c.Value,
		c.Operator.String(),
	)
}

func (c OracleSpecCondition) IntoProto() *oraclespb.Condition {
	return &oraclespb.Condition{
		Operator: c.Operator,
		Value:    c.Value,
	}
}

func (c *OracleSpecCondition) DeepClone() *OracleSpecCondition {
	return &OracleSpecCondition{
		Operator: c.Operator,
		Value:    c.Value,
	}
}

func OracleSpecConditionFromProto(protoCondition *oraclespb.Condition) *OracleSpecCondition {
	return &OracleSpecCondition{
		Operator: protoCondition.Operator,
		Value:    protoCondition.Value,
	}
}

func OracleSpecConditionsFromProto(protoConditions []*oraclespb.Condition) []*OracleSpecCondition {
	conditions := make([]*OracleSpecCondition, 0, len(protoConditions))
	for _, protoCondition := range protoConditions {
		conditions = append(conditions, OracleSpecConditionFromProto(protoCondition))
	}
	return conditions
}

func DeepCloneOracleSpecConditions(conditions []*OracleSpecCondition) []*OracleSpecCondition {
	othConditions := make([]*OracleSpecCondition, 0, len(conditions))
	for _, condition := range conditions {
		othConditions = append(othConditions, condition.DeepClone())
	}
	return othConditions
}

type OracleSpecConditionOperator = oraclespb.Condition_Operator

type OracleSpecPropertyKey struct {
	Name string
	Type OracleSpecPropertyKeyType
}

func (k OracleSpecPropertyKey) String() string {
	return fmt.Sprintf(
		"name(%s) type(%s)",
		k.Name,
		k.Type.String(),
	)
}

func (k OracleSpecPropertyKey) IntoProto() *oraclespb.PropertyKey {
	return &oraclespb.PropertyKey{
		Name: k.Name,
		Type: k.Type,
	}
}

func (k *OracleSpecPropertyKey) DeepClone() *OracleSpecPropertyKey {
	return &OracleSpecPropertyKey{
		Name: k.Name,
		Type: k.Type,
	}
}

func OracleSpecPropertyKeyFromProto(protoKey *oraclespb.PropertyKey) *OracleSpecPropertyKey {
	return &OracleSpecPropertyKey{
		Name: protoKey.Name,
		Type: protoKey.Type,
	}
}

type OracleSpecPropertyKeyType = oraclespb.PropertyKey_Type

type OracleSpecBindingForFuture struct {
	SettlementPriceProperty    string
	TradingTerminationProperty string
}

func (b OracleSpecBindingForFuture) String() string {
	return fmt.Sprintf(
		"settlementPrice(%s) tradingTermination(%s)",
		b.SettlementPriceProperty,
		b.TradingTerminationProperty,
	)
}

func (b OracleSpecBindingForFuture) IntoProto() *vegapb.OracleSpecToFutureBinding {
	return &vegapb.OracleSpecToFutureBinding{
		SettlementPriceProperty:    b.SettlementPriceProperty,
		TradingTerminationProperty: b.TradingTerminationProperty,
	}
}

func (b *OracleSpecBindingForFuture) DeepClone() *OracleSpecBindingForFuture {
	return &OracleSpecBindingForFuture{
		SettlementPriceProperty:    b.SettlementPriceProperty,
		TradingTerminationProperty: b.TradingTerminationProperty,
	}
}

func OracleSpecBindingForFutureFromProto(o *vegapb.OracleSpecToFutureBinding) *OracleSpecBindingForFuture {
	return &OracleSpecBindingForFuture{
		SettlementPriceProperty:    o.SettlementPriceProperty,
		TradingTerminationProperty: o.TradingTerminationProperty,
	}
}

type OracleSpecStatus = oraclespb.OracleSpec_Status

type OracleSpec struct {
	ID        string
	CreatedAt int64
	UpdatedAt int64
	PubKeys   []string
	Filters   []*OracleSpecFilter
	Status    OracleSpecStatus
}

func (s *OracleSpec) IntoProto() *oraclespb.OracleSpec {
	cpyPks := make([]string, len(s.PubKeys))
	copy(cpyPks, s.PubKeys)

	return &oraclespb.OracleSpec{
		Id:        s.ID,
		CreatedAt: s.CreatedAt,
		UpdatedAt: s.UpdatedAt,
		PubKeys:   cpyPks,
		Filters:   OracleSpecFilters(s.Filters).IntoProto(),
		Status:    s.Status,
	}
}

func (s *OracleSpec) String() string {
	return fmt.Sprintf(
		"ID(%s) createdAt(%v) updatedAt(%v) pubKeys(%v) filters(%v) status(%s)",
		s.ID,
		s.CreatedAt,
		s.UpdatedAt,
		OracleSpecPubKeys(s.PubKeys).String(),
		OracleSpecFilters(s.Filters).String(),
		s.Status.String(),
	)
}

func OracleSpecFromProto(specProto *oraclespb.OracleSpec) *OracleSpec {
	return &OracleSpec{
		ID:        specProto.Id,
		CreatedAt: specProto.CreatedAt,
		UpdatedAt: specProto.UpdatedAt,
		PubKeys:   specProto.PubKeys,
		Filters:   OracleSpecFiltersFromProto(specProto.Filters),
		Status:    specProto.Status,
	}
}

type OracleSpecPubKeys []string

func (o OracleSpecPubKeys) String() string {
	return "[" + strings.Join(o, ", ") + "]"
}
