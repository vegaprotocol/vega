// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.DATANODE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package sqlsubscribers

import (
	"context"

	"github.com/pkg/errors"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/protos/vega"
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
