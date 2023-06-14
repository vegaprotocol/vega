package ethcall_test

import (
	"context"
	"math/big"
	"testing"

	"code.vegaprotocol.io/vega/core/evtforward/ethcall"
	"code.vegaprotocol.io/vega/core/types"
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

	spec := types.EthCallSpec{
		ArgsJson:   argsJson,
		Address:    tc.contractAddr.Hex(),
		AbiJson:    tc.abiBytes,
		Method:     "testy1",
		Normaliser: map[string]string{"badger": `$[0]`, "static": "66"},
	}

	call, err := ethcall.NewCall(spec)
	require.NoError(t, err)

	res, err := call.Call(ctx, tc.client, 1)
	require.NoError(t, err)
	assert.NotEmpty(t, res.Bytes)

	assert.Equal(t, []any{int64(42), big.NewInt(42), "hello", true, common.HexToAddress("0xb794f5ea0ba39494ce839613fffba74279579268")}, res.Values)
	assert.Equal(t, map[string]string{"badger": "42", "static": "66"}, res.Normalised)
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

	spec := types.EthCallSpec{
		ArgsJson: argsJson,
		Address:  tc.contractAddr.Hex(),
		AbiJson:  tc.abiBytes,
		Method:   "testy2",
		Normaliser: map[string]string{
			// "inside_bigint_list": `$[0][1]`, // doesn't work
			// "inside_struct":   `$[1].Name`, // doesn't work - wants  map[string]interface{} not custom struct; work to be done
			"static": "66",
		},
	}

	call, err := ethcall.NewCall(spec)
	require.NoError(t, err)

	res, err := call.Call(ctx, tc.client, 1)
	require.NoError(t, err)
	assert.NotEmpty(t, res.Bytes)

	assert.Equal(t, args, res.Values)

	assert.Equal(t, map[string]string{"static": "66"}, res.Normalised)
}
