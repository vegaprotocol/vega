package types_test

import (
	"errors"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/types"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
	datapb "code.vegaprotocol.io/vega/protos/vega/data/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestNewDataSourceDefinitionWith(t *testing.T) {
	t.Run("oracle", func(t *testing.T) {
		t.Run("empty", func(t *testing.T) {
			nds := types.NewDataSourceDefinitionWith(nil)
			assert.NotNil(t, nds)
			assert.IsType(t, &types.DataSourceDefinition{}, nds)
			assert.Nil(t, nds.Content())
		})

		t.Run("non-empty oracle", func(t *testing.T) {
			// Do we need to test that?
		})
	})

	t.Run("ethereum oracle", func(t *testing.T) {
	})

	t.Run("internal time termination", func(t *testing.T) {
	})
}

func TestNewDataSourceDefinition(t *testing.T) {
	ds := types.NewDataSourceDefinition(types.DataSourceContentTypeOracle)
	assert.NotNil(t, ds)
	cnt := ds.Content()
	assert.IsType(t, types.DataSourceSpecConfiguration{}, cnt)

	ds = types.NewDataSourceDefinition(types.DataSourceContentTypeEthOracle)
	assert.NotNil(t, ds)
	cnt = ds.Content()
	assert.IsType(t, types.EthCallSpec{}, cnt)

	ds = types.NewDataSourceDefinition(types.DataSourceContentTypeInternalTimeTermination)
	assert.NotNil(t, ds)
	cnt = ds.Content()
	assert.IsType(t, types.DataSourceSpecConfigurationTime{}, cnt)
}

func TestDataSourceDefinitionIntoProto(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		ds := &types.DataSourceDefinition{}
		protoDs := ds.IntoProto()
		assert.IsType(t, &vegapb.DataSourceDefinition{}, protoDs)
		assert.Nil(t, protoDs.SourceType)
	})

	t.Run("external dataSourceDefinition", func(t *testing.T) {
		t.Run("non-empty but no content", func(t *testing.T) {
			ds := &types.DataSourceDefinition{}
			protoDs := ds.IntoProto()
			assert.IsType(t, &vegapb.DataSourceDefinition{}, protoDs)
			assert.Nil(t, protoDs.SourceType)
		})

		t.Run("oracle", func(t *testing.T) {
			ds := types.NewDataSourceDefinitionWith(
				&types.DataSourceSpecConfiguration{
					Signers: []*types.Signer{
						{
							Signer: &types.SignerETHAddress{
								ETHAddress: &types.ETHAddress{
									Address: "test-eth-address-0",
								},
							},
						},
						{
							Signer: &types.SignerETHAddress{
								ETHAddress: &types.ETHAddress{
									Address: "test-eth-address-1",
								},
							},
						},
					},
					Filters: []*types.DataSourceSpecFilter{
						{
							Key: &types.DataSourceSpecPropertyKey{
								Name: "test-property-key-name-0",
								Type: types.DataSourceSpecPropertyKeyType(0),
							},
							Conditions: []*types.DataSourceSpecCondition{
								{
									Operator: types.DataSourceSpecConditionOperator(0),
									Value:    "12",
								},
							},
						},
					},
				},
			)

			dsProto := ds.IntoProto()
			assert.NotNil(t, dsProto.SourceType)
			assert.IsType(t, &vegapb.DataSourceDefinition_External{}, dsProto.SourceType)
			o := dsProto.GetExternal().GetOracle()
			assert.NotNil(t, o)
			assert.IsType(t, &vegapb.DataSourceSpecConfiguration{}, o)
			signers := dsProto.GetSigners()
			assert.Equal(t, 2, len(signers))
			assert.IsType(t, &datapb.Signer_EthAddress{}, signers[0].Signer)
			assert.Equal(t, "test-eth-address-0", signers[0].GetEthAddress().Address)
			assert.IsType(t, &datapb.Signer_EthAddress{}, signers[1].Signer)
			assert.Equal(t, "test-eth-address-1", signers[1].GetEthAddress().Address)
			filters := dsProto.GetFilters()
			assert.Equal(t, 1, len(filters))
			assert.IsType(t, &datapb.Filter{}, filters[0])
			assert.IsType(t, &datapb.PropertyKey{}, filters[0].Key)
			assert.Equal(t, "test-property-key-name-0", filters[0].Key.Name)
			assert.Equal(t, datapb.PropertyKey_Type(0), filters[0].Key.Type)
			assert.Equal(t, 1, len(filters[0].Conditions))
			assert.IsType(t, &datapb.Condition{}, filters[0].Conditions[0])
			assert.Equal(t, datapb.Condition_OPERATOR_UNSPECIFIED, filters[0].Conditions[0].Operator)
			assert.Equal(t, "12", filters[0].Conditions[0].Value)
		})

		t.Run("eth oracle", func(t *testing.T) {
			timeNow := time.Now()
			ds := types.NewDataSourceDefinitionWith(
				&types.EthCallSpec{
					Address: "some-eth-address",
					AbiJson: []byte(`
[
	{"inputs":
		[
			{"internalType":"uint256","name":"input","type":"uint256"}
		],
		"name":"get_uint256",
		"outputs":
			[
				{"internalType":"uint256","name":"","type":"uint256"}
			],
		"stateMutability":"pure",
		"type":"function"
	}
]
`),
					Method: "method",
					Trigger: &types.EthTimeTrigger{
						Initial: uint64(timeNow.UnixNano()),
					},
					RequiredConfirmations: 256,
					Filters: []*types.DataSourceSpecFilter{
						{
							Key: &types.DataSourceSpecPropertyKey{
								Name: "test-key-name-0",
								Type: types.DataSourceSpecPropertyKeyType(5),
							},
						},
					},
				})
			dsProto := ds.IntoProto()
			assert.NotNil(t, dsProto.SourceType)
			assert.IsType(t, &vegapb.DataSourceDefinition_External{}, dsProto.SourceType)
			eo := dsProto.GetExternal().GetEthOracle()
			assert.NotNil(t, eo)
			assert.IsType(t, &vegapb.EthCallSpec{}, eo)
			assert.Equal(t, "some-eth-address", eo.Address)
			assert.IsType(t, &structpb.ListValue{}, eo.Abi)
			assert.Equal(t, "method", eo.Method)
			filters := eo.GetFilters()
			assert.Equal(t, 1, len(filters))
			assert.IsType(t, &datapb.PropertyKey{}, filters[0].Key)
			assert.Equal(t, "test-key-name-0", filters[0].Key.Name)
			assert.Equal(t, datapb.PropertyKey_Type(5), filters[0].Key.Type)
			assert.IsType(t, &vegapb.EthCallTrigger{}, eo.Trigger)
			assert.Equal(t, uint64(256), eo.RequiredConfirmations)
		})
	})

	t.Run("internal dataSourceDefinition", func(t *testing.T) {
		t.Run("time termination", func(t *testing.T) {
			ds := types.NewDataSourceDefinitionWith(
				&types.DataSourceSpecConfigurationTime{
					Conditions: []*types.DataSourceSpecCondition{
						{
							Operator: types.DataSourceSpecConditionOperator(0),
							Value:    "12",
						},
					},
				},
			)

			dsProto := ds.IntoProto()
			assert.NotNil(t, dsProto.SourceType)
			assert.IsType(t, &vegapb.DataSourceDefinition_Internal{}, dsProto.SourceType)
			cond := dsProto.GetInternal().GetTime()
			assert.NotNil(t, cond)
			assert.IsType(t, &vegapb.DataSourceSpecConfigurationTime{}, cond)
			assert.Equal(t, 1, len(cond.Conditions))
			assert.IsType(t, &datapb.Condition{}, cond.Conditions[0])
			assert.Equal(t, datapb.Condition_OPERATOR_UNSPECIFIED, cond.Conditions[0].Operator)
			assert.Equal(t, "12", cond.Conditions[0].Value)
		})
	})
}

