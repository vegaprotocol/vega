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
