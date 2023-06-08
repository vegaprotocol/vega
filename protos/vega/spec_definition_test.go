package vega_test

import (
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/types"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
	datapb "code.vegaprotocol.io/vega/protos/vega/data/v1"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestGetFilters(t *testing.T) {
	t.Run("testGetFiltersExternal", func(t *testing.T) {
		t.Run("NotEmpty Oracle", func(t *testing.T) {
			dsd := &vegapb.DataSourceDefinition{
				SourceType: &vegapb.DataSourceDefinition_External{
					External: &vegapb.DataSourceDefinitionExternal{
						SourceType: &vegapb.DataSourceDefinitionExternal_Oracle{
							Oracle: &vegapb.DataSourceSpecConfiguration{
								Signers: []*datapb.Signer{},
								Filters: []*datapb.Filter{
									{
										Key: &datapb.PropertyKey{
											Name: "prices.ETH.value",
											Type: datapb.PropertyKey_TYPE_INTEGER,
										},
										Conditions: []*datapb.Condition{
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
										Key: &datapb.PropertyKey{
											Name: "key-name-string",
											Type: datapb.PropertyKey_TYPE_STRING,
										},
										Conditions: []*datapb.Condition{
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
						},
					},
				},
			}

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
			timeNow := uint64(time.Now().UnixNano())
			dsd := &vegapb.DataSourceDefinition{
				SourceType: &vegapb.DataSourceDefinition_External{
					External: &vegapb.DataSourceDefinitionExternal{
						SourceType: &vegapb.DataSourceDefinitionExternal_EthOracle{
							EthOracle: &vegapb.EthCallSpec{
								Address: "some-eth-address",
								Abi: &structpb.ListValue{
									Values: []*structpb.Value{
										{
											Kind: &structpb.Value_StringValue{
												StringValue: "string-value",
											},
										},
									},
								},
								Method: "test-method",
								Args: []*structpb.Value{
									{
										Kind: &structpb.Value_StringValue{
											StringValue: "string-arg",
										},
									},
								},
								Trigger: &vegapb.EthCallTrigger{
									Trigger: &vegapb.EthCallTrigger_TimeTrigger{
										TimeTrigger: &vegapb.EthTimeTrigger{
											Initial: &timeNow,
										},
									},
								},
								RequiredConfirmations: 256,
								Filters: []*datapb.Filter{
									{
										Key: &datapb.PropertyKey{
											Name: "test-key-name-0",
											Type: types.DataSourceSpecPropertyKeyType(2),
										},
									},
								},
							},
						},
					},
				},
			}

			filters := dsd.GetFilters()
			assert.Equal(t, 1, len(filters))
			assert.Equal(t, "test-key-name-0", filters[0].Key.Name)
			assert.Equal(t, datapb.PropertyKey_TYPE_INTEGER, filters[0].Key.Type)
			assert.Equal(t, 0, len(filters[0].Conditions))
		})

		t.Run("Empty", func(t *testing.T) {
			dsd := &vegapb.DataSourceDefinition{
				SourceType: &vegapb.DataSourceDefinition_External{},
			}

			filters := dsd.GetFilters()
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

			filters = dsd.GetFilters()
			assert.Equal(t, 0, len(filters))
		})
	})

	t.Run("testGetFiltersInternal", func(t *testing.T) {
		t.Run("NotEmpty", func(t *testing.T) {
			dsd := &vegapb.DataSourceDefinition{
				SourceType: &vegapb.DataSourceDefinition_Internal{
					Internal: &vegapb.DataSourceDefinitionInternal{
						SourceType: &vegapb.DataSourceDefinitionInternal_Time{
							Time: &vegapb.DataSourceSpecConfigurationTime{
								Conditions: []*datapb.Condition{
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
					},
				},
			}

			filters := dsd.GetFilters()
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

			filters := dsd.GetFilters()

			assert.Equal(t, 0, len(filters))

			dsd = &vegapb.DataSourceDefinition{
				SourceType: &vegapb.DataSourceDefinition_Internal{
					Internal: &vegapb.DataSourceDefinitionInternal{
						SourceType: &vegapb.DataSourceDefinitionInternal_Time{
							Time: &vegapb.DataSourceSpecConfigurationTime{
								Conditions: []*datapb.Condition{},
							},
						},
					},
				},
			}

			filters = dsd.GetFilters()
			assert.Equal(t, 0, len(filters))
		})
	})
}

func TestSetOracleConfig(t *testing.T) {
	t.Run("empty", func(t *testing.T) {

	})
	t.Run("non-empty oracle", func(t *testing.T) {
		dsd := &vegapb.DataSourceDefinition{
			SourceType: &vegapb.DataSourceDefinition_External{
				External: &vegapb.DataSourceDefinitionExternal{
					SourceType: &vegapb.DataSourceDefinitionExternal_Oracle{
						Oracle: nil,
					},
				},
			},
		}

		udsd := dsd.SetOracleConfig(
			&vegapb.DataSourceDefinitionExternal_Oracle{
				Oracle: &vegapb.DataSourceSpecConfiguration{
					Signers: types.SignersIntoProto(
						[]*types.Signer{
							types.CreateSignerFromString("0xSOMEKEYX", types.DataSignerTypePubKey),
							types.CreateSignerFromString("0xSOMEKEYY", types.DataSignerTypePubKey),
						}),
					Filters: []*datapb.Filter{
						{
							Key: &datapb.PropertyKey{
								Name: "prices.ETH.value",
								Type: datapb.PropertyKey_TYPE_INTEGER,
							},
							Conditions: []*datapb.Condition{
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
							Key: &datapb.PropertyKey{
								Name: "key-name-string",
								Type: datapb.PropertyKey_TYPE_STRING,
							},
							Conditions: []*datapb.Condition{
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
			},
		)

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
		dsd := &vegapb.DataSourceDefinition{
			SourceType: &vegapb.DataSourceDefinition_External{
				External: &vegapb.DataSourceDefinitionExternal{
					SourceType: &vegapb.DataSourceDefinitionExternal_EthOracle{
						EthOracle: nil,
					},
				},
			},
		}
		timeNow := uint64(time.Now().UnixNano())
		udsd := dsd.SetOracleConfig(
			&vegapb.DataSourceDefinitionExternal_EthOracle{
				EthOracle: &vegapb.EthCallSpec{
					Address: "some-eth-address",
					Abi: &structpb.ListValue{
						Values: []*structpb.Value{
							{
								Kind: &structpb.Value_StringValue{
									StringValue: "string-value",
								},
							},
						},
					},
					Method: "test-method",
					Args: []*structpb.Value{
						{
							Kind: &structpb.Value_StringValue{
								StringValue: "string-arg",
							},
						},
					},
					Trigger: &vegapb.EthCallTrigger{
						Trigger: &vegapb.EthCallTrigger_TimeTrigger{
							TimeTrigger: &vegapb.EthTimeTrigger{
								Initial: &timeNow,
							},
						},
					},
					RequiredConfirmations: 256,
					Filters: []*datapb.Filter{
						{
							Key: &datapb.PropertyKey{
								Name: "prices.ETH.value",
								Type: datapb.PropertyKey_TYPE_INTEGER,
							},
							Conditions: []*datapb.Condition{
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
							Key: &datapb.PropertyKey{
								Name: "key-name-string",
								Type: datapb.PropertyKey_TYPE_STRING,
							},
							Conditions: []*datapb.Condition{
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
			},
		)

		eo := udsd.GetExternal().GetEthOracle()
		assert.Equal(t, "some-eth-address", eo.Address)
		assert.Equal(t, 1, len(eo.GetAbi().Values))
		assert.Equal(t, "string-value", eo.GetAbi().Values[0].GetStringValue())
		assert.Equal(t, "test-method", eo.Method)
		assert.Equal(t, 1, len(eo.GetArgs()))
		assert.Equal(t, "string-arg", eo.GetArgs()[0].GetStringValue())
		assert.Equal(t, &timeNow, eo.GetTrigger().GetTimeTrigger().Initial)

		filters := eo.GetFilters()
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
		dsd := &vegapb.DataSourceDefinition{
			SourceType: &vegapb.DataSourceDefinition_Internal{
				Internal: &vegapb.DataSourceDefinitionInternal{
					SourceType: &vegapb.DataSourceDefinitionInternal_Time{
						Time: &vegapb.DataSourceSpecConfigurationTime{
							Conditions: []*datapb.Condition{
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
				},
			},
		}

		iudsd := dsd.SetOracleConfig(
			&vegapb.DataSourceDefinitionExternal_Oracle{
				Oracle: &vegapb.DataSourceSpecConfiguration{
					Filters: []*datapb.Filter{
						{
							Key: &datapb.PropertyKey{
								Name: "prices.ETH.value",
								Type: datapb.PropertyKey_TYPE_INTEGER,
							},
							Conditions: []*datapb.Condition{
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
							Key: &datapb.PropertyKey{
								Name: "key-name-string",
								Type: datapb.PropertyKey_TYPE_STRING,
							},
							Conditions: []*datapb.Condition{
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
	dsd := &vegapb.DataSourceDefinition{
		SourceType: &vegapb.DataSourceDefinition_External{
			External: &vegapb.DataSourceDefinitionExternal{
				SourceType: &vegapb.DataSourceDefinitionExternal_Oracle{
					Oracle: nil,
				},
			},
		},
	}

	udsd := dsd.SetTimeTriggerConditionConfig(
		[]*datapb.Condition{
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

	t.Run("non-empty internal data source", func(t *testing.T) {
		dsd := &vegapb.DataSourceDefinition{
			SourceType: &vegapb.DataSourceDefinition_Internal{
				Internal: &vegapb.DataSourceDefinitionInternal{
					SourceType: &vegapb.DataSourceDefinitionInternal_Time{
						Time: &vegapb.DataSourceSpecConfigurationTime{
							Conditions: []*datapb.Condition{
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
				},
			},
		}

		iudsd := dsd.SetTimeTriggerConditionConfig(
			[]*datapb.Condition{
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

func TestContent(t *testing.T) {
	t.Run("testContent", func(t *testing.T) {
		t.Run("empty content", func(t *testing.T) {
			d := &vegapb.DataSourceDefinition{}
			assert.Nil(t, d.Content())

			d = &vegapb.DataSourceDefinition{
				SourceType: &vegapb.DataSourceDefinition_External{
					External: &vegapb.DataSourceDefinitionExternal{
						SourceType: &vegapb.DataSourceDefinitionExternal_Oracle{
							Oracle: nil,
						},
					},
				},
			}

			assert.Nil(t, d.Content())

			d = &vegapb.DataSourceDefinition{
				SourceType: &vegapb.DataSourceDefinition_External{
					External: &vegapb.DataSourceDefinitionExternal{
						SourceType: &vegapb.DataSourceDefinitionExternal_Oracle{
							Oracle: &vegapb.DataSourceSpecConfiguration{},
						},
					},
				},
			}

			c := d.Content()
			assert.NotNil(t, c)
			tp, ok := c.(*vegapb.DataSourceSpecConfiguration)
			assert.True(t, ok)
			assert.Equal(t, 0, len(tp.Filters))
			assert.Equal(t, 0, len(tp.Signers))

			d = &vegapb.DataSourceDefinition{
				SourceType: &vegapb.DataSourceDefinition_Internal{
					Internal: &vegapb.DataSourceDefinitionInternal{
						SourceType: &vegapb.DataSourceDefinitionInternal_Time{
							Time: nil,
						},
					},
				},
			}

			assert.Nil(t, d.Content())

			d = &vegapb.DataSourceDefinition{
				SourceType: &vegapb.DataSourceDefinition_Internal{
					Internal: &vegapb.DataSourceDefinitionInternal{
						SourceType: &vegapb.DataSourceDefinitionInternal_Time{
							Time: &vegapb.DataSourceSpecConfigurationTime{},
						},
					},
				},
			}

			ct := d.Content()
			assert.NotNil(t, ct)
			tpt, ok := ct.(*vegapb.DataSourceSpecConfigurationTime)
			assert.True(t, ok)
			assert.Equal(t, 0, len(tpt.Conditions))
		})

		t.Run("non-empty content with time termiation source", func(t *testing.T) {
			d := &vegapb.DataSourceDefinition{
				SourceType: &vegapb.DataSourceDefinition_Internal{
					Internal: &vegapb.DataSourceDefinitionInternal{
						SourceType: &vegapb.DataSourceDefinitionInternal_Time{
							Time: &vegapb.DataSourceSpecConfigurationTime{
								Conditions: []*datapb.Condition{
									{
										Operator: datapb.Condition_OPERATOR_EQUALS,
										Value:    "ext-test-value-0",
									},
									{
										Operator: datapb.Condition_OPERATOR_GREATER_THAN,
										Value:    "ext-test-value-1",
									},
								},
							},
						},
					},
				},
			}

			c := d.Content()
			assert.NotNil(t, c)
			assert.IsType(t, &vegapb.DataSourceSpecConfigurationTime{}, c)
			tp, ok := c.(*vegapb.DataSourceSpecConfigurationTime)
			assert.True(t, ok)
			assert.Equal(t, 2, len(tp.Conditions))
			assert.Equal(t, "ext-test-value-0", tp.Conditions[0].Value)
			assert.Equal(t, "ext-test-value-1", tp.Conditions[1].Value)
		})

		t.Run("non-empty content with oracle", func(t *testing.T) {
			d := &vegapb.DataSourceDefinition{
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
			c := d.Content()
			assert.NotNil(t, c)
			assert.IsType(t, &vegapb.DataSourceSpecConfiguration{}, c)
			tp, ok := c.(*vegapb.DataSourceSpecConfiguration)
			assert.True(t, ok)
			assert.Equal(t, 2, len(tp.Signers))
			assert.Equal(t, "0xSOMEKEYX", tp.Signers[0].GetPubKey().GetKey())
		})

		t.Run("non-empty content with ethereum oracle", func(t *testing.T) {
			timeNow := uint64(time.Now().UnixNano())
			d := &vegapb.DataSourceDefinition{
				SourceType: &vegapb.DataSourceDefinition_External{
					External: &vegapb.DataSourceDefinitionExternal{
						SourceType: &vegapb.DataSourceDefinitionExternal_EthOracle{
							EthOracle: &vegapb.EthCallSpec{
								Address: "some-eth-address",
								Abi: &structpb.ListValue{
									Values: []*structpb.Value{
										{
											Kind: &structpb.Value_StringValue{
												StringValue: "string-value",
											},
										},
									},
								},
								Method: "test-method",
								Args: []*structpb.Value{
									{
										Kind: &structpb.Value_StringValue{
											StringValue: "string-arg",
										},
									},
								},
								Trigger: &vegapb.EthCallTrigger{
									Trigger: &vegapb.EthCallTrigger_TimeTrigger{
										TimeTrigger: &vegapb.EthTimeTrigger{
											Initial: &timeNow,
										},
									},
								},
								RequiredConfirmations: 256,
								Filters: []*datapb.Filter{
									{
										Key: &datapb.PropertyKey{
											Name: "prices.ETH.value",
											Type: datapb.PropertyKey_TYPE_INTEGER,
										},
										Conditions: []*datapb.Condition{
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
										Key: &datapb.PropertyKey{
											Name: "key-name-string",
											Type: datapb.PropertyKey_TYPE_STRING,
										},
										Conditions: []*datapb.Condition{
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
						},
					},
				},
			}
			c := d.Content()
			assert.NotNil(t, c)
			assert.IsType(t, &vegapb.EthCallSpec{}, c)
			eo, ok := c.(*vegapb.EthCallSpec)
			assert.True(t, ok)
			assert.Equal(t, "some-eth-address", eo.Address)
			assert.Equal(t, 1, len(eo.GetAbi().Values))
			assert.Equal(t, "string-value", eo.GetAbi().Values[0].GetStringValue())
			assert.Equal(t, "test-method", eo.Method)
			assert.Equal(t, 1, len(eo.GetArgs()))
			assert.Equal(t, "string-arg", eo.GetArgs()[0].GetStringValue())
			assert.Equal(t, &timeNow, eo.GetTrigger().GetTimeTrigger().Initial)

			filters := eo.GetFilters()
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
	})
}

func TestNewDataSourceDefinition(t *testing.T) {

}
