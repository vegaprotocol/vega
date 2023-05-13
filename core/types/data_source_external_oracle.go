package types

import (
	"fmt"

	vegapb "code.vegaprotocol.io/vega/protos/vega"
	datapb "code.vegaprotocol.io/vega/protos/vega/data/v1"
)

// DataSourceSpecConfiguration is used only by Oracles without a type wrapper at the moment.
type DataSourceSpecConfiguration struct {
	Signers []*Signer
	Filters []*DataSourceSpecFilter
}

func (s *DataSourceSpecConfiguration) isDataSourceType() {}

func (s *DataSourceSpecConfiguration) oneOfProto() interface{} {
	return s
}

// /
// String returns the content of DataSourceSpecConfiguration as a string.
func (s *DataSourceSpecConfiguration) String() string {
	return fmt.Sprintf(
		"signers(%v) filters(%v)",
		s.Signers,
		s.Filters,
	)
}

// /
// IntoProto tries to build the proto object from DataSourceSpecConfiguration.
func (s *DataSourceSpecConfiguration) IntoProto() *vegapb.DataSourceSpecConfiguration {
	signers := []*datapb.Signer{}
	filters := []*datapb.Filter{}

	dsc := &vegapb.DataSourceSpecConfiguration{}
	if s != nil {
		if s.Signers != nil {
			signers = SignersIntoProto(s.Signers)
		}

		if s.Filters != nil {
			filters = DataSourceSpecFilters(s.Filters).IntoProto()
		}

		dsc = &vegapb.DataSourceSpecConfiguration{
			// SignersIntoProto returns a list of signers after checking the list length.
			Signers: signers,
			Filters: filters,
		}
	}

	return dsc
}

func (s *DataSourceSpecConfiguration) DeepClone() dataSourceType {
	return &DataSourceSpecConfiguration{
		Signers: s.Signers,
		Filters: DeepCloneDataSourceSpecFilters(s.Filters),
	}
}

// DataSourceSpecConfigurationFromProto tries to build the DataSourceSpecConfiguration object
// from the given proto object.
func DataSourceSpecConfigurationFromProto(protoConfig *vegapb.DataSourceSpecConfiguration) *DataSourceSpecConfiguration {
	ds := &DataSourceSpecConfiguration{}
	if protoConfig != nil {
		// SignersFromProto returns a list of signers after checking the list length.
		ds.Signers = SignersFromProto(protoConfig.Signers)
		ds.Filters = DataSourceSpecFiltersFromProto(protoConfig.Filters)
	}

	return ds
}

// This is the base data source.
type DataSourceDefinitionExternalOracle struct {
	Oracle *DataSourceSpecConfiguration
}

func (e *DataSourceDefinitionExternalOracle) isDataSourceType() {}

func (e *DataSourceDefinitionExternalOracle) String() string {
	if e.Oracle == nil {
		return ""
	}

	return e.Oracle.String()
}

// /
// IntoProto tries to build the proto object from DataSourceDefinitionExternalOracle.
func (e *DataSourceDefinitionExternalOracle) IntoProto() *vegapb.DataSourceDefinitionExternal_Oracle {
	eds := &vegapb.DataSourceSpecConfiguration{}

	if e.Oracle != nil {
		eds = e.Oracle.IntoProto()
	}

	return &vegapb.DataSourceDefinitionExternal_Oracle{
		Oracle: eds,
	}
}

func (e *DataSourceDefinitionExternalOracle) oneOfProto() interface{} {
	return e.IntoProto()
}

func (e *DataSourceDefinitionExternalOracle) DeepClone() dataSourceType {
	if e.Oracle == nil {
		return &DataSourceDefinitionExternalOracle{
			Oracle: &DataSourceSpecConfiguration{},
		}
	}

	return &DataSourceDefinitionExternalOracle{}
}

// DataSourceDefinitionExternalOracleFromProto tries to build the DataSourceDefinitionExternalOracle object
// from the given proto object.
func DataSourceDefinitionExternalOracleFromProto(protoConfig *vegapb.DataSourceDefinitionExternal_Oracle) *DataSourceDefinitionExternalOracle {
	eds := &DataSourceDefinitionExternalOracle{
		Oracle: &DataSourceSpecConfiguration{},
	}

	if protoConfig != nil {
		if protoConfig.Oracle != nil {
			eds.Oracle = DataSourceSpecConfigurationFromProto(protoConfig.Oracle)
		}
	}

	return eds
}
