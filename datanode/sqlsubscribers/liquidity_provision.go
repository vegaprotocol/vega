// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
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
	"time"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/events"
	"github.com/pkg/errors"
)

type LiquidityProvisionEvent interface {
	events.Event
	LiquidityProvision() *vega.LiquidityProvision
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/liquidity_provision_mock.go -package mocks code.vegaprotocol.io/data-node/sqlsubscribers LiquidityProvisionStore
type LiquidityProvisionStore interface {
	Upsert(context.Context, entities.LiquidityProvision) error
	Flush(ctx context.Context) error
}

type LiquidityProvision struct {
	subscriber
	store LiquidityProvisionStore
	log   *logging.Logger

	eventDeduplicator *eventDeduplicator[string, *vega.LiquidityProvision]
}

func NewLiquidityProvision(store LiquidityProvisionStore, log *logging.Logger) *LiquidityProvision {
	return &LiquidityProvision{
		store: store,
		log:   log,
		eventDeduplicator: NewEventDeduplicator[string, *vega.LiquidityProvision](func(ctx context.Context,
			lp *vega.LiquidityProvision, vegaTime time.Time) (string, error) {
			return lp.Id, nil
		}),
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

	updates := lp.eventDeduplicator.Flush()
	for _, update := range updates {
		entity, err := entities.LiquidityProvisionFromProto(update, lp.vegaTime)

		if err != nil {
			return errors.Wrap(err, "converting liquidity provision to database entity failed")
		}
		lp.store.Upsert(ctx, entity)
	}

	err := lp.store.Flush(ctx)

	return errors.Wrap(err, "flushing liquidity provisions")
}

func (lp *LiquidityProvision) consume(ctx context.Context, event LiquidityProvisionEvent) error {
	provision := event.LiquidityProvision()
	lp.eventDeduplicator.AddEvent(ctx, provision, lp.vegaTime)
	return nil
}
