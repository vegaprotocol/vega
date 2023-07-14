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

package events_test

import (
	"context"
	"testing"

	dstypes "code.vegaprotocol.io/vega/core/datasource/common"
	"code.vegaprotocol.io/vega/core/events"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
	datapb "code.vegaprotocol.io/vega/protos/vega/data/v1"
	"github.com/stretchr/testify/assert"
)

func TestOracleSpecDeepClone(t *testing.T) {
	ctx := context.Background()
	pubKeys := []*dstypes.Signer{
		dstypes.CreateSignerFromString("PubKey1", dstypes.SignerTypePubKey),
		dstypes.CreateSignerFromString("PubKey1", dstypes.SignerTypePubKey),
	}

	os := vegapb.OracleSpec{
		ExternalDataSourceSpec: &vegapb.ExternalDataSourceSpec{
			Spec: &vegapb.DataSourceSpec{
				Id:        "Id",
				CreatedAt: 10000,
				UpdatedAt: 20000,
				Data: &vegapb.DataSourceDefinition{
					SourceType: &vegapb.DataSourceDefinition_External{
						External: &vegapb.DataSourceDefinitionExternal{
							SourceType: &vegapb.DataSourceDefinitionExternal_Oracle{
								Oracle: &vegapb.DataSourceSpecConfiguration{
									Signers: dstypes.SignersIntoProto(pubKeys),
									Filters: []*datapb.Filter{
										{
											Key: &datapb.PropertyKey{
												Name: "Name",
												Type: datapb.PropertyKey_TYPE_BOOLEAN,
											},
											Conditions: []*datapb.Condition{
												{
													Operator: datapb.Condition_OPERATOR_EQUALS,
													Value:    "Value",
												},
											},
										},
									},
								},
							},
						},
					},
				},
				Status: vegapb.DataSourceSpec_STATUS_ACTIVE,
			},
		},
	}

	osEvent := events.NewOracleSpecEvent(ctx, os)
	os2 := osEvent.OracleSpec()

	// Change the original values
	pk1 := dstypes.CreateSignerFromString("Changed1", dstypes.SignerTypePubKey)
	pk2 := dstypes.CreateSignerFromString("Changed2", dstypes.SignerTypePubKey)

	os.ExternalDataSourceSpec.Spec.Id = "Changed"
	os.ExternalDataSourceSpec.Spec.CreatedAt = 999
	os.ExternalDataSourceSpec.Spec.UpdatedAt = 999
	os.ExternalDataSourceSpec.Spec.Status = vegapb.DataSourceSpec_STATUS_UNSPECIFIED

	signers := []*datapb.Signer{
		pk1.IntoProto(), pk2.IntoProto(),
	}

	filters := []*datapb.Filter{
		{
			Key: &datapb.PropertyKey{
				Name: "Changed",
				Type: datapb.PropertyKey_TYPE_EMPTY,
			},
			Conditions: []*datapb.Condition{
				{
					Operator: datapb.Condition_OPERATOR_GREATER_THAN_OR_EQUAL,
					Value:    "Changed",
				},
			},
		},
	}

	os.ExternalDataSourceSpec.Spec.Data.SetOracleConfig(
		&vegapb.DataSourceDefinitionExternal_Oracle{
			Oracle: &vegapb.DataSourceSpecConfiguration{
				Signers: signers,
				Filters: filters,
			},
		},
	)

	// Check things have changed
	os2DataSourceSpec := os2.ExternalDataSourceSpec.Spec
	osDataSourceSpec := *os.ExternalDataSourceSpec.Spec
	assert.NotEqual(t, osDataSourceSpec.Id, os2DataSourceSpec.Id)
	assert.NotEqual(t, osDataSourceSpec.CreatedAt, os2DataSourceSpec.CreatedAt)
	assert.NotEqual(t, osDataSourceSpec.UpdatedAt, os2DataSourceSpec.UpdatedAt)
	assert.NotEqual(t, osDataSourceSpec.Data.GetSigners()[0], os2DataSourceSpec.Data.GetSigners()[0])
	assert.NotEqual(t, osDataSourceSpec.Data.GetSigners()[1], os2DataSourceSpec.Data.GetSigners()[1])
	assert.NotEqual(t, osDataSourceSpec.Data.GetFilters()[0].Key.Name, os2DataSourceSpec.Data.GetFilters()[0].Key.Name)
	assert.NotEqual(t, osDataSourceSpec.Data.GetFilters()[0].Key.Type, os2DataSourceSpec.Data.GetFilters()[0].Key.Type)
	assert.NotEqual(t, osDataSourceSpec.Data.GetFilters()[0].Conditions[0].Operator, os2DataSourceSpec.Data.GetFilters()[0].Conditions[0].Operator)
	assert.NotEqual(t, osDataSourceSpec.Data.GetFilters()[0].Conditions[0].Value, os2DataSourceSpec.Data.GetFilters()[0].Conditions[0].Value)
	assert.NotEqual(t, osDataSourceSpec.Status, os2DataSourceSpec.Status)
}
