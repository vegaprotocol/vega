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
	chainID    string
	v1         bool
}

func NewERC20Logic(signer Signer, bridgeAddr string, chainID string, v1 bool) *ERC20Logic {
	return &ERC20Logic{
		signer:     signer,
		bridgeAddr: bridgeAddr,
		chainID:    chainID,
		v1:         v1,
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
		"listAsset",
	}...)
	if err != nil {
		return nil, fmt.Errorf("couldn't pack abi message: %w", err)
	}

	msg, err := packScheme(buf, e.bridgeAddr, e.chainID, e.v1)
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
		"listAsset",
	}...)
	if err != nil {
		return nil, fmt.Errorf("couldn't pack abi message: %w", err)
	}

	msg, err := packScheme(buf, e.bridgeAddr, e.chainID, e.v1)
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
		tokenAddressEth, nonce.BigInt(), "removeAsset",
	}...)
	if err != nil {
		return nil, fmt.Errorf("couldn't pack abi message: %w", err)
	}

	msg, err := packScheme(buf, e.bridgeAddr, e.chainID, e.v1)
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
	msg, err := e.buildWithdrawAssetMessage(
		tokenAddress, amount, ethPartyAddress, creation, nonce,
	)
	if err != nil {
		return nil, err
	}

	return sign(e.signer, msg)
}

func (e ERC20Logic) buildWithdrawAssetMessage(
	tokenAddress string,
	amount *num.Uint,
	ethPartyAddress string,
	creation time.Time,
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
		"withdrawAsset",
	}...)
	if err != nil {
		return nil, fmt.Errorf("couldn't pack abi message: %w", err)
	}

	return packScheme(buf, e.bridgeAddr, e.chainID, e.v1)
}

func (e ERC20Logic) VerifyWithdrawAsset(
	tokenAddress string,
	amount *num.Uint,
	ethPartyAddress string,
	creation time.Time,
	nonce *num.Uint,
	signatures string,
) ([]string, error) {
	msg, err := e.buildWithdrawAssetMessage(
		tokenAddress, amount, ethPartyAddress, creation, nonce,
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
		"setAssetLimits",
	}...)
	if err != nil {
		return nil, fmt.Errorf("couldn't pack abi message: %w", err)
	}

	msg, err := packScheme(buf, e.bridgeAddr, e.chainID, e.v1)
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
		"setAssetLimits",
	}...)
	if err != nil {
		return nil, fmt.Errorf("couldn't pack abi message: %w", err)
	}

	msg, err := packScheme(buf, e.bridgeAddr, e.chainID, e.v1)
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
		"setWithdrawDelay",
	}...)
	if err != nil {
		return nil, fmt.Errorf("couldn't pack abi message: %w", err)
	}

	return packScheme(buf, e.bridgeAddr, e.chainID, e.v1)
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
		"globalStop",
	}...)
	if err != nil {
		return nil, fmt.Errorf("couldn't pack abi message: %w", err)
	}

	msg, err := packScheme(buf, e.bridgeAddr, e.chainID, e.v1)
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
		"globalResume",
	}...)
	if err != nil {
		return nil, fmt.Errorf("couldn't pack abi message: %w", err)
	}

	msg, err := packScheme(buf, e.bridgeAddr, e.chainID, e.v1)
	if err != nil {
		return nil, fmt.Errorf("couldn't pack abi message: %w", err)
	}

	return sign(e.signer, msg)
}

func (e ERC20Logic) VerifyGlobalResume(
	nonce *num.Uint,
	signatures string,
) ([]string, error) {
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
		"globalResume",
	}...)
	if err != nil {
		return nil, fmt.Errorf("couldn't pack abi message: %w", err)
	}

	msg, err := packScheme(buf, e.bridgeAddr, e.chainID, e.v1)
	if err != nil {
		return nil, fmt.Errorf("couldn't pack abi message: %w", err)
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
