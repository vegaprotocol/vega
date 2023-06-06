package ethcall_test

import (
	"math/big"
	"testing"

	"code.vegaprotocol.io/vega/core/evtforward/ethcall"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestSpec(t *testing.T) {
	abiBytes, err := testData.ReadFile("testdata/MyContract.abi")
	require.NoError(t, err)

	args := []any{
		int64(42),
		big.NewInt(42),
		"hello",
		true,
		common.HexToAddress("0xb794f5ea0ba39494ce839613fffba74279579268"),
		[]*big.Int{big.NewInt(10), big.NewInt(20)},
		struct {
			Name string `json:"name"`
			Age  uint16 `json:"age"`
		}{Name: "test", Age: 42},
	}

	_, err = ethcall.NewCall("testy", args, "0x123", abiBytes)

	require.NoError(t, err)
}
