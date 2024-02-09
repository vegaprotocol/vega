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
	"code.vegaprotocol.io/vega/protos/vega"

	"github.com/pkg/errors"
)

type LiquidityProvisionEvent interface {
	events.Event
	LiquidityProvision() *vega.LiquidityProvision
}

type LiquidityProvisionStore interface {
	Upsert(context.Context, entities.LiquidityProvision) error
	Flush(ctx context.Context) error
}

type LiquidityProvision struct {
	subscriber
	store LiquidityProvisionStore
}

func NewLiquidityProvision(store LiquidityProvisionStore) *LiquidityProvision {
	return &LiquidityProvision{
		store: store,
	}
}

func (lp *LiquidityProvision) Types() []events.Type {
	return []events.Type{events.LiquidityProvisionEvent}
}

func (lp *LiquidityProvision) Flush(ctx context.Context) error {
	err := lp.flush(ctx)
	if err != nil {
		return errors.Wrap(err, "flushing liquidity provisions")
	}

	return nil
}

func (lp *LiquidityProvision) Push(ctx context.Context, evt events.Event) error {
	return lp.consume(ctx, evt.(LiquidityProvisionEvent))
}

func (lp *LiquidityProvision) flush(ctx context.Context) error {
	return errors.Wrap(lp.store.Flush(ctx), "flushing liquidity provisions")
}

func (lp *LiquidityProvision) consume(ctx context.Context, event LiquidityProvisionEvent) error {
	entity, err := entities.LiquidityProvisionFromProto(event.LiquidityProvision(), entities.TxHash(event.TxHash()), lp.vegaTime)
	if err != nil {
		return errors.Wrap(err, "converting liquidity provision event to database entity failed")
	}

	err = lp.store.Upsert(ctx, entity)
	return errors.Wrap(err, "adding liquidity provision to store")
}
