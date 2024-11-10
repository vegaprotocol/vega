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

package sqlsubscribers

import (
	"context"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/datanode/entities"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"

	"github.com/pkg/errors"
)

type VaultEvent interface {
	events.Event
	VaultEvent() eventspb.VaultState
}

type VaultStore interface {
	Add(context.Context, *entities.VaultState) error
}

type Vault struct {
	subscriber
	store VaultStore
}

func NewVault(store VaultStore) *Vault {
	t := &Vault{
		store: store,
	}
	return t
}

func (v *Vault) Types() []events.Type {
	return []events.Type{events.VaultStateEvent}
}

func (v *Vault) Push(ctx context.Context, evt events.Event) error {
	return v.consume(ctx, evt.(VaultEvent))
}

func (v *Vault) consume(ctx context.Context, event VaultEvent) error {
	vaultEvent := event.VaultEvent()
	vaultState, err := entities.VaultStateFromProto(&vaultEvent, v.vegaTime)
	if err != nil {
		return errors.Wrap(err, "unable to parse vault state")
	}

	v.store.Add(ctx, vaultState)
	return nil
}

func (v *Vault) Name() string {
	return "Vault"
}
