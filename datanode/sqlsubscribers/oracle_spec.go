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
	vegapb "code.vegaprotocol.io/vega/protos/vega"
)

type OracleSpecEvent interface {
	events.Event
	OracleSpec() vegapb.OracleSpec
}

type OracleSpecStore interface {
	Upsert(context.Context, *entities.OracleSpec) error
}

type OracleSpec struct {
	subscriber
	store OracleSpecStore
}

func NewOracleSpec(store OracleSpecStore) *OracleSpec {
	return &OracleSpec{
		store: store,
	}
}

func (od *OracleSpec) Types() []events.Type {
	return []events.Type{events.OracleSpecEvent}
}

func (od *OracleSpec) Push(ctx context.Context, evt events.Event) error {
	return od.consume(ctx, evt.(OracleSpecEvent))
}

func (od *OracleSpec) consume(ctx context.Context, event OracleSpecEvent) error {
	spec := event.OracleSpec()
	entity, err := entities.OracleSpecFromProto(&spec, entities.TxHash(event.TxHash()), od.vegaTime)
	if err != nil {
		return errors.Wrap(err, "converting oracle spec to database entity failed")
	}

	return errors.Wrap(od.store.Upsert(ctx, entity), "inserting oracle spec to SQL store failed")
}
