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
	"fmt"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/datanode/entities"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

type ProtocolUpgradeProposalEvent interface {
	events.Event
	ProtocolUpgradeProposalEvent() eventspb.ProtocolUpgradeEvent
}

type pupAdder interface {
	AddProposal(context.Context, entities.ProtocolUpgradeProposal) error
}

type ProtocolUpgrade struct {
	subscriber
	store pupAdder
}

func NewProtocolUpgrade(store pupAdder) *ProtocolUpgrade {
	ps := &ProtocolUpgrade{
		store: store,
	}
	return ps
}

func (ps *ProtocolUpgrade) Types() []events.Type {
	return []events.Type{events.ProtocolUpgradeEvent}
}

func (ps *ProtocolUpgrade) Push(ctx context.Context, evt events.Event) error {
	return ps.consume(ctx, evt.(ProtocolUpgradeProposalEvent))
}

func (ps *ProtocolUpgrade) consume(ctx context.Context, event ProtocolUpgradeProposalEvent) error {
	pupProto := event.ProtocolUpgradeProposalEvent()
	protocolUpgradeProposal := entities.ProtocolUpgradeProposalFromProto(&pupProto, entities.TxHash(event.TxHash()), ps.vegaTime)

	if err := ps.store.AddProposal(ctx, protocolUpgradeProposal); err != nil {
		return fmt.Errorf("error adding protocol upgrade proposal: %w", err)
	}

	return nil
}

func (ps *ProtocolUpgrade) Name() string {
	return "ProtocolUpgrade"
}
