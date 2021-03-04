package oracles_test

import (
	"testing"

	"code.vegaprotocol.io/vega/oracles"
	oraclespb "code.vegaprotocol.io/vega/proto/oracles/v1"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOracleSpec(t *testing.T) {
	t.Run("Creating without required public keys fails", testOracleSpecCreatingWithoutPubKeysFails)
	t.Run("Creating without filters fails", testOracleSpecCreatingWithoutFiltersFails)
	t.Run("Creating with split filters with same type works", testOracleSpecCreatingWithSplitFiltersWithSameTypeWorks)
	t.Run("Creating with split filters with different type fails", testOracleSpecCreatingWithSplitFiltersWithDifferentTypeWorks)
	t.Run("Creating with filters with inconvertible type fails", testOracleSpecCreatingWithFiltersWithInconvertibleTypeFails)
	t.Run("Matching with unauthorized public keys fails", testOracleSpecMatchingUnauthorizedPubKeysFails)
	t.Run("Matching with authorized public keys succeeds", testOracleSpecMatchingAuthorizedPubKeysSucceeds)
	t.Run("Matching with equal properties works", testOracleSpecMatchingEqualPropertiesWorks)
	t.Run("Matching with greater than properties works", testOracleSpecMatchingGreaterThanPropertiesWorks)
	t.Run("Matching with greater than or equal properties works", testOracleSpecMatchingGreaterThanOrEqualPropertiesWorks)
	t.Run("Matching with less than properties works", testOracleSpecMatchingLessThanPropertiesWorks)
	t.Run("Matching with less than or equal properties works", testOracleSpecMatchingLessThanOrEqualPropertiesWorks)
	t.Run("Matching presence of present properties succeeds", testOracleSpecMatchingPropertiesPresenceSucceeds)
	t.Run("Matching presence of missing properties fails", testOracleSpecMatchingPropertiesPresenceFails)
	t.Run("Matching with inconvertible type fails", testOracleSpecMatchingWithInconvertibleTypeFails)
	t.Run("Verifying binding of property works", testOracleSpecVerifyingBindingWorks)
}

func testOracleSpecCreatingWithoutPubKeysFails(t *testing.T) {
	// given
	spec := oraclespb.OracleSpec{
		PubKeys: []string{},
		Filters: []*oraclespb.Filter{
			{
				Key: &oraclespb.PropertyKey{
					Name: "price",
					Type: oraclespb.PropertyKey_TYPE_INTEGER,
				},
				Conditions: []*oraclespb.Condition{},
			},
		},
	}

	// when
	oracleSpec, err := oracles.NewOracleSpec(spec)

	// then
	require.Error(t, err)
	assert.Equal(t, "public keys are required", err.Error())
	assert.Nil(t, oracleSpec)
}

func testOracleSpecCreatingWithoutFiltersFails(t *testing.T) {
	// given
	spec := oraclespb.OracleSpec{
		PubKeys: []string{
			"0xCAFED00D",
		},
		Filters: []*oraclespb.Filter{},
	}

	// when
	oracleSpec, err := oracles.NewOracleSpec(spec)

	// then
	require.Error(t, err)
	assert.Equal(t, "at least one filter is required", err.Error())
	assert.Nil(t, oracleSpec)
}

func testOracleSpecCreatingWithSplitFiltersWithSameTypeWorks(t *testing.T) {
	// given
	spec, _ := oracles.NewOracleSpec(oraclespb.OracleSpec{
		PubKeys: []string{
			"0xDEADBEEF",
			"0xCAFED00D",
		},
		Filters: []*oraclespb.Filter{
			{
				Key: &oraclespb.PropertyKey{
					Name: "prices.BTC.value",
					Type: oraclespb.PropertyKey_TYPE_INTEGER,
				},
				Conditions: []*oraclespb.Condition{
					{
						Value:    "42",
						Operator: oraclespb.Condition_OPERATOR_GREATER_THAN,
					},
				},
			}, {
				Key: &oraclespb.PropertyKey{
					Name: "prices.BTC.value",
					Type: oraclespb.PropertyKey_TYPE_INTEGER,
				},
				Conditions: []*oraclespb.Condition{
					{
						Value:    "84",
						Operator: oraclespb.Condition_OPERATOR_LESS_THAN,
					},
				},
			},
		},
	})

	matchedData := oracles.OracleData{
		PubKeys: []string{
			"0xDEADBEEF",
			"0xCAFED00D",
		},
		Data: map[string]string{
			"prices.BTC.value": "50",
		},
	}

	unmatchedData := oracles.OracleData{
		PubKeys: []string{
			"0xDEADBEEF",
			"0xCAFED00D",
		},
		Data: map[string]string{
			"prices.BTC.value": "100",
		},
	}

	// when
	matched, err := spec.MatchData(matchedData)

	// then
	require.NoError(t, err)
	assert.True(t, matched)

	// when
	matched, err = spec.MatchData(unmatchedData)

	// then
	require.NoError(t, err)
	assert.False(t, matched)
}

func testOracleSpecCreatingWithSplitFiltersWithDifferentTypeWorks(t *testing.T) {
	// given
	originalSpec := oraclespb.OracleSpec{
		PubKeys: []string{
			"0xDEADBEEF",
			"0xCAFED00D",
		},
		Filters: []*oraclespb.Filter{
			{
				Key: &oraclespb.PropertyKey{
					Name: "prices.BTC.value",
					Type: oraclespb.PropertyKey_TYPE_INTEGER,
				},
				Conditions: []*oraclespb.Condition{
					{
						Value:    "42",
						Operator: oraclespb.Condition_OPERATOR_GREATER_THAN,
					},
				},
			}, {
				Key: &oraclespb.PropertyKey{
					Name: "prices.BTC.value",
					Type: oraclespb.PropertyKey_TYPE_TIMESTAMP,
				},
				Conditions: []*oraclespb.Condition{
					{
						Value:    "84",
						Operator: oraclespb.Condition_OPERATOR_LESS_THAN,
					},
				},
			},
		},
	}

	// when
	spec, err := oracles.NewOracleSpec(originalSpec)

	// then
	require.Error(t, err)
	assert.Equal(t, "cannot redeclared property prices.BTC.value with different type, first TYPE_INTEGER then TYPE_TIMESTAMP", err.Error())
	assert.Nil(t, spec)
}

func testOracleSpecCreatingWithFiltersWithInconvertibleTypeFails(t *testing.T) {
	cases := []struct {
		msg   string
		typ   oraclespb.PropertyKey_Type
		value string
	}{
		{
			msg:   "not an integer",
			typ:   oraclespb.PropertyKey_TYPE_INTEGER,
			value: "not an integer",
		}, {
			msg:   "not a boolean",
			typ:   oraclespb.PropertyKey_TYPE_BOOLEAN,
			value: "42",
		}, {
			msg:   "not a decimal",
			typ:   oraclespb.PropertyKey_TYPE_DECIMAL,
			value: "not a decimal",
		}, {
			msg:   "not a timestamp",
			typ:   oraclespb.PropertyKey_TYPE_TIMESTAMP,
			value: "not a timestamp",
		},
	}

	for _, c := range cases {
		t.Run(c.msg, func(t *testing.T) {
			// given
			originalSpec := oraclespb.OracleSpec{
				PubKeys: []string{
					"0xCAFED00D",
				},
				Filters: []*oraclespb.Filter{
					{
						Key: &oraclespb.PropertyKey{
							Name: "prices.BTC.value",
							Type: c.typ,
						},
						Conditions: []*oraclespb.Condition{
							{
								Value:    c.value,
								Operator: oraclespb.Condition_OPERATOR_EQUALS,
							},
						},
					},
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
	spec, _ := oracles.NewOracleSpec(oraclespb.OracleSpec{
		PubKeys: []string{
			"0xDEADBEEF",
			"0xCAFED00D",
		},
		Filters: []*oraclespb.Filter{
			{
				Key: &oraclespb.PropertyKey{
					Name: "prices.BTC.value",
					Type: oraclespb.PropertyKey_TYPE_INTEGER,
				},
				Conditions: []*oraclespb.Condition{
					{
						Value:    "42",
						Operator: oraclespb.Condition_OPERATOR_EQUALS,
					},
				},
			},
		},
	})

	data := oracles.OracleData{
		PubKeys: []string{
			"0xDEADBEEF",
			"0xBADDCAFE",
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
	spec, _ := oracles.NewOracleSpec(oraclespb.OracleSpec{
		PubKeys: []string{
			"0xDEADBEEF",
			"0xCAFED00D",
			"0xBADDCAFE",
		},
		Filters: []*oraclespb.Filter{
			{
				Key: &oraclespb.PropertyKey{
					Name: "prices.BTC.value",
					Type: oraclespb.PropertyKey_TYPE_INTEGER,
				},
				Conditions: []*oraclespb.Condition{
					{
						Value:    "42",
						Operator: oraclespb.Condition_OPERATOR_EQUALS,
					},
				},
			},
		},
	})

	data := oracles.OracleData{
		PubKeys: []string{
			"0xDEADBEEF",
			"0xBADDCAFE",
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
		keyType   oraclespb.PropertyKey_Type
		specValue string
		dataValue string
		matched   bool
	}{
		{
			msg:       "integer values should be equal",
			keyType:   oraclespb.PropertyKey_TYPE_INTEGER,
			specValue: "42",
			dataValue: "42",
			matched:   true,
		}, {
			msg:       "integer values should not be equal",
			keyType:   oraclespb.PropertyKey_TYPE_INTEGER,
			specValue: "84",
			dataValue: "42",
			matched:   false,
		}, {
			msg:       "boolean values should be equal",
			keyType:   oraclespb.PropertyKey_TYPE_BOOLEAN,
			specValue: "true",
			dataValue: "true",
			matched:   true,
		}, {
			msg:       "boolean values should not be equal",
			keyType:   oraclespb.PropertyKey_TYPE_BOOLEAN,
			specValue: "true",
			dataValue: "false",
			matched:   false,
		}, {
			msg:       "decimal values should be equal",
			keyType:   oraclespb.PropertyKey_TYPE_DECIMAL,
			specValue: "1.2",
			dataValue: "1.2",
			matched:   true,
		}, {
			msg:       "decimal values should not be equal",
			keyType:   oraclespb.PropertyKey_TYPE_DECIMAL,
			specValue: "3.4",
			dataValue: "1.2",
			matched:   false,
		}, {
			msg:       "string values should be equal",
			keyType:   oraclespb.PropertyKey_TYPE_STRING,
			specValue: "hello, world!",
			dataValue: "hello, world!",
			matched:   true,
		}, {
			msg:       "string values should not be equal",
			keyType:   oraclespb.PropertyKey_TYPE_STRING,
			specValue: "hello, world!",
			dataValue: "hello, galaxy!",
			matched:   false,
		}, {
			msg:       "timestamp values should be equal",
			keyType:   oraclespb.PropertyKey_TYPE_TIMESTAMP,
			specValue: "1612279145",
			dataValue: "1612279145",
			matched:   true,
		}, {
			msg:       "timestamp values should not be equal",
			keyType:   oraclespb.PropertyKey_TYPE_TIMESTAMP,
			specValue: "1111111111",
			dataValue: "2222222222",
			matched:   false,
		},
	}

	for _, c := range cases {
		t.Run(c.msg, func(t *testing.T) {

			// given
			spec, _ := oracles.NewOracleSpec(oraclespb.OracleSpec{
				PubKeys: []string{
					"0xCAFED00D",
				},
				Filters: []*oraclespb.Filter{
					{
						Key: &oraclespb.PropertyKey{
							Name: "prices.BTC.value",
							Type: c.keyType,
						},
						Conditions: []*oraclespb.Condition{
							{
								Value:    c.specValue,
								Operator: oraclespb.Condition_OPERATOR_EQUALS,
							},
						},
					}, {
						Key: &oraclespb.PropertyKey{
							Name: "prices.ETH.value",
							Type: oraclespb.PropertyKey_TYPE_INTEGER,
						},
						Conditions: []*oraclespb.Condition{
							{
								Value:    "42",
								Operator: oraclespb.Condition_OPERATOR_EQUALS,
							},
						},
					},
				},
			})

			data := oracles.OracleData{
				PubKeys: []string{
					"0xCAFED00D",
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
		keyType   oraclespb.PropertyKey_Type
		specValue string
		dataValue string
		matched   bool
	}{
		{
			msg:       "integer: data value should be greater than spec value",
			keyType:   oraclespb.PropertyKey_TYPE_INTEGER,
			specValue: "42",
			dataValue: "84",
			matched:   true,
		}, {
			msg:       "integer: data value should not be greater than spec value",
			keyType:   oraclespb.PropertyKey_TYPE_INTEGER,
			specValue: "84",
			dataValue: "42",
			matched:   false,
		}, {
			msg:       "decimal: data value should be greater than spec value",
			keyType:   oraclespb.PropertyKey_TYPE_DECIMAL,
			specValue: "1.2",
			dataValue: "3.4",
			matched:   true,
		}, {
			msg:       "decimal: data value should not be greater than spec value",
			keyType:   oraclespb.PropertyKey_TYPE_DECIMAL,
			specValue: "3.4",
			dataValue: "1.2",
			matched:   false,
		}, {
			msg:       "timestamp: data value should be greater than spec value",
			keyType:   oraclespb.PropertyKey_TYPE_TIMESTAMP,
			specValue: "1111111111",
			dataValue: "2222222222",
			matched:   true,
		}, {
			msg:       "timestamp: data value should not be greater than spec value",
			keyType:   oraclespb.PropertyKey_TYPE_TIMESTAMP,
			specValue: "2222222222",
			dataValue: "1111111111",
			matched:   false,
		},
	}

	for _, c := range cases {
		t.Run(c.msg, func(t *testing.T) {

			// given
			spec, _ := oracles.NewOracleSpec(oraclespb.OracleSpec{
				PubKeys: []string{
					"0xCAFED00D",
				},
				Filters: []*oraclespb.Filter{
					{
						Key: &oraclespb.PropertyKey{
							Name: "prices.BTC.value",
							Type: c.keyType,
						},
						Conditions: []*oraclespb.Condition{
							{
								Value:    c.specValue,
								Operator: oraclespb.Condition_OPERATOR_GREATER_THAN,
							},
						},
					}, {
						Key: &oraclespb.PropertyKey{
							Name: "prices.ETH.value",
							Type: oraclespb.PropertyKey_TYPE_INTEGER,
						},
						Conditions: []*oraclespb.Condition{
							{
								Value:    "42",
								Operator: oraclespb.Condition_OPERATOR_GREATER_THAN,
							},
						},
					},
				},
			})

			data := oracles.OracleData{
				PubKeys: []string{
					"0xCAFED00D",
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
		keyType   oraclespb.PropertyKey_Type
		specValue string
		dataValue string
		matched   bool
	}{
		{
			msg:       "integer: data value should be equal to spec value",
			keyType:   oraclespb.PropertyKey_TYPE_INTEGER,
			specValue: "42",
			dataValue: "42",
			matched:   true,
		}, {
			msg:       "integer: data value should be greater than spec value",
			keyType:   oraclespb.PropertyKey_TYPE_INTEGER,
			specValue: "42",
			dataValue: "84",
			matched:   true,
		}, {
			msg:       "integer: data value should not be greater than spec value",
			keyType:   oraclespb.PropertyKey_TYPE_INTEGER,
			specValue: "84",
			dataValue: "42",
			matched:   false,
		}, {
			msg:       "decimal: data value should be equal to spec value",
			keyType:   oraclespb.PropertyKey_TYPE_DECIMAL,
			specValue: "1.2",
			dataValue: "1.2",
			matched:   true,
		}, {
			msg:       "decimal: data value should be greater than spec value",
			keyType:   oraclespb.PropertyKey_TYPE_DECIMAL,
			specValue: "1.2",
			dataValue: "3.4",
			matched:   true,
		}, {
			msg:       "decimal: data value should not be greater than spec value",
			keyType:   oraclespb.PropertyKey_TYPE_DECIMAL,
			specValue: "3.4",
			dataValue: "1.2",
			matched:   false,
		}, {
			msg:       "timestamp: data value should be equal to spec value",
			keyType:   oraclespb.PropertyKey_TYPE_TIMESTAMP,
			specValue: "1111111111",
			dataValue: "1111111111",
			matched:   true,
		}, {
			msg:       "timestamp: data value should be greater than spec value",
			keyType:   oraclespb.PropertyKey_TYPE_TIMESTAMP,
			specValue: "1111111111",
			dataValue: "2222222222",
			matched:   true,
		}, {
			msg:       "timestamp: data value should not be greater than spec value",
			keyType:   oraclespb.PropertyKey_TYPE_TIMESTAMP,
			specValue: "2222222222",
			dataValue: "1111111111",
			matched:   false,
		},
	}

	for _, c := range cases {
		t.Run(c.msg, func(t *testing.T) {

			// given
			spec, _ := oracles.NewOracleSpec(oraclespb.OracleSpec{
				PubKeys: []string{
					"0xCAFED00D",
				},
				Filters: []*oraclespb.Filter{
					{
						Key: &oraclespb.PropertyKey{
							Name: "prices.BTC.value",
							Type: c.keyType,
						},
						Conditions: []*oraclespb.Condition{
							{
								Value:    c.specValue,
								Operator: oraclespb.Condition_OPERATOR_GREATER_THAN_OR_EQUAL,
							},
						},
					}, {
						Key: &oraclespb.PropertyKey{
							Name: "prices.ETH.value",
							Type: oraclespb.PropertyKey_TYPE_INTEGER,
						},
						Conditions: []*oraclespb.Condition{
							{
								Value:    "42",
								Operator: oraclespb.Condition_OPERATOR_GREATER_THAN_OR_EQUAL,
							},
						},
					},
				},
			})

			data := oracles.OracleData{
				PubKeys: []string{
					"0xCAFED00D",
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

func testOracleSpecMatchingLessThanPropertiesWorks(t *testing.T) {
	cases := []struct {
		msg       string
		keyType   oraclespb.PropertyKey_Type
		specValue string
		dataValue string
		matched   bool
	}{
		{
			msg:       "integer: data value should be less than spec value",
			keyType:   oraclespb.PropertyKey_TYPE_INTEGER,
			specValue: "84",
			dataValue: "42",
			matched:   true,
		}, {
			msg:       "integer: data value should not be less than spec value",
			keyType:   oraclespb.PropertyKey_TYPE_INTEGER,
			specValue: "42",
			dataValue: "84",
			matched:   false,
		}, {
			msg:       "decimal: data value should be less than spec value",
			keyType:   oraclespb.PropertyKey_TYPE_DECIMAL,
			specValue: "3.4",
			dataValue: "1.2",
			matched:   true,
		}, {
			msg:       "decimal: data value should not be less than spec value",
			keyType:   oraclespb.PropertyKey_TYPE_DECIMAL,
			specValue: "1.2",
			dataValue: "3.4",
			matched:   false,
		}, {
			msg:       "timestamp: data value should be less than spec value",
			keyType:   oraclespb.PropertyKey_TYPE_TIMESTAMP,
			specValue: "2222222222",
			dataValue: "1111111111",
			matched:   true,
		}, {
			msg:       "timestamp: data value should not be less than spec value",
			keyType:   oraclespb.PropertyKey_TYPE_TIMESTAMP,
			specValue: "1111111111",
			dataValue: "2222222222",
			matched:   false,
		},
	}

	for _, c := range cases {
		t.Run(c.msg, func(t *testing.T) {

			// given
			spec, _ := oracles.NewOracleSpec(oraclespb.OracleSpec{
				PubKeys: []string{
					"0xCAFED00D",
				},
				Filters: []*oraclespb.Filter{
					{
						Key: &oraclespb.PropertyKey{
							Name: "prices.BTC.value",
							Type: c.keyType,
						},
						Conditions: []*oraclespb.Condition{
							{
								Value:    c.specValue,
								Operator: oraclespb.Condition_OPERATOR_LESS_THAN,
							},
						},
					}, {
						Key: &oraclespb.PropertyKey{
							Name: "prices.ETH.value",
							Type: oraclespb.PropertyKey_TYPE_INTEGER,
						},
						Conditions: []*oraclespb.Condition{
							{
								Value:    "42",
								Operator: oraclespb.Condition_OPERATOR_LESS_THAN,
							},
						},
					},
				},
			})

			data := oracles.OracleData{
				PubKeys: []string{
					"0xCAFED00D",
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
		})
	}
}

func testOracleSpecMatchingLessThanOrEqualPropertiesWorks(t *testing.T) {
	cases := []struct {
		msg       string
		keyType   oraclespb.PropertyKey_Type
		specValue string
		dataValue string
		matched   bool
	}{
		{
			msg:       "integer: data value should be equal to spec value",
			keyType:   oraclespb.PropertyKey_TYPE_INTEGER,
			specValue: "42",
			dataValue: "42",
			matched:   true,
		}, {
			msg:       "integer: data value should be less than spec value",
			keyType:   oraclespb.PropertyKey_TYPE_INTEGER,
			specValue: "84",
			dataValue: "42",
			matched:   true,
		}, {
			msg:       "integer: data value should not be less than spec value",
			keyType:   oraclespb.PropertyKey_TYPE_INTEGER,
			specValue: "42",
			dataValue: "84",
			matched:   false,
		}, {
			msg:       "decimal: data value should be equal to spec value",
			keyType:   oraclespb.PropertyKey_TYPE_DECIMAL,
			specValue: "1.2",
			dataValue: "1.2",
			matched:   true,
		}, {
			msg:       "decimal: data value should be less than spec value",
			keyType:   oraclespb.PropertyKey_TYPE_DECIMAL,
			specValue: "3.4",
			dataValue: "1.2",
			matched:   true,
		}, {
			msg:       "decimal: data value should not be less than spec value",
			keyType:   oraclespb.PropertyKey_TYPE_DECIMAL,
			specValue: "1.2",
			dataValue: "3.4",
			matched:   false,
		}, {
			msg:       "timestamp: data value should be equal to spec value",
			keyType:   oraclespb.PropertyKey_TYPE_TIMESTAMP,
			specValue: "1111111111",
			dataValue: "1111111111",
			matched:   true,
		}, {
			msg:       "timestamp: data value should be less than spec value",
			keyType:   oraclespb.PropertyKey_TYPE_TIMESTAMP,
			specValue: "2222222222",
			dataValue: "1111111111",
			matched:   true,
		}, {
			msg:       "timestamp: data value should not be less than spec value",
			keyType:   oraclespb.PropertyKey_TYPE_TIMESTAMP,
			specValue: "1111111111",
			dataValue: "2222222222",
			matched:   false,
		},
	}

	for _, c := range cases {
		t.Run(c.msg, func(t *testing.T) {

			// given
			spec, _ := oracles.NewOracleSpec(oraclespb.OracleSpec{
				PubKeys: []string{
					"0xCAFED00D",
				},
				Filters: []*oraclespb.Filter{
					{
						Key: &oraclespb.PropertyKey{
							Name: "prices.BTC.value",
							Type: c.keyType,
						},
						Conditions: []*oraclespb.Condition{
							{
								Value:    c.specValue,
								Operator: oraclespb.Condition_OPERATOR_LESS_THAN_OR_EQUAL,
							},
						},
					}, {
						Key: &oraclespb.PropertyKey{
							Name: "prices.ETH.value",
							Type: oraclespb.PropertyKey_TYPE_INTEGER,
						},
						Conditions: []*oraclespb.Condition{
							{
								Value:    "42",
								Operator: oraclespb.Condition_OPERATOR_LESS_THAN_OR_EQUAL,
							},
						},
					},
				},
			})

			data := oracles.OracleData{
				PubKeys: []string{
					"0xCAFED00D",
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

func testOracleSpecMatchingPropertiesPresenceSucceeds(t *testing.T) {
	cases := []struct {
		msg     string
		keyType oraclespb.PropertyKey_Type
	}{
		{
			msg:     "integer values is present",
			keyType: oraclespb.PropertyKey_TYPE_INTEGER,
		}, {
			msg:     "boolean values is present",
			keyType: oraclespb.PropertyKey_TYPE_BOOLEAN,
		}, {
			msg:     "decimal values is present",
			keyType: oraclespb.PropertyKey_TYPE_DECIMAL,
		}, {
			msg:     "string values is present",
			keyType: oraclespb.PropertyKey_TYPE_STRING,
		}, {
			msg:     "timestamp values is present",
			keyType: oraclespb.PropertyKey_TYPE_TIMESTAMP,
		},
	}

	for _, c := range cases {
		t.Run(c.msg, func(t *testing.T) {

			// given
			spec, _ := oracles.NewOracleSpec(oraclespb.OracleSpec{
				PubKeys: []string{
					"0xCAFED00D",
				},
				Filters: []*oraclespb.Filter{
					{
						Key: &oraclespb.PropertyKey{
							Name: "prices.BTC.value",
							Type: c.keyType,
						},
						Conditions: []*oraclespb.Condition{},
					},
					{
						Key: &oraclespb.PropertyKey{
							Name: "prices.ETH.value",
							Type: oraclespb.PropertyKey_TYPE_INTEGER,
						},
						Conditions: []*oraclespb.Condition{},
					},
				},
			})

			data := oracles.OracleData{
				PubKeys: []string{
					"0xCAFED00D",
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
		keyType oraclespb.PropertyKey_Type
	}{
		{
			msg:     "integer values is absent",
			keyType: oraclespb.PropertyKey_TYPE_INTEGER,
		}, {
			msg:     "boolean values is absent",
			keyType: oraclespb.PropertyKey_TYPE_BOOLEAN,
		}, {
			msg:     "decimal values is absent",
			keyType: oraclespb.PropertyKey_TYPE_DECIMAL,
		}, {
			msg:     "string values is absent",
			keyType: oraclespb.PropertyKey_TYPE_STRING,
		}, {
			msg:     "timestamp values is absent",
			keyType: oraclespb.PropertyKey_TYPE_TIMESTAMP,
		},
	}

	for _, c := range cases {
		t.Run(c.msg, func(t *testing.T) {

			// given
			spec, _ := oracles.NewOracleSpec(oraclespb.OracleSpec{
				PubKeys: []string{
					"0xCAFED00D",
				},
				Filters: []*oraclespb.Filter{
					{
						Key: &oraclespb.PropertyKey{
							Name: "prices.BTC.value",
							Type: c.keyType,
						},
						Conditions: []*oraclespb.Condition{},
					},
					{
						Key: &oraclespb.PropertyKey{
							Name: "prices.ETH.value",
							Type: oraclespb.PropertyKey_TYPE_INTEGER,
						},
						Conditions: []*oraclespb.Condition{},
					},
				},
			})

			data := oracles.OracleData{
				PubKeys: []string{
					"0xCAFED00D",
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
		keyType   oraclespb.PropertyKey_Type
		specValue string
		dataValue string
	}{
		{
			msg:       "not an integer",
			keyType:   oraclespb.PropertyKey_TYPE_INTEGER,
			specValue: "42",
			dataValue: "not an integer",
		}, {
			msg:       "not a boolean",
			keyType:   oraclespb.PropertyKey_TYPE_BOOLEAN,
			specValue: "true",
			dataValue: "not a boolean",
		}, {
			msg:       "not a decimal",
			keyType:   oraclespb.PropertyKey_TYPE_DECIMAL,
			specValue: "1.2",
			dataValue: "not a decimal",
		}, {
			msg:       "not a timestamp",
			keyType:   oraclespb.PropertyKey_TYPE_TIMESTAMP,
			specValue: "1111111111",
			dataValue: "not a timestamp",
		},
	}

	for _, c := range cases {
		t.Run(c.msg, func(t *testing.T) {

			// given
			spec, _ := oracles.NewOracleSpec(oraclespb.OracleSpec{
				PubKeys: []string{
					"0xCAFED00D",
				},
				Filters: []*oraclespb.Filter{
					{
						Key: &oraclespb.PropertyKey{
							Name: "prices.BTC.value",
							Type: c.keyType,
						},
						Conditions: []*oraclespb.Condition{
							{
								Value:    c.specValue,
								Operator: oraclespb.Condition_OPERATOR_EQUALS,
							},
						},
					},
				},
			})

			data := oracles.OracleData{
				PubKeys: []string{
					"0xCAFED00D",
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
		keyType          oraclespb.PropertyKey_Type
		declaredProperty string
		boundProperty    string
		expectedMatch    bool
	}{
		{
			msg:              "same integer properties can be bound",
			keyType:          oraclespb.PropertyKey_TYPE_INTEGER,
			declaredProperty: "price.ETH.value",
			boundProperty:    "price.ETH.value",
			expectedMatch:    true,
		}, {
			msg:              "different integer properties cannot be bound",
			keyType:          oraclespb.PropertyKey_TYPE_INTEGER,
			declaredProperty: "price.USD.value",
			boundProperty:    "price.BTC.value",
			expectedMatch:    false,
		}, {
			msg:              "same integer properties can be bound",
			keyType:          oraclespb.PropertyKey_TYPE_BOOLEAN,
			declaredProperty: "price.ETH.value",
			boundProperty:    "price.ETH.value",
			expectedMatch:    true,
		}, {
			msg:              "different integer properties can be bound",
			keyType:          oraclespb.PropertyKey_TYPE_BOOLEAN,
			declaredProperty: "price.USD.value",
			boundProperty:    "price.BTC.value",
			expectedMatch:    false,
		}, {
			msg:              "same integer properties can be bound",
			keyType:          oraclespb.PropertyKey_TYPE_DECIMAL,
			declaredProperty: "price.ETH.value",
			boundProperty:    "price.ETH.value",
			expectedMatch:    true,
		}, {
			msg:              "different integer properties can be bound",
			keyType:          oraclespb.PropertyKey_TYPE_DECIMAL,
			declaredProperty: "price.USD.value",
			boundProperty:    "price.BTC.value",
			expectedMatch:    false,
		}, {
			msg:              "same integer properties can be bound",
			keyType:          oraclespb.PropertyKey_TYPE_STRING,
			declaredProperty: "price.ETH.value",
			boundProperty:    "price.ETH.value",
			expectedMatch:    true,
		}, {
			msg:              "different integer properties can be bound",
			keyType:          oraclespb.PropertyKey_TYPE_STRING,
			declaredProperty: "price.USD.value",
			boundProperty:    "price.BTC.value",
			expectedMatch:    false,
		}, {
			msg:              "same integer properties can be bound",
			keyType:          oraclespb.PropertyKey_TYPE_TIMESTAMP,
			declaredProperty: "price.ETH.value",
			boundProperty:    "price.ETH.value",
			expectedMatch:    true,
		}, {
			msg:              "different integer properties can be bound",
			keyType:          oraclespb.PropertyKey_TYPE_TIMESTAMP,
			declaredProperty: "price.USD.value",
			boundProperty:    "price.BTC.value",
			expectedMatch:    false,
		},
	}

	for _, c := range cases {
		t.Run(c.msg, func(t *testing.T) {

			// given
			spec, _ := oracles.NewOracleSpec(oraclespb.OracleSpec{
				PubKeys: []string{
					"0xCAFED00D",
				},
				Filters: []*oraclespb.Filter{
					{
						Key: &oraclespb.PropertyKey{
							Name: c.declaredProperty,
							Type: c.keyType,
						},
						Conditions: []*oraclespb.Condition{},
					},
				},
			})

			// when
			matched := spec.CanBindProperty(c.boundProperty)

			// then
			assert.Equal(t, c.expectedMatch, matched)
		})
	}
}
