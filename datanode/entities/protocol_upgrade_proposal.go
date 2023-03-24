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

package entities

import (
	"encoding/json"
	"fmt"
	"time"

	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

type ProtocolUpgradeProposal struct {
	VegaTime           time.Time
	VegaReleaseTag     string
	TxHash             TxHash
	Approvers          []string
	UpgradeBlockHeight uint64
	Status             ProtocolUpgradeProposalStatus
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
	VegaReleaseTag     string
	UpgradeBlockHeight uint64
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
