package ethcall_test

import (
	"math/big"
	"testing"

	"code.vegaprotocol.io/vega/core/datasource/external/ethcall"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var TEST_ABI = `[
    {
        "inputs": [
            {"internalType": "int64","name": "","type": "int64"},
            {"internalType": "uint256","name": "","type": "uint256"},
            {"internalType": "string","name": "","type": "string"},
            {"internalType": "bool","name": "","type": "bool"},
            {"internalType": "address","name": "","type": "address"},
            {"internalType": "int256[]","name": "","type": "int256[]"},
            {
                "components": [
                    {"internalType": "string","name": "name","type": "string"},
                    {"internalType": "uint16","name": "age","type": "uint16"
                    }
                ],"internalType": "struct MyContract.Person","name": "","type": "tuple"
            }
        ],
        "name": "testy",
        "outputs": [
            {"internalType": "uint256","name": "","type": "uint256"}
        ],
        "stateMutability": "pure",
        "type": "function"
    }
]`

func TestJsonArgsToAny(t *testing.T) {
	goArgs := []any{
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

	jsonArgs, err := ethcall.AnyArgsToJson(goArgs)
	require.NoError(t, err)

	anyArgs, err := ethcall.JsonArgsToAny("testy", jsonArgs, []byte(TEST_ABI))
	require.NoError(t, err)
	assert.Equal(t, goArgs, anyArgs)
}
