package abci

import (
	"context"
	"errors"
	"sort"

	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/types"
	"github.com/golang/protobuf/proto"
)

var (
	key = (&types.PayloadReplayProtection{}).Key()

	hashKeys = []string{
		key,
	}

	ErrSnapshotKeyDoesNotExist = errors.New("unknown key for replay protection snapshot")
)

type replaySnapshotState struct {
	changed    bool
	hash       []byte
	serialised []byte
}

func (rp *ReplayProtector) Namespace() types.SnapshotNamespace {
	return types.ReplayProtectionSnapshot
}

func (rp *ReplayProtector) Keys() []string {
	return hashKeys
}

func (rp *ReplayProtector) serialiseReplayProtection() ([]byte, error) {
	blocks := []*types.ReplayBlockTransactions{}
	for _, block := range rp.txs {
		txs := make([]string, 0, len(block))
		for tx := range block {
			txs = append(txs, tx)
		}
		sort.Strings(txs)
		blocks = append(blocks, &types.ReplayBlockTransactions{Transactions: txs})
	}

	payload := types.Payload{
		Data: &types.PayloadReplayProtection{
			Blocks: blocks,
		},
	}
	x := payload.IntoProto()
	return proto.Marshal(x)
}

// get the serialised form and hash of the given key.
func (rp *ReplayProtector) getSerialisedAndHash(k string) ([]byte, []byte, error) {
	if k != key {
		return nil, nil, ErrSnapshotKeyDoesNotExist
	}

	if !rp.rss.changed {
		return rp.rss.serialised, rp.rss.hash, nil
	}

	data, err := rp.serialiseReplayProtection()
	if err != nil {
		return nil, nil, err
	}

	hash := crypto.Hash(data)
	rp.rss.serialised = data
	rp.rss.hash = hash
	rp.rss.changed = false
	return data, hash, nil
}

func (rp *ReplayProtector) GetHash(k string) ([]byte, error) {
	_, hash, err := rp.getSerialisedAndHash(k)
	return hash, err
}

func (rp *ReplayProtector) GetState(k string) ([]byte, error) {
	state, _, err := rp.getSerialisedAndHash(k)
	return state, err
}

func (rp *ReplayProtector) Snapshot() (map[string][]byte, error) {
	r := make(map[string][]byte, len(hashKeys))
	for _, k := range hashKeys {
		state, err := rp.GetState(k)
		if err != nil {
			return nil, err
		}
		r[k] = state
	}
	return r, nil
}

func (rp *ReplayProtector) LoadState(ctx context.Context, p *types.Payload) ([]types.StateProvider, error) {
	if rp.Namespace() != p.Data.Namespace() {
		return nil, types.ErrInvalidSnapshotNamespace
	}
	var err error
	// see what we're reloading
	switch pl := p.Data.(type) {
	case *types.PayloadReplayProtection:
		err = rp.restoreReplayState(ctx, pl.Blocks)
	default:
		err = types.ErrUnknownSnapshotType
	}
	return nil, err
}

func (rp *ReplayProtector) restoreReplayState(ctx context.Context, blockTransactions []*types.ReplayBlockTransactions) error {
	for i := range rp.txs {
		rp.txs[i] = make(map[string]struct{})
	}

	for i, block := range blockTransactions {
		for _, tx := range block.Transactions {
			rp.txs[i][tx] = struct{}{}
		}
	}

	rp.rss.changed = true
	return nil
}
