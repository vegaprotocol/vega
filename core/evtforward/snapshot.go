// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package evtforward

import (
	"context"

	commandspb "code.vegaprotocol.io/protos/vega/commands/v1"
	"code.vegaprotocol.io/vega/core/types"

	"code.vegaprotocol.io/vega/core/libs/proto"
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
	payload := types.Payload{
		Data: &types.PayloadEventForwarder{
			Events: f.ackedEvtsSlice,
		},
	}
	return proto.Marshal(payload.IntoProto())
}

// get the serialised form of the given key.
func (f *Forwarder) getSerialised(k string) (data []byte, err error) {
	if k != key {
		return nil, types.ErrSnapshotKeyDoesNotExist
	}

	if !f.HasChanged(k) {
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
	// return f.efss.changed
	return true
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
	f.ackedEvtsSlice = events

	var err error
	f.efss.changed = false
	f.efss.serialised, err = proto.Marshal(p.IntoProto())
	return err
}
