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

package gql

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	vegapb "code.vegaprotocol.io/vega/protos/vega"
	v1 "code.vegaprotocol.io/vega/protos/vega/data/v1"

	"github.com/stretchr/testify/assert"
)

func Test_oracleSpecResolver_DataSourceSpec(t *testing.T) {
	type args struct {
		in0 context.Context
		obj *vegapb.OracleSpec
	}
	var timeBasic time.Time
	timeNow := uint64(timeBasic.UnixNano())
	tests := []struct {
		name    string
		o       oracleSpecResolver
		args    args
		wantJsn string
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "success: DataSourceDefinition_External",
			args: args{
				obj: &vegapb.OracleSpec{
					ExternalDataSourceSpec: &vegapb.ExternalDataSourceSpec{
						Spec: &vegapb.DataSourceSpec{
							Status: vegapb.DataSourceSpec_STATUS_ACTIVE,
							Data: &vegapb.DataSourceDefinition{
								SourceType: &vegapb.DataSourceDefinition_External{
									External: &vegapb.DataSourceDefinitionExternal{
										SourceType: &vegapb.DataSourceDefinitionExternal_Oracle{
											Oracle: &vegapb.DataSourceSpecConfiguration{
												Signers: []*v1.Signer{
													{
														Signer: &v1.Signer_PubKey{
															PubKey: &v1.PubKey{
																Key: "key",
															},
														},
													}, {
														Signer: &v1.Signer_EthAddress{
															EthAddress: &v1.ETHAddress{
																Address: "address",
															},
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			wantJsn: `{"spec":{"id":"","createdAt":0,"data":{"SourceType":{"External":{"SourceType":{"Oracle":{"signers":[{"Signer":{"PubKey":{"key":"key"}}},{"Signer":{"EthAddress":{"address":"address"}}}]}}}}},"status":"STATUS_ACTIVE"}}`,
			wantErr: assert.NoError,
		},
		{
			name: "success: DataSourceDefinition_Internal",
			args: args{
				obj: &vegapb.OracleSpec{
					ExternalDataSourceSpec: &vegapb.ExternalDataSourceSpec{
						Spec: &vegapb.DataSourceSpec{
							Status: vegapb.DataSourceSpec_STATUS_ACTIVE,
							Data: &vegapb.DataSourceDefinition{
								SourceType: &vegapb.DataSourceDefinition_Internal{
									Internal: &vegapb.DataSourceDefinitionInternal{
										SourceType: &vegapb.DataSourceDefinitionInternal_Time{
											Time: &vegapb.DataSourceSpecConfigurationTime{
												Conditions: []*v1.Condition{
													{
														Operator: 12,
														Value:    "blah",
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			wantJsn: `{"spec":{"id":"","createdAt":0,"data":{"SourceType":{"Internal":{"SourceType":{"Time":{"conditions":[{"operator":12,"value":"blah"}]}}}}},"status":"STATUS_ACTIVE"}}`,
			wantErr: assert.NoError,
		},
		{
			name: "success: DataSourceDefinition_External",
			args: args{
				obj: &vegapb.OracleSpec{
					ExternalDataSourceSpec: &vegapb.ExternalDataSourceSpec{
						Spec: &vegapb.DataSourceSpec{
							Status: vegapb.DataSourceSpec_STATUS_ACTIVE,
							Data: &vegapb.DataSourceDefinition{
								SourceType: &vegapb.DataSourceDefinition_External{
									External: &vegapb.DataSourceDefinitionExternal{
										SourceType: &vegapb.DataSourceDefinitionExternal_EthOracle{
											EthOracle: &vegapb.EthCallSpec{
												Address: "test-address",
												Abi:     "",
												Args:    nil,
												Method:  "stake",
												Trigger: &vegapb.EthCallTrigger{
													Trigger: &vegapb.EthCallTrigger_TimeTrigger{
														TimeTrigger: &vegapb.EthTimeTrigger{
															Initial: &timeNow,
														},
													},
												},
												RequiredConfirmations: uint64(0),
												Filters: []*v1.Filter{
													{
														Key: &v1.PropertyKey{
															Name: "property-name",
															Type: v1.PropertyKey_TYPE_BOOLEAN,
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			wantJsn: `{"spec":{"id":"","createdAt":0,"data":{"SourceType":{"External":{"SourceType":{"EthOracle":{"address":"test-address","method":"stake","trigger":{"Trigger":{"TimeTrigger":{"initial":11651379494838206464}}},"filters":[{"key":{"name":"property-name","type":4}}]}}}}},"status":"STATUS_ACTIVE"}}`,
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.o.DataSourceSpec(tt.args.in0, tt.args.obj)
			if !tt.wantErr(t, err, fmt.Sprintf("DataSourceSpec(%v, %v)", tt.args.in0, tt.args.obj)) {
				return
			}

			gotJsn, _ := json.Marshal(got)
			assert.JSONEqf(t, tt.wantJsn, string(gotJsn), "mismatch(%v):\n\twant: %s \n\tgot: %s", tt.name, tt.wantJsn, string(gotJsn))
		})
	}
}