func TestContent(t *testing.T) {
	t.Run("testContent", func(t *testing.T) {
		t.Run("non-empty content with time termination source", func(t *testing.T) {
			d := types.NewDataSourceDefinitionWith(types.DataSourceSpecConfigurationTime{
				Conditions: []*types.DataSourceSpecCondition{
					{
						Operator: datapb.Condition_OPERATOR_EQUALS,
						Value:    "ext-test-value-0",
					},
					{
						Operator: datapb.Condition_OPERATOR_GREATER_THAN,
						Value:    "ext-test-value-1",
					},
				},
			})

			c := d.Content()
			assert.NotNil(t, c)
			tp, ok := c.(types.DataSourceSpecConfigurationTime)
			assert.True(t, ok)
			assert.Equal(t, 2, len(tp.Conditions))
			assert.Equal(t, "ext-test-value-0", tp.Conditions[0].Value)
			assert.Equal(t, "ext-test-value-1", tp.Conditions[1].Value)
		})

		t.Run("non-empty content with ethereum oracle source", func(t *testing.T) {
			timeNow := time.Now()
			d := types.NewDataSourceDefinitionWith(types.EthCallSpec{
				Address: "some-eth-address",
				AbiJson: []byte("abi-json-test"),
				Method:  "method",
				Trigger: &types.EthTimeTrigger{
					Initial: uint64(timeNow.UnixNano()),
				},
				RequiredConfirmations: 256,
				Filters: []*types.DataSourceSpecFilter{
					{
						Key: &types.DataSourceSpecPropertyKey{
							Name: "test-key-name-0",
							Type: types.DataSourceSpecPropertyKeyType(5),
						},
					},
				},
			})

			content := d.Content()
			assert.NotNil(t, content)
			assert.IsType(t, types.EthCallSpec{}, content)
			c, ok := content.(types.EthCallSpec)
			assert.True(t, ok)
			assert.Equal(t, "some-eth-address", c.Address)
			assert.Equal(t, []byte("abi-json-test"), c.AbiJson)
			assert.Equal(t, "method", c.Method)
			filters := c.Filters
			assert.Equal(t, 1, len(filters))
			assert.IsType(t, &types.DataSourceSpecPropertyKey{}, filters[0].Key)
			assert.Equal(t, "test-key-name-0", filters[0].Key.Name)
			assert.Equal(t, datapb.PropertyKey_Type(5), filters[0].Key.Type)
			assert.IsType(t, &types.EthTimeTrigger{}, c.Trigger)
			assert.Equal(t, uint64(256), c.RequiredConfirmations)
		})

		t.Run("non-empty content with oracle", func(t *testing.T) {
			d := types.NewDataSourceDefinitionWith(types.DataSourceSpecConfiguration{
				Signers: []*types.Signer{
					types.CreateSignerFromString("0xSOMEKEYX", types.DataSignerTypePubKey),
					types.CreateSignerFromString("0xSOMEKEYY", types.DataSignerTypePubKey),
				},
			})

			c := d.Content()
			assert.NotNil(t, c)
			tp, ok := c.(types.DataSourceSpecConfiguration)
			assert.True(t, ok)
			assert.Equal(t, 0, len(tp.Filters))
			assert.Equal(t, 2, len(tp.Signers))
			assert.Equal(t, "0xSOMEKEYX", tp.Signers[0].GetSignerPubKey().Key)
			assert.Equal(t, "0xSOMEKEYY", tp.Signers[1].GetSignerPubKey().Key)
		})
	})
}

