package ethcall_test

import (
	"context"
	"math/big"
	"testing"

	"code.vegaprotocol.io/vega/core/evtforward/ethcall"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContractCall(t *testing.T) {
	ctx := context.Background()
	tc, err := NewToyChain()
	require.NoError(t, err)

	call, err := ethcall.NewCall("get_uint256", []any{big.NewInt(42)}, tc.contractAddr.Hex(), tc.abiBytes)
	require.NoError(t, err)

	res, err := call.Call(ctx, tc.client, nil)
	require.NoError(t, err)
	assert.NotEmpty(t, res.Bytes)

	values, err := res.Values()
	require.NoError(t, err)
	assert.Equal(t, []any{big.NewInt(42)}, values)
}
