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

package entities

import (
	"encoding/json"
	"fmt"
	"time"

	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

type ProtocolUpgradeProposal struct {
	UpgradeBlockHeight uint64
	VegaReleaseTag     string
	Approvers          []string
	Status             ProtocolUpgradeProposalStatus
	TxHash             TxHash
	VegaTime           time.Time
}

func ProtocolUpgradeProposalFromProto(p *eventspb.ProtocolUpgradeEvent, txHash TxHash, vegaTime time.Time) ProtocolUpgradeProposal {
	proposal := ProtocolUpgradeProposal{
		UpgradeBlockHeight: p.UpgradeBlockHeight,
		VegaReleaseTag:     p.VegaReleaseTag,
		Approvers:          p.Approvers,
		Status:             ProtocolUpgradeProposalStatus(p.Status),
		TxHash:             txHash,
		VegaTime:           vegaTime,
	}
	return proposal
}

func (p ProtocolUpgradeProposal) ToProto() *eventspb.ProtocolUpgradeEvent {
	return &eventspb.ProtocolUpgradeEvent{
		UpgradeBlockHeight: p.UpgradeBlockHeight,
		VegaReleaseTag:     p.VegaReleaseTag,
		Approvers:          p.Approvers,
		Status:             eventspb.ProtocolUpgradeProposalStatus(p.Status),
	}
}

func (p ProtocolUpgradeProposal) Cursor() *Cursor {
	pc := ProtocolUpgradeProposalCursor{
		VegaTime:           p.VegaTime,
		UpgradeBlockHeight: p.UpgradeBlockHeight,
		VegaReleaseTag:     p.VegaReleaseTag,
	}
	return NewCursor(pc.String())
}

func (p ProtocolUpgradeProposal) ToProtoEdge(_ ...any) (*v2.ProtocolUpgradeProposalEdge, error) {
	return &v2.ProtocolUpgradeProposalEdge{
		Node:   p.ToProto(),
		Cursor: p.Cursor().Encode(),
	}, nil
}

type ProtocolUpgradeProposalCursor struct {
	VegaTime           time.Time
	UpgradeBlockHeight uint64
	VegaReleaseTag     string
}

func (pc ProtocolUpgradeProposalCursor) String() string {
	bs, err := json.Marshal(pc)
	if err != nil {
		panic(fmt.Errorf("failed to marshal protocol upgrade proposal cursor: %w", err))
	}
	return string(bs)
}

func (pc *ProtocolUpgradeProposalCursor) Parse(cursorString string) error {
	if cursorString == "" {
		return nil
	}
	return json.Unmarshal([]byte(cursorString), pc)
}
