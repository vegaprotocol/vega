package bridges

import (
	"fmt"

	"code.vegaprotocol.io/vega/types/num"

	"github.com/ethereum/go-ethereum/accounts/abi"
	ethcmn "github.com/ethereum/go-ethereum/common"
)

type ERC20AssetPool struct {
	signer   Signer
	poolAddr string
}

func NewERC20AssetPool(signer Signer, poolAddr string) *ERC20AssetPool {
	return &ERC20AssetPool{
		signer:   signer,
		poolAddr: poolAddr,
	}
}

func (e ERC20AssetPool) SetBridgeAddress(
	newAddress string,
	nonce *num.Uint,
) (*SignaturePayload, error) {
	typAddr, err := abi.NewType("address", "", nil)
	if err != nil {
		return nil, err
	}
	typString, err := abi.NewType("string", "", nil)
	if err != nil {
		return nil, err
	}
	typU256, err := abi.NewType("uint256", "", nil)
	if err != nil {
		return nil, err
	}

	args := abi.Arguments([]abi.Argument{
		{
			Name: "address",
			Type: typAddr,
		},
		{
			Name: "nonce",
			Type: typU256,
		},
		{
			Name: "func_name",
			Type: typString,
		},
	})

	newAddressEth := ethcmn.HexToAddress(newAddress)
	buf, err := args.Pack([]interface{}{
		newAddressEth, nonce.BigInt(), "set_bridge_address",
	}...)
	if err != nil {
		return nil, fmt.Errorf("couldn't pack abi message: %w", err)
	}

	msg, err := packBufAndSubmitter(buf, e.poolAddr)
	if err != nil {
		return nil, fmt.Errorf("couldn't pack abi message: %w", err)
	}

	return sign(e.signer, msg)
}
