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

package stubs

import (
	"code.vegaprotocol.io/vega/core/types"
)

type BridgeViewStub struct{}

func NewBridgeViewStub() *BridgeViewStub {
	return &BridgeViewStub{}
}

func (*BridgeViewStub) FindAssetList(al *types.ERC20AssetList, blockNumber, logIndex uint64) error {
	return nil
}

func (*BridgeViewStub) FindBridgeStopped(al *types.ERC20EventBridgeStopped, blockNumber, logIndex uint64) error {
	return nil
}

func (*BridgeViewStub) FindBridgeResumed(al *types.ERC20EventBridgeResumed, blockNumber, logIndex uint64) error {
	return nil
}

func (*BridgeViewStub) FindDeposit(d *types.ERC20Deposit, blockNumber, logIndex uint64, ethAssetAddress string) error {
	return nil
}

func (*BridgeViewStub) FindAssetLimitsUpdated(w *types.ERC20AssetLimitsUpdated, blockNumber, logIndex uint64, ethAssetAddress string) error {
	return nil
}
