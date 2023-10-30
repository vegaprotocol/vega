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

package events

import (
	"context"

	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

type ProtocolUpgradeProposalEvent struct {
	*Base
	UpgradeBlockHeight uint64
	VegaReleaseTag     string
	AcceptedBy         []string
	ProposalStatus     eventspb.ProtocolUpgradeProposalStatus
}

func NewProtocolUpgradeProposalEvent(ctx context.Context, upgradeBlockHeight uint64, vegaReleaseTag string, acceptedBy []string, status eventspb.ProtocolUpgradeProposalStatus) *ProtocolUpgradeProposalEvent {
	return &ProtocolUpgradeProposalEvent{
		Base:               newBase(ctx, ProtocolUpgradeEvent),
		UpgradeBlockHeight: upgradeBlockHeight,
		VegaReleaseTag:     vegaReleaseTag,
		AcceptedBy:         acceptedBy,
		ProposalStatus:     status,
	}
}

func (pup ProtocolUpgradeProposalEvent) Proto() eventspb.ProtocolUpgradeEvent {
	return eventspb.ProtocolUpgradeEvent{
		VegaReleaseTag:     pup.VegaReleaseTag,
		UpgradeBlockHeight: pup.UpgradeBlockHeight,
		Approvers:          pup.AcceptedBy,
		Status:             pup.ProposalStatus,
	}
}

func (pup ProtocolUpgradeProposalEvent) ProtocolUpgradeProposalEvent() eventspb.ProtocolUpgradeEvent {
	return pup.Proto()
}

func (pup ProtocolUpgradeProposalEvent) StreamMessage() *eventspb.BusEvent {
	p := pup.Proto()
	busEvent := newBusEventFromBase(pup.Base)
	busEvent.Event = &eventspb.BusEvent_ProtocolUpgradeEvent{
		ProtocolUpgradeEvent: &p,
	}

	return busEvent
}

func ProtocolUpgradeProposalEventFromStream(ctx context.Context, be *eventspb.BusEvent) *ProtocolUpgradeProposalEvent {
	event := be.GetProtocolUpgradeEvent()
	if event == nil {
		return nil
	}

	return &ProtocolUpgradeProposalEvent{
		Base:               newBaseFromBusEvent(ctx, ProtocolUpgradeEvent, be),
		UpgradeBlockHeight: event.UpgradeBlockHeight,
		VegaReleaseTag:     event.VegaReleaseTag,
		AcceptedBy:         event.Approvers,
		ProposalStatus:     event.Status,
	}
}
