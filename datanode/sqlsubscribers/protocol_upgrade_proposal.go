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
