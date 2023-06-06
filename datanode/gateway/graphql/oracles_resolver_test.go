package gql

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	vegapb "code.vegaprotocol.io/vega/protos/vega"
	v1 "code.vegaprotocol.io/vega/protos/vega/data/v1"
)

func Test_oracleSpecResolver_DataSourceSpec(t *testing.T) {
	type args struct {
		in0 context.Context
		obj *vegapb.OracleSpec
	}
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
			wantJsn: `{"spec":{"id":"","createdAt":0,"updatedAt":null,"data":{"SourceType":{"External":{"SourceType":{"Oracle":{"signers":[{"Signer":{"PubKey":{"key":"key"}}},{"Signer":{"EthAddress":{"address":"address"}}}]}}}}},"status":"STATUS_ACTIVE"}}`,
			wantErr: assert.NoError,
		}, {
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
			wantJsn: `{"spec":{"id":"","createdAt":0,"updatedAt":null,"data":{"SourceType":{"Internal":{"SourceType":{"Time":{"conditions":[{"operator":12,"value":"blah"}]}}}}},"status":"STATUS_ACTIVE"}}`,
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
			assert.JSONEqf(t, tt.wantJsn, string(gotJsn), "mismatch:\n\twant: %s \n\tgot: %s", tt.wantJsn, string(gotJsn))
		})
	}
}
