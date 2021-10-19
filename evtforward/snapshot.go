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

func (e *EvtForwarder) Namespace() types.SnapshotNamespace {
	return types.EventForwarderSnapshot
}

func (e *EvtForwarder) Keys() []string {
	return hashKeys
}

func (e *EvtForwarder) serialise() ([]byte, error) {
	// this is done without the lock because nothing can be acked during the commit phase which is when the snapshot is taken
	keys := make([]string, 0, len(e.ackedEvts))
	events := make([]*commandspb.ChainEvent, 0, len(e.ackedEvts))
	for key := range e.ackedEvts {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		events = append(events, e.ackedEvts[key])
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
func (e *EvtForwarder) getSerialisedAndHash(k string) ([]byte, []byte, error) {
	if k != key {
		return nil, nil, ErrSnapshotKeyDoesNotExist
	}

	if !e.efss.changed {
		return e.efss.serialised, e.efss.hash, nil
	}

	data, err := e.serialise()
	if err != nil {
		return nil, nil, err
	}

	hash := crypto.Hash(data)
	e.efss.serialised = data
	e.efss.hash = hash
	e.efss.changed = false
	return data, hash, nil
}

func (e *EvtForwarder) GetHash(k string) ([]byte, error) {
	_, hash, err := e.getSerialisedAndHash(k)
	return hash, err
}

func (e *EvtForwarder) GetState(k string) ([]byte, error) {
	state, _, err := e.getSerialisedAndHash(k)
	return state, err
}

func (e *EvtForwarder) Snapshot() (map[string][]byte, error) {
	r := make(map[string][]byte, len(hashKeys))
	for _, k := range hashKeys {
		state, err := e.GetState(k)
		if err != nil {
			return nil, err
		}
		r[k] = state
	}
	return r, nil
}

func (e *EvtForwarder) LoadState(ctx context.Context, p *types.Payload) error {
	if e.Namespace() != p.Data.Namespace() {
		return types.ErrInvalidSnapshotNamespace
	}
	// see what we're reloading
	switch pl := p.Data.(type) {
	case *types.PayloadEventForwarder:
		return e.restore(ctx, pl.Events)
	default:
		return types.ErrUnknownSnapshotType
	}
}

func (e *EvtForwarder) restore(ctx context.Context, events []*commandspb.ChainEvent) error {
	e.ackedEvts = map[string]*commandspb.ChainEvent{}
	for _, event := range events {
		key, err := e.getEvtKey(event)
		if err != nil {
			return err
		}
		e.ackedEvts[key] = event
	}

	e.efss.changed = true
	return nil
}
