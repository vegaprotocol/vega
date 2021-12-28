package abci

import (
	"context"
	"encoding/hex"
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
			// tx is []byte cast to string, can contain invalid UTF-8 characters.
			// we need to encode this properly for proto marshalling to work.
			txs = append(txs, hex.EncodeToString([]byte(tx)))
		}
		sort.Strings(txs)
		blocks = append(blocks, &types.ReplayBlockTransactions{Transactions: txs})
	}

	payload := types.Payload{
		Data: &types.PayloadReplayProtection{
			Blocks: blocks,
		},
	}
	return proto.Marshal(payload.IntoProto())
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

func (rp *ReplayProtector) GetState(k string) ([]byte, []types.StateProvider, error) {
	state, _, err := rp.getSerialisedAndHash(k)
	return state, nil, err
}

func (rp *ReplayProtector) Snapshot() (map[string][]byte, error) {
	r := make(map[string][]byte, len(hashKeys))
	for _, k := range hashKeys {
		state, _, err := rp.GetState(k)
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
	// see what we're reloading
	switch pl := p.Data.(type) {
	case *types.PayloadReplayProtection:
		return nil, rp.restoreReplayState(ctx, pl.Blocks)
	default:
		return nil, types.ErrUnknownSnapshotType
	}
}

func (rp *ReplayProtector) restoreReplayState(ctx context.Context, blockTransactions []*types.ReplayBlockTransactions) error {
	for i, block := range blockTransactions {
		rp.txs[i] = make(map[string]struct{}, len(block.Transactions))
		for _, tx := range block.Transactions {
			// convert to byte slice that was cast to string
			bs, err := hex.DecodeString(tx)
			if err != nil {
				return err
			}
			// cast bytes as string, this can contain invalid UTF-8 characters,
			// which is why we need the hex.EncodeToString stuff.
			tx = string(bs)
			rp.txs[i][tx] = struct{}{}
		}
	}

	rp.rss.changed = true
	return nil
}
