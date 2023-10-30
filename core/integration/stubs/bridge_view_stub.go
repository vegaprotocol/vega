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

package stubs

import (
	"code.vegaprotocol.io/vega/core/types"
)

type BridgeViewStub struct{}

func NewBridgeViewStub() *BridgeViewStub {
	return &BridgeViewStub{}
}

func (*BridgeViewStub) FindAssetList(al *types.ERC20AssetList, blockNumber, logIndex uint64, txHash string) error {
	return nil
}

func (*BridgeViewStub) FindBridgeStopped(al *types.ERC20EventBridgeStopped, blockNumber, logIndex uint64, txHash string) error {
	return nil
}

func (*BridgeViewStub) FindBridgeResumed(al *types.ERC20EventBridgeResumed, blockNumber, logIndex uint64, txHash string) error {
	return nil
}

func (*BridgeViewStub) FindDeposit(d *types.ERC20Deposit, blockNumber, logIndex uint64, ethAssetAddress string, txHash string) error {
	return nil
}

func (*BridgeViewStub) FindAssetLimitsUpdated(w *types.ERC20AssetLimitsUpdated, blockNumber, logIndex uint64, ethAssetAddress string, txHash string) error {
	return nil
}
