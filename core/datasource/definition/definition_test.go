package definition_test

import (
	"errors"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/datasource/common"
	"code.vegaprotocol.io/vega/core/datasource/definition"
	ethcallcommon "code.vegaprotocol.io/vega/core/datasource/external/ethcall/common"
	"code.vegaprotocol.io/vega/core/datasource/external/signedoracle"
	"code.vegaprotocol.io/vega/core/datasource/internal/timetrigger"
	"code.vegaprotocol.io/vega/core/datasource/internal/vegatime"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
	datapb "code.vegaprotocol.io/vega/protos/vega/data/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDefinitionWith(t *testing.T) {
	t.Run("oracle", func(t *testing.T) {
		t.Run("empty", func(t *testing.T) {
			nds := definition.NewWith(nil)
			assert.NotNil(t, nds)
			assert.IsType(t, &definition.Definition{}, nds)
			assert.Nil(t, nds.Content())
		})
		// Non-empty cases are tested implicitly by TestDefinitionIntoProto
	})
}

func TestNewDefinition(t *testing.T) {
	ds := definition.New(definition.ContentTypeOracle)
	assert.NotNil(t, ds)
	cnt := ds.Content()
	assert.IsType(t, signedoracle.SpecConfiguration{}, cnt)

	ds = definition.New(definition.ContentTypeEthOracle)
	assert.NotNil(t, ds)
	cnt = ds.Content()
	assert.IsType(t, ethcallcommon.Spec{}, cnt)

	ds = definition.New(definition.ContentTypeInternalTimeTermination)
	assert.NotNil(t, ds)
	cnt = ds.Content()
	assert.IsType(t, vegatime.SpecConfiguration{}, cnt)
}

