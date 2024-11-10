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

type RedemptionRequestEvent interface {
	events.Event
	RedemptionRequestEvent() eventspb.RedemptionRequest
}

type VaultRedemptionStore interface {
	Add(ctx context.Context, redemptionRequest *entities.RedemptionRequest) error
}

type VaultRedemptions struct {
	subscriber
	store VaultRedemptionStore
}

func NewVaultRedemptions(store VaultRedemptionStore) *VaultRedemptions {
	t := &VaultRedemptions{
		store: store,
	}
	return t
}

func (vr *VaultRedemptions) Types() []events.Type {
	return []events.Type{events.RedemptionRequestEvent}
}

func (vr *VaultRedemptions) Push(ctx context.Context, evt events.Event) error {
	return vr.consume(ctx, evt.(RedemptionRequestEvent))
}

func (vr *VaultRedemptions) consume(ctx context.Context, event RedemptionRequestEvent) error {
	rrEvent := event.RedemptionRequestEvent()
	rrEntity, err := entities.RedemptionRequestFromProto(&rrEvent, vr.vegaTime)
	if err != nil {
		return errors.Wrap(err, "unable to parse vault redemption request")
	}

	vr.store.Add(ctx, rrEntity)
	return nil
}

func (vr *VaultRedemptions) Name() string {
	return "RedemptionRequest"
}
