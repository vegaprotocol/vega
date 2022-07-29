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
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"time"

	typespb "code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/core/assets/common"
	"code.vegaprotocol.io/vega/core/bridges"
	"code.vegaprotocol.io/vega/core/contracts/erc20"
	bridge "code.vegaprotocol.io/vega/core/contracts/erc20_bridge_logic_restricted"
	"code.vegaprotocol.io/vega/core/metrics"
	ethnw "code.vegaprotocol.io/vega/core/nodewallets/eth"
	"code.vegaprotocol.io/vega/core/types"
	vgerrors "code.vegaprotocol.io/vega/libs/errors"
	"code.vegaprotocol.io/vega/libs/num"

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
	wallet    *ethnw.Wallet
	ethClient ETHClient
}

func New(
	id string,
	asset *types.AssetDetails,
	w *ethnw.Wallet,
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

// SetValidNonValidator this method is here temporarsy
// to avoid requiring ethclient for the non-validators
// will be removed once the eth client can be removed from this type.
func (e *ERC20) SetValidNonValidator() {
	e.ok = true
}

func (e *ERC20) Validate() error {
	t, err := erc20.NewErc20(ethcommon.HexToAddress(e.address), e.ethClient)
	if err != nil {
		return err
	}

	validationErrs := vgerrors.NewCumulatedErrors()

	if name, err := t.Name(&bind.CallOpts{}); err != nil {
		validationErrs.Add(fmt.Errorf("couldn't get name: %w", err))
	} else if name != e.asset.Details.Name {
		validationErrs.Add(fmt.Errorf("invalid name, expected(%s), got(%s)", e.asset.Details.Name, name))
	}

	if symbol, err := t.Symbol(&bind.CallOpts{}); err != nil {
		validationErrs.Add(fmt.Errorf("couldn't get symbol: %w", err))
	} else if symbol != e.asset.Details.Symbol {
		validationErrs.Add(fmt.Errorf("invalid symbol, expected(%s), got(%s)", e.asset.Details.Symbol, symbol))
	}

	if decimals, err := t.Decimals(&bind.CallOpts{}); err != nil {
		validationErrs.Add(fmt.Errorf("couldn't get decimals: %w", err))
	} else if uint64(decimals) != e.asset.Details.Decimals {
		validationErrs.Add(fmt.Errorf("invalid decimals, expected(%d), got(%d)", e.asset.Details.Decimals, decimals))
	}

	// FIXME: We do not check the total supply for now.
	// It's for normal asset never really used, and will also vary
	// if new coins are minted...
	// if totalSupply, err := t.TotalSupply(&bind.CallOpts{}); err != nil {
	// 	carryErr = fmt.Errorf("couldn't get totalSupply %v: %w", err, carryErr)
	// } else if totalSupply.String() != b.asset.Details.TotalSupply {
	// 	carryErr = maybeError(carryErr, "invalid symbol, expected(%s), got(%s)", b.asset.Details.TotalSupply, totalSupply)
	// }

	if validationErrs.HasAny() {
		return validationErrs
	}

	e.ok = true
	return nil
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

func (e *ERC20) ValidateAssetList(w *types.ERC20AssetList, blockNumber, txIndex uint64) error {
	bf, err := bridge.NewErc20BridgeLogicRestrictedFilterer(
		e.ethClient.CollateralBridgeAddress(), e.ethClient)
	if err != nil {
		return err
	}

	resp := "ok"
	defer func() {
		metrics.EthCallInc("validate_allowlist", e.asset.ID, resp)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	iter, err := bf.FilterAssetListed(
		&bind.FilterOpts{
			Start:   blockNumber - 1,
			Context: ctx,
		},
		[]ethcommon.Address{ethcommon.HexToAddress(e.address)},
		[][32]byte{},
	)
	if err != nil {
		resp = getMaybeHTTPStatus(err)
		return err
	}

	defer iter.Close()
	var event *bridge.Erc20BridgeLogicRestrictedAssetListed

	assetID := strings.TrimPrefix(w.VegaAssetID, "0x")
	for iter.Next() {
		if hex.EncodeToString(iter.Event.VegaAssetId[:]) == assetID &&
			iter.Event.Raw.BlockNumber == blockNumber &&
			uint64(iter.Event.Raw.Index) == txIndex {
			event = iter.Event

			break
		}
	}

	if event == nil {
		return ErrUnableToFindERC20AssetList
	}

	// now ensure we have enough confirmations
	if err := e.checkConfirmations(event.Raw.BlockNumber); err != nil {
		return err
	}

	return nil
}

func (e *ERC20) ValidateAssetLimitsUpdated(update *types.ERC20AssetLimitsUpdated, blockNumber uint64, txIndex uint64) error {
	bf, err := bridge.NewErc20BridgeLogicRestrictedFilterer(e.ethClient.CollateralBridgeAddress(), e.ethClient)
	if err != nil {
		return err
	}

	resp := "ok"
	defer func() {
		metrics.EthCallInc("validate_asset_limits_updated", e.asset.ID, resp)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	iter, err := bf.FilterAssetLimitsUpdated(
		&bind.FilterOpts{
			Start:   blockNumber - 1,
			Context: ctx,
		},
		[]ethcommon.Address{ethcommon.HexToAddress(e.address)},
	)
	if err != nil {
		resp = getMaybeHTTPStatus(err)
		return err
	}

	defer iter.Close()
	var event *bridge.Erc20BridgeLogicRestrictedAssetLimitsUpdated
	for iter.Next() {
		eventLifetimeLimit, _ := num.UintFromBig(iter.Event.LifetimeLimit)
		eventWithdrawThreshold, _ := num.UintFromBig(iter.Event.WithdrawThreshold)
		if update.LifetimeLimits.EQ(eventLifetimeLimit) &&
			update.WithdrawThreshold.EQ(eventWithdrawThreshold) &&
			iter.Event.Raw.BlockNumber == blockNumber &&
			uint64(iter.Event.Raw.Index) == txIndex {
			event = iter.Event
			break
		}
	}

	if event == nil {
		return ErrUnableToFindERC20AssetLimitsUpdated
	}

	// now ensure we have enough confirmations
	if err := e.checkConfirmations(event.Raw.BlockNumber); err != nil {
		return err
	}

	return nil
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

func (e *ERC20) ValidateWithdrawal(w *types.ERC20Withdrawal, blockNumber, txIndex uint64) (*big.Int, string, uint, error) {
	bf, err := bridge.NewErc20BridgeLogicRestrictedFilterer(
		e.ethClient.CollateralBridgeAddress(), e.ethClient)
	if err != nil {
		return nil, "", 0, err
	}

	resp := "ok"
	defer func() {
		metrics.EthCallInc("validate_withdrawal", e.asset.ID, resp)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	iter, err := bf.FilterAssetWithdrawn(
		&bind.FilterOpts{
			Start:   blockNumber - 1,
			Context: ctx,
		},
		// user_address
		[]ethcommon.Address{ethcommon.HexToAddress(w.TargetEthereumAddress)},
		// asset_source
		[]ethcommon.Address{ethcommon.HexToAddress(e.address)})
	if err != nil {
		resp = getMaybeHTTPStatus(err)
		return nil, "", 0, err
	}

	defer iter.Close()
	var event *bridge.Erc20BridgeLogicRestrictedAssetWithdrawn
	nonce := &big.Int{}
	_, ok := nonce.SetString(w.ReferenceNonce, 10)
	if !ok {
		return nil, "", 0, fmt.Errorf("could not use reference nonce, expected base 10 integer: %v", w.ReferenceNonce)
	}
	for iter.Next() {
		if nonce.Cmp(iter.Event.Nonce) == 0 &&
			iter.Event.Raw.BlockNumber == blockNumber &&
			uint64(iter.Event.Raw.Index) == txIndex {
			event = iter.Event

			break
		}
	}

	if event == nil {
		return nil, "", 0, ErrUnableToFindWithdrawal
	}

	// now ensure we have enough confirmations
	if err := e.checkConfirmations(event.Raw.BlockNumber); err != nil {
		return nil, "", 0, err
	}

	return nonce, event.Raw.TxHash.Hex(), event.Raw.Index, nil
}

func (e *ERC20) ValidateDeposit(d *types.ERC20Deposit, blockNumber, txIndex uint64) error {
	bf, err := bridge.NewErc20BridgeLogicRestrictedFilterer(
		e.ethClient.CollateralBridgeAddress(), e.ethClient)
	if err != nil {
		return err
	}

	resp := "ok"
	defer func() {
		metrics.EthCallInc("validate_deposit", e.asset.ID, resp)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	iter, err := bf.FilterAssetDeposited(
		&bind.FilterOpts{
			Start:   blockNumber - 1,
			Context: ctx,
		},
		// user_address
		[]ethcommon.Address{ethcommon.HexToAddress(d.SourceEthereumAddress)},
		// asset_source
		[]ethcommon.Address{ethcommon.HexToAddress(e.address)})
	if err != nil {
		resp = getMaybeHTTPStatus(err)
		return err
	}

	depamount := d.Amount.BigInt()
	defer iter.Close()
	var event *bridge.Erc20BridgeLogicRestrictedAssetDeposited
	targetPartyID := strings.TrimPrefix(d.TargetPartyID, "0x")
	for iter.Next() {
		if hex.EncodeToString(iter.Event.VegaPublicKey[:]) == targetPartyID &&
			iter.Event.Amount.Cmp(depamount) == 0 &&
			iter.Event.Raw.BlockNumber == blockNumber &&
			uint64(iter.Event.Raw.Index) == txIndex {
			event = iter.Event
			break
		}
	}

	if event == nil {
		return ErrUnableToFindDeposit
	}

	// now ensure we have enough confirmations
	if err := e.checkConfirmations(event.Raw.BlockNumber); err != nil {
		return err
	}

	return nil
}

func (e *ERC20) checkConfirmations(txBlock uint64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	curBlock, err := e.ethClient.CurrentHeight(ctx)
	if err != nil {
		return err
	}

	if curBlock < txBlock || (curBlock-txBlock) < e.ethClient.ConfirmationsRequired() {
		return ErrMissingConfirmations
	}

	return nil
}

func (e *ERC20) String() string {
	return fmt.Sprintf("id(%v) name(%v) symbol(%v) totalSupply(%v) decimals(%v)",
		e.asset.ID, e.asset.Details.Name, e.asset.Details.Symbol, e.asset.Details.TotalSupply,
		e.asset.Details.Decimals)
}

func getMaybeHTTPStatus(err error) string {
	errstr := err.Error()
	if len(errstr) < 3 {
		return "tooshort"
	}
	i, err := strconv.Atoi(errstr[:3])
	if err != nil {
		return "nan"
	}
	if http.StatusText(i) == "" {
		return "unknown"
	}

	return errstr[:3]
}