func TestGetFilters(t *testing.T) {
	t.Run("testGetFiltersExternal", func(t *testing.T) {
		t.Run("NotEmpty Oracle", func(t *testing.T) {
			dsd := types.NewDataSourceDefinitionWith(types.DataSourceSpecConfiguration{
				Signers: []*types.Signer{
					types.CreateSignerFromString("0xSOMEKEYX", types.DataSignerTypePubKey),
					types.CreateSignerFromString("0xSOMEKEYY", types.DataSignerTypePubKey),
				},
				Filters: []*types.DataSourceSpecFilter{
					{
						Key: &types.DataSourceSpecPropertyKey{
							Name: "prices.ETH.value",
							Type: datapb.PropertyKey_TYPE_INTEGER,
						},
						Conditions: []*types.DataSourceSpecCondition{
							{
								Operator: datapb.Condition_OPERATOR_EQUALS,
								Value:    "ext-test-value-1",
							},
							{
								Operator: datapb.Condition_OPERATOR_GREATER_THAN,
								Value:    "ext-test-value-2",
							},
						},
					},
					{
						Key: &types.DataSourceSpecPropertyKey{
							Name: "key-name-string",
							Type: datapb.PropertyKey_TYPE_STRING,
						},
						Conditions: []*types.DataSourceSpecCondition{
							{
								Operator: datapb.Condition_OPERATOR_GREATER_THAN_OR_EQUAL,
								Value:    "ext-test-value-3",
							},
							{
								Operator: datapb.Condition_OPERATOR_GREATER_THAN,
								Value:    "ext-test-value-4",
							},
						},
					},
				},
			})

			filters := dsd.GetFilters()
			assert.Equal(t, 2, len(filters))
			assert.Equal(t, "prices.ETH.value", filters[0].Key.Name)
			assert.Equal(t, datapb.PropertyKey_TYPE_INTEGER, filters[0].Key.Type)
			assert.Equal(t, datapb.Condition_OPERATOR_EQUALS, filters[0].Conditions[0].Operator)
			assert.Equal(t, "ext-test-value-1", filters[0].Conditions[0].Value)
			assert.Equal(t, datapb.Condition_OPERATOR_GREATER_THAN, filters[0].Conditions[1].Operator)
			assert.Equal(t, "ext-test-value-2", filters[0].Conditions[1].Value)

			assert.Equal(t, "key-name-string", filters[1].Key.Name)
			assert.Equal(t, datapb.PropertyKey_TYPE_STRING, filters[1].Key.Type)
			assert.Equal(t, datapb.Condition_OPERATOR_GREATER_THAN_OR_EQUAL, filters[1].Conditions[0].Operator)
			assert.Equal(t, "ext-test-value-3", filters[1].Conditions[0].Value)

			assert.Equal(t, datapb.Condition_OPERATOR_GREATER_THAN, filters[1].Conditions[1].Operator)
			assert.Equal(t, "ext-test-value-4", filters[1].Conditions[1].Value)
		})

		t.Run("NotEmpty EthOracle", func(t *testing.T) {
			dsd := types.NewDataSourceDefinitionWith(types.EthCallSpec{
				Address: "some-eth-address",
				AbiJson: []byte("abi-json-test"),
				Method:  "method",
				Trigger: &types.EthTimeTrigger{
					Initial: uint64(time.Now().UnixNano()),
				},
				RequiredConfirmations: 256,
				Filters: []*types.DataSourceSpecFilter{
					{
						Key: &types.DataSourceSpecPropertyKey{
							Name: "test-key-name-0",
							Type: types.DataSourceSpecPropertyKeyType(3),
						},
					},
				},
			})

			filters := dsd.GetFilters()
			assert.Equal(t, 1, len(filters))
			assert.Equal(t, "test-key-name-0", filters[0].Key.Name)
			assert.Equal(t, datapb.PropertyKey_TYPE_STRING, filters[0].Key.Type)
			assert.Equal(t, 0, len(filters[0].Conditions))
		})
	})

	t.Run("testGetFiltersInternal", func(t *testing.T) {
		t.Run("NotEmpty", func(t *testing.T) {
			dsd := types.NewDataSourceDefinitionWith(
				types.DataSourceSpecConfigurationTime{
					Conditions: []*types.DataSourceSpecCondition{
						{
							Operator: datapb.Condition_OPERATOR_GREATER_THAN_OR_EQUAL,
							Value:    "int-test-value-1",
						},
						{
							Operator: datapb.Condition_OPERATOR_GREATER_THAN,
							Value:    "int-test-value-2",
						},
					},
				})

			filters := dsd.GetFilters()
			// Ensure only a single filter has been created, that holds all given conditions
			assert.Equal(t, 1, len(filters))

			assert.Equal(t, "vegaprotocol.builtin.timestamp", filters[0].Key.Name)
			assert.Equal(t, datapb.PropertyKey_TYPE_TIMESTAMP, filters[0].Key.Type)

			assert.Equal(t, datapb.Condition_OPERATOR_GREATER_THAN_OR_EQUAL, filters[0].Conditions[0].Operator)
			assert.Equal(t, "int-test-value-1", filters[0].Conditions[0].Value)
		})
	})
}

