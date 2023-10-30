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

	"code.vegaprotocol.io/vega/libs/num"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

type ValidatorScore struct {
	*Base
	NodeID               string
	EpochSeq             string
	ValidatorScore       string
	NormalisedScore      string
	RawValidatorScore    string
	ValidatorPerformance string
	MultisigScore        string
	ValidatorStatus      string
}

func NewValidatorScore(ctx context.Context, nodeID, epochSeq string, score, normalisedScore, rawValidatorScore,
	validatorPerformance num.Decimal, multisigScore num.Decimal, validatorStatus string,
) *ValidatorScore {
	return &ValidatorScore{
		Base:                 newBase(ctx, ValidatorScoreEvent),
		NodeID:               nodeID,
		EpochSeq:             epochSeq,
		ValidatorScore:       score.String(),
		NormalisedScore:      normalisedScore.String(),
		RawValidatorScore:    rawValidatorScore.String(),
		ValidatorPerformance: validatorPerformance.String(),
		MultisigScore:        multisigScore.String(),
		ValidatorStatus:      validatorStatus,
	}
}

func (vd ValidatorScore) Proto() eventspb.ValidatorScoreEvent {
	return eventspb.ValidatorScoreEvent{
		NodeId:               vd.NodeID,
		EpochSeq:             vd.EpochSeq,
		ValidatorScore:       vd.ValidatorScore,
		NormalisedScore:      vd.NormalisedScore,
		ValidatorPerformance: vd.ValidatorPerformance,
		RawValidatorScore:    vd.RawValidatorScore,
		MultisigScore:        vd.MultisigScore,
		ValidatorStatus:      vd.ValidatorStatus,
	}
}

func (vd ValidatorScore) ValidatorScoreEvent() eventspb.ValidatorScoreEvent {
	return vd.Proto()
}

func (vd ValidatorScore) StreamMessage() *eventspb.BusEvent {
	p := vd.Proto()
	busEvent := newBusEventFromBase(vd.Base)
	busEvent.Event = &eventspb.BusEvent_ValidatorScore{
		ValidatorScore: &p,
	}

	return busEvent
}

func ValidatorScoreEventFromStream(ctx context.Context, be *eventspb.BusEvent) *ValidatorScore {
	event := be.GetValidatorScore()
	if event == nil {
		return nil
	}

	return &ValidatorScore{
		Base:                 newBaseFromBusEvent(ctx, ValidatorScoreEvent, be),
		NodeID:               event.GetNodeId(),
		EpochSeq:             event.GetEpochSeq(),
		ValidatorScore:       event.ValidatorScore,
		NormalisedScore:      event.NormalisedScore,
		RawValidatorScore:    event.RawValidatorScore,
		ValidatorPerformance: event.ValidatorPerformance,
		MultisigScore:        event.MultisigScore,
		ValidatorStatus:      event.ValidatorStatus,
	}
}
