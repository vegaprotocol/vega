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

package banking

import (
	"errors"

	"code.vegaprotocol.io/vega/core/assets"
	"code.vegaprotocol.io/vega/core/assets/common"
	"code.vegaprotocol.io/vega/core/types"
	snapshot "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"
)

var ErrUnknownAssetAction = errors.New("unknown asset action")

type assetAction struct {
	id    string
	state uint32
	asset *assets.Asset

	// erc20 specifics
	blockNumber uint64
	txIndex     uint64
	hash        string

	// all deposit related types
	builtinD *types.BuiltinAssetDeposit
	erc20D   *types.ERC20Deposit
	erc20AL  *types.ERC20AssetList

	erc20AssetLimitsUpdated *types.ERC20AssetLimitsUpdated

	erc20BridgeStopped *types.ERC20EventBridgeStopped
	erc20BridgeResumed *types.ERC20EventBridgeResumed

	bridgeView ERC20BridgeView
}

func (t *assetAction) GetID() string {
	return t.id
}

func (t *assetAction) IsBuiltinAssetDeposit() bool {
	return t.builtinD != nil
}

func (t *assetAction) IsERC20BridgeStopped() bool {
	return t.erc20BridgeStopped != nil
}

func (t *assetAction) IsERC20BridgeResumed() bool {
	return t.erc20BridgeResumed != nil
}

func (t *assetAction) IsERC20Deposit() bool {
	return t.erc20D != nil
}

func (t *assetAction) IsERC20AssetLimitsUpdated() bool {
	return t.erc20AssetLimitsUpdated != nil
}

func (t *assetAction) IsERC20AssetList() bool {
	return t.erc20AL != nil
}

func (t *assetAction) BuiltinAssetDesposit() *types.BuiltinAssetDeposit {
	return t.builtinD
}

func (t *assetAction) ERC20Deposit() *types.ERC20Deposit {
	return t.erc20D
}

func (t *assetAction) ERC20AssetLimitsUpdated() *types.ERC20AssetLimitsUpdated {
	return t.erc20AssetLimitsUpdated
}

func (t *assetAction) ERC20AssetList() *types.ERC20AssetList {
	return t.erc20AL
}

func (t *assetAction) String() string {
	switch {
	case t.IsBuiltinAssetDeposit():
		return t.builtinD.String()
	case t.IsERC20Deposit():
		return t.erc20D.String()
	case t.IsERC20AssetList():
		return t.erc20AL.String()
	case t.IsERC20AssetLimitsUpdated():
		return t.erc20AssetLimitsUpdated.String()
	case t.IsERC20BridgeStopped():
		return t.erc20BridgeStopped.String()
	case t.IsERC20BridgeResumed():
		return t.erc20BridgeResumed.String()
	default:
		return ""
	}
}

func (t *assetAction) Check() error {
	switch {
	case t.IsBuiltinAssetDeposit():
		return t.checkBuiltinAssetDeposit()
	case t.IsERC20Deposit():
		return t.checkERC20Deposit()
	case t.IsERC20AssetList():
		return t.checkERC20AssetList()
	case t.IsERC20AssetLimitsUpdated():
		return t.checkERC20AssetLimitsUpdated()
	case t.IsERC20BridgeStopped():
		return t.checkERC20BridgeStopped()
	case t.IsERC20BridgeResumed():
		return t.checkERC20BridgeResumed()
	default:
		return ErrUnknownAssetAction
	}
}

func (t *assetAction) checkBuiltinAssetDeposit() error {
	return nil
}

func (t *assetAction) checkERC20BridgeStopped() error {
	return t.bridgeView.FindBridgeStopped(
		t.erc20BridgeStopped, t.blockNumber, t.txIndex)
}

func (t *assetAction) checkERC20BridgeResumed() error {
	return t.bridgeView.FindBridgeResumed(
		t.erc20BridgeResumed, t.blockNumber, t.txIndex)
}

func (t *assetAction) checkERC20Deposit() error {
	asset, _ := t.asset.ERC20()
	return asset.ValidateDeposit(t.erc20D, t.blockNumber, t.txIndex)
}

func (t *assetAction) checkERC20AssetList() error {
	asset, _ := t.asset.ERC20()
	return asset.ValidateAssetList(t.erc20AL, t.blockNumber, t.txIndex)
}

func (t *assetAction) checkERC20AssetLimitsUpdated() error {
	asset, _ := t.asset.ERC20()
	return asset.ValidateAssetLimitsUpdated(t.erc20AssetLimitsUpdated, t.blockNumber, t.txIndex)
}

func (t *assetAction) getRef() snapshot.TxRef {
	switch {
	case t.IsBuiltinAssetDeposit():
		return snapshot.TxRef{Asset: string(common.Builtin), BlockNr: 0, Hash: t.hash, LogIndex: 0}
	case t.IsERC20Deposit():
		return snapshot.TxRef{Asset: string(common.ERC20), BlockNr: t.blockNumber, Hash: t.hash, LogIndex: t.txIndex}
	case t.IsERC20AssetList():
		return snapshot.TxRef{Asset: string(common.ERC20), BlockNr: t.blockNumber, Hash: t.hash, LogIndex: t.txIndex}
	case t.IsERC20AssetLimitsUpdated():
		return snapshot.TxRef{Asset: string(common.ERC20), BlockNr: t.blockNumber, Hash: t.hash, LogIndex: t.txIndex}
	case t.IsERC20BridgeStopped():
		return snapshot.TxRef{Asset: string(common.ERC20), BlockNr: t.blockNumber, Hash: t.hash, LogIndex: t.txIndex}
	case t.IsERC20BridgeResumed():
		return snapshot.TxRef{Asset: string(common.ERC20), BlockNr: t.blockNumber, Hash: t.hash, LogIndex: t.txIndex}
	default:
		return snapshot.TxRef{} // this is basically unreachable
	}
}
