// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

//lint:file-ignore ST1003 Ignore underscores in names, this is straigh copied from the proto package to ease introducing the domain types

package signedoracle

import (
	"fmt"

	"code.vegaprotocol.io/vega/core/datasource/common"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
	datapb "code.vegaprotocol.io/vega/protos/vega/data/v1"
)

// SpecConfiguration is used only by Oracles without a type wrapper at the moment.
type SpecConfiguration struct {
	Signers []*common.Signer
	Filters []*common.SpecFilter
}

// IntoProto tries to build the proto object from SpecConfiguration.
func (s *SpecConfiguration) IntoProto() *vegapb.DataSourceSpecConfiguration {
	signers := []*datapb.Signer{}
	filters := []*datapb.Filter{}

	dsc := &vegapb.DataSourceSpecConfiguration{}
	if s != nil {
		if s.Signers != nil {
			signers = common.SignersIntoProto(s.Signers)
		}

		if s.Filters != nil {
			filters = common.SpecFilters(s.Filters).IntoProto()
		}

		dsc = &vegapb.DataSourceSpecConfiguration{
			// SignersIntoProto returns a list of signers after checking the list length.
			Signers: signers,
			Filters: filters,
		}
	}

	return dsc
}

func (s *SpecConfiguration) ToDefinitionProto(_ uint64) (*vegapb.DataSourceDefinition, error) {
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
func (s SpecConfiguration) String() string {
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

func (s SpecConfiguration) DeepClone() common.DataSourceType {
	return &SpecConfiguration{
		Signers: s.Signers,
		Filters: common.DeepCloneSpecFilters(s.Filters),
	}
}

func (s SpecConfiguration) GetFilters() []*common.SpecFilter {
	return s.Filters
}

// SpecConfigurationFromProto tries to build the SpecConfiguration object
// from the given proto object.
func SpecConfigurationFromProto(protoConfig *vegapb.DataSourceSpecConfiguration) *SpecConfiguration {
	if protoConfig == nil {
		return &SpecConfiguration{}
	}

	return &SpecConfiguration{
		Filters: common.SpecFiltersFromProto(protoConfig.Filters),
		Signers: common.SignersFromProto(protoConfig.Signers),
	}
}
