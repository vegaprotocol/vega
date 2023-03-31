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
