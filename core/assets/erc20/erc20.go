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

var (
	ErrUnableToFindDeposit                 = errors.New("unable to find erc20 deposit event")
	ErrUnableToFindWithdrawal              = errors.New("unable to find erc20 withdrawal event")
	ErrUnableToFindERC20AssetList          = errors.New("unable to find erc20 asset list event")
	ErrUnableToFindERC20AssetLimitsUpdated = errors.New("unable to find ERC20 asset limits updated event")
	ErrMissingConfirmations                = errors.New("missing confirmation from ethereum")
	ErrNotAnErc20Asset                     = errors.New("not an erc20 asset")
)

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
