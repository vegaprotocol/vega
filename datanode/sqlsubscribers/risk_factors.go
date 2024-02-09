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

type RiskFactorEvent interface {
	events.Event
	RiskFactor() vega.RiskFactor
}

type RiskFactorStore interface {
	Upsert(context.Context, *entities.RiskFactor) error
}

type RiskFactor struct {
	subscriber
	store RiskFactorStore
}

func NewRiskFactor(store RiskFactorStore) *RiskFactor {
	return &RiskFactor{
		store: store,
	}
}

func (rf *RiskFactor) Types() []events.Type {
	return []events.Type{events.RiskFactorEvent}
}

func (rf *RiskFactor) Push(ctx context.Context, evt events.Event) error {
	return rf.consume(ctx, evt.(RiskFactorEvent))
}

func (rf *RiskFactor) consume(ctx context.Context, event RiskFactorEvent) error {
	riskFactor := event.RiskFactor()
	record, err := entities.RiskFactorFromProto(&riskFactor, entities.TxHash(event.TxHash()), rf.vegaTime)
	if err != nil {
		return errors.Wrap(err, "converting risk factor proto to database entity failed")
	}

	return errors.Wrap(rf.store.Upsert(ctx, record), "inserting risk factor to SQL store failed")
}