func TestUpdateFilters(t *testing.T) {
	t.Run("testUpdateFiltersExternal", func(t *testing.T) {
		t.Run("Empty", func(t *testing.T) {
			dsd := &vegapb.DataSourceDefinition{
				SourceType: &vegapb.DataSourceDefinition_External{},
			}

			dsdt, _ := types.DataSourceDefinitionFromProto(dsd)
			//if err != nil {
			//
			//}

			err := dsdt.(*types.DataSourceDefinition).UpdateFilters([]*types.DataSourceSpecFilter{})
			assert.NoError(t, err)
			filters := dsdt.(*types.DataSourceDefinition).GetFilters()
			assert.Equal(t, 0, len(filters))

			dsd = &vegapb.DataSourceDefinition{
				SourceType: &vegapb.DataSourceDefinition_External{
					External: &vegapb.DataSourceDefinitionExternal{
						SourceType: &vegapb.DataSourceDefinitionExternal_Oracle{
							Oracle: nil,
						},
					},
				},
			}

			dsdt, _ = types.DataSourceDefinitionFromProto(dsd)
			//if err != nil {
			//
			//}
			filters = dsdt.(*types.DataSourceDefinition).GetFilters()
			assert.Equal(t, 0, len(filters))
		})

		t.Run("NotEmpty Oracle", func(t *testing.T) {
			dsd := &vegapb.DataSourceDefinition{
				SourceType: &vegapb.DataSourceDefinition_External{
					External: &vegapb.DataSourceDefinitionExternal{
						SourceType: &vegapb.DataSourceDefinitionExternal_Oracle{
							Oracle: &vegapb.DataSourceSpecConfiguration{
								Signers: types.SignersIntoProto(
									[]*types.Signer{
										types.CreateSignerFromString("0xSOMEKEYX", types.DataSignerTypePubKey),
										types.CreateSignerFromString("0xSOMEKEYY", types.DataSignerTypePubKey),
									}),
							},
						},
					},
				},
			}

			dsdt, _ := types.DataSourceDefinitionFromProto(dsd)
			//if err != nil {
			//
			//}
			dst := types.NewDataSourceDefinitionWith(dsdt)
			err := dst.UpdateFilters(
				[]*types.DataSourceSpecFilter{
					{
						Key: &types.DataSourceSpecPropertyKey{
							Name: "prices.ETH.value",
							Type: datapb.PropertyKey_TYPE_INTEGER,
						},
						Conditions: []*types.DataSourceSpecCondition{
							{
								Operator: datapb.Condition_OPERATOR_EQUALS,
								Value:    "ext-test-value-1",
							},
							{
								Operator: datapb.Condition_OPERATOR_GREATER_THAN,
								Value:    "ext-test-value-2",
							},
						},
					},
					{
						Key: &types.DataSourceSpecPropertyKey{
							Name: "key-name-string",
							Type: datapb.PropertyKey_TYPE_STRING,
						},
						Conditions: []*types.DataSourceSpecCondition{
							{
								Operator: datapb.Condition_OPERATOR_GREATER_THAN_OR_EQUAL,
								Value:    "ext-test-value-3",
							},
							{
								Operator: datapb.Condition_OPERATOR_GREATER_THAN,
								Value:    "ext-test-value-4",
							},
						},
					},
				},
			)

			assert.NoError(t, err)

			filters := dst.GetFilters()
			require.Equal(t, 2, len(filters))
			assert.Equal(t, "prices.ETH.value", filters[0].Key.Name)
			assert.Equal(t, datapb.PropertyKey_TYPE_INTEGER, filters[0].Key.Type)
			assert.Equal(t, datapb.Condition_OPERATOR_EQUALS, filters[0].Conditions[0].Operator)
			assert.Equal(t, "ext-test-value-1", filters[0].Conditions[0].Value)
			assert.Equal(t, datapb.Condition_OPERATOR_GREATER_THAN, filters[0].Conditions[1].Operator)
			assert.Equal(t, "ext-test-value-2", filters[0].Conditions[1].Value)

			assert.Equal(t, "key-name-string", filters[1].Key.Name)
			assert.Equal(t, datapb.PropertyKey_TYPE_STRING, filters[1].Key.Type)
			assert.Equal(t, datapb.Condition_OPERATOR_GREATER_THAN_OR_EQUAL, filters[1].Conditions[0].Operator)
			assert.Equal(t, "ext-test-value-3", filters[1].Conditions[0].Value)

			assert.Equal(t, datapb.Condition_OPERATOR_GREATER_THAN, filters[1].Conditions[1].Operator)
			assert.Equal(t, "ext-test-value-4", filters[1].Conditions[1].Value)
		})

		t.Run("NotEmpty EthOracle", func(t *testing.T) {
			dsd := &vegapb.DataSourceDefinition{
				SourceType: &vegapb.DataSourceDefinition_External{
					External: &vegapb.DataSourceDefinitionExternal{
						SourceType: &vegapb.DataSourceDefinitionExternal_EthOracle{
							EthOracle: &vegapb.EthCallSpec{
								Address: "some-eth-address",
								Filters: []*datapb.Filter{
									{
										Key: &datapb.PropertyKey{
											Name: "test-key-name-0",
											Type: types.DataSourceSpecPropertyKeyType(5),
										},
									},
								},
							},
						},
					},
				},
			}

			dsdt, _ := types.DataSourceDefinitionFromProto(dsd)
			//if err != nil {
			//
			//}
			dst := types.NewDataSourceDefinitionWith(dsdt)
			err := dst.UpdateFilters(
				[]*types.DataSourceSpecFilter{
					{
						Key: &types.DataSourceSpecPropertyKey{
							Name: "eth-spec-new-property-key",
							Type: datapb.PropertyKey_TYPE_INTEGER,
						},
						Conditions: []*types.DataSourceSpecCondition{
							{
								Operator: datapb.Condition_OPERATOR_EQUALS,
								Value:    "ext-test-value-1",
							},
							{
								Operator: datapb.Condition_OPERATOR_GREATER_THAN,
								Value:    "ext-test-value-2",
							},
						},
					},
					{
						Key: &types.DataSourceSpecPropertyKey{
							Name: "key-name-string",
							Type: datapb.PropertyKey_TYPE_STRING,
						},
						Conditions: []*types.DataSourceSpecCondition{
							{
								Operator: datapb.Condition_OPERATOR_GREATER_THAN_OR_EQUAL,
								Value:    "ext-test-value-3",
							},
							{
								Operator: datapb.Condition_OPERATOR_GREATER_THAN,
								Value:    "ext-test-value-4",
							},
						},
					},
				},
			)
			assert.NoError(t, err)

			filters := dst.GetFilters()
			require.Equal(t, 2, len(filters))
			assert.Equal(t, "eth-spec-new-property-key", filters[0].Key.Name)
			assert.Equal(t, datapb.PropertyKey_TYPE_INTEGER, filters[0].Key.Type)
			assert.Equal(t, datapb.Condition_OPERATOR_EQUALS, filters[0].Conditions[0].Operator)
			assert.Equal(t, "ext-test-value-1", filters[0].Conditions[0].Value)
			assert.Equal(t, datapb.Condition_OPERATOR_GREATER_THAN, filters[0].Conditions[1].Operator)
			assert.Equal(t, "ext-test-value-2", filters[0].Conditions[1].Value)

			assert.Equal(t, "key-name-string", filters[1].Key.Name)
			assert.Equal(t, datapb.PropertyKey_TYPE_STRING, filters[1].Key.Type)
			assert.Equal(t, datapb.Condition_OPERATOR_GREATER_THAN_OR_EQUAL, filters[1].Conditions[0].Operator)
			assert.Equal(t, "ext-test-value-3", filters[1].Conditions[0].Value)

			assert.Equal(t, datapb.Condition_OPERATOR_GREATER_THAN, filters[1].Conditions[1].Operator)
			assert.Equal(t, "ext-test-value-4", filters[1].Conditions[1].Value)
		})
	})

	t.Run("testUpdateFiltersInternal", func(t *testing.T) {
		t.Run("NotEmpty", func(t *testing.T) {
			dsd := &vegapb.DataSourceDefinition{
				SourceType: &vegapb.DataSourceDefinition_Internal{
					Internal: &vegapb.DataSourceDefinitionInternal{
						SourceType: &vegapb.DataSourceDefinitionInternal_Time{
							Time: &vegapb.DataSourceSpecConfigurationTime{},
						},
					},
				},
			}

			dsdt, _ := types.DataSourceDefinitionFromProto(dsd)
			//if err != nil {
			//
			//}
			dst := types.NewDataSourceDefinitionWith(dsdt)
			err := dst.UpdateFilters(
				[]*types.DataSourceSpecFilter{
					{
						Conditions: []*types.DataSourceSpecCondition{
							{
								Operator: datapb.Condition_OPERATOR_GREATER_THAN_OR_EQUAL,
								Value:    "int-test-value-1",
							},
							{
								Operator: datapb.Condition_OPERATOR_GREATER_THAN,
								Value:    "int-test-value-2",
							},
						},
					},
				},
			)
			assert.NoError(t, err)
			filters := dst.GetFilters()
			// Ensure only a single filter has been created, that holds all given conditions
			assert.Equal(t, 1, len(filters))

			assert.Equal(t, "vegaprotocol.builtin.timestamp", filters[0].Key.Name)
			assert.Equal(t, datapb.PropertyKey_TYPE_TIMESTAMP, filters[0].Key.Type)

			assert.Equal(t, datapb.Condition_OPERATOR_GREATER_THAN_OR_EQUAL, filters[0].Conditions[0].Operator)
			assert.Equal(t, "int-test-value-1", filters[0].Conditions[0].Value)
		})

		t.Run("Empty", func(t *testing.T) {
			dsd := &vegapb.DataSourceDefinition{
				SourceType: &vegapb.DataSourceDefinition_Internal{},
			}

			dsdt, _ := types.DataSourceDefinitionFromProto(dsd)
			//if err != nil {
			//
			//}
			dst := types.NewDataSourceDefinitionWith(dsdt)
			err := dst.UpdateFilters(
				[]*types.DataSourceSpecFilter{},
			)

			assert.NoError(t, err)
			filters := dsdt.(*types.DataSourceDefinition).GetFilters()
			assert.Equal(t, 0, len(filters))
		})
	})
}

