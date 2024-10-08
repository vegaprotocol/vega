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
	AMMPool() *eventspb.AMM
}

type AMMPoolStore interface {
	Upsert(ctx context.Context, pool entities.AMMPool) error
}

type AMMPools struct {
	subscriber
	store       AMMPoolStore
	marketDepth MarketDepthService
}

func NewAMMPools(store AMMPoolStore, marketDepth MarketDepthService) *AMMPools {
	return &AMMPools{
		store:       store,
		marketDepth: marketDepth,
	}
}

func (p *AMMPools) Types() []events.Type {
	return []events.Type{events.AMMPoolEvent}
}

func (p *AMMPools) Push(ctx context.Context, evt events.Event) error {
	return p.consume(ctx, evt.(AMMPoolEvent), evt.Sequence())
}

func (p *AMMPools) consume(ctx context.Context, pe AMMPoolEvent, seqNum uint64) error {
	ammPool, err := entities.AMMPoolFromProto(pe.AMMPool(), p.vegaTime)
	if err != nil {
		return fmt.Errorf("cannot parse AMM Pool event from proto message: %w", err)
	}

	err = p.store.Upsert(ctx, ammPool)
	if err != nil {
		return fmt.Errorf("could not save AMM Pool event: %w", err)
	}

	if ammPool.Status == entities.AMMStatusRejected || ammPool.Status == entities.AMMStatusUnspecified {
		return nil
	}

	// send it to the market-depth service
	p.marketDepth.OnAMMUpdate(ammPool, p.vegaTime, seqNum)

	return nil
}

func (p *AMMPools) Name() string {
	return "AMMPools"
}
