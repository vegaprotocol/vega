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

package ethcall_test

import (
	"context"
	"math/big"
	"testing"

	dscommon "code.vegaprotocol.io/vega/core/datasource/common"
	"code.vegaprotocol.io/vega/core/datasource/external/ethcall"
	ethcallcommon "code.vegaprotocol.io/vega/core/datasource/external/ethcall/common"
	v1 "code.vegaprotocol.io/vega/protos/vega/data/v1"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContractCall(t *testing.T) {
	ctx := context.Background()
	tc, err := NewToyChain()
	require.NoError(t, err)

	args := []any{
		int64(42),
		big.NewInt(42),
		"hello",
		true,
		common.HexToAddress("0xb794f5ea0ba39494ce839613fffba74279579268"),
	}

	argsJson, err := ethcall.AnyArgsToJson(args)
	require.NoError(t, err)

	spec := ethcallcommon.Spec{
		ArgsJson:    argsJson,
		Address:     tc.contractAddr.Hex(),
		AbiJson:     tc.abiBytes,
		Method:      "testy1",
		Normalisers: map[string]string{"badger": `$[0]`, "static": "66"},
	}

	call, err := ethcall.NewCall(spec)
	require.NoError(t, err)

	res, err := call.Call(ctx, tc.client, 1)
	require.NoError(t, err)
	assert.NotEmpty(t, res.Bytes)

	assert.Equal(t, []any{int64(42), big.NewInt(42), "hello", true, common.HexToAddress("0xb794f5ea0ba39494ce839613fffba74279579268")}, res.Values)
	assert.Equal(t, map[string]string{"badger": "42", "static": "66"}, res.Normalised)
}

func TestContractCallWithStaticBool(t *testing.T) {
	ctx := context.Background()
	tc, err := NewToyChain()
	require.NoError(t, err)

	args := []any{
		int64(42),
		big.NewInt(42),
		"hello",
		true,
		common.HexToAddress("0xb794f5ea0ba39494ce839613fffba74279579268"),
	}

	argsJson, err := ethcall.AnyArgsToJson(args)
	require.NoError(t, err)

	spec := ethcallcommon.Spec{
		ArgsJson:    argsJson,
		Address:     tc.contractAddr.Hex(),
		AbiJson:     tc.abiBytes,
		Method:      "testy1",
		Normalisers: map[string]string{"badger": `$[0]`, "static": "true"},
	}

	call, err := ethcall.NewCall(spec)
	require.NoError(t, err)

	res, err := call.Call(ctx, tc.client, 1)
	require.NoError(t, err)
	assert.NotEmpty(t, res.Bytes)

	assert.Equal(t, []any{int64(42), big.NewInt(42), "hello", true, common.HexToAddress("0xb794f5ea0ba39494ce839613fffba74279579268")}, res.Values)
	assert.Equal(t, map[string]string{"badger": "42", "static": "true"}, res.Normalised)
}

func TestContractCall2(t *testing.T) {
	ctx := context.Background()
	tc, err := NewToyChain()
	require.NoError(t, err)

	args := []any{
		[]*big.Int{big.NewInt(10), big.NewInt(20)},
		struct {
			Name string `json:"name"`
			Age  uint16 `json:"age"`
		}{Name: "test", Age: 42},
	}

	argsJson, err := ethcall.AnyArgsToJson(args)
	require.NoError(t, err)

	spec := ethcallcommon.Spec{
		ArgsJson: argsJson,
		Address:  tc.contractAddr.Hex(),
		AbiJson:  tc.abiBytes,
		Method:   "testy2",
		Normalisers: map[string]string{
			"inside_bigint_list": `$[0][1]`,
			"inside_struct":      `$[1].name`,
			"static":             "66",
		},
	}

	call, err := ethcall.NewCall(spec)
	require.NoError(t, err)

	res, err := call.Call(ctx, tc.client, 1)
	require.NoError(t, err)
	assert.NotEmpty(t, res.Bytes)

	assert.Equal(t, args, res.Values)

	assert.Equal(t, map[string]string{"static": "66", "inside_struct": "test", "inside_bigint_list": "20"}, res.Normalised)
}

func TestContractFilters(t *testing.T) {
	ctx := context.Background()
	tc, err := NewToyChain()
	require.NoError(t, err)

	args := []any{
		int64(42),
		big.NewInt(42),
		"hello",
		true,
		common.HexToAddress("0xb794f5ea0ba39494ce839613fffba74279579268"),
	}

	argsJson, err := ethcall.AnyArgsToJson(args)
	require.NoError(t, err)

	spec := ethcallcommon.Spec{
		ArgsJson:    argsJson,
		Address:     tc.contractAddr.Hex(),
		AbiJson:     tc.abiBytes,
		Method:      "testy1",
		Normalisers: map[string]string{"badger": `$[0]`, "static": "66"},
		Filters: []*dscommon.SpecFilter{
			{
				Key: &dscommon.SpecPropertyKey{
					Name: "badger",
					Type: v1.PropertyKey_TYPE_INTEGER,
				},
				Conditions: []*dscommon.SpecCondition{
					{
						Operator: v1.Condition_OPERATOR_GREATER_THAN_OR_EQUAL,
						Value:    "50",
					},
				},
			},
		},
	}

	call, err := ethcall.NewCall(spec)
	require.NoError(t, err)

	res, err := call.Call(ctx, tc.client, 1)
	require.NoError(t, err)
	assert.NotEmpty(t, res.Bytes)
	assert.False(t, res.PassesFilters)
	assert.Equal(t, []any{int64(42), big.NewInt(42), "hello", true, common.HexToAddress("0xb794f5ea0ba39494ce839613fffba74279579268")}, res.Values)
	assert.Equal(t, map[string]string{"badger": "42", "static": "66"}, res.Normalised)
}
