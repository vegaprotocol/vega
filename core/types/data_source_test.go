package types_test

import (
	"errors"
	"testing"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/protos/vega"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
	datapb "code.vegaprotocol.io/vega/protos/vega/data/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDataSourceDefinitionIntoProto(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		ds := &types.DataSourceDefinition{}
		protoDs := ds.IntoProto()
		assert.IsType(t, &vegapb.DataSourceDefinition{}, protoDs)
		assert.Nil(t, protoDs.SourceType)
	})

	t.Run("external dataSourceDefinition", func(t *testing.T) {
		t.Run("non-empty oracle with empty lists", func(t *testing.T) {
			ds := types.NewDataSourceDefinitionWith(types.DataSourceSpecConfiguration{})
			protoDs := ds.IntoProto()
			assert.IsType(t, &vegapb.DataSourceDefinition{}, protoDs)
			assert.NotNil(t, protoDs.SourceType)
			ext := protoDs.GetExternal()
			assert.NotNil(t, ext)
			o := ext.GetOracle()
			assert.Equal(t, 0, len(o.Signers))
			assert.Equal(t, 0, len(o.Filters))
		})

		t.Run("non-empty oracle with data", func(t *testing.T) {
			ds := types.NewDataSourceDefinitionWith(types.DataSourceSpecConfiguration{
				Signers: []*types.Signer{
					{},
				},
				Filters: []*types.DataSourceSpecFilter{
					{
						Key: &types.DataSourceSpecPropertyKey{
							Name: "test-name",
							Type: types.DataSourceSpecPropertyKeyType(0),
						},
					},
				},
			})

			protoDs := ds.IntoProto()
			assert.IsType(t, &vegapb.DataSourceDefinition{}, protoDs)
			assert.NotNil(t, protoDs.SourceType)
			ext := protoDs.GetExternal()
			assert.NotNil(t, ext)
			o := ext.GetOracle()
			assert.Equal(t, 1, len(o.Signers))
			assert.Nil(t, o.Signers[0].Signer)
			assert.Equal(t, 1, len(o.Filters))
			assert.NotNil(t, o.Filters[0].Conditions)
			assert.NotNil(t, o.Filters[0].Key)
		})
	})

	t.Run("internal dataSourceDefinition", func(t *testing.T) {
		t.Run("non-empty time source with empty lists", func(t *testing.T) {
			ds := types.NewDataSourceDefinitionWith(types.DataSourceSpecConfigurationTime{})
			protoDs := ds.IntoProto()
			assert.IsType(t, &vegapb.DataSourceDefinition{}, protoDs)
			assert.NotNil(t, protoDs.SourceType)
			ext := protoDs.GetInternal()
			assert.NotNil(t, ext)
			o := ext.GetTime()
			assert.Equal(t, 0, len(o.Conditions))
		})

		t.Run("non-empty time source with data", func(t *testing.T) {
			ds := types.NewDataSourceDefinitionWith(types.DataSourceSpecConfigurationTime{
				Conditions: []*types.DataSourceSpecCondition{
					{},
					{
						Operator: datapb.Condition_OPERATOR_EQUALS,
						Value:    "14",
					},
					{
						Operator: datapb.Condition_OPERATOR_GREATER_THAN,
						Value:    "9",
					},
				},
			})

			protoDs := ds.IntoProto()
			assert.IsType(t, &vegapb.DataSourceDefinition{}, protoDs)
			assert.NotNil(t, protoDs.SourceType)
			ext := protoDs.GetInternal()
			assert.NotNil(t, ext)
			o := ext.GetTime()
			assert.Equal(t, 3, len(o.Conditions))
			assert.Equal(t, datapb.Condition_Operator(0), o.Conditions[0].Operator)
			assert.Equal(t, "", o.Conditions[0].Value)
			assert.Equal(t, datapb.Condition_OPERATOR_EQUALS, o.Conditions[1].Operator)
			assert.Equal(t, "14", o.Conditions[1].Value)
			assert.Equal(t, datapb.Condition_OPERATOR_GREATER_THAN, o.Conditions[2].Operator)
			assert.Equal(t, "9", o.Conditions[2].Value)
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
		t.Run("NotEmpty", func(t *testing.T) {
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
		t.Run("NotEmpty", func(t *testing.T) {
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

			dsdt := types.DataSourceDefinitionFromProto(dsd)
			err := dsdt.UpdateFilters(
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

			filters := dsdt.GetFilters()
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

		t.Run("Empty", func(t *testing.T) {
			dsd := &vegapb.DataSourceDefinition{
				SourceType: &vegapb.DataSourceDefinition_External{},
			}

			dsdt := types.DataSourceDefinitionFromProto(dsd)

			err := dsdt.UpdateFilters([]*types.DataSourceSpecFilter{})
			assert.NoError(t, err)
			filters := dsdt.GetFilters()
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

			dsdt = types.DataSourceDefinitionFromProto(dsd)
			filters = dsdt.GetFilters()
			assert.Equal(t, 0, len(filters))
		})
	})

	t.Run("testUpdateFiltersInternal", func(t *testing.T) {
		t.Run("NotEmpty", func(t *testing.T) {
			dsd := &vegapb.DataSourceDefinition{
				SourceType: &vegapb.DataSourceDefinition_Internal{
					Internal: &vegapb.DataSourceDefinitionInternal{
						SourceType: &vegapb.DataSourceDefinitionInternal_Time{
							Time: &vega.DataSourceSpecConfigurationTime{},
						},
					},
				},
			}

			dsdt := types.DataSourceDefinitionFromProto(dsd)
			err := dsdt.UpdateFilters(
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
			filters := dsdt.GetFilters()
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

			dsdt := types.DataSourceDefinitionFromProto(dsd)
			filters := dsdt.GetFilters()

			assert.Equal(t, 0, len(filters))

			dsd = &vegapb.DataSourceDefinition{
				SourceType: &vegapb.DataSourceDefinition_Internal{
					Internal: &vegapb.DataSourceDefinitionInternal{
						SourceType: &vegapb.DataSourceDefinitionInternal_Time{
							Time: &vega.DataSourceSpecConfigurationTime{
								Conditions: []*datapb.Condition{},
							},
						},
					},
				},
			}

			dsdt = types.DataSourceDefinitionFromProto(dsd)
			err := dsdt.UpdateFilters(
				[]*types.DataSourceSpecFilter{},
			)
			assert.NoError(t, err)
			filters = dsdt.GetFilters()
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
