package evtforward

import (
	"context"
	"sort"

	commandspb "code.vegaprotocol.io/protos/vega/commands/v1"
	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/types"

	"github.com/golang/protobuf/proto"
)

var (
	key = (&types.PayloadEventForwarder{}).Key()

	hashKeys = []string{
		key,
	}
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
	return proto.Marshal(payload.IntoProto())
}

// get the serialised form and hash of the given key.
func (e *EvtForwarder) getSerialisedAndHash(k string) (data []byte, hash []byte, err error) {
	if k != key {
		return nil, nil, types.ErrSnapshotKeyDoesNotExist
	}

	if !e.efss.changed {
		return e.efss.serialised, e.efss.hash, nil
	}

	e.efss.serialised, err = e.serialise()
	if err != nil {
		return nil, nil, err
	}

	e.efss.hash = crypto.Hash(e.efss.serialised)
	e.efss.changed = false
	return e.efss.serialised, e.efss.hash, nil
}

func (e *EvtForwarder) GetHash(k string) ([]byte, error) {
	_, hash, err := e.getSerialisedAndHash(k)
	return hash, err
}

func (e *EvtForwarder) GetState(k string) ([]byte, []types.StateProvider, error) {
	state, _, err := e.getSerialisedAndHash(k)
	return state, nil, err
}

func (e *EvtForwarder) LoadState(ctx context.Context, p *types.Payload) ([]types.StateProvider, error) {
	if e.Namespace() != p.Data.Namespace() {
		return nil, types.ErrInvalidSnapshotNamespace
	}
	// see what we're reloading
	if pl, ok := p.Data.(*types.PayloadEventForwarder); ok {
		return nil, e.restore(ctx, pl.Events)
	}

	return nil, types.ErrUnknownSnapshotType
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
