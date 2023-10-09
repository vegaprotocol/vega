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

	"code.vegaprotocol.io/vega/core/types"
)

func (e *Engine) BridgeStopped(
	ctx context.Context,
	stopped bool,
	id string,
	block, logIndex uint64,
	ethTxHash string,
) error {
	aa := &assetAction{
		id:                 id,
		state:              newPendingState(),
		erc20BridgeStopped: &types.ERC20EventBridgeStopped{BridgeStopped: stopped},
		blockHeight:        block,
		logIndex:           logIndex,
		txHash:             ethTxHash,
		bridgeView:         e.bridgeView,
	}
	e.assetActs[aa.id] = aa
	return e.witness.StartCheck(aa, e.onCheckDone, e.timeService.GetTimeNow().Add(defaultValidationDuration))
}

func (e *Engine) BridgeResumed(
	ctx context.Context,
	resumed bool,
	id string,
	block, logIndex uint64,
	ethTxHash string,
) error {
	aa := &assetAction{
		id:                 id,
		state:              newPendingState(),
		erc20BridgeResumed: &types.ERC20EventBridgeResumed{BridgeResumed: resumed},
		blockHeight:        block,
		logIndex:           logIndex,
		txHash:             ethTxHash,
		bridgeView:         e.bridgeView,
	}
	e.assetActs[aa.id] = aa
	return e.witness.StartCheck(aa, e.onCheckDone, e.timeService.GetTimeNow().Add(defaultValidationDuration))
}

type bridgeState struct {
	// is the operation suspended, or as usual
	active bool
	// last block + log index we received an update from the bridge
	// this will be used later to verify no new state of the bridge is processed
	// in a wrong orderi
	block, logIndex uint64
}

func (b *bridgeState) IsStopped() bool {
	return !b.active
}

func (b *bridgeState) NewBridgeStopped(
	block, logIndex uint64,
) {
	if b.isNewerEvent(block, logIndex) {
		b.active, b.block, b.logIndex = false, block, logIndex
	}
}

func (b *bridgeState) NewBridgeResumed(
	block, logIndex uint64,
) {
	if b.isNewerEvent(block, logIndex) {
		b.active, b.block, b.logIndex = true, block, logIndex
	}
}

func (b *bridgeState) isNewerEvent(
	block, logIndex uint64,
) bool {
	if block == b.block {
		return logIndex > b.logIndex
	}
	return block > b.block
}
