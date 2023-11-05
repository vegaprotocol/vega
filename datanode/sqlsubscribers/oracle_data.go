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

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/datanode/entities"
	vegapb "code.vegaprotocol.io/vega/protos/vega"

	"github.com/pkg/errors"
)

type OracleDataEvent interface {
	events.Event
	OracleData() vegapb.OracleData
}

type OracleDataStore interface {
	Add(context.Context, *entities.OracleData) error
}

type OracleData struct {
	subscriber
	store OracleDataStore
}

func NewOracleData(store OracleDataStore) *OracleData {
	return &OracleData{
		store: store,
	}
}

func (od *OracleData) Types() []events.Type {
	return []events.Type{events.OracleDataEvent}
}

func (od *OracleData) Push(ctx context.Context, evt events.Event) error {
	return od.consume(ctx, evt.(OracleDataEvent))
}

func (od *OracleData) consume(ctx context.Context, event OracleDataEvent) error {
	data := event.OracleData()
	entity, err := entities.OracleDataFromProto(&data, entities.TxHash(event.TxHash()), od.vegaTime, event.Sequence())
	if err != nil {
		errors.Wrap(err, "converting oracle data proto to database entity failed")
	}

	return errors.Wrap(od.store.Add(ctx, entity), "inserting oracle data proto to SQL store failed")
}
