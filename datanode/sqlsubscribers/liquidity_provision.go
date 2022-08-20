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

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/protos/vega"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
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
	log   *logging.Logger

	eventDeduplicator *eventDeduplicator[string, LiquidityProvisionEvent]
}

func NewLiquidityProvision(store LiquidityProvisionStore, log *logging.Logger) *LiquidityProvision {
	return &LiquidityProvision{
		store: store,
		log:   log,
		eventDeduplicator: NewEventDeduplicator(func(ctx context.Context, lpe LiquidityProvisionEvent) string {
			return lpe.LiquidityProvision().Id
		},
			func(lpe1 LiquidityProvisionEvent, lpe2 LiquidityProvisionEvent) bool {
				lp1 := lpe1.LiquidityProvision()
				lp2 := lpe2.LiquidityProvision()
				return proto.Equal(lp1, lp2)
			},
		),
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
	events := lp.eventDeduplicator.Flush()
	for _, event := range events {
		provision := event.LiquidityProvision()
		entity, err := entities.LiquidityProvisionFromProto(provision, entities.TxHash(event.TxHash()), lp.vegaTime)
		if err != nil {
			return errors.Wrap(err, "converting liquidity provision to database entity failed")
		}
		lp.store.Upsert(ctx, entity)
	}

	err := lp.store.Flush(ctx)

	return errors.Wrap(err, "flushing liquidity provisions")
}

func (lp *LiquidityProvision) consume(ctx context.Context, event LiquidityProvisionEvent) error {
	lp.eventDeduplicator.AddEvent(ctx, event)
	return nil
}
