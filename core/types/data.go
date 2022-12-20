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
	"fmt"
	"strconv"
	"strings"

	vegapb "code.vegaprotocol.io/vega/protos/vega"
	datapb "code.vegaprotocol.io/vega/protos/vega/data/v1"
)

type ExternalDataSourceSpecConfiguration struct {
	DataSourceSpec *DataSourceSpecConfiguration
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

type DataSourceSpecPropertyKeyType = datapb.PropertyKey_Type

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

type DataSourceSpecStatus = vegapb.DataSourceSpec_Status

type DataSourceSpec struct {
	ID        string
	CreatedAt int64
	UpdatedAt int64
	Data      *DataSourceDefinition
	Status    DataSourceSpecStatus
}

func (s *DataSourceSpec) IntoProto() *vegapb.DataSourceSpec {
	config := &vegapb.DataSourceDefinition{}
	if s.Data != nil {
		config = s.Data.IntoProto()
	}

	return &vegapb.DataSourceSpec{
		Id:        s.ID,
		CreatedAt: s.CreatedAt,
		UpdatedAt: s.UpdatedAt,
		Data:      config,
		Status:    s.Status,
	}
}

func (s *DataSourceSpec) String() string {
	configAsString := ""
	if s.Data != nil {
		configAsString = s.Data.String()
	}

	return fmt.Sprintf(
		"ID(%s) createdAt(%v) updatedAt(%v) data(%s) status(%s)",
		s.ID,
		s.CreatedAt,
		s.UpdatedAt,
		configAsString,
		s.Status.String(),
	)
}

// /
// ToExternalDataSourceSpec wraps the DataSourceSpec receiver into ExternalDataSourceSpec.
// Used for aligning with required types in the code.
func (s *DataSourceSpec) ToExternalDataSourceSpec() *ExternalDataSourceSpec {
	return &ExternalDataSourceSpec{
		Spec: s,
	}
}

func DataSourceSpecFromProto(specProto *vegapb.DataSourceSpec) *DataSourceSpec {
	return &DataSourceSpec{
		ID:        specProto.Id,
		CreatedAt: specProto.CreatedAt,
		UpdatedAt: specProto.UpdatedAt,
		Data:      DataSourceDefinitionFromProto(specProto.Data),
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
	Name                string
	Type                DataSourceSpecPropertyKeyType
	NumberDecimalPlaces *uint64
}

func (k DataSourceSpecPropertyKey) String() string {
	return fmt.Sprintf(
		"name(%s) type(%s) decimals(%s)",
		k.Name,
		k.Type.String(),
		strconv.FormatUint(*k.NumberDecimalPlaces, 10),
	)
}

func (k DataSourceSpecPropertyKey) IntoProto() *datapb.PropertyKey {
	pk := &datapb.PropertyKey{
		Name:                k.Name,
		Type:                k.Type,
		NumberDecimalPlaces: k.NumberDecimalPlaces,
	}

	return pk
}

func (k *DataSourceSpecPropertyKey) DeepClone() *DataSourceSpecPropertyKey {
	c := k
	return c
}

func DataSourceSpecPropertyKeyFromProto(protoKey *datapb.PropertyKey) *DataSourceSpecPropertyKey {
	return &DataSourceSpecPropertyKey{
		Name:                protoKey.Name,
		Type:                protoKey.Type,
		NumberDecimalPlaces: protoKey.NumberDecimalPlaces,
	}
}

func DataSourceSpecPropertyKeyIsEmpty(key *DataSourceSpecPropertyKey) bool {
	if key.Name == "" && key.Type == 0 {
		return true
	}

	return false
}

type ExternalDataSourceSpec struct {
	Spec *DataSourceSpec
}

func (s *ExternalDataSourceSpec) IntoProto() *vegapb.ExternalDataSourceSpec {
	return &vegapb.ExternalDataSourceSpec{
		Spec: s.Spec.IntoProto(),
	}
}

func (s *ExternalDataSourceSpec) String() string {
	return s.Spec.String()
}

func ExternalDataSourceSpecFromProto(specProto *vegapb.ExternalDataSourceSpec) *ExternalDataSourceSpec {
	if specProto.Spec != nil {
		r := DataSourceSpecFromProto(specProto.Spec)
		return &ExternalDataSourceSpec{
			Spec: r,
		}
	}

	return &ExternalDataSourceSpec{
		Spec: &DataSourceSpec{},
	}
}
