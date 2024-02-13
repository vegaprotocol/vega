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

package erc20

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"code.vegaprotocol.io/vega/core/assets/common"
	"code.vegaprotocol.io/vega/core/bridges"
	ethnw "code.vegaprotocol.io/vega/core/nodewallets/eth"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	typespb "code.vegaprotocol.io/vega/protos/vega"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	ethcommon "github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
)

var ErrNotAnErc20Asset = errors.New("not an erc20 asset")

//go:generate go run github.com/golang/mock/mockgen -destination mocks/eth_client_mock.go -package mocks code.vegaprotocol.io/vega/core/assets/erc20 ETHClient
type ETHClient interface {
	bind.ContractBackend
	HeaderByNumber(context.Context, *big.Int) (*ethtypes.Header, error)
	CollateralBridgeAddress() ethcommon.Address
	CurrentHeight(context.Context) (uint64, error)
	ConfirmationsRequired() uint64
}

type ERC20 struct {
	asset     *types.Asset
	address   string
	chainID   string
	ok        bool
	wallet    ethnw.EthereumWallet
	ethClient ETHClient
}

func New(
	id string,
	asset *types.AssetDetails,
	w ethnw.EthereumWallet,
	ethClient ETHClient,
) (*ERC20, error) {
	source := asset.GetERC20()
	if source == nil {
		return nil, ErrNotAnErc20Asset
	}

	return &ERC20{
		asset: &types.Asset{
			ID:      id,
			Details: asset,
			Status:  types.AssetStatusProposed,
		},
		chainID:   source.ChainID,
		address:   source.ContractAddress,
		wallet:    w,
		ethClient: ethClient,
	}, nil
}

func (e *ERC20) SetPendingListing() {
	e.asset.Status = types.AssetStatusPendingListing
}

func (e *ERC20) SetRejected() {
	e.asset.Status = types.AssetStatusRejected
}

func (e *ERC20) SetEnabled() {
	e.asset.Status = types.AssetStatusEnabled
}

func (e *ERC20) Update(updatedAsset *types.Asset) {
	e.asset = updatedAsset
}

func (e *ERC20) Address() string {
	return e.address
}

func (e *ERC20) ProtoAsset() *typespb.Asset {
	return e.asset.IntoProto()
}

func (e ERC20) Type() *types.Asset {
	return e.asset.DeepClone()
}

func (e *ERC20) GetAssetClass() common.AssetClass {
	return common.ERC20
}

func (e *ERC20) IsValid() bool {
	return e.ok
}

func (e *ERC20) SetValid() {
	e.ok = true
}

// SignListAsset create and sign the message to
// be sent to the bridge to whitelist the asset
// return the generated message and the signature for this message.
func (e *ERC20) SignListAsset() (msg []byte, sig []byte, err error) {
	bridgeAddress := e.ethClient.CollateralBridgeAddress().Hex()
	// use the asset ID converted into a uint256
	// trim left all 0 as these makes for an invalid base16 numbers
	nonce, err := num.UintFromHex("0x" + strings.TrimLeft(e.asset.ID, "0"))
	if err != nil {
		return nil, nil, err
	}

	source := e.asset.Details.GetERC20()
	bundle, err := bridges.NewERC20Logic(e.wallet, bridgeAddress).
		ListAsset(e.address, e.asset.ID, source.LifetimeLimit, source.WithdrawThreshold, nonce)
	if err != nil {
		return nil, nil, err
	}

	return bundle.Message, bundle.Signature, nil
}

func (e *ERC20) SignSetAssetLimits(nonce *num.Uint, lifetimeLimit *num.Uint, withdrawThreshold *num.Uint) (msg []byte, sig []byte, err error) {
	bridgeAddress := e.ethClient.CollateralBridgeAddress().Hex()
	bundle, err := bridges.NewERC20Logic(e.wallet, bridgeAddress).
		SetAssetLimits(e.address, lifetimeLimit, withdrawThreshold, nonce)
	if err != nil {
		return nil, nil, err
	}

	return bundle.Message, bundle.Signature, nil
}

func (e *ERC20) SignWithdrawal(
	amount *num.Uint,
	ethPartyAddress string,
	withdrawRef *big.Int,
	now time.Time,
) (msg []byte, sig []byte, err error) {
	nonce, _ := num.UintFromBig(withdrawRef)
	bridgeAddress := e.ethClient.CollateralBridgeAddress().Hex()
	bundle, err := bridges.NewERC20Logic(e.wallet, bridgeAddress).
		WithdrawAsset(e.address, amount, ethPartyAddress, now, nonce)
	if err != nil {
		return nil, nil, err
	}

	return bundle.Message, bundle.Signature, nil
}

func (e *ERC20) String() string {
	return fmt.Sprintf("id(%v) name(%v) symbol(%v) decimals(%v)",
		e.asset.ID, e.asset.Details.Name, e.asset.Details.Symbol, e.asset.Details.Decimals)
}
