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

package banking

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"

	"code.vegaprotocol.io/vega/core/assets"
	"code.vegaprotocol.io/vega/core/assets/common"
	"code.vegaprotocol.io/vega/core/types"
	snapshot "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"
)

var ErrUnknownAssetAction = errors.New("unknown asset action")

type assetAction struct {
	id    string
	state *atomic.Uint32
	asset *assets.Asset

	// erc20 specifics
	blockHeight uint64
	logIndex    uint64
	txHash      string
	chainID     string

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

func (t *assetAction) GetType() types.NodeVoteType {
	switch {
	case t.IsBuiltinAssetDeposit():
		return types.NodeVoteTypeFundsDeposited
	case t.IsERC20Deposit():
		return types.NodeVoteTypeFundsDeposited
	case t.IsERC20AssetList():
		return types.NodeVoteTypeAssetListed
	case t.IsERC20AssetLimitsUpdated():
		return types.NodeVoteTypeAssetLimitsUpdated
	case t.IsERC20BridgeStopped():
		return types.NodeVoteTypeBridgeStopped
	case t.IsERC20BridgeResumed():
		return types.NodeVoteTypeBridgeResumed
	default:
		return types.NodeVoteTypeUnspecified
	}
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
		return fmt.Sprintf("builtinAssetDeposit(%s)", t.builtinD.String())
	case t.IsERC20Deposit():
		return fmt.Sprintf("erc20Deposit(%s)", t.erc20D.String())
	case t.IsERC20AssetList():
		return fmt.Sprintf("erc20AssetList(%s)", t.erc20AL.String())
	case t.IsERC20AssetLimitsUpdated():
		return fmt.Sprintf("erc20AssetLimitsUpdated(%s)", t.erc20AssetLimitsUpdated.String())
	case t.IsERC20BridgeStopped():
		return fmt.Sprintf("erc20BridgeStopped(%s)", t.erc20BridgeStopped.String())
	case t.IsERC20BridgeResumed():
		return fmt.Sprintf("erc20BridgeResumed(%s)", t.erc20BridgeResumed.String())
	default:
		return ""
	}
}

func (t *assetAction) Check(_ context.Context) error {
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
		t.erc20BridgeStopped, t.blockHeight, t.logIndex, t.txHash)
}

func (t *assetAction) checkERC20BridgeResumed() error {
	return t.bridgeView.FindBridgeResumed(
		t.erc20BridgeResumed, t.blockHeight, t.logIndex, t.txHash)
}

func (t *assetAction) checkERC20Deposit() error {
	asset, _ := t.asset.ERC20()
	return t.bridgeView.FindDeposit(
		t.erc20D, t.blockHeight, t.logIndex, asset.Address(), t.txHash,
	)
}

func (t *assetAction) checkERC20AssetList() error {
	return t.bridgeView.FindAssetList(t.erc20AL, t.blockHeight, t.logIndex, t.txHash)
}

func (t *assetAction) checkERC20AssetLimitsUpdated() error {
	asset, _ := t.asset.ERC20()
	return t.bridgeView.FindAssetLimitsUpdated(
		t.erc20AssetLimitsUpdated, t.blockHeight, t.logIndex, asset.Address(), t.txHash,
	)
}

func (t *assetAction) getRef() snapshot.TxRef {
	switch {
	case t.IsBuiltinAssetDeposit():
		return snapshot.TxRef{Asset: string(common.Builtin), BlockNr: 0, Hash: t.txHash, LogIndex: 0}
	case t.IsERC20Deposit():
		return snapshot.TxRef{Asset: string(common.ERC20), BlockNr: t.blockHeight, Hash: t.txHash, LogIndex: t.logIndex, ChainId: t.chainID}
	case t.IsERC20AssetList():
		return snapshot.TxRef{Asset: string(common.ERC20), BlockNr: t.blockHeight, Hash: t.txHash, LogIndex: t.logIndex, ChainId: t.chainID}
	case t.IsERC20AssetLimitsUpdated():
		return snapshot.TxRef{Asset: string(common.ERC20), BlockNr: t.blockHeight, Hash: t.txHash, LogIndex: t.logIndex, ChainId: t.chainID}
	case t.IsERC20BridgeStopped():
		return snapshot.TxRef{Asset: string(common.ERC20), BlockNr: t.blockHeight, Hash: t.txHash, LogIndex: t.logIndex, ChainId: t.chainID}
	case t.IsERC20BridgeResumed():
		return snapshot.TxRef{Asset: string(common.ERC20), BlockNr: t.blockHeight, Hash: t.txHash, LogIndex: t.logIndex, ChainId: t.chainID}
	default:
		return snapshot.TxRef{} // this is basically unreachable
	}
}