func TestGetSigners(t *testing.T) {
	t.Run("empty signers", func(t *testing.T) {
		ds := types.NewDataSourceDefinitionWith(types.DataSourceSpecConfiguration{})

		signers := ds.GetSigners()
		assert.Equal(t, 0, len(signers))
	})

	t.Run("non-empty list but empty signers", func(t *testing.T) {
		ds := types.NewDataSourceDefinitionWith(types.DataSourceSpecConfiguration{
			Signers: []*types.Signer{
				{},
				{},
			},
		})

		signers := ds.GetSigners()
		assert.Equal(t, 2, len(signers))
	})

	t.Run("non-empty signers", func(t *testing.T) {
		ds := types.NewDataSourceDefinitionWith(types.DataSourceSpecConfiguration{
			Signers: []*types.Signer{
				{
					types.CreateSignerFromString("0xTESTSIGN", types.DataSignerTypePubKey).Signer,
				},
			},
		})

		signers := ds.GetSigners()
		assert.Equal(t, 1, len(signers))
		assert.IsType(t, &types.Signer{}, signers[0])
		assert.IsType(t, &types.SignerPubKey{}, signers[0].Signer)
		assert.Equal(t, "0xTESTSIGN", signers[0].GetSignerPubKey().Key)
	})
}

func TestGetDataSourceSpecConfiguration(t *testing.T) {
	ds := types.NewDataSourceDefinitionWith(
		types.DataSourceSpecConfiguration{
			Signers: []*types.Signer{
				{
					types.CreateSignerFromString("0xTESTSIGN", types.DataSignerTypePubKey).Signer,
				},
			},
		})

	spec := ds.GetDataSourceSpecConfiguration()
	assert.NotNil(t, spec)
	assert.IsType(t, types.DataSourceSpecConfiguration{}, spec)
	assert.Equal(t, 1, len(spec.Signers))
	assert.Equal(t, "0xTESTSIGN", spec.Signers[0].GetSignerPubKey().Key)
}

