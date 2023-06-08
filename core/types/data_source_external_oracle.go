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

func (s DataSourceSpecConfiguration) ToDataSourceDefinitionProto() (*vegapb.DataSourceDefinition, error) {
	return &vegapb.DataSourceDefinition{
		SourceType: &vegapb.DataSourceDefinition_External{
			External: &vegapb.DataSourceDefinitionExternal{
				SourceType: &vegapb.DataSourceDefinitionExternal_Oracle{
					Oracle: s.IntoProto(),
				},
			},
		},
	}, nil
}

// String returns the content of DataSourceSpecConfiguration as a string.
func (s DataSourceSpecConfiguration) String() string {
	signers := ""
	for i, signer := range s.Signers {
		if i == 0 {
			signers = signer.String()
		} else {
			signers = signers + fmt.Sprintf(", %s", signer.String())
		}
	}

	filters := ""
	for i, filter := range s.Filters {
		if i == 0 {
			filters = filter.String()
		} else {
			filters = filters + fmt.Sprintf(", %s", filter.String())
		}
	}
	return fmt.Sprintf(
		"signers(%v) filters(%v)",
		signers,
		filters,
	)
}

func (s DataSourceSpecConfiguration) DeepClone() dataSourceType {
	return DataSourceSpecConfiguration{
		Signers: s.Signers,
		Filters: DeepCloneDataSourceSpecFilters(s.Filters),
	}
}

// DataSourceSpecConfigurationFromProto tries to build the DataSourceSpecConfiguration object
// from the given proto object.
func DataSourceSpecConfigurationFromProto(protoConfig *vegapb.DataSourceSpecConfiguration) DataSourceSpecConfiguration {
	if protoConfig == nil {
		return DataSourceSpecConfiguration{}
	}

	return DataSourceSpecConfiguration{
		Filters: DataSourceSpecFiltersFromProto(protoConfig.Filters),
		Signers: SignersFromProto(protoConfig.Signers),
	}
}
