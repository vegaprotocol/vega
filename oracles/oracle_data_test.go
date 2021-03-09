package oracles_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.vegaprotocol.io/vega/oracles"
)

func TestOracleData(t *testing.T) {
	t.Run("Getting integer when not present fails", testOracleDataGetMissingIntegerFails)
	t.Run("Getting decimal when not present fails", testOracleDataGetMissingDecimalFails)
	t.Run("Getting boolean when not present fails", testOracleDataGetMissingBooleanFails)
	t.Run("Getting timestamp when not present fails", testOracleDataGetMissingTimestampFails)
	t.Run("Getting string when not present fails", testOracleDataGetMissingStringFails)
	t.Run("Getting integer when not an integer fails", testOracleDataGetIntegerFails)
	t.Run("Getting decimal when not a decimal fails", testOracleDataGetDecimalFails)
	t.Run("Getting boolean when not a boolean fails", testOracleDataGetBooleanFails)
	t.Run("Getting timestamp when not a timestamp fails", testOracleDataGetTimestampFails)
	t.Run("Getting integer succeeds", testOracleDataGetIntegerSucceeds)
	t.Run("Getting decimal succeeds", testOracleDataGetDecimalSucceeds)
	t.Run("Getting boolean succeeds", testOracleDataGetBooleanSucceeds)
	t.Run("Getting timestamp succeeds", testOracleDataGetTimestampSucceeds)
	t.Run("Getting string succeeds", testOracleDataGetStringSucceeds)
}

func testOracleDataGetMissingIntegerFails(t *testing.T) {
	// given
	data := oracles.OracleData{
		PubKeys: []string{
			"0xDEADBEEF",
		},
		Data: map[string]string{
			"my_key": "42",
		},
	}

	// when
	_, err := data.GetInteger("my_other_key")

	// then
	require.Error(t, err)
	assert.Equal(t, "property \"my_other_key\" not found", err.Error())
}

func testOracleDataGetMissingDecimalFails(t *testing.T) {
	// given
	data := oracles.OracleData{
		PubKeys: []string{
			"0xDEADBEEF",
		},
		Data: map[string]string{
			"my_key": "12.34",
		},
	}

	// when
	_, err := data.GetDecimal("my_other_key")

	// then
	require.Error(t, err)
	assert.Equal(t, "property \"my_other_key\" not found", err.Error())
}

func testOracleDataGetMissingBooleanFails(t *testing.T) {
	// given
	data := oracles.OracleData{
		PubKeys: []string{
			"0xDEADBEEF",
		},
		Data: map[string]string{
			"my_key": "true",
		},
	}

	// when
	_, err := data.GetBoolean("my_other_key")

	// then
	require.Error(t, err)
	assert.Equal(t, "property \"my_other_key\" not found", err.Error())
}

func testOracleDataGetMissingTimestampFails(t *testing.T) {
	// given
	data := oracles.OracleData{
		PubKeys: []string{
			"0xDEADBEEF",
		},
		Data: map[string]string{
			"my_key": "10000000",
		},
	}

	// when
	_, err := data.GetTimestamp("my_other_key")

	// then
	require.Error(t, err)
	assert.Equal(t, "property \"my_other_key\" not found", err.Error())
}

func testOracleDataGetMissingStringFails(t *testing.T) {
	// given
	data := oracles.OracleData{
		PubKeys: []string{
			"0xDEADBEEF",
		},
		Data: map[string]string{
			"my_key": "hello",
		},
	}

	// when
	_, err := data.GetString("my_other_key")

	// then
	require.Error(t, err)
	assert.Equal(t, "property \"my_other_key\" not found", err.Error())
}

func testOracleDataGetIntegerFails(t *testing.T) {
	// given
	data := oracles.OracleData{
		PubKeys: []string{
			"0xDEADBEEF",
		},
		Data: map[string]string{
			"my_key": "not an integer",
		},
	}

	// when
	_, err := data.GetInteger("my_key")

	// then
	require.Error(t, err)
}

func testOracleDataGetDecimalFails(t *testing.T) {
	// given
	data := oracles.OracleData{
		PubKeys: []string{
			"0xDEADBEEF",
		},
		Data: map[string]string{
			"my_key": "not a decimal",
		},
	}

	// when
	_, err := data.GetDecimal("my_key")

	// then
	require.Error(t, err)
}

func testOracleDataGetBooleanFails(t *testing.T) {
	// given
	data := oracles.OracleData{
		PubKeys: []string{
			"0xDEADBEEF",
		},
		Data: map[string]string{
			"my_key": "not a boolean",
		},
	}

	// when
	_, err := data.GetBoolean("my_key")

	// then
	require.Error(t, err)
}

func testOracleDataGetTimestampFails(t *testing.T) {
	// given
	data := oracles.OracleData{
		PubKeys: []string{
			"0xDEADBEEF",
		},
		Data: map[string]string{
			"my_key": "not an integer",
		},
	}

	// when
	_, err := data.GetTimestamp("my_key")

	// then
	require.Error(t, err)
}

func testOracleDataGetIntegerSucceeds(t *testing.T) {
	// given
	data := oracles.OracleData{
		PubKeys: []string{
			"0xDEADBEEF",
		},
		Data: map[string]string{
			"my_key": "42",
		},
	}

	// when
	value, err := data.GetInteger("my_key")

	// then
	require.NoError(t, err)
	assert.Equal(t, int64(42), value)
}

func testOracleDataGetDecimalSucceeds(t *testing.T) {
	// given
	data := oracles.OracleData{
		PubKeys: []string{
			"0xDEADBEEF",
		},
		Data: map[string]string{
			"my_key": "1.2",
		},
	}

	// when
	value, err := data.GetDecimal("my_key")

	// then
	require.NoError(t, err)
	assert.Equal(t, 1.2, value)
}


func testOracleDataGetBooleanSucceeds(t *testing.T) {
	// given
	data := oracles.OracleData{
		PubKeys: []string{
			"0xDEADBEEF",
		},
		Data: map[string]string{
			"my_key": "true",
		},
	}

	// when
	value, err := data.GetBoolean("my_key")

	// then
	require.NoError(t, err)
	assert.True(t, value)

}

func testOracleDataGetTimestampSucceeds(t *testing.T) {
	// given
	data := oracles.OracleData{
		PubKeys: []string{
			"0xDEADBEEF",
		},
		Data: map[string]string{
			"my_key": "10000000",
		},
	}

	// when
	value, err := data.GetTimestamp("my_key")

	// then
	require.NoError(t, err)
	assert.EqualValues(t, 10000000, value)
}

func testOracleDataGetStringSucceeds(t *testing.T) {
	// given
	data := oracles.OracleData{
		PubKeys: []string{
			"0xDEADBEEF",
		},
		Data: map[string]string{
			"my_key": "hello",
		},
	}

	// when
	value, err := data.GetString("my_key")

	// then
	require.NoError(t, err)
	assert.Equal(t, "hello", value)
}