func TestGetEthCallSpec(t *testing.T) {
	ds := types.NewDataSourceDefinitionWith(types.EthCallSpec{
		Address: "some-eth-address",
		AbiJson: []byte("abi-json-test"),
		Method:  "method",
		Trigger: &types.EthTimeTrigger{
			Initial: uint64(time.Now().UnixNano()),
		},
		RequiredConfirmations: 256,
		Filters: []*types.DataSourceSpecFilter{
			{
				Key: &types.DataSourceSpecPropertyKey{
					Name: "test-key-name-0",
					Type: types.DataSourceSpecPropertyKeyType(3),
				},
			},
		},
	})

	dsSpec := ds.GetEthCallSpec()
	assert.NotNil(t, dsSpec)
	assert.IsType(t, types.EthCallSpec{}, dsSpec)
	assert.Equal(t, "some-eth-address", dsSpec.Address)
	assert.Equal(t, []byte("abi-json-test"), dsSpec.AbiJson)
	assert.Equal(t, "method", dsSpec.Method)
	assert.Equal(t, uint64(256), dsSpec.RequiredConfirmations)
	filters := dsSpec.Filters
	assert.Equal(t, 1, len(filters))
	assert.Equal(t, "test-key-name-0", filters[0].Key.Name)
	assert.Equal(t, datapb.PropertyKey_TYPE_STRING, filters[0].Key.Type)
	assert.Equal(t, 0, len(filters[0].Conditions))
}

func TestGetDataSourceSpecConfigurationTime(t *testing.T) {
	ds := types.NewDataSourceDefinitionWith(types.DataSourceSpecConfigurationTime{
		Conditions: []*types.DataSourceSpecCondition{
			{
				Operator: datapb.Condition_OPERATOR_GREATER_THAN,
				Value:    "1",
			},
		},
	})

	spec := ds.GetDataSourceSpecConfigurationTime()
	assert.NotNil(t, spec)
	assert.IsType(t, types.DataSourceSpecConfigurationTime{}, spec)
	assert.NotNil(t, spec.Conditions)
	assert.Equal(t, datapb.Condition_OPERATOR_GREATER_THAN, spec.Conditions[0].Operator)
	assert.Equal(t, "1", spec.Conditions[0].Value)
}

func TestIsExternal(t *testing.T) {
	dsDef := types.NewDataSourceDefinitionWith(types.DataSourceSpecConfiguration{})

	res, err := dsDef.IsExternal()
	assert.NoError(t, err)
	assert.True(t, res)

	dsDef = types.NewDataSourceDefinitionWith(types.EthCallSpec{})

	res, err = dsDef.IsExternal()
	assert.NoError(t, err)
	assert.True(t, res)

	dsDef = types.NewDataSourceDefinitionWith(types.DataSourceSpecConfigurationTime{
		Conditions: []*types.DataSourceSpecCondition{},
	})

	res, err = dsDef.IsExternal()
	assert.NoError(t, err)
	assert.False(t, res)

	dsDef = types.NewDataSourceDefinitionWith(nil)
	res, _ = dsDef.IsExternal()

	assert.Error(t, errors.New("unknown type of data source provided"))
	assert.False(t, res)
}

