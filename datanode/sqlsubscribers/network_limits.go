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

type NetworkLimitsEvent interface {
	events.Event
	NetworkLimits() *vega.NetworkLimits
}

type NetworkLimitStore interface {
	Add(context.Context, entities.NetworkLimits) error
}

type NetworkLimits struct {
	subscriber
	store NetworkLimitStore
}

func NewNetworkLimitSub(store NetworkLimitStore) *NetworkLimits {
	t := &NetworkLimits{
		store: store,
	}
	return t
}

func (nl *NetworkLimits) Types() []events.Type {
	return []events.Type{events.NetworkLimitsEvent}
}

func (nl *NetworkLimits) Push(ctx context.Context, evt events.Event) error {
	return nl.consume(ctx, evt.(NetworkLimitsEvent))
}

func (nl *NetworkLimits) consume(ctx context.Context, event NetworkLimitsEvent) error {
	protoLimits := event.NetworkLimits()
	limits := entities.NetworkLimitsFromProto(protoLimits, entities.TxHash(event.TxHash()))
	limits.VegaTime = nl.vegaTime

	return errors.Wrap(nl.store.Add(ctx, limits), "error adding network limits")
}

func (nl *NetworkLimits) Name() string {
	return "NetworkLimits"
}
