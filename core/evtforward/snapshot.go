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

package evtforward

import (
	"context"
	"slices"

	"code.vegaprotocol.io/vega/core/types"
	vgcontext "code.vegaprotocol.io/vega/libs/context"
	"code.vegaprotocol.io/vega/libs/proto"
	snapshotpb "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"

	"github.com/emirpasic/gods/sets/treeset"
	"golang.org/x/exp/maps"
)

var (
	key = (&types.PayloadEventForwarder{}).Key()

	hashKeys = []string{
		key,
	}
)

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
	slice := make([]*snapshotpb.EventForwarderBucket, 0, f.ackedEvts.Size())
	iter := f.ackedEvts.events.Iterator()
	for iter.Next() {
		v := iter.Value().(*ackedEvtBucket)
		hashes := maps.Keys(v.hashes)
		slices.Sort(hashes)
		slice = append(slice, &snapshotpb.EventForwarderBucket{
			Ts:     v.ts,
			Hashes: hashes,
		})
	}

	payload := types.Payload{
		Data: &types.PayloadEventForwarder{
			Buckets: slice,
		},
	}
	return proto.Marshal(payload.IntoProto())
}

// get the serialised form of the given key.
func (f *Forwarder) getSerialised(k string) (data []byte, err error) {
	if k != key {
		return nil, types.ErrSnapshotKeyDoesNotExist
	}

	return f.serialise()
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
		f.restore(ctx, pl)
		return nil, nil
	}

	return nil, types.ErrUnknownSnapshotType
}

func (f *Forwarder) restore(ctx context.Context, p *types.PayloadEventForwarder) {
	f.ackedEvts = &ackedEvents{
		timeService: f.timeService,
		events:      treeset.NewWith(ackedEvtBucketComparator),
	}

	// if we are executing a protocol upgrade,
	// let's force bucketing things. This will reduce
	// increase performance at startup, and everyone is starting
	// from the same snapshot, so that will keep state consistent
	if vgcontext.InProgressUpgrade(ctx) {
		for _, v := range p.Buckets {
			f.ackedEvts.AddAt(v.Ts, v.Hashes...)
		}
		return
	}

	for _, v := range p.Buckets {
		f.ackedEvts.RestoreExactAt(v.Ts, v.Hashes...)
	}
}
