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

package parties

import (
	"context"
	"fmt"
	"strings"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/proto"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
	snapshotpb "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"

	"golang.org/x/exp/slices"
)

type SnapshottedEngine struct {
	*Engine

	pl types.Payload

	stopped bool

	hashKeys   []string
	partiesKey string
}

func (e *SnapshottedEngine) Namespace() types.SnapshotNamespace {
	return types.PartiesSnapshot
}

func (e *SnapshottedEngine) Keys() []string {
	return e.hashKeys
}

func (e *SnapshottedEngine) GetState(k string) ([]byte, []types.StateProvider, error) {
	state, err := e.serialise(k)
	return state, nil, err
}

func (e *SnapshottedEngine) LoadState(_ context.Context, p *types.Payload) ([]types.StateProvider, error) {
	if e.Namespace() != p.Data.Namespace() {
		return nil, types.ErrInvalidSnapshotNamespace
	}

	switch data := p.Data.(type) {
	case *types.PayloadParties:
		e.Engine.loadPartiesFromSnapshot(data)
		return nil, nil
	default:
		return nil, types.ErrUnknownSnapshotType
	}
}

func (e *SnapshottedEngine) Stopped() bool {
	return e.stopped
}

func (e *SnapshottedEngine) StopSnapshots() {
	e.stopped = true
}

func (e *SnapshottedEngine) serialise(k string) ([]byte, error) {
	if e.stopped {
		return nil, nil
	}

	switch k {
	case e.partiesKey:
		return e.serialiseParties()
	default:
		return nil, types.ErrSnapshotKeyDoesNotExist
	}
}

func (e *SnapshottedEngine) serialiseParties() ([]byte, error) {
	profiles := e.Engine.profiles
	profilesSnapshot := make([]*snapshotpb.PartyProfile, 0, len(profiles))
	for _, profile := range profiles {
		profileSnapshot := &snapshotpb.PartyProfile{
			PartyId: profile.PartyID.String(),
			Alias:   profile.Alias,
		}
		for k, v := range profile.Metadata {
			profileSnapshot.Metadata = append(profileSnapshot.Metadata, &vegapb.Metadata{
				Key:   k,
				Value: v,
			})
		}

		// Ensure deterministic order among the metadata.
		slices.SortStableFunc(profileSnapshot.Metadata, func(a, b *vegapb.Metadata) int {
			return strings.Compare(a.Key, b.Key)
		})

		for k := range profile.DerivedKeys {
			profileSnapshot.DerivedKeys = append(profileSnapshot.DerivedKeys, k)
		}

		// Ensure deterministic order among the derived keys.
		slices.Sort(profileSnapshot.DerivedKeys)

		profilesSnapshot = append(profilesSnapshot, profileSnapshot)
	}

	// Ensure deterministic order among the parties.
	slices.SortStableFunc(profilesSnapshot, func(a, b *snapshotpb.PartyProfile) int {
		return strings.Compare(a.PartyId, b.PartyId)
	})

	payload := &snapshotpb.Payload{
		Data: &snapshotpb.Payload_Parties{
			Parties: &snapshotpb.Parties{
				Profiles: profilesSnapshot,
			},
		},
	}

	serialisedProfiles, err := proto.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("could not serialize parties payload: %w", err)
	}

	return serialisedProfiles, nil
}

func (e *SnapshottedEngine) buildHashKeys() {
	e.partiesKey = (&types.PayloadParties{}).Key()

	e.hashKeys = append([]string{}, e.partiesKey)
}

func NewSnapshottedEngine(broker Broker) *SnapshottedEngine {
	se := &SnapshottedEngine{
		Engine:  NewEngine(broker),
		pl:      types.Payload{},
		stopped: false,
	}

	se.buildHashKeys()

	return se
}
