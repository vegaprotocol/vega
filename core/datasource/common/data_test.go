// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package common_test

import (
	"testing"

	"code.vegaprotocol.io/vega/core/datasource/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.vegaprotocol.io/vega/libs/num"
)

func TestOracleData(t *testing.T) {
	t.Run("Getting integer when not present fails", testDataGetMissingIntegerFails)
	t.Run("Getting decimal when not present fails", testDataGetMissingDecimalFails)
	t.Run("Getting boolean when not present fails", testDataGetMissingBooleanFails)
	t.Run("Getting timestamp when not present fails", testDataGetMissingTimestampFails)
	t.Run("Getting string when not present fails", testDataGetMissingStringFails)
	t.Run("Getting integer when not an integer fails", testDataGetIntegerFails)
	t.Run("Getting decimal when not a decimal fails", testDataGetDecimalFails)
	t.Run("Getting boolean when not a boolean fails", testDataGetBooleanFails)
	t.Run("Getting timestamp when not a timestamp fails", testDataGetTimestampFails)
	t.Run("Getting integer succeeds", testDataGetIntegerSucceeds)
	t.Run("Getting decimal succeeds", testDataGetDecimalSucceeds)
	t.Run("Getting boolean succeeds", testDataGetBooleanSucceeds)
	t.Run("Getting timestamp succeeds", testDataGetTimestampSucceeds)
	t.Run("Getting string succeeds", testDataGetStringSucceeds)
	t.Run("Getting uint when not present fails", testDataGetMissingUintFails)
	t.Run("Getting uint when not a uint fails", testDataGetUintFails)
	t.Run("Getting uint succeeds", testDataGetUintSucceeds)
	t.Run("Determining the origin succeeds", testDataDeterminingOriginSucceeds)
}

func testDataGetMissingUintFails(t *testing.T) {
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

func testDataGetUintFails(t *testing.T) {
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

func testDataGetUintSucceeds(t *testing.T) {
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

func testDataGetMissingIntegerFails(t *testing.T) {
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

func testDataGetMissingDecimalFails(t *testing.T) {
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

func testDataGetMissingBooleanFails(t *testing.T) {
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

func testDataGetMissingTimestampFails(t *testing.T) {
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

func testDataGetMissingStringFails(t *testing.T) {
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

func testDataGetIntegerFails(t *testing.T) {
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

func testDataGetDecimalFails(t *testing.T) {
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

func testDataGetBooleanFails(t *testing.T) {
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

func testDataGetTimestampFails(t *testing.T) {
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

func testDataGetIntegerSucceeds(t *testing.T) {
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

func testDataGetDecimalSucceeds(t *testing.T) {
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

func testDataGetBooleanSucceeds(t *testing.T) {
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

func testDataGetTimestampSucceeds(t *testing.T) {
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

func testDataGetStringSucceeds(t *testing.T) {
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

func testDataDeterminingOriginSucceeds(t *testing.T) {
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
