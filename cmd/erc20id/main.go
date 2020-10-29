package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"os"

	types "code.vegaprotocol.io/vega/proto"
	"golang.org/x/crypto/sha3"
)

var (
	contractAddress string
)

func init() {
	flag.StringVar(&contractAddress, "contract-address", "", "the ethereum contract address")
}

func main() {
	flag.Parse()

	if len(contractAddress) <= 0 {
		fmt.Printf("please specify contract address\n")
		os.Exit(1)
	}

	as := types.AssetSource{
		Source: &types.AssetSource_Erc20{
			Erc20: &types.ERC20{
				ContractAddress: contractAddress,
			},
		},
	}

	h := func(key []byte) []byte {
		hasher := sha3.New256()
		hasher.Write([]byte(key))
		return hasher.Sum(nil)
	}
	id := hex.EncodeToString(h([]byte(as.String())))
	fmt.Printf("%v\n", id)
}
