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

package validators

import (
	"context"
	"errors"
	"sort"
	"sync"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/proto"
	"code.vegaprotocol.io/vega/logging"
)

var (
	key = (&types.PayloadWitness{}).Key()

	hashKeys = []string{
		key,
	}

	ErrSnapshotKeyDoesNotExist = errors.New("unknown key for witness snapshot")
)

type witnessSnapshotState struct {
	serialised []byte
	mu         sync.Mutex
}

func (w *Witness) Namespace() types.SnapshotNamespace {
	return types.WitnessSnapshot
}

func (w *Witness) Keys() []string {
	return hashKeys
}

func (w *Witness) serialiseWitness() ([]byte, error) {
	w.log.Debug("serialising witness resources", logging.Int("n", len(w.resources)))
	resources := make([]*types.Resource, 0, len(w.resources))
	for id, r := range w.resources {
		r.mu.Lock()
		votes := make([]string, 0, len(r.votes))
		for v := range r.votes {
			votes = append(votes, v)
		}
		sort.Strings(votes)
		resources = append(resources, &types.Resource{
			ID:         id,
			CheckUntil: r.checkUntil,
			Votes:      votes,
		})
		r.mu.Unlock()
	}

	// sort the resources
	sort.SliceStable(resources, func(i, j int) bool { return resources[i].ID < resources[j].ID })

	payload := types.Payload{
		Data: &types.PayloadWitness{
			Witness: &types.Witness{
				Resources: resources,
			},
		},
	}
	x := payload.IntoProto()
	return proto.Marshal(x)
}

// get the serialised form of the given key.
func (w *Witness) serialise(k string) ([]byte, error) {
	if k != key {
		return nil, ErrSnapshotKeyDoesNotExist
	}

	w.wss.mu.Lock()
	defer w.wss.mu.Unlock()
	data, err := w.serialiseWitness()
	if err != nil {
		return nil, err
	}

	w.wss.serialised = data
	return data, nil
}

func (w *Witness) Stopped() bool { return false }

func (w *Witness) GetState(k string) ([]byte, []types.StateProvider, error) {
	state, err := w.serialise(k)
	return state, nil, err
}

func (w *Witness) LoadState(_ context.Context, p *types.Payload) ([]types.StateProvider, error) {
	if w.Namespace() != p.Data.Namespace() {
		return nil, types.ErrInvalidSnapshotNamespace
	}
	// see what we're reloading
	switch pl := p.Data.(type) {
	case *types.PayloadWitness:
		return nil, w.restore(pl.Witness, p)
	default:
		return nil, types.ErrUnknownSnapshotType
	}
}

func (w *Witness) restore(witness *types.Witness, p *types.Payload) error {
	w.resources = map[string]*res{}
	w.needResendRes = map[string]struct{}{}

	w.log.Debug("restoring witness resources", logging.Int("n", len(witness.Resources)))
	for _, r := range witness.Resources {
		w.resources[r.ID] = &res{
			checkUntil: r.CheckUntil,
			votes:      map[string]struct{}{},
		}
		selfVoted := false
		for _, v := range r.Votes {
			w.resources[r.ID].votes[v] = struct{}{}
			if v == w.top.SelfVegaPubKey() {
				selfVoted = true
			}
		}

		// if not a validator or we've seen a self vote set the state to vote sent
		// otherwise we stay in validated state
		state := notValidated
		if !w.top.IsValidator() || selfVoted {
			state = voteSent
		}

		w.resources[r.ID].state.Store(state)
	}

	w.wss.mu.Lock()
	var err error
	w.wss.serialised, err = proto.Marshal(p.IntoProto())
	w.wss.mu.Unlock()
	return err
}

func (w *Witness) RestoreResource(r Resource, cb func(interface{}, bool)) error {
	w.log.Info("finalising restored resource", logging.String("id", r.GetID()))
	if _, ok := w.resources[r.GetID()]; !ok {
		return ErrInvalidResourceIDForNodeVote
	}

	res := w.resources[r.GetID()]
	res.cb = cb
	res.res = r
	ctx, cfunc := context.WithDeadline(context.Background(), res.checkUntil)
	res.cfunc = cfunc
	if w.top.IsValidator() && res.state.Load() != voteSent {
		// no initial delay
		go w.start(ctx, res, nil)
	}
	return nil
}
