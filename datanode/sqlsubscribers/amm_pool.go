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
	"fmt"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/datanode/entities"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

type AMMPoolEvent interface {
	events.Event
	AMMPool() *eventspb.AMMPool
}

type AMMPoolStore interface {
	Upsert(ctx context.Context, pool entities.AMMPool) error
}

type AMMPools struct {
	subscriber
	store AMMPoolStore
}

func NewAMMPools(store AMMPoolStore) *AMMPools {
	return &AMMPools{
		store: store,
	}
}

func (p *AMMPools) Types() []events.Type {
	return []events.Type{events.AMMPoolEvent}
}

func (p *AMMPools) Push(ctx context.Context, evt events.Event) error {
	return p.consume(ctx, evt.(AMMPoolEvent))
}

func (p *AMMPools) consume(ctx context.Context, pe AMMPoolEvent) error {
	ammPool, err := entities.AMMPoolFromProto(pe.AMMPool(), p.vegaTime)
	if err != nil {
		return fmt.Errorf("cannot parse AMM Pool event from proto message: %w", err)
	}

	err = p.store.Upsert(ctx, ammPool)
	if err != nil {
		return fmt.Errorf("could not save AMM Pool event: %w", err)
	}

	return nil
}
