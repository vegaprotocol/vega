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
	"fmt"

	"code.vegaprotocol.io/vega/core/nodewallets"
	"code.vegaprotocol.io/vega/core/txn"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/proto"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	snapshotpb "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"
)

func NewTxCache(commander *nodewallets.Commander) *TxCache {
	return &TxCache{
		commander: commander,
	}
}

type TxCache struct {
	commander *nodewallets.Commander
	rtxs      [][]byte
}

func (t *TxCache) NewDelayedTransaction(ctx context.Context, delayed [][]byte) []byte {
	payload := &commandspb.DelayedTransactionsWrapper{Transactions: delayed}
	tx, err := t.commander.NewTransaction(ctx, txn.DelayedTransactionsWrapper, payload)
	if err != nil {
		panic(err.Error())
	}
	return tx
}

func (t *TxCache) SetRawTxs(rtx [][]byte) {
	t.rtxs = rtx
}

func (t *TxCache) GetRawTxs() [][]byte {
	return t.rtxs
}

func (t *TxCache) Namespace() types.SnapshotNamespace {
	return types.TxCacheSnapshot
}

func (t *TxCache) Keys() []string {
	return []string{(&types.PayloadTxCache{}).Key()}
}

func (t *TxCache) GetState(k string) ([]byte, []types.StateProvider, error) {
	payload := &snapshotpb.Payload{
		Data: &snapshotpb.Payload_TxCache{
			TxCache: &snapshotpb.TxCache{
				Txs: t.rtxs,
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
		t.rtxs = data.TxCache.Txs
		return nil, nil
	default:
		return nil, types.ErrUnknownSnapshotType
	}
}

func (t *TxCache) Stopped() bool {
	return false
}

func (e *TxCache) StopSnapshots() {}
