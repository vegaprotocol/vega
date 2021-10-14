package bridges

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"time"

	"code.vegaprotocol.io/vega/types/num"

	"github.com/ethereum/go-ethereum/accounts/abi"
	ethcmn "github.com/ethereum/go-ethereum/common"
)

// ERC20Logic yea that's a weird name but
// it just matches the name of the contract.
type ERC20Logic struct {
	signer     Signer
	bridgeAddr string
}

func NewERC20Logic(signer Signer, bridgeAddr string) *ERC20Logic {
	return &ERC20Logic{
		signer:     signer,
		bridgeAddr: bridgeAddr,
	}
}

func (e ERC20Logic) ListAsset(
	tokenAddress string,
	vegaAssetID string,
	nonce *num.Uint,
) (*SignaturePayload, error) {
	typAddr, err := abi.NewType("address", "", nil)
	if err != nil {
		return nil, err
	}
	typBytes32, err := abi.NewType("bytes32", "", nil)
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
			Name: "vega_asset_id",
			Type: typBytes32,
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

	tokenAddressEth := ethcmn.HexToAddress(tokenAddress)
	vegaAssetIDBytes, _ := hex.DecodeString(vegaAssetID)
	var vegaAssetIDArray [32]byte
	copy(vegaAssetIDArray[:], vegaAssetIDBytes[:32])
	buf, err := args.Pack([]interface{}{
		tokenAddressEth, vegaAssetIDArray, nonce.BigInt(), "list_asset",
	}...)
	if err != nil {
		return nil, fmt.Errorf("couldn't pack abi message: %w", err)
	}

	msg, err := packBufAndSubmitter(buf, e.bridgeAddr)
	if err != nil {
		return nil, fmt.Errorf("couldn't pack abi message: %w", err)
	}

	return sign(e.signer, msg)
}

func (e ERC20Logic) RemoveAsset(
	tokenAddress string,
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

	tokenAddressEth := ethcmn.HexToAddress(tokenAddress)
	buf, err := args.Pack([]interface{}{
		tokenAddressEth, nonce.BigInt(), "remove_asset",
	}...)
	if err != nil {
		return nil, fmt.Errorf("couldn't pack abi message: %w", err)
	}

	msg, err := packBufAndSubmitter(buf, e.bridgeAddr)
	if err != nil {
		return nil, fmt.Errorf("couldn't pack abi message: %w", err)
	}

	return sign(e.signer, msg)
}

func (e ERC20Logic) WithdrawAsset(
	tokenAddress string,
	amount *num.Uint,
	ethPartyAddress string,
	expiry time.Time,
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
			Name: "uint256",
			Type: typU256,
		},
		{
			Name: "uint256",
			Type: typU256,
		},
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

	ethTokenAddr := ethcmn.HexToAddress(tokenAddress)
	hexEthPartyAddress := ethcmn.HexToAddress(ethPartyAddress)
	expiryUnix := expiry.Unix() // require in unix by the bridge

	buf, err := args.Pack([]interface{}{
		ethTokenAddr, amount.BigInt(), big.NewInt(expiryUnix),
		hexEthPartyAddress, nonce.BigInt(), "withdraw_asset",
	}...)
	if err != nil {
		return nil, fmt.Errorf("couldn't pack abi message: %w", err)
	}

	msg, err := packBufAndSubmitter(buf, e.bridgeAddr)
	if err != nil {
		return nil, fmt.Errorf("couldn't pack abi message: %w", err)
	}

	return sign(e.signer, msg)
}

func (e ERC20Logic) SetDepositMaximum(
	tokenAddress string,
	maximumAmount *num.Uint,
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
			Name: "uint256",
			Type: typU256,
		},
		{
			Name: "uint256",
			Type: typU256,
		},
		{
			Name: "func_name",
			Type: typString,
		},
	})

	ethTokenAddr := ethcmn.HexToAddress(tokenAddress)

	buf, err := args.Pack([]interface{}{
		ethTokenAddr, maximumAmount.BigInt(),
		nonce.BigInt(), "set_deposit_maximum",
	}...)
	if err != nil {
		return nil, fmt.Errorf("couldn't pack abi message: %w", err)
	}

	msg, err := packBufAndSubmitter(buf, e.bridgeAddr)
	if err != nil {
		return nil, fmt.Errorf("couldn't pack abi message: %w", err)
	}

	return sign(e.signer, msg)
}

func (e ERC20Logic) SetDepositMinimum(
	tokenAddress string,
	minimumAmount *num.Uint,
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
			Name: "uint256",
			Type: typU256,
		},
		{
			Name: "uint256",
			Type: typU256,
		},
		{
			Name: "func_name",
			Type: typString,
		},
	})

	ethTokenAddr := ethcmn.HexToAddress(tokenAddress)

	buf, err := args.Pack([]interface{}{
		ethTokenAddr, minimumAmount.BigInt(),
		nonce.BigInt(), "set_deposit_minimum",
	}...)
	if err != nil {
		return nil, fmt.Errorf("couldn't pack abi message: %w", err)
	}

	msg, err := packBufAndSubmitter(buf, e.bridgeAddr)
	if err != nil {
		return nil, fmt.Errorf("couldn't pack abi message: %w", err)
	}

	return sign(e.signer, msg)
}
