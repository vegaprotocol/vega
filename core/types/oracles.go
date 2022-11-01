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

package types

import (
	vegapb "code.vegaprotocol.io/vega/protos/vega"
	datapb "code.vegaprotocol.io/vega/protos/vega/data/v1"
)

type OracleSpecConfiguration struct {
	ExternalDataSourceSpec *ExternalDataSourceSpecConfiguration
}

func OracleSpecConfigurationFromProto(protoConfig *datapb.DataSourceSpecConfiguration) *OracleSpecConfiguration {
	return &OracleSpecConfiguration{
		ExternalDataSourceSpec: &ExternalDataSourceSpecConfiguration{
			DataSourceSpec: &DataSourceSpecConfiguration{
				Signers: SignersFromProto(protoConfig.Signers),
				Filters: DataSourceSpecFiltersFromProto(protoConfig.Filters),
			},
		},
	}
}

type OracleSpecFilters []*OracleSpecFilter

type OracleSpecFilter struct {
	DataSourceSpec *DataSourceSpecFilter
}

func OracleSpecFilterFromProto(protoFilter *datapb.Filter) *OracleSpecFilter {
	return &OracleSpecFilter{
		DataSourceSpec: &DataSourceSpecFilter{
			Key:        OracleSpecPropertyKeyFromProto(protoFilter.Key),
			Conditions: OracleSpecConditionsFromProto(protoFilter.Conditions),
		},
	}
}

func OracleSpecFiltersFromProto(protoFilters []*datapb.Filter) []*OracleSpecFilter {
	osf := make([]*OracleSpecFilter, len(protoFilters))
	for i, pf := range protoFilters {
		osf[i].DataSourceSpec = DataSourceSpecFilterFromProto(pf)
	}
	return osf
}

func DeepCloneOracleSpecFilters(filters []*OracleSpecFilter) []*OracleSpecFilter {
	clonedFilters := make([]*OracleSpecFilter, len(filters))
	for i, f := range filters {
		clonedFilters[i] = &OracleSpecFilter{DataSourceSpec: f.DataSourceSpec.DeepClone()}
	}

	return clonedFilters
}

type OracleSpecConditions DataSourceSpecConditions

type OracleSpecPropertyKey = DataSourceSpecPropertyKey

func OracleSpecPropertyKeyFromProto(protoKey *datapb.PropertyKey) *OracleSpecPropertyKey {
	return &OracleSpecPropertyKey{
		Name: protoKey.Name,
		Type: protoKey.Type,
	}
}

type OracleSpecCondition = DataSourceSpecCondition

func OracleSpecConditionsFromProto(protoCondition []*datapb.Condition) []*OracleSpecCondition {
	return DataSourceSpecConditionsFromProto(protoCondition)
}

type OracleSpecConditionOperator = DataSourceSpecConditionOperator

type OracleSpecToFutureBinding = DataSourceSpecToFutureBinding

func OracleSpecBindingForFutureFromProto(o *vegapb.DataSourceSpecToFutureBinding) *DataSourceSpecBindingForFuture {
	return DataSourceSpecBindingForFutureFromProto(o)
}

type OracleSpecBindingForFuture = DataSourceSpecBindingForFuture

type OracleSpec struct {
	ExternalDataSourceSpec *ExternalDataSourceSpec
}

func OracleSpecFromProto(specProto *datapb.OracleSpec) *OracleSpec {
	if specProto.ExternalDataSourceSpec != nil {
		return &OracleSpec{
			ExternalDataSourceSpecFromProto(specProto.ExternalDataSourceSpec),
		}
	}

	return &OracleSpec{
		ExternalDataSourceSpec: &ExternalDataSourceSpec{},
	}
}

type OracleSpecSigners = DataSourceSpecSigners

type OracleSpecStatus = DataSourceSpecStatus
