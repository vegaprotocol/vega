// Copyright (c) 2022 Gobalsky Labs Limited
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

package types

import (
	"encoding/hex"
	"fmt"
	"strings"

	"code.vegaprotocol.io/vega/libs/crypto"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
	datapb "code.vegaprotocol.io/vega/protos/vega/data/v1"
)

type ExternalDataSourceSpecConfiguration struct {
	DataSourceSpec *DataSourceSpecConfiguration
}

type DataSourceSpecConfiguration struct {
	Signers []*Signer
	Filters []*DataSourceSpecFilter
}

func SpecID(signers []*Signer, filters []*datapb.Filter) string {
	buf := []byte{}
	for _, filter := range filters {
		s := filter.Key.Name + filter.Key.Type.String()
		for _, c := range filter.Conditions {
			s += c.Operator.String() + c.Value
		}

		buf = append(buf, []byte(s)...)
	}
	allSigners := []string{}
	for _, signer := range signers {
		allSigners = append(allSigners, signer.String())
	}
	buf = append(buf, []byte(strings.Join(allSigners, ""))...)

	return hex.EncodeToString(crypto.Hash(buf))
}

func (s DataSourceSpecConfiguration) String() string {
	return fmt.Sprintf(
		"signers(%v) filters(%v)",
		s.Signers,
		s.Filters,
	)
}

func (s *DataSourceSpecConfiguration) IntoProto() *datapb.DataSourceSpecConfiguration {
	return &datapb.DataSourceSpecConfiguration{
		// SignersIntoProto returns a list of signers after checking the list length.
		Signers: SignersIntoProto(s.Signers),
		Filters: DataSourceSpecFilters(s.Filters).IntoProto(),
	}
}

func (s *DataSourceSpecConfiguration) DeepClone() *DataSourceSpecConfiguration {
	return &DataSourceSpecConfiguration{
		Signers: s.Signers,
		Filters: DeepCloneDataSourceSpecFilters(s.Filters),
	}
}

func (s *DataSourceSpecConfiguration) ToDataSourceSpec() *DataSourceSpec {
	return &DataSourceSpec{
		ID: SpecID(
			s.Signers,
			DataSourceSpecFilters(s.Filters).IntoProto()),
		Config: &DataSourceSpecConfiguration{
			Signers: s.Signers,
			Filters: s.Filters,
		},
	}
}

func (s *DataSourceSpecConfiguration) ToExternalDataSourceSpec() *ExternalDataSourceSpec {
	return &ExternalDataSourceSpec{
		Spec: s.ToDataSourceSpec(),
	}
}

func DataSourceSpecConfigurationFromProto(protoConfig *datapb.DataSourceSpecConfiguration) *DataSourceSpecConfiguration {
	ds := &DataSourceSpecConfiguration{}
	if protoConfig != nil {
		// SignersFromProto returns a list of signers after checking the list length.
		ds.Signers = SignersFromProto(protoConfig.Signers)
		ds.Filters = DataSourceSpecFiltersFromProto(protoConfig.Filters)
	}

	return ds
}

type DataSourceSpecFilter struct {
	Key        *DataSourceSpecPropertyKey
	Conditions []*DataSourceSpecCondition
}

func DataSourceSpecFilterFromProto(protoFilter *datapb.Filter) *DataSourceSpecFilter {
	filter := &DataSourceSpecFilter{
		Key:        &DataSourceSpecPropertyKey{},
		Conditions: []*DataSourceSpecCondition{},
	}

	if protoFilter.Key != nil {
		filter.Key = DataSourceSpecPropertyKeyFromProto(protoFilter.Key)
	}

	filter.Conditions = DataSourceSpecConditionsFromProto(protoFilter.Conditions)
	return filter
}

func (f DataSourceSpecFilter) String() string {
	return fmt.Sprintf(
		"key(%s) conditions(%v)",
		f.Key.String(),
		DataSourceSpecConditions(f.Conditions).String(),
	)
}

