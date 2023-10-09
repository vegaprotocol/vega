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
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/datanode/entities"
)

type PartyEvent interface {
	events.Event
	Party() types.Party
}

type PartyStore interface {
	Add(context.Context, entities.Party) error
}

type Party struct {
	subscriber
	store PartyStore
}

func NewParty(store PartyStore) *Party {
	ps := &Party{
		store: store,
	}
	return ps
}

func (ps *Party) Types() []events.Type {
	return []events.Type{events.PartyEvent}
}

func (ps *Party) Push(ctx context.Context, evt events.Event) error {
	return ps.consume(ctx, evt.(PartyEvent))
}

func (ps *Party) consume(ctx context.Context, event PartyEvent) error {
	pp := event.Party().IntoProto()
	p := entities.PartyFromProto(pp, entities.TxHash(event.TxHash()))
	vt := ps.vegaTime
	p.VegaTime = &vt

	return errors.Wrap(ps.store.Add(ctx, p), "error adding party:%w")
}
