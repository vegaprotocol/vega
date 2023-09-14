// Copyright (c) 2023 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.DATANODE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by vers

package entities_test

import (
	"testing"
	"time"

	dstypes "code.vegaprotocol.io/vega/core/datasource/common"
	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/protos/vega"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
	datapb "code.vegaprotocol.io/vega/protos/vega/data/v1"
	"github.com/stretchr/testify/assert"
	structpb "google.golang.org/protobuf/types/known/structpb"
)

func TestExternalDataSourceSpecFromProto(t *testing.T) {
	timeNow := time.Now()
	timeCreated := timeNow
	timeUpdated := timeCreated.Add(time.Second)
	timeNowu := uint64(timeNow.UnixNano())
	tHash := entities.TxHash("test-hash")

	t.Run("nil spec", func(t *testing.T) {
		r := entities.ExternalDataSourceSpecFromProto(nil, tHash, timeNow)

		assert.NotNil(t, r)
		assert.NotNil(t, r.Spec.Data)
		assert.Nil(t, r.Spec.Data.DataSourceDefinition)
	})

	t.Run("empty spec", func(t *testing.T) {
		r := entities.ExternalDataSourceSpecFromProto(
			&vegapb.ExternalDataSourceSpec{},
			tHash,
			timeNow,
		)
		assert.NotNil(t, r.Spec.Data)
		assert.Nil(t, r.Spec.Data.DataSourceDefinition)
		assert.Equal(t, entities.TxHash(""), r.Spec.TxHash)
	})

	t.Run("non-empty spec but empty data", func(t *testing.T) {
		s := &vega.DataSourceSpec{
			Id:        "test-id-0",
			CreatedAt: timeCreated.UnixNano(),
			UpdatedAt: timeUpdated.UnixNano(),
			Data:      nil,
		}

		r := entities.ExternalDataSourceSpecFromProto(&vegapb.ExternalDataSourceSpec{Spec: s}, tHash, timeNow)
		assert.NotNil(t, r.Spec.Data)
		assert.Nil(t, r.Spec.Data.DataSourceDefinition)
		assert.Equal(t, r.Spec.ID, entities.SpecID("test-id-0"))
		assert.Equal(t, r.Spec.CreatedAt.UnixNano(), timeCreated.UnixNano())
		assert.Equal(t, r.Spec.UpdatedAt.UnixNano(), timeUpdated.UnixNano())
		assert.Equal(t, r.Spec.TxHash, tHash)
		assert.Equal(t, entities.DataSourceSpecStatus(0), r.Spec.Status)
		assert.Equal(t, r.Spec.VegaTime, timeNow)
	})

	t.Run("with external data definition", func(t *testing.T) {
		s := &vegapb.DataSourceSpec{
			Id:        "test-id-1",
			CreatedAt: timeCreated.UnixNano(),
			UpdatedAt: timeUpdated.UnixNano(),
			Data: vegapb.NewDataSourceDefinition(
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

		r := entities.ExternalDataSourceSpecFromProto(&vegapb.ExternalDataSourceSpec{Spec: s}, tHash, timeNow)
		spec := r.Spec
		data := spec.Data
		assert.NotNil(t, spec)
		assert.NotNil(t, spec.Data.DataSourceDefinition)
		assert.Equal(t, r.Spec.ID, entities.SpecID("test-id-1"))
		assert.Equal(t, timeCreated.UnixNano(), r.Spec.CreatedAt.UnixNano())
		assert.Equal(t, timeUpdated.UnixNano(), r.Spec.UpdatedAt.UnixNano())
		assert.Equal(t, tHash, spec.TxHash)
		assert.Equal(t, entities.DataSourceSpecStatus(0), spec.Status)
		assert.Equal(t, r.Spec.VegaTime, timeNow)
		assert.Nil(t, data.GetInternal())
		assert.NotNil(t, data.GetExternal())

		o := data.GetExternal().GetOracle()
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

	t.Run("with external ethereum data definition", func(t *testing.T) {
		s := &vegapb.DataSourceSpec{
			Id:        "test-id-1",
			CreatedAt: timeCreated.UnixNano(),
			UpdatedAt: timeUpdated.UnixNano(),
			Data: vegapb.NewDataSourceDefinition(
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
									Initial: &timeNowu,
								},
							},
						},
						Filters: []*datapb.Filter{
							{
								Key: &datapb.PropertyKey{
									Name: "test-key",
									Type: datapb.PropertyKey_Type(2),
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

		r := entities.ExternalDataSourceSpecFromProto(&vegapb.ExternalDataSourceSpec{Spec: s}, tHash, timeNow)
		spec := r.Spec
		data := spec.Data
		assert.NotNil(t, spec)
		assert.NotNil(t, spec.Data.DataSourceDefinition)
		assert.Equal(t, r.Spec.ID, entities.SpecID("test-id-1"))
		assert.Equal(t, timeCreated.UnixNano(), r.Spec.CreatedAt.UnixNano())
		assert.Equal(t, timeUpdated.UnixNano(), r.Spec.UpdatedAt.UnixNano())
		assert.Equal(t, tHash, spec.TxHash)
		assert.Equal(t, entities.DataSourceSpecStatus(0), spec.Status)
		assert.Equal(t, r.Spec.VegaTime, timeNow)
		assert.Nil(t, data.GetInternal())
		assert.NotNil(t, data.GetExternal())

		o := data.GetExternal().GetEthOracle()
		assert.NotNil(t, o)

		assert.Equal(t, "test-eth-address", o.Address)
		assert.Equal(t, string("string_value:\"test-arg-value\""), o.Args[0].String())
		assert.Equal(t, string("5"), o.Abi)
		assert.Equal(t, "test-method", o.Method)
		filters := o.Filters
		assert.Equal(t, 1, len(filters))
		assert.Equal(t, 1, len(filters[0].Conditions))
		assert.Equal(t, datapb.Condition_Operator(1), filters[0].Conditions[0].Operator)
		assert.Equal(t, "12", filters[0].Conditions[0].Value)
		assert.Equal(t, "test-key", filters[0].Key.Name)
		assert.Equal(t, datapb.PropertyKey_TYPE_INTEGER, filters[0].Key.Type)
	})

	t.Run("with internal data definition", func(t *testing.T) {
		s := &vegapb.DataSourceSpec{
			Id:        "test-id-2",
			CreatedAt: timeCreated.UnixNano(),
			UpdatedAt: timeUpdated.UnixNano(),
			Data: vegapb.NewDataSourceDefinition(
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

		r := entities.ExternalDataSourceSpecFromProto(&vegapb.ExternalDataSourceSpec{Spec: s}, tHash, timeNow)
		spec := r.Spec
		data := spec.Data
		assert.NotNil(t, spec)
		assert.NotNil(t, spec.Data.DataSourceDefinition)
		assert.Equal(t, r.Spec.ID, entities.SpecID("test-id-2"))
		assert.Equal(t, r.Spec.CreatedAt.UnixNano(), timeCreated.UnixNano())
		assert.Equal(t, r.Spec.UpdatedAt.UnixNano(), timeUpdated.UnixNano())
		assert.Equal(t, tHash, spec.TxHash)
		assert.Equal(t, entities.DataSourceSpecStatus(0), spec.Status)
		assert.Equal(t, r.Spec.VegaTime, timeNow)
		assert.Nil(t, data.GetExternal())
		assert.NotNil(t, data.GetInternal())

		conditions := data.GetInternal().GetTime().Conditions
		assert.Equal(t, 1, len(conditions))
		assert.Equal(t, datapb.Condition_OPERATOR_GREATER_THAN, conditions[0].Operator)
		assert.Equal(t, "12", conditions[0].Value)
	})
}

func TestExternalDataSourceSpecToProto(t *testing.T) {
	timeNow := time.Now()
	timeCreated := timeNow
	timeUpdated := timeCreated.Add(time.Second)
	tHash := entities.TxHash("test-hash")
	timeNowu := uint64(timeNow.UnixNano())

	t.Run("nil spec", func(t *testing.T) {
		protoSpec := entities.ExternalDataSourceSpecFromProto(nil, tHash, timeNow).ToProto()
		assert.NotNil(t, protoSpec)
	})

	t.Run("empty spec", func(t *testing.T) {
		protoSpec := entities.ExternalDataSourceSpecFromProto(&vegapb.ExternalDataSourceSpec{}, tHash, timeNow).ToProto()
		assert.NotNil(t, protoSpec)
	})

	t.Run("non-empty spec but empty data", func(t *testing.T) {
		protoSpec := entities.ExternalDataSourceSpecFromProto(&vegapb.ExternalDataSourceSpec{
			Spec: &vegapb.DataSourceSpec{
				Id:        "test-id-01",
				CreatedAt: timeCreated.UnixNano(),
				UpdatedAt: timeUpdated.UnixNano(),
				Data:      nil,
			},
		}, tHash, timeNow).ToProto()
		assert.NotNil(t, protoSpec)
		assert.Equal(t, "", protoSpec.Spec.Id)
	})

	t.Run("with external data definition", func(t *testing.T) {
		s := &vegapb.DataSourceSpec{
			Id:        "test-id-02",
			CreatedAt: timeCreated.Unix(),
			UpdatedAt: timeUpdated.Unix(),
			Data: vegapb.NewDataSourceDefinition(
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

		protoResult := entities.ExternalDataSourceSpecFromProto(&vegapb.ExternalDataSourceSpec{Spec: s}, tHash, timeNow).ToProto()
		protoSpec := protoResult.GetSpec()
		assert.Equal(t, timeCreated.Unix(), protoSpec.CreatedAt)
		assert.Equal(t, timeUpdated.Unix(), protoSpec.UpdatedAt)

		oracleData := protoSpec.Data.GetExternal()
		assert.NotNil(t, oracleData.GetOracle())
		o := oracleData.GetOracle()
		assert.Equal(t, 1, len(o.Filters))
		assert.NotNil(t, o.GetFilters())
		assert.Equal(t, "trading.terminated", o.GetFilters()[0].Key.Name)
		assert.Equal(t, datapb.PropertyKey_TYPE_BOOLEAN, o.GetFilters()[0].Key.Type)
		assert.Equal(t, 1, len(o.GetFilters()[0].Conditions))
		assert.Equal(t, datapb.Condition_OPERATOR_EQUALS, o.GetFilters()[0].Conditions[0].Operator)
		assert.Equal(t, "12", o.GetFilters()[0].Conditions[0].Value)
		assert.NotNil(t, o.GetSigners())
		assert.Equal(t, "0xTESTSIGN", o.GetSigners()[0].GetPubKey().Key)
	})

	t.Run("with external ethereum spec data definition", func(t *testing.T) {
		s := &vegapb.DataSourceSpec{
			Id:        "test-id-02",
			CreatedAt: timeCreated.Unix(),
			UpdatedAt: timeUpdated.Unix(),
			Data: vegapb.NewDataSourceDefinition(
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
									Initial: &timeNowu,
								},
							},
						},
						Filters: []*datapb.Filter{
							{
								Key: &datapb.PropertyKey{
									Name: "test-key",
									Type: datapb.PropertyKey_Type(2),
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

		protoResult := entities.ExternalDataSourceSpecFromProto(&vegapb.ExternalDataSourceSpec{Spec: s}, tHash, timeNow).ToProto()
		protoSpec := protoResult.GetSpec()
		assert.Equal(t, timeCreated.Unix(), protoSpec.CreatedAt)
		assert.Equal(t, timeUpdated.Unix(), protoSpec.UpdatedAt)

		oracleData := protoSpec.Data.GetExternal()
		assert.NotNil(t, oracleData.GetEthOracle())
		o := oracleData.GetEthOracle()
		assert.NotNil(t, o)
		assert.Equal(t, "test-eth-address", o.Address)
		assert.Equal(t, string("string_value:\"test-arg-value\""), o.Args[0].String())
		assert.Equal(t, string("5"), o.Abi)
		assert.Equal(t, "test-method", o.Method)
		assert.Equal(t, 1, len(o.Filters))
		assert.NotNil(t, o.GetFilters())
		assert.Equal(t, "test-key", o.GetFilters()[0].Key.Name)
		assert.Equal(t, datapb.PropertyKey_TYPE_INTEGER, o.GetFilters()[0].Key.Type)
		assert.Equal(t, 1, len(o.GetFilters()[0].Conditions))
		assert.Equal(t, datapb.Condition_OPERATOR_EQUALS, o.GetFilters()[0].Conditions[0].Operator)
		assert.Equal(t, "12", o.GetFilters()[0].Conditions[0].Value)
	})

	t.Run("with internal data definition", func(t *testing.T) {
		s := &vegapb.DataSourceSpec{
			CreatedAt: timeCreated.Unix(),
			UpdatedAt: timeUpdated.Unix(),
			Data: vegapb.NewDataSourceDefinition(
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

		r := entities.ExternalDataSourceSpecFromProto(&vegapb.ExternalDataSourceSpec{Spec: s}, tHash, time.Now())
		protoResult := r.ToProto()
		protoSpec := protoResult.GetSpec()
		assert.Equal(t, timeCreated.Unix(), protoSpec.CreatedAt)
		assert.Equal(t, timeUpdated.Unix(), protoSpec.UpdatedAt)

		timeTermData := protoSpec.Data.GetInternal()
		assert.NotNil(t, timeTermData)
		timeTerms := timeTermData.GetTime()
		assert.Equal(t, 1, len(timeTerms.Conditions))
		assert.Equal(t, datapb.Condition_OPERATOR_GREATER_THAN, timeTerms.Conditions[0].Operator)
		assert.Equal(t, "12", timeTerms.Conditions[0].Value)
	})
}