func TestSetOracleConfig(t *testing.T) {
	t.Run("non-empty oracle", func(t *testing.T) {
		dsd := types.NewDataSourceDefinitionWith(types.DataSourceSpecConfiguration{})

		udsd := dsd.SetOracleConfig(
			&types.DataSourceSpecConfiguration{
				Signers: []*types.Signer{
					types.CreateSignerFromString("0xSOMEKEYX", types.DataSignerTypePubKey),
					types.CreateSignerFromString("0xSOMEKEYY", types.DataSignerTypePubKey),
				},
				Filters: []*types.DataSourceSpecFilter{
					{
						Key: &types.DataSourceSpecPropertyKey{
							Name: "prices.ETH.value",
							Type: datapb.PropertyKey_TYPE_INTEGER,
						},
						Conditions: []*types.DataSourceSpecCondition{
							{
								Operator: datapb.Condition_OPERATOR_EQUALS,
								Value:    "ext-test-value-1",
							},
							{
								Operator: datapb.Condition_OPERATOR_GREATER_THAN,
								Value:    "ext-test-value-2",
							},
						},
					},
					{
						Key: &types.DataSourceSpecPropertyKey{
							Name: "key-name-string",
							Type: datapb.PropertyKey_TYPE_STRING,
						},
						Conditions: []*types.DataSourceSpecCondition{
							{
								Operator: datapb.Condition_OPERATOR_GREATER_THAN_OR_EQUAL,
								Value:    "ext-test-value-3",
							},
							{
								Operator: datapb.Condition_OPERATOR_GREATER_THAN,
								Value:    "ext-test-value-4",
							},
						},
					},
				},
			},
		)

		signers := udsd.GetSigners()
		assert.Equal(t, 2, len(signers))
		assert.Equal(t, "0xSOMEKEYX", signers[0].GetSignerPubKey().Key)
		assert.Equal(t, "0xSOMEKEYY", signers[1].GetSignerPubKey().Key)
		filters := udsd.GetFilters()
		assert.Equal(t, 2, len(filters))
		assert.Equal(t, "prices.ETH.value", filters[0].Key.Name)
		assert.Equal(t, datapb.PropertyKey_TYPE_INTEGER, filters[0].Key.Type)
		assert.Equal(t, datapb.Condition_OPERATOR_EQUALS, filters[0].Conditions[0].Operator)
		assert.Equal(t, "ext-test-value-1", filters[0].Conditions[0].Value)
		assert.Equal(t, datapb.Condition_OPERATOR_GREATER_THAN, filters[0].Conditions[1].Operator)
		assert.Equal(t, "ext-test-value-2", filters[0].Conditions[1].Value)

		assert.Equal(t, "key-name-string", filters[1].Key.Name)
		assert.Equal(t, datapb.PropertyKey_TYPE_STRING, filters[1].Key.Type)
		assert.Equal(t, datapb.Condition_OPERATOR_GREATER_THAN_OR_EQUAL, filters[1].Conditions[0].Operator)
		assert.Equal(t, "ext-test-value-3", filters[1].Conditions[0].Value)

		assert.Equal(t, datapb.Condition_OPERATOR_GREATER_THAN, filters[1].Conditions[1].Operator)
		assert.Equal(t, "ext-test-value-4", filters[1].Conditions[1].Value)
	})

	t.Run("non-empty eth oracle", func(t *testing.T) {
		dsd := types.NewDataSourceDefinitionWith(types.EthCallSpec{})

		udsd := dsd.SetOracleConfig(
			&types.EthCallSpec{
				Address: "some-eth-address",
				AbiJson: []byte("abi-json-test"),
				Method:  "method",
				Trigger: &types.EthTimeTrigger{
					Initial: uint64(time.Now().UnixNano()),
				},
				RequiredConfirmations: 256,
				Filters: []*types.DataSourceSpecFilter{
					{
						Key: &types.DataSourceSpecPropertyKey{
							Name: "prices.ETH.value",
							Type: datapb.PropertyKey_TYPE_INTEGER,
						},
						Conditions: []*types.DataSourceSpecCondition{
							{
								Operator: datapb.Condition_OPERATOR_EQUALS,
								Value:    "ext-test-value-1",
							},
							{
								Operator: datapb.Condition_OPERATOR_GREATER_THAN,
								Value:    "ext-test-value-2",
							},
						},
					},
					{
						Key: &types.DataSourceSpecPropertyKey{
							Name: "key-name-string",
							Type: datapb.PropertyKey_TYPE_STRING,
						},
						Conditions: []*types.DataSourceSpecCondition{
							{
								Operator: datapb.Condition_OPERATOR_GREATER_THAN_OR_EQUAL,
								Value:    "ext-test-value-3",
							},
							{
								Operator: datapb.Condition_OPERATOR_GREATER_THAN,
								Value:    "ext-test-value-4",
							},
						},
					},
				},
			},
		)

		dsSpec := udsd.GetEthCallSpec()
		assert.NotNil(t, dsSpec)
		assert.IsType(t, types.EthCallSpec{}, dsSpec)
		assert.Equal(t, "some-eth-address", dsSpec.Address)
		assert.Equal(t, []byte("abi-json-test"), dsSpec.AbiJson)
		assert.Equal(t, "method", dsSpec.Method)
		assert.Equal(t, uint64(256), dsSpec.RequiredConfirmations)
		filters := udsd.GetFilters()
		assert.Equal(t, 2, len(filters))
		assert.Equal(t, "prices.ETH.value", filters[0].Key.Name)
		assert.Equal(t, datapb.PropertyKey_TYPE_INTEGER, filters[0].Key.Type)
		assert.Equal(t, datapb.Condition_OPERATOR_EQUALS, filters[0].Conditions[0].Operator)
		assert.Equal(t, "ext-test-value-1", filters[0].Conditions[0].Value)
		assert.Equal(t, datapb.Condition_OPERATOR_GREATER_THAN, filters[0].Conditions[1].Operator)
		assert.Equal(t, "ext-test-value-2", filters[0].Conditions[1].Value)

		assert.Equal(t, "key-name-string", filters[1].Key.Name)
		assert.Equal(t, datapb.PropertyKey_TYPE_STRING, filters[1].Key.Type)
		assert.Equal(t, datapb.Condition_OPERATOR_GREATER_THAN_OR_EQUAL, filters[1].Conditions[0].Operator)
		assert.Equal(t, "ext-test-value-3", filters[1].Conditions[0].Value)

		assert.Equal(t, datapb.Condition_OPERATOR_GREATER_THAN, filters[1].Conditions[1].Operator)
		assert.Equal(t, "ext-test-value-4", filters[1].Conditions[1].Value)
	})

	t.Run("try to set oracle config to internal data source", func(t *testing.T) {
		dsd := types.NewDataSourceDefinitionWith(
			types.DataSourceSpecConfigurationTime{
				Conditions: []*types.DataSourceSpecCondition{
					{
						Operator: datapb.Condition_OPERATOR_GREATER_THAN_OR_EQUAL,
						Value:    "int-test-value-1",
					},
					{
						Operator: datapb.Condition_OPERATOR_GREATER_THAN,
						Value:    "int-test-value-2",
					},
				},
			})

		iudsd := dsd.SetOracleConfig(
			&types.DataSourceSpecConfiguration{
				Filters: []*types.DataSourceSpecFilter{
					{
						Key: &types.DataSourceSpecPropertyKey{
							Name: "prices.ETH.value",
							Type: datapb.PropertyKey_TYPE_INTEGER,
						},
						Conditions: []*types.DataSourceSpecCondition{
							{
								Operator: datapb.Condition_OPERATOR_EQUALS,
								Value:    "ext-test-value-1",
							},
							{
								Operator: datapb.Condition_OPERATOR_GREATER_THAN,
								Value:    "ext-test-value-2",
							},
						},
					},
					{
						Key: &types.DataSourceSpecPropertyKey{
							Name: "key-name-string",
							Type: datapb.PropertyKey_TYPE_STRING,
						},
						Conditions: []*types.DataSourceSpecCondition{
							{
								Operator: datapb.Condition_OPERATOR_GREATER_THAN_OR_EQUAL,
								Value:    "ext-test-value-3",
							},
							{
								Operator: datapb.Condition_OPERATOR_GREATER_THAN,
								Value:    "ext-test-value-4",
							},
						},
					},
				},
			},
		)

		filters := iudsd.GetFilters()
		assert.Equal(t, 1, len(filters))

		assert.Equal(t, "vegaprotocol.builtin.timestamp", filters[0].Key.Name)
		assert.Equal(t, datapb.PropertyKey_TYPE_TIMESTAMP, filters[0].Key.Type)

		assert.Equal(t, datapb.Condition_OPERATOR_GREATER_THAN_OR_EQUAL, filters[0].Conditions[0].Operator)
		assert.Equal(t, "int-test-value-1", filters[0].Conditions[0].Value)
	})
}

