package evtforward

import (
	"context"
	"errors"
	"sort"

	commandspb "code.vegaprotocol.io/protos/vega/commands/v1"
	"github.com/golang/protobuf/proto"

	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/types"
)

var (
	key = (&types.PayloadEventForwarder{}).Key()

	hashKeys = []string{
		key,
	}

	ErrSnapshotKeyDoesNotExist = errors.New("unknown key for event forwarder snapshot")
)

type efSnapshotState struct {
	changed    bool
	hash       []byte
	serialised []byte
}

func (ef *EvtForwarder) Namespace() types.SnapshotNamespace {
	return types.EventForwarderSnapshot
}

func (ef *EvtForwarder) Keys() []string {
	return hashKeys
}

func (ef *EvtForwarder) serialise() ([]byte, error) {
	// this is done without the lock because nothing can be acked during the commit phase which is when the snapshot is taken
	keys := make([]string, 0, len(ef.ackedEvts))
	events := make([]*commandspb.ChainEvent, 0, len(ef.ackedEvts))
	for key := range ef.ackedEvts {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		events = append(events, ef.ackedEvts[key])
	}

	payload := types.Payload{
		Data: &types.PayloadEventForwarder{
			Events: events,
		},
	}
	x := payload.IntoProto()
	return proto.Marshal(x)
}

// get the serialised form and hash of the given key.
func (ef *EvtForwarder) getSerialisedAndHash(k string) ([]byte, []byte, error) {
	if k != key {
		return nil, nil, ErrSnapshotKeyDoesNotExist
	}

	if !ef.efss.changed {
		return ef.efss.serialised, ef.efss.hash, nil
	}

	data, err := ef.serialise()
	if err != nil {
		return nil, nil, err
	}

	hash := crypto.Hash(data)
	ef.efss.serialised = data
	ef.efss.hash = hash
	ef.efss.changed = false
	return data, hash, nil
}

func (ef *EvtForwarder) GetHash(k string) ([]byte, error) {
	_, hash, err := ef.getSerialisedAndHash(k)
	return hash, err
}

func (ef *EvtForwarder) GetState(k string) ([]byte, error) {
	state, _, err := ef.getSerialisedAndHash(k)
	return state, err
}

func (ef *EvtForwarder) Snapshot() (map[string][]byte, error) {
	r := make(map[string][]byte, len(hashKeys))
	for _, k := range hashKeys {
		state, err := ef.GetState(k)
		if err != nil {
			return nil, err
		}
		r[k] = state
	}
	return r, nil
}

func (ef *EvtForwarder) LoadState(ctx context.Context, p *types.Payload) error {
	if ef.Namespace() != p.Data.Namespace() {
		return types.ErrInvalidSnapshotNamespace
	}
	// see what we're reloading
	switch pl := p.Data.(type) {
	case *types.PayloadEventForwarder:
		return ef.restore(ctx, pl.Events)
	default:
		return types.ErrUnknownSnapshotType
	}
}

func (ef *EvtForwarder) restore(ctx context.Context, events []*commandspb.ChainEvent) error {
	ef.ackedEvts = map[string]*commandspb.ChainEvent{}
	for _, event := range events {
		key, err := ef.getEvtKey(event)
		if err != nil {
			return err
		}
		ef.ackedEvts[key] = event
	}

	ef.efss.changed = true
	return nil
}
