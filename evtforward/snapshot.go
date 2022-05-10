package evtforward

import (
	"context"

	commandspb "code.vegaprotocol.io/protos/vega/commands/v1"
	"code.vegaprotocol.io/vega/types"
	"github.com/jfcg/sorty/v2"

	"code.vegaprotocol.io/vega/libs/proto"
)

var (
	key = (&types.PayloadEventForwarder{}).Key()

	hashKeys = []string{
		key,
	}
)

type efSnapshotState struct {
	changed    bool
	serialised []byte
}

func (f *Forwarder) Namespace() types.SnapshotNamespace {
	return types.EventForwarderSnapshot
}

func (f *Forwarder) Keys() []string {
	return hashKeys
}

func (f *Forwarder) Stopped() bool {
	return false
}

func (f *Forwarder) serialise() ([]byte, error) {
	// this is done without the lock because nothing can be acked during the commit phase which is when the snapshot is taken
	keys := make([]string, 0, len(f.ackedEvts))
	events := make([]*commandspb.ChainEvent, 0, len(f.ackedEvts))
	for key := range f.ackedEvts {
		keys = append(keys, key)
	}
	lsw := func(i, k, r, s int) bool {
		if keys[i] < keys[k] { // strict comparator like < or >
			if r != s {
				keys[r], keys[s] = keys[s], keys[r]
			}
			return true
		}
		return false
	}

	sorty.Sort(len(keys), lsw)

	for _, key := range keys {
		events = append(events, f.ackedEvts[key])
	}

	payload := types.Payload{
		Data: &types.PayloadEventForwarder{
			Events: events,
		},
	}
	return proto.Marshal(payload.IntoProto())
}

// get the serialised form of the given key.
func (f *Forwarder) getSerialised(k string) (data []byte, err error) {
	if k != key {
		return nil, types.ErrSnapshotKeyDoesNotExist
	}

	if !f.efss.changed {
		return f.efss.serialised, nil
	}

	f.efss.serialised, err = f.serialise()
	if err != nil {
		return nil, err
	}

	f.efss.changed = false
	return f.efss.serialised, nil
}

func (f *Forwarder) HasChanged(k string) bool {
	return f.efss.changed
}

func (f *Forwarder) GetState(k string) ([]byte, []types.StateProvider, error) {
	state, err := f.getSerialised(k)
	return state, nil, err
}

func (f *Forwarder) LoadState(ctx context.Context, p *types.Payload) ([]types.StateProvider, error) {
	if f.Namespace() != p.Data.Namespace() {
		return nil, types.ErrInvalidSnapshotNamespace
	}
	// see what we're reloading
	if pl, ok := p.Data.(*types.PayloadEventForwarder); ok {
		return nil, f.restore(ctx, pl.Events, p)
	}

	return nil, types.ErrUnknownSnapshotType
}

func (f *Forwarder) restore(ctx context.Context, events []*commandspb.ChainEvent, p *types.Payload) error {
	f.ackedEvts = map[string]*commandspb.ChainEvent{}
	for _, event := range events {
		key, err := f.getEvtKey(event)
		if err != nil {
			return err
		}
		f.ackedEvts[key] = event
	}

	var err error
	f.efss.changed = false
	f.efss.serialised, err = proto.Marshal(p.IntoProto())
	return err
}
