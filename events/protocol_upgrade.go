// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package events

import (
	"context"

	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
)

type ProtocolUpgradeProposalEvent struct {
	*Base
	UpgradeBlockHeight uint64
	VegaReleaseTag     string
	DataNodeReleaseTag string
	AcceptedBy         []string
	ProposalStatus     eventspb.ProtocolUpgradeProposalStatus
}

func NewProtocolUpgradeProposalEvent(ctx context.Context, upgradeBlockHeight uint64, vegaReleaseTag string, dataNodeReleaseTag string, acceptedBy []string, status eventspb.ProtocolUpgradeProposalStatus) *ProtocolUpgradeProposalEvent {
	return &ProtocolUpgradeProposalEvent{
		Base:               newBase(ctx, ValidatorScoreEvent),
		UpgradeBlockHeight: upgradeBlockHeight,
		VegaReleaseTag:     vegaReleaseTag,
		DataNodeReleaseTag: dataNodeReleaseTag,
		AcceptedBy:         acceptedBy,
		ProposalStatus:     status,
	}
}

func (pup ProtocolUpgradeProposalEvent) Proto() eventspb.ProtocolUpgradeEvent {
	return eventspb.ProtocolUpgradeEvent{
		VegaReleaseTag:     pup.VegaReleaseTag,
		DataNodeReleaseTag: pup.DataNodeReleaseTag,
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
		DataNodeReleaseTag: event.DataNodeReleaseTag,
		AcceptedBy:         event.Approvers,
		ProposalStatus:     event.Status,
	}
}
