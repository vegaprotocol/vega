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

package txcache

import (
	"context"
	"encoding/hex"
	"fmt"
	"sort"
	"sync"

	"code.vegaprotocol.io/vega/core/nodewallets"
	"code.vegaprotocol.io/vega/core/txn"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/proto"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	snapshotpb "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"

	"github.com/cometbft/cometbft/crypto/tmhash"
)

func NewTxCache(commander *nodewallets.Commander) *TxCache {
	return &TxCache{
		commander:             commander,
		marketToDelayRequired: map[string]bool{},
		heightToTxs:           map[uint64][][]byte{},
		cachedTxs:             map[string]struct{}{},
	}
}

type TxCache struct {
	commander   *nodewallets.Commander
	heightToTxs map[uint64][][]byte
	// network param
	numBlocksToDelay uint64
	// no need to include is snapshot - is updated when markets are created/updated/loaded from snapshot
	marketToDelayRequired map[string]bool
	// map of transactions that have been picked up for delay
	cachedTxs map[string]struct{}
	lock      sync.RWMutex
}

func (t *TxCache) IsTxInCache(txHash []byte) bool {
	t.lock.RLock()
	defer t.lock.RUnlock()
	_, ok := t.cachedTxs[hex.EncodeToString(txHash)]
	return ok
}

func (t *TxCache) removeHeightFromCache(height uint64) {
	t.lock.Lock()
	defer t.lock.Unlock()
	if txs, ok := t.heightToTxs[height]; ok {
		for _, tx := range txs {
			delete(t.cachedTxs, hex.EncodeToString(tmhash.Sum(tx)))
		}
	}
}

func (t *TxCache) addHeightToCache(txs [][]byte) {
	t.lock.Lock()
	defer t.lock.Unlock()
	for _, tx := range txs {
		t.cachedTxs[hex.EncodeToString(tmhash.Sum(tx))] = struct{}{}
	}
}

// MarketDelayRequiredUpdated is called when the market configuration is created/updated with support for
// transaction reordering.
func (t *TxCache) MarketDelayRequiredUpdated(marketID string, required bool) {
	t.marketToDelayRequired[marketID] = required
}

// IsDelayRequired returns true if the market supports transaction reordering.
func (t *TxCache) IsDelayRequired(marketID string) bool {
	delay, ok := t.marketToDelayRequired[marketID]
	return ok && delay
}

// IsDelayRequiredAnyMarket returns true of there is any market that supports transaction reordering.
func (t *TxCache) IsDelayRequiredAnyMarket() bool {
	return len(t.marketToDelayRequired) > 0
}

// OnNumBlocksToDelayUpdated is called when the network parameter for the number of blocks to delay
// transactions is updated.
func (t *TxCache) OnNumBlocksToDelayUpdated(_ context.Context, blocks *num.Uint) error {
	t.numBlocksToDelay = blocks.Uint64()
	return nil
}

// NewDelayedTransaction creates a new delayed transaction with a target block height being the current
// block being proposed + the configured network param indicating the target delay.
func (t *TxCache) NewDelayedTransaction(ctx context.Context, delayed [][]byte, currentHeight uint64) []byte {
	height := currentHeight + t.numBlocksToDelay
	payload := &commandspb.DelayedTransactionsWrapper{Transactions: delayed, Height: height}
	tx, err := t.commander.NewTransaction(ctx, txn.DelayedTransactionsWrapper, payload)
	if err != nil {
		panic(err.Error())
	}
	return tx
}

func (t *TxCache) SetRawTxs(rtx [][]byte, height uint64) {
	if rtx == nil {
		t.removeHeightFromCache(height)
		delete(t.heightToTxs, height)
	} else {
		t.heightToTxs[height] = rtx
		t.addHeightToCache(rtx)
	}
}

func (t *TxCache) GetRawTxs(height uint64) [][]byte {
	return t.heightToTxs[height]
}

func (t *TxCache) Namespace() types.SnapshotNamespace {
	return types.TxCacheSnapshot
}

func (t *TxCache) Keys() []string {
	return []string{(&types.PayloadTxCache{}).Key()}
}

func (t *TxCache) GetState(k string) ([]byte, []types.StateProvider, error) {
	delays := make([]*snapshotpb.DelayedTx, 0, len(t.heightToTxs))
	for delay, txs := range t.heightToTxs {
		delays = append(delays, &snapshotpb.DelayedTx{
			Height: delay,
			Tx:     txs,
		})
	}
	sort.Slice(delays, func(i, j int) bool {
		return delays[i].Height < delays[j].Height
	})

	payload := &snapshotpb.Payload{
		Data: &snapshotpb.Payload_TxCache{
			TxCache: &snapshotpb.TxCache{
				Txs: delays,
			},
		},
	}

	serialised, err := proto.Marshal(payload)
	if err != nil {
		return nil, nil, fmt.Errorf("could not serialize tx cache payload: %w", err)
	}
	return serialised, nil, err
}

func (t *TxCache) LoadState(_ context.Context, p *types.Payload) ([]types.StateProvider, error) {
	if t.Namespace() != p.Data.Namespace() {
		return nil, types.ErrInvalidSnapshotNamespace
	}

	switch data := p.Data.(type) {
	case *types.PayloadTxCache:
		t.heightToTxs = map[uint64][][]byte{}
		for _, tx := range data.TxCache.Txs {
			t.heightToTxs[tx.Height] = tx.Tx
			t.addHeightToCache(tx.Tx)
		}
		return nil, nil
	default:
		return nil, types.ErrUnknownSnapshotType
	}
}

func (t *TxCache) Stopped() bool {
	return false
}

func (e *TxCache) StopSnapshots() {}
