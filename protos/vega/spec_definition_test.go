package vega_test

import (
	"testing"

	"code.vegaprotocol.io/vega/core/types"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
	datapb "code.vegaprotocol.io/vega/protos/vega/data/v1"
	"github.com/stretchr/testify/assert"
)

func TestGetFilters(t *testing.T) {
	t.Run("testGetFiltersExternal", func(t *testing.T) {
		t.Run("NotEmpty", func(t *testing.T) {
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
		&vegapb.DataSourceSpecConfiguration{
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
			&vegapb.DataSourceSpecConfiguration{
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
