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

package entities_test

import (
	"testing"
	"time"

	dstypes "code.vegaprotocol.io/vega/core/datasource/common"
	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/protos/vega"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
	datapb "code.vegaprotocol.io/vega/protos/vega/data/v1"
	v1 "code.vegaprotocol.io/vega/protos/vega/data/v1"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestDataSourceDefinitionGetOracle(t *testing.T) {
	t.Run("nil source definition", func(t *testing.T) {
		ds := &entities.DataSourceDefinition{}
		r, err := ds.GetOracle()

		assert.Nil(t, err)
		assert.IsType(t, r, &entities.DataSourceSpecConfiguration{})
		assert.NotNil(t, r)
		assert.Equal(t, 0, len(r.Signers))
		assert.Equal(t, 0, len(r.Filters))
	})

	t.Run("empty source definition", func(t *testing.T) {
		ds := &entities.DataSourceDefinition{&vega.DataSourceDefinition{}}
		r, err := ds.GetOracle()

		assert.Nil(t, err)
		assert.IsType(t, r, &entities.DataSourceSpecConfiguration{})
		assert.NotNil(t, r)
		assert.Equal(t, 0, len(r.Signers))
		assert.Equal(t, 0, len(r.Filters))
	})

	t.Run("non-empty external data source definition", func(t *testing.T) {
		t.Run("data source oracle", func(t *testing.T) {
			ds := &entities.DataSourceDefinition{
				vega.NewDataSourceDefinition(
					vegapb.DataSourceContentTypeOracle,
				).SetOracleConfig(
					&vega.DataSourceDefinitionExternal_Oracle{
						Oracle: &vegapb.DataSourceSpecConfiguration{
							Signers: dstypes.SignersIntoProto(
								[]*dstypes.Signer{dstypes.CreateSignerFromString("0xTESTSIGN", dstypes.SignerTypePubKey)},
							),
							Filters: []*datapb.Filter{
								{
									Key: &datapb.PropertyKey{
										Name: "trading.terminated",
										Type: datapb.PropertyKey_TYPE_BOOLEAN,
									},
									Conditions: []*datapb.Condition{
										{
											Operator: datapb.Condition_OPERATOR_EQUALS,
											Value:    "12",
										},
									},
								},
							},
						},
					},
				),
			}

			r, err := ds.GetOracle()
			assert.Nil(t, err)
			assert.IsType(t, r, &entities.DataSourceSpecConfiguration{})
			assert.NotNil(t, r)
			assert.Equal(t, 1, len(r.Signers))
			assert.Equal(t, "\x00TESTSIGN", string(r.Signers[0]))

			assert.Equal(t, 1, len(r.Filters))
			filters := r.Filters
			assert.Equal(t, 1, len(filters))
			assert.Equal(t, 1, len(filters[0].Conditions))
			assert.Equal(t, datapb.Condition_Operator(1), filters[0].Conditions[0].Operator)
			assert.Equal(t, "12", filters[0].Conditions[0].Value)
			assert.Equal(t, "trading.terminated", filters[0].Key.Name)
			assert.Equal(t, datapb.PropertyKey_TYPE_BOOLEAN, filters[0].Key.Type)
		})

		t.Run("data source ethereum oracle", func(t *testing.T) {
			timeNow := uint64(time.Now().UnixNano())
			ds := &entities.DataSourceDefinition{
				vega.NewDataSourceDefinition(
					vegapb.DataSourceContentTypeEthOracle,
				).SetOracleConfig(
					&vega.DataSourceDefinitionExternal_EthOracle{
						EthOracle: &vegapb.EthCallSpec{
							Address: "test-eth-address",
							Abi:     "5",
							Method:  "test-method",
							Args: []*structpb.Value{
								structpb.NewStringValue("test-arg-value"),
							},
							Trigger: &vegapb.EthCallTrigger{
								Trigger: &vegapb.EthCallTrigger_TimeTrigger{
									TimeTrigger: &vegapb.EthTimeTrigger{
										Initial: &timeNow,
									},
								},
							},
							Filters: []*v1.Filter{
								{
									Key: &v1.PropertyKey{
										Name: "test-key",
										Type: v1.PropertyKey_Type(1),
									},
									Conditions: []*datapb.Condition{
										{
											Operator: datapb.Condition_OPERATOR_EQUALS,
											Value:    "12",
										},
									},
								},
							},
							Normalisers: []*vegapb.Normaliser{
								{
									Name:       "test-normaliser-name",
									Expression: "test-normaliser-expression",
								},
							},
						},
					},
				),
			}

			r, err := ds.GetEthOracle()
			assert.Nil(t, err)
			assert.IsType(t, r, &entities.EthCallSpec{})
			assert.NotNil(t, r)
			assert.Equal(t, "test-eth-address", r.Address)
			assert.Equal(t, []byte("5"), r.Abi)
			assert.Equal(t, "test-method", r.Method)
			assert.Equal(t, []string{"\"test-arg-value\""}, r.ArgsJson)
			assert.Equal(t, 1, len(r.Filters))
			filters := r.Filters
			assert.Equal(t, 1, len(filters))
			assert.Equal(t, 1, len(filters[0].Conditions))
			assert.Equal(t, "test-key", filters[0].Key.Name)
			assert.Equal(t, 1, len(r.Normalisers))
			assert.Equal(t, "test-normaliser-name", r.Normalisers[0].Name)
			assert.Equal(t, "test-normaliser-expression", r.Normalisers[0].Expression)
		})
	})

	t.Run("non-empty internal data source definition", func(t *testing.T) {
		ds := &entities.DataSourceDefinition{
			vega.NewDataSourceDefinition(
				vegapb.DataSourceContentTypeInternalTimeTermination,
			).SetTimeTriggerConditionConfig(
				[]*datapb.Condition{
					{
						Operator: datapb.Condition_OPERATOR_GREATER_THAN,
						Value:    "12",
					},
				},
			),
		}

		r, err := ds.GetOracle()
		assert.Nil(t, err)
		assert.IsType(t, r, &entities.DataSourceSpecConfiguration{})
		assert.NotNil(t, r)
		assert.Equal(t, 0, len(r.Signers))
		assert.Equal(t, 0, len(r.Filters))
	})
}

func TestDataSourceDefinitionGetInternalTimeTrigger(t *testing.T) {
	t.Run("nil source definition", func(t *testing.T) {
		ds := &entities.DataSourceDefinition{}
		r := ds.GetInternalTimeTrigger()

		assert.IsType(t, r, &entities.DataSourceSpecConfigurationTime{})
		assert.NotNil(t, r)
		assert.Equal(t, 0, len(r.Conditions))
	})

	t.Run("empty source definition", func(t *testing.T) {
		ds := &entities.DataSourceDefinition{&vega.DataSourceDefinition{}}
		r := ds.GetInternalTimeTrigger()

		assert.IsType(t, r, &entities.DataSourceSpecConfigurationTime{})
		assert.NotNil(t, r)
		assert.Equal(t, 0, len(r.Conditions))
	})

	t.Run("non-empty external data source definition", func(t *testing.T) {
		ds := &entities.DataSourceDefinition{
			vega.NewDataSourceDefinition(
				vegapb.DataSourceContentTypeOracle,
			).SetOracleConfig(
				&vega.DataSourceDefinitionExternal_Oracle{
					Oracle: &vegapb.DataSourceSpecConfiguration{
						Signers: dstypes.SignersIntoProto(
							[]*dstypes.Signer{dstypes.CreateSignerFromString("0xTESTSIGN", dstypes.SignerTypePubKey)},
						),
						Filters: []*datapb.Filter{
							{
								Key: &datapb.PropertyKey{
									Name: "trading.terminated",
									Type: datapb.PropertyKey_TYPE_BOOLEAN,
								},
								Conditions: []*datapb.Condition{
									{
										Operator: datapb.Condition_OPERATOR_EQUALS,
										Value:    "12",
									},
								},
							},
						},
					},
				},
			),
		}

		r := ds.GetInternalTimeTrigger()
		assert.IsType(t, r, &entities.DataSourceSpecConfigurationTime{})
		assert.NotNil(t, r)

		assert.Equal(t, 0, len(r.Conditions))
	})

	t.Run("non-empry internal data source definition", func(t *testing.T) {
		ds := &entities.DataSourceDefinition{
			vega.NewDataSourceDefinition(
				vegapb.DataSourceContentTypeInternalTimeTermination,
			).SetTimeTriggerConditionConfig(
				[]*datapb.Condition{
					{
						Operator: datapb.Condition_OPERATOR_GREATER_THAN,
						Value:    "12",
					},
				},
			),
		}

		r := ds.GetInternalTimeTrigger()
		assert.IsType(t, r, &entities.DataSourceSpecConfigurationTime{})
		assert.NotNil(t, r)

		assert.Equal(t, 1, len(r.Conditions))
		assert.Equal(t, datapb.Condition_Operator(2), r.Conditions[0].Operator)
		assert.Equal(t, "12", r.Conditions[0].Value)
	})
}

func TestDataSourceDefinitionGetSigners(t *testing.T) {
	t.Run("nil source definition", func(t *testing.T) {
		ds := &entities.DataSourceDefinition{}
		r, err := ds.GetSigners()

		assert.Nil(t, err)
		assert.IsType(t, r, entities.Signers{})
		assert.NotNil(t, r)
		assert.Equal(t, 0, len(r))
	})

	t.Run("empty source definition", func(t *testing.T) {
		ds := &entities.DataSourceDefinition{&vega.DataSourceDefinition{}}
		r, err := ds.GetSigners()

		assert.Nil(t, err)
		assert.IsType(t, r, entities.Signers{})
		assert.NotNil(t, r)
		assert.Equal(t, 0, len(r))
	})

	t.Run("non-empty external data source definition", func(t *testing.T) {
		ds := &entities.DataSourceDefinition{
			vega.NewDataSourceDefinition(
				vegapb.DataSourceContentTypeOracle,
			).SetOracleConfig(
				&vega.DataSourceDefinitionExternal_Oracle{
					Oracle: &vegapb.DataSourceSpecConfiguration{
						Signers: dstypes.SignersIntoProto(
							[]*dstypes.Signer{dstypes.CreateSignerFromString("0xTESTSIGN", dstypes.SignerTypePubKey)},
						),
						Filters: []*datapb.Filter{
							{
								Key: &datapb.PropertyKey{
									Name: "trading.terminated",
									Type: datapb.PropertyKey_TYPE_BOOLEAN,
								},
								Conditions: []*datapb.Condition{
									{
										Operator: datapb.Condition_OPERATOR_EQUALS,
										Value:    "12",
									},
								},
							},
						},
					},
				},
			),
		}

		r, err := ds.GetSigners()

		assert.Nil(t, err)
		assert.IsType(t, r, entities.Signers{})
		assert.NotNil(t, r)
		assert.Equal(t, 1, len(r))
		assert.Equal(t, "\x00TESTSIGN", string(r[0]))
	})

	t.Run("non-empry internal data source definition", func(t *testing.T) {
		ds := &entities.DataSourceDefinition{
			vega.NewDataSourceDefinition(
				vegapb.DataSourceContentTypeInternalTimeTermination,
			).SetTimeTriggerConditionConfig(
				[]*datapb.Condition{
					{
						Operator: datapb.Condition_OPERATOR_GREATER_THAN,
						Value:    "12",
					},
				},
			),
		}

		r, err := ds.GetSigners()
		assert.Nil(t, err)
		assert.IsType(t, r, entities.Signers{})
		assert.NotNil(t, r)
		assert.Equal(t, 0, len(r))
	})
}

func TestDataSourceDefinitionGetFilters(t *testing.T) {
	t.Run("nil source definition", func(t *testing.T) {
		ds := &entities.DataSourceDefinition{}
		r := ds.GetFilters()

		assert.IsType(t, r, []entities.Filter{})
		assert.NotNil(t, r)
		assert.Equal(t, 0, len(r))
	})

	t.Run("empty source definition", func(t *testing.T) {
		ds := &entities.DataSourceDefinition{&vega.DataSourceDefinition{}}
		r := ds.GetFilters()

		assert.IsType(t, r, []entities.Filter{})
		assert.NotNil(t, r)
		assert.Equal(t, 0, len(r))
	})

	t.Run("non-empty external data source definition", func(t *testing.T) {
		ds := &entities.DataSourceDefinition{
			vega.NewDataSourceDefinition(
				vegapb.DataSourceContentTypeOracle,
			).SetOracleConfig(
				&vega.DataSourceDefinitionExternal_Oracle{
					Oracle: &vegapb.DataSourceSpecConfiguration{
						Signers: dstypes.SignersIntoProto(
							[]*dstypes.Signer{dstypes.CreateSignerFromString("0xTESTSIGN", dstypes.SignerTypePubKey)},
						),
						Filters: []*datapb.Filter{
							{
								Key: &datapb.PropertyKey{
									Name: "trading.terminated",
									Type: datapb.PropertyKey_TYPE_BOOLEAN,
								},
								Conditions: []*datapb.Condition{
									{
										Operator: datapb.Condition_OPERATOR_EQUALS,
										Value:    "12",
									},
								},
							},
						},
					},
				},
			),
		}

		r := ds.GetFilters()

		assert.IsType(t, r, []entities.Filter{})
		assert.NotNil(t, r)
		assert.Equal(t, 1, len(r))
		assert.Equal(t, 1, len(r[0].Conditions))
		assert.Equal(t, datapb.Condition_Operator(1), r[0].Conditions[0].Operator)
		assert.Equal(t, "12", r[0].Conditions[0].Value)
		assert.Equal(t, "trading.terminated", r[0].Key.Name)
		assert.Equal(t, datapb.PropertyKey_TYPE_BOOLEAN, r[0].Key.Type)
	})

	t.Run("non-empry internal data source definition", func(t *testing.T) {
		ds := &entities.DataSourceDefinition{
			vega.NewDataSourceDefinition(
				vegapb.DataSourceContentTypeInternalTimeTermination,
			).SetTimeTriggerConditionConfig(
				[]*datapb.Condition{
					{
						Operator: datapb.Condition_OPERATOR_GREATER_THAN,
						Value:    "12",
					},
				},
			),
		}

		r := ds.GetFilters()
		assert.IsType(t, r, []entities.Filter{})
		assert.NotNil(t, r)
		assert.Equal(t, 0, len(r))
	})
}

func TestDataSourceDefinitionGetConditions(t *testing.T) {
	t.Run("nil source definition", func(t *testing.T) {
		ds := &entities.DataSourceDefinition{}
		r := ds.GetConditions()

		assert.IsType(t, r, []entities.Condition{})
		assert.NotNil(t, r)
		assert.Equal(t, 0, len(r))
	})

	t.Run("empty source definition", func(t *testing.T) {
		ds := &entities.DataSourceDefinition{&vega.DataSourceDefinition{}}
		r := ds.GetConditions()

		assert.IsType(t, r, []entities.Condition{})
		assert.NotNil(t, r)
		assert.Equal(t, 0, len(r))
	})

	t.Run("non-empty external data source definition", func(t *testing.T) {
		ds := &entities.DataSourceDefinition{
			vega.NewDataSourceDefinition(
				vegapb.DataSourceContentTypeOracle,
			).SetOracleConfig(
				&vega.DataSourceDefinitionExternal_Oracle{
					Oracle: &vegapb.DataSourceSpecConfiguration{
						Signers: dstypes.SignersIntoProto(
							[]*dstypes.Signer{dstypes.CreateSignerFromString("0xTESTSIGN", dstypes.SignerTypePubKey)},
						),
						Filters: []*datapb.Filter{
							{
								Key: &datapb.PropertyKey{
									Name: "trading.terminated",
									Type: datapb.PropertyKey_TYPE_BOOLEAN,
								},
								Conditions: []*datapb.Condition{
									{
										Operator: datapb.Condition_OPERATOR_EQUALS,
										Value:    "12",
									},
								},
							},
						},
					},
				},
			),
		}

		r := ds.GetConditions()

		assert.IsType(t, r, []entities.Condition{})
		assert.NotNil(t, r)
		assert.Equal(t, 1, len(r))
		assert.Equal(t, "12", r[0].Value)
		assert.Equal(t, datapb.Condition_OPERATOR_EQUALS, r[0].Operator)
	})

	t.Run("non-empry internal data source definition", func(t *testing.T) {
		ds := &entities.DataSourceDefinition{
			vega.NewDataSourceDefinition(
				vegapb.DataSourceContentTypeInternalTimeTermination,
			).SetTimeTriggerConditionConfig(
				[]*datapb.Condition{
					{
						Operator: datapb.Condition_OPERATOR_GREATER_THAN,
						Value:    "10",
					},
				},
			),
		}

		r := ds.GetConditions()
		assert.IsType(t, r, []entities.Condition{})
		assert.NotNil(t, r)
		assert.Equal(t, 1, len(r))
		assert.Equal(t, datapb.Condition_Operator(2), r[0].Operator)
		assert.Equal(t, "10", r[0].Value)
	})
}

func TestDataSourceDefinitionFromProto(t *testing.T) {
	t.Run("nil source definition", func(t *testing.T) {
		ds := &entities.DataSourceDefinition{}
		r := entities.DataSourceDefinitionFromProto(ds.DataSourceDefinition)

		assert.IsType(t, r, entities.DataSourceDefinition{})
		assert.NotNil(t, r)
		assert.Nil(t, r.DataSourceDefinition)
	})

	t.Run("empty source definition", func(t *testing.T) {
		ds := &entities.DataSourceDefinition{&vega.DataSourceDefinition{}}
		r := entities.DataSourceDefinitionFromProto(ds.DataSourceDefinition)

		assert.IsType(t, r, entities.DataSourceDefinition{})
		assert.NotNil(t, r)
		assert.NotNil(t, r.DataSourceDefinition)
		assert.IsType(t, r.DataSourceDefinition, &vegapb.DataSourceDefinition{})
	})

	t.Run("non-empty external data source definition", func(t *testing.T) {
		ds := &entities.DataSourceDefinition{
			vega.NewDataSourceDefinition(
				vegapb.DataSourceContentTypeOracle,
			).SetOracleConfig(
				&vega.DataSourceDefinitionExternal_Oracle{
					Oracle: &vegapb.DataSourceSpecConfiguration{
						Signers: dstypes.SignersIntoProto(
							[]*dstypes.Signer{dstypes.CreateSignerFromString("0xTESTSIGN", dstypes.SignerTypePubKey)},
						),
						Filters: []*datapb.Filter{
							{
								Key: &datapb.PropertyKey{
									Name: "trading.terminated",
									Type: datapb.PropertyKey_TYPE_BOOLEAN,
								},
								Conditions: []*datapb.Condition{
									{
										Operator: datapb.Condition_OPERATOR_EQUALS,
										Value:    "12",
									},
								},
							},
						},
					},
				},
			),
		}

		r := entities.DataSourceDefinitionFromProto(ds.DataSourceDefinition)

		assert.IsType(t, r, entities.DataSourceDefinition{})
		assert.NotNil(t, r)

		assert.Nil(t, r.DataSourceDefinition.GetInternal())
		assert.NotNil(t, r.DataSourceDefinition.GetExternal())

		o := r.DataSourceDefinition.GetExternal().GetOracle()
		signers := o.Signers
		assert.Equal(t, 1, len(signers))
		assert.Equal(t, "0xTESTSIGN", signers[0].GetPubKey().Key)

		filters := o.Filters
		assert.Equal(t, 1, len(filters))
		assert.Equal(t, 1, len(filters[0].Conditions))
		assert.Equal(t, datapb.Condition_Operator(1), filters[0].Conditions[0].Operator)
		assert.Equal(t, "12", filters[0].Conditions[0].Value)
		assert.Equal(t, "trading.terminated", filters[0].Key.Name)
		assert.Equal(t, datapb.PropertyKey_TYPE_BOOLEAN, filters[0].Key.Type)
	})

	t.Run("non-empry internal data source definition", func(t *testing.T) {
		ds := &entities.DataSourceDefinition{
			vega.NewDataSourceDefinition(
				vegapb.DataSourceContentTypeInternalTimeTermination,
			).SetTimeTriggerConditionConfig(
				[]*datapb.Condition{
					{
						Operator: datapb.Condition_OPERATOR_GREATER_THAN,
						Value:    "10",
					},
				},
			),
		}

		r := entities.DataSourceDefinitionFromProto(ds.DataSourceDefinition)
		assert.IsType(t, r, entities.DataSourceDefinition{})
		assert.NotNil(t, r)

		assert.NotNil(t, r.DataSourceDefinition.GetInternal())
		assert.Nil(t, r.DataSourceDefinition.GetExternal())
		conditions := r.DataSourceDefinition.GetInternal().GetTime().Conditions
		assert.Equal(t, 1, len(conditions))
		assert.Equal(t, datapb.Condition_OPERATOR_GREATER_THAN, conditions[0].Operator)
		assert.Equal(t, "10", conditions[0].Value)
	})
}