func TestSetTimeTriggerConditionConfig(t *testing.T) {
	dsd := types.NewDataSourceDefinitionWith(types.DataSourceSpecConfiguration{})

	udsd := dsd.SetTimeTriggerConditionConfig(
		[]*types.DataSourceSpecCondition{
			{
				Operator: datapb.Condition_OPERATOR_GREATER_THAN_OR_EQUAL,
				Value:    "ext-test-value-3",
			},
			{
				Operator: datapb.Condition_OPERATOR_GREATER_THAN,
				Value:    "ext-test-value-4",
			},
		},
	)

	filters := udsd.GetFilters()
	assert.Equal(t, 0, len(filters))

	t.Run("try to set time trigger config to internal data source", func(t *testing.T) {
		dsd := types.NewDataSourceDefinitionWith(
			types.DataSourceSpecConfigurationTime{
				Conditions: []*types.DataSourceSpecCondition{
					{
						Operator: datapb.Condition_OPERATOR_GREATER_THAN_OR_EQUAL,
						Value:    "int-test-value-1",
					},
					{
						Operator: datapb.Condition_OPERATOR_GREATER_THAN,
						Value:    "int-test-value-2",
					},
				},
			})

		iudsd := dsd.SetTimeTriggerConditionConfig(
			[]*types.DataSourceSpecCondition{
				{
					Operator: datapb.Condition_OPERATOR_GREATER_THAN_OR_EQUAL,
					Value:    "int-test-value-3",
				},
				{
					Operator: datapb.Condition_OPERATOR_GREATER_THAN,
					Value:    "int-test-value-4",
				},
			},
		)

		filters := iudsd.GetFilters()
		assert.Equal(t, 1, len(filters))

		assert.Equal(t, "vegaprotocol.builtin.timestamp", filters[0].Key.Name)
		assert.Equal(t, datapb.PropertyKey_TYPE_TIMESTAMP, filters[0].Key.Type)

		assert.Equal(t, datapb.Condition_OPERATOR_GREATER_THAN_OR_EQUAL, filters[0].Conditions[0].Operator)
		assert.Equal(t, "int-test-value-3", filters[0].Conditions[0].Value)
	})
}

func TestType(t *testing.T) {
	ds := types.NewDataSourceDefinitionWith(&types.DataSourceSpecConfiguration{})
	tp, ext := ds.Type()
	assert.Equal(t, types.DataSourceContentTypeOracle, tp)
	assert.True(t, ext)

	ds = types.NewDataSourceDefinitionWith(&types.EthCallSpec{})
	tp, ext = ds.Type()
	assert.Equal(t, types.DataSourceContentTypeEthOracle, tp)
	assert.True(t, ext)

	ds = types.NewDataSourceDefinitionWith(&types.DataSourceSpecConfigurationTime{})
	tp, ext = ds.Type()
	assert.Equal(t, types.DataSourceContentTypeInternalTimeTermination, tp)
	assert.False(t, ext)

	ds = types.NewDataSourceDefinitionWith(&types.DataSourceDefinition{})
	tp, ext = ds.Type()
	assert.Equal(t, types.DataSourceContentTypeInvalid, tp)
	assert.False(t, ext)
}
