// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
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

	"code.vegaprotocol.io/vega/core/types"
	"github.com/emirpasic/gods/sets/treeset"

	"code.vegaprotocol.io/vega/libs/proto"
)

var (
	key = (&types.PayloadEventForwarder{}).Key()

	hashKeys = []string{
		key,
	}
)

type efSnapshotState struct {
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
	slice := make([]string, 0, f.ackedEvts.Size())
	iter := f.ackedEvts.Iterator()
	for iter.Next() {
		slice = append(slice, (iter.Value().(string)))
	}
	payload := types.Payload{
		Data: &types.PayloadEventForwarder{
			Keys: slice,
		},
	}
	return proto.Marshal(payload.IntoProto())
}

// get the serialised form of the given key.
func (f *Forwarder) getSerialised(k string) (data []byte, err error) {
	if k != key {
		return nil, types.ErrSnapshotKeyDoesNotExist
	}

	f.efss.serialised, err = f.serialise()
	if err != nil {
		return nil, err
	}

	return f.efss.serialised, nil
}

func (f *Forwarder) GetState(k string) ([]byte, []types.StateProvider, error) {
	state, err := f.getSerialised(k)
	return state, nil, err
}

func (f *Forwarder) LoadState(_ context.Context, p *types.Payload) ([]types.StateProvider, error) {
	if f.Namespace() != p.Data.Namespace() {
		return nil, types.ErrInvalidSnapshotNamespace
	}
	// see what we're reloading
	if pl, ok := p.Data.(*types.PayloadEventForwarder); ok {
		return nil, f.restore(pl.Keys, p)
	}

	return nil, types.ErrUnknownSnapshotType
}

func (f *Forwarder) restore(keys []string, p *types.Payload) error {
	f.ackedEvts = treeset.NewWithStringComparator()
	for _, v := range keys {
		f.ackedEvts.Add(v)
	}
	var err error
	f.efss.serialised, err = proto.Marshal(p.IntoProto())
	return err
}
