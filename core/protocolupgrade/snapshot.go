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

package protocolupgrade

import (
	"context"
	"sort"

	"code.vegaprotocol.io/vega/core/types"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
	snappb "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"

	"github.com/golang/protobuf/proto"
)

func (e *Engine) Namespace() types.SnapshotNamespace {
	return types.ProtocolUpgradeSnapshot
}

func (e *Engine) Keys() []string {
	return e.hashKeys
}

func (e *Engine) Stopped() bool {
	return false
}

// get the serialised form and hash of the given key.
func (e *Engine) serialise() ([]byte, error) {
	events := make([]*eventspb.ProtocolUpgradeEvent, 0, len(e.activeProposals))
	for _, evt := range e.events {
		events = append(events, evt)
	}

	sort.SliceStable(events, func(i, j int) bool {
		if events[i].VegaReleaseTag == events[j].VegaReleaseTag {
			return events[i].UpgradeBlockHeight < events[j].UpgradeBlockHeight
		}
		return events[i].VegaReleaseTag < events[j].VegaReleaseTag
	})

	payloadProtocolUpgradeProposals := &types.PayloadProtocolUpgradeProposals{
		Proposals: &snappb.ProtocolUpgradeProposals{
			ActiveProposals: events,
		},
	}

	if e.upgradeStatus.AcceptedReleaseInfo != nil {
		payloadProtocolUpgradeProposals.Proposals.AcceptedProposal = &snappb.AcceptedProtocolUpgradeProposal{
			UpgradeBlockHeight: e.upgradeStatus.AcceptedReleaseInfo.UpgradeBlockHeight,
			VegaReleaseTag:     e.upgradeStatus.AcceptedReleaseInfo.VegaReleaseTag,
		}
	}

	payload := types.Payload{
		Data: payloadProtocolUpgradeProposals,
	}

	data, err := proto.Marshal(payload.IntoProto())
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (e *Engine) GetState(k string) ([]byte, []types.StateProvider, error) {
	state, err := e.serialise()
	return state, nil, err
}

func (e *Engine) LoadState(ctx context.Context, p *types.Payload) ([]types.StateProvider, error) {
	if e.Namespace() != p.Data.Namespace() {
		return nil, types.ErrInvalidSnapshotNamespace
	}
	pl := p.Data.(*types.PayloadProtocolUpgradeProposals)
	e.activeProposals = make(map[string]*protocolUpgradeProposal, len(pl.Proposals.ActiveProposals))
	e.events = make(map[string]*eventspb.ProtocolUpgradeEvent, len(pl.Proposals.ActiveProposals))

	for _, pue := range pl.Proposals.ActiveProposals {
		ID := protocolUpgradeProposalID(pue.UpgradeBlockHeight, pue.VegaReleaseTag)
		e.events[ID] = pue
	}

	for ID, evt := range e.events {
		e.activeProposals[ID] = &protocolUpgradeProposal{
			vegaReleaseTag: evt.VegaReleaseTag,
			blockHeight:    evt.UpgradeBlockHeight,
			accepted:       make(map[string]struct{}, len(evt.Approvers)),
		}
		for _, approver := range evt.Approvers {
			e.activeProposals[ID].accepted[approver] = struct{}{}
		}
	}

	if pl.Proposals.AcceptedProposal != nil {
		e.upgradeStatus.AcceptedReleaseInfo = &types.ReleaseInfo{
			UpgradeBlockHeight: pl.Proposals.AcceptedProposal.UpgradeBlockHeight,
			VegaReleaseTag:     pl.Proposals.AcceptedProposal.VegaReleaseTag,
		}
	}

	return nil, nil
}
