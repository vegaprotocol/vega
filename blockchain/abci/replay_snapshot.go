package abci

import (
	"context"
	"encoding/hex"
	"errors"
	"sort"

	snapshot "code.vegaprotocol.io/protos/vega/snapshot/v1"
	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/proto"
	"code.vegaprotocol.io/vega/types"
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

func (rp *ReplayProtector) Stopped() bool {
	return false
}

func (rp *ReplayProtector) serialiseReplayProtection() ([]byte, error) {
	txs := make([]*snapshot.TransactionAtHeight, 0, len(rp.txs))
	for k, v := range rp.txs {
		txs = append(txs, &snapshot.TransactionAtHeight{
			Tx:     hex.EncodeToString([]byte(k)),
			Height: v,
		})
	}
	sort.SliceStable(txs, func(i, j int) bool {
		return txs[i].Tx < txs[j].Tx
	})

	payload := types.Payload{
		Data: &types.PayloadReplayProtection{
			Transactions: txs,
			BackTol:      rp.backwardTol,
			ForwardTol:   rp.forwardTol,
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
		return nil, rp.restoreReplayState(ctx, pl)
	default:
		return nil, types.ErrUnknownSnapshotType
	}
}

func (rp *ReplayProtector) restoreReplayState(ctx context.Context, pl *types.PayloadReplayProtection) error {
	rp.backwardTol = pl.BackTol
	rp.forwardTol = pl.ForwardTol
	rp.txs = make(map[string]uint64, len(pl.Transactions))

	for _, tx := range pl.Transactions {
		// convert to byte slice that was cast to string
		// cast bytes as string, this can contain invalid UTF-8 characters,
		// which is why we need the hex.EncodeToString stuff.
		bs, err := hex.DecodeString(tx.Tx)
		if err != nil {
			return err
		}

		rp.txs[string(bs)] = tx.Height
	}

	rp.rss.changed = true
	return nil
}
