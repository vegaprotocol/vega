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

package oracles_test

import (
	"errors"
	"testing"

	"code.vegaprotocol.io/vega/core/oracles"
	"code.vegaprotocol.io/vega/core/types"
	datapb "code.vegaprotocol.io/vega/protos/vega/data/v1"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOracleSpec(t *testing.T) {
	t.Run("Creating builtin oracle without pubkeys succeeeds", testBuiltInOracleSpecCreatingWithoutPubKeysSucceeds)
	t.Run("Creating with filters but without key fails", testOracleSpecCreatingWithFiltersWithoutKeyFails)
	t.Run("Creating with split filters with same type works", testOracleSpecCreatingWithSplitFiltersWithSameTypeFails)
	t.Run("Creating with filters with inconvertible type fails", testOracleSpecCreatingWithFiltersWithInconvertibleTypeFails)
	t.Run("Matching with unauthorized public keys fails", testOracleSpecMatchingUnauthorizedPubKeysFails)
	t.Run("Matching with authorized public keys succeeds", testOracleSpecMatchingAuthorizedPubKeysSucceeds)
	t.Run("Matching with equal properties works", testOracleSpecMatchingEqualPropertiesWorks)
	t.Run("Matching with greater than properties works", testOracleSpecMatchingGreaterThanPropertiesWorks)
	t.Run("Matching with greater than or equal properties works", testOracleSpecMatchingGreaterThanOrEqualPropertiesWorks)
	t.Run("Matching with less than properties succeeds only for non-time based spec", testOracleSpecMatchingLessThanPropertiesSucceedsOnlyForNonTimestamp)
	t.Run("Matching with less than or equal properties succeeds only for non-time based spec", testOracleSpecMatchingLessThanOrEqualPropertiesSucceedsOnlyForNonTimestamp)
	t.Run("Matching presence of present properties succeeds", testOracleSpecMatchingPropertiesPresenceSucceeds)
	t.Run("Matching presence of missing properties fails", testOracleSpecMatchingPropertiesPresenceFails)
	t.Run("Matching with inconvertible type fails", testOracleSpecMatchingWithInconvertibleTypeFails)
	t.Run("Verifying binding of property works", testOracleSpecVerifyingBindingWorks)
}

func testBuiltInOracleSpecCreatingWithoutPubKeysSucceeds(t *testing.T) {
	// given
	spec := types.ExternalDataSourceSpec{
		Spec: &types.DataSourceSpec{
			Data: types.NewDataSourceDefinition(
				types.DataSourceContentTypeOracle,
			).SetOracleConfig(
				&types.DataSourceSpecConfiguration{
					Signers: []*types.Signer{},
					Filters: []*types.DataSourceSpecFilter{
						{
							Key: &types.DataSourceSpecPropertyKey{
								Name: "vegaprotocol.builtin.timestamp",
								Type: datapb.PropertyKey_TYPE_TIMESTAMP,
							},
							Conditions: []*types.DataSourceSpecCondition{},
						},
					},
				},
			),
		},
	}

	// when
	oracleSpec, err := oracles.NewOracleSpec(spec)

	// then
	require.NoError(t, err)
	assert.NotNil(t, oracleSpec)
}

func testOracleSpecCreatingWithFiltersWithoutKeyFails(t *testing.T) {
	// given
	spec := types.ExternalDataSourceSpec{
		Spec: &types.DataSourceSpec{
			Data: types.NewDataSourceDefinition(
				types.DataSourceContentTypeOracle,
			).SetOracleConfig(
				&types.DataSourceSpecConfiguration{
					Signers: []*types.Signer{
						types.CreateSignerFromString("0xCAFED00D", types.DataSignerTypePubKey),
					},
					Filters: []*types.DataSourceSpecFilter{
						{
							Key:        nil,
							Conditions: nil,
						},
					},
				},
			),
		},
	}

	// when
	oracleSpec, err := oracles.NewOracleSpec(spec)

	// then
	require.Error(t, err)
	assert.Equal(t, "a property key is required", err.Error())
	assert.Nil(t, oracleSpec)
}

func testOracleSpecCreatingWithSplitFiltersWithSameTypeFails(t *testing.T) {
	// given
	spec, err := oracles.NewOracleSpec(types.ExternalDataSourceSpec{
		Spec: &types.DataSourceSpec{
			Data: types.NewDataSourceDefinition(
				types.DataSourceContentTypeOracle,
			).SetOracleConfig(
				&types.DataSourceSpecConfiguration{
					Signers: []*types.Signer{
						types.CreateSignerFromString("0xDEADBEEF", types.DataSignerTypePubKey),
						types.CreateSignerFromString("0xCAFED00D", types.DataSignerTypePubKey),
					},
					Filters: []*types.DataSourceSpecFilter{
						{
							Key: &types.DataSourceSpecPropertyKey{
								Name: "prices.BTC.value",
								Type: datapb.PropertyKey_TYPE_INTEGER,
							},
							Conditions: []*types.DataSourceSpecCondition{
								{
									Value:    "42",
									Operator: datapb.Condition_OPERATOR_GREATER_THAN,
								},
							},
						}, {
							Key: &types.DataSourceSpecPropertyKey{
								Name: "prices.BTC.value",
								Type: datapb.PropertyKey_TYPE_INTEGER,
							},
							Conditions: []*types.DataSourceSpecCondition{
								{
									Value:    "84",
									Operator: datapb.Condition_OPERATOR_LESS_THAN,
								},
							},
						},
					},
				},
			),
		},
	})

	assert.ErrorIs(t, types.ErrDataSourceSpecHasMultipleSameKeyNamesInFilterList, err)
	assert.Nil(t, spec)
}

func testOracleSpecCreatingWithFiltersWithInconvertibleTypeFails(t *testing.T) {
	cases := []struct {
		msg   string
		typ   datapb.PropertyKey_Type
		value string
	}{
		{
			msg:   "not an integer",
			typ:   datapb.PropertyKey_TYPE_INTEGER,
			value: "not an integer",
		}, {
			msg:   "not a boolean",
			typ:   datapb.PropertyKey_TYPE_BOOLEAN,
			value: "42",
		}, {
			msg:   "not a decimal",
			typ:   datapb.PropertyKey_TYPE_DECIMAL,
			value: "not a decimal",
		}, {
			msg:   "not a timestamp",
			typ:   datapb.PropertyKey_TYPE_TIMESTAMP,
			value: "not a timestamp",
		},
	}

	for _, c := range cases {
		t.Run(c.msg, func(t *testing.T) {
			// given
			originalSpec := types.ExternalDataSourceSpec{
				Spec: &types.DataSourceSpec{
					Data: types.NewDataSourceDefinition(
						types.DataSourceContentTypeOracle,
					).SetOracleConfig(
						&types.DataSourceSpecConfiguration{
							Signers: []*types.Signer{
								types.CreateSignerFromString("0xCAFED00D", types.DataSignerTypePubKey),
							},
							Filters: []*types.DataSourceSpecFilter{
								{
									Key: &types.DataSourceSpecPropertyKey{
										Name: "prices.BTC.value",
										Type: c.typ,
									},
									Conditions: []*types.DataSourceSpecCondition{
										{
											Value:    c.value,
											Operator: datapb.Condition_OPERATOR_EQUALS,
										},
									},
								},
							},
						},
					),
				},
			}

			// when
			spec, err := oracles.NewOracleSpec(originalSpec)

			// then
			require.Error(t, err)
			assert.Nil(t, spec)
		})
	}
}

func testOracleSpecMatchingUnauthorizedPubKeysFails(t *testing.T) {
	// given
	spec, _ := oracles.NewOracleSpec(types.ExternalDataSourceSpec{
		Spec: &types.DataSourceSpec{
			Data: types.NewDataSourceDefinition(
				types.DataSourceContentTypeOracle,
			).SetOracleConfig(
				&types.DataSourceSpecConfiguration{
					Signers: []*types.Signer{
						types.CreateSignerFromString("0xDEADBEEF", types.DataSignerTypePubKey),
						types.CreateSignerFromString("0xCAFED00D", types.DataSignerTypePubKey),
					},
					Filters: []*types.DataSourceSpecFilter{
						{
							Key: &types.DataSourceSpecPropertyKey{
								Name: "prices.BTC.value",
								Type: datapb.PropertyKey_TYPE_INTEGER,
							},
							Conditions: []*types.DataSourceSpecCondition{
								{
									Value:    "42",
									Operator: datapb.Condition_OPERATOR_EQUALS,
								},
							},
						},
					},
				},
			),
		},
	})

	data := oracles.OracleData{
		Signers: []*types.Signer{
			types.CreateSignerFromString("0xDEADBEEF", types.DataSignerTypePubKey),
			types.CreateSignerFromString("0xBADDCAFE", types.DataSignerTypePubKey),
		},
		Data: map[string]string{
			"prices.BTC.value": "42",
		},
	}

	// when
	matched, err := spec.MatchData(data)

	// then
	require.NoError(t, err)
	assert.False(t, matched)
}

func testOracleSpecMatchingAuthorizedPubKeysSucceeds(t *testing.T) {
	// given
	spec, _ := oracles.NewOracleSpec(types.ExternalDataSourceSpec{
		Spec: &types.DataSourceSpec{
			Data: types.NewDataSourceDefinition(
				types.DataSourceContentTypeOracle,
			).SetOracleConfig(
				&types.DataSourceSpecConfiguration{
					Signers: []*types.Signer{
						types.CreateSignerFromString("0xDEADBEEF", types.DataSignerTypePubKey),
						types.CreateSignerFromString("0xCAFED00D", types.DataSignerTypePubKey),
						types.CreateSignerFromString("0xBADDCAFE", types.DataSignerTypePubKey),
					},
					Filters: []*types.DataSourceSpecFilter{
						{
							Key: &types.DataSourceSpecPropertyKey{
								Name: "prices.BTC.value",
								Type: datapb.PropertyKey_TYPE_INTEGER,
							},
							Conditions: []*types.DataSourceSpecCondition{
								{
									Value:    "42",
									Operator: datapb.Condition_OPERATOR_EQUALS,
								},
							},
						},
					},
				},
			),
		},
	})

	data := oracles.OracleData{
		Signers: []*types.Signer{
			types.CreateSignerFromString("0xDEADBEEF", types.DataSignerTypePubKey),
			types.CreateSignerFromString("0xBADDCAFE", types.DataSignerTypePubKey),
		},
		Data: map[string]string{
			"prices.BTC.value": "42",
		},
	}

	// when
	matched, err := spec.MatchData(data)

	// then
	require.NoError(t, err)
	assert.True(t, matched)
}

func testOracleSpecMatchingEqualPropertiesWorks(t *testing.T) {
	cases := []struct {
		msg       string
		keyType   datapb.PropertyKey_Type
		specValue string
		dataValue string
		matched   bool
	}{
		{
			msg:       "integer values should be equal",
			keyType:   datapb.PropertyKey_TYPE_INTEGER,
			specValue: "42",
			dataValue: "42",
			matched:   true,
		}, {
			msg:       "integer values should not be equal",
			keyType:   datapb.PropertyKey_TYPE_INTEGER,
			specValue: "84",
			dataValue: "42",
			matched:   false,
		}, {
			msg:       "boolean values should be equal",
			keyType:   datapb.PropertyKey_TYPE_BOOLEAN,
			specValue: "true",
			dataValue: "true",
			matched:   true,
		}, {
			msg:       "boolean values should not be equal",
			keyType:   datapb.PropertyKey_TYPE_BOOLEAN,
			specValue: "true",
			dataValue: "false",
			matched:   false,
		}, {
			msg:       "decimal values should be equal",
			keyType:   datapb.PropertyKey_TYPE_DECIMAL,
			specValue: "1.2",
			dataValue: "1.2",
			matched:   true,
		}, {
			msg:       "decimal values should not be equal",
			keyType:   datapb.PropertyKey_TYPE_DECIMAL,
			specValue: "3.4",
			dataValue: "1.2",
			matched:   false,
		}, {
			msg:       "string values should be equal",
			keyType:   datapb.PropertyKey_TYPE_STRING,
			specValue: "hello, world!",
			dataValue: "hello, world!",
			matched:   true,
		}, {
			msg:       "string values should not be equal",
			keyType:   datapb.PropertyKey_TYPE_STRING,
			specValue: "hello, world!",
			dataValue: "hello, galaxy!",
			matched:   false,
		}, {
			msg:       "timestamp values should be equal",
			keyType:   datapb.PropertyKey_TYPE_TIMESTAMP,
			specValue: "1612279145",
			dataValue: "1612279145",
			matched:   true,
		}, {
			msg:       "timestamp values should not be equal",
			keyType:   datapb.PropertyKey_TYPE_TIMESTAMP,
			specValue: "1111111111",
			dataValue: "2222222222",
			matched:   false,
		},
	}

	for _, c := range cases {
		t.Run(c.msg, func(t *testing.T) {
			// given
			spec, _ := oracles.NewOracleSpec(types.ExternalDataSourceSpec{
				Spec: &types.DataSourceSpec{
					Data: types.NewDataSourceDefinition(
						types.DataSourceContentTypeOracle,
					).SetOracleConfig(
						&types.DataSourceSpecConfiguration{
							Signers: []*types.Signer{
								types.CreateSignerFromString("0xCAFED00D", types.DataSignerTypePubKey),
							},
							Filters: []*types.DataSourceSpecFilter{
								{
									Key: &types.DataSourceSpecPropertyKey{
										Name: "prices.BTC.value",
										Type: c.keyType,
									},
									Conditions: []*types.DataSourceSpecCondition{
										{
											Value:    c.specValue,
											Operator: datapb.Condition_OPERATOR_EQUALS,
										},
									},
								}, {
									Key: &types.DataSourceSpecPropertyKey{
										Name: "prices.ETH.value",
										Type: datapb.PropertyKey_TYPE_INTEGER,
									},
									Conditions: []*types.DataSourceSpecCondition{
										{
											Value:    "42",
											Operator: datapb.Condition_OPERATOR_EQUALS,
										},
									},
								},
							},
						},
					),
				},
			})

			data := oracles.OracleData{
				Signers: []*types.Signer{
					types.CreateSignerFromString("0xCAFED00D", types.DataSignerTypePubKey),
				},
				Data: map[string]string{
					"prices.BTC.value": c.dataValue,
					"prices.ETH.value": "42",
				},
			}

			// when
			matched, err := spec.MatchData(data)

			// then
			require.NoError(t, err)
			assert.Equal(t, c.matched, matched)
		})
	}
}

func testOracleSpecMatchingGreaterThanPropertiesWorks(t *testing.T) {
	cases := []struct {
		msg       string
		keyType   datapb.PropertyKey_Type
		specValue string
		dataValue string
		matched   bool
	}{
		{
			msg:       "integer: data value should be greater than spec value",
			keyType:   datapb.PropertyKey_TYPE_INTEGER,
			specValue: "42",
			dataValue: "84",
			matched:   true,
		}, {
			msg:       "integer: data value should not be greater than spec value",
			keyType:   datapb.PropertyKey_TYPE_INTEGER,
			specValue: "84",
			dataValue: "42",
			matched:   false,
		}, {
			msg:       "decimal: data value should be greater than spec value",
			keyType:   datapb.PropertyKey_TYPE_DECIMAL,
			specValue: "1.2",
			dataValue: "3.4",
			matched:   true,
		}, {
			msg:       "decimal: data value should not be greater than spec value",
			keyType:   datapb.PropertyKey_TYPE_DECIMAL,
			specValue: "3.4",
			dataValue: "1.2",
			matched:   false,
		}, {
			msg:       "timestamp: data value should be greater than spec value",
			keyType:   datapb.PropertyKey_TYPE_TIMESTAMP,
			specValue: "1111111111",
			dataValue: "2222222222",
			matched:   true,
		}, {
			msg:       "timestamp: data value should not be greater than spec value",
			keyType:   datapb.PropertyKey_TYPE_TIMESTAMP,
			specValue: "2222222222",
			dataValue: "1111111111",
			matched:   false,
		},
	}

	for _, c := range cases {
		t.Run(c.msg, func(t *testing.T) {
			// given
			spec, _ := oracles.NewOracleSpec(types.ExternalDataSourceSpec{
				Spec: &types.DataSourceSpec{
					Data: types.NewDataSourceDefinition(
						types.DataSourceContentTypeOracle,
					).SetOracleConfig(
						&types.DataSourceSpecConfiguration{
							Signers: []*types.Signer{
								types.CreateSignerFromString("0xCAFED00D", types.DataSignerTypePubKey),
							},
							Filters: []*types.DataSourceSpecFilter{
								{
									Key: &types.DataSourceSpecPropertyKey{
										Name: "prices.BTC.value",
										Type: c.keyType,
									},
									Conditions: []*types.DataSourceSpecCondition{
										{
											Value:    c.specValue,
											Operator: datapb.Condition_OPERATOR_GREATER_THAN,
										},
									},
								}, {
									Key: &types.DataSourceSpecPropertyKey{
										Name: "prices.ETH.value",
										Type: datapb.PropertyKey_TYPE_INTEGER,
									},
									Conditions: []*types.DataSourceSpecCondition{
										{
											Value:    "42",
											Operator: datapb.Condition_OPERATOR_GREATER_THAN,
										},
									},
								},
							},
						},
					),
				},
			})

			data := oracles.OracleData{
				Signers: []*types.Signer{
					types.CreateSignerFromString("0xCAFED00D", types.DataSignerTypePubKey),
				},
				Data: map[string]string{
					"prices.BTC.value": c.dataValue,
					"prices.ETH.value": "84",
				},
			}

			// when
			matched, err := spec.MatchData(data)

			// then
			require.NoError(t, err)
			assert.Equal(t, c.matched, matched)
		})
	}
}

func testOracleSpecMatchingGreaterThanOrEqualPropertiesWorks(t *testing.T) {
	cases := []struct {
		msg       string
		keyType   datapb.PropertyKey_Type
		specValue string
		dataValue string
		matched   bool
	}{
		{
			msg:       "integer: data value should be equal to spec value",
			keyType:   datapb.PropertyKey_TYPE_INTEGER,
			specValue: "42",
			dataValue: "42",
			matched:   true,
		}, {
			msg:       "integer: data value should be greater than spec value",
			keyType:   datapb.PropertyKey_TYPE_INTEGER,
			specValue: "42",
			dataValue: "84",
			matched:   true,
		}, {
			msg:       "integer: data value should not be greater than spec value",
			keyType:   datapb.PropertyKey_TYPE_INTEGER,
			specValue: "84",
			dataValue: "42",
			matched:   false,
		}, {
			msg:       "decimal: data value should be equal to spec value",
			keyType:   datapb.PropertyKey_TYPE_DECIMAL,
			specValue: "1.2",
			dataValue: "1.2",
			matched:   true,
		}, {
			msg:       "decimal: data value should be greater than spec value",
			keyType:   datapb.PropertyKey_TYPE_DECIMAL,
			specValue: "1.2",
			dataValue: "3.4",
			matched:   true,
		}, {
			msg:       "decimal: data value should not be greater than spec value",
			keyType:   datapb.PropertyKey_TYPE_DECIMAL,
			specValue: "3.4",
			dataValue: "1.2",
			matched:   false,
		}, {
			msg:       "timestamp: data value should be equal to spec value",
			keyType:   datapb.PropertyKey_TYPE_TIMESTAMP,
			specValue: "1111111111",
			dataValue: "1111111111",
			matched:   true,
		}, {
			msg:       "timestamp: data value should be greater than spec value",
			keyType:   datapb.PropertyKey_TYPE_TIMESTAMP,
			specValue: "1111111111",
			dataValue: "2222222222",
			matched:   true,
		}, {
			msg:       "timestamp: data value should not be greater than spec value",
			keyType:   datapb.PropertyKey_TYPE_TIMESTAMP,
			specValue: "2222222222",
			dataValue: "1111111111",
			matched:   false,
		},
	}

	for _, c := range cases {
		t.Run(c.msg, func(t *testing.T) {
			// given
			spec, _ := oracles.NewOracleSpec(types.ExternalDataSourceSpec{
				Spec: &types.DataSourceSpec{
					Data: types.NewDataSourceDefinition(
						types.DataSourceContentTypeOracle,
					).SetOracleConfig(
						&types.DataSourceSpecConfiguration{
							Signers: []*types.Signer{
								types.CreateSignerFromString("0xCAFED00D", types.DataSignerTypePubKey),
							},
							Filters: []*types.DataSourceSpecFilter{
								{
									Key: &types.DataSourceSpecPropertyKey{
										Name: "prices.BTC.value",
										Type: c.keyType,
									},
									Conditions: []*types.DataSourceSpecCondition{
										{
											Value:    c.specValue,
											Operator: datapb.Condition_OPERATOR_GREATER_THAN_OR_EQUAL,
										},
									},
								}, {
									Key: &types.DataSourceSpecPropertyKey{
										Name: "prices.ETH.value",
										Type: datapb.PropertyKey_TYPE_INTEGER,
									},
									Conditions: []*types.DataSourceSpecCondition{
										{
											Value:    "42",
											Operator: datapb.Condition_OPERATOR_GREATER_THAN_OR_EQUAL,
										},
									},
								},
							},
						},
					),
				},
			})

			data := oracles.OracleData{
				Signers: []*types.Signer{
					types.CreateSignerFromString("0xCAFED00D", types.DataSignerTypePubKey),
				},
				Data: map[string]string{
					"prices.BTC.value": c.dataValue,
					"prices.ETH.value": "42",
				},
			}

			// when
			matched, err := spec.MatchData(data)

			// then
			require.NoError(t, err)
			assert.Equal(t, c.matched, matched)
		})
	}
}

func testOracleSpecMatchingLessThanPropertiesSucceedsOnlyForNonTimestamp(t *testing.T) {
	cases := []struct {
		msg       string
		keyType   datapb.PropertyKey_Type
		specValue string
		dataValue string
		matched   bool
	}{
		{
			msg:       "integer: data value should be less than spec value",
			keyType:   datapb.PropertyKey_TYPE_INTEGER,
			specValue: "84",
			dataValue: "42",
			matched:   true,
		}, {
			msg:       "integer: data value should not be less than spec value",
			keyType:   datapb.PropertyKey_TYPE_INTEGER,
			specValue: "42",
			dataValue: "84",
			matched:   false,
		}, {
			msg:       "decimal: data value should be less than spec value",
			keyType:   datapb.PropertyKey_TYPE_DECIMAL,
			specValue: "3.4",
			dataValue: "1.2",
			matched:   true,
		}, {
			msg:       "decimal: data value should not be less than spec value",
			keyType:   datapb.PropertyKey_TYPE_DECIMAL,
			specValue: "1.2",
			dataValue: "3.4",
			matched:   false,
		}, {
			msg:       "timestamp: data value should be less than spec value",
			keyType:   datapb.PropertyKey_TYPE_TIMESTAMP,
			specValue: "2222222222",
			dataValue: "1111111111",
			matched:   true,
		}, {
			msg:       "timestamp: data value should not be less than spec value",
			keyType:   datapb.PropertyKey_TYPE_TIMESTAMP,
			specValue: "1111111111",
			dataValue: "2222222222",
			matched:   false,
		},
	}

	for _, c := range cases {
		t.Run(c.msg, func(t *testing.T) {
			// given
			spec, err := oracles.NewOracleSpec(types.ExternalDataSourceSpec{
				Spec: &types.DataSourceSpec{
					Data: types.NewDataSourceDefinition(
						types.DataSourceContentTypeOracle,
					).SetOracleConfig(
						&types.DataSourceSpecConfiguration{
							Signers: []*types.Signer{
								types.CreateSignerFromString("0xCAFED00D", types.DataSignerTypePubKey),
							},
							Filters: []*types.DataSourceSpecFilter{
								{
									Key: &types.DataSourceSpecPropertyKey{
										Name: "prices.BTC.value",
										Type: c.keyType,
									},
									Conditions: []*types.DataSourceSpecCondition{
										{
											Value:    c.specValue,
											Operator: datapb.Condition_OPERATOR_LESS_THAN,
										},
									},
								}, {
									Key: &types.DataSourceSpecPropertyKey{
										Name: "prices.ETH.value",
										Type: datapb.PropertyKey_TYPE_INTEGER,
									},
									Conditions: []*types.DataSourceSpecCondition{
										{
											Value:    "42",
											Operator: datapb.Condition_OPERATOR_LESS_THAN,
										},
									},
								},
							},
						},
					),
				},
			})

			if c.keyType == datapb.PropertyKey_TYPE_TIMESTAMP {
				assert.Error(t, err)
				assert.EqualError(t, err, types.ErrDataSourceSpecHasInvalidTimeCondition.Error())
				assert.Nil(t, spec)
			} else {
				data := oracles.OracleData{
					Signers: []*types.Signer{
						types.CreateSignerFromString("0xCAFED00D", types.DataSignerTypePubKey),
					},
					Data: map[string]string{
						"prices.BTC.value": c.dataValue,
						"prices.ETH.value": "21",
					},
				}

				// when
				matched, err := spec.MatchData(data)

				// then
				require.NoError(t, err)
				assert.Equal(t, c.matched, matched)
			}
		})
	}
}

func testOracleSpecMatchingLessThanOrEqualPropertiesSucceedsOnlyForNonTimestamp(t *testing.T) {
	cases := []struct {
		msg       string
		keyType   datapb.PropertyKey_Type
		specValue string
		dataValue string
		matched   bool
	}{
		{
			msg:       "integer: data value should be equal to spec value",
			keyType:   datapb.PropertyKey_TYPE_INTEGER,
			specValue: "42",
			dataValue: "42",
			matched:   true,
		}, {
			msg:       "integer: data value should be less than spec value",
			keyType:   datapb.PropertyKey_TYPE_INTEGER,
			specValue: "84",
			dataValue: "42",
			matched:   true,
		}, {
			msg:       "integer: data value should not be less than spec value",
			keyType:   datapb.PropertyKey_TYPE_INTEGER,
			specValue: "42",
			dataValue: "84",
			matched:   false,
		}, {
			msg:       "decimal: data value should be equal to spec value",
			keyType:   datapb.PropertyKey_TYPE_DECIMAL,
			specValue: "1.2",
			dataValue: "1.2",
			matched:   true,
		}, {
			msg:       "decimal: data value should be less than spec value",
			keyType:   datapb.PropertyKey_TYPE_DECIMAL,
			specValue: "3.4",
			dataValue: "1.2",
			matched:   true,
		}, {
			msg:       "decimal: data value should not be less than spec value",
			keyType:   datapb.PropertyKey_TYPE_DECIMAL,
			specValue: "1.2",
			dataValue: "3.4",
			matched:   false,
		}, {
			msg:       "timestamp: data value should be equal to spec value",
			keyType:   datapb.PropertyKey_TYPE_TIMESTAMP,
			specValue: "1111111111",
			dataValue: "1111111111",
			matched:   true,
		}, {
			msg:       "timestamp: data value should be less than spec value",
			keyType:   datapb.PropertyKey_TYPE_TIMESTAMP,
			specValue: "2222222222",
			dataValue: "1111111111",
			matched:   true,
		}, {
			msg:       "timestamp: data value should not be less than spec value",
			keyType:   datapb.PropertyKey_TYPE_TIMESTAMP,
			specValue: "1111111111",
			dataValue: "2222222222",
			matched:   false,
		},
	}

	for _, c := range cases {
		t.Run(c.msg, func(t *testing.T) {
			// given
			spec, err := oracles.NewOracleSpec(types.ExternalDataSourceSpec{
				Spec: &types.DataSourceSpec{
					Data: types.NewDataSourceDefinition(
						types.DataSourceContentTypeOracle,
					).SetOracleConfig(
						&types.DataSourceSpecConfiguration{
							Signers: []*types.Signer{
								types.CreateSignerFromString("0xCAFED00D", types.DataSignerTypePubKey),
							},
							Filters: []*types.DataSourceSpecFilter{
								{
									Key: &types.DataSourceSpecPropertyKey{
										Name: "prices.BTC.value",
										Type: c.keyType,
									},
									Conditions: []*types.DataSourceSpecCondition{
										{
											Value:    c.specValue,
											Operator: datapb.Condition_OPERATOR_LESS_THAN_OR_EQUAL,
										},
									},
								}, {
									Key: &types.DataSourceSpecPropertyKey{
										Name: "prices.ETH.value",
										Type: datapb.PropertyKey_TYPE_INTEGER,
									},
									Conditions: []*types.DataSourceSpecCondition{
										{
											Value:    "42",
											Operator: datapb.Condition_OPERATOR_LESS_THAN_OR_EQUAL,
										},
									},
								},
							},
						},
					),
				},
			})

			if c.keyType == datapb.PropertyKey_TYPE_TIMESTAMP {
				assert.Error(t, err)
				assert.EqualError(t, err, types.ErrDataSourceSpecHasInvalidTimeCondition.Error())
				assert.Nil(t, spec)
			} else {
				data := oracles.OracleData{
					Signers: []*types.Signer{
						types.CreateSignerFromString("0xCAFED00D", types.DataSignerTypePubKey),
					},
					Data: map[string]string{
						"prices.BTC.value": c.dataValue,
						"prices.ETH.value": "42",
					},
				}

				// when
				matched, err := spec.MatchData(data)

				// then
				require.NoError(t, err)
				assert.Equal(t, c.matched, matched)
			}
		})
	}
}

func testOracleSpecMatchingPropertiesPresenceSucceeds(t *testing.T) {
	cases := []struct {
		msg     string
		keyType datapb.PropertyKey_Type
	}{
		{
			msg:     "integer values is present",
			keyType: datapb.PropertyKey_TYPE_INTEGER,
		}, {
			msg:     "boolean values is present",
			keyType: datapb.PropertyKey_TYPE_BOOLEAN,
		}, {
			msg:     "decimal values is present",
			keyType: datapb.PropertyKey_TYPE_DECIMAL,
		}, {
			msg:     "string values is present",
			keyType: datapb.PropertyKey_TYPE_STRING,
		}, {
			msg:     "timestamp values is present",
			keyType: datapb.PropertyKey_TYPE_TIMESTAMP,
		},
	}

	for _, c := range cases {
		t.Run(c.msg, func(t *testing.T) {
			// given
			spec, _ := oracles.NewOracleSpec(types.ExternalDataSourceSpec{
				Spec: &types.DataSourceSpec{
					Data: types.NewDataSourceDefinition(
						types.DataSourceContentTypeOracle,
					).SetOracleConfig(
						&types.DataSourceSpecConfiguration{
							Signers: []*types.Signer{
								types.CreateSignerFromString("0xCAFED00D", types.DataSignerTypePubKey),
							},
							Filters: []*types.DataSourceSpecFilter{
								{
									Key: &types.DataSourceSpecPropertyKey{
										Name: "prices.BTC.value",
										Type: c.keyType,
									},
									Conditions: []*types.DataSourceSpecCondition{},
								},
								{
									Key: &types.DataSourceSpecPropertyKey{
										Name: "prices.ETH.value",
										Type: datapb.PropertyKey_TYPE_INTEGER,
									},
									Conditions: []*types.DataSourceSpecCondition{},
								},
							},
						},
					),
				},
			})

			data := oracles.OracleData{
				Signers: []*types.Signer{
					types.CreateSignerFromString("0xCAFED00D", types.DataSignerTypePubKey),
				},
				Data: map[string]string{
					"prices.BTC.value": "42",
					"prices.ETH.value": "42",
				},
			}

			// when
			matched, err := spec.MatchData(data)

			// then
			require.NoError(t, err)
			assert.True(t, matched)
		})
	}
}

func testOracleSpecMatchingPropertiesPresenceFails(t *testing.T) {
	cases := []struct {
		msg     string
		keyType datapb.PropertyKey_Type
	}{
		{
			msg:     "integer values is absent",
			keyType: datapb.PropertyKey_TYPE_INTEGER,
		}, {
			msg:     "boolean values is absent",
			keyType: datapb.PropertyKey_TYPE_BOOLEAN,
		}, {
			msg:     "decimal values is absent",
			keyType: datapb.PropertyKey_TYPE_DECIMAL,
		}, {
			msg:     "string values is absent",
			keyType: datapb.PropertyKey_TYPE_STRING,
		}, {
			msg:     "timestamp values is absent",
			keyType: datapb.PropertyKey_TYPE_TIMESTAMP,
		},
	}

	for _, c := range cases {
		t.Run(c.msg, func(t *testing.T) {
			// given
			spec, _ := oracles.NewOracleSpec(types.ExternalDataSourceSpec{
				Spec: &types.DataSourceSpec{
					Data: types.NewDataSourceDefinition(
						types.DataSourceContentTypeOracle,
					).SetOracleConfig(
						&types.DataSourceSpecConfiguration{
							Signers: []*types.Signer{
								types.CreateSignerFromString("0xCAFED00D", types.DataSignerTypePubKey),
							},
							Filters: []*types.DataSourceSpecFilter{
								{
									Key: &types.DataSourceSpecPropertyKey{
										Name: "prices.BTC.value",
										Type: c.keyType,
									},
									Conditions: []*types.DataSourceSpecCondition{},
								},
								{
									Key: &types.DataSourceSpecPropertyKey{
										Name: "prices.ETH.value",
										Type: datapb.PropertyKey_TYPE_INTEGER,
									},
									Conditions: []*types.DataSourceSpecCondition{},
								},
							},
						},
					),
				},
			})

			data := oracles.OracleData{
				Signers: []*types.Signer{
					types.CreateSignerFromString("0xCAFED00D", types.DataSignerTypePubKey),
				},
				Data: map[string]string{
					"prices.ETH.value": "42",
				},
			}

			// when
			matched, err := spec.MatchData(data)

			// then
			require.NoError(t, err)
			assert.False(t, matched)
		})
	}
}

func testOracleSpecMatchingWithInconvertibleTypeFails(t *testing.T) {
	cases := []struct {
		msg       string
		keyType   datapb.PropertyKey_Type
		specValue string
		dataValue string
	}{
		{
			msg:       "not an integer",
			keyType:   datapb.PropertyKey_TYPE_INTEGER,
			specValue: "42",
			dataValue: "not an integer",
		}, {
			msg:       "not a boolean",
			keyType:   datapb.PropertyKey_TYPE_BOOLEAN,
			specValue: "true",
			dataValue: "not a boolean",
		}, {
			msg:       "not a decimal",
			keyType:   datapb.PropertyKey_TYPE_DECIMAL,
			specValue: "1.2",
			dataValue: "not a decimal",
		}, {
			msg:       "not a timestamp",
			keyType:   datapb.PropertyKey_TYPE_TIMESTAMP,
			specValue: "1111111111",
			dataValue: "not a timestamp",
		},
	}

	for _, c := range cases {
		t.Run(c.msg, func(t *testing.T) {
			// given
			spec, _ := oracles.NewOracleSpec(types.ExternalDataSourceSpec{
				Spec: &types.DataSourceSpec{
					Data: types.NewDataSourceDefinition(
						types.DataSourceContentTypeOracle,
					).SetOracleConfig(
						&types.DataSourceSpecConfiguration{
							Signers: []*types.Signer{
								types.CreateSignerFromString("0xCAFED00D", types.DataSignerTypePubKey),
							},
							Filters: []*types.DataSourceSpecFilter{
								{
									Key: &types.DataSourceSpecPropertyKey{
										Name: "prices.BTC.value",
										Type: c.keyType,
									},
									Conditions: []*types.DataSourceSpecCondition{
										{
											Value:    c.specValue,
											Operator: datapb.Condition_OPERATOR_EQUALS,
										},
									},
								},
							},
						},
					),
				},
			})

			data := oracles.OracleData{
				Signers: []*types.Signer{
					types.CreateSignerFromString("0xCAFED00D", types.DataSignerTypePubKey),
				},
				Data: map[string]string{
					"prices.BTC.value": c.dataValue,
				},
			}

			// when
			matched, err := spec.MatchData(data)

			// then
			require.Error(t, err)
			assert.False(t, matched)
		})
	}
}

func testOracleSpecVerifyingBindingWorks(t *testing.T) {
	cases := []struct {
		msg              string
		declaredType     datapb.PropertyKey_Type
		declaredProperty string
		decimalPlaces    uint64
		boundType        datapb.PropertyKey_Type
		boundProperty    string
		expectedError    error
	}{
		{
			msg:              "same integer properties can be bound",
			declaredType:     datapb.PropertyKey_TYPE_INTEGER,
			declaredProperty: "price.ETH.value",
			decimalPlaces:    7,
			boundType:        datapb.PropertyKey_TYPE_INTEGER,
			boundProperty:    "price.ETH.value",
			expectedError:    nil,
		}, {
			msg:              "different integer properties cannot be bound",
			declaredType:     datapb.PropertyKey_TYPE_INTEGER,
			declaredProperty: "price.USD.value",
			decimalPlaces:    19,
			boundType:        datapb.PropertyKey_TYPE_INTEGER,
			boundProperty:    "price.BTC.value",
			expectedError:    errors.New("bound property \"price.BTC.value\" not filtered by oracle spec"),
		}, {
			msg:              "same integer properties can be bound",
			declaredType:     datapb.PropertyKey_TYPE_BOOLEAN,
			declaredProperty: "price.ETH.value",
			decimalPlaces:    2,
			boundType:        datapb.PropertyKey_TYPE_BOOLEAN,
			boundProperty:    "price.ETH.value",
			expectedError:    nil,
		}, {
			msg:              "different integer properties can be bound",
			declaredType:     datapb.PropertyKey_TYPE_BOOLEAN,
			decimalPlaces:    4,
			declaredProperty: "price.USD.value",
			boundType:        datapb.PropertyKey_TYPE_BOOLEAN,
			boundProperty:    "price.BTC.value",
			expectedError:    errors.New("bound property \"price.BTC.value\" not filtered by oracle spec"),
		}, {
			msg:              "same integer properties can be bound",
			declaredType:     datapb.PropertyKey_TYPE_DECIMAL,
			decimalPlaces:    0,
			declaredProperty: "price.ETH.value",
			boundType:        datapb.PropertyKey_TYPE_DECIMAL,
			boundProperty:    "price.ETH.value",
			expectedError:    nil,
		}, {
			msg:              "different integer properties can be bound",
			declaredType:     datapb.PropertyKey_TYPE_DECIMAL,
			declaredProperty: "price.USD.value",
			boundType:        datapb.PropertyKey_TYPE_DECIMAL,
			boundProperty:    "price.BTC.value",
			expectedError:    errors.New("bound property \"price.BTC.value\" not filtered by oracle spec"),
		}, {
			msg:              "same integer properties can be bound",
			declaredType:     datapb.PropertyKey_TYPE_STRING,
			declaredProperty: "price.ETH.value",
			boundType:        datapb.PropertyKey_TYPE_STRING,
			boundProperty:    "price.ETH.value",
			expectedError:    nil,
		}, {
			msg:              "different integer properties can be bound",
			declaredType:     datapb.PropertyKey_TYPE_STRING,
			declaredProperty: "price.USD.value",
			boundType:        datapb.PropertyKey_TYPE_STRING,
			boundProperty:    "price.BTC.value",
			expectedError:    errors.New("bound property \"price.BTC.value\" not filtered by oracle spec"),
		}, {
			msg:              "same integer properties can be bound",
			declaredType:     datapb.PropertyKey_TYPE_TIMESTAMP,
			declaredProperty: "price.ETH.value",
			boundType:        datapb.PropertyKey_TYPE_TIMESTAMP,
			boundProperty:    "price.ETH.value",
			expectedError:    nil,
		}, {
			msg:              "different integer properties can be bound",
			declaredType:     datapb.PropertyKey_TYPE_TIMESTAMP,
			declaredProperty: "price.USD.value",
			boundType:        datapb.PropertyKey_TYPE_TIMESTAMP,
			boundProperty:    "price.BTC.value",
			expectedError:    errors.New("bound property \"price.BTC.value\" not filtered by oracle spec"),
		}, {
			msg:              "same properties but different type can't be bound",
			declaredType:     datapb.PropertyKey_TYPE_TIMESTAMP,
			declaredProperty: "price.USD.value",
			boundType:        datapb.PropertyKey_TYPE_STRING,
			boundProperty:    "price.USD.value",
			expectedError:    errors.New("bound type \"TYPE_STRING\" doesn't match filtered property type \"TYPE_TIMESTAMP\""),
		},
	}

	for _, c := range cases {
		t.Run(c.msg, func(t *testing.T) {
			// given
			spec, _ := oracles.NewOracleSpec(types.ExternalDataSourceSpec{
				Spec: &types.DataSourceSpec{
					Data: types.NewDataSourceDefinition(
						types.DataSourceContentTypeOracle,
					).SetOracleConfig(
						&types.DataSourceSpecConfiguration{
							Signers: []*types.Signer{
								types.CreateSignerFromString("0xCAFED00D", types.DataSignerTypePubKey),
							},
							Filters: []*types.DataSourceSpecFilter{
								{
									Key: &types.DataSourceSpecPropertyKey{
										Name:                c.declaredProperty,
										Type:                c.declaredType,
										NumberDecimalPlaces: &c.decimalPlaces,
									},
									Conditions: []*types.DataSourceSpecCondition{},
								},
							},
						},
					),
				},
			})

			// when
			err := spec.EnsureBoundableProperty(c.boundProperty, c.boundType)

			// then
			assert.Equal(t, c.expectedError, err)
		})
	}
}