func TestDefinitionIntoProto(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		ds := &definition.Definition{}
		protoDs := ds.IntoProto()
		assert.IsType(t, &vegapb.DataSourceDefinition{}, protoDs)
		assert.Nil(t, protoDs.SourceType)
	})

	t.Run("external dataSourceDefinition", func(t *testing.T) {
		t.Run("non-empty but no content", func(t *testing.T) {
			ds := &definition.Definition{}
			protoDs := ds.IntoProto()
			assert.IsType(t, &vegapb.DataSourceDefinition{}, protoDs)
			assert.Nil(t, protoDs.SourceType)
		})

		t.Run("oracle", func(t *testing.T) {
			ds := definition.NewWith(
				&signedoracle.SpecConfiguration{
					Signers: []*common.Signer{
						{
							Signer: &common.SignerETHAddress{
								ETHAddress: &common.ETHAddress{
									Address: "test-eth-address-0",
								},
							},
						},
						{
							Signer: &common.SignerETHAddress{
								ETHAddress: &common.ETHAddress{
									Address: "test-eth-address-1",
								},
							},
						},
					},
					Filters: []*common.SpecFilter{
						{
							Key: &common.SpecPropertyKey{
								Name: "test-property-key-name-0",
								Type: common.SpecPropertyKeyType(0),
							},
							Conditions: []*common.SpecCondition{
								{
									Operator: common.SpecConditionOperator(0),
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
			ds := definition.NewWith(
				&ethcallcommon.Spec{
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
					Trigger: &ethcallcommon.TimeTrigger{
						Initial: uint64(timeNow.UnixNano()),
					},
					RequiredConfirmations: 256,
					Filters: []*common.SpecFilter{
						{
							Key: &common.SpecPropertyKey{
								Name: "test-key-name-0",
								Type: common.SpecPropertyKeyType(5),
							},
							Conditions: []*common.SpecCondition{
								{
									Operator: common.SpecConditionOperator(0),
									Value:    "12",
								},
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
			assert.IsType(t, "", eo.Abi)
			assert.Equal(t, "method", eo.Method)
			filters := eo.GetFilters()
			assert.Equal(t, 1, len(filters))
			assert.IsType(t, &datapb.PropertyKey{}, filters[0].Key)
			assert.Equal(t, "test-key-name-0", filters[0].Key.Name)
			assert.Equal(t, datapb.PropertyKey_Type(5), filters[0].Key.Type)
			assert.Equal(t, 1, len(filters[0].Conditions))
			assert.IsType(t, &datapb.Condition{}, filters[0].Conditions[0])
			assert.Equal(t, datapb.Condition_OPERATOR_UNSPECIFIED, filters[0].Conditions[0].Operator)
			assert.Equal(t, "12", filters[0].Conditions[0].Value)
			assert.IsType(t, &vegapb.EthCallTrigger{}, eo.Trigger)
			assert.Equal(t, uint64(256), eo.RequiredConfirmations)
		})
	})

	t.Run("internal datasource Definition", func(t *testing.T) {
		t.Run("time termination", func(t *testing.T) {
			ds := definition.NewWith(
				&vegatime.SpecConfiguration{
					Conditions: []*common.SpecCondition{
						{
							Operator: common.SpecConditionOperator(0),
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

		t.Run("time trigger termination", func(t *testing.T) {
			timeNow := time.Now()
			ds := definition.NewWith(
				&timetrigger.SpecConfiguration{
					Conditions: []*common.SpecCondition{
						{
							Operator: common.SpecConditionOperator(0),
							Value:    "12",
						},
						{
							Operator: common.SpecConditionOperator(0),
							Value:    "17",
						},
					},
					Triggers: common.InternalTimeTriggers{
						{
							Initial: &timeNow,
							Every:   int64(15),
						},
					},
				},
			)

			dsProto := ds.IntoProto()
			assert.NotNil(t, dsProto.SourceType)
			assert.IsType(t, &vegapb.DataSourceDefinition_Internal{}, dsProto.SourceType)
			cond := dsProto.GetInternal().GetTimeTrigger()
			assert.NotNil(t, cond)
			assert.IsType(t, &vegapb.DataSourceSpecConfigurationTimeTrigger{}, cond)
			assert.Equal(t, 2, len(cond.Conditions))
			assert.IsType(t, &datapb.Condition{}, cond.Conditions[0])
			assert.Equal(t, datapb.Condition_OPERATOR_UNSPECIFIED, cond.Conditions[0].Operator)
			assert.Equal(t, "12", cond.Conditions[0].Value)
			assert.IsType(t, &datapb.Condition{}, cond.Conditions[1])
			assert.Equal(t, datapb.Condition_OPERATOR_UNSPECIFIED, cond.Conditions[1].Operator)
			assert.Equal(t, "17", cond.Conditions[1].Value)
			assert.Equal(t, timeNow.Unix(), *cond.Triggers[0].Initial)
			assert.Equal(t, int64(15), cond.Triggers[0].Every)
		})
	})
}

func TestContent(t *testing.T) {
	t.Run("testContent", func(t *testing.T) {
		t.Run("non-empty content with time termination source", func(t *testing.T) {
			d := definition.NewWith(vegatime.SpecConfiguration{
				Conditions: []*common.SpecCondition{
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
			tp, ok := c.(vegatime.SpecConfiguration)
			assert.True(t, ok)
			assert.Equal(t, 2, len(tp.Conditions))
			assert.Equal(t, "ext-test-value-0", tp.Conditions[0].Value)
			assert.Equal(t, "ext-test-value-1", tp.Conditions[1].Value)
		})

		t.Run("non-empty content with ethereum oracle source", func(t *testing.T) {
			timeNow := time.Now()
			d := definition.NewWith(ethcallcommon.Spec{
				Address: "some-eth-address",
				AbiJson: []byte("abi-json-test"),
				Method:  "method",
				Trigger: &ethcallcommon.TimeTrigger{
					Initial: uint64(timeNow.UnixNano()),
				},
				RequiredConfirmations: 256,
				Filters: []*common.SpecFilter{
					{
						Key: &common.SpecPropertyKey{
							Name: "test-key-name-0",
							Type: common.SpecPropertyKeyType(5),
						},
					},
				},
			})

			content := d.Content()
			assert.NotNil(t, content)
			assert.IsType(t, ethcallcommon.Spec{}, content)
			c, ok := content.(ethcallcommon.Spec)
			assert.True(t, ok)
			assert.Equal(t, "some-eth-address", c.Address)
			assert.Equal(t, []byte("abi-json-test"), c.AbiJson)
			assert.Equal(t, "method", c.Method)
			filters := c.Filters
			assert.Equal(t, 1, len(filters))
			assert.IsType(t, &common.SpecPropertyKey{}, filters[0].Key)
			assert.Equal(t, "test-key-name-0", filters[0].Key.Name)
			assert.Equal(t, datapb.PropertyKey_Type(5), filters[0].Key.Type)
			assert.IsType(t, &ethcallcommon.TimeTrigger{}, c.Trigger)
			assert.Equal(t, uint64(256), c.RequiredConfirmations)
		})

		t.Run("non-empty content with oracle", func(t *testing.T) {
			d := definition.NewWith(signedoracle.SpecConfiguration{
				Signers: []*common.Signer{
					common.CreateSignerFromString("0xSOMEKEYX", common.SignerTypePubKey),
					common.CreateSignerFromString("0xSOMEKEYY", common.SignerTypePubKey),
				},
			})

			c := d.Content()
			assert.NotNil(t, c)
			tp, ok := c.(signedoracle.SpecConfiguration)
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
			dsd := definition.NewWith(signedoracle.SpecConfiguration{
				Signers: []*common.Signer{
					common.CreateSignerFromString("0xSOMEKEYX", common.SignerTypePubKey),
					common.CreateSignerFromString("0xSOMEKEYY", common.SignerTypePubKey),
				},
				Filters: []*common.SpecFilter{
					{
						Key: &common.SpecPropertyKey{
							Name: "prices.ETH.value",
							Type: datapb.PropertyKey_TYPE_INTEGER,
						},
						Conditions: []*common.SpecCondition{
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
						Key: &common.SpecPropertyKey{
							Name: "key-name-string",
							Type: datapb.PropertyKey_TYPE_STRING,
						},
						Conditions: []*common.SpecCondition{
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
			dsd := definition.NewWith(ethcallcommon.Spec{
				Address: "some-eth-address",
				AbiJson: []byte("abi-json-test"),
				Method:  "method",
				Trigger: &ethcallcommon.TimeTrigger{
					Initial: uint64(time.Now().UnixNano()),
				},
				RequiredConfirmations: 256,
				Filters: []*common.SpecFilter{
					{
						Key: &common.SpecPropertyKey{
							Name: "test-key-name-0",
							Type: common.SpecPropertyKeyType(3),
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
			dsd := definition.NewWith(
				vegatime.SpecConfiguration{
					Conditions: []*common.SpecCondition{
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

			dsdt, err := definition.FromProto(dsd, nil)
			assert.NoError(t, err)

			err = dsdt.(*definition.Definition).UpdateFilters([]*common.SpecFilter{})
			assert.NoError(t, err)
			filters := dsdt.(*definition.Definition).GetFilters()
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

			dsdt, err = definition.FromProto(dsd, nil)
			assert.NoError(t, err)
			filters = dsdt.(*definition.Definition).GetFilters()
			assert.Equal(t, 0, len(filters))
		})

		t.Run("NotEmpty Oracle", func(t *testing.T) {
			dsd := &vegapb.DataSourceDefinition{
				SourceType: &vegapb.DataSourceDefinition_External{
					External: &vegapb.DataSourceDefinitionExternal{
						SourceType: &vegapb.DataSourceDefinitionExternal_Oracle{
							Oracle: &vegapb.DataSourceSpecConfiguration{
								Signers: common.SignersIntoProto(
									[]*common.Signer{
										common.CreateSignerFromString("0xSOMEKEYX", common.SignerTypePubKey),
										common.CreateSignerFromString("0xSOMEKEYY", common.SignerTypePubKey),
									}),
							},
						},
					},
				},
			}

			dsdt, err := definition.FromProto(dsd, nil)
			assert.NoError(t, err)
			dst := definition.NewWith(dsdt)
			err = dst.UpdateFilters(
				[]*common.SpecFilter{
					{
						Key: &common.SpecPropertyKey{
							Name: "prices.ETH.value",
							Type: datapb.PropertyKey_TYPE_INTEGER,
						},
						Conditions: []*common.SpecCondition{
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
						Key: &common.SpecPropertyKey{
							Name: "key-name-string",
							Type: datapb.PropertyKey_TYPE_STRING,
						},
						Conditions: []*common.SpecCondition{
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
			timeNow := uint64(time.Now().UnixNano())
			dsd := &vegapb.DataSourceDefinition{
				SourceType: &vegapb.DataSourceDefinition_External{
					External: &vegapb.DataSourceDefinitionExternal{
						SourceType: &vegapb.DataSourceDefinitionExternal_EthOracle{
							EthOracle: &vegapb.EthCallSpec{
								Address: "some-eth-address",
								Trigger: &vegapb.EthCallTrigger{
									Trigger: &vegapb.EthCallTrigger_TimeTrigger{
										TimeTrigger: &vegapb.EthTimeTrigger{
											Initial: &timeNow,
										},
									},
								},

								Filters: []*datapb.Filter{
									{
										Key: &datapb.PropertyKey{
											Name: "test-key-name-0",
											Type: common.SpecPropertyKeyType(5),
										},
									},
								},
							},
						},
					},
				},
			}

			dsdt, err := definition.FromProto(dsd, nil)
			assert.NoError(t, err)
			dst := definition.NewWith(dsdt)
			err = dst.UpdateFilters(
				[]*common.SpecFilter{
					{
						Key: &common.SpecPropertyKey{
							Name: "eth-spec-new-property-key",
							Type: datapb.PropertyKey_TYPE_INTEGER,
						},
						Conditions: []*common.SpecCondition{
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
						Key: &common.SpecPropertyKey{
							Name: "key-name-string",
							Type: datapb.PropertyKey_TYPE_STRING,
						},
						Conditions: []*common.SpecCondition{
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

			dsdt, err := definition.FromProto(dsd, nil)
			assert.NoError(t, err)
			dst := definition.NewWith(dsdt)
			err = dst.UpdateFilters(
				[]*common.SpecFilter{
					{
						Conditions: []*common.SpecCondition{
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

		t.Run("NotEmpty timetrigger", func(t *testing.T) {
			dsd := &vegapb.DataSourceDefinition{
				SourceType: &vegapb.DataSourceDefinition_Internal{
					Internal: &vegapb.DataSourceDefinitionInternal{
						SourceType: &vegapb.DataSourceDefinitionInternal_TimeTrigger{
							TimeTrigger: &vegapb.DataSourceSpecConfigurationTimeTrigger{},
						},
					},
				},
			}

			tn := time.Now()
			dsdt, err := definition.FromProto(dsd, &tn)
			assert.NoError(t, err)
			dst := definition.NewWith(dsdt)
			err = dst.UpdateFilters(
				[]*common.SpecFilter{
					{
						Conditions: []*common.SpecCondition{
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

			assert.Equal(t, "vegaprotocol.builtin.timetrigger", filters[0].Key.Name)
			assert.Equal(t, datapb.PropertyKey_TYPE_TIMESTAMP, filters[0].Key.Type)

			assert.Equal(t, 2, len(filters[0].Conditions))
			assert.Equal(t, datapb.Condition_OPERATOR_GREATER_THAN_OR_EQUAL, filters[0].Conditions[0].Operator)
			assert.Equal(t, "int-test-value-1", filters[0].Conditions[0].Value)
			assert.Equal(t, datapb.Condition_OPERATOR_GREATER_THAN, filters[0].Conditions[1].Operator)
			assert.Equal(t, "int-test-value-2", filters[0].Conditions[1].Value)
		})

		t.Run("Empty", func(t *testing.T) {
			dsd := &vegapb.DataSourceDefinition{
				SourceType: &vegapb.DataSourceDefinition_Internal{},
			}

			dsdt, err := definition.FromProto(dsd, nil)
			assert.NoError(t, err)
			dst := definition.NewWith(dsdt)
			err = dst.UpdateFilters(
				[]*common.SpecFilter{},
			)

			assert.NoError(t, err)
			filters := dsdt.(*definition.Definition).GetFilters()
			assert.Equal(t, 0, len(filters))
		})
	})
}

func TestGetSigners(t *testing.T) {
	t.Run("empty signers", func(t *testing.T) {
		ds := definition.NewWith(signedoracle.SpecConfiguration{})

		signers := ds.GetSigners()
		assert.Equal(t, 0, len(signers))
	})

	t.Run("non-empty list but empty signers", func(t *testing.T) {
		ds := definition.NewWith(signedoracle.SpecConfiguration{
			Signers: []*common.Signer{
				{},
				{},
			},
		})

		signers := ds.GetSigners()
		assert.Equal(t, 2, len(signers))
	})

	t.Run("non-empty signers", func(t *testing.T) {
		ds := definition.NewWith(signedoracle.SpecConfiguration{
			Signers: []*common.Signer{
				{
					common.CreateSignerFromString("0xTESTSIGN", common.SignerTypePubKey).Signer,
				},
			},
		})

		signers := ds.GetSigners()
		assert.Equal(t, 1, len(signers))
		assert.IsType(t, &common.Signer{}, signers[0])
		assert.IsType(t, &common.SignerPubKey{}, signers[0].Signer)
		assert.Equal(t, "0xTESTSIGN", signers[0].GetSignerPubKey().Key)
	})
}

func TestGetDataSourceSpecConfiguration(t *testing.T) {
	ds := definition.NewWith(
		signedoracle.SpecConfiguration{
			Signers: []*common.Signer{
				{
					common.CreateSignerFromString("0xTESTSIGN", common.SignerTypePubKey).Signer,
				},
			},
		})

	spec := ds.GetSpecConfiguration()
	assert.NotNil(t, spec)
	assert.IsType(t, signedoracle.SpecConfiguration{}, spec)

	assert.Equal(t, 1, len(spec.(signedoracle.SpecConfiguration).Signers))
	assert.Equal(t, "0xTESTSIGN", spec.(signedoracle.SpecConfiguration).Signers[0].GetSignerPubKey().Key)
}

func TestGetEthCallSpec(t *testing.T) {
	ds := definition.NewWith(ethcallcommon.Spec{
		Address: "some-eth-address",
		AbiJson: []byte("abi-json-test"),
		Method:  "method",
		Trigger: &ethcallcommon.TimeTrigger{
			Initial: uint64(time.Now().UnixNano()),
		},
		RequiredConfirmations: 256,
		Filters: []*common.SpecFilter{
			{
				Key: &common.SpecPropertyKey{
					Name: "test-key-name-0",
					Type: common.SpecPropertyKeyType(3),
				},
			},
		},
	})

	dsSpec := ds.GetEthCallSpec()
	assert.NotNil(t, dsSpec)
	assert.IsType(t, ethcallcommon.Spec{}, dsSpec)
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
	ds := definition.NewWith(vegatime.SpecConfiguration{
		Conditions: []*common.SpecCondition{
			{
				Operator: datapb.Condition_OPERATOR_GREATER_THAN,
				Value:    "1",
			},
		},
	})

	spec := ds.GetSpecConfiguration()
	assert.NotNil(t, spec)
	assert.IsType(t, vegatime.SpecConfiguration{}, spec)

	assert.NotNil(t, spec.(vegatime.SpecConfiguration).Conditions)
	assert.Equal(t, datapb.Condition_OPERATOR_GREATER_THAN, spec.(vegatime.SpecConfiguration).Conditions[0].Operator)
	assert.Equal(t, "1", spec.(vegatime.SpecConfiguration).Conditions[0].Value)
}

func TestGetDataSourceSpecConfigurationTimeTrigger(t *testing.T) {
	ds := definition.NewWith(timetrigger.SpecConfiguration{
		Conditions: []*common.SpecCondition{
			{
				Operator: datapb.Condition_OPERATOR_GREATER_THAN,
				Value:    "1",
			},
		},
		Triggers: common.InternalTimeTriggers{
			{},
		},
	})

	spec := ds.GetSpecConfiguration()
	assert.NotNil(t, spec)
	assert.IsType(t, timetrigger.SpecConfiguration{}, spec)

	assert.NotNil(t, spec.(timetrigger.SpecConfiguration).Conditions)
	assert.Equal(t, datapb.Condition_OPERATOR_GREATER_THAN, spec.(timetrigger.SpecConfiguration).Conditions[0].Operator)
	assert.Equal(t, "1", spec.(timetrigger.SpecConfiguration).Conditions[0].Value)

	assert.IsType(t, common.InternalTimeTriggers{}, spec.(timetrigger.SpecConfiguration).Triggers)
}

func TestIsExternal(t *testing.T) {
	dsDef := definition.NewWith(signedoracle.SpecConfiguration{})

	res, err := dsDef.IsExternal()
	assert.NoError(t, err)
	assert.True(t, res)

	dsDef = definition.NewWith(ethcallcommon.Spec{})

	res, err = dsDef.IsExternal()
	assert.NoError(t, err)
	assert.True(t, res)

	dsDef = definition.NewWith(vegatime.SpecConfiguration{
		Conditions: []*common.SpecCondition{},
	})

	res, err = dsDef.IsExternal()
	assert.NoError(t, err)
	assert.False(t, res)

	dsDef = definition.NewWith(timetrigger.SpecConfiguration{
		Conditions: []*common.SpecCondition{},
		Triggers:   common.InternalTimeTriggers{},
	})

	res, err = dsDef.IsExternal()
	assert.NoError(t, err)
	assert.False(t, res)

	dsDef = definition.NewWith(nil)
	res, _ = dsDef.IsExternal()

	assert.Error(t, errors.New("unknown type of data source provided"))
	assert.False(t, res)
}

func TestSetOracleConfig(t *testing.T) {
	t.Run("non-empty oracle", func(t *testing.T) {
		dsd := definition.NewWith(signedoracle.SpecConfiguration{})

		udsd := dsd.SetOracleConfig(
			&signedoracle.SpecConfiguration{
				Signers: []*common.Signer{
					common.CreateSignerFromString("0xSOMEKEYX", common.SignerTypePubKey),
					common.CreateSignerFromString("0xSOMEKEYY", common.SignerTypePubKey),
				},
				Filters: []*common.SpecFilter{
					{
						Key: &common.SpecPropertyKey{
							Name: "prices.ETH.value",
							Type: datapb.PropertyKey_TYPE_INTEGER,
						},
						Conditions: []*common.SpecCondition{
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
						Key: &common.SpecPropertyKey{
							Name: "key-name-string",
							Type: datapb.PropertyKey_TYPE_STRING,
						},
						Conditions: []*common.SpecCondition{
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
		dsd := definition.NewWith(ethcallcommon.Spec{})

		udsd := dsd.SetOracleConfig(
			&ethcallcommon.Spec{
				Address: "some-eth-address",
				AbiJson: []byte("abi-json-test"),
				Method:  "method",
				Trigger: &ethcallcommon.TimeTrigger{
					Initial: uint64(time.Now().UnixNano()),
				},
				RequiredConfirmations: 256,
				Filters: []*common.SpecFilter{
					{
						Key: &common.SpecPropertyKey{
							Name: "prices.ETH.value",
							Type: datapb.PropertyKey_TYPE_INTEGER,
						},
						Conditions: []*common.SpecCondition{
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
						Key: &common.SpecPropertyKey{
							Name: "key-name-string",
							Type: datapb.PropertyKey_TYPE_STRING,
						},
						Conditions: []*common.SpecCondition{
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
		assert.IsType(t, ethcallcommon.Spec{}, dsSpec)
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
		dsd := definition.NewWith(
			vegatime.SpecConfiguration{
				Conditions: []*common.SpecCondition{
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
			&signedoracle.SpecConfiguration{
				Filters: []*common.SpecFilter{
					{
						Key: &common.SpecPropertyKey{
							Name: "prices.ETH.value",
							Type: datapb.PropertyKey_TYPE_INTEGER,
						},
						Conditions: []*common.SpecCondition{
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
						Key: &common.SpecPropertyKey{
							Name: "key-name-string",
							Type: datapb.PropertyKey_TYPE_STRING,
						},
						Conditions: []*common.SpecCondition{
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
	dsd := definition.NewWith(signedoracle.SpecConfiguration{})

	udsd := dsd.SetTimeTriggerConditionConfig(
		[]*common.SpecCondition{
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

	t.Run("try to set time config to internal time data source", func(t *testing.T) {
		dsd := definition.NewWith(
			vegatime.SpecConfiguration{
				Conditions: []*common.SpecCondition{
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
			[]*common.SpecCondition{
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

		assert.Equal(t, 1, len(filters[0].Conditions))
		assert.Equal(t, datapb.Condition_OPERATOR_GREATER_THAN_OR_EQUAL, filters[0].Conditions[0].Operator)
		assert.Equal(t, "int-test-value-3", filters[0].Conditions[0].Value)
	})

	t.Run("try to set time trigger config to internal time data source", func(t *testing.T) {
		dsd := definition.NewWith(
			timetrigger.SpecConfiguration{
				Conditions: []*common.SpecCondition{
					{
						Operator: datapb.Condition_OPERATOR_GREATER_THAN_OR_EQUAL,
						Value:    "int-test-value-1",
					},
					{
						Operator: datapb.Condition_OPERATOR_GREATER_THAN,
						Value:    "int-test-value-2",
					},
				},
				Triggers: common.InternalTimeTriggers{
					{},
				},
			})

		iudsd := dsd.SetTimeTriggerConditionConfig(
			[]*common.SpecCondition{
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

		assert.Equal(t, "vegaprotocol.builtin.timetrigger", filters[0].Key.Name)
		assert.Equal(t, datapb.PropertyKey_TYPE_TIMESTAMP, filters[0].Key.Type)

		assert.Equal(t, 2, len(filters[0].Conditions))
		assert.Equal(t, datapb.Condition_OPERATOR_GREATER_THAN_OR_EQUAL, filters[0].Conditions[0].Operator)
		assert.Equal(t, "int-test-value-3", filters[0].Conditions[0].Value)
		assert.Equal(t, datapb.Condition_OPERATOR_GREATER_THAN, filters[0].Conditions[1].Operator)
		assert.Equal(t, "int-test-value-4", filters[0].Conditions[1].Value)
	})
}

func TestType(t *testing.T) {
	ds := definition.NewWith(&signedoracle.SpecConfiguration{})
	tp, ext := ds.Type()
	assert.Equal(t, definition.ContentTypeOracle, tp)
	assert.True(t, ext)

	ds = definition.NewWith(&ethcallcommon.Spec{})
	tp, ext = ds.Type()
	assert.Equal(t, definition.ContentTypeEthOracle, tp)
	assert.True(t, ext)

	ds = definition.NewWith(&vegatime.SpecConfiguration{})
	tp, ext = ds.Type()
	assert.Equal(t, definition.ContentTypeInternalTimeTermination, tp)
	assert.False(t, ext)

	ds = definition.NewWith(&definition.Definition{})
	tp, ext = ds.Type()
	assert.Equal(t, definition.ContentTypeInvalid, tp)
	assert.False(t, ext)
}
