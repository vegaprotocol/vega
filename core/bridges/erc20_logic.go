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

package bridges

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"time"

	crypto "code.vegaprotocol.io/vega/libs/crypto/signature"
	"code.vegaprotocol.io/vega/libs/num"

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
	lifetimeLimit *num.Uint,
	withdrawThreshold *num.Uint,
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
			Name: "lifetime_limit",
			Type: typU256,
		},
		{
			Name: "withdraw_treshold",
			Type: typU256,
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
		tokenAddressEth,
		vegaAssetIDArray,
		lifetimeLimit.BigInt(),
		withdrawThreshold.BigInt(),
		nonce.BigInt(),
		"list_asset",
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

func (e ERC20Logic) buildListAssetMessage(
	tokenAddress string,
	vegaAssetID string,
	lifetimeLimit *num.Uint,
	withdrawThreshold *num.Uint,
	nonce *num.Uint,
) ([]byte, error) {
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
			Name: "lifetime_limit",
			Type: typU256,
		},
		{
			Name: "withdraw_treshold",
			Type: typU256,
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
		tokenAddressEth,
		vegaAssetIDArray,
		lifetimeLimit.BigInt(),
		withdrawThreshold.BigInt(),
		nonce.BigInt(),
		"list_asset",
	}...)
	if err != nil {
		return nil, fmt.Errorf("couldn't pack abi message: %w", err)
	}

	msg, err := packBufAndSubmitter(buf, e.bridgeAddr)
	if err != nil {
		return nil, fmt.Errorf("couldn't pack abi message: %w", err)
	}

	return msg, nil
}

func (e ERC20Logic) VerifyListAsset(
	tokenAddress string,
	vegaAssetID string,
	lifetimeLimit *num.Uint,
	withdrawThreshold *num.Uint,
	nonce *num.Uint,
	signatures string,
) ([]string, error) {
	msg, err := e.buildListAssetMessage(
		tokenAddress, vegaAssetID, lifetimeLimit, withdrawThreshold, nonce,
	)
	if err != nil {
		return nil, err
	}

	addresses := []string{}
	var hexCurrent string
	signatures = signatures[2:]
	for len(signatures) > 0 {
		hexCurrent, signatures = signatures[0:130], signatures[130:]
		current, err := hex.DecodeString(hexCurrent)
		if err != nil {
			return nil, fmt.Errorf("invalid signature format: %w", err)
		}

		address, err := crypto.RecoverEthereumAddress(msg, current)
		if err != nil {
			return nil, fmt.Errorf("error recovering ethereum address: %w", err)
		}

		addresses = append(addresses, address.Hex())
	}

	return addresses, nil
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
	creation time.Time,
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
			Name: "address",
			Type: typAddr,
		},
		{
			Name: "uint256",
			Type: typU256,
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
	timestamp := big.NewInt(creation.Unix())

	buf, err := args.Pack([]interface{}{
		ethTokenAddr,
		amount.BigInt(),
		hexEthPartyAddress,
		timestamp,
		nonce.BigInt(),
		"withdraw_asset",
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

func (e ERC20Logic) SetAssetLimits(
	tokenAddress string,
	lifetimeLimit *num.Uint,
	withdrawThreshold *num.Uint,
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
		ethTokenAddr,
		lifetimeLimit.BigInt(),
		withdrawThreshold.BigInt(),
		nonce.BigInt(),
		"set_asset_limits",
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

func (e ERC20Logic) buildSetAssetLimitsMessage(
	tokenAddress string,
	lifetimeLimit *num.Uint,
	withdrawThreshold *num.Uint,
	nonce *num.Uint,
) ([]byte, error) {
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
		ethTokenAddr,
		lifetimeLimit.BigInt(),
		withdrawThreshold.BigInt(),
		nonce.BigInt(),
		"set_asset_limits",
	}...)
	if err != nil {
		return nil, fmt.Errorf("couldn't pack abi message: %w", err)
	}

	msg, err := packBufAndSubmitter(buf, e.bridgeAddr)
	if err != nil {
		return nil, fmt.Errorf("couldn't pack abi message: %w", err)
	}

	return msg, nil
}

func (e ERC20Logic) VerifySetAssetLimits(
	tokenAddress string,
	lifetimeLimit *num.Uint,
	withdrawThreshold *num.Uint,
	nonce *num.Uint,
	signatures string,
) ([]string, error) {
	msg, err := e.buildSetAssetLimitsMessage(
		tokenAddress, lifetimeLimit, withdrawThreshold, nonce,
	)
	if err != nil {
		return nil, err
	}

	addresses := []string{}
	var hexCurrent string
	signatures = signatures[2:]
	for len(signatures) > 0 {
		hexCurrent, signatures = signatures[0:130], signatures[130:]
		current, err := hex.DecodeString(hexCurrent)
		if err != nil {
			return nil, fmt.Errorf("invalid signature format: %w", err)
		}

		fmt.Printf("hexCurrent: %v\n", hexCurrent)

		address, err := crypto.RecoverEthereumAddress(msg, current)
		if err != nil {
			return nil, fmt.Errorf("error recovering ethereum address: %w", err)
		}

		addresses = append(addresses, address.Hex())
	}

	return addresses, nil
}

func (e ERC20Logic) SetWithdrawDelay(
	delay time.Duration,
	nonce *num.Uint,
) (*SignaturePayload, error) {
	msg, err := e.buildWithdrawDelayMessage(
		delay, nonce,
	)
	if err != nil {
		return nil, err
	}

	return sign(e.signer, msg)
}

func (e ERC20Logic) buildWithdrawDelayMessage(
	delay time.Duration,
	nonce *num.Uint,
) ([]byte, error) {
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

	delayBig := big.NewInt(int64(delay.Seconds()))
	buf, err := args.Pack([]interface{}{
		delayBig,
		nonce.BigInt(),
		"set_withdraw_delay",
	}...)
	if err != nil {
		return nil, fmt.Errorf("couldn't pack abi message: %w", err)
	}

	return packBufAndSubmitter(buf, e.bridgeAddr)
}

func (e ERC20Logic) VerifyWithdrawDelay(
	delay time.Duration,
	nonce *num.Uint,
	signatures string,
) ([]string, error) {
	msg, err := e.buildWithdrawDelayMessage(
		delay, nonce,
	)
	if err != nil {
		return nil, err
	}

	addresses := []string{}
	var hexCurrent string
	signatures = signatures[2:]
	for len(signatures) > 0 {
		hexCurrent, signatures = signatures[0:130], signatures[130:]
		current, err := hex.DecodeString(hexCurrent)
		if err != nil {
			return nil, fmt.Errorf("invalid signature format: %w", err)
		}

		address, err := crypto.RecoverEthereumAddress(msg, current)
		if err != nil {
			return nil, fmt.Errorf("error recovering ethereum address: %w", err)
		}

		addresses = append(addresses, address.Hex())
	}

	return addresses, nil
}

func (e ERC20Logic) GlobalStop(
	nonce *num.Uint,
) (*SignaturePayload, error) {
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
			Name: "uint256",
			Type: typU256,
		},
		{
			Name: "func_name",
			Type: typString,
		},
	})

	buf, err := args.Pack([]interface{}{
		nonce.BigInt(),
		"global_stop",
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

func (e ERC20Logic) GlobalResume(
	nonce *num.Uint,
) (*SignaturePayload, error) {
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
			Name: "uint256",
			Type: typU256,
		},
		{
			Name: "func_name",
			Type: typString,
		},
	})

	buf, err := args.Pack([]interface{}{
		nonce.BigInt(),
		"global_resume",
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