// IntoProto return proto version of the filter receiver
// taking into account if its fields are empty.
func (f *DataSourceSpecFilter) IntoProto() *datapb.Filter {
	filter := &datapb.Filter{
		Key:        &datapb.PropertyKey{},
		Conditions: []*datapb.Condition{},
	}
	if f.Key != nil {
		filter.Key = f.Key.IntoProto()
	}

	if len(f.Conditions) > 0 {
		filter.Conditions = DataSourceSpecConditions(f.Conditions).IntoProto()
	}
	return filter
}

// DeepClone clones the filter receiver taking into account if its fields are empty.
func (f *DataSourceSpecFilter) DeepClone() *DataSourceSpecFilter {
	filter := &DataSourceSpecFilter{
		Key:        &DataSourceSpecPropertyKey{},
		Conditions: []*DataSourceSpecCondition{},
	}
	if f.Key != nil {
		filter.Key = f.Key.DeepClone()
	}

	if len(f.Conditions) > 0 {
		filter.Conditions = DeepCloneDataSourceSpecConditions(f.Conditions)
	}
	return filter
}

type DataSourceSpecFilters []*DataSourceSpecFilter

func (df DataSourceSpecFilters) IntoProto() []*datapb.Filter {
	protoFilters := make([]*datapb.Filter, 0, len(df))
	for _, filter := range df {
		protoFilters = append(protoFilters, filter.IntoProto())
	}
	return protoFilters
}

func (df DataSourceSpecFilters) String() string {
	if df == nil {
		return "[]"
	}
	strs := make([]string, 0, len(df))
	for _, f := range df {
		strs = append(strs, f.String())
	}
	return "[" + strings.Join(strs, ", ") + "]"
}

func DataSourceSpecFiltersFromProto(protoFilters []*datapb.Filter) []*DataSourceSpecFilter {
	if len(protoFilters) > 0 {
		dsf := make([]*DataSourceSpecFilter, len(protoFilters))
		if len(protoFilters) > 0 {
			for i, protoFilter := range protoFilters {
				dsf[i] = DataSourceSpecFilterFromProto(protoFilter)
			}
		}

		return dsf
	}
	return []*DataSourceSpecFilter{}
}

func DeepCloneDataSourceSpecFilters(filters []*DataSourceSpecFilter) []*DataSourceSpecFilter {
	clonedFilters := make([]*DataSourceSpecFilter, 0, len(filters))
	for _, filter := range filters {
		clonedFilters = append(clonedFilters, filter.DeepClone())
	}
	return clonedFilters
}

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
	strs := make([]string, 0, len(sc))
	for _, c := range sc {
		strs = append(strs, c.String())
	}
	return "[" + strings.Join(strs, ", ") + "]"
}

type DataSourceSpecCondition struct {
	Operator DataSourceSpecConditionOperator
	Value    string
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

func (c *DataSourceSpecCondition) DeepClone() *DataSourceSpecCondition {
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
		othConditions = append(othConditions, condition.DeepClone())
	}
	return othConditions
}

type DataSourceSpecPropertyKeyType = datapb.PropertyKey_Type

type DataSourceSpecConditionOperator = datapb.Condition_Operator

type DataSourceSpecToFutureBinding struct{}

type DataSourceSpecBindingForFuture struct {
	SettlementDataProperty     string
	TradingTerminationProperty string
}

func (b DataSourceSpecBindingForFuture) String() string {
	return fmt.Sprintf(
		"settlementData(%s) tradingTermination(%s)",
		b.SettlementDataProperty,
		b.TradingTerminationProperty,
	)
}

func (b DataSourceSpecBindingForFuture) IntoProto() *vegapb.DataSourceSpecToFutureBinding {
	return &vegapb.DataSourceSpecToFutureBinding{
		SettlementDataProperty:     b.SettlementDataProperty,
		TradingTerminationProperty: b.TradingTerminationProperty,
	}
}

func (b DataSourceSpecBindingForFuture) DeepClone() *DataSourceSpecBindingForFuture {
	return &DataSourceSpecBindingForFuture{
		SettlementDataProperty:     b.SettlementDataProperty,
		TradingTerminationProperty: b.TradingTerminationProperty,
	}
}

