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

func (s SpecConfiguration) ToDefinitionProto() (*vegapb.DataSourceDefinition, error) {
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
	return SpecConfiguration{
		Signers: s.Signers,
		Filters: common.DeepCloneSpecFilters(s.Filters),
	}
}

func (s SpecConfiguration) GetFilters() []*common.SpecFilter {
	return s.Filters
}

// SpecConfigurationFromProto tries to build the SpecConfiguration object
// from the given proto object.
func SpecConfigurationFromProto(protoConfig *vegapb.DataSourceSpecConfiguration) SpecConfiguration {
	if protoConfig == nil {
		return SpecConfiguration{}
	}

	return SpecConfiguration{
		Filters: common.SpecFiltersFromProto(protoConfig.Filters),
		Signers: common.SignersFromProto(protoConfig.Signers),
	}
}
