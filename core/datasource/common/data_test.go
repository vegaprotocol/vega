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

package common_test

import (
	"testing"

	"code.vegaprotocol.io/vega/core/datasource/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.vegaprotocol.io/vega/libs/num"
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
	t.Run("Getting uint when not present fails", testOracleDataGetMissingUintFails)
	t.Run("Getting uint when not a uint fails", testOracleDataGetUintFails)
	t.Run("Getting uint succeeds", testOracleDataGetUintSucceeds)
	t.Run("Determining the origin succeeds", testOracleDataDeterminingOriginSucceeds)
}

func testOracleDataGetMissingUintFails(t *testing.T) {
	// given
	data := common.Data{
		Signers: []*common.Signer{
			common.CreateSignerFromString("0xDEADBEEF", common.SignerTypePubKey),
		},
		Data: map[string]string{
			"my_key": "42",
		},
	}

	// when
	_, err := data.GetUint("my_other_key")

	// then
	require.Error(t, err)
	assert.Equal(t, "property \"my_other_key\" not found", err.Error())
}

func testOracleDataGetUintFails(t *testing.T) {
	// given
	data := common.Data{
		Signers: []*common.Signer{
			common.CreateSignerFromString("0xDEADBEEF", common.SignerTypePubKey),
		},
		Data: map[string]string{
			"my_key": "not an integer",
		},
	}

	// when
	_, err := data.GetUint("my_key")

	// then
	require.Error(t, err)
	assert.Equal(t, "could not parse value 'not an integer' for property 'my_key'", err.Error())
}

func testOracleDataGetUintSucceeds(t *testing.T) {
	expect := num.NewUint(123)
	// given
	data := common.Data{
		Signers: []*common.Signer{
			common.CreateSignerFromString("0xDEADBEEF", common.SignerTypePubKey),
		},
		Data: map[string]string{
			"my_key": expect.String(),
		},
	}

	// when
	value, err := data.GetUint("my_key")

	// then
	require.NoError(t, err)
	require.True(t, expect.EQ(value))
}

func testOracleDataGetMissingIntegerFails(t *testing.T) {
	// given
	data := common.Data{
		Signers: []*common.Signer{
			common.CreateSignerFromString("0xDEADBEEF", common.SignerTypePubKey),
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
	data := common.Data{
		Signers: []*common.Signer{
			common.CreateSignerFromString("0xDEADBEEF", common.SignerTypePubKey),
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
	data := common.Data{
		Signers: []*common.Signer{
			common.CreateSignerFromString("0xDEADBEEF", common.SignerTypePubKey),
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
	data := common.Data{
		Signers: []*common.Signer{
			common.CreateSignerFromString("0xDEADBEEF", common.SignerTypePubKey),
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
	data := common.Data{
		Signers: []*common.Signer{
			common.CreateSignerFromString("0xDEADBEEF", common.SignerTypePubKey),
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
	data := common.Data{
		Signers: []*common.Signer{
			common.CreateSignerFromString("0xDEADBEEF", common.SignerTypePubKey),
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
	data := common.Data{
		Signers: []*common.Signer{
			common.CreateSignerFromString("0xDEADBEEF", common.SignerTypePubKey),
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
	data := common.Data{
		Signers: []*common.Signer{
			common.CreateSignerFromString("0xDEADBEEF", common.SignerTypePubKey),
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
	data := common.Data{
		Signers: []*common.Signer{
			common.CreateSignerFromString("0xDEADBEEF", common.SignerTypePubKey),
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
	data := common.Data{
		Signers: []*common.Signer{
			common.CreateSignerFromString("0xDEADBEEF", common.SignerTypePubKey),
		},
		Data: map[string]string{
			"my_key": "42",
		},
	}

	// when
	value, err := data.GetInteger("my_key")

	// then
	require.NoError(t, err)
	assert.True(t, num.NewInt(42).EQ(value))
}

func testOracleDataGetDecimalSucceeds(t *testing.T) {
	// given
	data := common.Data{
		Signers: []*common.Signer{
			common.CreateSignerFromString("0xDEADBEEF", common.SignerTypePubKey),
		},
		Data: map[string]string{
			"my_key": "1.2",
		},
	}

	// when
	value, err := data.GetDecimal("my_key")

	// then
	require.NoError(t, err)
	assert.True(t, num.DecimalFromFloat(1.2).Equal(value))
}

func testOracleDataGetBooleanSucceeds(t *testing.T) {
	// given
	data := common.Data{
		Signers: []*common.Signer{
			common.CreateSignerFromString("0xDEADBEEF", common.SignerTypePubKey),
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
	data := common.Data{
		Signers: []*common.Signer{
			common.CreateSignerFromString("0xDEADBEEF", common.SignerTypePubKey),
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
	data := common.Data{
		Signers: []*common.Signer{
			common.CreateSignerFromString("0xDEADBEEF", common.SignerTypePubKey),
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

func testOracleDataDeterminingOriginSucceeds(t *testing.T) {
	tcs := []struct {
		name                 string
		pubkeys              []*common.Signer
		isFromInternalOracle bool
	}{
		{
			name:                 "considered from internal oracle without public keys",
			pubkeys:              []*common.Signer{},
			isFromInternalOracle: true,
		}, {
			name: "considered from external oracle with public keys",
			pubkeys: []*common.Signer{
				common.CreateSignerFromString("0xDEADBEEF", common.SignerTypePubKey),
			},
			isFromInternalOracle: false,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			// given
			data := common.Data{
				Signers: tc.pubkeys,
				Data: map[string]string{
					"my_key": "hello",
				},
			}

			// then
			assert.Equal(tt, tc.isFromInternalOracle, data.FromInternalOracle())
		})
	}
}