func DataSourceSpecBindingForFutureFromProto(o *vegapb.DataSourceSpecToFutureBinding) *DataSourceSpecBindingForFuture {
	return &DataSourceSpecBindingForFuture{
		SettlementDataProperty:     o.SettlementDataProperty,
		TradingTerminationProperty: o.TradingTerminationProperty,
	}
}

type DataSourceSpecStatus = datapb.DataSourceSpec_Status

type DataSourceSpec struct {
	ID        string
	CreatedAt int64
	UpdatedAt int64
	Config    *DataSourceSpecConfiguration
	Status    DataSourceSpecStatus
}

func (s *DataSourceSpec) IntoProto() *datapb.DataSourceSpec {
	config := &datapb.DataSourceSpecConfiguration{}
	if s.Config != nil {
		config = s.Config.IntoProto()
	}

	return &datapb.DataSourceSpec{
		Id:        s.ID,
		CreatedAt: s.CreatedAt,
		UpdatedAt: s.UpdatedAt,
		Config:    config,
		Status:    s.Status,
	}
}

func (s *DataSourceSpec) String() string {
	configAsString := ""
	if s.Config != nil {
		configAsString = s.Config.String()
	}

	return fmt.Sprintf(
		"ID(%s) createdAt(%v) updatedAt(%v) config(%s) status(%s)",
		s.ID,
		s.CreatedAt,
		s.UpdatedAt,
		configAsString,
		s.Status.String(),
	)
}

func (s *DataSourceSpec) ToExternalDataSourceSpec() *ExternalDataSourceSpec {
	return &ExternalDataSourceSpec{
		Spec: s,
	}
}

func DataSourceSpecFromProto(specProto *datapb.DataSourceSpec) *DataSourceSpec {
	return &DataSourceSpec{
		ID:        specProto.Id,
		CreatedAt: specProto.CreatedAt,
		UpdatedAt: specProto.UpdatedAt,
		Config:    DataSourceSpecConfigurationFromProto(specProto.Config),
		Status:    specProto.Status,
	}
}

type DataSourceSpecSigners []*Signer

func (s DataSourceSpecSigners) String() string {
	allSigners := []string{}
	for _, signer := range s {
		allSigners = append(allSigners, signer.String())
	}
	return "[" + strings.Join(allSigners, ", ") + "]"
}

type DataSourceSpecPropertyKey struct {
	Name string
	Type DataSourceSpecPropertyKeyType
}

func (k DataSourceSpecPropertyKey) String() string {
	return fmt.Sprintf(
		"name(%s) type(%s)",
		k.Name,
		k.Type.String(),
	)
}

func (k DataSourceSpecPropertyKey) IntoProto() *datapb.PropertyKey {
	return &datapb.PropertyKey{
		Name: k.Name,
		Type: k.Type,
	}
}

func (k *DataSourceSpecPropertyKey) DeepClone() *DataSourceSpecPropertyKey {
	return &DataSourceSpecPropertyKey{
		Name: k.Name,
		Type: k.Type,
	}
}

func DataSourceSpecPropertyKeyFromProto(protoKey *datapb.PropertyKey) *DataSourceSpecPropertyKey {
	return &DataSourceSpecPropertyKey{
		Name: protoKey.Name,
		Type: protoKey.Type,
	}
}

type ExternalDataSourceSpec struct {
	Spec *DataSourceSpec
}

func (s *ExternalDataSourceSpec) IntoProto() *datapb.ExternalDataSourceSpec {
	return &datapb.ExternalDataSourceSpec{
		Spec: s.Spec.IntoProto(),
	}
}

func (s *ExternalDataSourceSpec) String() string {
	return s.Spec.String()
}

func ExternalDataSourceSpecFromProto(specProto *datapb.ExternalDataSourceSpec) *ExternalDataSourceSpec {
	if specProto.Spec != nil {
		return &ExternalDataSourceSpec{
			Spec: DataSourceSpecFromProto(specProto.Spec),
		}
	}

	return &ExternalDataSourceSpec{
		Spec: &DataSourceSpec{},
	}
}
